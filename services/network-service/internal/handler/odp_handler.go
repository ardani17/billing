// odp_handler.go menangani HTTP request untuk manajemen ODP (Optical Distribution Point).
// Termasuk: CRUD dan list dengan paginasi.
package handler

import (
	"errors"
	"strconv"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"

	"github.com/ispboss/ispboss/pkg/tenant"
	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// ODPHandler menangani HTTP request untuk operasi ODP.
type ODPHandler struct {
	odpManager domain.ODPManager
	validate   *validator.Validate
}

// NewODPHandler membuat instance baru ODPHandler.
func NewODPHandler(odpManager domain.ODPManager) *ODPHandler {
	return &ODPHandler{
		odpManager: odpManager,
		validate:   validator.New(),
	}
}

// CreateODP menangani POST /odp.
// Parse body, validasi, extract tenant_id, lalu buat ODP baru.
func (h *ODPHandler) CreateODP(c *fiber.Ctx) error {
	tenantID := tenant.FromContext(c.UserContext())
	if tenantID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant tidak teridentifikasi")
	}

	var req domain.CreateODPRequest
	if err := c.BodyParser(&req); err != nil {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "request body tidak valid")
	}

	if err := h.validate.Struct(req); err != nil {
		return h.validationError(c, err)
	}

	resp, err := h.odpManager.Create(c.UserContext(), tenantID, req)
	if err != nil {
		return h.mapError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusCreated, resp)
}

// ListODPs menangani GET /odp.
// Parse query params (page, page_size, olt_id, pon_port) dan return paginated list.
func (h *ODPHandler) ListODPs(c *fiber.Ctx) error {
	tenantID := tenant.FromContext(c.UserContext())
	if tenantID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant tidak teridentifikasi")
	}

	page, _ := strconv.Atoi(c.Query("page", "1"))
	pageSize, _ := strconv.Atoi(c.Query("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}

	params := domain.ODPListParams{
		TenantID: tenantID,
		Page:     page,
		PageSize: pageSize,
		OLTID:    c.Query("olt_id"),
	}

	// Parse pon_port filter (opsional)
	if ponPortStr := c.Query("pon_port"); ponPortStr != "" {
		ponPort, err := strconv.Atoi(ponPortStr)
		if err != nil {
			return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "pon_port tidak valid")
		}
		params.PONPortIndex = &ponPort
	}

	result, err := h.odpManager.List(c.UserContext(), params)
	if err != nil {
		return h.mapError(c, err)
	}

	return domain.PaginatedResponse(c, fiber.StatusOK, result.Data, result.Total, result.Page, result.PageSize, result.TotalPages)
}

// GetODP menangani GET /odp/:id.
// Mengambil detail ODP berdasarkan ID.
func (h *ODPHandler) GetODP(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "ODP ID wajib diisi")
	}

	resp, err := h.odpManager.GetByID(c.UserContext(), id)
	if err != nil {
		return h.mapError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, resp)
}

// UpdateODP menangani PUT /odp/:id.
// Parse body, validasi, lalu update data ODP.
func (h *ODPHandler) UpdateODP(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "ODP ID wajib diisi")
	}

	var req domain.UpdateODPRequest
	if err := c.BodyParser(&req); err != nil {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "request body tidak valid")
	}

	if err := h.validate.Struct(req); err != nil {
		return h.validationError(c, err)
	}

	resp, err := h.odpManager.Update(c.UserContext(), id, req)
	if err != nil {
		return h.mapError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, resp)
}

// DeleteODP menangani DELETE /odp/:id.
// Soft-delete ODP.
func (h *ODPHandler) DeleteODP(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "ODP ID wajib diisi")
	}

	if err := h.odpManager.Delete(c.UserContext(), id); err != nil {
		return h.mapError(c, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// mapError memetakan domain error ODP ke HTTP error response.
func (h *ODPHandler) mapError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, domain.ErrODPNotFound):
		return domain.ErrorResponse(c, fiber.StatusNotFound, "ODP_NOT_FOUND", err.Error())
	case errors.Is(err, domain.ErrODPNameExists):
		return domain.ErrorResponse(c, fiber.StatusConflict, "ODP_NAME_EXISTS", err.Error())
	case errors.Is(err, domain.ErrODPFull):
		return domain.ErrorResponse(c, fiber.StatusUnprocessableEntity, "ODP_FULL", err.Error())
	case errors.Is(err, domain.ErrInvalidSplitterType):
		return domain.ErrorResponse(c, fiber.StatusUnprocessableEntity, "INVALID_SPLITTER_TYPE", err.Error())
	case errors.Is(err, domain.ErrOLTNotFound):
		return domain.ErrorResponse(c, fiber.StatusNotFound, "OLT_NOT_FOUND", err.Error())
	default:
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "terjadi kesalahan internal")
	}
}

// validationError menangani error validasi dari go-playground/validator.
func (h *ODPHandler) validationError(c *fiber.Ctx, err error) error {
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
