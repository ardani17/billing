package domain

import "time"

// AuditLog merepresentasikan catatan perubahan pada entitas.
type AuditLog struct {
	ID         string                 `json:"id"`
	TenantID   string                 `json:"tenant_id"`
	EntityType string                 `json:"entity_type"`
	EntityID   string                 `json:"entity_id"`
	Action     string                 `json:"action"`
	ActorID    string                 `json:"actor_id"`
	ActorName  string                 `json:"actor_name"`
	Changes    map[string]interface{} `json:"changes,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt  time.Time              `json:"created_at"`
}
