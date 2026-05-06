package domain

import (
	"errors"
	"time"
)

type InventoryItem struct {
	ID          string    `json:"id"`
	TenantID    string    `json:"tenant_id"`
	Name        string    `json:"name"`
	Category    string    `json:"category"`
	Unit        string    `json:"unit"`
	TrackSerial bool      `json:"track_serial"`
	MinStock    int       `json:"min_stock"`
	DefaultCost int64     `json:"default_cost"`
	IsActive    bool      `json:"is_active"`
	Stock       int       `json:"stock,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type InventoryAsset struct {
	ID                 string     `json:"id"`
	TenantID           string     `json:"tenant_id"`
	ItemID             string     `json:"item_id"`
	ItemName           string     `json:"item_name,omitempty"`
	SerialNumber       string     `json:"serial_number"`
	MacAddress         string     `json:"mac_address,omitempty"`
	Status             string     `json:"status"`
	LocationType       string     `json:"location_type"`
	LocationID         string     `json:"location_id,omitempty"`
	AssignedCustomerID string     `json:"assigned_customer_id,omitempty"`
	AssignedCustomer   string     `json:"assigned_customer_name,omitempty"`
	PurchaseCost       int64      `json:"purchase_cost"`
	PurchaseDate       *time.Time `json:"purchase_date,omitempty"`
	WarrantyUntil      *time.Time `json:"warranty_until,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

type InventoryMovement struct {
	ID               string    `json:"id"`
	TenantID         string    `json:"tenant_id"`
	ItemID           string    `json:"item_id"`
	ItemName         string    `json:"item_name,omitempty"`
	AssetID          string    `json:"asset_id,omitempty"`
	MovementType     string    `json:"movement_type"`
	Quantity         int       `json:"quantity"`
	FromLocationType string    `json:"from_location_type,omitempty"`
	FromLocationID   string    `json:"from_location_id,omitempty"`
	ToLocationType   string    `json:"to_location_type,omitempty"`
	ToLocationID     string    `json:"to_location_id,omitempty"`
	CustomerID       string    `json:"customer_id,omitempty"`
	CustomerName     string    `json:"customer_name,omitempty"`
	ExpenseID        string    `json:"expense_id,omitempty"`
	UnitCost         int64     `json:"unit_cost"`
	Notes            string    `json:"notes,omitempty"`
	CreatedByID      string    `json:"created_by_id"`
	CreatedAt        time.Time `json:"created_at"`
}

type InventoryStockItem struct {
	ItemID      string `json:"item_id"`
	ItemName    string `json:"item_name"`
	Category    string `json:"category"`
	Unit        string `json:"unit"`
	TrackSerial bool   `json:"track_serial"`
	MinStock    int    `json:"min_stock"`
	Stock       int    `json:"stock"`
	Status      string `json:"status"`
}

type CreateInventoryItemRequest struct {
	Name        string `json:"name" validate:"required,min=1,max=255"`
	Category    string `json:"category" validate:"required,min=1,max=100"`
	Unit        string `json:"unit" validate:"required,min=1,max=50"`
	TrackSerial bool   `json:"track_serial"`
	MinStock    int    `json:"min_stock" validate:"min=0"`
	DefaultCost int64  `json:"default_cost" validate:"min=0"`
}

type UpdateInventoryItemRequest struct {
	Name        string `json:"name" validate:"omitempty,min=1,max=255"`
	Category    string `json:"category" validate:"omitempty,min=1,max=100"`
	Unit        string `json:"unit" validate:"omitempty,min=1,max=50"`
	TrackSerial *bool  `json:"track_serial,omitempty"`
	MinStock    *int   `json:"min_stock,omitempty" validate:"omitempty,min=0"`
	DefaultCost *int64 `json:"default_cost,omitempty" validate:"omitempty,min=0"`
	IsActive    *bool  `json:"is_active,omitempty"`
}

type CreateInventoryAssetRequest struct {
	ItemID             string `json:"item_id" validate:"required,uuid"`
	SerialNumber       string `json:"serial_number" validate:"required,min=1,max=255"`
	MacAddress         string `json:"mac_address,omitempty"`
	Status             string `json:"status,omitempty"`
	LocationType       string `json:"location_type,omitempty"`
	LocationID         string `json:"location_id,omitempty"`
	AssignedCustomerID string `json:"assigned_customer_id,omitempty"`
	PurchaseCost       int64  `json:"purchase_cost" validate:"min=0"`
	PurchaseDate       string `json:"purchase_date,omitempty"`
	WarrantyUntil      string `json:"warranty_until,omitempty"`
}

type UpdateInventoryAssetRequest struct {
	MacAddress         string `json:"mac_address,omitempty"`
	Status             string `json:"status,omitempty"`
	LocationType       string `json:"location_type,omitempty"`
	LocationID         string `json:"location_id,omitempty"`
	AssignedCustomerID string `json:"assigned_customer_id,omitempty"`
	PurchaseCost       *int64 `json:"purchase_cost,omitempty" validate:"omitempty,min=0"`
	PurchaseDate       string `json:"purchase_date,omitempty"`
	WarrantyUntil      string `json:"warranty_until,omitempty"`
}

type CreateInventoryMovementRequest struct {
	ItemID            string `json:"item_id" validate:"required,uuid"`
	AssetID           string `json:"asset_id,omitempty"`
	MovementType      string `json:"movement_type" validate:"required,oneof=purchase install return transfer adjustment damaged lost rma retired"`
	Quantity          int    `json:"quantity" validate:"required"`
	FromLocationType  string `json:"from_location_type,omitempty"`
	FromLocationID    string `json:"from_location_id,omitempty"`
	ToLocationType    string `json:"to_location_type,omitempty"`
	ToLocationID      string `json:"to_location_id,omitempty"`
	CustomerID        string `json:"customer_id,omitempty"`
	UnitCost          int64  `json:"unit_cost" validate:"min=0"`
	Notes             string `json:"notes,omitempty"`
	CreateExpense     bool   `json:"create_expense"`
	ExpenseCategoryID string `json:"expense_category_id,omitempty"`
}

type AssetActionRequest struct {
	CustomerID   string `json:"customer_id,omitempty"`
	LocationType string `json:"location_type,omitempty"`
	LocationID   string `json:"location_id,omitempty"`
	Notes        string `json:"notes,omitempty"`
}

type CashflowSummary struct {
	OpeningBalance         int64                 `json:"opening_balance"`
	TotalCashIn            int64                 `json:"total_cash_in"`
	TotalCashOut           int64                 `json:"total_cash_out"`
	NetCashflow            int64                 `json:"net_cashflow"`
	ClosingBalanceEstimate int64                 `json:"closing_balance_estimate"`
	Breakdown              []CashflowBreakdown   `json:"breakdown"`
	LatestTransactions     []CashflowTransaction `json:"latest_transactions"`
}

type CashflowBreakdown struct {
	Direction string `json:"direction"`
	Source    string `json:"source"`
	Category  string `json:"category"`
	Amount    int64  `json:"amount"`
}

type CashflowTransaction struct {
	ID          string    `json:"id"`
	Date        time.Time `json:"date"`
	Direction   string    `json:"direction"`
	Source      string    `json:"source"`
	Category    string    `json:"category"`
	Description string    `json:"description"`
	Amount      int64     `json:"amount"`
}

type CashflowTrendPoint struct {
	Date    string `json:"date"`
	CashIn  int64  `json:"cash_in"`
	CashOut int64  `json:"cash_out"`
	Net     int64  `json:"net"`
}

type CreateManualCashflowRequest struct {
	Direction       string `json:"direction" validate:"required,oneof=in out"`
	Category        string `json:"category" validate:"required,min=1,max=100"`
	Description     string `json:"description" validate:"required,min=1,max=500"`
	Amount          int64  `json:"amount" validate:"required,gt=0"`
	TransactionDate string `json:"transaction_date" validate:"required"`
}

var (
	ErrInventoryItemNotFound      = errors.New("item inventaris tidak ditemukan")
	ErrInventoryAssetNotFound     = errors.New("aset inventaris tidak ditemukan")
	ErrInventorySerialDuplicate   = errors.New("serial number sudah terdaftar")
	ErrInventoryStockInsufficient = errors.New("stok inventaris tidak mencukupi")
	ErrInventorySerialRequired    = errors.New("item serial wajib memakai aset serial")
)
