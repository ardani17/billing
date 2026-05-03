package repository

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// VoucherRepo mengimplementasikan domain.VoucherRepository dengan membungkus
// sqlc-generated Queries dan pgxpool.Pool untuk dynamic list query.
type VoucherRepo struct {
	// queries adalah sqlc-generated Queries untuk operasi voucher.
	queries *Queries

	// pool digunakan untuk dynamic list query (raw SQL dengan pgx).
	pool *pgxpool.Pool
}

// NewVoucherRepo membuat instance baru VoucherRepo.
func NewVoucherRepo(queries *Queries, pool *pgxpool.Pool) *VoucherRepo {
	return &VoucherRepo{
		queries: queries,
		pool:    pool,
	}
}

// --- Helper function untuk mapping sqlc Voucher → domain.Voucher ---

// mapVoucherRow memetakan Voucher (sqlc model) ke domain.Voucher.
func mapVoucherRow(row Voucher) *domain.Voucher {
	return &domain.Voucher{
		ID:                    uuidToString(row.ID),
		TenantID:              uuidToString(row.TenantID),
		Code:                  row.Code,
		PackageID:             uuidToString(row.PackageID),
		ResellerID:            uuidToString(row.ResellerID),
		Status:                domain.VoucherStatus(row.Status),
		SellPriceSnapshot:     int8ToInt64Ptr(row.SellPriceSnapshot),
		ResellerPriceSnapshot: int8ToInt64Ptr(row.ResellerPriceSnapshot),
		PurchasedAt:           timestamptzToTimePtr(row.PurchasedAt),
		ActivatedAt:           timestamptzToTimePtr(row.ActivatedAt),
		ExpiresAt:             timestamptzToTimePtr(row.ExpiresAt),
		VoidedAt:              timestamptzToTimePtr(row.VoidedAt),
		CreatedAt:             timestamptzToTime(row.CreatedAt),
		UpdatedAt:             timestamptzToTime(row.UpdatedAt),
	}
}

// mapGetVoucherByIDRow memetakan GetVoucherByIDRow (sqlc model) ke domain.Voucher.
// Termasuk joined field PackageName dan ResellerName.
func mapGetVoucherByIDRow(row GetVoucherByIDRow) *domain.Voucher {
	return &domain.Voucher{
		ID:                    uuidToString(row.ID),
		TenantID:              uuidToString(row.TenantID),
		Code:                  row.Code,
		PackageID:             uuidToString(row.PackageID),
		PackageName:           row.PackageName,
		ResellerID:            uuidToString(row.ResellerID),
		ResellerName:          textToString(row.ResellerName),
		Status:                domain.VoucherStatus(row.Status),
		SellPriceSnapshot:     int8ToInt64Ptr(row.SellPriceSnapshot),
		ResellerPriceSnapshot: int8ToInt64Ptr(row.ResellerPriceSnapshot),
		PurchasedAt:           timestamptzToTimePtr(row.PurchasedAt),
		ActivatedAt:           timestamptzToTimePtr(row.ActivatedAt),
		ExpiresAt:             timestamptzToTimePtr(row.ExpiresAt),
		VoidedAt:              timestamptzToTimePtr(row.VoidedAt),
		CreatedAt:             timestamptzToTime(row.CreatedAt),
		UpdatedAt:             timestamptzToTime(row.UpdatedAt),
	}
}

// --- Implementasi domain.VoucherRepository ---

// BulkCreate membuat beberapa voucher sekaligus menggunakan PostgreSQL COPY protocol.
// Mengembalikan voucher yang dibuat (diambil ulang dari database untuk mendapatkan ID dan timestamp).
func (r *VoucherRepo) BulkCreate(ctx context.Context, vouchers []*domain.Voucher) ([]*domain.Voucher, error) {
	if len(vouchers) == 0 {
		return []*domain.Voucher{}, nil
	}

	// Siapkan parameter untuk sqlc copyfrom
	params := make([]BulkCreateVouchersParams, len(vouchers))
	for i, v := range vouchers {
		params[i] = BulkCreateVouchersParams{
			TenantID:  stringToUUID(v.TenantID),
			Code:      v.Code,
			PackageID: stringToUUID(v.PackageID),
			Status:    string(v.Status),
		}
	}

	// Eksekusi bulk insert via COPY protocol
	_, err := r.queries.BulkCreateVouchers(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("repository: gagal bulk create vouchers: %w", err)
	}

	// Ambil voucher yang baru dibuat berdasarkan kode (karena COPY tidak mengembalikan ID)
	codes := make([]string, len(vouchers))
	for i, v := range vouchers {
		codes[i] = v.Code
	}

	// Query voucher yang baru dibuat berdasarkan tenant_id dan kode
	tenantID := vouchers[0].TenantID
	result := make([]*domain.Voucher, 0, len(vouchers))
	for _, code := range codes {
		row, err := r.queries.GetVoucherByCode(ctx, GetVoucherByCodeParams{
			TenantID: stringToUUID(tenantID),
			Code:     code,
		})
		if err != nil {
			return nil, fmt.Errorf("repository: gagal mengambil voucher setelah bulk create (code=%s): %w", code, err)
		}
		result = append(result, mapVoucherRow(row))
	}

	return result, nil
}

// GetByID mengambil voucher berdasarkan ID beserta nama paket dan nama reseller (joined).
func (r *VoucherRepo) GetByID(ctx context.Context, id string) (*domain.Voucher, error) {
	row, err := r.queries.GetVoucherByID(ctx, stringToUUID(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrVoucherNotFound
		}
		return nil, fmt.Errorf("repository: gagal mengambil voucher by ID: %w", err)
	}
	return mapGetVoucherByIDRow(row), nil
}

// GetByCode mengambil voucher berdasarkan kode dalam tenant tertentu.
func (r *VoucherRepo) GetByCode(ctx context.Context, tenantID, code string) (*domain.Voucher, error) {
	row, err := r.queries.GetVoucherByCode(ctx, GetVoucherByCodeParams{
		TenantID: stringToUUID(tenantID),
		Code:     code,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrVoucherNotFound
		}
		return nil, fmt.Errorf("repository: gagal mengambil voucher by code: %w", err)
	}
	return mapVoucherRow(row), nil
}

// UpdateStatus memperbarui status voucher dan mengembalikan voucher yang diperbarui.
func (r *VoucherRepo) UpdateStatus(ctx context.Context, id string, status domain.VoucherStatus) (*domain.Voucher, error) {
	// Gunakan query khusus untuk void agar voided_at juga di-set
	if status == domain.VoucherStatusVoid {
		row, err := r.queries.UpdateVoucherVoid(ctx, stringToUUID(id))
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, domain.ErrVoucherNotFound
			}
			return nil, fmt.Errorf("repository: gagal memperbarui voucher ke void: %w", err)
		}
		return mapVoucherRow(row), nil
	}

	// Gunakan query khusus untuk expired
	if status == domain.VoucherStatusExpired {
		row, err := r.queries.UpdateVoucherExpired(ctx, stringToUUID(id))
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, domain.ErrVoucherNotFound
			}
			return nil, fmt.Errorf("repository: gagal memperbarui voucher ke expired: %w", err)
		}
		return mapVoucherRow(row), nil
	}

	// Query umum untuk status lainnya
	row, err := r.queries.UpdateVoucherStatus(ctx, UpdateVoucherStatusParams{
		ID:     stringToUUID(id),
		Status: string(status),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrVoucherNotFound
		}
		return nil, fmt.Errorf("repository: gagal memperbarui status voucher: %w", err)
	}
	return mapVoucherRow(row), nil
}

// allowedVoucherSortColumns adalah whitelist kolom yang diizinkan untuk sorting voucher.
// Mencegah SQL injection pada ORDER BY clause.
var allowedVoucherSortColumns = map[string]string{
	"code":         "v.code",
	"status":       "v.status",
	"created_at":   "v.created_at",
	"purchased_at": "v.purchased_at",
}

// List mengambil daftar voucher dengan dynamic filtering, search, sorting, dan pagination.
// Menggunakan raw SQL karena sqlc tidak mendukung dynamic WHERE clause.
// Termasuk joined field package_name dan reseller_name.
func (r *VoucherRepo) List(ctx context.Context, params domain.VoucherListParams) (*domain.VoucherListResult, error) {
	// Default values
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PageSize < 1 {
		params.PageSize = 25
	}

	// Build WHERE clauses
	var conditions []string
	var args []interface{}
	argIdx := 1

	// Tenant filter (wajib)
	conditions = append(conditions, fmt.Sprintf("v.tenant_id = $%d", argIdx))
	args = append(args, stringToUUID(params.TenantID))
	argIdx++

	// Search filter (case-insensitive ILIKE pada code)
	if params.Search != "" {
		searchPattern := "%" + params.Search + "%"
		conditions = append(conditions, fmt.Sprintf("v.code ILIKE $%d", argIdx))
		args = append(args, searchPattern)
		argIdx++
	}

	// Status filter
	if params.Status != "" {
		conditions = append(conditions, fmt.Sprintf("v.status = $%d", argIdx))
		args = append(args, params.Status)
		argIdx++
	}

	// Package ID filter
	if params.PackageID != "" {
		conditions = append(conditions, fmt.Sprintf("v.package_id = $%d", argIdx))
		args = append(args, stringToUUID(params.PackageID))
		argIdx++
	}

	// Reseller ID filter
	if params.ResellerID != "" {
		conditions = append(conditions, fmt.Sprintf("v.reseller_id = $%d", argIdx))
		args = append(args, stringToUUID(params.ResellerID))
		argIdx++
	}

	whereClause := "WHERE " + strings.Join(conditions, " AND ")

	// Count total
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM vouchers v %s", whereClause)
	var total int64
	err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("repository: gagal menghitung total voucher: %w", err)
	}

	// Build ORDER BY
	orderBy := "v.created_at"
	if params.SortBy != "" {
		if col, ok := allowedVoucherSortColumns[params.SortBy]; ok {
			orderBy = col
		}
	}
	sortOrder := "ASC"
	if strings.EqualFold(params.SortOrder, "desc") {
		sortOrder = "DESC"
	}

	// Build pagination
	offset := (params.Page - 1) * params.PageSize

	// Build data query dengan joined package_name dan reseller_name
	dataQuery := fmt.Sprintf(`SELECT v.id, v.tenant_id, v.code, v.package_id, v.reseller_id,
		v.status, v.sell_price_snapshot, v.reseller_price_snapshot,
		v.purchased_at, v.activated_at, v.expires_at, v.voided_at,
		v.created_at, v.updated_at,
		p.name AS package_name,
		COALESCE(re.name, '') AS reseller_name
		FROM vouchers v
		JOIN packages p ON p.id = v.package_id
		LEFT JOIN resellers re ON re.id = v.reseller_id
		%s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d`,
		whereClause, orderBy, sortOrder, argIdx, argIdx+1,
	)
	args = append(args, params.PageSize, offset)

	rows, err := r.pool.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("repository: gagal mengambil daftar voucher: %w", err)
	}
	defer rows.Close()

	vouchers := make([]*domain.Voucher, 0)
	for rows.Next() {
		var (
			id                    pgtype.UUID
			tenantID              pgtype.UUID
			code                  string
			packageID             pgtype.UUID
			resellerID            pgtype.UUID
			status                string
			sellPriceSnapshot     pgtype.Int8
			resellerPriceSnapshot pgtype.Int8
			purchasedAt           pgtype.Timestamptz
			activatedAt           pgtype.Timestamptz
			expiresAt             pgtype.Timestamptz
			voidedAt              pgtype.Timestamptz
			createdAt             pgtype.Timestamptz
			updatedAt             pgtype.Timestamptz
			packageName           string
			resellerName          string
		)
		if err := rows.Scan(
			&id, &tenantID, &code, &packageID, &resellerID,
			&status, &sellPriceSnapshot, &resellerPriceSnapshot,
			&purchasedAt, &activatedAt, &expiresAt, &voidedAt,
			&createdAt, &updatedAt,
			&packageName, &resellerName,
		); err != nil {
			return nil, fmt.Errorf("repository: gagal scan voucher row: %w", err)
		}
		vouchers = append(vouchers, &domain.Voucher{
			ID:                    uuidToString(id),
			TenantID:              uuidToString(tenantID),
			Code:                  code,
			PackageID:             uuidToString(packageID),
			PackageName:           packageName,
			ResellerID:            uuidToString(resellerID),
			ResellerName:          resellerName,
			Status:                domain.VoucherStatus(status),
			SellPriceSnapshot:     int8ToInt64Ptr(sellPriceSnapshot),
			ResellerPriceSnapshot: int8ToInt64Ptr(resellerPriceSnapshot),
			PurchasedAt:           timestamptzToTimePtr(purchasedAt),
			ActivatedAt:           timestamptzToTimePtr(activatedAt),
			ExpiresAt:             timestamptzToTimePtr(expiresAt),
			VoidedAt:              timestamptzToTimePtr(voidedAt),
			CreatedAt:             timestamptzToTime(createdAt),
			UpdatedAt:             timestamptzToTime(updatedAt),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("repository: gagal iterasi voucher rows: %w", err)
	}

	totalPages := int(math.Ceil(float64(total) / float64(params.PageSize)))
	if totalPages < 1 {
		totalPages = 1
	}

	return &domain.VoucherListResult{
		Data: vouchers,
		Pagination: domain.PaginationMeta{
			Total:      total,
			Page:       params.Page,
			PageSize:   params.PageSize,
			TotalPages: totalPages,
		},
	}, nil
}

// allowedResellerVoucherSortColumns adalah whitelist kolom sorting untuk list voucher reseller.
var allowedResellerVoucherSortColumns = map[string]string{
	"code":         "v.code",
	"status":       "v.status",
	"purchased_at": "v.purchased_at",
}

// ListByReseller mengambil daftar voucher milik reseller tertentu dengan filtering dan pagination.
// Menggunakan raw SQL karena sqlc tidak mendukung dynamic WHERE clause.
func (r *VoucherRepo) ListByReseller(ctx context.Context, params domain.ResellerVoucherListParams) (*domain.VoucherListResult, error) {
	// Default values
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PageSize < 1 {
		params.PageSize = 25
	}

	// Build WHERE clauses
	var conditions []string
	var args []interface{}
	argIdx := 1

	// Tenant filter (wajib)
	conditions = append(conditions, fmt.Sprintf("v.tenant_id = $%d", argIdx))
	args = append(args, stringToUUID(params.TenantID))
	argIdx++

	// Reseller filter (wajib)
	conditions = append(conditions, fmt.Sprintf("v.reseller_id = $%d", argIdx))
	args = append(args, stringToUUID(params.ResellerID))
	argIdx++

	// Status filter
	if params.Status != "" {
		conditions = append(conditions, fmt.Sprintf("v.status = $%d", argIdx))
		args = append(args, params.Status)
		argIdx++
	}

	// Package ID filter
	if params.PackageID != "" {
		conditions = append(conditions, fmt.Sprintf("v.package_id = $%d", argIdx))
		args = append(args, stringToUUID(params.PackageID))
		argIdx++
	}

	whereClause := "WHERE " + strings.Join(conditions, " AND ")

	// Count total
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM vouchers v %s", whereClause)
	var total int64
	err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("repository: gagal menghitung total voucher reseller: %w", err)
	}

	// Build ORDER BY
	orderBy := "v.purchased_at"
	if params.SortBy != "" {
		if col, ok := allowedResellerVoucherSortColumns[params.SortBy]; ok {
			orderBy = col
		}
	}
	sortOrder := "DESC"
	if strings.EqualFold(params.SortOrder, "asc") {
		sortOrder = "ASC"
	}

	// Build pagination
	offset := (params.Page - 1) * params.PageSize

	// Build data query dengan joined package_name
	dataQuery := fmt.Sprintf(`SELECT v.id, v.tenant_id, v.code, v.package_id, v.reseller_id,
		v.status, v.sell_price_snapshot, v.reseller_price_snapshot,
		v.purchased_at, v.activated_at, v.expires_at, v.voided_at,
		v.created_at, v.updated_at,
		p.name AS package_name
		FROM vouchers v
		JOIN packages p ON p.id = v.package_id
		%s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d`,
		whereClause, orderBy, sortOrder, argIdx, argIdx+1,
	)
	args = append(args, params.PageSize, offset)

	rows, err := r.pool.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("repository: gagal mengambil daftar voucher reseller: %w", err)
	}
	defer rows.Close()

	vouchers := make([]*domain.Voucher, 0)
	for rows.Next() {
		var (
			id                    pgtype.UUID
			tenantID              pgtype.UUID
			code                  string
			packageID             pgtype.UUID
			resellerID            pgtype.UUID
			status                string
			sellPriceSnapshot     pgtype.Int8
			resellerPriceSnapshot pgtype.Int8
			purchasedAt           pgtype.Timestamptz
			activatedAt           pgtype.Timestamptz
			expiresAt             pgtype.Timestamptz
			voidedAt              pgtype.Timestamptz
			createdAt             pgtype.Timestamptz
			updatedAt             pgtype.Timestamptz
			packageName           string
		)
		if err := rows.Scan(
			&id, &tenantID, &code, &packageID, &resellerID,
			&status, &sellPriceSnapshot, &resellerPriceSnapshot,
			&purchasedAt, &activatedAt, &expiresAt, &voidedAt,
			&createdAt, &updatedAt,
			&packageName,
		); err != nil {
			return nil, fmt.Errorf("repository: gagal scan voucher reseller row: %w", err)
		}
		vouchers = append(vouchers, &domain.Voucher{
			ID:                    uuidToString(id),
			TenantID:              uuidToString(tenantID),
			Code:                  code,
			PackageID:             uuidToString(packageID),
			PackageName:           packageName,
			ResellerID:            uuidToString(resellerID),
			Status:                domain.VoucherStatus(status),
			SellPriceSnapshot:     int8ToInt64Ptr(sellPriceSnapshot),
			ResellerPriceSnapshot: int8ToInt64Ptr(resellerPriceSnapshot),
			PurchasedAt:           timestamptzToTimePtr(purchasedAt),
			ActivatedAt:           timestamptzToTimePtr(activatedAt),
			ExpiresAt:             timestamptzToTimePtr(expiresAt),
			VoidedAt:              timestamptzToTimePtr(voidedAt),
			CreatedAt:             timestamptzToTime(createdAt),
			UpdatedAt:             timestamptzToTime(updatedAt),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("repository: gagal iterasi voucher reseller rows: %w", err)
	}

	totalPages := int(math.Ceil(float64(total) / float64(params.PageSize)))
	if totalPages < 1 {
		totalPages = 1
	}

	return &domain.VoucherListResult{
		Data: vouchers,
		Pagination: domain.PaginationMeta{
			Total:      total,
			Page:       params.Page,
			PageSize:   params.PageSize,
			TotalPages: totalPages,
		},
	}, nil
}

// GetAvailableByPackage mengambil voucher tersedia (status=tersedia) untuk paket tertentu.
// Digunakan untuk assign voucher ke reseller.
func (r *VoucherRepo) GetAvailableByPackage(ctx context.Context, packageID string, limit int) ([]*domain.Voucher, error) {
	query := `SELECT id, tenant_id, code, package_id, reseller_id, status,
		sell_price_snapshot, reseller_price_snapshot,
		purchased_at, activated_at, expires_at, voided_at,
		created_at, updated_at
		FROM vouchers
		WHERE package_id = $1 AND status = 'tersedia'
		ORDER BY created_at ASC
		LIMIT $2`

	rows, err := r.pool.Query(ctx, query, stringToUUID(packageID), limit)
	if err != nil {
		return nil, fmt.Errorf("repository: gagal mengambil voucher tersedia by package: %w", err)
	}
	defer rows.Close()

	vouchers := make([]*domain.Voucher, 0)
	for rows.Next() {
		var v Voucher
		if err := rows.Scan(
			&v.ID, &v.TenantID, &v.Code, &v.PackageID, &v.ResellerID,
			&v.Status, &v.SellPriceSnapshot, &v.ResellerPriceSnapshot,
			&v.PurchasedAt, &v.ActivatedAt, &v.ExpiresAt, &v.VoidedAt,
			&v.CreatedAt, &v.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("repository: gagal scan voucher tersedia row: %w", err)
		}
		vouchers = append(vouchers, mapVoucherRow(v))
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("repository: gagal iterasi voucher tersedia rows: %w", err)
	}

	return vouchers, nil
}

// BulkUpdateStatus memperbarui status beberapa voucher sekaligus.
// Mengembalikan hasil per item (sukses/gagal) untuk error handling granular.
func (r *VoucherRepo) BulkUpdateStatus(ctx context.Context, ids []string, status domain.VoucherStatus) ([]domain.BulkResult, error) {
	results := make([]domain.BulkResult, 0, len(ids))
	for _, id := range ids {
		_, err := r.UpdateStatus(ctx, id, status)
		if err != nil {
			results = append(results, domain.BulkResult{
				ID:      id,
				Success: false,
				Error:   fmt.Errorf("gagal update status: %w", err),
			})
		} else {
			results = append(results, domain.BulkResult{
				ID:      id,
				Success: true,
			})
		}
	}
	return results, nil
}

// BulkAssign meng-assign voucher ke reseller oleh admin (tanpa potong saldo, tanpa snapshot).
// Mengembalikan hasil per item (sukses/gagal) untuk error handling granular.
func (r *VoucherRepo) BulkAssign(ctx context.Context, ids []string, resellerID string) ([]domain.BulkResult, error) {
	results := make([]domain.BulkResult, 0, len(ids))
	for _, id := range ids {
		_, err := r.queries.AdminAssignVoucher(ctx, AdminAssignVoucherParams{
			ID:         stringToUUID(id),
			ResellerID: stringToUUID(resellerID),
		})
		if err != nil {
			results = append(results, domain.BulkResult{
				ID:      id,
				Success: false,
				Error:   fmt.Errorf("gagal assign voucher: %w", err),
			})
		} else {
			results = append(results, domain.BulkResult{
				ID:      id,
				Success: true,
			})
		}
	}
	return results, nil
}

// AssignToReseller meng-assign voucher ke reseller saat pembelian.
// Set snapshot harga, purchased_at, dan expires_at.
func (r *VoucherRepo) AssignToReseller(ctx context.Context, id string, resellerID string, sellSnapshot, resellerSnapshot int64, expiresAt time.Time) (*domain.Voucher, error) {
	row, err := r.queries.AssignVoucherToReseller(ctx, AssignVoucherToResellerParams{
		ID:                    stringToUUID(id),
		ResellerID:            stringToUUID(resellerID),
		SellPriceSnapshot:     pgtype.Int8{Int64: sellSnapshot, Valid: true},
		ResellerPriceSnapshot: pgtype.Int8{Int64: resellerSnapshot, Valid: true},
		PurchasedAt:           pgtype.Timestamptz{Time: time.Now(), Valid: true},
		ExpiresAt:             pgtype.Timestamptz{Time: expiresAt, Valid: true},
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrVoucherNotFound
		}
		return nil, fmt.Errorf("repository: gagal assign voucher ke reseller: %w", err)
	}
	return mapVoucherRow(row), nil
}

// GetExpiredVouchers mengambil voucher terjual yang sudah melewati expires_at (untuk cron expiry).
func (r *VoucherRepo) GetExpiredVouchers(ctx context.Context, batchSize int) ([]*domain.Voucher, error) {
	rows, err := r.queries.GetExpiredVouchers(ctx, int32(batchSize))
	if err != nil {
		return nil, fmt.Errorf("repository: gagal mengambil expired vouchers: %w", err)
	}

	vouchers := make([]*domain.Voucher, 0, len(rows))
	for _, row := range rows {
		vouchers = append(vouchers, mapVoucherRow(row))
	}
	return vouchers, nil
}

// CodeExists mengecek apakah kode voucher sudah ada di tenant.
func (r *VoucherRepo) CodeExists(ctx context.Context, tenantID, code string) (bool, error) {
	exists, err := r.queries.VoucherCodeExists(ctx, VoucherCodeExistsParams{
		TenantID: stringToUUID(tenantID),
		Code:     code,
	})
	if err != nil {
		return false, fmt.Errorf("repository: gagal mengecek code exists: %w", err)
	}
	return exists, nil
}

// GetByIDs mengambil beberapa voucher berdasarkan array of IDs.
func (r *VoucherRepo) GetByIDs(ctx context.Context, ids []string) ([]*domain.Voucher, error) {
	uuids := make([]pgtype.UUID, len(ids))
	for i, id := range ids {
		uuids[i] = stringToUUID(id)
	}

	rows, err := r.queries.GetVouchersByIDs(ctx, uuids)
	if err != nil {
		return nil, fmt.Errorf("repository: gagal mengambil vouchers by IDs: %w", err)
	}

	vouchers := make([]*domain.Voucher, 0, len(rows))
	for _, row := range rows {
		vouchers = append(vouchers, mapVoucherRow(row))
	}
	return vouchers, nil
}

// CountByResellerAndStatus menghitung voucher per reseller dan array status.
func (r *VoucherRepo) CountByResellerAndStatus(ctx context.Context, resellerID string, statuses []domain.VoucherStatus) (int, error) {
	// Konversi []VoucherStatus ke []string untuk sqlc query
	statusStrs := make([]string, len(statuses))
	for i, s := range statuses {
		statusStrs[i] = string(s)
	}

	count, err := r.queries.CountVouchersByResellerAndStatus(ctx, CountVouchersByResellerAndStatusParams{
		ResellerID: stringToUUID(resellerID),
		Status:     statusStrs,
	})
	if err != nil {
		return 0, fmt.Errorf("repository: gagal menghitung voucher by reseller dan status: %w", err)
	}
	return int(count), nil
}

// CountSoldToday menghitung voucher yang dibeli reseller hari ini (berdasarkan purchased_at).
func (r *VoucherRepo) CountSoldToday(ctx context.Context, resellerID string) (int, error) {
	count, err := r.queries.CountVouchersSoldToday(ctx, stringToUUID(resellerID))
	if err != nil {
		return 0, fmt.Errorf("repository: gagal menghitung voucher sold today: %w", err)
	}
	return int(count), nil
}
