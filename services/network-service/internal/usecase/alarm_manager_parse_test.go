// Package usecase — test cases untuk trap PDU parsing.
// Menguji parseTrapPDU dengan berbagai tipe alarm dan edge cases.
package usecase

import (
	"testing"

	"github.com/gosnmp/gosnmp"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// =============================================================================
// Test: parseTrapPDU — ONT LOS
// =============================================================================

func TestParseTrapPDU_ONTLOS(t *testing.T) {
	pdus := []gosnmp.SnmpPDU{
		{Name: ".1.3.6.1.4.1.3902.1.100", Value: []byte("ONT Loss of Signal detected")},
	}
	alarm := parseTrapPDU(pdus, "10.0.0.1")
	if alarm == nil {
		t.Fatal("expected alarm, got nil")
	}
	if alarm.AlarmType != domain.AlarmTypeONTLOS {
		t.Fatalf("expected alarm type %q, got %q", domain.AlarmTypeONTLOS, alarm.AlarmType)
	}
	if alarm.Severity != domain.SeverityCritical {
		t.Fatalf("expected severity %q, got %q", domain.SeverityCritical, alarm.Severity)
	}
	if alarm.Source != domain.AlarmSourceTrap {
		t.Fatalf("expected source trap, got %q", alarm.Source)
	}
}

// =============================================================================
// Test: parseTrapPDU — Dying Gasp
// =============================================================================

func TestParseTrapPDU_DyingGasp(t *testing.T) {
	pdus := []gosnmp.SnmpPDU{
		{Name: ".1.3.6.1.6.3.1.1.4.1.0", Value: []byte("ONT Dying Gasp alarm")},
	}
	alarm := parseTrapPDU(pdus, "10.0.0.1")
	if alarm == nil {
		t.Fatal("expected alarm, got nil")
	}
	if alarm.AlarmType != domain.AlarmTypeONTDyingGasp {
		t.Fatalf("expected alarm type %q, got %q", domain.AlarmTypeONTDyingGasp, alarm.AlarmType)
	}
	if alarm.Severity != domain.SeverityCritical {
		t.Fatalf("expected severity critical, got %q", alarm.Severity)
	}
}

// =============================================================================
// Test: parseTrapPDU — PON Port Down
// =============================================================================

func TestParseTrapPDU_PONPortDown(t *testing.T) {
	pdus := []gosnmp.SnmpPDU{
		{Name: ".1.3.6.1.6.3.1.1.4.1.0", Value: []byte("PON Port Down on port 3")},
	}
	alarm := parseTrapPDU(pdus, "10.0.0.1")
	if alarm == nil {
		t.Fatal("expected alarm, got nil")
	}
	if alarm.AlarmType != domain.AlarmTypePONPortDown {
		t.Fatalf("expected alarm type %q, got %q", domain.AlarmTypePONPortDown, alarm.AlarmType)
	}
	if alarm.Severity != domain.SeverityMajor {
		t.Fatalf("expected severity major, got %q", alarm.Severity)
	}
}

// =============================================================================
// Test: parseTrapPDU — High Temperature
// =============================================================================

func TestParseTrapPDU_HighTemperature(t *testing.T) {
	pdus := []gosnmp.SnmpPDU{
		{Name: ".1.3.6.1.4.1.3902.1.200", Value: "High Temperature warning on slot 1"},
	}
	alarm := parseTrapPDU(pdus, "10.0.0.1")
	if alarm == nil {
		t.Fatal("expected alarm, got nil")
	}
	if alarm.AlarmType != domain.AlarmTypeHighTemperature {
		t.Fatalf("expected alarm type %q, got %q", domain.AlarmTypeHighTemperature, alarm.AlarmType)
	}
	if alarm.Severity != domain.SeverityMajor {
		t.Fatalf("expected severity major, got %q", alarm.Severity)
	}
}

// =============================================================================
// Test: parseTrapPDU — Power Failure
// =============================================================================

func TestParseTrapPDU_PowerFailure(t *testing.T) {
	pdus := []gosnmp.SnmpPDU{
		{Name: ".1.3.6.1.4.1.3902.1.300", Value: []byte("Power Failure detected")},
	}
	alarm := parseTrapPDU(pdus, "10.0.0.1")
	if alarm == nil {
		t.Fatal("expected alarm, got nil")
	}
	if alarm.AlarmType != domain.AlarmTypePowerFailure {
		t.Fatalf("expected alarm type %q, got %q", domain.AlarmTypePowerFailure, alarm.AlarmType)
	}
	if alarm.Severity != domain.SeverityCritical {
		t.Fatalf("expected severity critical, got %q", alarm.Severity)
	}
}

// =============================================================================
// Test: parseTrapPDU — Signal Degraded
// =============================================================================

func TestParseTrapPDU_SignalDegraded(t *testing.T) {
	pdus := []gosnmp.SnmpPDU{
		{Name: ".1.3.6.1.4.1.3902.1.400", Value: []byte("ONT Signal Degraded on port 2")},
	}
	alarm := parseTrapPDU(pdus, "10.0.0.1")
	if alarm == nil {
		t.Fatal("expected alarm, got nil")
	}
	if alarm.AlarmType != domain.AlarmTypeONTSignalDegraded {
		t.Fatalf("expected alarm type %q, got %q", domain.AlarmTypeONTSignalDegraded, alarm.AlarmType)
	}
	if alarm.Severity != domain.SeverityWarning {
		t.Fatalf("expected severity warning, got %q", alarm.Severity)
	}
}

// =============================================================================
// Test: parseTrapPDU — PDU kosong
// =============================================================================

func TestParseTrapPDU_EmptyPDU(t *testing.T) {
	alarm := parseTrapPDU(nil, "10.0.0.1")
	if alarm != nil {
		t.Fatal("expected nil for nil PDU, got alarm")
	}

	alarm = parseTrapPDU([]gosnmp.SnmpPDU{}, "10.0.0.1")
	if alarm != nil {
		t.Fatal("expected nil for empty PDU slice, got alarm")
	}
}

// =============================================================================
// Test: parseTrapPDU — unrecognized trap returns generic alarm
// =============================================================================

func TestParseTrapPDU_UnrecognizedTrap(t *testing.T) {
	pdus := []gosnmp.SnmpPDU{
		{Name: ".1.3.6.1.2.1.1.3.0", Value: 12345},
	}
	alarm := parseTrapPDU(pdus, "10.0.0.1")
	if alarm == nil {
		t.Fatal("expected generic alarm, got nil")
	}
	if alarm.Severity != domain.SeverityWarning {
		t.Fatalf("expected severity warning for unrecognized trap, got %q", alarm.Severity)
	}
}
