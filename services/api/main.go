package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	jobmodel "slideagent/libs/jobmodel-go"
)

type createJobRequest struct {
	// createJobRequest はジョブ作成APIの入力。
	PitchText   string `json:"pitchText"`
	MaxAttempts *int   `json:"maxAttempts,omitempty"`
}

type Artifact struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	Path string `json:"path"`
}

func jobsDir() string {
	// JOBS_DIR が設定されていればそれを使い、未設定なら既定パスを返す。
	if v := strings.TrimSpace(os.Getenv("JOBS_DIR")); v != "" {
		return v
	}
	return filepath.Join("..", "..", "output", "jobs")
}

func ensureJobsDir() error {
	/* ディレクトリが存在しない場合は作成する関数 */
	return os.MkdirAll(jobsDir(), 0o755)
}

func jobPath(id string) string {
	/* ジョブIDから保存ファイルのパスを返す関数 */
	return filepath.Join(jobsDir(), id+".json")
}

func saveJob(job jobmodel.Job) error {
	/* Job を JSON ファイルとして保存する関数 */
	b, err := json.MarshalIndent(job, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(jobPath(job.ID), b, 0o644)
}

func loadJob(id string) (jobmodel.Job, error) {
	/* ジョブIDからJSONファイルを読み込んでJob構造体に変換して返す関数 */
	var job jobmodel.Job
	b, err := os.ReadFile(jobPath(id))
	if err != nil {
		return job, err
	}
	if err := json.Unmarshal(b, &job); err != nil {
		return job, err
	}
	return job, nil
}

func listJobs(status string, limit int) ([]jobmodel.Job, error) {
	/* 保存済みジョブを一覧取得する関数（新しい順、必要なら絞り込み） */
	files, err := filepath.Glob(filepath.Join(jobsDir(), "job_*.json"))
	if err != nil {
		return nil, err
	}
	sort.Sort(sort.Reverse(sort.StringSlice(files)))

	jobs := make([]jobmodel.Job, 0, len(files))
	for _, p := range files {
		b, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		var job jobmodel.Job
		if err := json.Unmarshal(b, &job); err != nil {
			continue
		}
		if status != "" && job.Status != status {
			continue
		}
		jobs = append(jobs, job)
		if limit > 0 && len(jobs) >= limit {
			break
		}
	}
	return jobs, nil
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	/* HTTPレスポンスにJSONを返す関数 */
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func artifactTypeFromPath(p string) string {
	switch strings.ToLower(filepath.Ext(p)) {
	case ".pptx":
		return "pptx"
	case ".png":
		return "png"
	case ".pdf":
		return "pdf"
	case ".json":
		return "json"
	default:
		return "file"
	}
}

func defaultMaxAttempts() int {
	if v := strings.TrimSpace(os.Getenv("JOB_MAX_ATTEMPTS")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return 2
}

func createJobHandler(w http.ResponseWriter, r *http.Request) {
	/* ジョブ作成APIのハンドラ関数 */
	if r.Method != http.MethodPost && r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}

	if r.Method == http.MethodGet {
		status := strings.TrimSpace(r.URL.Query().Get("status"))
		limit := 0
		if limitRaw := strings.TrimSpace(r.URL.Query().Get("limit")); limitRaw != "" {
			n, err := strconv.Atoi(limitRaw)
			if err != nil || n <= 0 {
				writeJSON(w, http.StatusBadRequest, map[string]any{"error": "limit must be a positive integer"})
				return
			}
			limit = n
		}

		jobs, err := listJobs(status, limit)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"jobs": jobs})
		return
	}

	defer r.Body.Close()

	var req createJobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json"})
		return
	}
	if strings.TrimSpace(req.PitchText) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "pitchText is required"})
		return
	}

	now := time.Now().UTC().Format(time.RFC3339)
	id := fmt.Sprintf("job_%d", time.Now().UnixNano())
	maxAttempts := defaultMaxAttempts()
	if req.MaxAttempts != nil {
		if *req.MaxAttempts <= 0 {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "maxAttempts must be > 0"})
			return
		}
		maxAttempts = *req.MaxAttempts
	}
	job := jobmodel.Job{
		ID:          id,
		Status:      jobmodel.StatusQueued,
		PitchText:   strings.TrimSpace(req.PitchText),
		Attempts:    0,
		MaxAttempts: maxAttempts,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := saveJob(job); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusAccepted, job)
}

func getJobHandler(w http.ResponseWriter, r *http.Request) {
	/* ジョブ取得・リトライAPIのハンドラ関数 */
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/jobs/")
	path = strings.Trim(path, "/")
	if path == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid path"})
		return
	}

	parts := strings.Split(path, "/")
	id := parts[0]
	job, err := loadJob(id)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			writeJSON(w, http.StatusNotFound, map[string]any{"error": "job not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	if len(parts) == 2 && parts[1] == "retry" {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
			return
		}
		job.Status = jobmodel.StatusQueued
		job.ArtifactPath = ""
		job.ErrorMessage = ""
		job.Attempts = 0
		if job.MaxAttempts <= 0 {
			job.MaxAttempts = defaultMaxAttempts()
		}
		job.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
		if err := saveJob(job); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, job)
		return
	}

	if len(parts) == 2 && parts[1] == "artifacts" {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
			return
		}
		artifacts := []Artifact{}
		if strings.TrimSpace(job.ArtifactPath) != "" {
			artifacts = append(artifacts, Artifact{
				ID:   fmt.Sprintf("%s_art_1", job.ID),
				Type: artifactTypeFromPath(job.ArtifactPath),
				Path: job.ArtifactPath,
			})
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"jobId":     job.ID,
			"artifacts": artifacts,
		})
		return
	}

	if len(parts) > 1 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid path"})
		return
	}
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	writeJSON(w, http.StatusOK, job)
}

func main() {
	// 起動時に保存先を用意し、APIルーティングを開始する。
	if err := ensureJobsDir(); err != nil {
		panic(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/jobs", createJobHandler)
	mux.HandleFunc("/jobs/", getJobHandler)

	addr := ":8080"
	if v := strings.TrimSpace(os.Getenv("API_ADDR")); v != "" {
		addr = v
	}
	fmt.Println("api listening on", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		panic(err)
	}
}
