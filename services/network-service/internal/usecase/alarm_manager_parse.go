// Package usecase berisi implementasi business logic untuk network-service.
// File ini berisi logika parsing SNMP trap PDU menjadi alarm OLT.
package usecase

import (
	"strings"

	"github.com/gosnmp/gosnmp"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// parseTrapPDU menganalisis PDU dari SNMP trap dan mengembalikan OLTAlarm.
// Mendeteksi tipe alarm berdasarkan OID dan value dalam PDU.
// Mengembalikan nil jika PDU kosong.
func parseTrapPDU(pdus []gosnmp.SnmpPDU, sourceIP string) *domain.OLTAlarm {
	if len(pdus) == 0 {
		return nil
	}

	alarm := &domain.OLTAlarm{
		Source:  domain.AlarmSourceTrap,
		Message: "SNMP trap dari " + sourceIP,
	}

	// Analisis setiap PDU untuk menentukan tipe alarm
	for _, pdu := range pdus {
		oid := pdu.Name
		val := extractPDUString(pdu)

		switch {
		case strings.Contains(oid, "1.3.6.1.4.1") && containsAny(val, "los", "loss of signal"):
			alarm.AlarmType = domain.AlarmTypeONTLOS
			alarm.Severity = domain.SeverityCritical
			alarm.Message = "ONT Loss of Signal: " + val
		case containsAny(val, "power failure", "power supply"):
			alarm.AlarmType = domain.AlarmTypePowerFailure
			alarm.Severity = domain.SeverityCritical
			alarm.Message = "Power Failure: " + val
		case containsAny(val, "dying gasp", "dyinggasp"):
			alarm.AlarmType = domain.AlarmTypeONTDyingGasp
			alarm.Severity = domain.SeverityCritical
			alarm.Message = "ONT Dying Gasp: " + val
		case containsAny(val, "pon port down", "pon down", "link down"):
			alarm.AlarmType = domain.AlarmTypePONPortDown
			alarm.Severity = domain.SeverityMajor
			alarm.Message = "PON Port Down: " + val
		case containsAny(val, "temperature", "high temp", "overheat"):
			alarm.AlarmType = domain.AlarmTypeHighTemperature
			alarm.Severity = domain.SeverityMajor
			alarm.Message = "High Temperature: " + val
		case containsAny(val, "signal degraded", "signal degrade", "low signal"):
			alarm.AlarmType = domain.AlarmTypeONTSignalDegraded
			alarm.Severity = domain.SeverityWarning
			alarm.Message = "ONT Signal Degraded: " + val
		}

		if alarm.AlarmType != "" {
			return alarm
		}
	}

	// Jika tidak ada PDU yang cocok, kembalikan alarm generik
	alarm.AlarmType = domain.AlarmTypeONTLOS
	alarm.Severity = domain.SeverityWarning
	alarm.Message = "Unrecognized trap dari " + sourceIP
	return alarm
}

// extractPDUString mengekstrak nilai string dari SnmpPDU.
func extractPDUString(pdu gosnmp.SnmpPDU) string {
	switch v := pdu.Value.(type) {
	case string:
		return v
	case []byte:
		return string(v)
	default:
		return ""
	}
}

// containsAny memeriksa apakah string mengandung salah satu substring (case-insensitive).
func containsAny(s string, substrs ...string) bool {
	lower := strings.ToLower(s)
	for _, sub := range substrs {
		if strings.Contains(lower, sub) {
			return true
		}
	}
	return false
}
