package handler

import (
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/ispboss/ispboss/services/notification/internal/domain"
	"github.com/ispboss/ispboss/services/notification/internal/usecase"
)

// SendHandler menangani HTTP permintaan untuk pengiriman notifikasi.
// Menyediakan endpoint untuk test send, manual send, dan resend.
type SendHandler struct {
	pipeline *usecase.DeliveryPipeline
}

// NewSendHandler membuat instance SendHandler baru dengan dependensi DeliveryPipeline.
func NewSendHandler(pipeline *usecase.DeliveryPipeline) *SendHandler {
	return &SendHandler{pipeline: pipeline}
}

// TestSend menangani POST /api/v1/notifications/test.
// Bypass deduplikasi, quiet hours, dan throttle.
func (h *SendHandler) TestSend(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	if tenantID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant_id tidak ditemukan")
	}

	var req domain.TestSendRequest
	if err := c.BodyParser(&req); err != nil {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "format request tidak valid")
	}
	if req.TemplateID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnprocessableEntity, "VALIDATION_ERROR", "template_id wajib diisi")
	}
	if req.Channel == "" {
		return domain.ErrorResponse(c, fiber.StatusUnprocessableEntity, "VALIDATION_ERROR", "channel wajib diisi")
	}
	if req.Recipient == "" {
		return domain.ErrorResponse(c, fiber.StatusUnprocessableEntity, "VALIDATION_ERROR", "recipient wajib diisi")
	}

	log, err := h.pipeline.SendTest(c.UserContext(), req)
	if err != nil {
		return h.mapTestSendError(c, err)
	}
	return domain.SuccessResponse(c, fiber.StatusOK, log)
}

// ManualSend menangani POST /api/v1/notifications/send.
// Mengirim notifikasi manual ke pelanggan menggunakan template atau kustom body.
func (h *SendHandler) ManualSend(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	if tenantID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant_id tidak ditemukan")
	}

	var req domain.ManualSendRequest
	if err := c.BodyParser(&req); err != nil {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "format request tidak valid")
	}
	if req.CustomerID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnprocessableEntity, "VALIDATION_ERROR", "customer_id wajib diisi")
	}
	if req.Channel == "" {
		return domain.ErrorResponse(c, fiber.StatusUnprocessableEntity, "VALIDATION_ERROR", "channel wajib diisi")
	}
	if req.TemplateID == "" && req.CustomBody == "" {
		return domain.ErrorResponse(c, fiber.StatusUnprocessableEntity, "VALIDATION_ERROR", "template_id atau custom_body harus diisi")
	}

	log, err := h.pipeline.SendManual(c.UserContext(), req)
	if err != nil {
		return h.mapManualSendError(c, err)
	}
	return domain.SuccessResponse(c, fiber.StatusOK, log)
}

// Resend menangani POST /api/v1/notifications/logs/:id/resend.
// Hanya notifikasi dengan status "failed" yang bisa dikirim ulang.
func (h *SendHandler) Resend(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	if tenantID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant_id tidak ditemukan")
	}

	logID := c.Params("id")
	if logID == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "id log tidak boleh kosong")
	}

	log, err := h.pipeline.Resend(c.UserContext(), logID)
	if err != nil {
		return h.mapResendError(c, err)
	}
	return domain.SuccessResponse(c, fiber.StatusOK, log)
}

// mapTestSendError memetakan error dari SendTest ke HTTP respons yang sesuai.
func (h *SendHandler) mapTestSendError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, domain.ErrTemplateNotFound):
		return domain.ErrorResponse(c, fiber.StatusNotFound, "TEMPLATE_NOT_FOUND", err.Error())
	case errors.Is(err, domain.ErrProviderNotConfigured):
		return domain.ErrorResponse(c, fiber.StatusUnprocessableEntity, "PROVIDER_NOT_CONFIGURED", err.Error())
	default:
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal mengirim notifikasi test")
	}
}

// mapManualSendError memetakan error dari SendManual ke HTTP respons yang sesuai.
func (h *SendHandler) mapManualSendError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, domain.ErrCustomerNotFound):
		return domain.ErrorResponse(c, fiber.StatusNotFound, "CUSTOMER_NOT_FOUND", err.Error())
	case errors.Is(err, domain.ErrDailyLimitExceeded):
		return domain.ErrorResponse(c, fiber.StatusTooManyRequests, "DAILY_LIMIT_EXCEEDED", err.Error())
	case errors.Is(err, domain.ErrTemplateNotFound):
		return domain.ErrorResponse(c, fiber.StatusNotFound, "TEMPLATE_NOT_FOUND", err.Error())
	default:
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal mengirim notifikasi")
	}
}

// mapResendError memetakan error dari Resend ke HTTP respons yang sesuai.
func (h *SendHandler) mapResendError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, domain.ErrLogNotFound):
		return domain.ErrorResponse(c, fiber.StatusNotFound, "LOG_NOT_FOUND", err.Error())
	case errors.Is(err, domain.ErrNotResendable):
		return domain.ErrorResponse(c, fiber.StatusUnprocessableEntity, "NOT_RESENDABLE", err.Error())
	case errors.Is(err, domain.ErrDailyLimitExceeded):
		return domain.ErrorResponse(c, fiber.StatusTooManyRequests, "DAILY_LIMIT_EXCEEDED", err.Error())
	default:
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal mengirim ulang notifikasi")
	}
}
