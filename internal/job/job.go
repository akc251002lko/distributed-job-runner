package job

import (
	"encoding/json"
	"time"
)

type Job struct {
	ID string `json:"id"`
	Type string `json:"type"`
	Payload json.RawMessage `json:"payload"`
	Status string `json:"status"`
	Attempts int `json:"attempts"`
	MaxAttempts int `json:"max_attempts"`
	RunAt time.Time `json:"run_at"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	LeaseExpiresAt *time.Time `json:"lease_expires_at,omitempty"`
	LastError string `json:"last_error,omitempty"`
	WorkerID string `json:"worker_id,omitempty"`
	LeaseToken string `json:"-"`
}
