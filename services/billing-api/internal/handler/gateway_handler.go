// gateway_handler.go menangani endpoint konfigurasi payment gateway.
// Termasuk: create, list, update, deactivate, dan test config.
// Endpoint payment link dan walled garden ada di gateway_handler_link.go.
package handler

import (
	"errors"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
	"github.com/ispboss/ispboss/services/billing-api/internal/usecase"
)

// GatewayHandler menangani HTTP request untuk konfigurasi gateway dan payment link.
type GatewayHandler struct {
	gatewayUsecase *usecase.GatewayUsecase
	webhookRepo    domain.WebhookLogRepository
	linkRepo       domain.PaymentLinkRepository
	validate       *validator.Validate
	logger         zerolog.Logger
}

// NewGatewayHandler membuat instance baru GatewayHandler.
func NewGatewayHandler(
	gatewayUsecase *usecase.GatewayUsecase,
	webhookRepo domain.WebhookLogRepository,
	linkRepo domain.PaymentLinkRepository,
	logger zerolog.Logger,
) *GatewayHandler {
	return &GatewayHandler{
		gatewayUsecase: gatewayUsecase,
		webhookRepo:    webhookRepo,
		linkRepo:       linkRepo,
		validate:       validator.New(),
		logger:         logger,
	}
}

// CreateConfig menangani POST /v1/settings/payment-gateways.
// Membuat konfigurasi gateway baru untuk tenant yang terautentikasi.
func (h *GatewayHandler) CreateConfig(c *fiber.Ctx) error {
	tenantID, ok := c.Locals("tenant_id").(string)
	if !ok || tenantID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant tidak teridentifikasi")
	}

	var req domain.CreateGatewayConfigRequest
	if err := c.BodyParser(&req); err != nil {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "request body tidak valid")
	}

	if err := h.validate.Struct(req); err != nil {
		var ve validator.ValidationErrors
		if errors.As(err, &ve) {
			return domain.ErrorResponse(c, fiber.StatusBadRequest, "VALIDATION_ERROR", "validasi gagal", mapValidationErrors(ve)...)
		}
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "VALIDATION_ERROR", err.Error())
	}

	config, err := h.gatewayUsecase.CreateConfig(c.Context(), tenantID, req)
	if err != nil {
		return h.mapGatewayError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusCreated, config)
}

// ListConfigs menangani GET /v1/settings/payment-gateways.
// Mengembalikan semua konfigurasi gateway untuk tenant (API key di-mask).
func (h *GatewayHandler) ListConfigs(c *fiber.Ctx) error {
	tenantID, ok := c.Locals("tenant_id").(string)
	if !ok || tenantID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant tidak teridentifikasi")
	}

	configs, err := h.gatewayUsecase.ListConfigs(c.Context(), tenantID)
	if err != nil {
		h.logger.Error().Err(err).Msg("gagal mengambil daftar konfigurasi gateway")
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal mengambil daftar konfigurasi gateway")
	}

	return domain.SuccessResponse(c, fiber.StatusOK, configs)
}

// UpdateConfig menangani PUT /v1/settings/payment-gateways/:id.
// Memperbarui konfigurasi gateway yang sudah ada.
func (h *GatewayHandler) UpdateConfig(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "config ID wajib diisi")
	}

	var req domain.UpdateGatewayConfigRequest
	if err := c.BodyParser(&req); err != nil {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "request body tidak valid")
	}

	if err := h.validate.Struct(req); err != nil {
		var ve validator.ValidationErrors
		if errors.As(err, &ve) {
			return domain.ErrorResponse(c, fiber.StatusBadRequest, "VALIDATION_ERROR", "validasi gagal", mapValidationErrors(ve)...)
		}
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "VALIDATION_ERROR", err.Error())
	}

	config, err := h.gatewayUsecase.UpdateConfig(c.Context(), id, req)
	if err != nil {
		return h.mapGatewayError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, config)
}

// DeactivateConfig menangani DELETE /v1/settings/payment-gateways/:id.
// Menonaktifkan konfigurasi gateway (soft delete).
func (h *GatewayHandler) DeactivateConfig(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "config ID wajib diisi")
	}

	if err := h.gatewayUsecase.DeactivateConfig(c.Context(), id); err != nil {
		return h.mapGatewayError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, fiber.Map{
		"message": "konfigurasi gateway berhasil dinonaktifkan",
	})
}

// TestConfig menangani POST /v1/settings/payment-gateways/:id/test.
// Menguji koneksi dan kredensial ke payment gateway.
func (h *GatewayHandler) TestConfig(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "config ID wajib diisi")
	}

	result, err := h.gatewayUsecase.TestConfig(c.Context(), id)
	if err != nil {
		return h.mapGatewayError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, result)
}

// mapGatewayError memetakan domain error ke HTTP error response untuk gateway.
func (h *GatewayHandler) mapGatewayError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, domain.ErrGatewayConfigNotFound):
		return domain.ErrorResponse(c, fiber.StatusNotFound, "GATEWAY_CONFIG_NOT_FOUND", err.Error())
	case errors.Is(err, domain.ErrGatewayConfigDuplicate):
		return domain.ErrorResponse(c, fiber.StatusConflict, "GATEWAY_CONFIG_DUPLICATE", err.Error())
	case errors.Is(err, domain.ErrInvalidEnabledMethods):
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "INVALID_ENABLED_METHODS", err.Error())
	case errors.Is(err, domain.ErrEncryptionFailed):
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "ENCRYPTION_FAILED", err.Error())
	case errors.Is(err, domain.ErrDecryptionFailed):
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "DECRYPTION_FAILED", err.Error())
	case errors.Is(err, domain.ErrNoActiveGateway):
		return domain.ErrorResponse(c, fiber.StatusNotFound, "NO_ACTIVE_GATEWAY", err.Error())
	case errors.Is(err, domain.ErrGatewayUnavailable):
		return domain.ErrorResponse(c, fiber.StatusBadGateway, "GATEWAY_UNAVAILABLE", err.Error())
	case errors.Is(err, domain.ErrGatewayInvalidAPIKey):
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "INVALID_API_KEY", err.Error())
	case errors.Is(err, domain.ErrPaymentLinkNotFound):
		return domain.ErrorResponse(c, fiber.StatusNotFound, "PAYMENT_LINK_NOT_FOUND", err.Error())
	case errors.Is(err, domain.ErrPaymentLinkExpired):
		return domain.ErrorResponse(c, fiber.StatusGone, "PAYMENT_LINK_EXPIRED", err.Error())
	default:
		h.logger.Error().Err(err).Msg("internal error pada gateway handler")
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "terjadi kesalahan internal")
	}
}
