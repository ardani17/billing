package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ispboss/ispboss/pkg/queue"
	"github.com/ispboss/ispboss/services/notification/internal/domain"
	"github.com/ispboss/ispboss/services/notification/internal/repository"
	"github.com/rs/zerolog"
)

// CustomerDataFetcher mengambil data pelanggan dari database (shared DB).
type CustomerDataFetcher interface {
	GetCustomerByID(ctx context.Context, customerID string) (*repository.CustomerData, error)
}

// TenantDataFetcher mengambil data tenant dari database (shared DB).
type TenantDataFetcher interface {
	GetTenantByID(ctx context.Context, tenantID string) (*repository.TenantData, error)
}

// DeliveryPipeline mengorkestrasikan seluruh alur pengiriman notifikasi.
type DeliveryPipeline struct {
	configRepo    domain.ConfigRepository
	templateRepo  domain.TemplateRepository
	logRepo       domain.LogRepository
	customerRepo  CustomerDataFetcher
	tenantRepo    TenantDataFetcher
	engine        *domain.TemplateEngine
	dedup         *DedupChecker
	quietHours    *QuietHoursChecker
	throttle      *ThrottleChecker
	waProvider    domain.WhatsAppProvider
	smsProvider   domain.SMSProvider
	emailProvider domain.EmailProvider
	logger        zerolog.Logger
}

// NewDeliveryPipeline membuat instance baru DeliveryPipeline.
func NewDeliveryPipeline(cr domain.ConfigRepository, tr domain.TemplateRepository, lr domain.LogRepository, cdr CustomerDataFetcher, tdr TenantDataFetcher, eng *domain.TemplateEngine, dd *DedupChecker, qh *QuietHoursChecker, th *ThrottleChecker, wa domain.WhatsAppProvider, sms domain.SMSProvider, em domain.EmailProvider, log zerolog.Logger) *DeliveryPipeline {
	return &DeliveryPipeline{configRepo: cr, templateRepo: tr, logRepo: lr, customerRepo: cdr, tenantRepo: tdr, engine: eng, dedup: dd, quietHours: qh, throttle: th, waProvider: wa, smsProvider: sms, emailProvider: em, logger: log}
}

// ProcessEvent memproses satu event dari queue melalui delivery pipeline.
func (p *DeliveryPipeline) ProcessEvent(ctx context.Context, env *queue.TaskEnvelope) error {
	var payload map[string]interface{}
	if err := json.Unmarshal(env.Payload, &payload); err != nil {
		p.logger.Warn().Err(err).Str("event_type", env.EventType).Msg("gagal decode payload")
		return nil
	}
	tid, cid := env.TenantID, strVal(payload, "customer_id")
	periode := strVal(payload, "periode")

	// Resolve template berdasarkan event_type
	tmpl, err := p.templateRepo.GetByEventType(ctx, tid, env.EventType)
	if err != nil || tmpl == nil || !tmpl.IsActive {
		p.logger.Warn().Str("event_type", env.EventType).Msg("template tidak ditemukan, skip")
		return nil
	}
	// Ambil data pelanggan dan tenant
	cust, err := p.customerRepo.GetCustomerByID(ctx, cid)
	if err != nil {
		return fmt.Errorf("gagal mengambil data pelanggan: %w", err)
	}
	tenant, err := p.tenantRepo.GetTenantByID(ctx, tid)
	if err != nil {
		return fmt.Errorf("gagal mengambil data tenant: %w", err)
	}
	// Cek deduplikasi
	dk := GenerateDedupKey(tid, cid, tmpl.Slug, periode)
	if dup, err := p.dedup.CheckDuplicate(ctx, dk); err != nil {
		return fmt.Errorf("gagal cek dedup: %w", err)
	} else if dup {
		p.mkLog(ctx, tid, cid, tmpl.ID, domain.ChannelWhatsApp, cust.Phone, "-", domain.StatusSkipped, dk, map[string]interface{}{"reason": "duplicate"})
		return nil
	}
	// Ambil settings tenant (bawaan jika belum ada)
	s, _ := p.configRepo.GetSettings(ctx, tid)
	if s == nil {
		s = &domain.ConfigSettings{ChannelPriority: []domain.Channel{domain.ChannelWhatsApp, domain.ChannelSMS, domain.ChannelEmail}, Timezone: "Asia/Jakarta", QuietHoursStart: "07:00", QuietHoursEnd: "21:00", DailyLimitPerCust: 5, CooldownMinutes: 30}
	}
	// Cek quiet hours
	if !p.quietHours.IsBypassEvent(env.EventType) && p.quietHours.IsQuietHours(time.Now(), s.Timezone, s.QuietHoursStart, s.QuietHoursEnd) {
		sa := p.quietHours.CalculateScheduledAt(s.Timezone, s.QuietHoursStart)
		p.mkLog(ctx, tid, cid, tmpl.ID, domain.ChannelWhatsApp, cust.Phone, "-", domain.StatusPending, dk, map[string]interface{}{"delayed_reason": "quiet_hours", "scheduled_at": sa})
		return nil
	}
	// Cek throttle (daily limit + cooldown)
	if !p.throttle.IsBypassEvent(env.EventType) {
		if exc, err := p.throttle.CheckDailyLimit(ctx, tid, cid, s.Timezone, s.DailyLimitPerCust); err != nil {
			return fmt.Errorf("gagal cek daily limit: %w", err)
		} else if exc {
			p.mkLog(ctx, tid, cid, tmpl.ID, domain.ChannelWhatsApp, cust.Phone, "-", domain.StatusSkipped, dk, map[string]interface{}{"reason": "daily_limit_exceeded"})
			return nil
		}
		if dly, until, err := p.throttle.CheckCooldown(ctx, tid, cid, s.CooldownMinutes); err != nil {
			return fmt.Errorf("gagal cek cooldown: %w", err)
		} else if dly {
			p.mkLog(ctx, tid, cid, tmpl.ID, domain.ChannelWhatsApp, cust.Phone, "-", domain.StatusPending, dk, map[string]interface{}{"delayed_reason": "cooldown", "scheduled_at": until})
			return nil
		}
	}
	// Bangun data map untuk rendering template
	data := map[string]string{"nama": cust.Name, "id_pelanggan": cust.CustomerIDSeq, "nama_isp": tenant.Name}
	for k, v := range payload {
		if sv, ok := v.(string); ok {
			data[k] = sv
		}
	}
	// Pilih channel dan kirim dengan retry + cadangan
	cfgs, _ := p.configRepo.GetByTenant(ctx, tid)
	return p.sendRetry(ctx, tmpl, p.pickChannels(s.ChannelPriority, tmpl.Channels, cfgs), cfgs, data, cust, dk, tid, cid)
}

// pickChannels memilih channel yang ada di prioritas, template, dan config aktif.
func (p *DeliveryPipeline) pickChannels(prio, tmplCh []domain.Channel, cfgs []*domain.NotificationConfig) []domain.Channel {
	ts, cs := make(map[domain.Channel]bool, len(tmplCh)), make(map[domain.Channel]bool, len(cfgs))
	for _, c := range tmplCh {
		ts[c] = true
	}
	for _, c := range cfgs {
		if c.IsEnabled {
			cs[c.Channel] = true
		}
	}
	var out []domain.Channel
	for _, ch := range prio {
		if ts[ch] && cs[ch] {
			out = append(out, ch)
		}
	}
	return out
}

// sendRetry mengirim notifikasi: 3 percobaan per channel, lalu cadangan ke channel berikutnya.
func (p *DeliveryPipeline) sendRetry(ctx context.Context, tmpl *domain.NotificationTemplate, chs []domain.Channel, cfgs []*domain.NotificationConfig, data map[string]string, cust *repository.CustomerData, dk, tid, cid string) error {
	var lastErr string
	for _, ch := range chs {
		body, rcpt, prov := p.renderBody(tmpl, ch, data), p.rcpt(ch, cust), p.prov(ch, cfgs)
		for att := 0; att < 3; att++ {
			res, err := p.send(ctx, ch, rcpt, body, tmpl.BodyEmailSubject)
			if err == nil && res.Status == "sent" {
				now := time.Now()
				_, _ = p.logRepo.Create(ctx, &domain.NotificationLog{TenantID: tid, CustomerID: cid, TemplateID: tmpl.ID, Channel: ch, Provider: prov, Recipient: rcpt, Body: body, Status: domain.StatusSent, RetryCount: att, MaxRetries: 3, DedupKey: dk, SentAt: &now})
				return nil
			}
			if err != nil {
				lastErr = err.Error()
			} else {
				lastErr = res.ErrorDetail
			}
		}
	}
	p.mkLog(ctx, tid, cid, tmpl.ID, domain.ChannelWhatsApp, cust.Phone, "-", domain.StatusFailed, dk, map[string]interface{}{"error": lastErr})
	return nil
}

// renderBody merender template body sesuai channel.
func (p *DeliveryPipeline) renderBody(t *domain.NotificationTemplate, ch domain.Channel, d map[string]string) string {
	switch ch {
	case domain.ChannelWhatsApp:
		return p.engine.Render(t.BodyWhatsApp, d)
	case domain.ChannelSMS:
		return p.engine.Render(t.BodySMS, d)
	case domain.ChannelEmail:
		return p.engine.Render(t.BodyEmailHTML, d)
	}
	return ""
}

// rcpt mengembalikan alamat penerima sesuai channel.
func (p *DeliveryPipeline) rcpt(ch domain.Channel, c *repository.CustomerData) string {
	if ch == domain.ChannelEmail {
		return c.Email
	}
	return c.Phone
}

// prov mengembalikan nama provider untuk channel dari config.
func (p *DeliveryPipeline) prov(ch domain.Channel, cfgs []*domain.NotificationConfig) string {
	for _, c := range cfgs {
		if c.Channel == ch {
			return c.Provider
		}
	}
	return "-"
}

// send mengirim pesan melalui provider adapter sesuai channel.
func (p *DeliveryPipeline) send(ctx context.Context, ch domain.Channel, rcpt, body, subj string) (domain.SendResult, error) {
	switch ch {
	case domain.ChannelWhatsApp:
		return p.waProvider.Send(ctx, domain.WhatsAppMessage{Recipient: rcpt, Body: body})
	case domain.ChannelSMS:
		return p.smsProvider.Send(ctx, domain.SMSMessage{Recipient: rcpt, Body: body})
	case domain.ChannelEmail:
		return p.emailProvider.Send(ctx, domain.EmailMessage{Recipient: rcpt, Subject: subj, HTMLBody: body})
	}
	return domain.SendResult{Status: "failed", ErrorDetail: "channel tidak dikenal"}, nil
}

// mkLog membuat catatan log notifikasi di database.
func (p *DeliveryPipeline) mkLog(ctx context.Context, tid, cid, tplID string, ch domain.Channel, rcpt, body string, st domain.LogStatus, dk string, meta map[string]interface{}) {
	_, _ = p.logRepo.Create(ctx, &domain.NotificationLog{TenantID: tid, CustomerID: cid, TemplateID: tplID, Channel: ch, Provider: "-", Recipient: rcpt, Body: body, Status: st, MaxRetries: 3, DedupKey: dk, Metadata: meta})
}

// strVal mengambil string value dari map payload.
func strVal(m map[string]interface{}, key string) string {
	v, _ := m[key].(string)
	return v
}
