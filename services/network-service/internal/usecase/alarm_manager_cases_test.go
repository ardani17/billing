// Package usecase — test cases untuk alarm manager business logic.
// Menguji PollAlarms, GetAlarms, dan PurgeOldAlarms.
package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// =============================================================================
// Test: PollAlarms menyimpan alarm dan publish event
// =============================================================================

func TestAlarmManager_PollAlarmsSavesAndPublishes(t *testing.T) {
	port := 1
	adapter := &amAdapter{
		alarms: []domain.OLTAlarm{
			{AlarmType: domain.AlarmTypeONTLOS, Severity: domain.SeverityCritical, PONPortIndex: &port, Message: "ONT LOS detected"},
			{AlarmType: domain.AlarmTypeHighTemperature, Severity: domain.SeverityMajor, Message: "Suhu tinggi"},
		},
	}
	am, alarmRepo, eventPub := newTestAlarmManager(adapter)
	ctx := context.Background()

	alarms, err := am.PollAlarms(ctx, "olt-001")
	if err != nil {
		t.Fatalf("PollAlarms gagal: %v", err)
	}
	if len(alarms) != 2 {
		t.Fatalf("expected 2 alarms, got %d", len(alarms))
	}

	// Verifikasi alarm disimpan ke repo
	alarmRepo.mu.Lock()
	defer alarmRepo.mu.Unlock()
	if len(alarmRepo.created) != 2 {
		t.Fatalf("expected 2 alarm records created, got %d", len(alarmRepo.created))
	}
	if alarmRepo.created[0].OLTID != "olt-001" {
		t.Fatalf("expected olt_id 'olt-001', got %q", alarmRepo.created[0].OLTID)
	}
	if alarmRepo.created[0].TenantID != "tenant-001" {
		t.Fatalf("expected tenant_id 'tenant-001', got %q", alarmRepo.created[0].TenantID)
	}
	if alarmRepo.created[0].Source != domain.AlarmSourcePolling {
		t.Fatalf("expected source 'polling', got %q", alarmRepo.created[0].Source)
	}

	// Verifikasi event dipublish
	eventPub.mu.Lock()
	defer eventPub.mu.Unlock()
	if len(eventPub.alarms) != 2 {
		t.Fatalf("expected 2 alarm events published, got %d", len(eventPub.alarms))
	}
	if eventPub.alarms[0].OLTID != "olt-001" {
		t.Fatalf("expected event olt_id 'olt-001', got %q", eventPub.alarms[0].OLTID)
	}
}

// =============================================================================
// Test: PollAlarms OLT tidak ditemukan
// =============================================================================

func TestAlarmManager_PollAlarmsOLTNotFound(t *testing.T) {
	am, _, _ := newTestAlarmManager(&amAdapter{})
	ctx := context.Background()

	_, err := am.PollAlarms(ctx, "olt-nonexistent")
	if err != domain.ErrOLTNotFound {
		t.Fatalf("expected ErrOLTNotFound, got %v", err)
	}
}

// =============================================================================
// Test: GetAlarms mendelegasikan ke repo
// =============================================================================

func TestAlarmManager_GetAlarmsDelegatesToRepo(t *testing.T) {
	am, alarmRepo, _ := newTestAlarmManager(&amAdapter{})
	ctx := context.Background()

	expected := &domain.AlarmListResult{
		Data:       []*domain.OLTAlarmRecord{{ID: "alarm-001", AlarmType: domain.AlarmTypeONTLOS}},
		Total:      1,
		Page:       1,
		PageSize:   20,
		TotalPages: 1,
	}
	alarmRepo.listResult = expected

	params := domain.AlarmListParams{Page: 1, PageSize: 20, Severity: domain.SeverityCritical}
	result, err := am.GetAlarms(ctx, "olt-001", params)
	if err != nil {
		t.Fatalf("GetAlarms gagal: %v", err)
	}
	if result.Total != 1 {
		t.Fatalf("expected total 1, got %d", result.Total)
	}
	if result.Data[0].ID != "alarm-001" {
		t.Fatalf("expected alarm ID 'alarm-001', got %q", result.Data[0].ID)
	}
}

// =============================================================================
// Test: PurgeOldAlarms dengan threshold 90 hari
// =============================================================================

func TestAlarmManager_PurgeOldAlarms90Days(t *testing.T) {
	am, alarmRepo, _ := newTestAlarmManager(&amAdapter{})
	ctx := context.Background()

	beforeCall := time.Now()
	count, err := am.PurgeOldAlarms(ctx)
	if err != nil {
		t.Fatalf("PurgeOldAlarms gagal: %v", err)
	}
	if count != 5 {
		t.Fatalf("expected purge count 5, got %d", count)
	}

	alarmRepo.mu.Lock()
	defer alarmRepo.mu.Unlock()
	if alarmRepo.purgedBefore == nil {
		t.Fatal("PurgeOlderThan tidak dipanggil")
	}

	// Verifikasi threshold ~90 hari (toleransi 1 menit)
	expectedBefore := beforeCall.Add(-90 * 24 * time.Hour)
	diff := alarmRepo.purgedBefore.Sub(expectedBefore)
	if diff < -time.Minute || diff > time.Minute {
		t.Fatalf("purge threshold tidak sesuai 90 hari, diff: %v", diff)
	}
}
