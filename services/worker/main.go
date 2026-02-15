package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	jobmodel "slideagent/libs/jobmodel-go"
)

type clientResult struct {
	Status   string `json:"status"`
	PptxPath string `json:"pptxPath"`
}

func jobsDir() string {
	/* JOBS_DIR 環境変数が設定されていればそれを返し、未設定なら既定のパスを返す関数 */
	if v := strings.TrimSpace(os.Getenv("JOBS_DIR")); v != "" {
		return v
	}
	return filepath.Join("..", "..", "output", "jobs")
}

func ensureJobsDir() error {
	/* ジョブ保存用ディレクトリが存在しない場合は作成する関数 */
	return os.MkdirAll(jobsDir(), 0o755)
}

func loadJobFromPath(p string) (jobmodel.Job, error) {
	/* ジョブファイルのパスからJSONを読み込んでJob構造体に変換して返す関数 */
	var job jobmodel.Job
	b, err := os.ReadFile(p)
	if err != nil {
		return job, err
	}
	if err := json.Unmarshal(b, &job); err != nil {
		return job, err
	}
	return job, nil
}

func saveJob(job jobmodel.Job) error {
	/* Job構造体をJSONファイルとして保存する関数 */
	b, err := json.MarshalIndent(job, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(jobsDir(), job.ID+".json"), b, 0o644)
}

func findNextQueuedJob() (jobmodel.Job, error) {
	/* ジョブディレクトリから queued 状態のジョブを探して返す関数 */
	files, err := filepath.Glob(filepath.Join(jobsDir(), "job_*.json"))
	if err != nil {
		return jobmodel.Job{}, err
	}
	sort.Strings(files)
	for _, p := range files {
		job, err := loadJobFromPath(p)
		if err != nil {
			continue
		}
		if job.Status == jobmodel.StatusQueued {
			return job, nil
		}
	}
	return jobmodel.Job{}, os.ErrNotExist
}

func nowRFC3339() string {
	return time.Now().UTC().Format(time.RFC3339)
}

func findProjectRoot() (string, error) {
	if v := strings.TrimSpace(os.Getenv("SLIDEAGENT_ROOT")); v != "" {
		return v, nil
	}
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	cur := wd
	for {
		if _, err := os.Stat(filepath.Join(cur, ".git")); err == nil {
			return cur, nil
		}
		next := filepath.Dir(cur)
		if next == cur {
			break
		}
		cur = next
	}
	return "", errors.New("project root not found")
}

func runClient(pitch string) (string, error) {
	root, err := findProjectRoot()
	if err != nil {
		return "", err
	}
	cmd := exec.Command("go", "run", ".", "-pitch", pitch, "-json")
	cmd.Dir = filepath.Join(root, "client")
	cmd.Env = os.Environ()
	out, err := cmd.Output()
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			return "", fmt.Errorf("client failed (dir=%s args=%v): %s", cmd.Dir, cmd.Args, strings.TrimSpace(string(ee.Stderr)))
		}
		return "", err
	}

	var res clientResult
	if err := json.Unmarshal(out, &res); err != nil {
		return "", fmt.Errorf("failed to parse client output: %w output=%s", err, strings.TrimSpace(string(out)))
	}
	if strings.TrimSpace(res.PptxPath) == "" {
		return "", errors.New("client output missing pptxPath")
	}
	return res.PptxPath, nil
}

func processOne() (bool, error) {
	/* ジョブを1件処理する関数 */
	job, err := findNextQueuedJob()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, err
	}

	job.Status = jobmodel.StatusRunning
	job.UpdatedAt = nowRFC3339()
	if err := saveJob(job); err != nil {
		return true, err
	}

	pptxPath, err := runClient(job.PitchText)
	if err != nil {
		job.Attempts++
		if job.MaxAttempts <= 0 {
			job.MaxAttempts = 2
		}
		if job.Attempts < job.MaxAttempts {
			job.Status = jobmodel.StatusQueued
		} else {
			job.Status = jobmodel.StatusFailed
		}
		job.ErrorMessage = err.Error()
		job.UpdatedAt = nowRFC3339()
		_ = saveJob(job)
		return true, err
	}

	job.Status = jobmodel.StatusSucceeded
	job.ArtifactPath = pptxPath
	job.ErrorMessage = ""
	job.UpdatedAt = nowRFC3339()
	if err := saveJob(job); err != nil {
		return true, err
	}
	return true, nil
}

func main() {
	if err := ensureJobsDir(); err != nil {
		panic(err)
	}

	// ジョブが見つからない場合は一定時間待ってから再度探す
	interval := 3 * time.Second
	if v := strings.TrimSpace(os.Getenv("WORKER_POLL_SECONDS")); v != "" {
		if sec, err := time.ParseDuration(v + "s"); err == nil {
			interval = sec
		}
	}

	// ワーカー開始
	fmt.Println("worker started; jobsDir=", jobsDir())
	for {
		processed, err := processOne()
		if err != nil {
			fmt.Println("worker error:", err)
		}
		if !processed {
			time.Sleep(interval)
		}
	}
}
