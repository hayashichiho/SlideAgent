package jobmodel

const (
	StatusQueued    = "queued"
	StatusRunning   = "running"
	StatusSucceeded = "succeeded"
	StatusFailed    = "failed"
)

// Job はジョブ状態を表す永続化データ。
type Job struct {
	ID           string `json:"id"`
	Status       string `json:"status"`
	PitchText    string `json:"pitchText"`
	ArtifactPath string `json:"artifactPath,omitempty"`
	ErrorMessage string `json:"errorMessage,omitempty"`
	CreatedAt    string `json:"createdAt"`
	UpdatedAt    string `json:"updatedAt"`
}
