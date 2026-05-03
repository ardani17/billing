// share_handler.go menangani HTTP request untuk share link peta.
// Termasuk: buat share link, list, akses publik (tanpa auth), dan hapus.
package handler

import (
	"errors"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"

	"github.com/ispboss/ispboss/pkg/tenant"
	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// ShareHandler menangani HTTP request untuk operasi share link.
type ShareHandler struct {
	manager  domain.ShareManager
	validate *validator.Validate
}

// NewShareHandler membuat instance baru ShareHandler.
func NewShareHandler(manager domain.ShareManager) *ShareHandler {
	return &ShareHandler{
		manager:  manager,
		validate: validator.New(),
	}
}

// CreateShareLink menangani POST /share.
// Parse body, validasi, buat share link baru dengan opsi expiry dan password.
func (h *ShareHandler) CreateShareLink(c *fiber.Ctx) error {
	tenantID := tenant.FromContext(c.UserContext())
	if tenantID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant tidak teridentifikasi")
	}

	var req domain.CreateShareLinkRequest
	if err := c.BodyParser(&req); err != nil {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "request body tidak valid")
	}

	if err := h.validate.Struct(req); err != nil {
		return h.validationError(c, err)
	}

	resp, err := h.manager.CreateShareLink(c.UserContext(), tenantID, tenantID, req)
	if err != nil {
		return h.mapError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusCreated, resp)
}

// ListShareLinks menangani GET /share.
// Mengambil daftar share link untuk tenant saat ini.
func (h *ShareHandler) ListShareLinks(c *fiber.Ctx) error {
	tenantID := tenant.FromContext(c.UserContext())
	if tenantID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant tidak teridentifikasi")
	}

	links, err := h.manager.ListShareLinks(c.UserContext(), tenantID)
	if err != nil {
		return h.mapError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, links)
}

// GetSharedMap menangani GET /share/:token.
// Endpoint publik (tanpa auth) — akses peta read-only via share token.
// Menerima password opsional via query param atau header.
func (h *ShareHandler) GetSharedMap(c *fiber.Ctx) error {
	token := c.Params("token")
	if token == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "token wajib diisi")
	}

	// Password bisa dikirim via query param atau header X-Share-Password
	password := c.Query("password")
	if password == "" {
		password = c.Get("X-Share-Password")
	}

	data, err := h.manager.GetSharedMap(c.UserContext(), token, password)
	if err != nil {
		return h.mapError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, data)
}

// DeleteShareLink menangani DELETE /share/:token.
// Menghapus share link berdasarkan token.
func (h *ShareHandler) DeleteShareLink(c *fiber.Ctx) error {
	token := c.Params("token")
	if token == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "token wajib diisi")
	}

	if err := h.manager.DeleteShareLink(c.UserContext(), token); err != nil {
		return h.mapError(c, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// mapError memetakan domain error share link ke HTTP error response.
func (h *ShareHandler) mapError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, domain.ErrShareLinkNotFound):
		return domain.ErrorResponse(c, fiber.StatusNotFound, "SHARE_LINK_NOT_FOUND", err.Error())
	case errors.Is(err, domain.ErrShareLinkExpired):
		return domain.ErrorResponse(c, fiber.StatusGone, "SHARE_LINK_EXPIRED", err.Error())
	case errors.Is(err, domain.ErrShareLinkPassword):
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "SHARE_LINK_PASSWORD", err.Error())
	default:
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "terjadi kesalahan internal")
	}
}

// validationError menangani error validasi dari go-playground/validator.
func (h *ShareHandler) validationError(c *fiber.Ctx, err error) error {
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
