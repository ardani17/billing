package handler

import (
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/ispboss/ispboss/services/network-service/internal/domain"
	"github.com/ispboss/ispboss/services/network-service/internal/usecase"
	"github.com/rs/zerolog"
)

type BackupHandler struct {
	manager usecase.BackupManager
	logger  zerolog.Logger
}

func NewBackupHandler(manager usecase.BackupManager, logger zerolog.Logger) *BackupHandler {
	return &BackupHandler{manager: manager, logger: logger}
}

func (h *BackupHandler) Create(c *fiber.Ctx) error {
	if !canUseMikroTikTerminal(c) {
		return domain.ErrorResponse(c, fiber.StatusForbidden, "FORBIDDEN", "role tidak diizinkan membuat backup MikroTik")
	}
	ctx := usecase.WithMikroTikAuditActor(c.UserContext(), fiberLocalsString(c, "user_id"), c.IP())
	item, err := h.manager.CreateBackup(ctx, c.Params("id"))
	if err != nil {
		return h.mapError(c, err)
	}
	item.Content = ""
	return domain.SuccessResponse(c, fiber.StatusCreated, item)
}

func (h *BackupHandler) List(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	pageSize, _ := strconv.Atoi(c.Query("page_size", "20"))
	result, err := h.manager.ListBackups(c.UserContext(), domain.RouterBackupListParams{
		RouterID: c.Params("id"), Page: page, PageSize: pageSize,
	})
	if err != nil {
		return h.mapError(c, err)
	}
	return domain.SuccessResponse(c, fiber.StatusOK, result)
}

func (h *BackupHandler) Download(c *fiber.Ctx) error {
	item, err := h.manager.GetBackup(c.UserContext(), c.Params("backup_id"))
	if err != nil {
		return h.mapError(c, err)
	}
	c.Set(fiber.HeaderContentType, "text/plain; charset=utf-8")
	c.Set(fiber.HeaderContentDisposition, `attachment; filename="`+item.FileName+`"`)
	return c.SendString(item.Content)
}

func (h *BackupHandler) Delete(c *fiber.Ctx) error {
	if !canUseMikroTikTerminal(c) {
		return domain.ErrorResponse(c, fiber.StatusForbidden, "FORBIDDEN", "role tidak diizinkan menghapus backup MikroTik")
	}
	ctx := usecase.WithMikroTikAuditActor(c.UserContext(), fiberLocalsString(c, "user_id"), c.IP())
	if err := h.manager.DeleteBackup(ctx, c.Params("backup_id")); err != nil {
		return h.mapError(c, err)
	}
	return domain.SuccessResponse(c, fiber.StatusOK, fiber.Map{"message": "backup berhasil dihapus"})
}

func (h *BackupHandler) Firmware(c *fiber.Ctx) error {
	info, err := h.manager.GetFirmware(c.UserContext(), c.Params("id"))
	if err != nil {
		return h.mapError(c, err)
	}
	return domain.SuccessResponse(c, fiber.StatusOK, info)
}

func (h *BackupHandler) mapError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, domain.ErrRouterNotFound):
		return domain.ErrorResponse(c, fiber.StatusNotFound, "ROUTER_NOT_FOUND", err.Error())
	case errors.Is(err, domain.ErrRouterBackupNotFound):
		return domain.ErrorResponse(c, fiber.StatusNotFound, "BACKUP_NOT_FOUND", err.Error())
	case errors.Is(err, domain.ErrConnectionFailed):
		return domain.ErrorResponse(c, fiber.StatusBadGateway, "CONNECTION_FAILED", err.Error())
	case errors.Is(err, domain.ErrConnectionTimeout):
		return domain.ErrorResponse(c, fiber.StatusGatewayTimeout, "CONNECTION_TIMEOUT", err.Error())
	case errors.Is(err, domain.ErrRouterPermissionDenied):
		return domain.ErrorResponse(c, fiber.StatusForbidden, "ROUTER_PERMISSION_DENIED", err.Error())
	case errors.Is(err, domain.ErrDecryptionFailed):
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "CREDENTIAL_ERROR", "gagal membaca credential router")
	default:
		h.logger.Error().Err(err).Msg("unhandled error di backup handler")
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "terjadi kesalahan internal")
	}
}
