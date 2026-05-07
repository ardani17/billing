package repository

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PackageRepo mengimplementasikan domain.PackageRepository dengan membungkus
// sqlc-generated Queries dan pgxpool.Pool untuk dynamic list query.
type PackageRepo struct {
	// queries adalah sqlc-generated Queries untuk operasi paket.
	queries *Queries

	// pool digunakan untuk dynamic list query (raw SQL dengan pgx).
	pool *pgxpool.Pool
}

// NewPackageRepo membuat instance baru PackageRepo.
func NewPackageRepo(queries *Queries, pool *pgxpool.Pool) *PackageRepo {
	return &PackageRepo{
		queries: queries,
		pool:    pool,
	}
}

// --- Implementasi domain.PackageRepository ---

// Create membuat paket baru dan mengembalikan paket yang dibuat.
func (r *PackageRepo) Create(ctx context.Context, pkg *domain.Package) (*domain.Package, error) {
	row, err := r.queries.CreatePackage(ctx, domainPkgToCreateParams(pkg))
	if err != nil {
		return nil, fmt.Errorf("repository: gagal membuat paket: %w", err)
	}
	return mapPackageRow(row), nil
}

// GetByID mengambil paket berdasarkan ID (termasuk customer_count).
func (r *PackageRepo) GetByID(ctx context.Context, id string) (*domain.Package, error) {
	row, err := r.queries.GetPackageByID(ctx, stringToUUID(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrPackageNotFound
		}
		return nil, fmt.Errorf("repository: gagal mengambil paket by ID: %w", err)
	}
	return mapGetPackageByIDRow(row), nil
}

// Update memperbarui data paket dan mengembalikan paket yang diperbarui.
func (r *PackageRepo) Update(ctx context.Context, pkg *domain.Package) (*domain.Package, error) {
	row, err := r.queries.UpdatePackage(ctx, domainPkgToUpdateParams(pkg))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrPackageNotFound
		}
		return nil, fmt.Errorf("repository: gagal memperbarui paket: %w", err)
	}
	return mapPackageRow(row), nil
}

// Delete menghapus paket secara permanen (hard delete).
func (r *PackageRepo) Delete(ctx context.Context, id string) error {
	err := r.queries.DeletePackage(ctx, stringToUUID(id))
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			switch pgErr.ConstraintName {
			case "fk_customers_package_id", "customers_package_id_fkey":
				return domain.ErrPackageHasCustomers
			case "vouchers_package_id_fkey":
				return domain.ErrPackageHasVouchers
			}
		}
		return fmt.Errorf("repository: gagal menghapus paket: %w", err)
	}
	return nil
}

// UpdateIsActive memperbarui status aktif paket.
func (r *PackageRepo) UpdateIsActive(ctx context.Context, id string, isActive bool) (*domain.Package, error) {
	row, err := r.queries.UpdatePackageIsActive(ctx, UpdatePackageIsActiveParams{
		ID:       stringToUUID(id),
		IsActive: isActive,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrPackageNotFound
		}
		return nil, fmt.Errorf("repository: gagal memperbarui status aktif paket: %w", err)
	}
	return mapPackageRow(row), nil
}

// NameExists mengecek apakah nama paket sudah ada di tenant (exclude ID tertentu).
func (r *PackageRepo) NameExists(ctx context.Context, tenantID, name, excludeID string) (bool, error) {
	// Jika excludeID kosong, gunakan UUID nil agar tidak mengecualikan siapapun
	exID := excludeID
	if exID == "" {
		exID = "00000000-0000-0000-0000-000000000000"
	}
	exists, err := r.queries.PackageNameExists(ctx, PackageNameExistsParams{
		TenantID: stringToUUID(tenantID),
		Name:     name,
		ID:       stringToUUID(exID),
	})
	if err != nil {
		return false, fmt.Errorf("repository: gagal mengecek nama paket exists: %w", err)
	}
	return exists, nil
}

// CustomerCount menghitung jumlah pelanggan aktif yang menggunakan paket.
func (r *PackageRepo) CustomerCount(ctx context.Context, id string) (int, error) {
	count, err := r.queries.PackageCustomerCount(ctx, stringToUUID(id))
	if err != nil {
		return 0, fmt.Errorf("repository: gagal menghitung pelanggan di paket: %w", err)
	}
	return int(count), nil
}

// ListNamesByPrefix mengambil daftar nama paket yang dimulai dengan prefix tertentu.
func (r *PackageRepo) ListNamesByPrefix(ctx context.Context, tenantID, prefix string) ([]string, error) {
	names, err := r.queries.ListPackageNamesByPrefix(ctx, ListPackageNamesByPrefixParams{
		TenantID: stringToUUID(tenantID),
		Name:     prefix + "%",
	})
	if err != nil {
		return nil, fmt.Errorf("repository: gagal mengambil nama paket by prefix: %w", err)
	}
	return names, nil
}

// allowedPackageSortColumns adalah whitelist kolom yang diizinkan untuk sorting paket.
// Mencegah SQL injection pada ORDER BY clause.
var allowedPackageSortColumns = map[string]string{
	"name":          "name",
	"monthly_price": "monthly_price",
	"sell_price":    "sell_price",
	"download_mbps": "download_mbps",
	"created_at":    "created_at",
}

// List mengambil daftar paket dengan dynamic filtering, search, sorting, dan paginasi.
// Menggunakan raw SQL karena sqlc tidak mendukung dynamic WHERE clause.
func (r *PackageRepo) List(ctx context.Context, params domain.PackageListParams) (*domain.PackageListResult, error) {
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
	conditions = append(conditions, fmt.Sprintf("p.tenant_id = $%d", argIdx))
	args = append(args, stringToUUID(params.TenantID))
	argIdx++

	// Type filter
	if params.Type != "" {
		conditions = append(conditions, fmt.Sprintf("p.type = $%d", argIdx))
		args = append(args, params.Type)
		argIdx++
	}

	// IsActive filter
	if params.IsActive != nil {
		conditions = append(conditions, fmt.Sprintf("p.is_active = $%d", argIdx))
		args = append(args, *params.IsActive)
		argIdx++
	}

	// Search filter (ILIKE pada nama paket)
	if params.Search != "" {
		searchPattern := "%" + params.Search + "%"
		conditions = append(conditions, fmt.Sprintf("p.name ILIKE $%d", argIdx))
		args = append(args, searchPattern)
		argIdx++
	}

	whereClause := "WHERE " + strings.Join(conditions, " AND ")

	// Count total
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM packages p %s", whereClause)
	var total int64
	err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("repository: gagal menghitung total paket: %w", err)
	}

	// Build ORDER BY
	orderBy := "p.created_at"
	if params.SortBy != "" {
		if col, ok := allowedPackageSortColumns[params.SortBy]; ok {
			orderBy = "p." + col
		}
	}
	sortOrder := "ASC"
	if strings.EqualFold(params.SortOrder, "desc") {
		sortOrder = "DESC"
	}

	// Build pagination
	offset := (params.Page - 1) * params.PageSize

	// Build data query dengan customer_count subquery
	dataQuery := fmt.Sprintf(`SELECT p.id, p.tenant_id, p.type, p.name, p.description, p.is_active,
		p.download_mbps, p.upload_mbps, p.bandwidth_type,
		p.burst_download_mbps, p.burst_upload_mbps, p.burst_threshold_mbps, p.burst_time_seconds,
		p.quota_type, p.quota_mb, p.quota_action, p.throttle_mbps,
		p.monthly_price, p.installation_fee, p.sell_price, p.reseller_price,
		p.duration_value, p.duration_unit, p.shared_users,
		p.mikrotik_profile_name, p.address_pool, p.parent_queue, p.hotspot_profile_name,
		p.created_at, p.updated_at,
		(SELECT COUNT(*) FROM customers c WHERE c.package_id = p.id AND c.deleted_at IS NULL) AS customer_count
		FROM packages p %s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d`,
		whereClause, orderBy, sortOrder, argIdx, argIdx+1,
	)
	args = append(args, params.PageSize, offset)

	rows, err := r.pool.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("repository: gagal mengambil daftar paket: %w", err)
	}
	defer rows.Close()

	packages := make([]*domain.Package, 0)
	for rows.Next() {
		var row GetPackageByIDRow
		if err := rows.Scan(
			&row.ID, &row.TenantID, &row.Type, &row.Name, &row.Description, &row.IsActive,
			&row.DownloadMbps, &row.UploadMbps, &row.BandwidthType,
			&row.BurstDownloadMbps, &row.BurstUploadMbps, &row.BurstThresholdMbps, &row.BurstTimeSeconds,
			&row.QuotaType, &row.QuotaMb, &row.QuotaAction, &row.ThrottleMbps,
			&row.MonthlyPrice, &row.InstallationFee, &row.SellPrice, &row.ResellerPrice,
			&row.DurationValue, &row.DurationUnit, &row.SharedUsers,
			&row.MikrotikProfileName, &row.AddressPool, &row.ParentQueue, &row.HotspotProfileName,
			&row.CreatedAt, &row.UpdatedAt,
			&row.CustomerCount,
		); err != nil {
			return nil, fmt.Errorf("repository: gagal scan paket row: %w", err)
		}
		packages = append(packages, mapGetPackageByIDRow(row))
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("repository: gagal iterasi paket rows: %w", err)
	}

	totalPages := int(math.Ceil(float64(total) / float64(params.PageSize)))
	if totalPages < 1 {
		totalPages = 1
	}

	return &domain.PackageListResult{
		Data: packages,
		Pagination: domain.PaginationMeta{
			Total:      total,
			Page:       params.Page,
			PageSize:   params.PageSize,
			TotalPages: totalPages,
		},
	}, nil
}
