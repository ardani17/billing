package domain

import (
	"encoding/json"
	"time"
)

// =============================================================================
// NotificationConfig - konfigurasi provider notifikasi per tenant per channel
// =============================================================================

// NotificationConfig merepresentasikan konfigurasi provider notifikasi per tenant per channel.
type NotificationConfig struct {
	ID          string          `json:"id"`
	TenantID    string          `json:"tenant_id"`
	Channel     Channel         `json:"channel"`
	Provider    string          `json:"provider"`
	Credentials json.RawMessage `json:"credentials"`
	IsEnabled   bool            `json:"is_enabled"`
	Priority    int             `json:"priority"`
	Settings    ConfigSettings  `json:"settings"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

// ConfigSettings berisi pengaturan umum notifikasi per tenant.
type ConfigSettings struct {
	ChannelPriority   []Channel `json:"channel_priority"`
	QuietHoursStart   string    `json:"quiet_hours_start"`
	QuietHoursEnd     string    `json:"quiet_hours_end"`
	Timezone          string    `json:"timezone"`
	DailyLimitPerCust int       `json:"daily_limit_per_customer"`
	CooldownMinutes   int       `json:"cooldown_minutes"`
}

// WhatsAppCredentials berisi credential untuk provider WhatsApp (Fonnte).
type WhatsAppCredentials struct {
	APIToken     string `json:"api_token"`
	SenderNumber string `json:"sender_number"`
}

// SMSCredentials berisi credential untuk provider SMS (Zenziva).
type SMSCredentials struct {
	APIKey  string `json:"api_key"`
	UserKey string `json:"user_key"`
}

// EmailCredentials berisi credential untuk provider Email SMTP.
type EmailCredentials struct {
	SMTPHost  string `json:"smtp_host"`
	SMTPPort  int    `json:"smtp_port"`
	Username  string `json:"username"`
	Password  string `json:"password"`
	FromName  string `json:"from_name"`
	FromEmail string `json:"from_email"`
}

// =============================================================================
// NotificationTemplate - template notifikasi per tenant
// =============================================================================

// NotificationTemplate merepresentasikan template notifikasi per tenant.
type NotificationTemplate struct {
	ID               string           `json:"id"`
	TenantID         string           `json:"tenant_id"`
	Slug             string           `json:"slug"`
	Name             string           `json:"name"`
	Category         TemplateCategory `json:"category"`
	EventType        string           `json:"event_type,omitempty"`
	Channels         []Channel        `json:"channels"`
	BodyWhatsApp     string           `json:"body_whatsapp,omitempty"`
	BodySMS          string           `json:"body_sms,omitempty"`
	BodyEmailSubject string           `json:"body_email_subject,omitempty"`
	BodyEmailHTML    string           `json:"body_email_html,omitempty"`
	Variables        []string         `json:"variables"`
	IsActive         bool             `json:"is_active"`
	IsDefault        bool             `json:"is_default"`
	CreatedAt        time.Time        `json:"created_at"`
	UpdatedAt        time.Time        `json:"updated_at"`
}

// =============================================================================
// NotificationLog - catatan pengiriman notifikasi
// =============================================================================

// NotificationLog merepresentasikan catatan pengiriman notifikasi.
type NotificationLog struct {
	ID           string                 `json:"id"`
	TenantID     string                 `json:"tenant_id"`
	CustomerID   string                 `json:"customer_id"`
	TemplateID   string                 `json:"template_id,omitempty"`
	Channel      Channel                `json:"channel"`
	Provider     string                 `json:"provider"`
	Recipient    string                 `json:"recipient"`
	Subject      string                 `json:"subject,omitempty"`
	Body         string                 `json:"body"`
	Status       LogStatus              `json:"status"`
	RetryCount   int                    `json:"retry_count"`
	MaxRetries   int                    `json:"max_retries"`
	ErrorMessage string                 `json:"error_message,omitempty"`
	DedupKey     string                 `json:"dedup_key,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	SentAt       *time.Time             `json:"sent_at,omitempty"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`

	// Field JOIN (opsional, dari kueri)
	CustomerName string `json:"customer_name,omitempty"`
	TemplateName string `json:"template_name,omitempty"`
}

// =============================================================================
// LogListParams & LogListResult - parameter dan hasil paginasi log
// =============================================================================

// LogListParams berisi parameter untuk kueri daftar log notifikasi.
type LogListParams struct {
	TenantID   string     `json:"tenant_id"`
	Channel    Channel    `json:"channel,omitempty"`
	Status     LogStatus  `json:"status,omitempty"`
	CustomerID string     `json:"customer_id,omitempty"`
	TemplateID string     `json:"template_id,omitempty"`
	DateFrom   *time.Time `json:"date_from,omitempty"`
	DateTo     *time.Time `json:"date_to,omitempty"`
	Page       int        `json:"page"`
	PageSize   int        `json:"page_size"`
}

// LogListResult berisi hasil kueri daftar log notifikasi dengan metadata paginasi.
type LogListResult struct {
	Data       []*NotificationLog `json:"data"`
	Total      int64              `json:"total"`
	Page       int                `json:"page"`
	PageSize   int                `json:"page_size"`
	TotalPages int                `json:"total_pages"`
}
