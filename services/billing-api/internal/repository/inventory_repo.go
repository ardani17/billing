package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type InventoryRepo struct {
	pool *pgxpool.Pool
}

func NewInventoryRepo(pool *pgxpool.Pool) *InventoryRepo {
	return &InventoryRepo{pool: pool}
}

func (r *InventoryRepo) ListItems(ctx context.Context, tenantID string) ([]*domain.InventoryItem, error) {
	rows, err := r.pool.Query(ctx, `SELECT
		i.id::text, i.tenant_id::text, i.name, i.category, i.unit, i.track_serial,
		i.min_stock, i.default_cost, i.is_active,
		COALESCE(SUM(CASE
			WHEN m.movement_type IN ('purchase','return') THEN m.quantity
			WHEN m.movement_type IN ('install','damaged','lost','rma','retired') THEN -ABS(m.quantity)
			WHEN m.movement_type = 'adjustment' THEN m.quantity
			ELSE 0
		END), 0)::int AS stock,
		i.created_at, i.updated_at
		FROM inventory_items i
		LEFT JOIN inventory_movements m ON m.item_id = i.id AND m.tenant_id = i.tenant_id
		WHERE i.tenant_id = $1 AND i.deleted_at IS NULL
		GROUP BY i.id
		ORDER BY i.name ASC`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("repository: gagal mengambil item inventaris: %w", err)
	}
	defer rows.Close()

	items := []*domain.InventoryItem{}
	for rows.Next() {
		item := &domain.InventoryItem{}
		if err := rows.Scan(
			&item.ID, &item.TenantID, &item.Name, &item.Category, &item.Unit, &item.TrackSerial,
			&item.MinStock, &item.DefaultCost, &item.IsActive, &item.Stock, &item.CreatedAt, &item.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("repository: gagal scan item inventaris: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *InventoryRepo) CreateItem(ctx context.Context, item *domain.InventoryItem) (*domain.InventoryItem, error) {
	row := r.pool.QueryRow(ctx, `INSERT INTO inventory_items
		(tenant_id, name, category, unit, track_serial, min_stock, default_cost)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
		RETURNING id::text, tenant_id::text, name, category, unit, track_serial,
			min_stock, default_cost, is_active, 0::int, created_at, updated_at`,
		item.TenantID, item.Name, item.Category, item.Unit, item.TrackSerial, item.MinStock, item.DefaultCost)
	return scanInventoryItem(row)
}

func (r *InventoryRepo) UpdateItem(ctx context.Context, item *domain.InventoryItem) (*domain.InventoryItem, error) {
	row := r.pool.QueryRow(ctx, `UPDATE inventory_items SET
		name = $3, category = $4, unit = $5, track_serial = $6,
		min_stock = $7, default_cost = $8, is_active = $9, updated_at = NOW()
		WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL
		RETURNING id::text, tenant_id::text, name, category, unit, track_serial,
			min_stock, default_cost, is_active, 0::int, created_at, updated_at`,
		item.ID, item.TenantID, item.Name, item.Category, item.Unit, item.TrackSerial,
		item.MinStock, item.DefaultCost, item.IsActive)
	updated, err := scanInventoryItem(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrInventoryItemNotFound
	}
	return updated, err
}

func (r *InventoryRepo) GetItem(ctx context.Context, tenantID, id string) (*domain.InventoryItem, error) {
	row := r.pool.QueryRow(ctx, `SELECT
		id::text, tenant_id::text, name, category, unit, track_serial,
		min_stock, default_cost, is_active, 0::int, created_at, updated_at
		FROM inventory_items
		WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`, tenantID, id)
	item, err := scanInventoryItem(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrInventoryItemNotFound
	}
	return item, err
}

func (r *InventoryRepo) DeleteItem(ctx context.Context, tenantID, id string) error {
	tag, err := r.pool.Exec(ctx, `UPDATE inventory_items
		SET deleted_at = NOW(), is_active = false, updated_at = NOW()
		WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`, tenantID, id)
	if err != nil {
		return fmt.Errorf("repository: gagal menghapus item inventaris: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrInventoryItemNotFound
	}
	return nil
}

func scanInventoryItem(row pgx.Row) (*domain.InventoryItem, error) {
	item := &domain.InventoryItem{}
	if err := row.Scan(
		&item.ID, &item.TenantID, &item.Name, &item.Category, &item.Unit, &item.TrackSerial,
		&item.MinStock, &item.DefaultCost, &item.IsActive, &item.Stock, &item.CreatedAt, &item.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return item, nil
}

func (r *InventoryRepo) ListAssets(ctx context.Context, tenantID string) ([]*domain.InventoryAsset, error) {
	rows, err := r.pool.Query(ctx, `SELECT
		a.id::text, a.tenant_id::text, a.item_id::text, i.name,
		a.serial_number, COALESCE(a.mac_address,''), a.status, a.location_type,
		COALESCE(a.location_id,''), COALESCE(a.assigned_customer_id::text,''),
		COALESCE(c.name,''), a.purchase_cost, a.purchase_date, a.warranty_until,
		a.created_at, a.updated_at
		FROM inventory_assets a
		JOIN inventory_items i ON i.id = a.item_id
		LEFT JOIN customers c ON c.id = a.assigned_customer_id
		WHERE a.tenant_id = $1 AND a.deleted_at IS NULL
		ORDER BY a.updated_at DESC`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("repository: gagal mengambil aset inventaris: %w", err)
	}
	defer rows.Close()

	assets := []*domain.InventoryAsset{}
	for rows.Next() {
		asset, err := scanInventoryAsset(rows)
		if err != nil {
			return nil, err
		}
		assets = append(assets, asset)
	}
	return assets, rows.Err()
}

func (r *InventoryRepo) CreateAsset(ctx context.Context, asset *domain.InventoryAsset) (*domain.InventoryAsset, error) {
	row := r.pool.QueryRow(ctx, `INSERT INTO inventory_assets
		(tenant_id, item_id, serial_number, mac_address, status, location_type, location_id,
		 assigned_customer_id, purchase_cost, purchase_date, warranty_until)
		VALUES ($1,$2,$3,$4,$5,$6,$7,NULLIF($8,'')::uuid,$9,$10,$11)
		RETURNING id::text`, asset.TenantID, asset.ItemID, asset.SerialNumber, asset.MacAddress,
		defaultString(asset.Status, "in_stock"), defaultString(asset.LocationType, "warehouse"),
		asset.LocationID, asset.AssignedCustomerID, asset.PurchaseCost, asset.PurchaseDate, asset.WarrantyUntil)
	var id string
	if err := row.Scan(&id); err != nil {
		return nil, mapInventoryError(err)
	}
	return r.GetAsset(ctx, asset.TenantID, id)
}

func (r *InventoryRepo) GetAsset(ctx context.Context, tenantID, id string) (*domain.InventoryAsset, error) {
	row := r.pool.QueryRow(ctx, `SELECT
		a.id::text, a.tenant_id::text, a.item_id::text, i.name,
		a.serial_number, COALESCE(a.mac_address,''), a.status, a.location_type,
		COALESCE(a.location_id,''), COALESCE(a.assigned_customer_id::text,''),
		COALESCE(c.name,''), a.purchase_cost, a.purchase_date, a.warranty_until,
		a.created_at, a.updated_at
		FROM inventory_assets a
		JOIN inventory_items i ON i.id = a.item_id
		LEFT JOIN customers c ON c.id = a.assigned_customer_id
		WHERE a.tenant_id = $1 AND a.id = $2 AND a.deleted_at IS NULL`, tenantID, id)
	asset, err := scanInventoryAsset(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrInventoryAssetNotFound
	}
	return asset, err
}

func (r *InventoryRepo) UpdateAsset(ctx context.Context, asset *domain.InventoryAsset) (*domain.InventoryAsset, error) {
	tag, err := r.pool.Exec(ctx, `UPDATE inventory_assets SET
		mac_address = $3, status = $4, location_type = $5, location_id = $6,
		assigned_customer_id = NULLIF($7,'')::uuid, purchase_cost = $8,
		purchase_date = $9, warranty_until = $10, updated_at = NOW()
		WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`,
		asset.TenantID, asset.ID, asset.MacAddress, asset.Status, asset.LocationType,
		asset.LocationID, asset.AssignedCustomerID, asset.PurchaseCost, asset.PurchaseDate, asset.WarrantyUntil)
	if err != nil {
		return nil, mapInventoryError(err)
	}
	if tag.RowsAffected() == 0 {
		return nil, domain.ErrInventoryAssetNotFound
	}
	return r.GetAsset(ctx, asset.TenantID, asset.ID)
}

func (r *InventoryRepo) UpdateAssetStatus(ctx context.Context, tenantID, id, status, locationType, locationID, customerID string) (*domain.InventoryAsset, error) {
	tag, err := r.pool.Exec(ctx, `UPDATE inventory_assets SET
		status = $3, location_type = $4, location_id = $5,
		assigned_customer_id = NULLIF($6,'')::uuid, updated_at = NOW()
		WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`,
		tenantID, id, status, locationType, locationID, customerID)
	if err != nil {
		return nil, fmt.Errorf("repository: gagal update status aset: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, domain.ErrInventoryAssetNotFound
	}
	return r.GetAsset(ctx, tenantID, id)
}

func scanInventoryAsset(row pgx.Row) (*domain.InventoryAsset, error) {
	var purchaseDate, warrantyUntil *time.Time
	asset := &domain.InventoryAsset{}
	if err := row.Scan(
		&asset.ID, &asset.TenantID, &asset.ItemID, &asset.ItemName,
		&asset.SerialNumber, &asset.MacAddress, &asset.Status, &asset.LocationType,
		&asset.LocationID, &asset.AssignedCustomerID, &asset.AssignedCustomer,
		&asset.PurchaseCost, &purchaseDate, &warrantyUntil, &asset.CreatedAt, &asset.UpdatedAt,
	); err != nil {
		return nil, err
	}
	asset.PurchaseDate = purchaseDate
	asset.WarrantyUntil = warrantyUntil
	return asset, nil
}

func (r *InventoryRepo) CurrentStock(ctx context.Context, tenantID, itemID string) (int, error) {
	var stock int
	err := r.pool.QueryRow(ctx, `SELECT COALESCE(SUM(CASE
		WHEN movement_type IN ('purchase','return') THEN quantity
		WHEN movement_type IN ('install','damaged','lost','rma','retired') THEN -ABS(quantity)
		WHEN movement_type = 'adjustment' THEN quantity
		ELSE 0 END), 0)::int
		FROM inventory_movements
		WHERE tenant_id = $1 AND item_id = $2`, tenantID, itemID).Scan(&stock)
	return stock, err
}

func (r *InventoryRepo) CreateMovement(ctx context.Context, movement *domain.InventoryMovement) (*domain.InventoryMovement, error) {
	row := r.pool.QueryRow(ctx, `INSERT INTO inventory_movements
		(tenant_id, item_id, asset_id, movement_type, quantity, from_location_type,
		 from_location_id, to_location_type, to_location_id, customer_id, expense_id,
		 unit_cost, notes, created_by_id)
		VALUES ($1,$2,NULLIF($3,'')::uuid,$4,$5,$6,$7,$8,$9,NULLIF($10,'')::uuid,
		 NULLIF($11,'')::uuid,$12,$13,$14)
		RETURNING id::text`, movement.TenantID, movement.ItemID, movement.AssetID,
		movement.MovementType, movement.Quantity, movement.FromLocationType,
		movement.FromLocationID, movement.ToLocationType, movement.ToLocationID,
		movement.CustomerID, movement.ExpenseID, movement.UnitCost, movement.Notes, movement.CreatedByID)
	var id string
	if err := row.Scan(&id); err != nil {
		return nil, fmt.Errorf("repository: gagal membuat mutasi inventaris: %w", err)
	}
	return r.GetMovement(ctx, movement.TenantID, id)
}

func (r *InventoryRepo) GetMovement(ctx context.Context, tenantID, id string) (*domain.InventoryMovement, error) {
	row := r.pool.QueryRow(ctx, baseMovementQuery()+` WHERE m.tenant_id = $1 AND m.id = $2`, tenantID, id)
	return scanMovement(row)
}

func (r *InventoryRepo) ListMovements(ctx context.Context, tenantID string) ([]*domain.InventoryMovement, error) {
	rows, err := r.pool.Query(ctx, baseMovementQuery()+` WHERE m.tenant_id = $1 ORDER BY m.created_at DESC LIMIT 100`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("repository: gagal mengambil mutasi inventaris: %w", err)
	}
	defer rows.Close()

	movements := []*domain.InventoryMovement{}
	for rows.Next() {
		movement, err := scanMovement(rows)
		if err != nil {
			return nil, err
		}
		movements = append(movements, movement)
	}
	return movements, rows.Err()
}

func (r *InventoryRepo) StockSummary(ctx context.Context, tenantID string) ([]*domain.InventoryStockItem, error) {
	rows, err := r.pool.Query(ctx, `SELECT
		i.id::text, i.name, i.category, i.unit, i.track_serial, i.min_stock,
		COALESCE(SUM(CASE
			WHEN m.movement_type IN ('purchase','return') THEN m.quantity
			WHEN m.movement_type IN ('install','damaged','lost','rma','retired') THEN -ABS(m.quantity)
			WHEN m.movement_type = 'adjustment' THEN m.quantity
			ELSE 0 END), 0)::int AS stock
		FROM inventory_items i
		LEFT JOIN inventory_movements m ON m.item_id = i.id AND m.tenant_id = i.tenant_id
		WHERE i.tenant_id = $1 AND i.deleted_at IS NULL
		GROUP BY i.id
		ORDER BY i.name ASC`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("repository: gagal mengambil stok inventaris: %w", err)
	}
	defer rows.Close()

	items := []*domain.InventoryStockItem{}
	for rows.Next() {
		item := &domain.InventoryStockItem{}
		if err := rows.Scan(&item.ItemID, &item.ItemName, &item.Category, &item.Unit, &item.TrackSerial, &item.MinStock, &item.Stock); err != nil {
			return nil, err
		}
		if item.Stock <= item.MinStock {
			item.Status = "rendah"
		} else {
			item.Status = "aman"
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func baseMovementQuery() string {
	return `SELECT m.id::text, m.tenant_id::text, m.item_id::text, i.name,
		COALESCE(m.asset_id::text,''), m.movement_type, m.quantity,
		COALESCE(m.from_location_type,''), COALESCE(m.from_location_id,''),
		COALESCE(m.to_location_type,''), COALESCE(m.to_location_id,''),
		COALESCE(m.customer_id::text,''), COALESCE(c.name,''),
		COALESCE(m.expense_id::text,''), m.unit_cost, COALESCE(m.notes,''),
		m.created_by_id::text, m.created_at
		FROM inventory_movements m
		JOIN inventory_items i ON i.id = m.item_id
		LEFT JOIN customers c ON c.id = m.customer_id`
}

func scanMovement(row pgx.Row) (*domain.InventoryMovement, error) {
	movement := &domain.InventoryMovement{}
	err := row.Scan(
		&movement.ID, &movement.TenantID, &movement.ItemID, &movement.ItemName,
		&movement.AssetID, &movement.MovementType, &movement.Quantity,
		&movement.FromLocationType, &movement.FromLocationID,
		&movement.ToLocationType, &movement.ToLocationID,
		&movement.CustomerID, &movement.CustomerName, &movement.ExpenseID,
		&movement.UnitCost, &movement.Notes, &movement.CreatedByID, &movement.CreatedAt,
	)
	return movement, err
}

func defaultString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func mapInventoryError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrInventoryItemNotFound
	}
	if pgErr := err.Error(); pgErr != "" && (contains(pgErr, "uq_inventory_assets_tenant_serial") || contains(pgErr, "duplicate key")) {
		return domain.ErrInventorySerialDuplicate
	}
	return err
}

func contains(value, needle string) bool {
	for i := 0; i+len(needle) <= len(value); i++ {
		if value[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
