// Package adapter — Helper functions untuk ZTEAdapter.
// Parsing sysDescr, konversi SNMP result, dan utilitas OID.
package adapter

import (
	"fmt"
	"strings"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// zteParseSystemDescr mengekstrak model dan firmware dari sysDescr ZTE.
// Contoh input: "ZTE ZXA10 C320 Version V2.1.0 Software"
// Output: model="C320", firmware="V2.1.0"
func zteParseSystemDescr(sysDescr string) (model, firmware string) {
	upper := strings.ToUpper(sysDescr)

	// Deteksi model ZTE dari sysDescr
	models := []string{"C600", "C320", "C300", "C220"}
	for _, m := range models {
		if strings.Contains(upper, m) {
			model = m
			break
		}
	}

	// Deteksi firmware version (pattern "V" diikuti angka)
	parts := strings.Fields(sysDescr)
	for _, p := range parts {
		if len(p) > 1 && (p[0] == 'V' || p[0] == 'v') && p[1] >= '0' && p[1] <= '9' {
			firmware = p
			break
		}
	}

	return model, firmware
}

// snmpResultToString mengkonversi SNMPResult ke string.
func snmpResultToString(r domain.SNMPResult) string {
	if r.Value == nil {
		return ""
	}
	switch v := r.Value.(type) {
	case string:
		return v
	case []byte:
		return string(v)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// snmpResultToInt64 mengkonversi SNMPResult ke int64.
func snmpResultToInt64(r domain.SNMPResult) int64 {
	if r.Value == nil {
		return 0
	}
	switch v := r.Value.(type) {
	case int64:
		return v
	case int:
		return int64(v)
	case int32:
		return int64(v)
	case uint32:
		return int64(v)
	case uint64:
		return int64(v)
	case float64:
		return int64(v)
	default:
		return 0
	}
}

// oidSuffix mengekstrak suffix OID setelah prefix tertentu.
// Contoh: oid=".1.3.6.1.2.1.2.2.1.7.100", prefix="1.3.6.1.2.1.2.2.1.7" → "100"
func oidSuffix(oid, prefix string) string {
	// Normalisasi: hapus leading dot
	cleanOID := strings.TrimPrefix(oid, ".")
	cleanPrefix := strings.TrimPrefix(prefix, ".")
	if strings.HasPrefix(cleanOID, cleanPrefix+".") {
		return strings.TrimPrefix(cleanOID, cleanPrefix+".")
	}
	return oid
}

// ifStatusToString mengkonversi SNMP ifAdminStatus/ifOperStatus integer ke string.
// 1 = up, 2 = down, 3 = testing
func ifStatusToString(val int64) string {
	switch val {
	case 1:
		return "up"
	case 2:
		return "down"
	case 3:
		return "testing"
	default:
		return "unknown"
	}
}

// classifySFPStatus mengklasifikasikan status SFP berdasarkan suhu.
// Normal: < 45°C, Warm: 45-60°C, Degraded: > 60°C
func classifySFPStatus(tempCelsius float64) string {
	switch {
	case tempCelsius <= 0:
		return "empty"
	case tempCelsius < 45:
		return "normal"
	case tempCelsius < 60:
		return "warm"
	default:
		return "degraded"
	}
}
