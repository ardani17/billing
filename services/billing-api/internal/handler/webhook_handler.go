// webhook_handler.go menangani endpoint webhook publik dari gateway pembayaran.
// Endpoint ini TIDAK menggunakan auth middleware.
// Keamanan via IP whitelist + verifikasi signature (async di webhook usecase).
package handler

import (
	"encoding/json"

	"github.com/gofiber/fiber/v2"
	"github.com/hibiken/asynq"
	"github.com/rs/zerolog"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

// TaskProcessWebhook adalah tipe task asynq untuk memproses webhook secara async.
const TaskProcessWebhook = "gateway.process_webhook"

// WebhookHandler menangani HTTP permintaan webhook dari Xendit dan Midtrans.
// Endpoint bersifat publik - keamanan via IP whitelist dan signature verification.
type WebhookHandler struct {
	webhookLogRepo domain.WebhookLogRepository
	queueClient    *asynq.Client
	xenditIPs      []string
	midtransIPs    []string
	logger         zerolog.Logger
}

// NewWebhookHandler membuat instance baru WebhookHandler.
func NewWebhookHandler(
	webhookLogRepo domain.WebhookLogRepository,
	queueClient *asynq.Client,
	xenditIPs []string,
	midtransIPs []string,
	logger zerolog.Logger,
) *WebhookHandler {
	return &WebhookHandler{
		webhookLogRepo: webhookLogRepo,
		queueClient:    queueClient,
		xenditIPs:      xenditIPs,
		midtransIPs:    midtransIPs,
		logger:         logger,
	}
}

// HandleXendit menangani POST /webhooks/xendit.
// Menerima notifikasi pembayaran dari Xendit, log ke webhook_logs,
// lalu antrekan task untuk pemrosesan async. Kembalikan 200 segera.
func (h *WebhookHandler) HandleXendit(c *fiber.Ctx) error {
	sourceIP := c.IP()

	// Cek IP whitelist (skip jika whitelist kosong - untuk dev/testing)
	if !h.checkIPWhitelist(sourceIP, h.xenditIPs) {
		h.logBlockedIP(c, domain.GatewayXendit, sourceIP)
		return domain.ErrorResponse(c, fiber.StatusForbidden, "IP_NOT_WHITELISTED", "ip_not_whitelisted")
	}

	// Parsing body sebagai JSON
	var payload map[string]interface{}
	if err := json.Unmarshal(c.Body(), &payload); err != nil {
		h.logger.Warn().Err(err).Str("source_ip", sourceIP).Msg("gagal parse body webhook xendit")
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "request body tidak valid")
	}

	// Ekstrak external_id dan event_type dari payload Xendit
	externalID, _ := payload["external_id"].(string)
	status, _ := payload["status"].(string)
	eventType := mapXenditStatus(status)

	// Simpan header x-callback-token di request_body untuk verifikasi signature nanti
	callbackToken := c.Get("x-callback-token")
	payload["_headers"] = map[string]string{
		"x-callback-token": callbackToken,
	}
	bodyWithHeaders, _ := json.Marshal(payload)

	// INSERT webhook_log dengan status=received
	log := &domain.WebhookLog{
		GatewayProvider:  domain.GatewayXendit,
		EventType:        eventType,
		ExternalID:       externalID,
		RequestBody:      bodyWithHeaders,
		SourceIP:         sourceIP,
		ProcessingStatus: domain.WebhookReceived,
	}

	created, err := h.webhookLogRepo.Create(c.Context(), log)
	if err != nil {
		h.logger.Error().Err(err).Msg("gagal menyimpan webhook log xendit")
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal menyimpan webhook log")
	}

	// Enqueue task untuk pemrosesan async
	h.enqueueProcessWebhook(created.ID)

	return domain.SuccessResponse(c, fiber.StatusOK, fiber.Map{"status": "received"})
}

// HandleMidtrans menangani POST /webhooks/midtrans.
// Menerima notifikasi pembayaran dari Midtrans, log ke webhook_logs,
// lalu antrekan task untuk pemrosesan async. Kembalikan 200 segera.
func (h *WebhookHandler) HandleMidtrans(c *fiber.Ctx) error {
	sourceIP := c.IP()

	// Cek IP whitelist (skip jika whitelist kosong - untuk dev/testing)
	if !h.checkIPWhitelist(sourceIP, h.midtransIPs) {
		h.logBlockedIP(c, domain.GatewayMidtrans, sourceIP)
		return domain.ErrorResponse(c, fiber.StatusForbidden, "IP_NOT_WHITELISTED", "ip_not_whitelisted")
	}

	// Parsing body sebagai JSON
	var payload map[string]interface{}
	if err := json.Unmarshal(c.Body(), &payload); err != nil {
		h.logger.Warn().Err(err).Str("source_ip", sourceIP).Msg("gagal parse body webhook midtrans")
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "request body tidak valid")
	}

	// Ekstrak order_id sebagai external_id dan map transaction_status ke event_type
	externalID, _ := payload["order_id"].(string)
	txStatus, _ := payload["transaction_status"].(string)
	eventType := mapMidtransStatus(txStatus)

	// INSERT webhook_log dengan status=received
	log := &domain.WebhookLog{
		GatewayProvider:  domain.GatewayMidtrans,
		EventType:        eventType,
		ExternalID:       externalID,
		RequestBody:      c.Body(),
		SourceIP:         sourceIP,
		ProcessingStatus: domain.WebhookReceived,
	}

	created, err := h.webhookLogRepo.Create(c.Context(), log)
	if err != nil {
		h.logger.Error().Err(err).Msg("gagal menyimpan webhook log midtrans")
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal menyimpan webhook log")
	}

	// Enqueue task untuk pemrosesan async
	h.enqueueProcessWebhook(created.ID)

	return domain.SuccessResponse(c, fiber.StatusOK, fiber.Map{"status": "received"})
}

// checkIPWhitelist mengecek apakah sourceIP ada dalam whitelist.
// Mengembalikan true jika whitelist kosong (skip validasi) atau IP ditemukan.
func (h *WebhookHandler) checkIPWhitelist(sourceIP string, whitelist []string) bool {
	if len(whitelist) == 0 {
		return true
	}
	for _, ip := range whitelist {
		if ip == sourceIP {
			return true
		}
	}
	return false
}

// logBlockedIP mencatat webhook log dengan status failed untuk IP yang diblokir.
func (h *WebhookHandler) logBlockedIP(c *fiber.Ctx, provider domain.GatewayProvider, sourceIP string) {
	h.logger.Warn().
		Str("source_ip", sourceIP).
		Str("provider", string(provider)).
		Msg("webhook ditolak: IP tidak ada dalam whitelist")

	blockedLog := &domain.WebhookLog{
		GatewayProvider:  provider,
		EventType:        "unknown",
		ExternalID:       "",
		RequestBody:      c.Body(),
		SourceIP:         sourceIP,
		ProcessingStatus: domain.WebhookFailed,
		ErrorMessage:     "ip_not_whitelisted",
	}
	if _, err := h.webhookLogRepo.Create(c.Context(), blockedLog); err != nil {
		h.logger.Error().Err(err).Msg("gagal menyimpan log webhook yang diblokir")
	}
}

// enqueueProcessWebhook mengirim task pemrosesan webhook ke asynq queue.
func (h *WebhookHandler) enqueueProcessWebhook(webhookLogID string) {
	payload, _ := json.Marshal(map[string]string{"webhook_log_id": webhookLogID})
	task := asynq.NewTask(TaskProcessWebhook, payload)
	if _, err := h.queueClient.Enqueue(task); err != nil {
		h.logger.Error().Err(err).Str("webhook_log_id", webhookLogID).Msg("gagal enqueue task process webhook")
	}
}

// mapXenditStatus memetakan status Xendit ke event type internal.
func mapXenditStatus(status string) string {
	mapping := map[string]string{
		"PAID":    "payment.paid",
		"EXPIRED": "payment.expired",
		"FAILED":  "payment.failed",
	}
	if evt, ok := mapping[status]; ok {
		return evt
	}
	return "unknown"
}

// mapMidtransStatus memetakan transaction_status Midtrans ke event type internal.
func mapMidtransStatus(status string) string {
	mapping := map[string]string{
		"capture":    "payment.paid",
		"settlement": "payment.paid",
		"expire":     "payment.expired",
		"deny":       "payment.failed",
		"cancel":     "payment.failed",
	}
	if evt, ok := mapping[status]; ok {
		return evt
	}
	return "unknown"
}
