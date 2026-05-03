// export_handler.go menangani HTTP request untuk export laporan.
// Termasuk: RequestExport (async PDF/XLSX, sync CSV) dan Status.
package handler

import (
	"errors"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

// ExportHandler menangani HTTP request untuk export laporan.
type ExportHandler struct {
	reportUsecase domain.ReportUsecase
	validate      *validator.Validate
	logger        zerolog.Logger
}

// NewExportHandler membuat instance baru ExportHandler.
func NewExportHandler(reportUsecase domain.ReportUsecase, logger zerolog.Logger) *ExportHandler {
	return &ExportHandler{
		reportUsecase: reportUsecase,
		validate:      validator.New(),
		logger:        logger,
	}
}

// RequestExport menangani POST /v1/reports/export.
// Untuk CSV: mengembalikan file langsung. Untuk PDF/XLSX: mengembalikan 202 dengan job_id.
func (h *ExportHandler) RequestExport(c *fiber.Ctx) error {
	tenantID, ok := c.Locals("tenant_id").(string)
	if !ok || tenantID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant tidak teridentifikasi")
	}

	userID, _ := c.Locals("user_id").(string)

	var req domain.ExportRequest
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

	var filter domain.ReportFilter
	if req.Filters != nil {
		filter = *req.Filters
	}

	jobID, err := h.reportUsecase.RequestExport(c.Context(), tenantID, userID, req.ReportType, req.Format, filter)
	if err != nil {
		return h.mapExportError(c, err)
	}

	// Untuk async export (PDF/XLSX), kembalikan 202 Accepted dengan job_id
	return domain.SuccessResponse(c, fiber.StatusAccepted, fiber.Map{
		"job_id":  jobID,
		"message": "export sedang diproses",
	})
}

// Status menangani GET /v1/reports/export/:job_id.
// Mengembalikan status job export beserta download_url jika sudah selesai.
func (h *ExportHandler) Status(c *fiber.Ctx) error {
	jobID := c.Params("job_id")
	if jobID == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "job_id wajib diisi")
	}

	job, err := h.reportUsecase.GetExportStatus(c.Context(), jobID)
	if err != nil {
		return h.mapExportError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, job)
}

// mapExportError memetakan domain error ke HTTP error response untuk export.
func (h *ExportHandler) mapExportError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, domain.ErrInvalidReportType):
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "INVALID_REPORT_TYPE", err.Error())
	case errors.Is(err, domain.ErrInvalidExportFormat):
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "INVALID_EXPORT_FORMAT", err.Error())
	case errors.Is(err, domain.ErrReportJobNotFound):
		return domain.ErrorResponse(c, fiber.StatusNotFound, "JOB_NOT_FOUND", err.Error())
	default:
		h.logger.Error().Err(err).Msg("internal error pada export handler")
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "terjadi kesalahan internal")
	}
}
