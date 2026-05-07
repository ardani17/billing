// Memverifikasi bahwa semua provisioning event (ont.provisioned, ont.decommissioned,
// ont.auto_provisioned, ont.auto_provision_failed, ont.port_migrated) memiliki
// payload lengkap dengan semua required field non-empty dan correlation_id terisi.
package usecase

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/hibiken/asynq"
	"github.com/rs/zerolog"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
	"pgregory.net/rapid"
)

// =============================================================================
// **Memvalidasi: Kebutuhan 13.2, 13.3, 13.4, 13.5, 13.6, 13.7**
//
// - non-empty correlation_id (UUID v4)
// - event-specific required field non-empty
// =============================================================================

// serialNumberGen menghasilkan serial number ONT acak yang realistis.
func serialNumberGen() *rapid.Generator[string] {
	return rapid.StringMatching(`[A-Z]{4}[A-Fa-f0-9]{8}`)
}

func provPonPortGen() *rapid.Generator[int] {
	return rapid.IntRange(0, 15)
}

func provOntIndexGen() *rapid.Generator[int] {
	return rapid.IntRange(1, 128)
}

// setupProvisioningRedis membuat miniredis dan asynq client untuk test.
func setupProvisioningRedis(t *testing.T) (*miniredis.Miniredis, *asynq.Client, *asynq.Inspector) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("gagal memulai miniredis: %v", err)
	}
	redisOpt := asynq.RedisClientOpt{Addr: mr.Addr()}
	client := asynq.NewClient(redisOpt)
	inspector := asynq.NewInspector(redisOpt)
	return mr, client, inspector
}

// getProvisioningEnvelope mengambil TaskEnvelope dari pending queue.
func getProvisioningEnvelope(t *testing.T, inspector *asynq.Inspector, eventType string) json.RawMessage {
	tasks, err := inspector.ListPendingTasks("default", asynq.PageSize(10))
	if err != nil {
		t.Fatalf("gagal list pending tasks: %v", err)
	}
	for _, task := range tasks {
		if task.Type == eventType {
			return task.Payload
		}
	}
	t.Fatalf("tidak ditemukan task dengan type %q di queue", eventType)
	return nil
}

// TestProperty7_ONTProvisionedPayload memverifikasi bahwa event ont.provisioned
// memiliki semua required field: correlation_id, ont_id, serial_number,
// customer_id, olt_id, olt_name, pon_port_index, vlan_id, tenant_id.
//
// **Memvalidasi: Kebutuhan 13.2**
func TestProperty7_ONTProvisionedPayload(t *testing.T) {
	logger := zerolog.Nop()

	rapid.Check(t, func(rt *rapid.T) {
		mr, client, inspector := setupProvisioningRedis(t)
		defer mr.Close()
		defer client.Close()
		defer inspector.Close()

		publisher := NewOLTEventPublisher(client, logger)

		payload := domain.ONTProvisionedPayload{
			ONTID:        uuidGen().Draw(rt, "ontID"),
			SerialNumber: serialNumberGen().Draw(rt, "serialNumber"),
			CustomerID:   uuidGen().Draw(rt, "customerID"),
			OLTID:        uuidGen().Draw(rt, "oltID"),
			OLTName:      oltNameGen().Draw(rt, "oltName"),
			PONPortIndex: provPonPortGen().Draw(rt, "ponPort"),
			VLANID:       uuidGen().Draw(rt, "vlanID"),
			TenantID:     uuidGen().Draw(rt, "tenantID"),
		}

		_ = publisher.PublishONTProvisioned(context.Background(), payload)
		raw := getProvisioningEnvelope(t, inspector, domain.EventONTProvisioned)

		// Decode payload dari envelope
		var env struct {
			Payload json.RawMessage `json:"payload"`
		}
		if err := json.Unmarshal(raw, &env); err != nil {
			t.Fatalf("gagal decode envelope: %v", err)
		}

		var decoded domain.ONTProvisionedPayload
		if err := json.Unmarshal(env.Payload, &decoded); err != nil {
			t.Fatalf("gagal decode payload: %v", err)
		}

		// Verifikasi semua required field non-empty
		if decoded.CorrelationID == "" {
			t.Error("correlation_id kosong")
		}
		if decoded.ONTID == "" {
			t.Error("ont_id kosong")
		}
		if decoded.SerialNumber == "" {
			t.Error("serial_number kosong")
		}
		if decoded.CustomerID == "" {
			t.Error("customer_id kosong")
		}
		if decoded.OLTID == "" {
			t.Error("olt_id kosong")
		}
		if decoded.OLTName == "" {
			t.Error("olt_name kosong")
		}
		if decoded.VLANID == "" {
			t.Error("vlan_id kosong")
		}
		if decoded.TenantID == "" {
			t.Error("tenant_id kosong")
		}
	})
}

// TestProperty7_ONTDecommissionedPayload memverifikasi bahwa event ont.decommissioned
// memiliki semua required field: correlation_id, ont_id, serial_number,
// customer_id, olt_id, olt_name, pon_port_index, tenant_id.
//
// **Memvalidasi: Kebutuhan 13.3**
func TestProperty7_ONTDecommissionedPayload(t *testing.T) {
	logger := zerolog.Nop()

	rapid.Check(t, func(rt *rapid.T) {
		mr, client, inspector := setupProvisioningRedis(t)
		defer mr.Close()
		defer client.Close()
		defer inspector.Close()

		publisher := NewOLTEventPublisher(client, logger)

		payload := domain.ONTDecommissionedPayload{
			ONTID:        uuidGen().Draw(rt, "ontID"),
			SerialNumber: serialNumberGen().Draw(rt, "serialNumber"),
			CustomerID:   uuidGen().Draw(rt, "customerID"),
			OLTID:        uuidGen().Draw(rt, "oltID"),
			OLTName:      oltNameGen().Draw(rt, "oltName"),
			PONPortIndex: provPonPortGen().Draw(rt, "ponPort"),
			TenantID:     uuidGen().Draw(rt, "tenantID"),
		}

		_ = publisher.PublishONTDecommissioned(context.Background(), payload)
		raw := getProvisioningEnvelope(t, inspector, domain.EventONTDecommissioned)

		var env struct {
			Payload json.RawMessage `json:"payload"`
		}
		if err := json.Unmarshal(raw, &env); err != nil {
			t.Fatalf("gagal decode envelope: %v", err)
		}

		var decoded domain.ONTDecommissionedPayload
		if err := json.Unmarshal(env.Payload, &decoded); err != nil {
			t.Fatalf("gagal decode payload: %v", err)
		}

		if decoded.CorrelationID == "" {
			t.Error("correlation_id kosong")
		}
		if decoded.ONTID == "" {
			t.Error("ont_id kosong")
		}
		if decoded.SerialNumber == "" {
			t.Error("serial_number kosong")
		}
		if decoded.CustomerID == "" {
			t.Error("customer_id kosong")
		}
		if decoded.OLTID == "" {
			t.Error("olt_id kosong")
		}
		if decoded.OLTName == "" {
			t.Error("olt_name kosong")
		}
		if decoded.TenantID == "" {
			t.Error("tenant_id kosong")
		}
	})
}

// TestProperty7_ONTAutoProvisionedPayload memverifikasi bahwa event ont.auto_provisioned
// memiliki semua required field.
//
// **Memvalidasi: Kebutuhan 13.4**
func TestProperty7_ONTAutoProvisionedPayload(t *testing.T) {
	logger := zerolog.Nop()

	rapid.Check(t, func(rt *rapid.T) {
		mr, client, inspector := setupProvisioningRedis(t)
		defer mr.Close()
		defer client.Close()
		defer inspector.Close()

		publisher := NewOLTEventPublisher(client, logger)

		payload := domain.ONTAutoProvisionedPayload{
			ONTID:        uuidGen().Draw(rt, "ontID"),
			SerialNumber: serialNumberGen().Draw(rt, "serialNumber"),
			CustomerID:   uuidGen().Draw(rt, "customerID"),
			OLTID:        uuidGen().Draw(rt, "oltID"),
			PONPortIndex: provPonPortGen().Draw(rt, "ponPort"),
			TenantID:     uuidGen().Draw(rt, "tenantID"),
		}

		_ = publisher.PublishONTAutoProvisioned(context.Background(), payload)
		raw := getProvisioningEnvelope(t, inspector, domain.EventONTAutoProvisioned)

		var env struct {
			Payload json.RawMessage `json:"payload"`
		}
		if err := json.Unmarshal(raw, &env); err != nil {
			t.Fatalf("gagal decode envelope: %v", err)
		}

		var decoded domain.ONTAutoProvisionedPayload
		if err := json.Unmarshal(env.Payload, &decoded); err != nil {
			t.Fatalf("gagal decode payload: %v", err)
		}

		if decoded.CorrelationID == "" {
			t.Error("correlation_id kosong")
		}
		if decoded.ONTID == "" {
			t.Error("ont_id kosong")
		}
		if decoded.SerialNumber == "" {
			t.Error("serial_number kosong")
		}
		if decoded.CustomerID == "" {
			t.Error("customer_id kosong")
		}
		if decoded.OLTID == "" {
			t.Error("olt_id kosong")
		}
		if decoded.TenantID == "" {
			t.Error("tenant_id kosong")
		}
	})
}

// TestProperty7_ONTAutoProvisionFailedPayload memverifikasi bahwa event
// ont.auto_provision_failed memiliki semua required field termasuk error_message.
//
// **Memvalidasi: Kebutuhan 13.5**
func TestProperty7_ONTAutoProvisionFailedPayload(t *testing.T) {
	logger := zerolog.Nop()

	rapid.Check(t, func(rt *rapid.T) {
		mr, client, inspector := setupProvisioningRedis(t)
		defer mr.Close()
		defer client.Close()
		defer inspector.Close()

		publisher := NewOLTEventPublisher(client, logger)

		payload := domain.ONTAutoProvisionFailedPayload{
			SerialNumber: serialNumberGen().Draw(rt, "serialNumber"),
			OLTID:        uuidGen().Draw(rt, "oltID"),
			PONPortIndex: provPonPortGen().Draw(rt, "ponPort"),
			ErrorMessage: rapid.StringMatching(`[a-z ]{5,50}`).Draw(rt, "errorMsg"),
			TenantID:     uuidGen().Draw(rt, "tenantID"),
		}

		_ = publisher.PublishONTAutoProvisionFailed(context.Background(), payload)
		raw := getProvisioningEnvelope(t, inspector, domain.EventONTAutoProvisionFail)

		var env struct {
			Payload json.RawMessage `json:"payload"`
		}
		if err := json.Unmarshal(raw, &env); err != nil {
			t.Fatalf("gagal decode envelope: %v", err)
		}

		var decoded domain.ONTAutoProvisionFailedPayload
		if err := json.Unmarshal(env.Payload, &decoded); err != nil {
			t.Fatalf("gagal decode payload: %v", err)
		}

		if decoded.CorrelationID == "" {
			t.Error("correlation_id kosong")
		}
		if decoded.SerialNumber == "" {
			t.Error("serial_number kosong")
		}
		if decoded.OLTID == "" {
			t.Error("olt_id kosong")
		}
		if decoded.ErrorMessage == "" {
			t.Error("error_message kosong")
		}
		if decoded.TenantID == "" {
			t.Error("tenant_id kosong")
		}
	})
}

// TestProperty7_ONTPortMigratedPayload memverifikasi bahwa event ont.port_migrated
// memiliki semua required field termasuk old/new port info.
//
// **Memvalidasi: Kebutuhan 13.6**
func TestProperty7_ONTPortMigratedPayload(t *testing.T) {
	logger := zerolog.Nop()

	rapid.Check(t, func(rt *rapid.T) {
		mr, client, inspector := setupProvisioningRedis(t)
		defer mr.Close()
		defer client.Close()
		defer inspector.Close()

		publisher := NewOLTEventPublisher(client, logger)

		oldPort := provPonPortGen().Draw(rt, "oldPort")
		newPort := provPonPortGen().Draw(rt, "newPort")
		oldONTIdx := provOntIndexGen().Draw(rt, "oldONTIdx")
		newONTIdx := provOntIndexGen().Draw(rt, "newONTIdx")

		payload := domain.ONTPortMigratedPayload{
			ONTID:        uuidGen().Draw(rt, "ontID"),
			SerialNumber: serialNumberGen().Draw(rt, "serialNumber"),
			OLTID:        uuidGen().Draw(rt, "oltID"),
			OldPortIndex: oldPort,
			NewPortIndex: newPort,
			OldONTIndex:  oldONTIdx,
			NewONTIndex:  newONTIdx,
			TenantID:     uuidGen().Draw(rt, "tenantID"),
		}

		_ = publisher.PublishONTPortMigrated(context.Background(), payload)
		raw := getProvisioningEnvelope(t, inspector, domain.EventONTPortMigrated)

		var env struct {
			Payload json.RawMessage `json:"payload"`
		}
		if err := json.Unmarshal(raw, &env); err != nil {
			t.Fatalf("gagal decode envelope: %v", err)
		}

		var decoded domain.ONTPortMigratedPayload
		if err := json.Unmarshal(env.Payload, &decoded); err != nil {
			t.Fatalf("gagal decode payload: %v", err)
		}

		if decoded.CorrelationID == "" {
			t.Error("correlation_id kosong")
		}
		if decoded.ONTID == "" {
			t.Error("ont_id kosong")
		}
		if decoded.SerialNumber == "" {
			t.Error("serial_number kosong")
		}
		if decoded.OLTID == "" {
			t.Error("olt_id kosong")
		}
		if decoded.TenantID == "" {
			t.Error("tenant_id kosong")
		}
	})
}

// TestProperty7_CorrelationID_AlwaysGenerated memverifikasi bahwa correlation_id
// selalu di-buat untuk semua tipe provisioning event, bahkan jika tidak diset.
//
// **Memvalidasi: Kebutuhan 13.7**
func TestProperty7_CorrelationID_AlwaysGenerated(t *testing.T) {
	logger := zerolog.Nop()

	rapid.Check(t, func(rt *rapid.T) {
		mr, client, inspector := setupProvisioningRedis(t)
		defer mr.Close()
		defer client.Close()
		defer inspector.Close()

		publisher := NewOLTEventPublisher(client, logger)

		// Kirim event TANPA correlation_id (kosong)
		payload := domain.ONTProvisionedPayload{
			CorrelationID: "", // sengaja kosong
			ONTID:         uuidGen().Draw(rt, "ontID"),
			SerialNumber:  serialNumberGen().Draw(rt, "serialNumber"),
			CustomerID:    uuidGen().Draw(rt, "customerID"),
			OLTID:         uuidGen().Draw(rt, "oltID"),
			OLTName:       oltNameGen().Draw(rt, "oltName"),
			PONPortIndex:  provPonPortGen().Draw(rt, "ponPort"),
			VLANID:        uuidGen().Draw(rt, "vlanID"),
			TenantID:      uuidGen().Draw(rt, "tenantID"),
		}

		_ = publisher.PublishONTProvisioned(context.Background(), payload)
		raw := getProvisioningEnvelope(t, inspector, domain.EventONTProvisioned)

		var env struct {
			Payload json.RawMessage `json:"payload"`
		}
		if err := json.Unmarshal(raw, &env); err != nil {
			t.Fatalf("gagal decode envelope: %v", err)
		}

		var decoded domain.ONTProvisionedPayload
		if err := json.Unmarshal(env.Payload, &decoded); err != nil {
			t.Fatalf("gagal decode payload: %v", err)
		}

		if decoded.CorrelationID == "" {
			t.Error("correlation_id harus di-generate otomatis jika kosong")
		}
	})
}
