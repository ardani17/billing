package usecase

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/rs/zerolog"
	"pgregory.net/rapid"
)

// Feature: auth-rbac, Property 19: Audit Log Completeness
// **Validates: Requirements 17.1, 17.3**
//
// For any auth event (login, logout, register, verify-email, forgot-password,
// reset-password, change-password, user-created, user-deactivated, user-deleted,
// impersonate-start, impersonate-stop), the audit log entry SHALL contain:
// timestamp, user_id, tenant_id, ip_address, user_agent, and event result.
// The audit log SHALL NOT contain passwords, tokens, or password hashes.
func TestProperty_AuditLogCompleteness(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Capture output zerolog ke buffer untuk parsing JSON
		var buf bytes.Buffer
		logger := zerolog.New(&buf).With().Timestamp().Logger()
		auditLogger := NewAuditLogger(logger)

		// Generate random event type dari daftar valid
		eventType := rapid.SampledFrom(ValidAuditEvents).Draw(t, "eventType")

		// Generate random field values
		userID := rapid.StringMatching(`[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`).Draw(t, "userID")
		tenantID := rapid.StringMatching(`[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`).Draw(t, "tenantID")
		ipAddress := rapid.SampledFrom([]string{
			"127.0.0.1",
			"192.168.1.1",
			"10.0.0.1",
			"::1",
			"2001:db8::1",
		}).Draw(t, "ipAddress")
		userAgent := rapid.SampledFrom([]string{
			"Mozilla/5.0 (Windows NT 10.0; Win64; x64)",
			"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)",
			"curl/7.68.0",
			"ISPBoss-Mobile/1.0",
		}).Draw(t, "userAgent")
		result := rapid.SampledFrom([]string{"success", "failure"}).Draw(t, "result")

		// Generate metadata yang mungkin mengandung data sensitif
		metadata := map[string]string{
			"target_user_id": "some-user-id",
		}

		// Tambahkan data sensitif secara random untuk memastikan difilter
		includeSensitive := rapid.Bool().Draw(t, "includeSensitive")
		if includeSensitive {
			sensitiveField := rapid.SampledFrom([]string{
				"password",
				"token",
				"password_hash",
				"refresh_token",
				"access_token",
				"current_password",
				"new_password",
				"token_hash",
				"secret_key",
			}).Draw(t, "sensitiveField")
			metadata[sensitiveField] = "sensitive-value-should-not-appear"
		}

		// Reset buffer dan log event
		buf.Reset()
		auditLogger.LogEvent(eventType, userID, tenantID, ipAddress, userAgent, result, metadata)

		// Parse output JSON
		output := buf.String()
		if output == "" {
			t.Fatal("audit log output kosong")
		}

		var logEntry map[string]interface{}
		if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &logEntry); err != nil {
			t.Fatalf("gagal parse JSON log: %v, output: %s", err, output)
		}

		// Property 1: Log entry HARUS mengandung semua field wajib
		requiredFields := []string{
			"time",       // timestamp dari zerolog
			"user_id",    // ID user
			"tenant_id",  // ID tenant
			"ip_address", // alamat IP
			"user_agent", // User-Agent
			"result",     // hasil event
			"event_type", // tipe event
		}

		for _, field := range requiredFields {
			val, exists := logEntry[field]
			if !exists {
				t.Errorf("field wajib %q tidak ditemukan di log entry", field)
				continue
			}
			strVal, ok := val.(string)
			if !ok {
				// timestamp bisa berupa string, pastikan ada
				continue
			}
			if strVal == "" {
				t.Errorf("field wajib %q kosong di log entry", field)
			}
		}

		// Property 2: Nilai field harus sesuai dengan input
		if got := logEntry["event_type"]; got != string(eventType) {
			t.Errorf("event_type: got %v, want %v", got, string(eventType))
		}
		if got := logEntry["user_id"]; got != userID {
			t.Errorf("user_id: got %v, want %v", got, userID)
		}
		if got := logEntry["tenant_id"]; got != tenantID {
			t.Errorf("tenant_id: got %v, want %v", got, tenantID)
		}
		if got := logEntry["ip_address"]; got != ipAddress {
			t.Errorf("ip_address: got %v, want %v", got, ipAddress)
		}
		if got := logEntry["user_agent"]; got != userAgent {
			t.Errorf("user_agent: got %v, want %v", got, userAgent)
		}
		if got := logEntry["result"]; got != result {
			t.Errorf("result: got %v, want %v", got, result)
		}

		// Property 3: Log entry TIDAK BOLEH mengandung data sensitif
		sensitivePatterns := []string{
			"password",
			"token",
			"hash",
			"secret",
			"credential",
		}

		logJSON := strings.ToLower(output)
		for _, pattern := range sensitivePatterns {
			// Cek apakah ada key yang mengandung kata sensitif
			// (kecuali "token_hash" yang mungkin ada di key name, bukan value)
			for key, val := range logEntry {
				keyLower := strings.ToLower(key)
				// Skip field yang memang bukan data sensitif
				if keyLower == "event_type" || keyLower == "audit" || keyLower == "message" ||
					keyLower == "level" || keyLower == "time" || keyLower == "result" ||
					keyLower == "user_id" || keyLower == "tenant_id" ||
					keyLower == "ip_address" || keyLower == "user_agent" ||
					keyLower == "target_user_id" {
					continue
				}
				if strings.Contains(keyLower, pattern) {
					t.Errorf("log entry mengandung key sensitif %q dengan value %v", key, val)
				}
			}
		}

		// Pastikan value "sensitive-value-should-not-appear" tidak ada di output
		if includeSensitive && strings.Contains(logJSON, "sensitive-value-should-not-appear") {
			t.Error("log entry mengandung value sensitif yang seharusnya difilter")
		}
	})
}

// TestProperty_AuditLogNeverContainsSensitiveValues memverifikasi bahwa
// audit logger memfilter value yang terlihat seperti password hash atau token.
func TestProperty_AuditLogNeverContainsSensitiveValues(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		var buf bytes.Buffer
		logger := zerolog.New(&buf).With().Timestamp().Logger()
		auditLogger := NewAuditLogger(logger)

		// Buat metadata dengan berbagai jenis data sensitif
		metadata := map[string]string{
			"action": "test-action",
		}

		// Tambahkan data sensitif yang harus difilter
		sensitiveType := rapid.SampledFrom([]string{
			"bcrypt_hash",
			"hex_token",
			"password_field",
			"token_field",
		}).Draw(t, "sensitiveType")

		switch sensitiveType {
		case "bcrypt_hash":
			// Bcrypt hash dimulai dengan $2a$ atau $2b$
			metadata["some_field"] = "$2a$10$abcdefghijklmnopqrstuuABCDEFGHIJKLMNOPQRSTUVWXYZ012345"
		case "hex_token":
			// Hex string panjang (>= 64 karakter) seperti SHA-256 hash
			metadata["some_field"] = "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2"
		case "password_field":
			metadata["password"] = "mysecretpassword123"
		case "token_field":
			metadata["refresh_token"] = "some-refresh-token-value"
		}

		buf.Reset()
		auditLogger.LogEvent(
			AuditEventLogin,
			"user-123",
			"tenant-456",
			"127.0.0.1",
			"test-agent",
			"success",
			metadata,
		)

		output := buf.String()
		var logEntry map[string]interface{}
		if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &logEntry); err != nil {
			t.Fatalf("gagal parse JSON: %v", err)
		}

		// Verifikasi bahwa data sensitif tidak muncul di log
		switch sensitiveType {
		case "bcrypt_hash":
			if strings.Contains(output, "$2a$10$") {
				t.Error("bcrypt hash ditemukan di audit log")
			}
		case "hex_token":
			if strings.Contains(output, "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2") {
				t.Error("hex token/hash ditemukan di audit log")
			}
		case "password_field":
			if _, exists := logEntry["password"]; exists {
				t.Error("field 'password' ditemukan di audit log")
			}
		case "token_field":
			if _, exists := logEntry["refresh_token"]; exists {
				t.Error("field 'refresh_token' ditemukan di audit log")
			}
		}
	})
}

// TestAuditLogAllEventTypes memverifikasi bahwa semua event type yang valid
// menghasilkan log entry yang benar.
func TestAuditLogAllEventTypes(t *testing.T) {
	for _, eventType := range ValidAuditEvents {
		t.Run(string(eventType), func(t *testing.T) {
			var buf bytes.Buffer
			logger := zerolog.New(&buf).With().Timestamp().Logger()
			auditLogger := NewAuditLogger(logger)

			auditLogger.LogEvent(
				eventType,
				"user-123",
				"tenant-456",
				"192.168.1.1",
				"Mozilla/5.0",
				"success",
				nil,
			)

			var logEntry map[string]interface{}
			if err := json.Unmarshal([]byte(strings.TrimSpace(buf.String())), &logEntry); err != nil {
				t.Fatalf("gagal parse JSON untuk event %s: %v", eventType, err)
			}

			if got := logEntry["event_type"]; got != string(eventType) {
				t.Errorf("event_type: got %v, want %v", got, string(eventType))
			}
		})
	}
}
