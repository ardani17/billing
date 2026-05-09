package provider

import (
	"context"
	"fmt"
	"net/smtp"
	"strings"
	"time"

	"github.com/ispboss/ispboss/services/notification/internal/domain"
)

// =============================================================================
// SMTPAdapter - adapter untuk pengiriman pesan Email via SMTP
// =============================================================================

// SMTPAdapter mengimplementasikan domain.EmailProvider menggunakan net/smtp.
// Mengirim email melalui server SMTP yang dikonfigurasi dengan autentikasi PlainAuth.
type SMTPAdapter struct {
	host      string
	port      int
	username  string
	password  string
	fromName  string
	fromEmail string
}

// NewSMTPAdapter membuat instance baru SMTPAdapter dengan konfigurasi SMTP.
func NewSMTPAdapter(host string, port int, username, password, fromName, fromEmail string) *SMTPAdapter {
	return &SMTPAdapter{
		host:      host,
		port:      port,
		username:  username,
		password:  password,
		fromName:  fromName,
		fromEmail: fromEmail,
	}
}

// Send mengirim pesan email ke penerima melalui SMTP server.
// Mengembalikan SendResult dengan status "sent" jika berhasil atau "failed" jika gagal.
func (a *SMTPAdapter) Send(ctx context.Context, req domain.EmailMessage) (domain.SendResult, error) {
	// Buat message ID unik berbasis timestamp
	messageID := generateMessageID(a.host)

	// Bangun pesan MIME dengan header dan body HTML
	mime := buildMIMEMessage(a.fromName, a.fromEmail, req.Recipient, req.Subject, req.HTMLBody, messageID)

	// Siapkan alamat server SMTP (host:port)
	addr := fmt.Sprintf("%s:%d", a.host, a.port)

	// Buat autentikasi PlainAuth
	auth := smtp.PlainAuth("", a.username, a.password, a.host)

	// Kirim email via net/smtp.SendMail
	err := smtp.SendMail(addr, auth, a.fromEmail, []string{req.Recipient}, []byte(mime))
	if err != nil {
		detail := classifyError(err)
		return domain.SendResult{
			MessageID:   messageID,
			Status:      "failed",
			ErrorDetail: detail,
		}, fmt.Errorf("gagal mengirim email via SMTP: %w", err)
	}

	return domain.SendResult{
		MessageID: messageID,
		Status:    "sent",
	}, nil
}

// buildMIMEMessage membangun pesan MIME lengkap dengan header dan body HTML.
// Format mengikuti standar RFC 2045 untuk Content-Type text/html.
func buildMIMEMessage(fromName, fromEmail, to, subject, htmlBody, messageID string) string {
	var b strings.Builder

	// Header From dengan display name
	b.WriteString(fmt.Sprintf("From: %s <%s>\r\n", fromName, fromEmail))
	b.WriteString(fmt.Sprintf("To: %s\r\n", to))
	b.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	b.WriteString(fmt.Sprintf("Message-ID: <%s>\r\n", messageID))
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/html; charset=\"UTF-8\"\r\n")
	b.WriteString("\r\n")
	b.WriteString(htmlBody)

	return b.String()
}

// generateMessageID menghasilkan message ID unik berbasis timestamp dan hostname.
// Format: <timestamp.nanodetik@hostname>
func generateMessageID(host string) string {
	now := time.Now()
	return fmt.Sprintf("%d.%d@%s", now.UnixMilli(), now.UnixNano()%1000000, host)
}

// classifyError mengklasifikasikan error SMTP ke pesan yang lebih deskriptif.
// Membantu membedakan antara error koneksi, autentikasi, dan pengiriman.
func classifyError(err error) string {
	msg := err.Error()

	// Deteksi error autentikasi
	if strings.Contains(msg, "authentication") ||
		strings.Contains(msg, "auth") ||
		strings.Contains(msg, "535") ||
		strings.Contains(msg, "534") {
		return fmt.Sprintf("gagal autentikasi SMTP: %v", err)
	}

	// Deteksi error koneksi
	if strings.Contains(msg, "connection") ||
		strings.Contains(msg, "dial") ||
		strings.Contains(msg, "timeout") ||
		strings.Contains(msg, "refused") ||
		strings.Contains(msg, "no such host") {
		return fmt.Sprintf("gagal koneksi ke SMTP server: %v", err)
	}

	// Error umum lainnya
	return fmt.Sprintf("gagal mengirim email: %v", err)
}
