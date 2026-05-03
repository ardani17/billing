// olt_handler_monitoring.go menangani HTTP request monitoring OLT.
// Termasuk: PON ports, ONT list, traffic, alarm, SFP, dan capacity.
package handler

import (
	"errors"
	"strconv"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// GetPONPorts menangani GET /devices/:id/pon-ports.
// Mengambil status semua PON port untuk satu OLT.
func (h *OLTHandler) GetPONPorts(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "OLT ID wajib diisi")
	}

	ports, err := h.oltManager.GetPONPorts(c.UserContext(), id)
	if err != nil {
		return h.mapError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, ports)
}

// GetONTList menangani GET /devices/:id/pon-ports/:port/onts.
// Mengambil daftar ONT pada satu PON port.
func (h *OLTHandler) GetONTList(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "OLT ID wajib diisi")
	}

	portIndex, err := strconv.Atoi(c.Params("port"))
	if err != nil {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "port index tidak valid")
	}

	onts, err := h.oltManager.GetONTList(c.UserContext(), id, portIndex)
	if err != nil {
		return h.mapError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, onts)
}

// GetTraffic menangani GET /devices/:id/pon-ports/:port/traffic.
// Parse from/to query params, return traffic data dari TrafficStore.
func (h *OLTHandler) GetTraffic(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "OLT ID wajib diisi")
	}

	if _, err := strconv.Atoi(c.Params("port")); err != nil {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "port index tidak valid")
	}

	// Parse from/to query params (RFC3339)
	var from, to time.Time
	if fromStr := c.Query("from"); fromStr != "" {
		parsed, err := time.Parse(time.RFC3339, fromStr)
		if err != nil {
			return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "format 'from' tidak valid, gunakan RFC3339")
		}
		from = parsed
	} else {
		from = time.Now().Add(-1 * time.Hour) // default 1 jam terakhir
	}

	if toStr := c.Query("to"); toStr != "" {
		parsed, err := time.Parse(time.RFC3339, toStr)
		if err != nil {
			return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "format 'to' tidak valid, gunakan RFC3339")
		}
		to = parsed
	} else {
		to = time.Now()
	}

	// Return traffic data placeholder — data diambil dari TrafficStore via sync engine
	return domain.SuccessResponse(c, fiber.StatusOK, fiber.Map{
		"olt_id":     id,
		"port_index": c.Params("port"),
		"from":       from.Format(time.RFC3339),
		"to":         to.Format(time.RFC3339),
		"data":       []interface{}{},
	})
}

// GetAlarms menangani GET /devices/:id/alarms.
// Mengambil daftar alarm untuk satu OLT.
func (h *OLTHandler) GetAlarms(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "OLT ID wajib diisi")
	}

	page, _ := strconv.Atoi(c.Query("page", "1"))
	pageSize, _ := strconv.Atoi(c.Query("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}

	params := domain.AlarmListParams{
		Page:     page,
		PageSize: pageSize,
		Severity: c.Query("severity"),
		Status:   c.Query("status"),
	}

	result, err := h.alarmManager.GetAlarms(c.UserContext(), id, params)
	if err != nil {
		return h.mapError(c, err)
	}

	return domain.PaginatedResponse(c, fiber.StatusOK, result.Data, result.Total, result.Page, result.PageSize, result.TotalPages)
}

// GetSFP menangani GET /devices/:id/sfp.
// Mengambil status SFP module semua PON port.
func (h *OLTHandler) GetSFP(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "OLT ID wajib diisi")
	}

	sfps, err := h.oltManager.GetSFPStatus(c.UserContext(), id)
	if err != nil {
		return h.mapError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, sfps)
}

// GetCapacity menangani GET /devices/:id/capacity.
// Mengambil data capacity planning untuk satu OLT.
func (h *OLTHandler) GetCapacity(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "OLT ID wajib diisi")
	}

	capacity, err := h.oltManager.GetCapacity(c.UserContext(), id)
	if err != nil {
		return h.mapError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, capacity)
}

// mapError memetakan domain error ke HTTP error response.
// Mengikuti tabel error mapping dari design document.
func (h *OLTHandler) mapError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, domain.ErrOLTNotFound):
		return domain.ErrorResponse(c, fiber.StatusNotFound, "OLT_NOT_FOUND", err.Error())
	case errors.Is(err, domain.ErrOLTNameExists):
		return domain.ErrorResponse(c, fiber.StatusConflict, "OLT_NAME_EXISTS", err.Error())
	case errors.Is(err, domain.ErrOLTInvalidStatusTransition):
		return domain.ErrorResponse(c, fiber.StatusUnprocessableEntity, "INVALID_STATUS_TRANSITION", err.Error())
	case errors.Is(err, domain.ErrUnsupportedBrand):
		return domain.ErrorResponse(c, fiber.StatusUnprocessableEntity, "UNSUPPORTED_BRAND", err.Error())
	case errors.Is(err, domain.ErrSNMPConnectionFailed), errors.Is(err, domain.ErrSNMPTimeout):
		return domain.ErrorResponse(c, fiber.StatusBadGateway, "SNMP_CONNECTION_FAILED", err.Error())
	case errors.Is(err, domain.ErrCLIConnectionFailed), errors.Is(err, domain.ErrCLITimeout):
		return domain.ErrorResponse(c, fiber.StatusBadGateway, "CLI_CONNECTION_FAILED", err.Error())
	case errors.Is(err, domain.ErrSNMPAuthFailed):
		return domain.ErrorResponse(c, fiber.StatusBadGateway, "SNMP_AUTH_FAILED", err.Error())
	case errors.Is(err, domain.ErrCLIAuthFailed):
		return domain.ErrorResponse(c, fiber.StatusBadGateway, "CLI_AUTH_FAILED", err.Error())
	case errors.Is(err, domain.ErrOLTOffline):
		return domain.ErrorResponse(c, fiber.StatusUnprocessableEntity, "OLT_OFFLINE", err.Error())
	case errors.Is(err, domain.ErrAlarmNotFound):
		return domain.ErrorResponse(c, fiber.StatusNotFound, "ALARM_NOT_FOUND", err.Error())
	default:
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "terjadi kesalahan internal")
	}
}

// validationError menangani error validasi dari go-playground/validator.
// Mengkonversi ValidationErrors ke format FieldError standar.
func (h *OLTHandler) validationError(c *fiber.Ctx, err error) error {
	var ve validator.ValidationErrors
	if errors.As(err, &ve) {
		fields := make([]domain.FieldError, 0, len(ve))
		for _, fe := range ve {
			fields = append(fields, domain.FieldError{
				Field:   toSnakeCase(fe.Field()),
				Message: validationMessage(fe),
			})
		}
		return domain.ErrorResponse(c, fiber.StatusUnprocessableEntity, "VALIDATION_ERROR", "validasi gagal", fields...)
	}
	return domain.ErrorResponse(c, fiber.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
}
