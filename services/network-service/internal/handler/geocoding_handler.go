// geocoding_handler.go menangani HTTP request untuk reverse geocoding.
// Menerima parameter lat dan lng, return alamat lengkap.
package handler

import (
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v2"

	"github.com/ispboss/ispboss/pkg/tenant"
	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// GeocodingHandler menangani HTTP request untuk reverse geocoding.
type GeocodingHandler struct {
	manager domain.GeocodingManager
}

// NewGeocodingHandler membuat instance baru GeocodingHandler.
func NewGeocodingHandler(manager domain.GeocodingManager) *GeocodingHandler {
	return &GeocodingHandler{manager: manager}
}

// ReverseGeocode menangani GET /geocode/reverse.
// Menerima parameter lat dan lng, return alamat hasil reverse geocoding.
func (h *GeocodingHandler) ReverseGeocode(c *fiber.Ctx) error {
	tenantID := tenant.FromContext(c.UserContext())
	if tenantID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant tidak teridentifikasi")
	}

	latStr := c.Query("lat")
	lngStr := c.Query("lng")
	if latStr == "" || lngStr == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "parameter lat dan lng wajib diisi")
	}

	lat, err := strconv.ParseFloat(latStr, 64)
	if err != nil {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "parameter lat tidak valid")
	}

	lng, err := strconv.ParseFloat(lngStr, 64)
	if err != nil {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "parameter lng tidak valid")
	}

	// Validasi range koordinat
	if lat < -90 || lat > 90 || lng < -180 || lng > 180 {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "INVALID_COORDINATES", "koordinat di luar range valid")
	}

	result, err := h.manager.ReverseGeocode(c.UserContext(), tenantID, lat, lng)
	if err != nil {
		return h.mapError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, result)
}

// mapError memetakan domain error geocoding ke HTTP error response.
func (h *GeocodingHandler) mapError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, domain.ErrGeocodingFailed):
		return domain.ErrorResponse(c, fiber.StatusBadGateway, "GEOCODING_FAILED", err.Error())
	case errors.Is(err, domain.ErrGeocodingRateLimit):
		return domain.ErrorResponse(c, fiber.StatusTooManyRequests, "GEOCODING_RATE_LIMIT", err.Error())
	case errors.Is(err, domain.ErrInvalidCoordinates):
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "INVALID_COORDINATES", err.Error())
	default:
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "terjadi kesalahan internal")
	}
}
