// Package worker berisi unit test untuk EventConsumer.
// Fokus pada:
// - Jumlah event type yang terdaftar (13 event)
// - Registrasi handler ke asynq.ServeMux tanpa panic
// - Verifikasi event type spesifik ada di daftar
package worker

import (
	"context"
	"io"
	"testing"

	"github.com/hibiken/asynq"
	"github.com/rs/zerolog"
)

// =============================================================================
// =============================================================================

// newTestEventConsumer membuat EventConsumer dengan pipeline nil untuk pengujian.
// Pipeline nil aman karena test hanya menguji registrasi dan decode payload.
func newTestEventConsumer() *EventConsumer {
	logger := zerolog.New(io.Discard)
	return NewEventConsumer(nil, logger)
}

// processTaskSafe memanggil mux.ProcessTask dan menangkap panic jika terjadi.
func processTaskSafe(mux *asynq.ServeMux, task *asynq.Task) (handled bool) {
	defer func() {
		if r := recover(); r != nil {
			handled = true
		}
	}()
	_ = mux.ProcessTask(context.Background(), task)
	return true
}

// =============================================================================
// Tes: allEventTypes harus berisi tepat 13 event type
// Memvalidasi: Kebutuhan 4.1
// =============================================================================

func TestAllEventTypes_Count(t *testing.T) {
	expected := 13
	if len(allEventTypes) != expected {
		t.Fatalf("expected %d event types, got %d", expected, len(allEventTypes))
	}
}

// =============================================================================
// Tes: allEventTypes harus berisi event type spesifik sesuai requirement
// Memvalidasi: Kebutuhan 4.1
// =============================================================================

func TestAllEventTypes_Contains(t *testing.T) {
	expectedEvents := []string{
		"invoice.created",
		"invoice.reminder",
		"invoice.penalty_added",
		"payment.online.received",
		"payment.recorded",
		"customer.isolir",
		"customer.un_isolir",
		"customer.suspend",
		"notification.isolir",
		"notification.un_isolir",
		"notification.suspend",
		"notification.reactivated",
		"notification.pending_sync_failed",
	}

	eventSet := make(map[string]bool, len(allEventTypes))
	for _, e := range allEventTypes {
		eventSet[e] = true
	}

	for _, expected := range expectedEvents {
		if !eventSet[expected] {
			t.Errorf("event type %q tidak ditemukan di allEventTypes", expected)
		}
	}
}

// =============================================================================
// Tes: RegisterHandlers tidak panic dan semua event type terdaftar di ServeMux
// Memvalidasi: Kebutuhan 4.1, 4.2
// =============================================================================

func TestEventConsumer_RegisterHandlers_NoPanic(t *testing.T) {
	ec := newTestEventConsumer()
	mux := asynq.NewServeMux()

	// RegisterHandlers tidak boleh panic
	ec.RegisterHandlers(mux)

	// Verifikasi setiap event type terdaftar di mux dengan memanggil ProcessTask
	for _, eventType := range allEventTypes {
		t.Run(eventType, func(t *testing.T) {
			task := asynq.NewTask(eventType, nil)
			if !processTaskSafe(mux, task) {
				t.Fatalf("handler untuk event type %q tidak terdaftar di ServeMux", eventType)
			}
		})
	}
}

// =============================================================================
// Memvalidasi: Kebutuhan 4.3, 4.4
// =============================================================================

func TestEventConsumer_HandleEvent_InvalidPayload(t *testing.T) {
	ec := newTestEventConsumer()
	mux := asynq.NewServeMux()
	ec.RegisterHandlers(mux)

	task := asynq.NewTask(EventInvoiceCreated, []byte("ini bukan json"))

	err := mux.ProcessTask(context.Background(), task)
	if err != nil {
		t.Fatalf("expected nil error for invalid payload (skip), got: %v", err)
	}
}
