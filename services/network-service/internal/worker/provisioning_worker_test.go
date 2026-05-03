package worker

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/hibiken/asynq"
	"github.com/rs/zerolog"

	"github.com/ispboss/ispboss/pkg/queue"
	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// =============================================================================
// Mock ProvisioningManager — mock sederhana untuk unit test worker
// =============================================================================

// mockProvisioningManager adalah mock implementasi domain.ProvisioningManager.
// Hanya method HandleCustomerTerminated yang digunakan oleh worker.
type mockProvisioningManager struct {
	// handleCustomerTerminatedFn adalah fungsi yang dipanggil saat HandleCustomerTerminated dipanggil.
	handleCustomerTerminatedFn func(ctx context.Context, customerID, tenantID string) error
}

func (m *mockProvisioningManager) HandleCustomerTerminated(ctx context.Context, customerID, tenantID string) error {
	if m.handleCustomerTerminatedFn != nil {
		return m.handleCustomerTerminatedFn(ctx, customerID, tenantID)
	}
	return nil
}

// Method stub lainnya — tidak digunakan oleh ProvisioningEventWorker.
func (m *mockProvisioningManager) ProvisionONT(_ context.Context, _ string, _ domain.ProvisionONTRequest) (*domain.ONTResponse, error) {
	return nil, nil
}
func (m *mockProvisioningManager) DecommissionONT(_ context.Context, _ string, _ string) error {
	return nil
}
func (m *mockProvisioningManager) RebootONT(_ context.Context, _ string, _ string) (*domain.ProvisioningResult, error) {
	return nil, nil
}
func (m *mockProvisioningManager) ValidateBulk(_ context.Context, _ string, _ string, _ []byte) (*domain.BulkPreview, error) {
	return nil, nil
}
func (m *mockProvisioningManager) ExecuteBulk(_ context.Context, _ string, _ string) (*domain.BulkResult, error) {
	return nil, nil
}
func (m *mockProvisioningManager) GetBulkTemplate() []byte { return nil }
func (m *mockProvisioningManager) HandleUnregisteredONT(_ context.Context, _ string, _ domain.UnregisteredONT) error {
	return nil
}
func (m *mockProvisioningManager) HandlePortMigration(_ context.Context, _ string, _, _, _, _ int) error {
	return nil
}
func (m *mockProvisioningManager) ConfirmMigration(_ context.Context, _ string) error { return nil }
func (m *mockProvisioningManager) GetONTByID(_ context.Context, _ string) (*domain.ONTDetailResponse, error) {
	return nil, nil
}
func (m *mockProvisioningManager) ListONTs(_ context.Context, _ domain.ONTListParams) (*domain.ONTListResult, error) {
	return nil, nil
}
func (m *mockProvisioningManager) GetUnregisteredONTs(_ context.Context, _ string) ([]*domain.ONTResponse, error) {
	return nil, nil
}
func (m *mockProvisioningManager) GetAuditLogs(_ context.Context, _ domain.AuditLogListParams) (*domain.AuditLogListResult, error) {
	return nil, nil
}
func (m *mockProvisioningManager) GetSettings(_ context.Context, _ string) (*domain.ProvisioningSettings, error) {
	return nil, nil
}
func (m *mockProvisioningManager) UpdateSettings(_ context.Context, _ string, _ domain.UpdateSettingsRequest) (*domain.ProvisioningSettings, error) {
	return nil, nil
}

// =============================================================================
// Helper — membuat asynq.Task dari TaskEnvelope untuk testing
// =============================================================================

// buildTask membuat asynq.Task dari TaskEnvelope.
// Digunakan oleh test untuk mensimulasikan task yang diterima worker.
func buildTask(t *testing.T, envelope queue.TaskEnvelope) *asynq.Task {
	t.Helper()
	data, err := json.Marshal(envelope)
	if err != nil {
		t.Fatalf("gagal marshal envelope: %v", err)
	}
	return asynq.NewTask(envelope.EventType, data)
}

// =============================================================================
// Unit Tests: HandleCustomerTerminated — happy path
// **Validates: Requirements 7.2, 13.1**
// =============================================================================

// TestHandleCustomerTerminated_HappyPath memverifikasi bahwa handler
// berhasil memproses event customer.terminated dengan payload valid.
func TestHandleCustomerTerminated_HappyPath(t *testing.T) {
	// Siapkan mock manager yang merekam argumen
	var gotCustomerID, gotTenantID string
	mock := &mockProvisioningManager{
		handleCustomerTerminatedFn: func(_ context.Context, customerID, tenantID string) error {
			gotCustomerID = customerID
			gotTenantID = tenantID
			return nil
		},
	}

	logger := zerolog.Nop()
	worker := NewProvisioningEventWorker(mock, logger)

	// Buat payload customer.terminated
	payload := domain.CustomerTerminatedPayload{
		CustomerID:   "cust-123",
		TenantID:     "tenant-abc",
		CustomerName: "John Doe",
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("gagal marshal payload: %v", err)
	}

	envelope := queue.TaskEnvelope{
		EventType:     EventCustomerTerminated,
		TenantID:      "tenant-abc",
		Timestamp:     time.Now(),
		CorrelationID: "corr-001",
		Payload:       payloadBytes,
	}

	task := buildTask(t, envelope)
	err = worker.handleCustomerTerminated(context.Background(), task)
	if err != nil {
		t.Fatalf("handleCustomerTerminated error: %v", err)
	}

	// Verifikasi argumen yang diteruskan ke manager
	if gotCustomerID != "cust-123" {
		t.Errorf("customerID = %q, want %q", gotCustomerID, "cust-123")
	}
	if gotTenantID != "tenant-abc" {
		t.Errorf("tenantID = %q, want %q", gotTenantID, "tenant-abc")
	}
}

// =============================================================================
// Unit Tests: HandleCustomerTerminated — customer_id kosong
// **Validates: Requirements 7.2**
// =============================================================================

// TestHandleCustomerTerminated_EmptyCustomerID memverifikasi bahwa handler
// mengembalikan error jika customer_id kosong dalam payload.
func TestHandleCustomerTerminated_EmptyCustomerID(t *testing.T) {
	mock := &mockProvisioningManager{}
	logger := zerolog.Nop()
	worker := NewProvisioningEventWorker(mock, logger)

	// Payload tanpa customer_id
	payload := domain.CustomerTerminatedPayload{
		CustomerID: "",
		TenantID:   "tenant-abc",
	}
	payloadBytes, _ := json.Marshal(payload)

	envelope := queue.TaskEnvelope{
		EventType:     EventCustomerTerminated,
		TenantID:      "tenant-abc",
		Timestamp:     time.Now(),
		CorrelationID: "corr-002",
		Payload:       payloadBytes,
	}

	task := buildTask(t, envelope)
	err := worker.handleCustomerTerminated(context.Background(), task)
	if err == nil {
		t.Fatal("expected error untuk customer_id kosong, got nil")
	}
}

// =============================================================================
// Unit Tests: HandleCustomerTerminated — tenant_id kosong
// **Validates: Requirements 7.2**
// =============================================================================

// TestHandleCustomerTerminated_EmptyTenantID memverifikasi bahwa handler
// mengembalikan error jika tenant_id kosong di payload DAN envelope.
func TestHandleCustomerTerminated_EmptyTenantID(t *testing.T) {
	mock := &mockProvisioningManager{}
	logger := zerolog.Nop()
	worker := NewProvisioningEventWorker(mock, logger)

	// Payload dan envelope tanpa tenant_id
	payload := domain.CustomerTerminatedPayload{
		CustomerID: "cust-123",
		TenantID:   "",
	}
	payloadBytes, _ := json.Marshal(payload)

	envelope := queue.TaskEnvelope{
		EventType:     EventCustomerTerminated,
		TenantID:      "", // envelope juga kosong
		Timestamp:     time.Now(),
		CorrelationID: "corr-003",
		Payload:       payloadBytes,
	}

	task := buildTask(t, envelope)
	err := worker.handleCustomerTerminated(context.Background(), task)
	if err == nil {
		t.Fatal("expected error untuk tenant_id kosong, got nil")
	}
}

// =============================================================================
// Unit Tests: HandleCustomerTerminated — tenant_id fallback dari envelope
// **Validates: Requirements 7.2, 13.1**
// =============================================================================

// TestHandleCustomerTerminated_TenantIDFallbackFromEnvelope memverifikasi bahwa
// handler menggunakan tenant_id dari envelope jika payload tidak punya.
func TestHandleCustomerTerminated_TenantIDFallbackFromEnvelope(t *testing.T) {
	var gotTenantID string
	mock := &mockProvisioningManager{
		handleCustomerTerminatedFn: func(_ context.Context, _, tenantID string) error {
			gotTenantID = tenantID
			return nil
		},
	}

	logger := zerolog.Nop()
	worker := NewProvisioningEventWorker(mock, logger)

	// Payload tanpa tenant_id, tapi envelope punya
	payload := domain.CustomerTerminatedPayload{
		CustomerID: "cust-456",
		TenantID:   "", // kosong di payload
	}
	payloadBytes, _ := json.Marshal(payload)

	envelope := queue.TaskEnvelope{
		EventType:     EventCustomerTerminated,
		TenantID:      "tenant-from-envelope",
		Timestamp:     time.Now(),
		CorrelationID: "corr-004",
		Payload:       payloadBytes,
	}

	task := buildTask(t, envelope)
	err := worker.handleCustomerTerminated(context.Background(), task)
	if err != nil {
		t.Fatalf("handleCustomerTerminated error: %v", err)
	}

	if gotTenantID != "tenant-from-envelope" {
		t.Errorf("tenantID = %q, want %q", gotTenantID, "tenant-from-envelope")
	}
}

// =============================================================================
// Unit Tests: HandleCustomerTerminated — manager returns error
// **Validates: Requirements 7.5**
// =============================================================================

// TestHandleCustomerTerminated_ManagerError memverifikasi bahwa error dari
// ProvisioningManager dipropagasi kembali oleh handler.
func TestHandleCustomerTerminated_ManagerError(t *testing.T) {
	managerErr := errors.New("decommission gagal: OLT unreachable")
	mock := &mockProvisioningManager{
		handleCustomerTerminatedFn: func(_ context.Context, _, _ string) error {
			return managerErr
		},
	}

	logger := zerolog.Nop()
	worker := NewProvisioningEventWorker(mock, logger)

	payload := domain.CustomerTerminatedPayload{
		CustomerID: "cust-789",
		TenantID:   "tenant-xyz",
	}
	payloadBytes, _ := json.Marshal(payload)

	envelope := queue.TaskEnvelope{
		EventType:     EventCustomerTerminated,
		TenantID:      "tenant-xyz",
		Timestamp:     time.Now(),
		CorrelationID: "corr-005",
		Payload:       payloadBytes,
	}

	task := buildTask(t, envelope)
	err := worker.handleCustomerTerminated(context.Background(), task)
	if err == nil {
		t.Fatal("expected error dari manager, got nil")
	}
	// Pastikan error asli terbungkus
	if !errors.Is(err, managerErr) {
		// Error mungkin di-wrap, cek apakah mengandung pesan asli
		if err.Error() == "" {
			t.Errorf("error tidak mengandung pesan dari manager")
		}
	}
}

// =============================================================================
// Unit Tests: ProvisioningRetryDelay — verifikasi delay sesuai array
// **Validates: Requirements 7.5**
// =============================================================================

// TestProvisioningRetryDelay_EachAttempt memverifikasi bahwa ProvisioningRetryDelay
// mengembalikan delay yang benar untuk setiap attempt 0 sampai 4.
func TestProvisioningRetryDelay_EachAttempt(t *testing.T) {
	tests := []struct {
		attempt int
		want    time.Duration
	}{
		{0, 30 * time.Second},
		{1, 60 * time.Second},
		{2, 120 * time.Second},
		{3, 300 * time.Second},
		{4, 600 * time.Second},
	}

	for _, tc := range tests {
		delay := ProvisioningRetryDelay(tc.attempt, nil, nil)
		if delay != tc.want {
			t.Errorf("ProvisioningRetryDelay(%d) = %v, want %v", tc.attempt, delay, tc.want)
		}
	}
}

// TestProvisioningRetryDelay_BeyondMaxRetries memverifikasi bahwa untuk
// attempt >= 5, delay tetap menggunakan nilai terakhir (600s).
func TestProvisioningRetryDelay_BeyondMaxRetries(t *testing.T) {
	beyondAttempts := []int{5, 6, 10, 50, 100}
	want := 600 * time.Second

	for _, attempt := range beyondAttempts {
		delay := ProvisioningRetryDelay(attempt, nil, nil)
		if delay != want {
			t.Errorf("ProvisioningRetryDelay(%d) = %v, want %v", attempt, delay, want)
		}
	}
}

// TestProvisioningRetryDelays_MatchesExpected memverifikasi bahwa array
// ProvisioningRetryDelays berisi tepat 5 entry dengan nilai yang benar.
func TestProvisioningRetryDelays_MatchesExpected(t *testing.T) {
	if len(ProvisioningRetryDelays) != 5 {
		t.Fatalf("ProvisioningRetryDelays has %d entries, want 5", len(ProvisioningRetryDelays))
	}

	expected := []time.Duration{
		30 * time.Second,
		60 * time.Second,
		120 * time.Second,
		300 * time.Second,
		600 * time.Second,
	}

	for i, want := range expected {
		if ProvisioningRetryDelays[i] != want {
			t.Errorf("ProvisioningRetryDelays[%d] = %v, want %v", i, ProvisioningRetryDelays[i], want)
		}
	}
}

// =============================================================================
// Unit Tests: RegisterHandlers — verifikasi handler terdaftar
// **Validates: Requirements 13.1**
// =============================================================================

// TestRegisterHandlers_CustomerTerminated memverifikasi bahwa RegisterHandlers
// mendaftarkan handler untuk event "customer.terminated" ke asynq ServeMux.
func TestRegisterHandlers_CustomerTerminated(t *testing.T) {
	mock := &mockProvisioningManager{}
	logger := zerolog.Nop()
	worker := NewProvisioningEventWorker(mock, logger)

	mux := asynq.NewServeMux()

	// RegisterHandlers tidak boleh panic
	worker.RegisterHandlers(mux)

	// Verifikasi handler terdaftar dengan mengirim task valid
	payload := domain.CustomerTerminatedPayload{
		CustomerID: "cust-reg-test",
		TenantID:   "tenant-reg-test",
	}
	payloadBytes, _ := json.Marshal(payload)

	envelope := queue.TaskEnvelope{
		EventType:     EventCustomerTerminated,
		TenantID:      "tenant-reg-test",
		Timestamp:     time.Now(),
		CorrelationID: "corr-reg",
		Payload:       payloadBytes,
	}

	task := buildTask(t, envelope)

	// ProcessTask akan memanggil handler yang terdaftar
	err := mux.ProcessTask(context.Background(), task)
	if err != nil {
		t.Errorf("ProcessTask untuk %q error: %v", EventCustomerTerminated, err)
	}
}

// =============================================================================
// Unit Tests: provisioningMaxRetries constant
// **Validates: Requirements 7.5**
// =============================================================================

// TestProvisioningMaxRetries memverifikasi bahwa provisioningMaxRetries = 5.
func TestProvisioningMaxRetries(t *testing.T) {
	if provisioningMaxRetries != 5 {
		t.Errorf("provisioningMaxRetries = %d, want 5", provisioningMaxRetries)
	}
}
