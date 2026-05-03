// Package usecase berisi business logic untuk billing-api.
// AuditLogger menyediakan structured logging untuk semua event autentikasi.
// Log disimpan dalam format JSON via zerolog dan TIDAK PERNAH mencatat
// data sensitif seperti password, token, atau hash.
package usecase

import (
	"strings"

	"github.com/rs/zerolog"
)

// AuditEventType mendefinisikan tipe event autentikasi yang di-audit.
type AuditEventType string

// Daftar semua event autentikasi yang didukung oleh AuditLogger.
const (
	AuditEventLogin            AuditEventType = "login"
	AuditEventLogout           AuditEventType = "logout"
	AuditEventRegister         AuditEventType = "register"
	AuditEventVerifyEmail      AuditEventType = "verify-email"
	AuditEventForgotPassword   AuditEventType = "forgot-password"
	AuditEventResetPassword    AuditEventType = "reset-password"
	AuditEventChangePassword   AuditEventType = "change-password"
	AuditEventUserCreated      AuditEventType = "user-created"
	AuditEventUserDeactivated  AuditEventType = "user-deactivated"
	AuditEventUserDeleted      AuditEventType = "user-deleted"
	AuditEventImpersonateStart AuditEventType = "impersonate-start"
	AuditEventImpersonateStop  AuditEventType = "impersonate-stop"
)

// ValidAuditEvents berisi semua event type yang valid untuk audit logging.
var ValidAuditEvents = []AuditEventType{
	AuditEventLogin,
	AuditEventLogout,
	AuditEventRegister,
	AuditEventVerifyEmail,
	AuditEventForgotPassword,
	AuditEventResetPassword,
	AuditEventChangePassword,
	AuditEventUserCreated,
	AuditEventUserDeactivated,
	AuditEventUserDeleted,
	AuditEventImpersonateStart,
	AuditEventImpersonateStop,
}

// sensitiveKeys berisi daftar key metadata yang dianggap sensitif.
// Key-key ini akan difilter dan TIDAK PERNAH dicatat ke audit log.
var sensitiveKeys = []string{
	"password",
	"token",
	"hash",
	"secret",
	"credential",
	"password_hash",
	"password_confirmation",
	"current_password",
	"new_password",
	"refresh_token",
	"access_token",
	"id_token",
	"token_hash",
}

// AuditLogger menyediakan structured audit logging untuk event autentikasi.
// Menggunakan zerolog untuk output JSON terstruktur yang bisa di-query
// berdasarkan tenant_id, user_id, event_type, dan date range.
type AuditLogger struct {
	logger zerolog.Logger
}

// NewAuditLogger membuat instance baru AuditLogger dengan zerolog.Logger yang diberikan.
func NewAuditLogger(logger zerolog.Logger) *AuditLogger {
	return &AuditLogger{
		logger: logger,
	}
}

// LogEvent mencatat event autentikasi ke audit log dalam format JSON terstruktur.
// Parameter:
//   - eventType: tipe event (login, logout, register, dll)
//   - userID: ID user yang melakukan aksi
//   - tenantID: ID tenant tempat user berada
//   - ipAddress: alamat IP client
//   - userAgent: User-Agent header dari client
//   - result: hasil event (success/failure)
//   - metadata: data tambahan (opsional, key sensitif akan difilter)
//
// Data sensitif (password, token, hash) TIDAK PERNAH dicatat ke log.
func (a *AuditLogger) LogEvent(eventType AuditEventType, userID, tenantID, ipAddress, userAgent, result string, metadata map[string]string) {
	// Buat log event dengan field wajib
	event := a.logger.Info().
		Str("audit", "auth").
		Str("event_type", string(eventType)).
		Str("user_id", userID).
		Str("tenant_id", tenantID).
		Str("ip_address", ipAddress).
		Str("user_agent", userAgent).
		Str("result", result)

	// Tambahkan metadata yang sudah difilter dari data sensitif
	if metadata != nil {
		sanitized := sanitizeMetadata(metadata)
		for k, v := range sanitized {
			event = event.Str(k, v)
		}
	}

	// Tulis log entry (timestamp otomatis ditambahkan oleh zerolog)
	event.Msg("audit_event")
}

// sanitizeMetadata memfilter key-key sensitif dari metadata.
// Key yang mengandung kata-kata sensitif (password, token, hash, dll)
// akan dihapus untuk mencegah kebocoran data sensitif ke log.
func sanitizeMetadata(metadata map[string]string) map[string]string {
	if metadata == nil {
		return nil
	}

	sanitized := make(map[string]string, len(metadata))
	for k, v := range metadata {
		if isSensitiveKey(k) {
			continue
		}
		// Juga filter value yang terlihat seperti token/hash (hex string panjang)
		if isSensitiveValue(v) {
			continue
		}
		sanitized[k] = v
	}
	return sanitized
}

// isSensitiveKey memeriksa apakah key metadata mengandung kata sensitif.
func isSensitiveKey(key string) bool {
	lower := strings.ToLower(key)
	for _, sensitive := range sensitiveKeys {
		if strings.Contains(lower, sensitive) {
			return true
		}
	}
	return false
}

// isSensitiveValue memeriksa apakah value terlihat seperti data sensitif.
// Mendeteksi string hex panjang yang kemungkinan token atau hash.
func isSensitiveValue(value string) bool {
	// Bcrypt hash selalu diawali dengan "$2a$" atau "$2b$"
	if strings.HasPrefix(value, "$2a$") || strings.HasPrefix(value, "$2b$") {
		return true
	}
	// Hex string panjang (>= 64 karakter) kemungkinan SHA-256 hash atau token
	if len(value) >= 64 && isHexString(value) {
		return true
	}
	return false
}

// isHexString memeriksa apakah string hanya berisi karakter hexadecimal.
func isHexString(s string) bool {
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}
