package types

import "time"

type IndexingJob struct {
	JobID      string          `json:"job_id"`
	Type       string          `json:"type"`
	CreatedAt  time.Time       `json:"created_at"`
	Payload    IndexingPayload `json:"payload"`
	RetryCount int             `json:"retry_count"`
}

type IndexingPayload struct {
	DocID    string            `json:"doc_id"`
	UserID   string            `json:"user_id"`
	FilePath string            `json:"file_path"`
	FileName string            `json:"file_name"`
	FileSize int64             `json:"size"`
	Metadata map[string]string `json:"metadata"`
}
