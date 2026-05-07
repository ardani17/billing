package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/big"
	"strings"
	"time"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// CustomerRepo mengimplementasikan domain.CustomerRepository dengan membungkus
// Query hasil buat sqlc dan pgxpool.Pool untuk kueri daftar dinamis.
type CustomerRepo struct {
	// queries adalah Query hasil buat sqlc untuk operasi customer.
	queries *Queries

	// pool digunakan untuk kueri daftar dinamis (SQL mentah dengan pgx).
	pool *pgxpool.Pool
}

// NewCustomerRepo membuat instance baru CustomerRepo.
func NewCustomerRepo(queries *Queries, pool *pgxpool.Pool) *CustomerRepo {
	return &CustomerRepo{
		queries: queries,
		pool:    pool,
	}
}

// --- Fungsi bantu untuk konversi pgtype.Numeric ↔ float64 ---

// numericToFloat64 mengkonversi pgtype.Numeric ke float64.
// Mengembalikan 0 jika Numeric tidak valid.
func numericToFloat64(n pgtype.Numeric) float64 {
	if !n.Valid {
		return 0
	}
	f, _ := n.Float64Value()
	return f.Float64
}

// float64ToNumeric mengkonversi float64 ke pgtype.Numeric.
func float64ToNumeric(f float64) pgtype.Numeric {
	var n pgtype.Numeric
	n.Valid = true
	// Gunakan big.Float untuk konversi presisi tinggi
	bf := new(big.Float).SetFloat64(f)
	// Konversi ke string lalu scan ke Numeric untuk presisi yang benar
	_ = n.Scan(bf.Text('f', 7))
	return n
}

// float64PtrToNumeric mengkonversi *float64 ke pgtype.Numeric.
// Mengembalikan Numeric tidak valid jika pointer nil.
func float64PtrToNumeric(f *float64) pgtype.Numeric {
	if f == nil {
		return pgtype.Numeric{Valid: false}
	}
	return float64ToNumeric(*f)
}

// numericToFloat64Ptr mengkonversi pgtype.Numeric ke *float64.
// Mengembalikan nil jika Numeric tidak valid.
func numericToFloat64Ptr(n pgtype.Numeric) *float64 {
	if !n.Valid {
		return nil
	}
	f := numericToFloat64(n)
	return &f
}

// dateToTime mengkonversi pgtype.Date ke time.Time.
// Mengembalikan zero time jika Date tidak valid.
func dateToTime(d pgtype.Date) time.Time {
	if !d.Valid {
		return time.Time{}
	}
	return d.Time
}

// timeToDate mengkonversi time.Time ke pgtype.Date.
func timeToDate(t time.Time) pgtype.Date {
	return pgtype.Date{Time: t, Valid: !t.IsZero()}
}

// --- Fungsi bantu untuk pemetaan sqlc Customer -> domain.Customer ---

// mapCustomerRow memetakan Customer (sqlc model) ke domain.Customer.
func mapCustomerRow(row Customer) *domain.Customer {
	return &domain.Customer{
		ID:               uuidToString(row.ID),
		TenantID:         uuidToString(row.TenantID),
		CustomerIDSeq:    textToString(row.CustomerIDSeq),
		Name:             row.Name,
		Phone:            row.Phone,
		Email:            textToString(row.Email),
		Address:          row.Address,
		AreaID:           uuidToString(row.AreaID),
		Latitude:         numericToFloat64(row.Latitude),
		Longitude:        numericToFloat64(row.Longitude),
		PackageID:        uuidToString(row.PackageID),
		ActivationDate:   dateToTime(row.ActivationDate),
		DueDate:          int(row.DueDate),
		ConnectionMethod: domain.ConnectionMethod(row.ConnectionMethod),
		PPPoEUsername:    textToString(row.PppoeUsername),
		PPPoEPassword:    textToString(row.PppoePassword),
		MACAddress:       textToString(row.MacAddress),
		RouterID:         uuidToString(row.RouterID),
		ODPPort:          textToString(row.OdpPort),
		CreditBalance:    row.CreditBalance,
		Notes:            textToString(row.Notes),
		Status:           domain.CustomerStatus(row.Status),
		DeletedAt:        timestamptzToTimePtr(row.DeletedAt),
		CreatedAt:        timestamptzToTime(row.CreatedAt),
		UpdatedAt:        timestamptzToTime(row.UpdatedAt),
	}
}

// --- Implementasi domain.CustomerRepository ---

// Buat membuat customer baru dan mengembalikan customer yang dibuat.
func (r *CustomerRepo) Create(ctx context.Context, customer *domain.Customer) (*domain.Customer, error) {
	row, err := r.queries.CreateCustomer(ctx, CreateCustomerParams{
		TenantID:         stringToUUID(customer.TenantID),
		CustomerIDSeq:    stringToText(customer.CustomerIDSeq),
		Name:             customer.Name,
		Phone:            customer.Phone,
		Email:            stringToText(customer.Email),
		Address:          customer.Address,
		AreaID:           stringToUUID(customer.AreaID),
		Latitude:         float64ToNumeric(customer.Latitude),
		Longitude:        float64ToNumeric(customer.Longitude),
		PackageID:        stringToUUID(customer.PackageID),
		ActivationDate:   timeToDate(customer.ActivationDate),
		DueDate:          int32(customer.DueDate),
		ConnectionMethod: string(customer.ConnectionMethod),
		PppoeUsername:    stringToText(customer.PPPoEUsername),
		PppoePassword:    stringToText(customer.PPPoEPassword),
		MacAddress:       stringToText(customer.MACAddress),
		RouterID:         stringToUUID(customer.RouterID),
		OdpPort:          stringToText(customer.ODPPort),
		CreditBalance:    customer.CreditBalance,
		Notes:            stringToText(customer.Notes),
		Status:           string(customer.Status),
	})
	if err != nil {
		return nil, fmt.Errorf("repository: gagal membuat customer: %w", err)
	}
	return mapCustomerRow(row), nil
}

// GetByID mengambil customer berdasarkan ID.
func (r *CustomerRepo) GetByID(ctx context.Context, id string) (*domain.Customer, error) {
	row, err := r.queries.GetCustomerByID(ctx, stringToUUID(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrCustomerNotFound
		}
		return nil, fmt.Errorf("repository: gagal mengambil customer by ID: %w", err)
	}
	customer := mapCustomerRow(row)
	if err := r.loadCustomerPackageName(ctx, customer); err != nil {
		return nil, err
	}
	return customer, nil
}

func (r *CustomerRepo) loadCustomerPackageName(ctx context.Context, customer *domain.Customer) error {
	if customer == nil || customer.PackageID == "" {
		return nil
	}

	var packageName string
	err := r.pool.QueryRow(ctx,
		`SELECT name FROM packages WHERE id = $1 AND tenant_id = $2`,
		stringToUUID(customer.PackageID),
		stringToUUID(customer.TenantID),
	).Scan(&packageName)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return fmt.Errorf("repository: gagal mengambil nama paket customer: %w", err)
	}
	customer.PackageName = packageName
	return nil
}

// Perbarui memperbarui data customer dan mengembalikan customer yang diperbarui.
func (r *CustomerRepo) Update(ctx context.Context, customer *domain.Customer) (*domain.Customer, error) {
	row, err := r.queries.UpdateCustomer(ctx, UpdateCustomerParams{
		ID:               stringToUUID(customer.ID),
		Name:             customer.Name,
		Phone:            customer.Phone,
		Email:            stringToText(customer.Email),
		Address:          customer.Address,
		AreaID:           stringToUUID(customer.AreaID),
		Latitude:         float64ToNumeric(customer.Latitude),
		Longitude:        float64ToNumeric(customer.Longitude),
		PackageID:        stringToUUID(customer.PackageID),
		ActivationDate:   timeToDate(customer.ActivationDate),
		DueDate:          int32(customer.DueDate),
		ConnectionMethod: string(customer.ConnectionMethod),
		PppoeUsername:    stringToText(customer.PPPoEUsername),
		PppoePassword:    stringToText(customer.PPPoEPassword),
		MacAddress:       stringToText(customer.MACAddress),
		RouterID:         stringToUUID(customer.RouterID),
		OdpPort:          stringToText(customer.ODPPort),
		CreditBalance:    customer.CreditBalance,
		Notes:            stringToText(customer.Notes),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrCustomerNotFound
		}
		return nil, fmt.Errorf("repository: gagal memperbarui customer: %w", err)
	}
	return mapCustomerRow(row), nil
}

// SoftDelete menandai customer sebagai dihapus (hapus lunak).
func (r *CustomerRepo) SoftDelete(ctx context.Context, id string) error {
	err := r.queries.SoftDeleteCustomer(ctx, stringToUUID(id))
	if err != nil {
		return fmt.Errorf("repository: gagal soft-delete customer: %w", err)
	}
	return nil
}

// allowedSortColumns adalah whitelist kolom yang diizinkan untuk pengurutan.
// Mencegah SQL injection pada klausa ORDER BY.
var allowedSortColumns = map[string]string{
	"name":            "c.name",
	"customer_id_seq": "c.customer_id_seq",
	"status":          "c.status",
	"created_at":      "c.created_at",
	"due_date":        "c.due_date",
}

// Daftar mengambil daftar customer dengan dinamis Filtering, Pencarian, pengurutan, dan paginasi.
// Menggunakan SQL mentah karena sqlc tidak mendukung klausa WHERE dinamis.
func (r *CustomerRepo) List(ctx context.Context, params domain.CustomerListParams) (*domain.CustomerListResult, error) {
	// Nilai bawaan
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PageSize < 1 {
		params.PageSize = 25
	}

	// Bangun klausa WHERE
	var conditions []string
	var args []interface{}
	argIdx := 1

	// Filter tenant (wajib)
	conditions = append(conditions, fmt.Sprintf("c.tenant_id = $%d", argIdx))
	args = append(args, stringToUUID(params.TenantID))
	argIdx++

	// Kecualikan hapus lunak
	conditions = append(conditions, "c.deleted_at IS NULL")

	// Filter pencarian tanpa membedakan huruf besar/kecil pada name, customer_id_seq, address, dan phone
	if params.Search != "" {
		searchPattern := "%" + params.Search + "%"
		conditions = append(conditions, fmt.Sprintf(
			"(c.name ILIKE $%d OR c.customer_id_seq ILIKE $%d OR c.address ILIKE $%d OR c.phone ILIKE $%d)",
			argIdx, argIdx, argIdx, argIdx,
		))
		args = append(args, searchPattern)
		argIdx++
	}

	// Filter status
	if params.Status != "" {
		conditions = append(conditions, fmt.Sprintf("c.status = $%d", argIdx))
		args = append(args, params.Status)
		argIdx++
	}

	// Filter ID paket
	if params.PackageID != "" {
		conditions = append(conditions, fmt.Sprintf("c.package_id = $%d", argIdx))
		args = append(args, stringToUUID(params.PackageID))
		argIdx++
	}

	// Filter ID area
	if params.AreaID != "" {
		conditions = append(conditions, fmt.Sprintf("c.area_id = $%d", argIdx))
		args = append(args, stringToUUID(params.AreaID))
		argIdx++
	}

	// Filter tanggal jatuh tempo
	if params.DueDate != nil {
		conditions = append(conditions, fmt.Sprintf("c.due_date = $%d", argIdx))
		args = append(args, *params.DueDate)
		argIdx++
	}

	whereClause := "WHERE " + strings.Join(conditions, " AND ")

	// Hitung total
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM customers c %s", whereClause)
	var total int64
	err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("repository: gagal menghitung total customer: %w", err)
	}

	// Bangun ORDER BY
	orderBy := "created_at"
	if params.SortBy != "" {
		if col, ok := allowedSortColumns[params.SortBy]; ok {
			orderBy = col
		}
	}
	sortOrder := "ASC"
	if strings.EqualFold(params.SortOrder, "desc") {
		sortOrder = "DESC"
	}

	// Bangun paginasi
	offset := (params.Page - 1) * params.PageSize

	// Bangun data kueri
	dataQuery := fmt.Sprintf(`SELECT c.id, c.tenant_id, c.customer_id_seq, c.name, c.phone, c.email, c.address,
		c.area_id, c.latitude, c.longitude, c.package_id, c.activation_date,
		c.due_date, c.connection_method, c.pppoe_username, c.pppoe_password,
		c.mac_address, c.router_id, c.odp_port, c.credit_balance, c.notes, c.status,
		c.deleted_at, c.created_at, c.updated_at, COALESCE(p.name, '') AS package_name
		FROM customers c
		LEFT JOIN packages p ON p.id = c.package_id AND p.tenant_id = c.tenant_id
		%s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d`,
		whereClause, orderBy, sortOrder, argIdx, argIdx+1,
	)
	args = append(args, params.PageSize, offset)

	rows, err := r.pool.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("repository: gagal mengambil daftar customer: %w", err)
	}
	defer rows.Close()

	customers := make([]*domain.Customer, 0)
	for rows.Next() {
		var c Customer
		var packageName string
		if err := rows.Scan(
			&c.ID, &c.TenantID, &c.CustomerIDSeq, &c.Name, &c.Phone, &c.Email, &c.Address,
			&c.AreaID, &c.Latitude, &c.Longitude, &c.PackageID, &c.ActivationDate,
			&c.DueDate, &c.ConnectionMethod, &c.PppoeUsername, &c.PppoePassword,
			&c.MacAddress, &c.RouterID, &c.OdpPort, &c.CreditBalance, &c.Notes, &c.Status,
			&c.DeletedAt, &c.CreatedAt, &c.UpdatedAt, &packageName,
		); err != nil {
			return nil, fmt.Errorf("repository: gagal scan customer row: %w", err)
		}
		customer := mapCustomerRow(c)
		customer.PackageName = packageName
		customers = append(customers, customer)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("repository: gagal iterasi customer rows: %w", err)
	}

	totalPages := int(math.Ceil(float64(total) / float64(params.PageSize)))
	if totalPages < 1 {
		totalPages = 1
	}

	return &domain.CustomerListResult{
		Data: customers,
		Pagination: domain.PaginationMeta{
			Total:      total,
			Page:       params.Page,
			PageSize:   params.PageSize,
			TotalPages: totalPages,
		},
	}, nil
}

// UpdateStatus memperbarui status customer dan mengembalikan customer yang diperbarui.
func (r *CustomerRepo) UpdateStatus(ctx context.Context, id string, status domain.CustomerStatus) (*domain.Customer, error) {
	row, err := r.queries.UpdateCustomerStatus(ctx, UpdateCustomerStatusParams{
		ID:     stringToUUID(id),
		Status: string(status),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrCustomerNotFound
		}
		return nil, fmt.Errorf("repository: gagal memperbarui status customer: %w", err)
	}
	return mapCustomerRow(row), nil
}

// UpdatePackage memperbarui package_id customer dan mengembalikan customer yang diperbarui.
func (r *CustomerRepo) UpdatePackage(ctx context.Context, id string, packageID string) (*domain.Customer, error) {
	row, err := r.queries.UpdateCustomerPackage(ctx, UpdateCustomerPackageParams{
		ID:        stringToUUID(id),
		PackageID: stringToUUID(packageID),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrCustomerNotFound
		}
		return nil, fmt.Errorf("repository: gagal memperbarui package customer: %w", err)
	}
	return mapCustomerRow(row), nil
}

// CountByStatus mengembalikan jumlah customer per status.
func (r *CustomerRepo) CountByStatus(ctx context.Context) (map[domain.CustomerStatus]int64, error) {
	result := make(map[domain.CustomerStatus]int64)

	pending, err := r.queries.CountCustomersByStatusPending(ctx)
	if err != nil {
		return nil, fmt.Errorf("repository: gagal menghitung customer pending: %w", err)
	}
	result[domain.CustomerStatusPending] = pending

	aktif, err := r.queries.CountCustomersByStatusAktif(ctx)
	if err != nil {
		return nil, fmt.Errorf("repository: gagal menghitung customer aktif: %w", err)
	}
	result[domain.CustomerStatusAktif] = aktif

	isolir, err := r.queries.CountCustomersByStatusIsolir(ctx)
	if err != nil {
		return nil, fmt.Errorf("repository: gagal menghitung customer isolir: %w", err)
	}
	result[domain.CustomerStatusIsolir] = isolir

	suspend, err := r.queries.CountCustomersByStatusSuspend(ctx)
	if err != nil {
		return nil, fmt.Errorf("repository: gagal menghitung customer suspend: %w", err)
	}
	result[domain.CustomerStatusSuspend] = suspend

	berhenti, err := r.queries.CountCustomersByStatusBerhenti(ctx)
	if err != nil {
		return nil, fmt.Errorf("repository: gagal menghitung customer berhenti: %w", err)
	}
	result[domain.CustomerStatusBerhenti] = berhenti

	return result, nil
}

// GetMaxSeq mengembalikan sequence number tertinggi untuk customer_id_seq di tenant.
func (r *CustomerRepo) GetMaxSeq(ctx context.Context, tenantID string) (int, error) {
	maxSeq, err := r.queries.GetMaxCustomerSeq(ctx, stringToUUID(tenantID))
	if err != nil {
		return 0, fmt.Errorf("repository: gagal mengambil max customer seq: %w", err)
	}
	return int(maxSeq), nil
}

// PhoneExists mengecek apakah nomor telepon sudah terdaftar di tenant yang sama.
// excludeID digunakan untuk mengecualikan customer tertentu (saat perbarui).
func (r *CustomerRepo) PhoneExists(ctx context.Context, tenantID, phone, excludeID string) (bool, error) {
	// Jika excludeID kosong, gunakan UUID nil agar tidak mengecualikan siapapun
	exID := excludeID
	if exID == "" {
		exID = "00000000-0000-0000-0000-000000000000"
	}
	exists, err := r.queries.PhoneExists(ctx, PhoneExistsParams{
		TenantID: stringToUUID(tenantID),
		Phone:    phone,
		ID:       stringToUUID(exID),
	})
	if err != nil {
		return false, fmt.Errorf("repository: gagal mengecek phone exists: %w", err)
	}
	return exists, nil
}

// BulkUpdateStatus memperbarui status untuk beberapa customer sekaligus.
// Mengembalikan hasil per item (sukses/gagal).
func (r *CustomerRepo) BulkUpdateStatus(ctx context.Context, ids []string, status domain.CustomerStatus) ([]domain.BulkResult, error) {
	results := make([]domain.BulkResult, 0, len(ids))
	for _, id := range ids {
		_, err := r.queries.UpdateCustomerStatus(ctx, UpdateCustomerStatusParams{
			ID:     stringToUUID(id),
			Status: string(status),
		})
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

// BulkUpdatefield memperbarui field tertentu untuk beberapa customer sekaligus.
// field yang didukung: area_id, due_date, notes.
func (r *CustomerRepo) BulkUpdateFields(ctx context.Context, ids []string, fields map[string]interface{}) ([]domain.BulkResult, error) {
	results := make([]domain.BulkResult, 0, len(ids))

	for _, id := range ids {
		// Ambil customer saat ini
		current, err := r.queries.GetCustomerByID(ctx, stringToUUID(id))
		if err != nil {
			results = append(results, domain.BulkResult{
				ID:      id,
				Success: false,
				Error:   fmt.Errorf("gagal mengambil customer: %w", err),
			})
			continue
		}

		// Terapkan pembaruan field
		updateParams := UpdateCustomerParams{
			ID:               current.ID,
			Name:             current.Name,
			Phone:            current.Phone,
			Email:            current.Email,
			Address:          current.Address,
			AreaID:           current.AreaID,
			Latitude:         current.Latitude,
			Longitude:        current.Longitude,
			PackageID:        current.PackageID,
			ActivationDate:   current.ActivationDate,
			DueDate:          current.DueDate,
			ConnectionMethod: current.ConnectionMethod,
			PppoeUsername:    current.PppoeUsername,
			PppoePassword:    current.PppoePassword,
			MacAddress:       current.MacAddress,
			RouterID:         current.RouterID,
			OdpPort:          current.OdpPort,
			CreditBalance:    current.CreditBalance,
			Notes:            current.Notes,
		}

		// Terapkan penimpaan field
		if v, ok := fields["area_id"]; ok {
			if areaID, ok := v.(string); ok {
				updateParams.AreaID = stringToUUID(areaID)
			}
		}
		if v, ok := fields["due_date"]; ok {
			switch dd := v.(type) {
			case int:
				updateParams.DueDate = int32(dd)
			case float64:
				updateParams.DueDate = int32(dd)
			case json.Number:
				n, _ := dd.Int64()
				updateParams.DueDate = int32(n)
			}
		}
		if v, ok := fields["notes"]; ok {
			if notes, ok := v.(string); ok {
				updateParams.Notes = stringToText(notes)
			}
		}

		_, err = r.queries.UpdateCustomer(ctx, updateParams)
		if err != nil {
			results = append(results, domain.BulkResult{
				ID:      id,
				Success: false,
				Error:   fmt.Errorf("gagal update fields: %w", err),
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

// BulkSoftDelete melakukan hapus lunak untuk beberapa customer sekaligus.
func (r *CustomerRepo) BulkSoftDelete(ctx context.Context, ids []string) ([]domain.BulkResult, error) {
	results := make([]domain.BulkResult, 0, len(ids))
	for _, id := range ids {
		err := r.queries.SoftDeleteCustomer(ctx, stringToUUID(id))
		if err != nil {
			results = append(results, domain.BulkResult{
				ID:      id,
				Success: false,
				Error:   fmt.Errorf("gagal soft-delete: %w", err),
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

// GetByIDs mengambil beberapa customer berdasarkan daftar ID.
func (r *CustomerRepo) GetByIDs(ctx context.Context, ids []string) ([]*domain.Customer, error) {
	customers := make([]*domain.Customer, 0, len(ids))
	for _, id := range ids {
		row, err := r.queries.GetCustomerByID(ctx, stringToUUID(id))
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				continue // lewati jika tidak ditemukan
			}
			return nil, fmt.Errorf("repository: gagal mengambil customer by ID %s: %w", id, err)
		}
		customers = append(customers, mapCustomerRow(row))
	}
	return customers, nil
}

// SearchForPayment mencari pelanggan berdasarkan nama, customer_id_seq, atau telepon.
// Mengembalikan maksimal 10 hasil, hanya status aktif/isolir.
// Digunakan untuk alur pembayaran cepat.
func (r *CustomerRepo) SearchForPayment(ctx context.Context, tenantID, searchTerm string) ([]*domain.Customer, error) {
	searchPattern := "%" + searchTerm + "%"
	rows, err := r.queries.SearchCustomersForPayment(ctx, SearchCustomersForPaymentParams{
		TenantID: stringToUUID(tenantID),
		Name:     searchPattern,
	})
	if err != nil {
		return nil, fmt.Errorf("repository: gagal mencari customer untuk pembayaran: %w", err)
	}

	customers := make([]*domain.Customer, 0, len(rows))
	for _, row := range rows {
		customers = append(customers, mapCustomerRow(row))
	}
	return customers, nil
}
