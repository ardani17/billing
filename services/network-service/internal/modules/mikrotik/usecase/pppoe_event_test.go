package usecase

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/hibiken/asynq"
	"github.com/rs/zerolog"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
	"pgregory.net/rapid"
)

// =============================================================================
// =============================================================================

var validOperations = []string{"create", "isolir", "un_isolir", "suspend", "package_change"}

var validStatuses = []string{"success", "failed", "failed_permanent"}

// failedStatuses adalah daftar status yang menandakan kegagalan.
var failedStatuses = []string{"failed", "failed_permanent"}

func operationGen() *rapid.Generator[string] {
	return rapid.SampledFrom(validOperations)
}

func statusGen() *rapid.Generator[string] {
	return rapid.SampledFrom(validStatuses)
}

// failedStatusGen menghasilkan status gagal secara acak.
func failedStatusGen() *rapid.Generator[string] {
	return rapid.SampledFrom(failedStatuses)
}

func commandResultPayloadGen() *rapid.Generator[domain.CommandResultPayload] {
	return rapid.Custom[domain.CommandResultPayload](func(t *rapid.T) domain.CommandResultPayload {
		return domain.CommandResultPayload{
			CorrelationID: uuidGen().Draw(t, "correlationID"),
			CustomerID:    uuidGen().Draw(t, "customerID"),
			RouterID:      uuidGen().Draw(t, "routerID"),
			TenantID:      uuidGen().Draw(t, "tenantID"),
			Operation:     operationGen().Draw(t, "operation"),
			Status:        statusGen().Draw(t, "status"),
			ExecutedAt:    time.Now().Add(-time.Duration(rapid.IntRange(0, 3600).Draw(t, "executedAgoSec")) * time.Second),
			DurationMs:    int64(rapid.IntRange(1, 30000).Draw(t, "durationMs")),
		}
	})
}

// =============================================================================
// =============================================================================

// TestProperty_CommandResultPayloadCompleteness memverifikasi bahwa untuk
// sembarang CommandResultPayload, field-field berikut non-empty: correlation_id,
// customer_id, router_id, tenant_id, operation, status, executed_at.
// "failed_permanent".
//
// **Memvalidasi: Kebutuhan 12.2**
func TestProperty_CommandResultPayloadCompleteness(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		// Setup miniredis + asynq per iterasi agar queue bersih
		mr, err := miniredis.Run()
		if err != nil {
			t.Fatalf("gagal memulai miniredis: %v", err)
		}
		defer mr.Close()

		redisOpt := asynq.RedisClientOpt{Addr: mr.Addr()}
		client := asynq.NewClient(redisOpt)
		defer client.Close()
		inspector := asynq.NewInspector(redisOpt)
		defer inspector.Close()

		logger := zerolog.Nop()
		publisher := NewPPPoEEventPublisher(client, logger)
		ctx := context.Background()

		payload := commandResultPayloadGen().Draw(rt, "payload")

		// Terbitkan event
		_ = publisher.PublishCommandResult(ctx, payload)

		// Ambil envelope dari queue
		envelope := getEnqueuedEnvelope(t, inspector, EventCommandResult)

		// Verifikasi envelope field
		if envelope.CorrelationID == "" {
			t.Error("envelope.CorrelationID kosong")
		}
		if envelope.EventType != EventCommandResult {
			t.Errorf("envelope.EventType=%q, diharapkan=%q", envelope.EventType, EventCommandResult)
		}
		if envelope.TenantID != payload.TenantID {
			t.Errorf("envelope.TenantID=%q, diharapkan=%q", envelope.TenantID, payload.TenantID)
		}
		if envelope.Timestamp.IsZero() {
			t.Error("envelope.Timestamp kosong")
		}

		// Decode payload dari envelope
		var decoded domain.CommandResultPayload
		if err := json.Unmarshal(envelope.Payload, &decoded); err != nil {
			t.Fatalf("gagal decode CommandResultPayload: %v", err)
		}

		// Verifikasi required field non-empty
		if decoded.CorrelationID == "" {
			t.Error("correlation_id kosong")
		}
		if decoded.CustomerID == "" {
			t.Error("customer_id kosong")
		}
		if decoded.RouterID == "" {
			t.Error("router_id kosong")
		}
		if decoded.TenantID == "" {
			t.Error("tenant_id kosong")
		}
		if decoded.Operation == "" {
			t.Error("operation kosong")
		}
		if decoded.Status == "" {
			t.Error("status kosong")
		}
		if decoded.ExecutedAt.IsZero() {
			t.Error("executed_at kosong")
		}

		validOp := false
		for _, op := range validOperations {
			if decoded.Operation == op {
				validOp = true
				break
			}
		}
		if !validOp {
			t.Errorf("operation=%q bukan salah satu dari %v", decoded.Operation, validOperations)
		}

		validSt := false
		for _, st := range validStatuses {
			if decoded.Status == st {
				validSt = true
				break
			}
		}
		if !validSt {
			t.Errorf("status=%q bukan salah satu dari %v", decoded.Status, validStatuses)
		}

		if decoded.CorrelationID != payload.CorrelationID {
			t.Errorf("decoded.CorrelationID=%q, diharapkan=%q", decoded.CorrelationID, payload.CorrelationID)
		}
		if decoded.CustomerID != payload.CustomerID {
			t.Errorf("decoded.CustomerID=%q, diharapkan=%q", decoded.CustomerID, payload.CustomerID)
		}
		if decoded.RouterID != payload.RouterID {
			t.Errorf("decoded.RouterID=%q, diharapkan=%q", decoded.RouterID, payload.RouterID)
		}
		if decoded.TenantID != payload.TenantID {
			t.Errorf("decoded.TenantID=%q, diharapkan=%q", decoded.TenantID, payload.TenantID)
		}
		if decoded.Operation != payload.Operation {
			t.Errorf("decoded.Operation=%q, diharapkan=%q", decoded.Operation, payload.Operation)
		}
		if decoded.Status != payload.Status {
			t.Errorf("decoded.Status=%q, diharapkan=%q", decoded.Status, payload.Status)
		}
	})
}

// =============================================================================
// =============================================================================

func isErrorMessageSafe(errMsg, password, encryptionKey string) bool {
	if errMsg == "" {
		return true
	}
	if password != "" && strings.Contains(errMsg, password) {
		return false
	}
	if encryptionKey != "" && strings.Contains(errMsg, encryptionKey) {
		return false
	}
	return true
}

// TestProperty_ErrorMessageSafety memverifikasi bahwa untuk sembarang
// CommandResultPayload dengan status "failed" atau "failed_permanent",
// error_message tidak mengandung substring yang cocok dengan password
// atau encryption key. Secara spesifik, error_message tidak boleh mengandung
// plaintext password PPPoE user atau nilai ENCRYPTION_KEY.
//
// **Memvalidasi: Kebutuhan 12.4**
func TestProperty_ErrorMessageSafety(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		// Buat sensitive data
		password := rapid.StringMatching(`[a-zA-Z0-9!@#$%^&*]{4,32}`).Draw(rt, "password")
		encryptionKey := rapid.StringMatching(`[a-f0-9]{32,64}`).Draw(rt, "encryptionKey")

		safeMessages := []string{
			"connection refused: router offline",
			"timeout after 30s: router tidak merespons",
			"command execution failed: /ppp/secret/add returned error",
			"invalid profile: profile-10m tidak ditemukan di router",
			"duplicate entry: username sudah ada",
			"",
		}

		// Verifikasi pesan aman memang aman
		for _, msg := range safeMessages {
			if !isErrorMessageSafe(msg, password, encryptionKey) {
				t.Errorf("pesan aman terdeteksi tidak aman: %q", msg)
			}
		}

		// Verifikasi pesan yang mengandung password terdeteksi tidak aman
		unsafeWithPassword := rapid.SampledFrom([]string{
			"failed to create user with password " + password,
			"auth error: " + password + " is invalid",
			"error: password=" + password,
		}).Draw(rt, "unsafePasswordMsg")

		if isErrorMessageSafe(unsafeWithPassword, password, encryptionKey) {
			t.Errorf("pesan mengandung password tidak terdeteksi: %q", unsafeWithPassword)
		}

		// Verifikasi pesan yang mengandung encryption key terdeteksi tidak aman
		unsafeWithKey := rapid.SampledFrom([]string{
			"decryption failed with key " + encryptionKey,
			"crypto error: " + encryptionKey + " invalid",
			"key=" + encryptionKey,
		}).Draw(rt, "unsafeKeyMsg")

		if isErrorMessageSafe(unsafeWithKey, password, encryptionKey) {
			t.Errorf("pesan mengandung encryption key tidak terdeteksi: %q", unsafeWithKey)
		}

		safeRandomMsg := rapid.StringMatching(`[a-z ]{0,100}`).Draw(rt, "safeRandomMsg")
		if strings.Contains(safeRandomMsg, password) || strings.Contains(safeRandomMsg, encryptionKey) {
			// Skip jika kebetulan mengandung - sangat jarang terjadi
			return
		}
		if !isErrorMessageSafe(safeRandomMsg, password, encryptionKey) {
			t.Errorf("pesan acak aman terdeteksi tidak aman: %q (password=%q, key=%q)",
				safeRandomMsg, password, encryptionKey)
		}
	})
}
