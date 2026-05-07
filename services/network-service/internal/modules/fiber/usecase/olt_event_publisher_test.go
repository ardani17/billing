package usecase

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/hibiken/asynq"
	"github.com/rs/zerolog"

	"github.com/ispboss/ispboss/pkg/queue"
	"github.com/ispboss/ispboss/services/network-service/internal/domain"
	"pgregory.net/rapid"
)

// =============================================================================
// **Memvalidasi: Kebutuhan 17.1, 17.2, 17.3, 17.4**
//
// Untuk sembarang OLT event (device_offline, device_online, alarm),
// =============================================================================

// oltNameGen menghasilkan nama OLT acak yang realistis.
func oltNameGen() *rapid.Generator[string] {
	return rapid.StringMatching(`[A-Za-z0-9_\-]{1,50}`)
}

func oltBrandGen() *rapid.Generator[string] {
	return rapid.SampledFrom([]string{"zte", "huawei", "fiberhome", "vsol", "hsgq"})
}

func alarmTypeGen() *rapid.Generator[string] {
	return rapid.SampledFrom([]string{
		"ont_los", "ont_dying_gasp", "pon_port_down",
		"power_failure", "high_temperature", "ont_signal_degraded",
	})
}

func severityGen() *rapid.Generator[string] {
	return rapid.SampledFrom([]string{"critical", "major", "minor", "warning", "clear"})
}

// setupOLTRedis membuat miniredis, asynq client, dan inspector untuk test.
func setupOLTRedis(t *testing.T) (*miniredis.Miniredis, *asynq.Client, *asynq.Inspector) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("gagal memulai miniredis: %v", err)
	}
	redisOpt := asynq.RedisClientOpt{Addr: mr.Addr()}
	client := asynq.NewClient(redisOpt)
	inspector := asynq.NewInspector(redisOpt)
	return mr, client, inspector
}

// getOLTEnvelope mengambil TaskEnvelope dari pending queue berdasarkan event type.
func getOLTEnvelope(t *testing.T, inspector *asynq.Inspector, eventType string) *queue.TaskEnvelope {
	tasks, err := inspector.ListPendingTasks("default", asynq.PageSize(10))
	if err != nil {
		t.Fatalf("gagal list pending tasks: %v", err)
	}
	for _, task := range tasks {
		if task.Type == eventType {
			var env queue.TaskEnvelope
			if err := json.Unmarshal(task.Payload, &env); err != nil {
				t.Fatalf("gagal decode TaskEnvelope: %v", err)
			}
			return &env
		}
	}
	t.Fatalf("tidak ditemukan task dengan type %q di queue", eventType)
	return nil
}

// TestProperty_OLTEventPayloadCompleteness memverifikasi bahwa untuk sembarang
// OLT event payload, semua required field non-empty dan correlation_id terisi.
//
// **Memvalidasi: Kebutuhan 17.1, 17.2, 17.3, 17.4**
func TestProperty_OLTEventPayloadCompleteness(t *testing.T) {
	logger := zerolog.Nop()

	// Sub-test: OLTDeviceOfflinePayload
	t.Run("olt_device_offline_payload", func(t *testing.T) {
		rapid.Check(t, func(rt *rapid.T) {
			mr, client, inspector := setupOLTRedis(t)
			defer mr.Close()
			defer client.Close()
			defer inspector.Close()

			publisher := NewOLTEventPublisher(client, logger)
			payload := domain.OLTDeviceOfflinePayload{
				OLTID:        uuidGen().Draw(rt, "oltID"),
				OLTName:      oltNameGen().Draw(rt, "oltName"),
				TenantID:     uuidGen().Draw(rt, "tenantID"),
				Brand:        oltBrandGen().Draw(rt, "brand"),
				LastOnlineAt: time.Now().Truncate(time.Second),
			}

			_ = publisher.PublishDeviceOffline(context.Background(), payload)
			env := getOLTEnvelope(t, inspector, domain.EventOLTDeviceOffline)

			// Verifikasi envelope level
			if env.CorrelationID == "" {
				t.Error("correlation_id kosong pada envelope")
			}
			if env.EventType != domain.EventOLTDeviceOffline {
				t.Errorf("event_type=%q, want=%q", env.EventType, domain.EventOLTDeviceOffline)
			}

			// Decode dan verifikasi payload field
			var decoded domain.OLTDeviceOfflinePayload
			if err := json.Unmarshal(env.Payload, &decoded); err != nil {
				t.Fatalf("gagal decode payload: %v", err)
			}
			if decoded.CorrelationID == "" {
				t.Error("payload.correlation_id kosong")
			}
			if decoded.OLTID == "" {
				t.Error("payload.olt_id kosong")
			}
			if decoded.OLTName == "" {
				t.Error("payload.olt_name kosong")
			}
			if decoded.TenantID == "" {
				t.Error("payload.tenant_id kosong")
			}
		})
	})

	// Sub-test: OLTDeviceOnlinePayload
	t.Run("olt_device_online_payload", func(t *testing.T) {
		rapid.Check(t, func(rt *rapid.T) {
			mr, client, inspector := setupOLTRedis(t)
			defer mr.Close()
			defer client.Close()
			defer inspector.Close()

			publisher := NewOLTEventPublisher(client, logger)
			downtimeSec := rapid.IntRange(1, 86400).Draw(rt, "downtimeSec")
			payload := domain.OLTDeviceOnlinePayload{
				OLTID:            uuidGen().Draw(rt, "oltID"),
				OLTName:          oltNameGen().Draw(rt, "oltName"),
				TenantID:         uuidGen().Draw(rt, "tenantID"),
				Brand:            oltBrandGen().Draw(rt, "brand"),
				DowntimeDuration: time.Duration(downtimeSec) * time.Second,
			}

			_ = publisher.PublishDeviceOnline(context.Background(), payload)
			env := getOLTEnvelope(t, inspector, domain.EventOLTDeviceOnline)

			if env.CorrelationID == "" {
				t.Error("correlation_id kosong pada envelope")
			}

			var decoded domain.OLTDeviceOnlinePayload
			if err := json.Unmarshal(env.Payload, &decoded); err != nil {
				t.Fatalf("gagal decode payload: %v", err)
			}
			if decoded.CorrelationID == "" {
				t.Error("payload.correlation_id kosong")
			}
			if decoded.OLTID == "" {
				t.Error("payload.olt_id kosong")
			}
			if decoded.OLTName == "" {
				t.Error("payload.olt_name kosong")
			}
			if decoded.TenantID == "" {
				t.Error("payload.tenant_id kosong")
			}
		})
	})

	// Sub-test: OLTAlarmPayload - tambahan alarm_type dan severity
	t.Run("olt_alarm_payload", func(t *testing.T) {
		rapid.Check(t, func(rt *rapid.T) {
			mr, client, inspector := setupOLTRedis(t)
			defer mr.Close()
			defer client.Close()
			defer inspector.Close()

			publisher := NewOLTEventPublisher(client, logger)
			payload := domain.OLTAlarmPayload{
				OLTID:     uuidGen().Draw(rt, "oltID"),
				OLTName:   oltNameGen().Draw(rt, "oltName"),
				TenantID:  uuidGen().Draw(rt, "tenantID"),
				AlarmType: alarmTypeGen().Draw(rt, "alarmType"),
				Severity:  severityGen().Draw(rt, "severity"),
				Message:   "alarm test message",
			}

			_ = publisher.PublishAlarm(context.Background(), payload)
			env := getOLTEnvelope(t, inspector, domain.EventOLTAlarm)

			if env.CorrelationID == "" {
				t.Error("correlation_id kosong pada envelope")
			}

			var decoded domain.OLTAlarmPayload
			if err := json.Unmarshal(env.Payload, &decoded); err != nil {
				t.Fatalf("gagal decode payload: %v", err)
			}
			if decoded.CorrelationID == "" {
				t.Error("payload.correlation_id kosong")
			}
			if decoded.OLTID == "" {
				t.Error("payload.olt_id kosong")
			}
			if decoded.OLTName == "" {
				t.Error("payload.olt_name kosong")
			}
			if decoded.TenantID == "" {
				t.Error("payload.tenant_id kosong")
			}
			if decoded.AlarmType == "" {
				t.Error("payload.alarm_type kosong")
			}
			if decoded.Severity == "" {
				t.Error("payload.severity kosong")
			}
		})
	})
}
