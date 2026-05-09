// report_handler.go menangani HTTP permintaan untuk laporan.
// Berisi ReportHandler struct, constructor, helper parseFilter, dan mapReportError.
// Method-method laporan dipecah ke file terpisah per kategori.
package handler

import (
	"errors"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

// ReportHandler menangani HTTP permintaan untuk semua laporan.
type ReportHandler struct {
	reportUsecase domain.ReportUsecase
	logger        zerolog.Logger
}

// NewReportHandler membuat instance baru ReportHandler.
func NewReportHandler(reportUsecase domain.ReportUsecase, logger zerolog.Logger) *ReportHandler {
	return &ReportHandler{
		reportUsecase: reportUsecase,
		logger:        logger,
	}
}

// parseFilter mengambil parameter filter dari kueri string.
// Mengembalikan ReportFilter dan error jika format tanggal tidak valid.
func (h *ReportHandler) parseFilter(c *fiber.Ctx) (domain.ReportFilter, error) {
	var filter domain.ReportFilter

	periodStart := c.Query("period_start")
	periodEnd := c.Query("period_end")

	if periodStart == "" || periodEnd == "" {
		return filter, errors.New("period_start dan period_end wajib diisi")
	}

	var err error
	filter.PeriodStart, err = time.Parse("2006-01-02", periodStart)
	if err != nil {
		return filter, errors.New("format period_start tidak valid (gunakan YYYY-MM-DD)")
	}

	filter.PeriodEnd, err = time.Parse("2006-01-02", periodEnd)
	if err != nil {
		return filter, errors.New("format period_end tidak valid (gunakan YYYY-MM-DD)")
	}

	if filter.PeriodStart.After(filter.PeriodEnd) {
		return filter, errors.New("period_start tidak boleh lebih besar dari period_end")
	}

	if cs := c.Query("compare_start"); cs != "" {
		t, err := time.Parse("2006-01-02", cs)
		if err != nil {
			return filter, errors.New("format compare_start tidak valid (gunakan YYYY-MM-DD)")
		}
		filter.CompareStart = &t
	}

	if ce := c.Query("compare_end"); ce != "" {
		t, err := time.Parse("2006-01-02", ce)
		if err != nil {
			return filter, errors.New("format compare_end tidak valid (gunakan YYYY-MM-DD)")
		}
		filter.CompareEnd = &t
	}

	filter.AreaID = c.Query("area_id")
	filter.PackageID = c.Query("package_id")
	filter.RouterID = c.Query("router_id")

	return filter, nil
}

// mapReportError memetakan domain error ke HTTP error respons untuk laporan.
func (h *ReportHandler) mapReportError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, domain.ErrInvalidReportType):
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "INVALID_REPORT_TYPE", err.Error())
	case errors.Is(err, domain.ErrInsufficientData):
		return domain.ErrorResponse(c, fiber.StatusUnprocessableEntity, "INSUFFICIENT_DATA", err.Error())
	case errors.Is(err, domain.ErrKPITargetNotFound):
		return domain.ErrorResponse(c, fiber.StatusNotFound, "KPI_TARGET_NOT_FOUND", err.Error())
	default:
		h.logger.Error().Err(err).Msg("internal error pada report handler")
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "terjadi kesalahan internal")
	}
}
