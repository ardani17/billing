package domain

import "errors"

// =============================================================================
// Domain Errors — error untuk modul pengeluaran, laporan, dan KPI
// =============================================================================

var (
	// ErrExpenseNotFound dikembalikan saat pengeluaran tidak ditemukan.
	ErrExpenseNotFound = errors.New("pengeluaran tidak ditemukan")

	// ErrExpenseCategoryNotFound dikembalikan saat kategori pengeluaran tidak ditemukan.
	ErrExpenseCategoryNotFound = errors.New("kategori pengeluaran tidak ditemukan")

	// ErrCategoryHasExpenses dikembalikan saat kategori masih memiliki pengeluaran.
	ErrCategoryHasExpenses = errors.New("kategori masih memiliki pengeluaran")

	// ErrCategoryNameDuplicate dikembalikan saat nama kategori sudah ada di tenant.
	ErrCategoryNameDuplicate = errors.New("nama kategori sudah ada")

	// ErrReportScheduleNotFound dikembalikan saat jadwal laporan tidak ditemukan.
	ErrReportScheduleNotFound = errors.New("jadwal laporan tidak ditemukan")

	// ErrReportJobNotFound dikembalikan saat job export tidak ditemukan.
	ErrReportJobNotFound = errors.New("job export tidak ditemukan")

	// ErrTemplateNotFound dikembalikan saat template laporan tidak ditemukan.
	ErrTemplateNotFound = errors.New("template laporan tidak ditemukan")

	// ErrKPITargetNotFound dikembalikan saat target KPI tidak ditemukan.
	ErrKPITargetNotFound = errors.New("target KPI tidak ditemukan")

	// ErrInsufficientData dikembalikan saat data historis belum cukup untuk proyeksi.
	ErrInsufficientData = errors.New("data historis belum cukup untuk proyeksi")

	// ErrInvalidReportType dikembalikan saat tipe laporan tidak valid.
	ErrInvalidReportType = errors.New("tipe laporan tidak valid")

	// ErrInvalidExportFormat dikembalikan saat format export tidak valid.
	ErrInvalidExportFormat = errors.New("format export tidak valid")

	// ErrMaxMetricsExceeded dikembalikan saat jumlah metrik melebihi batas maksimal.
	ErrMaxMetricsExceeded = errors.New("maksimal 3 metrik per laporan custom")
)
