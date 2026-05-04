package domain

import "time"

const (
	StaticIPStatusActive   = "active"
	StaticIPStatusIsolated = "isolated"
	StaticIPStatusDisabled = "disabled"
)

type StaticIPAssignment struct {
	ID          string
	TenantID    string
	RouterID    string
	CustomerID  string
	IPAddress   string
	AddressList string
	QueueName   string
	RateLimit   string
	Comment     string
	Status      string
	LastSyncAt  *time.Time
	SyncStatus  string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   *time.Time
}

type StaticIPAssignmentResponse struct {
	ID          string     `json:"id"`
	RouterID    string     `json:"router_id"`
	CustomerID  string     `json:"customer_id,omitempty"`
	IPAddress   string     `json:"ip_address"`
	AddressList string     `json:"address_list"`
	QueueName   string     `json:"queue_name,omitempty"`
	RateLimit   string     `json:"rate_limit,omitempty"`
	Comment     string     `json:"comment"`
	Status      string     `json:"status"`
	LastSyncAt  *time.Time `json:"last_sync_at,omitempty"`
	SyncStatus  string     `json:"sync_status"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

func (a *StaticIPAssignment) ToResponse() *StaticIPAssignmentResponse {
	if a == nil {
		return nil
	}
	return &StaticIPAssignmentResponse{
		ID:          a.ID,
		RouterID:    a.RouterID,
		CustomerID:  a.CustomerID,
		IPAddress:   a.IPAddress,
		AddressList: a.AddressList,
		QueueName:   a.QueueName,
		RateLimit:   a.RateLimit,
		Comment:     a.Comment,
		Status:      a.Status,
		LastSyncAt:  a.LastSyncAt,
		SyncStatus:  a.SyncStatus,
		CreatedAt:   a.CreatedAt,
		UpdatedAt:   a.UpdatedAt,
	}
}

type StaticIPAssignmentListParams struct {
	RouterID string
	Page     int
	PageSize int
	Search   string
}

type StaticIPAssignmentListResult struct {
	Data       []*StaticIPAssignmentResponse `json:"items"`
	Total      int64                         `json:"total"`
	Page       int                           `json:"page"`
	PageSize   int                           `json:"page_size"`
	TotalPages int                           `json:"total_pages"`
}

type CreateStaticIPAssignmentRequest struct {
	CustomerID string `json:"customer_id,omitempty"`
	IPAddress  string `json:"ip_address" validate:"required"`
	QueueName  string `json:"queue_name,omitempty"`
	RateLimit  string `json:"rate_limit,omitempty"`
	Comment    string `json:"comment,omitempty"`
}

type UpdateStaticIPAssignmentRequest struct {
	CustomerID *string `json:"customer_id,omitempty"`
	IPAddress  *string `json:"ip_address,omitempty"`
	QueueName  *string `json:"queue_name,omitempty"`
	RateLimit  *string `json:"rate_limit,omitempty"`
	Comment    *string `json:"comment,omitempty"`
	Status     *string `json:"status,omitempty"`
}

type DeleteStaticIPAssignmentRequest struct {
	ConfirmIP string `json:"confirm_ip" validate:"required"`
}
