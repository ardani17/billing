package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
	"github.com/ispboss/ispboss/services/billing-api/internal/repository"
	"github.com/rs/zerolog"
)

type InventoryUsecase struct {
	repo        *repository.InventoryRepo
	expenseRepo domain.ExpenseRepository
	auditRepo   domain.AuditLogRepository
	logger      zerolog.Logger
}

func NewInventoryUsecase(repo *repository.InventoryRepo, expenseRepo domain.ExpenseRepository, auditRepo domain.AuditLogRepository, logger zerolog.Logger) *InventoryUsecase {
	return &InventoryUsecase{repo: repo, expenseRepo: expenseRepo, auditRepo: auditRepo, logger: logger}
}

func (uc *InventoryUsecase) ListItems(ctx context.Context, tenantID string) ([]*domain.InventoryItem, error) {
	return uc.repo.ListItems(ctx, tenantID)
}

func (uc *InventoryUsecase) CreateItem(ctx context.Context, tenantID string, req domain.CreateInventoryItemRequest, actor domain.ActorInfo) (*domain.InventoryItem, error) {
	item, err := uc.repo.CreateItem(ctx, &domain.InventoryItem{
		TenantID: tenantID, Name: req.Name, Category: req.Category, Unit: req.Unit,
		TrackSerial: req.TrackSerial, MinStock: req.MinStock, DefaultCost: req.DefaultCost,
	})
	if err != nil {
		return nil, err
	}
	uc.writeAuditLog(ctx, tenantID, item.ID, "inventory_item.created", actor, map[string]interface{}{"name": item.Name})
	return item, nil
}

func (uc *InventoryUsecase) UpdateItem(ctx context.Context, tenantID, id string, req domain.UpdateInventoryItemRequest, actor domain.ActorInfo) (*domain.InventoryItem, error) {
	item, err := uc.repo.GetItem(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if req.Name != "" {
		item.Name = req.Name
	}
	if req.Category != "" {
		item.Category = req.Category
	}
	if req.Unit != "" {
		item.Unit = req.Unit
	}
	if req.TrackSerial != nil {
		item.TrackSerial = *req.TrackSerial
	}
	if req.MinStock != nil {
		item.MinStock = *req.MinStock
	}
	if req.DefaultCost != nil {
		item.DefaultCost = *req.DefaultCost
	}
	if req.IsActive != nil {
		item.IsActive = *req.IsActive
	}
	updated, err := uc.repo.UpdateItem(ctx, item)
	if err != nil {
		return nil, err
	}
	uc.writeAuditLog(ctx, tenantID, updated.ID, "inventory_item.updated", actor, map[string]interface{}{"name": updated.Name})
	return updated, nil
}

func (uc *InventoryUsecase) DeleteItem(ctx context.Context, tenantID, id string, actor domain.ActorInfo) error {
	if err := uc.repo.DeleteItem(ctx, tenantID, id); err != nil {
		return err
	}
	uc.writeAuditLog(ctx, tenantID, id, "inventory_item.deleted", actor, nil)
	return nil
}

func (uc *InventoryUsecase) ListAssets(ctx context.Context, tenantID string) ([]*domain.InventoryAsset, error) {
	return uc.repo.ListAssets(ctx, tenantID)
}

func (uc *InventoryUsecase) CreateAsset(ctx context.Context, tenantID string, req domain.CreateInventoryAssetRequest, actor domain.ActorInfo) (*domain.InventoryAsset, error) {
	purchaseDate, err := parseOptionalDate(req.PurchaseDate)
	if err != nil {
		return nil, err
	}
	warrantyUntil, err := parseOptionalDate(req.WarrantyUntil)
	if err != nil {
		return nil, err
	}
	status := req.Status
	if status == "" {
		status = "in_stock"
	}
	locationType := req.LocationType
	if locationType == "" {
		locationType = "warehouse"
	}
	asset, err := uc.repo.CreateAsset(ctx, &domain.InventoryAsset{
		TenantID: tenantID, ItemID: req.ItemID, SerialNumber: req.SerialNumber,
		MacAddress: req.MacAddress, Status: status, LocationType: locationType,
		LocationID: req.LocationID, AssignedCustomerID: req.AssignedCustomerID,
		PurchaseCost: req.PurchaseCost, PurchaseDate: purchaseDate, WarrantyUntil: warrantyUntil,
	})
	if err != nil {
		return nil, err
	}
	uc.writeAuditLog(ctx, tenantID, asset.ID, "inventory_asset.created", actor, map[string]interface{}{"serial_number": asset.SerialNumber})
	if actor.ActorID != "" {
		_, _ = uc.createMovementForAsset(ctx, tenantID, asset, "purchase", 1, actor, "Aset serial masuk stok")
	}
	return asset, nil
}

func (uc *InventoryUsecase) UpdateAsset(ctx context.Context, tenantID, id string, req domain.UpdateInventoryAssetRequest, actor domain.ActorInfo) (*domain.InventoryAsset, error) {
	asset, err := uc.repo.GetAsset(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	purchaseDate, err := parseOptionalDate(req.PurchaseDate)
	if err != nil {
		return nil, err
	}
	warrantyUntil, err := parseOptionalDate(req.WarrantyUntil)
	if err != nil {
		return nil, err
	}
	if req.MacAddress != "" {
		asset.MacAddress = req.MacAddress
	}
	if req.Status != "" {
		asset.Status = req.Status
	}
	if req.LocationType != "" {
		asset.LocationType = req.LocationType
	}
	if req.LocationID != "" {
		asset.LocationID = req.LocationID
	}
	if req.AssignedCustomerID != "" {
		asset.AssignedCustomerID = req.AssignedCustomerID
	}
	if req.PurchaseCost != nil {
		asset.PurchaseCost = *req.PurchaseCost
	}
	if purchaseDate != nil {
		asset.PurchaseDate = purchaseDate
	}
	if warrantyUntil != nil {
		asset.WarrantyUntil = warrantyUntil
	}
	updated, err := uc.repo.UpdateAsset(ctx, asset)
	if err != nil {
		return nil, err
	}
	uc.writeAuditLog(ctx, tenantID, updated.ID, "inventory_asset.updated", actor, map[string]interface{}{"status": updated.Status})
	return updated, nil
}

func (uc *InventoryUsecase) AssignAsset(ctx context.Context, tenantID, id string, req domain.AssetActionRequest, actor domain.ActorInfo) (*domain.InventoryAsset, error) {
	if req.CustomerID == "" {
		return nil, fmt.Errorf("customer_id wajib diisi")
	}
	asset, err := uc.repo.UpdateAssetStatus(ctx, tenantID, id, "assigned", "customer", req.CustomerID, req.CustomerID)
	if err != nil {
		return nil, err
	}
	uc.writeAuditLog(ctx, tenantID, id, "inventory_asset.assigned", actor, map[string]interface{}{"customer_id": req.CustomerID})
	_, _ = uc.createMovementForAsset(ctx, tenantID, asset, "install", 1, actor, req.Notes)
	return asset, nil
}

func (uc *InventoryUsecase) ReturnAsset(ctx context.Context, tenantID, id string, req domain.AssetActionRequest, actor domain.ActorInfo) (*domain.InventoryAsset, error) {
	locationType := req.LocationType
	if locationType == "" {
		locationType = "warehouse"
	}
	asset, err := uc.repo.UpdateAssetStatus(ctx, tenantID, id, "in_stock", locationType, req.LocationID, "")
	if err != nil {
		return nil, err
	}
	uc.writeAuditLog(ctx, tenantID, id, "inventory_asset.returned", actor, map[string]interface{}{"location_type": locationType})
	_, _ = uc.createMovementForAsset(ctx, tenantID, asset, "return", 1, actor, req.Notes)
	return asset, nil
}

func (uc *InventoryUsecase) MarkAssetDamaged(ctx context.Context, tenantID, id string, req domain.AssetActionRequest, actor domain.ActorInfo) (*domain.InventoryAsset, error) {
	asset, err := uc.repo.UpdateAssetStatus(ctx, tenantID, id, "damaged", "damaged", req.LocationID, "")
	if err != nil {
		return nil, err
	}
	uc.writeAuditLog(ctx, tenantID, id, "inventory_asset.damaged", actor, nil)
	_, _ = uc.createMovementForAsset(ctx, tenantID, asset, "damaged", 1, actor, req.Notes)
	return asset, nil
}

func (uc *InventoryUsecase) MarkAssetLost(ctx context.Context, tenantID, id string, req domain.AssetActionRequest, actor domain.ActorInfo) (*domain.InventoryAsset, error) {
	asset, err := uc.repo.UpdateAssetStatus(ctx, tenantID, id, "lost", "lost", req.LocationID, "")
	if err != nil {
		return nil, err
	}
	uc.writeAuditLog(ctx, tenantID, id, "inventory_asset.lost", actor, nil)
	_, _ = uc.createMovementForAsset(ctx, tenantID, asset, "lost", 1, actor, req.Notes)
	return asset, nil
}

func (uc *InventoryUsecase) MarkAssetRMA(ctx context.Context, tenantID, id string, req domain.AssetActionRequest, actor domain.ActorInfo) (*domain.InventoryAsset, error) {
	asset, err := uc.repo.UpdateAssetStatus(ctx, tenantID, id, "rma", "rma", req.LocationID, "")
	if err != nil {
		return nil, err
	}
	uc.writeAuditLog(ctx, tenantID, id, "inventory_asset.rma", actor, nil)
	_, _ = uc.createMovementForAsset(ctx, tenantID, asset, "rma", 1, actor, req.Notes)
	return asset, nil
}

func (uc *InventoryUsecase) RetireAsset(ctx context.Context, tenantID, id string, req domain.AssetActionRequest, actor domain.ActorInfo) (*domain.InventoryAsset, error) {
	asset, err := uc.repo.UpdateAssetStatus(ctx, tenantID, id, "retired", "warehouse", req.LocationID, "")
	if err != nil {
		return nil, err
	}
	uc.writeAuditLog(ctx, tenantID, id, "inventory_asset.retired", actor, nil)
	_, _ = uc.createMovementForAsset(ctx, tenantID, asset, "retired", 1, actor, req.Notes)
	return asset, nil
}

func (uc *InventoryUsecase) ListMovements(ctx context.Context, tenantID string) ([]*domain.InventoryMovement, error) {
	return uc.repo.ListMovements(ctx, tenantID)
}

func (uc *InventoryUsecase) CreateMovement(ctx context.Context, tenantID string, req domain.CreateInventoryMovementRequest, actor domain.ActorInfo) (*domain.InventoryMovement, error) {
	if req.Quantity == 0 {
		return nil, fmt.Errorf("quantity tidak boleh 0")
	}
	item, err := uc.repo.GetItem(ctx, tenantID, req.ItemID)
	if err != nil {
		return nil, err
	}
	if item.TrackSerial {
		if req.AssetID == "" {
			return nil, domain.ErrInventorySerialRequired
		}
		if abs(req.Quantity) != 1 {
			return nil, fmt.Errorf("mutasi item serial wajib berjumlah 1 per aset")
		}
	}
	if isNegativeStockMovement(req.MovementType, req.Quantity) {
		current, err := uc.repo.CurrentStock(ctx, tenantID, req.ItemID)
		if err != nil {
			return nil, err
		}
		if current < abs(req.Quantity) {
			return nil, domain.ErrInventoryStockInsufficient
		}
	}

	movement := &domain.InventoryMovement{
		TenantID: tenantID, ItemID: req.ItemID, AssetID: req.AssetID,
		MovementType: req.MovementType, Quantity: req.Quantity,
		FromLocationType: req.FromLocationType, FromLocationID: req.FromLocationID,
		ToLocationType: req.ToLocationType, ToLocationID: req.ToLocationID,
		CustomerID: req.CustomerID, UnitCost: req.UnitCost, Notes: req.Notes,
		CreatedByID: actor.ActorID,
	}
	if movement.CreatedByID == "" {
		return nil, fmt.Errorf("aktor tidak teridentifikasi")
	}

	if req.CreateExpense && req.UnitCost > 0 && req.ExpenseCategoryID != "" {
		expense, err := uc.expenseRepo.Create(ctx, &domain.Expense{
			TenantID: tenantID, CategoryID: req.ExpenseCategoryID,
			Amount:      int64(abs(req.Quantity)) * req.UnitCost,
			Description: fmt.Sprintf("Pembelian inventaris: %s", req.Notes),
			ExpenseDate: time.Now(), CreatedByID: actor.ActorID,
		})
		if err != nil {
			return nil, err
		}
		movement.ExpenseID = expense.ID
	}

	created, err := uc.repo.CreateMovement(ctx, movement)
	if err != nil {
		return nil, err
	}
	uc.writeAuditLog(ctx, tenantID, created.ID, "inventory_movement.created", actor, map[string]interface{}{
		"item_id":       created.ItemID,
		"movement_type": created.MovementType,
		"quantity":      created.Quantity,
	})
	if req.AssetID != "" {
		uc.applyMovementToAsset(ctx, tenantID, created, actor)
	}
	return created, nil
}

func (uc *InventoryUsecase) StockSummary(ctx context.Context, tenantID string) ([]*domain.InventoryStockItem, error) {
	return uc.repo.StockSummary(ctx, tenantID)
}

func parseOptionalDate(value string) (*time.Time, error) {
	if value == "" {
		return nil, nil
	}
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		return nil, fmt.Errorf("format tanggal tidak valid")
	}
	return &parsed, nil
}

func isNegativeStockMovement(kind string, quantity int) bool {
	if quantity < 0 {
		return true
	}
	return kind == "install" || kind == "damaged" || kind == "lost" || kind == "rma" || kind == "retired"
}

func abs(value int) int {
	if value < 0 {
		return -value
	}
	return value
}

func (uc *InventoryUsecase) createMovementForAsset(ctx context.Context, tenantID string, asset *domain.InventoryAsset, movementType string, quantity int, actor domain.ActorInfo, notes string) (*domain.InventoryMovement, error) {
	if actor.ActorID == "" {
		return nil, nil
	}
	movement := &domain.InventoryMovement{
		TenantID:     tenantID,
		ItemID:       asset.ItemID,
		AssetID:      asset.ID,
		MovementType: movementType,
		Quantity:     quantity,
		CustomerID:   asset.AssignedCustomerID,
		UnitCost:     asset.PurchaseCost,
		Notes:        notes,
		CreatedByID:  actor.ActorID,
	}
	created, err := uc.repo.CreateMovement(ctx, movement)
	if err != nil {
		uc.logger.Error().Err(err).Str("asset_id", asset.ID).Str("movement_type", movementType).Msg("gagal membuat mutasi aset serial")
		return nil, err
	}
	uc.writeAuditLog(ctx, tenantID, created.ID, "inventory_movement.created", actor, map[string]interface{}{
		"item_id":       created.ItemID,
		"asset_id":      asset.ID,
		"movement_type": created.MovementType,
		"quantity":      created.Quantity,
	})
	return created, nil
}

func (uc *InventoryUsecase) applyMovementToAsset(ctx context.Context, tenantID string, movement *domain.InventoryMovement, actor domain.ActorInfo) {
	var status, locationType, locationID, customerID string
	switch movement.MovementType {
	case "install":
		status, locationType, locationID, customerID = "assigned", "customer", movement.CustomerID, movement.CustomerID
	case "return":
		status, locationType, locationID, customerID = "in_stock", defaultString(movement.ToLocationType, "warehouse"), movement.ToLocationID, ""
	case "damaged":
		status, locationType, locationID, customerID = "damaged", "damaged", movement.ToLocationID, ""
	case "lost":
		status, locationType, locationID, customerID = "lost", "lost", movement.ToLocationID, ""
	case "rma":
		status, locationType, locationID, customerID = "rma", "rma", movement.ToLocationID, ""
	case "retired":
		status, locationType, locationID, customerID = "retired", "warehouse", movement.ToLocationID, ""
	default:
		return
	}
	asset, err := uc.repo.UpdateAssetStatus(ctx, tenantID, movement.AssetID, status, locationType, locationID, customerID)
	if err != nil {
		uc.logger.Error().Err(err).Str("asset_id", movement.AssetID).Msg("gagal menerapkan mutasi ke aset")
		return
	}
	uc.writeAuditLog(ctx, tenantID, asset.ID, "inventory_asset.status_changed", actor, map[string]interface{}{"status": status})
}

func defaultString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func (uc *InventoryUsecase) writeAuditLog(ctx context.Context, tenantID, entityID, action string, actor domain.ActorInfo, changes map[string]interface{}) {
	if uc.auditRepo == nil {
		return
	}
	log := &domain.AuditLog{
		TenantID:   tenantID,
		EntityType: "inventory",
		EntityID:   entityID,
		Action:     action,
		ActorID:    actor.ActorID,
		ActorName:  actor.ActorName,
		Changes:    changes,
	}
	if err := uc.auditRepo.Create(ctx, log); err != nil {
		uc.logger.Error().Err(err).Str("entity_id", entityID).Str("action", action).Msg("gagal menulis audit log inventaris")
	}
}
