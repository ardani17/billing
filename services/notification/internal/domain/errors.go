package domain

import "errors"

var (
	// ErrTemplateNotFound dikembalikan saat template tidak ditemukan
	ErrTemplateNotFound = errors.New("template tidak ditemukan")

	// ErrTemplateSlugExists dikembalikan saat slug template sudah ada di tenant yang sama
	ErrTemplateSlugExists = errors.New("slug template sudah ada")

	// ErrTemplateNotDeletable dikembalikan saat template bawaan tidak bisa dihapus
	ErrTemplateNotDeletable = errors.New("template default tidak bisa dihapus")

	// ErrConfigNotFound dikembalikan saat konfigurasi notifikasi tidak ditemukan untuk tenant
	ErrConfigNotFound = errors.New("konfigurasi notifikasi tidak ditemukan")

	// ErrProviderNotConfigured dikembalikan saat provider belum dikonfigurasi untuk channel yang diminta
	ErrProviderNotConfigured = errors.New("provider belum dikonfigurasi untuk channel ini")

	// ErrCustomerNotFound dikembalikan saat pelanggan tidak ditemukan atau milik tenant lain
	ErrCustomerNotFound = errors.New("pelanggan tidak ditemukan")

	// ErrLogNotFound dikembalikan saat log notifikasi tidak ditemukan
	ErrLogNotFound = errors.New("log notifikasi tidak ditemukan")

	// ErrNotResendable dikembalikan saat mencoba mengirim ulang notifikasi yang bukan berstatus gagal
	ErrNotResendable = errors.New("hanya notifikasi gagal yang bisa dikirim ulang")

	// ErrInvalidCredentials dikembalikan saat credential provider tidak valid atau tidak lengkap
	ErrInvalidCredentials = errors.New("credential tidak valid atau tidak lengkap")

	// ErrDailyLimitExceeded dikembalikan saat batas harian notifikasi per pelanggan tercapai
	ErrDailyLimitExceeded = errors.New("batas harian notifikasi tercapai")

	// ErrInvalidTimezone dikembalikan saat timezone yang diberikan tidak valid
	ErrInvalidTimezone = errors.New("timezone tidak valid")

	// ErrInvalidQuietHours dikembalikan saat konfigurasi jam tenang tidak valid
	ErrInvalidQuietHours = errors.New("jam mulai harus sebelum jam selesai")
)
