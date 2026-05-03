package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/ispboss/ispboss/services/notification/internal/domain"
	"github.com/ispboss/ispboss/services/notification/internal/repository"
)

// SendManual mengirim notifikasi manual ke pelanggan tertentu.
// Mendukung template (substitusi variabel) atau custom body.
// Menghormati throttle (daily limit dan cooldown). Metadata: trigger="manual".
func (p *DeliveryPipeline) SendManual(ctx context.Context, req domain.ManualSendRequest) (*domain.NotificationLog, error) {
	cust, err := p.customerRepo.GetCustomerByID(ctx, req.CustomerID)
	if err != nil {
		return nil, domain.ErrCustomerNotFound
	}
	tid := cust.ID
	s := p.getSettingsOrDefault(ctx, tid)

	// Cek throttle: daily limit dan cooldown
	if exc, err := p.throttle.CheckDailyLimit(ctx, tid, req.CustomerID, s.Timezone, s.DailyLimitPerCust); err != nil {
		return nil, fmt.Errorf("gagal cek daily limit: %w", err)
	} else if exc {
		return nil, domain.ErrDailyLimitExceeded
	}
	if dly, _, err := p.throttle.CheckCooldown(ctx, tid, req.CustomerID, s.CooldownMinutes); err != nil {
		return nil, fmt.Errorf("gagal cek cooldown: %w", err)
	} else if dly {
		return nil, domain.ErrDailyLimitExceeded
	}

	// Tentukan body: dari template atau custom body
	var body, subject, templateID string
	if req.TemplateID != "" {
		tmpl, err := p.templateRepo.GetByID(ctx, req.TemplateID)
		if err != nil {
			return nil, domain.ErrTemplateNotFound
		}
		data := map[string]string{"nama": cust.Name, "id_pelanggan": cust.CustomerIDSeq}
		body = p.renderBody(tmpl, req.Channel, data)
		subject = p.engine.Render(tmpl.BodyEmailSubject, data)
		templateID = tmpl.ID
	} else if req.CustomBody != "" {
		body = req.CustomBody
		subject = req.CustomSubject
	} else {
		return nil, fmt.Errorf("template_id atau custom_body harus diisi")
	}

	rcpt := p.rcptForChannel(req.Channel, cust)
	provName := p.provForChannel(ctx, tid, req.Channel)
	res, sendErr := p.send(ctx, req.Channel, rcpt, body, subject)
	status, errMsg, sentAt := resolveResult(res, sendErr)

	return p.logRepo.Create(ctx, &domain.NotificationLog{
		TenantID: tid, CustomerID: req.CustomerID, TemplateID: templateID,
		Channel: req.Channel, Provider: provName, Recipient: rcpt,
		Subject: subject, Body: body, Status: status, MaxRetries: 3,
		ErrorMessage: errMsg, Metadata: map[string]interface{}{"trigger": "manual"}, SentAt: sentAt,
	})
}

// SendTest mengirim notifikasi percobaan menggunakan template dan sample data.
// Bypass deduplikasi, quiet hours, dan throttle. Metadata: is_test=true.
func (p *DeliveryPipeline) SendTest(ctx context.Context, req domain.TestSendRequest) (*domain.NotificationLog, error) {
	tmpl, err := p.templateRepo.GetByID(ctx, req.TemplateID)
	if err != nil || tmpl == nil {
		return nil, domain.ErrTemplateNotFound
	}
	// Cek provider sudah dikonfigurasi dan aktif untuk channel
	cfg, err := p.configRepo.GetByTenantAndChannel(ctx, tmpl.TenantID, req.Channel)
	if err != nil || cfg == nil || !cfg.IsEnabled {
		return nil, domain.ErrProviderNotConfigured
	}

	// Render template dengan sample data
	sd := map[string]string{
		"nama": "Test User", "id_pelanggan": "CUST-0001",
		"nama_isp": "ISP Demo", "telepon_isp": "08123456789",
		"paket": "Paket 50 Mbps", "harga": "Rp 350.000",
		"periode": "April 2026", "no_invoice": "INV-2026-04-0001",
		"total_tagihan": "Rp 100.000", "jatuh_tempo": "20 April 2026",
		"sisa_hari": "3", "terlambat_hari": "0",
		"tanggal_bayar": "17 April 2026", "metode_bayar": "Transfer Bank",
		"jumlah_bayar": "Rp 100.000",
	}
	body := p.renderBody(tmpl, req.Channel, sd)
	subject := p.engine.Render(tmpl.BodyEmailSubject, sd)

	res, sendErr := p.send(ctx, req.Channel, req.Recipient, body, subject)
	status, errMsg, sentAt := resolveResult(res, sendErr)

	return p.logRepo.Create(ctx, &domain.NotificationLog{
		TenantID: tmpl.TenantID, CustomerID: "", TemplateID: tmpl.ID,
		Channel: req.Channel, Provider: cfg.Provider, Recipient: req.Recipient,
		Subject: subject, Body: body, Status: status, MaxRetries: 3,
		ErrorMessage: errMsg, Metadata: map[string]interface{}{"is_test": true}, SentAt: sentAt,
	})
}

// Resend mengirim ulang notifikasi yang gagal berdasarkan log ID.
// Hanya status "failed" yang bisa dikirim ulang. Bypass deduplikasi,
// tetap menghormati quiet hours dan throttle. Membuat log BARU: resend_of=logID.
func (p *DeliveryPipeline) Resend(ctx context.Context, logID string) (*domain.NotificationLog, error) {
	orig, err := p.logRepo.GetByID(ctx, logID)
	if err != nil || orig == nil {
		return nil, domain.ErrLogNotFound
	}
	if orig.Status != domain.StatusFailed {
		return nil, domain.ErrNotResendable
	}

	s := p.getSettingsOrDefault(ctx, orig.TenantID)

	// Cek quiet hours — jika di luar jam aktif, buat log pending
	if p.quietHours.IsQuietHours(time.Now(), s.Timezone, s.QuietHoursStart, s.QuietHoursEnd) {
		sa := p.quietHours.CalculateScheduledAt(s.Timezone, s.QuietHoursStart)
		return p.logRepo.Create(ctx, &domain.NotificationLog{
			TenantID: orig.TenantID, CustomerID: orig.CustomerID, TemplateID: orig.TemplateID,
			Channel: orig.Channel, Provider: orig.Provider, Recipient: orig.Recipient,
			Subject: orig.Subject, Body: orig.Body, Status: domain.StatusPending, MaxRetries: 3,
			Metadata: map[string]interface{}{"resend_of": logID, "delayed_reason": "quiet_hours", "scheduled_at": sa},
		})
	}

	// Cek throttle: daily limit dan cooldown
	if exc, err := p.throttle.CheckDailyLimit(ctx, orig.TenantID, orig.CustomerID, s.Timezone, s.DailyLimitPerCust); err != nil {
		return nil, fmt.Errorf("gagal cek daily limit: %w", err)
	} else if exc {
		return nil, domain.ErrDailyLimitExceeded
	}
	if dly, _, err := p.throttle.CheckCooldown(ctx, orig.TenantID, orig.CustomerID, s.CooldownMinutes); err != nil {
		return nil, fmt.Errorf("gagal cek cooldown: %w", err)
	} else if dly {
		return nil, domain.ErrDailyLimitExceeded
	}

	// Kirim menggunakan channel, recipient, dan body dari log asli
	res, sendErr := p.send(ctx, orig.Channel, orig.Recipient, orig.Body, orig.Subject)
	status, errMsg, sentAt := resolveResult(res, sendErr)

	return p.logRepo.Create(ctx, &domain.NotificationLog{
		TenantID: orig.TenantID, CustomerID: orig.CustomerID, TemplateID: orig.TemplateID,
		Channel: orig.Channel, Provider: orig.Provider, Recipient: orig.Recipient,
		Subject: orig.Subject, Body: orig.Body, Status: status, MaxRetries: 3,
		ErrorMessage: errMsg, Metadata: map[string]interface{}{"resend_of": logID}, SentAt: sentAt,
	})
}

// getSettingsOrDefault mengambil settings tenant, atau mengembalikan default.
func (p *DeliveryPipeline) getSettingsOrDefault(ctx context.Context, tenantID string) *domain.ConfigSettings {
	if s, err := p.configRepo.GetSettings(ctx, tenantID); err == nil && s != nil {
		return s
	}
	return &domain.ConfigSettings{
		ChannelPriority: []domain.Channel{domain.ChannelWhatsApp, domain.ChannelSMS, domain.ChannelEmail},
		Timezone: "Asia/Jakarta", QuietHoursStart: "07:00", QuietHoursEnd: "21:00",
		DailyLimitPerCust: 5, CooldownMinutes: 30,
	}
}

// rcptForChannel mengembalikan alamat penerima sesuai channel dari data pelanggan.
func (p *DeliveryPipeline) rcptForChannel(ch domain.Channel, c *repository.CustomerData) string {
	if ch == domain.ChannelEmail {
		return c.Email
	}
	return c.Phone
}

// provForChannel mengambil nama provider untuk channel dari config tenant.
func (p *DeliveryPipeline) provForChannel(ctx context.Context, tenantID string, ch domain.Channel) string {
	cfg, err := p.configRepo.GetByTenantAndChannel(ctx, tenantID, ch)
	if err != nil || cfg == nil {
		return "-"
	}
	return cfg.Provider
}

// resolveResult menentukan status, error message, dan sent_at dari hasil pengiriman.
func resolveResult(res domain.SendResult, sendErr error) (domain.LogStatus, string, *time.Time) {
	if sendErr != nil || res.Status != "sent" {
		msg := res.ErrorDetail
		if sendErr != nil {
			msg = sendErr.Error()
		}
		return domain.StatusFailed, msg, nil
	}
	now := time.Now()
	return domain.StatusSent, "", &now
}
