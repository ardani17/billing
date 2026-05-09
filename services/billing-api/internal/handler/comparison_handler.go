// comparison_handler.go menangani HTTP permintaan untuk perbandingan antar periode.
// Termasuk: Compare (MoM, YoY, QoQ, Custom).
package handler

import (
	"errors"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

// ComparisonHandler menangani HTTP permintaan untuk perbandingan periode.
type ComparisonHandler struct {
	comparisonUsecase domain.ComparisonUsecase
	logger            zerolog.Logger
}

// NewComparisonHandler membuat instance baru ComparisonHandler.
func NewComparisonHandler(comparisonUsecase domain.ComparisonUsecase, logger zerolog.Logger) *ComparisonHandler {
	return &ComparisonHandler{
		comparisonUsecase: comparisonUsecase,
		logger:            logger,
	}
}

// Compare menangani GET /v1/reports/comparison.
// Mengembalikan laporan perbandingan metrik antara dua periode.
func (h *ComparisonHandler) Compare(c *fiber.Ctx) error {
	tenantID, ok := c.Locals("tenant_id").(string)
	if !ok || tenantID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant tidak teridentifikasi")
	}

	// Parsing comparison_type (wajib)
	compTypeStr := c.Query("comparison_type")
	if compTypeStr == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "comparison_type wajib diisi (mom, yoy, qoq, custom)")
	}

	compType := domain.ComparisonType(compTypeStr)
	switch compType {
	case domain.ComparisonMoM, domain.ComparisonYoY, domain.ComparisonQoQ, domain.ComparisonCustom:
		// valid
	default:
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "comparison_type harus salah satu dari: mom, yoy, qoq, custom")
	}

	// Parsing periode dasar (wajib)
	basePeriodStart := c.Query("period_start")
	basePeriodEnd := c.Query("period_end")
	if basePeriodStart == "" || basePeriodEnd == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "period_start dan period_end wajib diisi")
	}

	bps, err := time.Parse("2006-01-02", basePeriodStart)
	if err != nil {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "format period_start tidak valid (gunakan YYYY-MM-DD)")
	}

	bpe, err := time.Parse("2006-01-02", basePeriodEnd)
	if err != nil {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "format period_end tidak valid (gunakan YYYY-MM-DD)")
	}

	// Parsing compare period (opsional, wajib untuk kustom)
	var comparePeriodStart, comparePeriodEnd *time.Time
	if cs := c.Query("compare_start"); cs != "" {
		t, err := time.Parse("2006-01-02", cs)
		if err != nil {
			return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "format compare_start tidak valid (gunakan YYYY-MM-DD)")
		}
		comparePeriodStart = &t
	}
	if ce := c.Query("compare_end"); ce != "" {
		t, err := time.Parse("2006-01-02", ce)
		if err != nil {
			return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "format compare_end tidak valid (gunakan YYYY-MM-DD)")
		}
		comparePeriodEnd = &t
	}

	// Untuk kustom, compare period wajib
	if compType == domain.ComparisonCustom && (comparePeriodStart == nil || comparePeriodEnd == nil) {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "compare_start dan compare_end wajib diisi untuk comparison_type=custom")
	}

	report, err := h.comparisonUsecase.GetComparisonReport(
		c.Context(), tenantID, compType, bps, bpe, comparePeriodStart, comparePeriodEnd,
	)
	if err != nil {
		if errors.Is(err, domain.ErrInsufficientData) {
			return domain.ErrorResponse(c, fiber.StatusUnprocessableEntity, "INSUFFICIENT_DATA", err.Error())
		}
		h.logger.Error().Err(err).Msg("gagal mengambil laporan perbandingan")
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal mengambil laporan perbandingan")
	}

	return domain.SuccessResponse(c, fiber.StatusOK, report)
}
