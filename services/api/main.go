package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Job struct {
	// Job はジョブ状態を表す永続化データ。
	ID           string `json:"id"`
	Status       string `json:"status"`
	PitchText    string `json:"pitchText"`
	ArtifactPath string `json:"artifactPath,omitempty"`
	ErrorMessage string `json:"errorMessage,omitempty"`
	CreatedAt    string `json:"createdAt"`
	UpdatedAt    string `json:"updatedAt"`
}

type createJobRequest struct {
	// createJobRequest はジョブ作成APIの入力。
	PitchText string `json:"pitchText"`
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

func saveJob(job Job) error {
	/* Job を JSON ファイルとして保存する関数 */
	b, err := json.MarshalIndent(job, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(jobPath(job.ID), b, 0o644)
}

func loadJob(id string) (Job, error) {
	/* ジョブIDからJSONファイルを読み込んでJob構造体に変換して返す関数 */
	var job Job
	b, err := os.ReadFile(jobPath(id))
	if err != nil {
		return job, err
	}
	if err := json.Unmarshal(b, &job); err != nil {
		return job, err
	}
	return job, nil
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	/* HTTPレスポンスにJSONを返す関数 */
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func createJobHandler(w http.ResponseWriter, r *http.Request) {
	/* ジョブ作成APIのハンドラ関数 */
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
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
	job := Job{
		ID:        id,
		Status:    "queued",
		PitchText: strings.TrimSpace(req.PitchText),
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := saveJob(job); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusAccepted, job)
}

func getJobHandler(w http.ResponseWriter, r *http.Request) {
	/* ジョブ取得APIのハンドラ関数 */
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}

	id := strings.TrimPrefix(r.URL.Path, "/jobs/")
	if id == "" || strings.Contains(id, "/") {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid job id"})
		return
	}

	job, err := loadJob(id)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			writeJSON(w, http.StatusNotFound, map[string]any{"error": "job not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
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
