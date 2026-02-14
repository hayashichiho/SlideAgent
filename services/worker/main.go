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
)

type Job struct {
	ID           string `json:"id"`
	Status       string `json:"status"`
	PitchText    string `json:"pitchText"`
	ArtifactPath string `json:"artifactPath,omitempty"`
	ErrorMessage string `json:"errorMessage,omitempty"`
	CreatedAt    string `json:"createdAt"`
	UpdatedAt    string `json:"updatedAt"`
}

type clientResult struct {
	Status   string `json:"status"`
	PptxPath string `json:"pptxPath"`
}

func jobsDir() string {
	if v := strings.TrimSpace(os.Getenv("JOBS_DIR")); v != "" {
		return v
	}
	return filepath.Join("..", "..", "output", "jobs")
}

func ensureJobsDir() error {
	return os.MkdirAll(jobsDir(), 0o755)
}

func loadJobFromPath(p string) (Job, error) {
	var job Job
	b, err := os.ReadFile(p)
	if err != nil {
		return job, err
	}
	if err := json.Unmarshal(b, &job); err != nil {
		return job, err
	}
	return job, nil
}

func saveJob(job Job) error {
	b, err := json.MarshalIndent(job, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(jobsDir(), job.ID+".json"), b, 0o644)
}

func findNextQueuedJob() (Job, error) {
	files, err := filepath.Glob(filepath.Join(jobsDir(), "job_*.json"))
	if err != nil {
		return Job{}, err
	}
	sort.Strings(files)
	for _, p := range files {
		job, err := loadJobFromPath(p)
		if err != nil {
			continue
		}
		if job.Status == "queued" {
			return job, nil
		}
	}
	return Job{}, os.ErrNotExist
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
	job, err := findNextQueuedJob()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, err
	}

	job.Status = "running"
	job.UpdatedAt = nowRFC3339()
	if err := saveJob(job); err != nil {
		return true, err
	}

	pptxPath, err := runClient(job.PitchText)
	if err != nil {
		job.Status = "failed"
		job.ErrorMessage = err.Error()
		job.UpdatedAt = nowRFC3339()
		_ = saveJob(job)
		return true, err
	}

	job.Status = "succeeded"
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

	interval := 3 * time.Second
	if v := strings.TrimSpace(os.Getenv("WORKER_POLL_SECONDS")); v != "" {
		if sec, err := time.ParseDuration(v + "s"); err == nil {
			interval = sec
		}
	}

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
