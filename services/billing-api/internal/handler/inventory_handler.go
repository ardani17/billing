package handler

import (
	"context"
	"errors"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
	"github.com/ispboss/ispboss/services/billing-api/internal/usecase"
	"github.com/rs/zerolog"
)

type InventoryHandler struct {
	usecase  *usecase.InventoryUsecase
	validate *validator.Validate
	logger   zerolog.Logger
}

func NewInventoryHandler(usecase *usecase.InventoryUsecase, logger zerolog.Logger) *InventoryHandler {
	return &InventoryHandler{usecase: usecase, validate: validator.New(), logger: logger}
}

func (h *InventoryHandler) ListItems(c *fiber.Ctx) error {
	tenantID, ok := tenantIDFromCtx(c)
	if !ok {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant tidak teridentifikasi")
	}
	items, err := h.usecase.ListItems(c.Context(), tenantID)
	if err != nil {
		h.logger.Error().Err(err).Msg("gagal mengambil item inventaris")
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal mengambil item inventaris")
	}
	if !canViewInventoryCost(c) {
		for _, item := range items {
			item.DefaultCost = 0
		}
	}
	return domain.SuccessResponse(c, fiber.StatusOK, items)
}

func (h *InventoryHandler) CreateItem(c *fiber.Ctx) error {
	tenantID, ok := tenantIDFromCtx(c)
	if !ok {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant tidak teridentifikasi")
	}
	var req domain.CreateInventoryItemRequest
	if err := parseAndValidate(c, h.validate, &req); err != nil {
		return err
	}
	item, err := h.usecase.CreateItem(c.Context(), tenantID, req, actorFromCtx(c))
	if err != nil {
		return h.mapError(c, err)
	}
	return domain.SuccessResponse(c, fiber.StatusCreated, item)
}

func (h *InventoryHandler) UpdateItem(c *fiber.Ctx) error {
	tenantID, ok := tenantIDFromCtx(c)
	if !ok {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant tidak teridentifikasi")
	}
	var req domain.UpdateInventoryItemRequest
	if err := parseAndValidate(c, h.validate, &req); err != nil {
		return err
	}
	item, err := h.usecase.UpdateItem(c.Context(), tenantID, c.Params("id"), req, actorFromCtx(c))
	if err != nil {
		return h.mapError(c, err)
	}
	return domain.SuccessResponse(c, fiber.StatusOK, item)
}

func (h *InventoryHandler) DeleteItem(c *fiber.Ctx) error {
	tenantID, ok := tenantIDFromCtx(c)
	if !ok {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant tidak teridentifikasi")
	}
	if err := h.usecase.DeleteItem(c.Context(), tenantID, c.Params("id"), actorFromCtx(c)); err != nil {
		return h.mapError(c, err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *InventoryHandler) ListAssets(c *fiber.Ctx) error {
	tenantID, ok := tenantIDFromCtx(c)
	if !ok {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant tidak teridentifikasi")
	}
	assets, err := h.usecase.ListAssets(c.Context(), tenantID)
	if err != nil {
		h.logger.Error().Err(err).Msg("gagal mengambil aset inventaris")
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal mengambil aset inventaris")
	}
	if !canViewInventoryCost(c) {
		for _, asset := range assets {
			asset.PurchaseCost = 0
		}
	}
	return domain.SuccessResponse(c, fiber.StatusOK, assets)
}

func (h *InventoryHandler) CreateAsset(c *fiber.Ctx) error {
	tenantID, ok := tenantIDFromCtx(c)
	if !ok {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant tidak teridentifikasi")
	}
	var req domain.CreateInventoryAssetRequest
	if err := parseAndValidate(c, h.validate, &req); err != nil {
		return err
	}
	asset, err := h.usecase.CreateAsset(c.Context(), tenantID, req, actorFromCtx(c))
	if err != nil {
		return h.mapError(c, err)
	}
	return domain.SuccessResponse(c, fiber.StatusCreated, asset)
}

func (h *InventoryHandler) UpdateAsset(c *fiber.Ctx) error {
	tenantID, ok := tenantIDFromCtx(c)
	if !ok {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant tidak teridentifikasi")
	}
	var req domain.UpdateInventoryAssetRequest
	if err := parseAndValidate(c, h.validate, &req); err != nil {
		return err
	}
	asset, err := h.usecase.UpdateAsset(c.Context(), tenantID, c.Params("id"), req, actorFromCtx(c))
	if err != nil {
		return h.mapError(c, err)
	}
	return domain.SuccessResponse(c, fiber.StatusOK, asset)
}

func (h *InventoryHandler) AssignAsset(c *fiber.Ctx) error {
	return h.assetAction(c, h.usecase.AssignAsset)
}

func (h *InventoryHandler) ReturnAsset(c *fiber.Ctx) error {
	return h.assetAction(c, h.usecase.ReturnAsset)
}

func (h *InventoryHandler) MarkDamaged(c *fiber.Ctx) error {
	return h.assetAction(c, h.usecase.MarkAssetDamaged)
}

func (h *InventoryHandler) MarkLost(c *fiber.Ctx) error {
	return h.assetAction(c, h.usecase.MarkAssetLost)
}

func (h *InventoryHandler) MarkRMA(c *fiber.Ctx) error {
	return h.assetAction(c, h.usecase.MarkAssetRMA)
}

func (h *InventoryHandler) RetireAsset(c *fiber.Ctx) error {
	return h.assetAction(c, h.usecase.RetireAsset)
}

func (h *InventoryHandler) assetAction(c *fiber.Ctx, fn func(context.Context, string, string, domain.AssetActionRequest, domain.ActorInfo) (*domain.InventoryAsset, error)) error {
	tenantID, ok := tenantIDFromCtx(c)
	if !ok {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant tidak teridentifikasi")
	}
	var req domain.AssetActionRequest
	if err := c.BodyParser(&req); err != nil {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "request body tidak valid")
	}
	asset, err := fn(c.Context(), tenantID, c.Params("id"), req, actorFromCtx(c))
	if err != nil {
		return h.mapError(c, err)
	}
	return domain.SuccessResponse(c, fiber.StatusOK, asset)
}

func (h *InventoryHandler) ListMovements(c *fiber.Ctx) error {
	tenantID, ok := tenantIDFromCtx(c)
	if !ok {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant tidak teridentifikasi")
	}
	movements, err := h.usecase.ListMovements(c.Context(), tenantID)
	if err != nil {
		h.logger.Error().Err(err).Msg("gagal mengambil mutasi inventaris")
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal mengambil mutasi inventaris")
	}
	if !canViewInventoryCost(c) {
		for _, movement := range movements {
			movement.UnitCost = 0
		}
	}
	return domain.SuccessResponse(c, fiber.StatusOK, movements)
}

func (h *InventoryHandler) CreateMovement(c *fiber.Ctx) error {
	tenantID, ok := tenantIDFromCtx(c)
	if !ok {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant tidak teridentifikasi")
	}
	var req domain.CreateInventoryMovementRequest
	if err := parseAndValidate(c, h.validate, &req); err != nil {
		return err
	}
	movement, err := h.usecase.CreateMovement(c.Context(), tenantID, req, actorFromCtx(c))
	if err != nil {
		return h.mapError(c, err)
	}
	return domain.SuccessResponse(c, fiber.StatusCreated, movement)
}

func (h *InventoryHandler) Stock(c *fiber.Ctx) error {
	tenantID, ok := tenantIDFromCtx(c)
	if !ok {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant tidak teridentifikasi")
	}
	stock, err := h.usecase.StockSummary(c.Context(), tenantID)
	if err != nil {
		h.logger.Error().Err(err).Msg("gagal mengambil stok inventaris")
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal mengambil stok inventaris")
	}
	return domain.SuccessResponse(c, fiber.StatusOK, stock)
}

func (h *InventoryHandler) mapError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, domain.ErrInventoryItemNotFound):
		return domain.ErrorResponse(c, fiber.StatusNotFound, "INVENTORY_ITEM_NOT_FOUND", err.Error())
	case errors.Is(err, domain.ErrInventoryAssetNotFound):
		return domain.ErrorResponse(c, fiber.StatusNotFound, "INVENTORY_ASSET_NOT_FOUND", err.Error())
	case errors.Is(err, domain.ErrInventorySerialDuplicate):
		return domain.ErrorResponse(c, fiber.StatusConflict, "INVENTORY_SERIAL_DUPLICATE", err.Error())
	case errors.Is(err, domain.ErrInventoryStockInsufficient):
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "INVENTORY_STOCK_INSUFFICIENT", err.Error())
	case errors.Is(err, domain.ErrInventorySerialRequired):
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "INVENTORY_SERIAL_REQUIRED", err.Error())
	default:
		h.logger.Error().Err(err).Msg("internal error pada inventory handler")
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "terjadi kesalahan internal")
	}
}

func canViewInventoryCost(c *fiber.Ctx) bool {
	role, _ := c.Locals("role").(string)
	return role == string(domain.RoleSuperAdmin) || role == string(domain.RoleTenantAdmin) || role == string(domain.RoleKasir)
}
