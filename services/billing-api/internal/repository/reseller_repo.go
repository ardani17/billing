package repository

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ResellerRepo mengimplementasikan domain.ResellerRepository dengan membungkus
// sqlc-generated Queries dan pgxpool.Pool untuk dynamic list query.
type ResellerRepo struct {
	// queries adalah sqlc-generated Queries untuk operasi reseller.
	queries *Queries

	// pool digunakan untuk dynamic list query (raw SQL dengan pgx).
	pool *pgxpool.Pool
}

// NewResellerRepo membuat instance baru ResellerRepo.
func NewResellerRepo(queries *Queries, pool *pgxpool.Pool) *ResellerRepo {
	return &ResellerRepo{
		queries: queries,
		pool:    pool,
	}
}

// --- Helper function untuk mapping sqlc Reseller → domain.Reseller ---

// mapResellerRow memetakan Reseller (sqlc model) ke domain.Reseller.
func mapResellerRow(row Reseller) *domain.Reseller {
	return &domain.Reseller{
		ID:                 uuidToString(row.ID),
		TenantID:           uuidToString(row.TenantID),
		Name:               row.Name,
		Phone:              row.Phone,
		Email:              textToString(row.Email),
		Address:            textToString(row.Address),
		PasswordHash:       row.PasswordHash,
		Balance:            row.Balance,
		DailyPurchaseLimit: int(row.DailyPurchaseLimit),
		Status:             domain.ResellerStatus(row.Status),
		LastLogin:          timestamptzToTimePtr(row.LastLogin),
		CreatedAt:          timestamptzToTime(row.CreatedAt),
		UpdatedAt:          timestamptzToTime(row.UpdatedAt),
	}
}

// mapGetResellerByIDRow memetakan GetResellerByIDRow (sqlc model) ke domain.Reseller.
// Termasuk field komputasi TotalVouchersSold.
func mapGetResellerByIDRow(row GetResellerByIDRow) *domain.Reseller {
	return &domain.Reseller{
		ID:                 uuidToString(row.ID),
		TenantID:           uuidToString(row.TenantID),
		Name:               row.Name,
		Phone:              row.Phone,
		Email:              textToString(row.Email),
		Address:            textToString(row.Address),
		PasswordHash:       row.PasswordHash,
		Balance:            row.Balance,
		DailyPurchaseLimit: int(row.DailyPurchaseLimit),
		Status:             domain.ResellerStatus(row.Status),
		LastLogin:          timestamptzToTimePtr(row.LastLogin),
		TotalVouchersSold:  int(row.TotalVouchersSold),
		CreatedAt:          timestamptzToTime(row.CreatedAt),
		UpdatedAt:          timestamptzToTime(row.UpdatedAt),
	}
}

// --- Implementasi domain.ResellerRepository ---

// Create membuat reseller baru dan mengembalikan reseller yang dibuat.
func (r *ResellerRepo) Create(ctx context.Context, reseller *domain.Reseller) (*domain.Reseller, error) {
	row, err := r.queries.CreateReseller(ctx, CreateResellerParams{
		TenantID:           stringToUUID(reseller.TenantID),
		Name:               reseller.Name,
		Phone:              reseller.Phone,
		Email:              stringToText(reseller.Email),
		Address:            stringToText(reseller.Address),
		PasswordHash:       reseller.PasswordHash,
		Balance:            reseller.Balance,
		DailyPurchaseLimit: int32(reseller.DailyPurchaseLimit),
		Status:             string(reseller.Status),
	})
	if err != nil {
		return nil, fmt.Errorf("repository: gagal membuat reseller: %w", err)
	}
	return mapResellerRow(row), nil
}

// GetByID mengambil reseller berdasarkan ID beserta total_vouchers_sold.
func (r *ResellerRepo) GetByID(ctx context.Context, id string) (*domain.Reseller, error) {
	row, err := r.queries.GetResellerByID(ctx, stringToUUID(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrResellerNotFound
		}
		return nil, fmt.Errorf("repository: gagal mengambil reseller by ID: %w", err)
	}
	return mapGetResellerByIDRow(row), nil
}

// GetByPhone mengambil reseller berdasarkan tenant_id dan nomor telepon (untuk login).
func (r *ResellerRepo) GetByPhone(ctx context.Context, tenantID, phone string) (*domain.Reseller, error) {
	row, err := r.queries.GetResellerByPhone(ctx, GetResellerByPhoneParams{
		TenantID: stringToUUID(tenantID),
		Phone:    phone,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrResellerNotFound
		}
		return nil, fmt.Errorf("repository: gagal mengambil reseller by phone: %w", err)
	}
	return mapResellerRow(row), nil
}

// GetByPhoneGlobal mengambil reseller berdasarkan phone saja (lintas tenant, bypass RLS).
// Digunakan untuk login reseller yang tidak memiliki konteks tenant.
func (r *ResellerRepo) GetByPhoneGlobal(ctx context.Context, phone string) (*domain.Reseller, error) {
	// Gunakan pool langsung untuk bypass RLS
	q := New(r.pool)
	row, err := q.GetResellerByPhoneGlobal(ctx, phone)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrResellerNotFound
		}
		return nil, fmt.Errorf("repository: gagal mengambil reseller by phone global: %w", err)
	}
	return mapResellerRow(row), nil
}

// Update memperbarui data reseller dan mengembalikan reseller yang diperbarui.
func (r *ResellerRepo) Update(ctx context.Context, reseller *domain.Reseller) (*domain.Reseller, error) {
	row, err := r.queries.UpdateReseller(ctx, UpdateResellerParams{
		ID:                 stringToUUID(reseller.ID),
		Name:               reseller.Name,
		Phone:              reseller.Phone,
		Email:              stringToText(reseller.Email),
		Address:            stringToText(reseller.Address),
		DailyPurchaseLimit: int32(reseller.DailyPurchaseLimit),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrResellerNotFound
		}
		return nil, fmt.Errorf("repository: gagal memperbarui reseller: %w", err)
	}
	return mapResellerRow(row), nil
}

// UpdateStatus memperbarui status reseller dan mengembalikan reseller yang diperbarui.
func (r *ResellerRepo) UpdateStatus(ctx context.Context, id string, status domain.ResellerStatus) (*domain.Reseller, error) {
	row, err := r.queries.UpdateResellerStatus(ctx, UpdateResellerStatusParams{
		ID:     stringToUUID(id),
		Status: string(status),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrResellerNotFound
		}
		return nil, fmt.Errorf("repository: gagal memperbarui status reseller: %w", err)
	}
	return mapResellerRow(row), nil
}

// UpdatePasswordHash memperbarui password_hash reseller (untuk reset password).
func (r *ResellerRepo) UpdatePasswordHash(ctx context.Context, id, hash string) error {
	err := r.queries.UpdateResellerPasswordHash(ctx, UpdateResellerPasswordHashParams{
		ID:           stringToUUID(id),
		PasswordHash: hash,
	})
	if err != nil {
		return fmt.Errorf("repository: gagal memperbarui password hash reseller: %w", err)
	}
	return nil
}

// UpdateLastLogin memperbarui timestamp last_login reseller ke waktu sekarang.
func (r *ResellerRepo) UpdateLastLogin(ctx context.Context, id string) error {
	err := r.queries.UpdateResellerLastLogin(ctx, stringToUUID(id))
	if err != nil {
		return fmt.Errorf("repository: gagal memperbarui last login reseller: %w", err)
	}
	return nil
}

// allowedResellerSortColumns adalah whitelist kolom yang diizinkan untuk sorting reseller.
// Mencegah SQL injection pada ORDER BY clause.
var allowedResellerSortColumns = map[string]string{
	"name":       "r.name",
	"balance":    "r.balance",
	"created_at": "r.created_at",
}

// List mengambil daftar reseller dengan dynamic filtering, search, sorting, dan pagination.
// Menggunakan raw SQL karena sqlc tidak mendukung dynamic WHERE clause.
func (r *ResellerRepo) List(ctx context.Context, params domain.ResellerListParams) (*domain.ResellerListResult, error) {
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
	conditions = append(conditions, fmt.Sprintf("r.tenant_id = $%d", argIdx))
	args = append(args, stringToUUID(params.TenantID))
	argIdx++

	// Search filter (case-insensitive ILIKE pada name atau phone)
	if params.Search != "" {
		searchPattern := "%" + params.Search + "%"
		conditions = append(conditions, fmt.Sprintf(
			"(r.name ILIKE $%d OR r.phone ILIKE $%d)",
			argIdx, argIdx,
		))
		args = append(args, searchPattern)
		argIdx++
	}

	// Status filter
	if params.Status != "" {
		conditions = append(conditions, fmt.Sprintf("r.status = $%d", argIdx))
		args = append(args, params.Status)
		argIdx++
	}

	whereClause := "WHERE " + strings.Join(conditions, " AND ")

	// Count total
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM resellers r %s", whereClause)
	var total int64
	err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("repository: gagal menghitung total reseller: %w", err)
	}

	// Build ORDER BY
	orderBy := "r.created_at"
	if params.SortBy != "" {
		if col, ok := allowedResellerSortColumns[params.SortBy]; ok {
			orderBy = col
		}
	}
	sortOrder := "ASC"
	if strings.EqualFold(params.SortOrder, "desc") {
		sortOrder = "DESC"
	}

	// Build pagination
	offset := (params.Page - 1) * params.PageSize

	// Build data query dengan subquery total_vouchers_sold
	dataQuery := fmt.Sprintf(`SELECT r.id, r.tenant_id, r.name, r.phone, r.email, r.address,
		r.password_hash, r.balance, r.daily_purchase_limit, r.status,
		r.last_login, r.created_at, r.updated_at,
		(SELECT COUNT(*) FROM vouchers v
		 WHERE v.reseller_id = r.id
		   AND v.status NOT IN ('tersedia', 'void')) AS total_vouchers_sold
		FROM resellers r %s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d`,
		whereClause, orderBy, sortOrder, argIdx, argIdx+1,
	)
	args = append(args, params.PageSize, offset)

	rows, err := r.pool.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("repository: gagal mengambil daftar reseller: %w", err)
	}
	defer rows.Close()

	resellers := make([]*domain.Reseller, 0)
	for rows.Next() {
		var (
			id                 pgtype.UUID
			tenantID           pgtype.UUID
			name               string
			phone              string
			email              pgtype.Text
			address            pgtype.Text
			passwordHash       string
			balance            int64
			dailyPurchaseLimit int32
			status             string
			lastLogin          pgtype.Timestamptz
			createdAt          pgtype.Timestamptz
			updatedAt          pgtype.Timestamptz
			totalVouchersSold  int64
		)
		if err := rows.Scan(
			&id, &tenantID, &name, &phone, &email, &address,
			&passwordHash, &balance, &dailyPurchaseLimit, &status,
			&lastLogin, &createdAt, &updatedAt,
			&totalVouchersSold,
		); err != nil {
			return nil, fmt.Errorf("repository: gagal scan reseller row: %w", err)
		}
		resellers = append(resellers, &domain.Reseller{
			ID:                 uuidToString(id),
			TenantID:           uuidToString(tenantID),
			Name:               name,
			Phone:              phone,
			Email:              textToString(email),
			Address:            textToString(address),
			PasswordHash:       passwordHash,
			Balance:            balance,
			DailyPurchaseLimit: int(dailyPurchaseLimit),
			Status:             domain.ResellerStatus(status),
			LastLogin:          timestamptzToTimePtr(lastLogin),
			TotalVouchersSold:  int(totalVouchersSold),
			CreatedAt:          timestamptzToTime(createdAt),
			UpdatedAt:          timestamptzToTime(updatedAt),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("repository: gagal iterasi reseller rows: %w", err)
	}

	totalPages := int(math.Ceil(float64(total) / float64(params.PageSize)))
	if totalPages < 1 {
		totalPages = 1
	}

	return &domain.ResellerListResult{
		Data: resellers,
		Pagination: domain.PaginationMeta{
			Total:      total,
			Page:       params.Page,
			PageSize:   params.PageSize,
			TotalPages: totalPages,
		},
	}, nil
}

// PhoneExists mengecek apakah nomor telepon sudah terdaftar di tenant yang sama.
// excludeID digunakan untuk mengecualikan reseller tertentu (saat update).
func (r *ResellerRepo) PhoneExists(ctx context.Context, tenantID, phone, excludeID string) (bool, error) {
	// Jika excludeID kosong, gunakan UUID nil agar tidak mengecualikan siapapun
	exID := excludeID
	if exID == "" {
		exID = "00000000-0000-0000-0000-000000000000"
	}
	exists, err := r.queries.ResellerPhoneExists(ctx, ResellerPhoneExistsParams{
		TenantID: stringToUUID(tenantID),
		Phone:    phone,
		ID:       stringToUUID(exID),
	})
	if err != nil {
		return false, fmt.Errorf("repository: gagal mengecek phone exists: %w", err)
	}
	return exists, nil
}

// GetForUpdate mengambil reseller dengan row lock (SELECT ... FOR UPDATE).
// Digunakan dalam transaksi untuk operasi balance atomik.
func (r *ResellerRepo) GetForUpdate(ctx context.Context, id string) (*domain.Reseller, error) {
	row, err := r.queries.GetResellerForUpdate(ctx, stringToUUID(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrResellerNotFound
		}
		return nil, fmt.Errorf("repository: gagal mengambil reseller for update: %w", err)
	}
	return mapResellerRow(row), nil
}

// UpdateBalance memperbarui saldo reseller (digunakan dalam transaksi atomik).
func (r *ResellerRepo) UpdateBalance(ctx context.Context, id string, newBalance int64) error {
	err := r.queries.UpdateResellerBalance(ctx, UpdateResellerBalanceParams{
		ID:      stringToUUID(id),
		Balance: newBalance,
	})
	if err != nil {
		return fmt.Errorf("repository: gagal memperbarui saldo reseller: %w", err)
	}
	return nil
}

// CountTodayPurchases menghitung jumlah voucher yang dibeli reseller hari ini.
func (r *ResellerRepo) CountTodayPurchases(ctx context.Context, resellerID string) (int, error) {
	count, err := r.queries.CountVouchersSoldToday(ctx, stringToUUID(resellerID))
	if err != nil {
		return 0, fmt.Errorf("repository: gagal menghitung pembelian hari ini: %w", err)
	}
	return int(count), nil
}
