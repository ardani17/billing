package domain

import (
	"time"

	"github.com/gofiber/fiber/v2"
)

// =============================================================================
// Request DTO — payload dari HTTP request
// =============================================================================

// CreateRouterRequest adalah payload untuk POST /api/v1/mikrotik/routers.
// Digunakan untuk mendaftarkan router MikroTik baru ke sistem.
type CreateRouterRequest struct {
	Name                   string   `json:"name" validate:"required,min=1,max=100"`
	Host                   string   `json:"host" validate:"required,max=255"`
	Port                   int      `json:"port" validate:"omitempty,min=1,max=65535"`
	Username               string   `json:"username" validate:"required,max=100"`
	Password               string   `json:"password" validate:"required"`
	UseSSL                 bool     `json:"use_ssl"`
	ServiceTypes           []string `json:"service_types" validate:"omitempty,dive,oneof=pppoe hotspot dhcp_binding static"`
	HealthCheckIntervalSec int      `json:"health_check_interval_sec" validate:"omitempty,min=10,max=3600"`
	Notes                  string   `json:"notes,omitempty"`
}

// UpdateRouterRequest adalah payload untuk PUT /api/v1/mikrotik/routers/:id.
// Field bersifat opsional — hanya field yang dikirim yang akan diupdate.
type UpdateRouterRequest struct {
	Name                   string `json:"name" validate:"omitempty,min=1,max=100"`
	Host                   string `json:"host" validate:"omitempty,max=255"`
	Port                   *int   `json:"port" validate:"omitempty,min=1,max=65535"`
	Username               string `json:"username" validate:"omitempty,max=100"`
	Password               string `json:"password,omitempty"`
	UseSSL                 *bool  `json:"use_ssl,omitempty"`
	HealthCheckIntervalSec *int   `json:"health_check_interval_sec" validate:"omitempty,min=10,max=3600"`
	Status                 string `json:"status" validate:"omitempty,oneof=online offline maintenance"`
	Notes                  string `json:"notes,omitempty"`
}

// RebootRequest adalah payload untuk POST /api/v1/mikrotik/routers/:id/reboot.
// Memerlukan konfirmasi nama router untuk mencegah reboot tidak sengaja.
type RebootRequest struct {
	ConfirmationName string `json:"confirmation_name" validate:"required"`
}

// =============================================================================
// Response DTO — format respons untuk router operations
// =============================================================================

// RouterResponse adalah respons untuk operasi CRUD router.
type RouterResponse struct {
	Router  *Router `json:"router"`
	Warning string  `json:"warning,omitempty"`
}

// RouterDetailResponse adalah respons untuk GET router detail.
// Menyertakan live metrics jika router sedang online.
type RouterDetailResponse struct {
	Router      *Router        `json:"router"`
	LiveMetrics *RouterMetrics `json:"live_metrics,omitempty"`
}

// RouterListParams berisi parameter untuk list router dengan paginasi.
type RouterListParams struct {
	TenantID string
	Page     int
	PageSize int
	Status   string
	Search   string
}

// RouterListResult berisi hasil list router dengan metadata paginasi.
type RouterListResult struct {
	Data       []*Router `json:"data"`
	Total      int64     `json:"total"`
	Page       int       `json:"page"`
	PageSize   int       `json:"page_size"`
	TotalPages int       `json:"total_pages"`
}

// =============================================================================
// Event Payloads — payload untuk event antar service via Redis queue
// =============================================================================

// RouterOfflinePayload adalah payload event mikrotik.router_offline.
type RouterOfflinePayload struct {
	RouterID     string    `json:"router_id"`
	RouterName   string    `json:"router_name"`
	TenantID     string    `json:"tenant_id"`
	LastOnlineAt time.Time `json:"last_online_at"`
}

// RouterOnlinePayload adalah payload event mikrotik.router_online.
type RouterOnlinePayload struct {
	RouterID         string        `json:"router_id"`
	RouterName       string        `json:"router_name"`
	TenantID         string        `json:"tenant_id"`
	DowntimeDuration time.Duration `json:"downtime_duration"`
}

// RouterRebootPayload adalah payload event mikrotik.router_unexpected_reboot.
type RouterRebootPayload struct {
	RouterID              string `json:"router_id"`
	RouterName            string `json:"router_name"`
	TenantID              string `json:"tenant_id"`
	PreviousUptimeSeconds int64  `json:"previous_uptime_seconds"`
	CurrentUptimeSeconds  int64  `json:"current_uptime_seconds"`
}

// =============================================================================
// API Response — format standar respons API (reuse pattern dari notification-service)
// =============================================================================

// APIResponse adalah format standar respons API.
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *APIError   `json:"error,omitempty"`
}

// APIError adalah format standar error API.
type APIError struct {
	Code    string       `json:"code"`
	Message string       `json:"message"`
	Details []FieldError `json:"details,omitempty"`
}

// FieldError adalah detail error per field untuk validation error.
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// PaginatedData membungkus data dengan metadata paginasi.
type PaginatedData struct {
	Items      interface{} `json:"items"`
	Total      int64       `json:"total"`
	Page       int         `json:"page"`
	PageSize   int         `json:"page_size"`
	TotalPages int         `json:"total_pages"`
}

// =============================================================================
// Response Helpers — fungsi pembantu untuk membuat respons API
// =============================================================================

// SuccessResponse mengembalikan respons sukses JSON dengan format standar.
func SuccessResponse(c *fiber.Ctx, status int, data interface{}) error {
	return c.Status(status).JSON(APIResponse{
		Success: true,
		Data:    data,
	})
}

// ErrorResponse mengembalikan respons error JSON dengan format standar.
func ErrorResponse(c *fiber.Ctx, status int, code, message string, details ...FieldError) error {
	resp := APIResponse{
		Success: false,
		Error: &APIError{
			Code:    code,
			Message: message,
		},
	}
	if len(details) > 0 {
		resp.Error.Details = details
	}
	return c.Status(status).JSON(resp)
}

// PaginatedResponse mengembalikan respons sukses JSON dengan metadata paginasi.
func PaginatedResponse(c *fiber.Ctx, status int, data interface{}, total int64, page, pageSize, totalPages int) error {
	return c.Status(status).JSON(APIResponse{
		Success: true,
		Data: PaginatedData{
			Items:      data,
			Total:      total,
			Page:       page,
			PageSize:   pageSize,
			TotalPages: totalPages,
		},
	})
}
