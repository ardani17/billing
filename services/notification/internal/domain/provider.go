package domain

import "context"

// =============================================================================
// WhatsAppProvider — interface adapter untuk pengiriman pesan WhatsApp
// =============================================================================

// WhatsAppProvider mendefinisikan interface untuk provider WhatsApp.
// Diimplementasikan oleh provider.FonnteAdapter.
type WhatsAppProvider interface {
	// Send mengirim pesan WhatsApp ke penerima dan mengembalikan hasil pengiriman.
	Send(ctx context.Context, req WhatsAppMessage) (SendResult, error)
}

// =============================================================================
// SMSProvider — interface adapter untuk pengiriman pesan SMS
// =============================================================================

// SMSProvider mendefinisikan interface untuk provider SMS.
// Diimplementasikan oleh provider.ZenzivaAdapter.
type SMSProvider interface {
	// Send mengirim pesan SMS ke penerima dan mengembalikan hasil pengiriman.
	Send(ctx context.Context, req SMSMessage) (SendResult, error)
}

// =============================================================================
// EmailProvider — interface adapter untuk pengiriman pesan Email
// =============================================================================

// EmailProvider mendefinisikan interface untuk provider Email.
// Diimplementasikan oleh provider.SMTPAdapter.
type EmailProvider interface {
	// Send mengirim pesan Email ke penerima dan mengembalikan hasil pengiriman.
	Send(ctx context.Context, req EmailMessage) (SendResult, error)
}

// =============================================================================
// Pesan — struct data pesan per channel
// =============================================================================

// WhatsAppMessage berisi data pesan WhatsApp yang akan dikirim via provider.
type WhatsAppMessage struct {
	Recipient string `json:"recipient"`
	Body      string `json:"body"`
	MediaURL  string `json:"media_url,omitempty"`
}

// SMSMessage berisi data pesan SMS yang akan dikirim via provider.
type SMSMessage struct {
	Recipient string `json:"recipient"`
	Body      string `json:"body"`
}

// EmailMessage berisi data pesan Email yang akan dikirim via provider.
type EmailMessage struct {
	Recipient string `json:"recipient"`
	Subject   string `json:"subject"`
	HTMLBody  string `json:"html_body"`
}

// =============================================================================
// SendResult — hasil pengiriman dari provider
// =============================================================================

// SendResult berisi hasil pengiriman dari provider adapter.
// Status bernilai "sent" jika berhasil atau "failed" jika gagal.
// ErrorDetail berisi detail error dari provider (kosong jika berhasil).
type SendResult struct {
	MessageID   string `json:"message_id"`
	Status      string `json:"status"`
	ErrorDetail string `json:"error_detail,omitempty"`
}
