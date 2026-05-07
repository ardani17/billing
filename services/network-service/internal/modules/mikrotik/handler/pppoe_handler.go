// pppoe_handler.go menangani HTTP permintaan untuk manajemen PPPoE user.
// Termasuk: CRUD user, disconnect, sync status, dan trigger sync.
package handler

import (
	"errors"
	"strconv"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"

	"github.com/ispboss/ispboss/pkg/tenant"
	"github.com/ispboss/ispboss/services/network-service/internal/domain"
	"github.com/ispboss/ispboss/services/network-service/internal/modules/mikrotik/usecase"
)

// PPPoEHandler menangani HTTP permintaan untuk operasi PPPoE user.
type PPPoEHandler struct {
	manager  usecase.PPPoEManager
	logger   zerolog.Logger
	validate *validator.Validate
}

// NewPPPoEHandler membuat instance baru PPPoEHandler.
func NewPPPoEHandler(manager usecase.PPPoEManager, logger zerolog.Logger) *PPPoEHandler {
	return &PPPoEHandler{manager: manager, logger: logger, validate: validator.New()}
}

// ListUsers menangani GET /api/v1/mikrotik/routers/:id/pppoe/users.
func (h *PPPoEHandler) ListUsers(c *fiber.Ctx) error {
	routerID := c.Params("id")
	if routerID == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "router ID wajib diisi")
	}
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

	params := domain.PPPoEUserListParams{
		RouterID: routerID, TenantID: tenantID,
		Page: page, PageSize: pageSize,
		SyncStatus: c.Query("sync_status"), Search: c.Query("search"),
	}
	result, err := h.manager.ListUsers(c.UserContext(), routerID, params)
	if err != nil {
		return h.mapError(c, err)
	}
	return domain.PaginatedResponse(c, fiber.StatusOK, result.Data, result.Total, result.Page, result.PageSize, result.TotalPages)
}

// CreateUser menangani POST /api/v1/mikrotik/routers/:id/pppoe/users.
func (h *PPPoEHandler) CreateUser(c *fiber.Ctx) error {
	routerID := c.Params("id")
	if routerID == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "router ID wajib diisi")
	}
	var req domain.CreatePPPoEUserRequest
	if err := c.BodyParser(&req); err != nil {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "request body tidak valid")
	}
	if err := h.validate.Struct(req); err != nil {
		return h.validationError(c, err)
	}
	user, err := h.manager.CreateUser(c.UserContext(), routerID, req)
	if err != nil {
		return h.mapError(c, err)
	}
	return domain.SuccessResponse(c, fiber.StatusCreated, user)
}

// UpdateUser menangani PUT /api/v1/mikrotik/routers/:id/pppoe/users/:user_id.
func (h *PPPoEHandler) UpdateUser(c *fiber.Ctx) error {
	routerID := c.Params("id")
	userID := c.Params("user_id")
	if routerID == "" || userID == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "router ID dan user ID wajib diisi")
	}
	var req domain.UpdatePPPoEUserRequest
	if err := c.BodyParser(&req); err != nil {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "request body tidak valid")
	}
	if err := h.validate.Struct(req); err != nil {
		return h.validationError(c, err)
	}
	user, err := h.manager.UpdateUser(c.UserContext(), routerID, userID, req)
	if err != nil {
		return h.mapError(c, err)
	}
	return domain.SuccessResponse(c, fiber.StatusOK, user)
}

// DeleteUser menangani DELETE /api/v1/mikrotik/routers/:id/pppoe/users/:user_id.
func (h *PPPoEHandler) DeleteUser(c *fiber.Ctx) error {
	routerID := c.Params("id")
	userID := c.Params("user_id")
	if routerID == "" || userID == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "router ID dan user ID wajib diisi")
	}
	if err := h.manager.DeleteUser(c.UserContext(), routerID, userID); err != nil {
		return h.mapError(c, err)
	}
	return domain.SuccessResponse(c, fiber.StatusOK, fiber.Map{"message": "pppoe user berhasil dihapus"})
}

// DisconnectUser menangani POST /api/v1/mikrotik/routers/:id/pppoe/users/:user_id/disconnect.
// Mencari active session untuk user lalu memutusnya.
func (h *PPPoEHandler) DisconnectUser(c *fiber.Ctx) error {
	routerID := c.Params("id")
	userID := c.Params("user_id")
	if routerID == "" || userID == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "router ID dan user ID wajib diisi")
	}

	if err := h.manager.DisconnectUser(c.UserContext(), routerID, userID); err != nil {
		if errors.Is(err, domain.ErrSessionNotFound) {
			return domain.SuccessResponse(c, fiber.StatusOK, fiber.Map{"message": "tidak ada active session untuk user ini"})
		}
		return h.mapError(c, err)
	}
	return domain.SuccessResponse(c, fiber.StatusOK, fiber.Map{"message": "session berhasil diputus"})
}

// GetSyncStatus menangani GET /api/v1/mikrotik/routers/:id/pppoe/sync-status.
func (h *PPPoEHandler) GetSyncStatus(c *fiber.Ctx) error {
	routerID := c.Params("id")
	if routerID == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "router ID wajib diisi")
	}
	summary, err := h.manager.GetSyncStatus(c.UserContext(), routerID)
	if err != nil {
		return h.mapError(c, err)
	}
	return domain.SuccessResponse(c, fiber.StatusOK, summary)
}

// TriggerSync menangani POST /api/v1/mikrotik/routers/:id/pppoe/sync.
func (h *PPPoEHandler) TriggerSync(c *fiber.Ctx) error {
	routerID := c.Params("id")
	if routerID == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "router ID wajib diisi")
	}
	result, err := h.manager.SyncRouter(c.UserContext(), routerID)
	if err != nil {
		return h.mapError(c, err)
	}
	return domain.SuccessResponse(c, fiber.StatusOK, result)
}

// mapError memetakan domain error ke HTTP error respons.
func (h *PPPoEHandler) mapError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, domain.ErrPPPoEUserNotFound):
		return domain.ErrorResponse(c, fiber.StatusNotFound, "PPPOE_USER_NOT_FOUND", err.Error())
	case errors.Is(err, domain.ErrPPPoEUsernameExists):
		return domain.ErrorResponse(c, fiber.StatusConflict, "PPPOE_USERNAME_EXISTS", err.Error())
	case errors.Is(err, domain.ErrRouterNotFound):
		return domain.ErrorResponse(c, fiber.StatusNotFound, "ROUTER_NOT_FOUND", err.Error())
	case errors.Is(err, domain.ErrRouterOffline):
		return domain.ErrorResponse(c, fiber.StatusServiceUnavailable, "ROUTER_OFFLINE", err.Error())
	case errors.Is(err, domain.ErrSyncInProgress):
		return domain.ErrorResponse(c, fiber.StatusConflict, "SYNC_IN_PROGRESS", err.Error())
	case errors.Is(err, domain.ErrSessionNotFound):
		return domain.ErrorResponse(c, fiber.StatusNotFound, "SESSION_NOT_FOUND", err.Error())
	case errors.Is(err, domain.ErrConnectionFailed):
		return domain.ErrorResponse(c, fiber.StatusBadGateway, "CONNECTION_FAILED", err.Error())
	case errors.Is(err, domain.ErrConnectionTimeout):
		return domain.ErrorResponse(c, fiber.StatusGatewayTimeout, "CONNECTION_TIMEOUT", err.Error())
	default:
		h.logger.Error().Err(err).Msg("unhandled error di pppoe handler")
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "terjadi kesalahan internal")
	}
}

// validationError menangani error validasi dari go-playground/validator.
func (h *PPPoEHandler) validationError(c *fiber.Ctx, err error) error {
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
