package usecase

import (
	"fmt"
	"testing"

	"pgregory.net/rapid"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// =============================================================================
// =============================================================================

// TestProperty_SyncComparisonCorrectness memverifikasi bahwa untuk sembarang
// dua atur ONT records, compareONTSets mengklasifikasikan setiap ONT ke tepat
// satu kategori. Total count semua kategori sama dengan union kedua atur.
// Tidak ada ONT yang muncul di lebih dari satu kategori.
//
// **Memvalidasi: Kebutuhan 13.2, 13.3, 13.4, 13.5, 13.6**
func TestProperty_SyncComparisonCorrectness(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Buat atur ONT dari OLT dengan serial number unik
		oltCount := rapid.IntRange(0, 20).Draw(t, "oltCount")
		oltONTs := generateONTSet(t, "olt", oltCount)

		// Buat atur ONT dari DB - sebagian overlap dengan OLT
		dbCount := rapid.IntRange(0, 20).Draw(t, "dbCount")
		dbONTs := generateONTSet(t, "db", dbCount)

		// Tambahkan beberapa ONT dari OLT ke DB (untuk overlap)
		overlapCount := rapid.IntRange(0, min(oltCount, 10)).Draw(t, "overlapCount")
		for i := 0; i < overlapCount && i < len(oltONTs); i++ {
			// Salin ONT dari OLT ke DB - sebagian identik, sebagian berbeda
			dbCopy := oltONTs[i]
			if rapid.Bool().Draw(t, fmt.Sprintf("modify_%d", i)) {
				// Ubah data agar masuk kategori "updated"
				dbCopy.Status = "offline"
				if oltONTs[i].Status == "offline" {
					dbCopy.Status = "online"
				}
			}
			dbONTs = append(dbONTs, dbCopy)
		}

		// Deduplikasi DB ONTs berdasarkan serial number (ambil yang terakhir)
		dbONTs = deduplicateONTs(dbONTs)

		result := compareONTSets(oltONTs, dbONTs)

		// Properti 1: Total count semua kategori = union kedua atur
		unionSize := countUnion(oltONTs, dbONTs)
		totalClassified := len(result.Unmanaged) + len(result.Missing) +
			len(result.Updated) + len(result.Synced)

		if totalClassified != unionSize {
			t.Errorf(
				"total klasifikasi (%d) != union size (%d): unmanaged=%d missing=%d updated=%d synced=%d",
				totalClassified, unionSize,
				len(result.Unmanaged), len(result.Missing),
				len(result.Updated), len(result.Synced),
			)
		}

		// Properti 2: Tidak ada ONT yang muncul di lebih dari satu kategori
		seen := make(map[string]string)
		checkDuplicate := func(onts []domain.ONTPortStatus, category string) {
			for _, ont := range onts {
				if prev, exists := seen[ont.SerialNumber]; exists {
					t.Errorf(
						"ONT %q muncul di kategori %q dan %q",
						ont.SerialNumber, prev, category,
					)
				}
				seen[ont.SerialNumber] = category
			}
		}
		checkDuplicate(result.Unmanaged, "unmanaged")
		checkDuplicate(result.Missing, "missing")
		checkDuplicate(result.Updated, "updated")
		checkDuplicate(result.Synced, "synced")

		// Properti 3: Unmanaged hanya berisi ONT yang ada di OLT tapi tidak di DB
		dbSNs := serialNumberSet(dbONTs)
		for _, ont := range result.Unmanaged {
			if dbSNs[ont.SerialNumber] {
				t.Errorf("ONT %q di unmanaged tapi ada di DB set", ont.SerialNumber)
			}
		}

		// Properti 4: Missing hanya berisi ONT yang ada di DB tapi tidak di OLT
		oltSNs := serialNumberSet(oltONTs)
		for _, ont := range result.Missing {
			if oltSNs[ont.SerialNumber] {
				t.Errorf("ONT %q di missing tapi ada di OLT set", ont.SerialNumber)
			}
		}

		// Properti 5: Updated dan Synced hanya berisi ONT yang ada di kedua atur
		for _, ont := range result.Updated {
			if !oltSNs[ont.SerialNumber] || !dbSNs[ont.SerialNumber] {
				t.Errorf("ONT %q di updated tapi tidak ada di kedua set", ont.SerialNumber)
			}
		}
		for _, ont := range result.Synced {
			if !oltSNs[ont.SerialNumber] || !dbSNs[ont.SerialNumber] {
				t.Errorf("ONT %q di synced tapi tidak ada di kedua set", ont.SerialNumber)
			}
		}
	})
}

// =============================================================================
// =============================================================================

// generateONTSet menghasilkan slice ONTStatus dengan serial number unik.
func generateONTSet(t *rapid.T, prefix string, count int) []domain.ONTPortStatus {
	onts := make([]domain.ONTPortStatus, count)
	for i := 0; i < count; i++ {
		onts[i] = domain.ONTPortStatus{
			ONTIndex:     i,
			SerialNumber: fmt.Sprintf("%s-SN-%04d", prefix, i),
			Name:         fmt.Sprintf("ONT-%s-%d", prefix, i),
			Status:       "online",
			RxSignalDBm:  -20.0,
		}
	}
	return onts
}

// deduplicateONTs menghapus duplikat berdasarkan serial number (ambil terakhir).
func deduplicateONTs(onts []domain.ONTPortStatus) []domain.ONTPortStatus {
	seen := make(map[string]int)
	for i, ont := range onts {
		seen[ont.SerialNumber] = i
	}
	result := make([]domain.ONTPortStatus, 0, len(seen))
	for _, idx := range seen {
		result = append(result, onts[idx])
	}
	return result
}

// countUnion menghitung jumlah serial number unik dari gabungan dua atur.
func countUnion(a, b []domain.ONTPortStatus) int {
	union := make(map[string]bool)
	for _, ont := range a {
		union[ont.SerialNumber] = true
	}
	for _, ont := range b {
		union[ont.SerialNumber] = true
	}
	return len(union)
}

// serialNumberSet membuat atur serial number dari slice ONTStatus.
func serialNumberSet(onts []domain.ONTPortStatus) map[string]bool {
	set := make(map[string]bool, len(onts))
	for _, ont := range onts {
		set[ont.SerialNumber] = true
	}
	return set
}

// min mengembalikan nilai terkecil dari dua integer.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
