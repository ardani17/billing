package usecase

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/hibiken/asynq"

	"github.com/ispboss/ispboss/pkg/queue"
	"github.com/ispboss/ispboss/services/network-service/internal/domain"
	"pgregory.net/rapid"
)

// =============================================================================
// Feature: mikrotik-router, Property 13: Event payload completeness with correlation ID
// =============================================================================

// routerGen menghasilkan domain.Router acak dengan field yang realistis.
func routerGen(t *rapid.T) *domain.Router {
	id := rapid.StringMatching(
		`[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}`,
	).Draw(t, "routerID")
	tenantID := rapid.StringMatching(
		`[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}`,
	).Draw(t, "tenantID")
	name := rapid.StringMatching(`[A-Za-z0-9_\-]{1,50}`).Draw(t, "routerName")

	// Generate LastOnlineAt — kadang nil, kadang ada
	var lastOnline *time.Time
	if rapid.Bool().Draw(t, "hasLastOnline") {
		ts := time.Now().Add(-time.Duration(rapid.IntRange(1, 86400).Draw(t, "lastOnlineSec")) * time.Second).Truncate(time.Second)
		lastOnline = &ts
	}

	return &domain.Router{
		ID:           id,
		TenantID:     tenantID,
		Name:         name,
		Host:         rapid.StringMatching(`192\.168\.[0-9]{1,3}\.[0-9]{1,3}`).Draw(t, "host"),
		Port:         rapid.IntRange(1, 65535).Draw(t, "port"),
		Status:       domain.StatusOnline,
		LastOnlineAt: lastOnline,
	}
}

// getEnqueuedEnvelope mengambil TaskEnvelope dari pending queue berdasarkan event type.
func getEnqueuedEnvelope(t *testing.T, inspector *asynq.Inspector, expectedType string) *queue.TaskEnvelope {
	tasks, err := inspector.ListPendingTasks("default", asynq.PageSize(10))
	if err != nil {
		t.Fatalf("gagal list pending tasks: %v", err)
	}

	for _, task := range tasks {
		if task.Type == expectedType {
			var envelope queue.TaskEnvelope
			if err := json.Unmarshal(task.Payload, &envelope); err != nil {
				t.Fatalf("gagal decode TaskEnvelope: %v", err)
			}
			return &envelope
		}
	}

	t.Fatalf("tidak ditemukan task dengan type %q di queue", expectedType)
	return nil
}

// TestProperty_EventPayloadCompletenessWithCorrelationID memverifikasi bahwa
// untuk sembarang router event (offline, online, atau unexpected_reboot),
// TaskEnvelope yang dipublish memiliki non-empty correlation_id, event_type
// yang benar, dan semua required payload fields untuk tipe event tersebut.
//
// **Validates: Requirements 10.1, 10.2, 10.3, 10.4**
func TestProperty_EventPayloadCompletenessWithCorrelationID(t *testing.T) {
	t.Run("router_offline_event", func(t *testing.T) {
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

			publisher := NewEventPublisher(client)
			router := routerGen(rt)
			ctx := context.Background()

			// Publish event router offline
			_ = publisher.PublishRouterOffline(ctx, router)

			// Ambil envelope dari queue
			envelope := getEnqueuedEnvelope(t, inspector, EventRouterOffline)

			// Verifikasi correlation_id non-empty (auto-generated oleh pkg/queue)
			if envelope.CorrelationID == "" {
				t.Error("correlation_id kosong pada event router_offline")
			}

			// Verifikasi event_type benar
			if envelope.EventType != EventRouterOffline {
				t.Errorf("event_type=%q, diharapkan=%q", envelope.EventType, EventRouterOffline)
			}

			// Verifikasi tenant_id pada envelope
			if envelope.TenantID != router.TenantID {
				t.Errorf("envelope.TenantID=%q, diharapkan=%q", envelope.TenantID, router.TenantID)
			}

			// Verifikasi timestamp non-zero
			if envelope.Timestamp.IsZero() {
				t.Error("timestamp kosong pada envelope")
			}

			// Decode dan verifikasi payload fields
			var payload domain.RouterOfflinePayload
			if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
				t.Fatalf("gagal decode RouterOfflinePayload: %v", err)
			}

			if payload.RouterID != router.ID {
				t.Errorf("payload.RouterID=%q, diharapkan=%q", payload.RouterID, router.ID)
			}
			if payload.RouterName != router.Name {
				t.Errorf("payload.RouterName=%q, diharapkan=%q", payload.RouterName, router.Name)
			}
			if payload.TenantID != router.TenantID {
				t.Errorf("payload.TenantID=%q, diharapkan=%q", payload.TenantID, router.TenantID)
			}
			// LastOnlineAt harus sesuai dengan router.LastOnlineAt
			if router.LastOnlineAt != nil && !payload.LastOnlineAt.Equal(*router.LastOnlineAt) {
				t.Errorf("payload.LastOnlineAt=%v, diharapkan=%v", payload.LastOnlineAt, *router.LastOnlineAt)
			}
		})
	})

	t.Run("router_online_event", func(t *testing.T) {
		rapid.Check(t, func(rt *rapid.T) {
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

			publisher := NewEventPublisher(client)
			router := routerGen(rt)
			ctx := context.Background()

			// Generate downtime duration acak (1 detik - 24 jam)
			downtimeSec := rapid.IntRange(1, 86400).Draw(rt, "downtimeSec")
			downtime := time.Duration(downtimeSec) * time.Second

			// Publish event router online
			_ = publisher.PublishRouterOnline(ctx, router, downtime)

			// Ambil envelope dari queue
			envelope := getEnqueuedEnvelope(t, inspector, EventRouterOnline)

			// Verifikasi correlation_id non-empty
			if envelope.CorrelationID == "" {
				t.Error("correlation_id kosong pada event router_online")
			}

			// Verifikasi event_type benar
			if envelope.EventType != EventRouterOnline {
				t.Errorf("event_type=%q, diharapkan=%q", envelope.EventType, EventRouterOnline)
			}

			// Verifikasi tenant_id pada envelope
			if envelope.TenantID != router.TenantID {
				t.Errorf("envelope.TenantID=%q, diharapkan=%q", envelope.TenantID, router.TenantID)
			}

			// Verifikasi timestamp non-zero
			if envelope.Timestamp.IsZero() {
				t.Error("timestamp kosong pada envelope")
			}

			// Decode dan verifikasi payload fields
			var payload domain.RouterOnlinePayload
			if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
				t.Fatalf("gagal decode RouterOnlinePayload: %v", err)
			}

			if payload.RouterID != router.ID {
				t.Errorf("payload.RouterID=%q, diharapkan=%q", payload.RouterID, router.ID)
			}
			if payload.RouterName != router.Name {
				t.Errorf("payload.RouterName=%q, diharapkan=%q", payload.RouterName, router.Name)
			}
			if payload.TenantID != router.TenantID {
				t.Errorf("payload.TenantID=%q, diharapkan=%q", payload.TenantID, router.TenantID)
			}
			if payload.DowntimeDuration != downtime {
				t.Errorf("payload.DowntimeDuration=%v, diharapkan=%v", payload.DowntimeDuration, downtime)
			}
		})
	})

	t.Run("router_unexpected_reboot_event", func(t *testing.T) {
		rapid.Check(t, func(rt *rapid.T) {
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

			publisher := NewEventPublisher(client)
			router := routerGen(rt)
			ctx := context.Background()

			// Generate uptime: prevUptime > currUptime (reboot terdeteksi)
			prevUptime := int64(rapid.IntRange(3600, 31536000).Draw(rt, "prevUptime"))
			currUptime := int64(rapid.IntRange(0, int(prevUptime)-1).Draw(rt, "currUptime"))

			// Publish event unexpected reboot
			_ = publisher.PublishUnexpectedReboot(ctx, router, prevUptime, currUptime)

			// Ambil envelope dari queue
			envelope := getEnqueuedEnvelope(t, inspector, EventRouterUnexpectedReboot)

			// Verifikasi correlation_id non-empty
			if envelope.CorrelationID == "" {
				t.Error("correlation_id kosong pada event unexpected_reboot")
			}

			// Verifikasi event_type benar
			if envelope.EventType != EventRouterUnexpectedReboot {
				t.Errorf("event_type=%q, diharapkan=%q", envelope.EventType, EventRouterUnexpectedReboot)
			}

			// Verifikasi tenant_id pada envelope
			if envelope.TenantID != router.TenantID {
				t.Errorf("envelope.TenantID=%q, diharapkan=%q", envelope.TenantID, router.TenantID)
			}

			// Verifikasi timestamp non-zero
			if envelope.Timestamp.IsZero() {
				t.Error("timestamp kosong pada envelope")
			}

			// Decode dan verifikasi payload fields
			var payload domain.RouterRebootPayload
			if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
				t.Fatalf("gagal decode RouterRebootPayload: %v", err)
			}

			if payload.RouterID != router.ID {
				t.Errorf("payload.RouterID=%q, diharapkan=%q", payload.RouterID, router.ID)
			}
			if payload.RouterName != router.Name {
				t.Errorf("payload.RouterName=%q, diharapkan=%q", payload.RouterName, router.Name)
			}
			if payload.TenantID != router.TenantID {
				t.Errorf("payload.TenantID=%q, diharapkan=%q", payload.TenantID, router.TenantID)
			}
			if payload.PreviousUptimeSeconds != prevUptime {
				t.Errorf("payload.PreviousUptimeSeconds=%d, diharapkan=%d", payload.PreviousUptimeSeconds, prevUptime)
			}
			if payload.CurrentUptimeSeconds != currUptime {
				t.Errorf("payload.CurrentUptimeSeconds=%d, diharapkan=%d", payload.CurrentUptimeSeconds, currUptime)
			}
		})
	})
}
