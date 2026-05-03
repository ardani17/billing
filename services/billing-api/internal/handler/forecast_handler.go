// forecast_handler.go menangani HTTP request untuk proyeksi/forecasting.
// Termasuk: Forecast (proyeksi 3 bulan ke depan berdasarkan data historis).
package handler

import (
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

// ForecastHandler menangani HTTP request untuk proyeksi bisnis.
type ForecastHandler struct {
	forecastUsecase domain.ForecastUsecase
	logger          zerolog.Logger
}

// NewForecastHandler membuat instance baru ForecastHandler.
func NewForecastHandler(forecastUsecase domain.ForecastUsecase, logger zerolog.Logger) *ForecastHandler {
	return &ForecastHandler{
		forecastUsecase: forecastUsecase,
		logger:          logger,
	}
}

// Forecast menangani GET /v1/reports/forecast.
// Mengembalikan proyeksi 3 bulan ke depan berdasarkan data historis 6 bulan.
func (h *ForecastHandler) Forecast(c *fiber.Ctx) error {
	tenantID, ok := c.Locals("tenant_id").(string)
	if !ok || tenantID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant tidak teridentifikasi")
	}

	report, err := h.forecastUsecase.GetForecastReport(c.Context(), tenantID)
	if err != nil {
		if errors.Is(err, domain.ErrInsufficientData) {
			// Data belum cukup, kembalikan response dengan flag insufficient_data
			return domain.SuccessResponse(c, fiber.StatusOK, &domain.ForecastReport{
				InsufficientData: true,
				Disclaimer:       "Data historis belum cukup untuk proyeksi. Minimal 3 bulan data diperlukan.",
			})
		}
		h.logger.Error().Err(err).Msg("gagal mengambil proyeksi")
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal mengambil proyeksi")
	}

	return domain.SuccessResponse(c, fiber.StatusOK, report)
}
