package domain

import (
	"encoding/json"

	"github.com/gofiber/fiber/v2"
)

// =============================================================================
// Permintaan DTO - payload dari HTTP permintaan
// =============================================================================

// UpdateConfigRequest adalah payload untuk PUT /api/v1/notifications/config.
// Digunakan untuk mengatur konfigurasi provider per channel.
type UpdateConfigRequest struct {
	Channel     Channel         `json:"channel"`
	Provider    string          `json:"provider"`
	Credentials json.RawMessage `json:"credentials"`
	IsEnabled   bool            `json:"is_enabled"`
	Priority    int             `json:"priority"`
}

// CreateTemplateRequest adalah payload untuk POST /api/v1/notifications/templates.
// Digunakan untuk membuat template notifikasi baru.
type CreateTemplateRequest struct {
	Slug             string           `json:"slug"`
	Name             string           `json:"name"`
	Category         TemplateCategory `json:"category"`
	EventType        string           `json:"event_type,omitempty"`
	Channels         []Channel        `json:"channels"`
	BodyWhatsApp     string           `json:"body_whatsapp,omitempty"`
	BodySMS          string           `json:"body_sms,omitempty"`
	BodyEmailSubject string           `json:"body_email_subject,omitempty"`
	BodyEmailHTML    string           `json:"body_email_html,omitempty"`
	Variables        []string         `json:"variables,omitempty"`
}

// UpdateTemplateRequest adalah payload untuk PUT /api/v1/notifications/templates/:id.
// Digunakan untuk memperbarui template notifikasi yang sudah ada.
type UpdateTemplateRequest struct {
	Name             string    `json:"name,omitempty"`
	Channels         []Channel `json:"channels,omitempty"`
	BodyWhatsApp     string    `json:"body_whatsapp,omitempty"`
	BodySMS          string    `json:"body_sms,omitempty"`
	BodyEmailSubject string    `json:"body_email_subject,omitempty"`
	BodyEmailHTML    string    `json:"body_email_html,omitempty"`
	IsActive         *bool     `json:"is_active,omitempty"`
}

// TestSendRequest adalah payload untuk POST /api/v1/notifications/test.
// Digunakan untuk mengirim notifikasi percobaan ke recipient tertentu.
type TestSendRequest struct {
	TemplateID string  `json:"template_id"`
	Channel    Channel `json:"channel"`
	Recipient  string  `json:"recipient"`
}

// ManualSendRequest adalah payload untuk POST /api/v1/notifications/send.
// Digunakan untuk mengirim notifikasi manual ke pelanggan tertentu.
type ManualSendRequest struct {
	CustomerID    string  `json:"customer_id"`
	TemplateID    string  `json:"template_id,omitempty"`
	Channel       Channel `json:"channel"`
	CustomBody    string  `json:"custom_body,omitempty"`
	CustomSubject string  `json:"custom_subject,omitempty"`
}

// ResendRequest adalah payload untuk POST /api/v1/notifications/logs/:id/resend.
// ID log diambil dari URL parameter, struct ini hanya sebagai penanda.
type ResendRequest struct {
	LogID string `json:"log_id"`
}

// =============================================================================
// API Respons - format standar respons API (reuse pattern dari billing-api)
// =============================================================================

// APIResponse adalah format standar respons API.
type APIResponse struct {
	// Success menunjukkan apakah permintaan berhasil
	Success bool `json:"success"`

	// Data berisi data respons jika sukses
	Data interface{} `json:"data,omitempty"`

	// Error berisi detail error jika gagal
	Error *APIError `json:"error,omitempty"`
}

// APIError adalah format standar error API.
type APIError struct {
	// Code adalah kode error (contoh: VALIDATION_ERROR, PROVIDER_NOT_CONFIGURED)
	Code string `json:"code"`

	// Message adalah pesan error yang bisa ditampilkan ke pengguna
	Message string `json:"message"`

	// Details berisi detail error per field untuk validation error
	Details []FieldError `json:"details,omitempty"`
}

// FieldError adalah detail error per field untuk validation error.
type FieldError struct {
	// Field adalah nama field yang tidak valid
	Field string `json:"field"`

	// Message adalah pesan error untuk field tersebut
	Message string `json:"message"`
}

// PaginatedData membungkus data dengan metadata paginasi.
type PaginatedData struct {
	// Items berisi data hasil kueri
	Items interface{} `json:"items"`

	// Total adalah jumlah total record
	Total int64 `json:"total"`

	// Page adalah halaman saat ini
	Page int `json:"page"`

	// PageSize adalah jumlah item per halaman
	PageSize int `json:"page_size"`

	// TotalPages adalah jumlah total halaman
	TotalPages int `json:"total_pages"`
}

// =============================================================================
// Respons Fungsi bantus - fungsi pembantu untuk membuat respons API
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
