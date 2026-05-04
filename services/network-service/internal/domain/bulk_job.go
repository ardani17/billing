package domain

import "time"

type MikroTikBulkAction string

const (
	MikroTikBulkActionBackup        MikroTikBulkAction = "backup"
	MikroTikBulkActionFirmwareCheck MikroTikBulkAction = "firmware_check"
	MikroTikBulkActionPPPoESync     MikroTikBulkAction = "pppoe_sync"
)

type MikroTikBulkJobStatus string

const (
	MikroTikBulkJobQueued        MikroTikBulkJobStatus = "queued"
	MikroTikBulkJobRunning       MikroTikBulkJobStatus = "running"
	MikroTikBulkJobSucceeded     MikroTikBulkJobStatus = "succeeded"
	MikroTikBulkJobPartialFailed MikroTikBulkJobStatus = "partial_failed"
	MikroTikBulkJobFailed        MikroTikBulkJobStatus = "failed"
)

type CreateMikroTikBulkJobRequest struct {
	Action    MikroTikBulkAction `json:"action" validate:"required"`
	RouterIDs []string           `json:"router_ids"`
	Scope     string             `json:"scope"`
}

type MikroTikBulkJob struct {
	ID           string                  `json:"id"`
	TenantID     string                  `json:"tenant_id"`
	Action       MikroTikBulkAction      `json:"action"`
	Status       MikroTikBulkJobStatus   `json:"status"`
	RouterIDs    []string                `json:"router_ids"`
	TotalCount   int                     `json:"total_count"`
	SuccessCount int                     `json:"success_count"`
	FailedCount  int                     `json:"failed_count"`
	Results      []MikroTikBulkJobResult `json:"results"`
	ErrorMessage string                  `json:"error_message,omitempty"`
	RequestedBy  string                  `json:"requested_by,omitempty"`
	StartedAt    *time.Time              `json:"started_at,omitempty"`
	FinishedAt   *time.Time              `json:"finished_at,omitempty"`
	CreatedAt    time.Time               `json:"created_at"`
	UpdatedAt    time.Time               `json:"updated_at"`
}

type MikroTikBulkJobResult struct {
	RouterID   string         `json:"router_id"`
	RouterName string         `json:"router_name"`
	Action     string         `json:"action"`
	Status     string         `json:"status"`
	Message    string         `json:"message,omitempty"`
	Details    map[string]any `json:"details,omitempty"`
	StartedAt  time.Time      `json:"started_at"`
	FinishedAt time.Time      `json:"finished_at"`
}

type MikroTikBulkJobListParams struct {
	Page     int
	PageSize int
	Action   string
	Status   string
}

type MikroTikBulkJobListResult struct {
	Data       []MikroTikBulkJob `json:"data"`
	Total      int64             `json:"total"`
	Page       int               `json:"page"`
	PageSize   int               `json:"page_size"`
	TotalPages int               `json:"total_pages"`
}

type CreateMikroTikBulkJobInput struct {
	TenantID    string
	Action      MikroTikBulkAction
	Status      MikroTikBulkJobStatus
	RouterIDs   []string
	TotalCount  int
	RequestedBy string
	StartedAt   *time.Time
}

type UpdateMikroTikBulkJobResultInput struct {
	ID           string
	Status       MikroTikBulkJobStatus
	SuccessCount int
	FailedCount  int
	Results      []MikroTikBulkJobResult
	ErrorMessage string
	FinishedAt   *time.Time
}
