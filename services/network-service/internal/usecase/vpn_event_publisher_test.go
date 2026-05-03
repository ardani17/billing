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
// Feature: mikrotik-vpn, Property 7: VPN event payload completeness
// **Validates: Requirements 16.1, 16.2, 16.3, 16.4**
// =============================================================================

// vpnProtocolGen menghasilkan generator protokol VPN acak yang valid.
func vpnProtocolGen() *rapid.Generator[string] {
	return rapid.SampledFrom([]string{"wireguard", "l2tp_ipsec", "pptp", "sstp", "openvpn"})
}

// vpnIPGen menghasilkan generator VPN IP acak dalam format 10.99.x.y.
func vpnIPGen() *rapid.Generator[string] {
	return rapid.StringMatching(`10\.99\.[0-9]{1,3}\.[0-9]{1,3}`)
}

// vpnTunnelNameGen menghasilkan generator nama tunnel acak.
func vpnTunnelNameGen() *rapid.Generator[string] {
	return rapid.StringMatching(`[A-Za-z0-9_\-]{1,50}`)
}

// setupVPNRedis membuat miniredis, asynq client, dan inspector untuk test.
func setupVPNRedis(t *testing.T) (*miniredis.Miniredis, *asynq.Client, *asynq.Inspector) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("gagal memulai miniredis: %v", err)
	}
	redisOpt := asynq.RedisClientOpt{Addr: mr.Addr()}
	client := asynq.NewClient(redisOpt)
	inspector := asynq.NewInspector(redisOpt)
	return mr, client, inspector
}

// getVPNEnvelope mengambil TaskEnvelope dari pending queue berdasarkan event type.
func getVPNEnvelope(t *testing.T, inspector *asynq.Inspector, eventType string) *queue.TaskEnvelope {
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

// TestProperty_VPNEventPayloadCompleteness memverifikasi bahwa untuk sembarang
// VPN event payload, semua required fields non-empty dan correlation_id terisi.
//
// **Validates: Requirements 16.1, 16.2, 16.3, 16.4**
func TestProperty_VPNEventPayloadCompleteness(t *testing.T) {
	logger := zerolog.Nop()

	// Sub-test: VPNTunnelDownPayload — non-empty correlation_id, tunnel_id,
	// tunnel_name, tenant_id, protocol, vpn_ip, disconnected_at
	t.Run("vpn_tunnel_down_payload", func(t *testing.T) {
		rapid.Check(t, func(rt *rapid.T) {
			mr, client, inspector := setupVPNRedis(t)
			defer mr.Close()
			defer client.Close()
			defer inspector.Close()

			publisher := NewVPNEventPublisher(client, logger)
			payload := domain.VPNTunnelDownPayload{
				TunnelID:       uuidGen().Draw(rt, "tunnelID"),
				TunnelName:     vpnTunnelNameGen().Draw(rt, "tunnelName"),
				TenantID:       uuidGen().Draw(rt, "tenantID"),
				Protocol:       vpnProtocolGen().Draw(rt, "protocol"),
				VPNIP:          vpnIPGen().Draw(rt, "vpnIP"),
				DisconnectedAt: time.Now().Truncate(time.Second),
			}

			_ = publisher.PublishTunnelDown(context.Background(), payload)
			env := getVPNEnvelope(t, inspector, EventVPNTunnelDown)

			// Verifikasi envelope level
			if env.CorrelationID == "" {
				t.Error("correlation_id kosong pada envelope")
			}
			if env.EventType != EventVPNTunnelDown {
				t.Errorf("event_type=%q, diharapkan=%q", env.EventType, EventVPNTunnelDown)
			}

			// Decode dan verifikasi payload fields
			var decoded domain.VPNTunnelDownPayload
			if err := json.Unmarshal(env.Payload, &decoded); err != nil {
				t.Fatalf("gagal decode payload: %v", err)
			}
			if decoded.CorrelationID == "" {
				t.Error("payload.correlation_id kosong")
			}
			if decoded.TunnelID == "" {
				t.Error("payload.tunnel_id kosong")
			}
			if decoded.TunnelName == "" {
				t.Error("payload.tunnel_name kosong")
			}
			if decoded.TenantID == "" {
				t.Error("payload.tenant_id kosong")
			}
			if decoded.Protocol == "" {
				t.Error("payload.protocol kosong")
			}
			if decoded.VPNIP == "" {
				t.Error("payload.vpn_ip kosong")
			}
			if decoded.DisconnectedAt.IsZero() {
				t.Error("payload.disconnected_at kosong")
			}
		})
	})

	// Sub-test: VPNTunnelUpPayload — non-empty correlation_id, tunnel_id,
	// tunnel_name, tenant_id, protocol, vpn_ip, connected_at, latency_ms >= 0
	t.Run("vpn_tunnel_up_payload", func(t *testing.T) {
		rapid.Check(t, func(rt *rapid.T) {
			mr, client, inspector := setupVPNRedis(t)
			defer mr.Close()
			defer client.Close()
			defer inspector.Close()

			publisher := NewVPNEventPublisher(client, logger)
			payload := domain.VPNTunnelUpPayload{
				TunnelID:    uuidGen().Draw(rt, "tunnelID"),
				TunnelName:  vpnTunnelNameGen().Draw(rt, "tunnelName"),
				TenantID:    uuidGen().Draw(rt, "tenantID"),
				Protocol:    vpnProtocolGen().Draw(rt, "protocol"),
				VPNIP:       vpnIPGen().Draw(rt, "vpnIP"),
				LatencyMs:   rapid.IntRange(0, 500).Draw(rt, "latencyMs"),
				ConnectedAt: time.Now().Truncate(time.Second),
			}

			_ = publisher.PublishTunnelUp(context.Background(), payload)
			env := getVPNEnvelope(t, inspector, EventVPNTunnelUp)

			if env.CorrelationID == "" {
				t.Error("correlation_id kosong pada envelope")
			}

			var decoded domain.VPNTunnelUpPayload
			if err := json.Unmarshal(env.Payload, &decoded); err != nil {
				t.Fatalf("gagal decode payload: %v", err)
			}
			if decoded.CorrelationID == "" {
				t.Error("payload.correlation_id kosong")
			}
			if decoded.TunnelID == "" {
				t.Error("payload.tunnel_id kosong")
			}
			if decoded.TunnelName == "" {
				t.Error("payload.tunnel_name kosong")
			}
			if decoded.TenantID == "" {
				t.Error("payload.tenant_id kosong")
			}
			if decoded.Protocol == "" {
				t.Error("payload.protocol kosong")
			}
			if decoded.VPNIP == "" {
				t.Error("payload.vpn_ip kosong")
			}
			if decoded.ConnectedAt.IsZero() {
				t.Error("payload.connected_at kosong")
			}
			if decoded.LatencyMs < 0 {
				t.Errorf("payload.latency_ms=%d, diharapkan >= 0", decoded.LatencyMs)
			}
		})
	})

	// Sub-test: VPNTunnelCreatedPayload — non-empty correlation_id, tunnel_id,
	// tunnel_name, tenant_id, protocol, status
	t.Run("vpn_tunnel_created_payload", func(t *testing.T) {
		rapid.Check(t, func(rt *rapid.T) {
			mr, client, inspector := setupVPNRedis(t)
			defer mr.Close()
			defer client.Close()
			defer inspector.Close()

			publisher := NewVPNEventPublisher(client, logger)
			payload := domain.VPNTunnelCreatedPayload{
				TunnelID:   uuidGen().Draw(rt, "tunnelID"),
				TunnelName: vpnTunnelNameGen().Draw(rt, "tunnelName"),
				TenantID:   uuidGen().Draw(rt, "tenantID"),
				Protocol:   vpnProtocolGen().Draw(rt, "protocol"),
				Status:     rapid.SampledFrom([]string{"pending", "error"}).Draw(rt, "status"),
			}

			_ = publisher.PublishTunnelCreated(context.Background(), payload)
			env := getVPNEnvelope(t, inspector, EventVPNTunnelCreated)

			if env.CorrelationID == "" {
				t.Error("correlation_id kosong pada envelope")
			}

			var decoded domain.VPNTunnelCreatedPayload
			if err := json.Unmarshal(env.Payload, &decoded); err != nil {
				t.Fatalf("gagal decode payload: %v", err)
			}
			if decoded.CorrelationID == "" {
				t.Error("payload.correlation_id kosong")
			}
			if decoded.TunnelID == "" {
				t.Error("payload.tunnel_id kosong")
			}
			if decoded.TunnelName == "" {
				t.Error("payload.tunnel_name kosong")
			}
			if decoded.TenantID == "" {
				t.Error("payload.tenant_id kosong")
			}
			if decoded.Protocol == "" {
				t.Error("payload.protocol kosong")
			}
			if decoded.Status == "" {
				t.Error("payload.status kosong")
			}
		})
	})
}
