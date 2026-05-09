// custom_report_handler.go menangani HTTP permintaan untuk laporan kustom.
// Termasuk: Preview, ListTemplates, CreateTemplate, DeleteTemplate.
package handler

import (
	"errors"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

// CustomReportHandler menangani HTTP permintaan untuk laporan kustom.
type CustomReportHandler struct {
	customReportUsecase domain.CustomReportTemplateUsecase
	validate            *validator.Validate
	logger              zerolog.Logger
}

// NewCustomReportHandler membuat instance baru CustomReportHandler.
func NewCustomReportHandler(customReportUsecase domain.CustomReportTemplateUsecase, logger zerolog.Logger) *CustomReportHandler {
	return &CustomReportHandler{
		customReportUsecase: customReportUsecase,
		validate:            validator.New(),
		logger:              logger,
	}
}

// previewRequest adalah payload untuk preview laporan kustom.
type previewRequest struct {
	Metrics     []string `json:"metrics" validate:"required,min=1,max=3,dive,required"`
	GroupBy     string   `json:"group_by" validate:"required"`
	SubGroupBy  string   `json:"sub_group_by,omitempty"`
	PeriodStart string   `json:"period_start" validate:"required"`
	PeriodEnd   string   `json:"period_end" validate:"required"`
	DisplayType string   `json:"display_type" validate:"required,oneof=table bar_chart line_chart pie_chart"`
}

// Preview menangani POST /v1/reports/kustom/preview.
// Menjalankan laporan kustom tanpa menyimpan template.
func (h *CustomReportHandler) Preview(c *fiber.Ctx) error {
	tenantID, ok := c.Locals("tenant_id").(string)
	if !ok || tenantID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant tidak teridentifikasi")
	}

	var req previewRequest
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

	ps, err := time.Parse("2006-01-02", req.PeriodStart)
	if err != nil {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "format period_start tidak valid")
	}
	pe, err := time.Parse("2006-01-02", req.PeriodEnd)
	if err != nil {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "format period_end tidak valid")
	}

	data, err := h.customReportUsecase.PreviewCustomReport(
		c.Context(), tenantID, req.Metrics, req.GroupBy, req.SubGroupBy, ps, pe, req.DisplayType,
	)
	if err != nil {
		return h.mapCustomReportError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, data)
}

// ListTemplates menangani GET /v1/reports/kustom/templates.
// Mengembalikan semua template laporan kustom untuk tenant.
func (h *CustomReportHandler) ListTemplates(c *fiber.Ctx) error {
	tenantID, ok := c.Locals("tenant_id").(string)
	if !ok || tenantID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant tidak teridentifikasi")
	}

	templates, err := h.customReportUsecase.ListTemplates(c.Context(), tenantID)
	if err != nil {
		h.logger.Error().Err(err).Msg("gagal mengambil daftar template")
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal mengambil daftar template")
	}

	return domain.SuccessResponse(c, fiber.StatusOK, templates)
}

// CreateTemplate menangani POST /v1/reports/kustom/templates.
// Menyimpan konfigurasi laporan kustom sebagai template.
func (h *CustomReportHandler) CreateTemplate(c *fiber.Ctx) error {
	tenantID, ok := c.Locals("tenant_id").(string)
	if !ok || tenantID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant tidak teridentifikasi")
	}

	var req domain.CreateTemplateRequest
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

	actor := h.extractActor(c)

	template, err := h.customReportUsecase.CreateTemplate(c.Context(), tenantID, req, actor)
	if err != nil {
		return h.mapCustomReportError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusCreated, template)
}

// DeleteTemplate menangani DELETE /v1/reports/kustom/templates/:id.
// Menghapus template laporan kustom.
func (h *CustomReportHandler) DeleteTemplate(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "template ID wajib diisi")
	}

	if err := h.customReportUsecase.DeleteTemplate(c.Context(), id); err != nil {
		return h.mapCustomReportError(c, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// extractActor mengambil informasi aktor dari Fiber locals.
func (h *CustomReportHandler) extractActor(c *fiber.Ctx) domain.ActorInfo {
	actorID, _ := c.Locals("user_id").(string)
	actorName, _ := c.Locals("user_name").(string)
	return domain.ActorInfo{
		ActorID:   actorID,
		ActorName: actorName,
	}
}

// mapCustomReportError memetakan domain error ke HTTP error respons.
func (h *CustomReportHandler) mapCustomReportError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, domain.ErrMaxMetricsExceeded):
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "MAX_METRICS_EXCEEDED", err.Error())
	case errors.Is(err, domain.ErrTemplateNotFound):
		return domain.ErrorResponse(c, fiber.StatusNotFound, "TEMPLATE_NOT_FOUND", err.Error())
	default:
		h.logger.Error().Err(err).Msg("internal error pada custom report handler")
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "terjadi kesalahan internal")
	}
}
