// router_handler.go menangani HTTP permintaan untuk manajemen router MikroTik.
// Termasuk: CRUD, test connection, reboot, dan list dengan paginasi.
package handler

import (
	"errors"
	"strconv"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"

	"github.com/ispboss/ispboss/pkg/tenant"
	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// RouterHandler menangani HTTP permintaan untuk operasi router MikroTik.
type RouterHandler struct {
	usecase  domain.RouterUsecase
	validate *validator.Validate
}

// NewRouterHandler membuat instance baru RouterHandler.
func NewRouterHandler(usecase domain.RouterUsecase) *RouterHandler {
	return &RouterHandler{
		usecase:  usecase,
		validate: validator.New(),
	}
}

// Buat menangani POST /api/v1/mikrotik/routers.
// Parsing body, validasi, extract tenant_id, lalu buat router baru.
func (h *RouterHandler) Create(c *fiber.Ctx) error {
	tenantID := tenant.FromContext(c.UserContext())
	if tenantID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant tidak teridentifikasi")
	}

	var req domain.CreateRouterRequest
	if err := c.BodyParser(&req); err != nil {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "request body tidak valid")
	}

	if err := h.validate.Struct(req); err != nil {
		return h.validationError(c, err)
	}

	resp, err := h.usecase.Create(c.UserContext(), tenantID, req)
	if err != nil {
		return h.mapError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusCreated, resp)
}

// GetByID menangani GET /api/v1/mikrotik/routers/:id.
// Mengambil detail router berdasarkan ID termasuk live metrics jika online.
func (h *RouterHandler) GetByID(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "router ID wajib diisi")
	}

	resp, err := h.usecase.GetByID(c.UserContext(), id)
	if err != nil {
		return h.mapError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, resp)
}

// Perbarui menangani PUT /api/v1/mikrotik/routers/:id.
// Parsing body, validasi, lalu perbarui data router.
func (h *RouterHandler) Update(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "router ID wajib diisi")
	}

	var req domain.UpdateRouterRequest
	if err := c.BodyParser(&req); err != nil {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "request body tidak valid")
	}

	if err := h.validate.Struct(req); err != nil {
		return h.validationError(c, err)
	}

	resp, err := h.usecase.Update(c.UserContext(), id, req)
	if err != nil {
		return h.mapError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, resp)
}

// Hapus menangani DELETE /api/v1/mikrotik/routers/:id.
// Soft-hapus router dan tutup pool koneksi.
func (h *RouterHandler) Delete(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "router ID wajib diisi")
	}

	if err := h.usecase.Delete(c.UserContext(), id); err != nil {
		return h.mapError(c, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// List menangani GET /api/v1/mikrotik/routers.
func (h *RouterHandler) List(c *fiber.Ctx) error {
	tenantID := tenant.FromContext(c.UserContext())
	if tenantID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant tidak teridentifikasi")
	}

	// Parsing kueri params dengan bawaan values
	page, _ := strconv.Atoi(c.Query("page", "1"))
	pageSize, _ := strconv.Atoi(c.Query("page_size", "20"))

	// Validasi batas minimum
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}

	params := domain.RouterListParams{
		TenantID: tenantID,
		Page:     page,
		PageSize: pageSize,
		Status:   c.Query("status"),
		Search:   c.Query("search"),
	}

	result, err := h.usecase.List(c.UserContext(), params)
	if err != nil {
		return h.mapError(c, err)
	}

	return domain.PaginatedResponse(c, fiber.StatusOK, result.Data, result.Total, result.Page, result.PageSize, result.TotalPages)
}

// TestConnection menangani POST /api/v1/mikrotik/routers/:id/test.
// Menguji koneksi ke router dan mengembalikan system info.
func (h *RouterHandler) TestConnection(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "router ID wajib diisi")
	}

	sysResource, err := h.usecase.TestConnection(c.UserContext(), id)
	if err != nil {
		return h.mapError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, sysResource)
}

// Reboot menangani POST /api/v1/mikrotik/routers/:id/reboot.
// Parsing body, validasi konfirmasi nama, lalu kirim perintah reboot.
func (h *RouterHandler) Reboot(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "router ID wajib diisi")
	}

	var req domain.RebootRequest
	if err := c.BodyParser(&req); err != nil {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "request body tidak valid")
	}

	if err := h.validate.Struct(req); err != nil {
		return h.validationError(c, err)
	}

	if err := h.usecase.Reboot(c.UserContext(), id, req.ConfirmationName); err != nil {
		return h.mapError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, fiber.Map{
		"message": "perintah reboot berhasil dikirim",
	})
}

// mapError memetakan domain error ke HTTP error respons.
// Mengikuti tabel error mapping dari design document.
func (h *RouterHandler) mapError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, domain.ErrRouterNotFound):
		return domain.ErrorResponse(c, fiber.StatusNotFound, "ROUTER_NOT_FOUND", err.Error())
	case errors.Is(err, domain.ErrRouterNameExists):
		return domain.ErrorResponse(c, fiber.StatusConflict, "ROUTER_NAME_EXISTS", err.Error())
	case errors.Is(err, domain.ErrInvalidStatusTransition):
		return domain.ErrorResponse(c, fiber.StatusUnprocessableEntity, "INVALID_STATUS_TRANSITION", err.Error())
	case errors.Is(err, domain.ErrConfirmationMismatch):
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "CONFIRMATION_MISMATCH", err.Error())
	case errors.Is(err, domain.ErrRouterOffline):
		return domain.ErrorResponse(c, fiber.StatusUnprocessableEntity, "ROUTER_OFFLINE", err.Error())
	case errors.Is(err, domain.ErrConnectionFailed):
		return domain.ErrorResponse(c, fiber.StatusBadGateway, "CONNECTION_FAILED", err.Error())
	case errors.Is(err, domain.ErrConnectionTimeout):
		return domain.ErrorResponse(c, fiber.StatusGatewayTimeout, "CONNECTION_TIMEOUT", err.Error())
	case errors.Is(err, domain.ErrPoolExhausted):
		return domain.ErrorResponse(c, fiber.StatusServiceUnavailable, "POOL_EXHAUSTED", err.Error())
	case errors.Is(err, domain.ErrRateLimited):
		return domain.ErrorResponse(c, fiber.StatusTooManyRequests, "RATE_LIMITED", err.Error())
	case errors.Is(err, domain.ErrEncryptionFailed), errors.Is(err, domain.ErrDecryptionFailed):
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "terjadi kesalahan internal")
	default:
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "terjadi kesalahan internal")
	}
}

// validationError menangani error validasi dari go-playground/validator.
// Mengkonversi ValidationErrors ke format FieldError standar.
func (h *RouterHandler) validationError(c *fiber.Ctx, err error) error {
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

// validationMessage menghasilkan pesan error deskriptif berdasarkan tag validasi.
func validationMessage(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return fe.Field() + " wajib diisi"
	case "min":
		return fe.Field() + " minimal " + fe.Param() + " karakter"
	case "max":
		return fe.Field() + " maksimal " + fe.Param() + " karakter"
	case "oneof":
		return fe.Field() + " harus salah satu dari: " + fe.Param()
	default:
		return fe.Field() + " tidak valid"
	}
}

// toSnakeCase mengkonversi PascalCase/camelCase ke snake_case.
func toSnakeCase(s string) string {
	var result strings.Builder
	runes := []rune(s)
	for i, r := range runes {
		if i > 0 && r >= 'A' && r <= 'Z' {
			prev := runes[i-1]
			if prev >= 'a' && prev <= 'z' {
				result.WriteByte('_')
			} else if prev >= 'A' && prev <= 'Z' && i+1 < len(runes) && runes[i+1] >= 'a' && runes[i+1] <= 'z' {
				result.WriteByte('_')
			}
		}
		result.WriteRune(r)
	}
	return strings.ToLower(result.String())
}
