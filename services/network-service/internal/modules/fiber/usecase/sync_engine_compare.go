// Package usecase berisi implementasi business logic untuk network-service.
// File ini berisi fungsi perbandingan ONT atur untuk sync engine.
// compareONTSets adalah pure function yang mengklasifikasikan ONT ke dalam
// kategori: unmanaged, missing, updated, synced.
package usecase

import "github.com/ispboss/ispboss/services/network-service/internal/domain"

// ontCompareResult berisi hasil perbandingan dua atur ONT.
// Setiap ONT diklasifikasikan ke tepat satu kategori.
type ontCompareResult struct {
	Unmanaged []domain.ONTPortStatus // ada di OLT tapi tidak di DB
	Missing   []domain.ONTPortStatus // ada di DB tapi tidak di OLT
	Updated   []domain.ONTPortStatus // ada di keduanya tapi data berbeda
	Synced    []domain.ONTPortStatus // ada di keduanya dengan data identik
}

// compareONTSets membandingkan dua atur ONT (dari OLT dan dari DB).
// Mengklasifikasikan setiap ONT ke tepat satu kategori berdasarkan serial number.
// OLT = sumber of truth untuk data fisik.
//
// Klasifikasi:
//   - Unmanaged: serial number ada di OLT tapi tidak di DB
//   - Missing: serial number ada di DB tapi tidak di OLT
//   - Updated: serial number ada di keduanya tapi data berbeda
//   - Synced: serial number ada di keduanya dengan data identik
func compareONTSets(oltONTs, dbONTs []domain.ONTPortStatus) ontCompareResult {
	result := ontCompareResult{}

	// Bangun map dari DB ONTs berdasarkan serial number
	dbMap := make(map[string]domain.ONTPortStatus, len(dbONTs))
	for _, ont := range dbONTs {
		dbMap[ont.SerialNumber] = ont
	}

	// Set untuk melacak serial number yang sudah diproses dari OLT
	processed := make(map[string]bool, len(oltONTs))

	// Iterasi ONT dari OLT
	for _, oltONT := range oltONTs {
		processed[oltONT.SerialNumber] = true

		dbONT, exists := dbMap[oltONT.SerialNumber]
		if !exists {
			// Ada di OLT tapi tidak di DB -> unmanaged
			result.Unmanaged = append(result.Unmanaged, oltONT)
			continue
		}

		// Ada di keduanya - bandingkan data
		if ontDataDiffers(oltONT, dbONT) {
			result.Updated = append(result.Updated, oltONT)
		} else {
			result.Synced = append(result.Synced, oltONT)
		}
	}

	// Cari ONT yang ada di DB tapi tidak di OLT -> missing
	for _, dbONT := range dbONTs {
		if !processed[dbONT.SerialNumber] {
			result.Missing = append(result.Missing, dbONT)
		}
	}

	return result
}

// ontDataDiffers memeriksa apakah data ONT dari OLT berbeda dengan data di DB.
// Membandingkan field yang relevan: status, port index, dan name.
func ontDataDiffers(oltONT, dbONT domain.ONTPortStatus) bool {
	if oltONT.Status != dbONT.Status {
		return true
	}
	if oltONT.PONPortIndex != dbONT.PONPortIndex {
		return true
	}
	if oltONT.ONTIndex != dbONT.ONTIndex {
		return true
	}
	if oltONT.Name != dbONT.Name {
		return true
	}
	return false
}
