// trash_handler.go menangani HTTP request untuk manajemen trash (soft-delete).
// Termasuk: list node yang dihapus dan restore node.
package handler

import (
	"errors"

	"github.com/gofiber/fiber/v2"

	"github.com/ispboss/ispboss/pkg/tenant"
	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// TrashHandler menangani HTTP request untuk trash management.
type TrashHandler struct {
	manager domain.MapNodeManager
}

// NewTrashHandler membuat instance baru TrashHandler.
func NewTrashHandler(manager domain.MapNodeManager) *TrashHandler {
	return &TrashHandler{manager: manager}
}

// ListTrashed menangani GET /trash.
// Mengambil daftar node yang sudah di-soft-delete untuk tenant saat ini.
func (h *TrashHandler) ListTrashed(c *fiber.Ctx) error {
	tenantID := tenant.FromContext(c.UserContext())
	if tenantID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant tidak teridentifikasi")
	}

	nodes, err := h.manager.ListTrashed(c.UserContext(), tenantID)
	if err != nil {
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "terjadi kesalahan internal")
	}

	return domain.SuccessResponse(c, fiber.StatusOK, nodes)
}

// RestoreNode menangani POST /trash/:id/restore.
// Mengembalikan node dari trash (clear deleted_at) dan catat riwayat.
func (h *TrashHandler) RestoreNode(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "node ID wajib diisi")
	}

	tenantID := tenant.FromContext(c.UserContext())
	if tenantID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant tidak teridentifikasi")
	}

	if err := h.manager.RestoreNode(c.UserContext(), id, tenantID); err != nil {
		return h.mapError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, fiber.Map{"message": "node berhasil di-restore"})
}

// mapError memetakan domain error trash ke HTTP error response.
func (h *TrashHandler) mapError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, domain.ErrMapNodeNotFound):
		return domain.ErrorResponse(c, fiber.StatusNotFound, "MAP_NODE_NOT_FOUND", err.Error())
	case errors.Is(err, domain.ErrMapNodeDuplicate):
		return domain.ErrorResponse(c, fiber.StatusConflict, "MAP_NODE_DUPLICATE", err.Error())
	default:
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "terjadi kesalahan internal")
	}
}
