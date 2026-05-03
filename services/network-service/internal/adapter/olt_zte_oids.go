// Package adapter — Konstanta OID SNMP khusus ZTE OLT.
// Referensi: ZTE ZXA10 C300/C320/C600 MIB documentation.
package adapter

// --- Standard MIB OIDs ---
// OID standar yang digunakan oleh semua brand OLT.

const (
	// zteSysDescr adalah OID sysDescr untuk informasi sistem.
	zteSysDescr = "1.3.6.1.2.1.1.1.0"

	// zteSysUpTime adalah OID sysUpTime untuk uptime dalam timeticks.
	zteSysUpTime = "1.3.6.1.2.1.1.3.0"

	// zteSysName adalah OID sysName untuk nama perangkat.
	zteSysName = "1.3.6.1.2.1.1.5.0"

	// zteIfAdminStatus adalah OID base ifAdminStatus untuk status admin interface.
	zteIfAdminStatus = "1.3.6.1.2.1.2.2.1.7"

	// zteIfOperStatus adalah OID base ifOperStatus untuk status operasional interface.
	zteIfOperStatus = "1.3.6.1.2.1.2.2.1.8"
)

// --- ZTE ONU Management Base OIDs ---
// Base: 1.3.6.1.4.1.3902.1012.3.28.1.1.{field}.{oltId}.{onuId}

const (
	// zteONUMgmtBase adalah base OID untuk manajemen ONU ZTE.
	zteONUMgmtBase = "1.3.6.1.4.1.3902.1012.3.28.1.1"

	// zteONUTypeName adalah field .1 — tipe ONU (model).
	zteONUTypeName = "1.3.6.1.4.1.3902.1012.3.28.1.1.1"

	// zteONUName adalah field .2 — nama ONU.
	zteONUName = "1.3.6.1.4.1.3902.1012.3.28.1.1.2"

	// zteONUDescription adalah field .3 — deskripsi ONU.
	zteONUDescription = "1.3.6.1.4.1.3902.1012.3.28.1.1.3"

	// zteONUSerialNumber adalah field .5 — serial number ONU.
	zteONUSerialNumber = "1.3.6.1.4.1.3902.1012.3.28.1.1.5"

	// zteONUTargetState adalah field .8 — target state ONU.
	zteONUTargetState = "1.3.6.1.4.1.3902.1012.3.28.1.1.8"

	// zteONURowStatus adalah field .9 — row status ONU.
	zteONURowStatus = "1.3.6.1.4.1.3902.1012.3.28.1.1.9"
)

// --- ZTE ONU Distance OID ---

const (
	// zteONUDistance adalah base OID untuk jarak ONU dalam meter.
	// Format: {base}.{oltId}.{onuId}
	zteONUDistance = "1.3.6.1.4.1.3902.1012.3.11.4.1.2"
)

// --- ZTE PON Port Traffic Stats OIDs ---
// Base: 1.3.6.1.4.1.3902.1015.1010.5.4.1.{field}.{oltId}

const (
	// ztePONStatsBase adalah base OID untuk statistik traffic PON port.
	ztePONStatsBase = "1.3.6.1.4.1.3902.1015.1010.5.4.1"

	// ztePONRxOctets adalah field .2 — total bytes diterima.
	ztePONRxOctets = "1.3.6.1.4.1.3902.1015.1010.5.4.1.2"

	// ztePONRxPkts adalah field .3 — total paket diterima.
	ztePONRxPkts = "1.3.6.1.4.1.3902.1015.1010.5.4.1.3"

	// ztePONTxOctets adalah field .17 — total bytes dikirim.
	ztePONTxOctets = "1.3.6.1.4.1.3902.1015.1010.5.4.1.17"

	// ztePONTxPkts adalah field .18 — total paket dikirim.
	ztePONTxPkts = "1.3.6.1.4.1.3902.1015.1010.5.4.1.18"
)

// --- ZTE Alarm OIDs ---

const (
	// zteAlarmBase adalah base OID untuk alarm aktif ZTE.
	zteAlarmBase = "1.3.6.1.4.1.3902.1012.3.50.12.1"
)

// --- ZTE SFP OIDs ---

const (
	// zteSFPTxPower adalah base OID untuk TX power SFP module.
	zteSFPTxPower = "1.3.6.1.4.1.3902.1015.1010.11.2.1.2"

	// zteSFPRxPower adalah base OID untuk RX power SFP module.
	zteSFPRxPower = "1.3.6.1.4.1.3902.1015.1010.11.2.1.3"

	// zteSFPTemperature adalah base OID untuk suhu SFP module.
	zteSFPTemperature = "1.3.6.1.4.1.3902.1015.1010.11.2.1.1"
)

// --- ZTE ONU Signal OIDs ---

const (
	// zteONURxPower adalah base OID untuk RX power ONU (signal level).
	zteONURxPower = "1.3.6.1.4.1.3902.1012.3.50.12.1.1.10"

	// zteONUTxPower adalah base OID untuk TX power ONU.
	zteONUTxPower = "1.3.6.1.4.1.3902.1012.3.50.12.1.1.14"
)

// --- ZTE Index Calculation ---

// zteCalculateOLTIndex menghitung oltId dari board dan pon number.
// Formula: (1 << 28) | (0 << 24) | (board << 16) | (pon << 8)
// Contoh: board=0, pon=0 → 0x10000000 = 268435456
func zteCalculateOLTIndex(board, pon int) int {
	return (1 << 28) | (0 << 24) | (board << 16) | (pon << 8)
}
