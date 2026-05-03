// service_profile_handler.go menangani HTTP request untuk manajemen service profile per OLT.
// Termasuk: CRUD service profile (create, list, update, delete).
package handler

import (
	"errors"
	"strconv"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"

	"github.com/ispboss/ispboss/pkg/tenant"
	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// ServiceProfileHandler menangani HTTP request untuk operasi service profile.
type ServiceProfileHandler struct {
	manager  domain.ServiceProfileManager
	validate *validator.Validate
}

// NewServiceProfileHandler membuat instance baru ServiceProfileHandler.
func NewServiceProfileHandler(manager domain.ServiceProfileManager) *ServiceProfileHandler {
	return &ServiceProfileHandler{
		manager:  manager,
		validate: validator.New(),
	}
}

// CreateServiceProfile menangani POST /devices/:id/service-profiles.
// Parse body, validasi, set OLT ID dari path, lalu buat service profile baru.
func (h *ServiceProfileHandler) CreateServiceProfile(c *fiber.Ctx) error {
	tenantID := tenant.FromContext(c.UserContext())
	if tenantID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant tidak teridentifikasi")
	}

	oltID := c.Params("id")
	if oltID == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "OLT ID wajib diisi")
	}

	var req domain.CreateServiceProfileRequest
	if err := c.BodyParser(&req); err != nil {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "request body tidak valid")
	}
	req.OLTID = oltID

	if err := h.validate.Struct(req); err != nil {
		return h.validationError(c, err)
	}

	resp, err := h.manager.Create(c.UserContext(), tenantID, req)
	if err != nil {
		return h.mapError(c, err)
	}
	return domain.SuccessResponse(c, fiber.StatusCreated, resp)
}

// ListServiceProfiles menangani GET /devices/:id/service-profiles.
// Parse query params (page, page_size) dan return paginated list.
func (h *ServiceProfileHandler) ListServiceProfiles(c *fiber.Ctx) error {
	oltID := c.Params("id")
	if oltID == "" {
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

	params := domain.ServiceProfileListParams{
		Page:     page,
		PageSize: pageSize,
	}

	result, err := h.manager.List(c.UserContext(), oltID, params)
	if err != nil {
		return h.mapError(c, err)
	}
	return domain.PaginatedResponse(c, fiber.StatusOK, result.Data, result.Total, result.Page, result.PageSize, result.TotalPages)
}

// UpdateServiceProfile menangani PUT /service-profiles/:id.
// Parse body, validasi, lalu update data service profile.
func (h *ServiceProfileHandler) UpdateServiceProfile(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "service profile ID wajib diisi")
	}

	var req domain.UpdateServiceProfileRequest
	if err := c.BodyParser(&req); err != nil {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "request body tidak valid")
	}
	if err := h.validate.Struct(req); err != nil {
		return h.validationError(c, err)
	}

	resp, err := h.manager.Update(c.UserContext(), id, req)
	if err != nil {
		return h.mapError(c, err)
	}
	return domain.SuccessResponse(c, fiber.StatusOK, resp)
}

// DeleteServiceProfile menangani DELETE /service-profiles/:id.
// Soft-delete service profile (cek tidak ada ONT aktif yang menggunakan).
func (h *ServiceProfileHandler) DeleteServiceProfile(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "service profile ID wajib diisi")
	}

	if err := h.manager.Delete(c.UserContext(), id); err != nil {
		return h.mapError(c, err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// mapError memetakan domain error service profile ke HTTP error response.
func (h *ServiceProfileHandler) mapError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, domain.ErrServiceProfileNotFound):
		return domain.ErrorResponse(c, fiber.StatusNotFound, "SERVICE_PROFILE_NOT_FOUND", err.Error())
	case errors.Is(err, domain.ErrServiceProfileExists):
		return domain.ErrorResponse(c, fiber.StatusConflict, "SERVICE_PROFILE_EXISTS", err.Error())
	case errors.Is(err, domain.ErrServiceProfileInUse):
		return domain.ErrorResponse(c, fiber.StatusConflict, "SERVICE_PROFILE_IN_USE", err.Error())
	case errors.Is(err, domain.ErrOLTNotFound):
		return domain.ErrorResponse(c, fiber.StatusNotFound, "OLT_NOT_FOUND", err.Error())
	default:
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "terjadi kesalahan internal")
	}
}

// validationError menangani error validasi dari go-playground/validator.
func (h *ServiceProfileHandler) validationError(c *fiber.Ctx, err error) error {
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
