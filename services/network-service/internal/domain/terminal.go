package domain

import "time"

type TerminalExecuteRequest struct {
	Command string            `json:"command"`
	Params  map[string]string `json:"params,omitempty"`
}

type TerminalExecuteResult struct {
	Command string              `json:"command"`
	Rows    []map[string]string `json:"rows"`
}

type MikroTikCommandAuditListParams struct {
	RouterID string
	Page     int
	PageSize int
	Status   string
}

type MikroTikCommandAuditListResult struct {
	Data       []MikroTikCommandAuditLog
	Total      int64
	Page       int
	PageSize   int
	TotalPages int
}

type MikroTikCommandAuditLog struct {
	ID           string    `json:"id,omitempty"`
	TenantID     string    `json:"tenant_id,omitempty"`
	RouterID     string    `json:"router_id,omitempty"`
	UserID       string    `json:"user_id,omitempty"`
	Action       string    `json:"action"`
	Command      string    `json:"command"`
	TargetType   string    `json:"target_type,omitempty"`
	TargetID     string    `json:"target_id,omitempty"`
	Status       string    `json:"status"`
	ErrorMessage string    `json:"error_message,omitempty"`
	RemoteAddr   string    `json:"remote_addr,omitempty"`
	CreatedAt    time.Time `json:"created_at,omitempty"`
}
