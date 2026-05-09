package domain

import "time"

// =============================================================================
// Prorate Calculation - perhitungan biaya prorate untuk pelanggan baru
// dan perubahan paket di tengah siklus billing
// =============================================================================

// daysPerMonth adalah jumlah hari tetap per bulan untuk perhitungan prorate.
// Menggunakan 30 hari tetap untuk menyederhanakan kalkulasi.
const daysPerMonth = 30

// CalculateProrate menghitung biaya prorate berdasarkan harga bulanan dan sisa hari.
// Menggunakan 30 hari tetap per bulan. Hasil dibulatkan ke atas ke kelipatan Rp 500.
// Rumus: RoundUpTo500(monthlyPrice * remainingDays / 30)
func CalculateProrate(monthlyPrice int64, remainingDays int) int64 {
	if monthlyPrice <= 0 || remainingDays <= 0 {
		return 0
	}
	raw := monthlyPrice * int64(remainingDays) / daysPerMonth
	return RoundUpTo500(raw)
}

// CalculateProrateCredit menghitung kredit prorate untuk downgrade paket.
// Menggunakan 30 hari tetap per bulan. Hasil dibulatkan ke bawah ke kelipatan Rp 500.
// Rumus: RoundDownTo500(monthlyPrice * remainingDays / 30)
func CalculateProrateCredit(monthlyPrice int64, remainingDays int) int64 {
	if monthlyPrice <= 0 || remainingDays <= 0 {
		return 0
	}
	raw := monthlyPrice * int64(remainingDays) / daysPerMonth
	return RoundDownTo500(raw)
}

// RoundUpTo500 membulatkan ke atas ke kelipatan Rp 500 terdekat.
// Jika nominal sudah kelipatan 500, dikembalikan apa adanya.
// Jika nominal <= 0, dikembalikan 0.
func RoundUpTo500(amount int64) int64 {
	if amount <= 0 {
		return 0
	}
	remainder := amount % 500
	if remainder == 0 {
		return amount
	}
	return amount + (500 - remainder)
}

// RoundDownTo500 membulatkan ke bawah ke kelipatan Rp 500 terdekat.
// Jika nominal sudah kelipatan 500, dikembalikan apa adanya.
// Jika nominal <= 0, dikembalikan 0.
func RoundDownTo500(amount int64) int64 {
	if amount <= 0 {
		return 0
	}
	return amount - (amount % 500)
}

// CalculateRemainingDays menghitung sisa hari dari changeDate ke nextDueDate.
// Hasil di-clamp antara 0 dan 30.
// Jika hasilnya <= 0, kembalikan 0. Jika > 30, kembalikan 30.
func CalculateRemainingDays(changeDate, nextDueDate time.Time) int {
	// Normalisasi ke awal hari (truncate waktu) agar perhitungan konsisten
	changeDay := time.Date(changeDate.Year(), changeDate.Month(), changeDate.Day(), 0, 0, 0, 0, time.UTC)
	dueDay := time.Date(nextDueDate.Year(), nextDueDate.Month(), nextDueDate.Day(), 0, 0, 0, 0, time.UTC)

	days := int(dueDay.Sub(changeDay).Hours() / 24)

	if days <= 0 {
		return 0
	}
	if days > daysPerMonth {
		return daysPerMonth
	}
	return days
}
