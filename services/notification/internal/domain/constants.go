package domain

// =============================================================================
// Channel — tipe channel pengiriman notifikasi
// =============================================================================

// Channel mendefinisikan media pengiriman notifikasi.
type Channel string

const (
	// ChannelWhatsApp untuk pengiriman via WhatsApp.
	ChannelWhatsApp Channel = "whatsapp"

	// ChannelSMS untuk pengiriman via SMS.
	ChannelSMS Channel = "sms"

	// ChannelEmail untuk pengiriman via Email.
	ChannelEmail Channel = "email"
)

// ValidChannels berisi daftar channel yang valid.
var ValidChannels = []Channel{
	ChannelWhatsApp,
	ChannelSMS,
	ChannelEmail,
}

// IsValidChannel memeriksa apakah channel valid.
func IsValidChannel(c Channel) bool {
	for _, v := range ValidChannels {
		if v == c {
			return true
		}
	}
	return false
}

// =============================================================================
// LogStatus — status pengiriman notifikasi
// =============================================================================

// LogStatus mendefinisikan status catatan pengiriman notifikasi.
type LogStatus string

const (
	// StatusPending menandakan notifikasi menunggu dikirim.
	StatusPending LogStatus = "pending"

	// StatusSending menandakan notifikasi sedang dalam proses pengiriman.
	StatusSending LogStatus = "sending"

	// StatusSent menandakan notifikasi berhasil dikirim ke provider.
	StatusSent LogStatus = "sent"

	// StatusDelivered menandakan notifikasi sudah diterima oleh penerima.
	StatusDelivered LogStatus = "delivered"

	// StatusRead menandakan notifikasi sudah dibaca oleh penerima.
	StatusRead LogStatus = "read"

	// StatusFailed menandakan notifikasi gagal dikirim setelah semua percobaan.
	StatusFailed LogStatus = "failed"

	// StatusRetrying menandakan notifikasi sedang dalam proses retry.
	StatusRetrying LogStatus = "retrying"

	// StatusSkipped menandakan notifikasi dilewati (duplikat, throttle, dll).
	StatusSkipped LogStatus = "skipped"
)

// ValidLogStatuses berisi daftar status log yang valid.
var ValidLogStatuses = []LogStatus{
	StatusPending,
	StatusSending,
	StatusSent,
	StatusDelivered,
	StatusRead,
	StatusFailed,
	StatusRetrying,
	StatusSkipped,
}

// IsValidLogStatus memeriksa apakah status log valid.
func IsValidLogStatus(s LogStatus) bool {
	for _, v := range ValidLogStatuses {
		if v == s {
			return true
		}
	}
	return false
}

// =============================================================================
// TemplateCategory — kategori template notifikasi
// =============================================================================

// TemplateCategory mendefinisikan kategori template notifikasi.
type TemplateCategory string

const (
	// CategoryTransactional untuk notifikasi transaksional (invoice, pembayaran).
	CategoryTransactional TemplateCategory = "transactional"

	// CategoryReminder untuk notifikasi pengingat (jatuh tempo, tagihan).
	CategoryReminder TemplateCategory = "reminder"

	// CategoryPromotion untuk notifikasi promosi (diskon, penawaran).
	CategoryPromotion TemplateCategory = "promotion"

	// CategoryInformation untuk notifikasi informasi umum (perubahan status, dll).
	CategoryInformation TemplateCategory = "information"
)

// ValidTemplateCategories berisi daftar kategori template yang valid.
var ValidTemplateCategories = []TemplateCategory{
	CategoryTransactional,
	CategoryReminder,
	CategoryPromotion,
	CategoryInformation,
}

// IsValidTemplateCategory memeriksa apakah kategori template valid.
func IsValidTemplateCategory(c TemplateCategory) bool {
	for _, v := range ValidTemplateCategories {
		if v == c {
			return true
		}
	}
	return false
}

// =============================================================================
// Bypass Event Types — event yang dikecualikan dari quiet hours dan throttle
// =============================================================================

// BypassEventTypes berisi daftar event_type yang dikecualikan dari
// pembatasan quiet hours dan throttle. Event ini dikirim langsung
// tanpa menunggu jam aktif atau memeriksa batas harian.
var BypassEventTypes = []string{
	"payment.online.received",
	"payment.recorded",
	"notification.un_isolir",
	"notification.reactivated",
}

// IsBypassEvent memeriksa apakah event_type termasuk dalam daftar bypass.
func IsBypassEvent(eventType string) bool {
	for _, v := range BypassEventTypes {
		if v == eventType {
			return true
		}
	}
	return false
}
