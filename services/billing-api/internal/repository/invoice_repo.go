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

// InvoiceRepo mengimplementasikan domain.InvoiceRepository dengan membungkus
// sqlc-generated Queries dan pgxpool.Pool untuk dynamic list query.
type InvoiceRepo struct {
	// queries adalah sqlc-generated Queries untuk operasi invoice.
	queries *Queries

	// pool digunakan untuk dynamic list query (raw SQL dengan pgx).
	pool *pgxpool.Pool
}

// NewInvoiceRepo membuat instance baru InvoiceRepo.
func NewInvoiceRepo(queries *Queries, pool *pgxpool.Pool) *InvoiceRepo {
	return &InvoiceRepo{
		queries: queries,
		pool:    pool,
	}
}

// --- Helper function untuk mapping sqlc Invoice → domain.Invoice ---

// mapInvoiceRow memetakan Invoice (sqlc model) ke domain.Invoice.
func mapInvoiceRow(row Invoice) *domain.Invoice {
	return &domain.Invoice{
		ID:             uuidToString(row.ID),
		TenantID:       uuidToString(row.TenantID),
		CustomerID:     uuidToString(row.CustomerID),
		InvoiceNumber:  row.InvoiceNumber,
		PeriodMonth:    int(row.PeriodMonth),
		PeriodYear:     int(row.PeriodYear),
		DueDate:        dateToTime(row.DueDate),
		Subtotal:       row.Subtotal,
		TaxAmount:      row.TaxAmount,
		PenaltyAmount:  row.PenaltyAmount,
		DiscountAmount: row.DiscountAmount,
		CreditApplied:  row.CreditApplied,
		TotalAmount:    row.TotalAmount,
		PaidAmount:     row.PaidAmount,
		Status:         domain.InvoiceStatus(row.Status),
		Notes:          textToString(row.Notes),
		IsPrepaid:      row.IsPrepaid,
		PrepaidMonths:  int4ToIntPtr(row.PrepaidMonths),
		Version:        int(row.Version),
		CreatedAt:      timestamptzToTime(row.CreatedAt),
		UpdatedAt:      timestamptzToTime(row.UpdatedAt),
	}
}

// mapGetInvoiceByIDRow memetakan GetInvoiceByIDRow (sqlc JOIN result) ke domain.Invoice.
func mapGetInvoiceByIDRow(row GetInvoiceByIDRow) *domain.Invoice {
	return &domain.Invoice{
		ID:              uuidToString(row.ID),
		TenantID:        uuidToString(row.TenantID),
		CustomerID:      uuidToString(row.CustomerID),
		InvoiceNumber:   row.InvoiceNumber,
		PeriodMonth:     int(row.PeriodMonth),
		PeriodYear:      int(row.PeriodYear),
		DueDate:         dateToTime(row.DueDate),
		Subtotal:        row.Subtotal,
		TaxAmount:       row.TaxAmount,
		PenaltyAmount:   row.PenaltyAmount,
		DiscountAmount:  row.DiscountAmount,
		CreditApplied:   row.CreditApplied,
		TotalAmount:     row.TotalAmount,
		PaidAmount:      row.PaidAmount,
		Status:          domain.InvoiceStatus(row.Status),
		Notes:           textToString(row.Notes),
		IsPrepaid:       row.IsPrepaid,
		PrepaidMonths:   int4ToIntPtr(row.PrepaidMonths),
		Version:         int(row.Version),
		CreatedAt:       timestamptzToTime(row.CreatedAt),
		UpdatedAt:       timestamptzToTime(row.UpdatedAt),
		CustomerName:    row.CustomerName,
		CustomerIDSeq:   textToString(row.CustomerIDSeq),
		CustomerPhone:   row.CustomerPhone,
		CustomerAddress: row.CustomerAddress,
		PackageName:     textToString(row.PackageName),
	}
}

// --- Implementasi domain.InvoiceRepository ---

// Create membuat invoice baru dan mengembalikan invoice yang dibuat.
func (r *InvoiceRepo) Create(ctx context.Context, invoice *domain.Invoice) (*domain.Invoice, error) {
	row, err := r.queries.CreateInvoice(ctx, CreateInvoiceParams{
		TenantID:       stringToUUID(invoice.TenantID),
		CustomerID:     stringToUUID(invoice.CustomerID),
		InvoiceNumber:  invoice.InvoiceNumber,
		PeriodMonth:    int32(invoice.PeriodMonth),
		PeriodYear:     int32(invoice.PeriodYear),
		DueDate:        timeToDate(invoice.DueDate),
		Subtotal:       invoice.Subtotal,
		TaxAmount:      invoice.TaxAmount,
		PenaltyAmount:  invoice.PenaltyAmount,
		DiscountAmount: invoice.DiscountAmount,
		CreditApplied:  invoice.CreditApplied,
		TotalAmount:    invoice.TotalAmount,
		PaidAmount:     invoice.PaidAmount,
		Status:         string(invoice.Status),
		Notes:          stringToText(invoice.Notes),
		IsPrepaid:      invoice.IsPrepaid,
		PrepaidMonths:  intPtrToInt4(invoice.PrepaidMonths),
		Version:        int32(invoice.Version),
	})
	if err != nil {
		return nil, fmt.Errorf("repository: gagal membuat invoice: %w", err)
	}
	return mapInvoiceRow(row), nil
}

// GetByID mengambil invoice berdasarkan ID (dengan JOIN ke customers dan packages).
func (r *InvoiceRepo) GetByID(ctx context.Context, id string) (*domain.Invoice, error) {
	row, err := r.queries.GetInvoiceByID(ctx, stringToUUID(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrInvoiceNotFound
		}
		return nil, fmt.Errorf("repository: gagal mengambil invoice by ID: %w", err)
	}
	return mapGetInvoiceByIDRow(row), nil
}

// Update memperbarui data invoice dan mengembalikan invoice yang diperbarui.
func (r *InvoiceRepo) Update(ctx context.Context, invoice *domain.Invoice) (*domain.Invoice, error) {
	row, err := r.queries.UpdateInvoice(ctx, UpdateInvoiceParams{
		ID:             stringToUUID(invoice.ID),
		DueDate:        timeToDate(invoice.DueDate),
		Subtotal:       invoice.Subtotal,
		TaxAmount:      invoice.TaxAmount,
		PenaltyAmount:  invoice.PenaltyAmount,
		DiscountAmount: invoice.DiscountAmount,
		CreditApplied:  invoice.CreditApplied,
		TotalAmount:    invoice.TotalAmount,
		Notes:          stringToText(invoice.Notes),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrInvoiceNotFound
		}
		return nil, fmt.Errorf("repository: gagal memperbarui invoice: %w", err)
	}
	return mapInvoiceRow(row), nil
}

// UpdateStatus memperbarui status invoice dengan optimistic locking via version.
// Mengembalikan error jika version tidak cocok (concurrent modification).
func (r *InvoiceRepo) UpdateStatus(ctx context.Context, id string, status domain.InvoiceStatus, version int) (*domain.Invoice, error) {
	row, err := r.queries.UpdateInvoiceStatus(ctx, UpdateInvoiceStatusParams{
		ID:      stringToUUID(id),
		Status:  string(status),
		Version: int32(version),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrInvoiceNotFound
		}
		return nil, fmt.Errorf("repository: gagal memperbarui status invoice: %w", err)
	}
	return mapInvoiceRow(row), nil
}

// UpdatePaidAmount memperbarui jumlah yang sudah dibayar dengan optimistic locking via version.
// Mengembalikan error jika version tidak cocok (concurrent modification).
func (r *InvoiceRepo) UpdatePaidAmount(ctx context.Context, id string, paidAmount int64, version int) (*domain.Invoice, error) {
	row, err := r.queries.UpdateInvoicePaidAmount(ctx, UpdateInvoicePaidAmountParams{
		ID:         stringToUUID(id),
		PaidAmount: paidAmount,
		Version:    int32(version),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrInvoiceNotFound
		}
		return nil, fmt.Errorf("repository: gagal memperbarui paid_amount invoice: %w", err)
	}
	return mapInvoiceRow(row), nil
}

// allowedInvoiceSortColumns adalah whitelist kolom yang diizinkan untuk sorting invoice.
// Mencegah SQL injection pada ORDER BY clause.
var allowedInvoiceSortColumns = map[string]string{
	"invoice_number": "i.invoice_number",
	"due_date":       "i.due_date",
	"total_amount":   "i.total_amount",
	"status":         "i.status",
	"created_at":     "i.created_at",
}

// List mengambil daftar invoice dengan dynamic filtering, search, sorting, dan pagination.
// Menggunakan raw SQL karena sqlc tidak mendukung dynamic WHERE clause.
// Query melakukan JOIN ke tabel customers dan packages untuk mendapatkan
// customer_name, customer_id_seq, dan package_name.
func (r *InvoiceRepo) List(ctx context.Context, params domain.InvoiceListParams) (*domain.InvoiceListResult, error) {
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
	conditions = append(conditions, fmt.Sprintf("i.tenant_id = $%d", argIdx))
	args = append(args, stringToUUID(params.TenantID))
	argIdx++

	// Status filter
	if params.Status != "" {
		conditions = append(conditions, fmt.Sprintf("i.status = $%d", argIdx))
		args = append(args, params.Status)
		argIdx++
	}

	// Customer filter
	if params.CustomerID != "" {
		conditions = append(conditions, fmt.Sprintf("i.customer_id = $%d", argIdx))
		args = append(args, stringToUUID(params.CustomerID))
		argIdx++
	}

	// Period month filter
	if params.PeriodMonth != nil {
		conditions = append(conditions, fmt.Sprintf("i.period_month = $%d", argIdx))
		args = append(args, *params.PeriodMonth)
		argIdx++
	}

	// Period year filter
	if params.PeriodYear != nil {
		conditions = append(conditions, fmt.Sprintf("i.period_year = $%d", argIdx))
		args = append(args, *params.PeriodYear)
		argIdx++
	}

	// Package ID filter (via customer JOIN)
	if params.PackageID != "" {
		conditions = append(conditions, fmt.Sprintf("c.package_id = $%d", argIdx))
		args = append(args, stringToUUID(params.PackageID))
		argIdx++
	}

	// Area ID filter (via customer JOIN)
	if params.AreaID != "" {
		conditions = append(conditions, fmt.Sprintf("c.area_id = $%d", argIdx))
		args = append(args, stringToUUID(params.AreaID))
		argIdx++
	}

	// Search filter (case-insensitive ILIKE pada invoice_number, customer name, customer_id_seq)
	if params.Search != "" {
		searchPattern := "%" + params.Search + "%"
		conditions = append(conditions, fmt.Sprintf(
			"(i.invoice_number ILIKE $%d OR c.name ILIKE $%d OR c.customer_id_seq ILIKE $%d)",
			argIdx, argIdx, argIdx,
		))
		args = append(args, searchPattern)
		argIdx++
	}

	whereClause := "WHERE " + strings.Join(conditions, " AND ")

	// Count total — menggunakan JOIN yang sama untuk filter yang benar
	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM invoices i
		JOIN customers c ON c.id = i.customer_id
		%s`, whereClause)
	var total int64
	err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("repository: gagal menghitung total invoice: %w", err)
	}

	// Build ORDER BY
	orderBy := "i.created_at"
	if params.SortBy != "" {
		if col, ok := allowedInvoiceSortColumns[params.SortBy]; ok {
			orderBy = col
		}
	}
	sortOrder := "DESC"
	if strings.EqualFold(params.SortOrder, "asc") {
		sortOrder = "ASC"
	}

	// Build pagination
	offset := (params.Page - 1) * params.PageSize

	// Build data query dengan JOIN ke customers dan packages
	dataQuery := fmt.Sprintf(`SELECT
		i.id, i.tenant_id, i.customer_id, i.invoice_number, i.period_month, i.period_year,
		i.due_date, i.subtotal, i.tax_amount, i.penalty_amount, i.discount_amount,
		i.credit_applied, i.total_amount, i.paid_amount, i.status, i.notes,
		i.is_prepaid, i.prepaid_months, i.version, i.created_at, i.updated_at,
		c.name AS customer_name,
		c.customer_id_seq AS customer_id_seq,
		COALESCE(p.name, '') AS package_name
		FROM invoices i
		JOIN customers c ON c.id = i.customer_id
		LEFT JOIN packages p ON p.id = c.package_id
		%s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d`,
		whereClause, orderBy, sortOrder, argIdx, argIdx+1,
	)
	args = append(args, params.PageSize, offset)

	rows, err := r.pool.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("repository: gagal mengambil daftar invoice: %w", err)
	}
	defer rows.Close()

	invoices := make([]*domain.Invoice, 0)
	for rows.Next() {
		var (
			id             pgtype.UUID
			tenantID       pgtype.UUID
			customerID     pgtype.UUID
			invoiceNumber  string
			periodMonth    int32
			periodYear     int32
			dueDate        pgtype.Date
			subtotal       int64
			taxAmount      int64
			penaltyAmount  int64
			discountAmount int64
			creditApplied  int64
			totalAmount    int64
			paidAmount     int64
			status         string
			notes          pgtype.Text
			isPrepaid      bool
			prepaidMonths  pgtype.Int4
			version        int32
			createdAt      pgtype.Timestamptz
			updatedAt      pgtype.Timestamptz
			customerName   string
			customerIDSeq  pgtype.Text
			packageName    string
		)

		if err := rows.Scan(
			&id, &tenantID, &customerID, &invoiceNumber, &periodMonth, &periodYear,
			&dueDate, &subtotal, &taxAmount, &penaltyAmount, &discountAmount,
			&creditApplied, &totalAmount, &paidAmount, &status, &notes,
			&isPrepaid, &prepaidMonths, &version, &createdAt, &updatedAt,
			&customerName, &customerIDSeq, &packageName,
		); err != nil {
			return nil, fmt.Errorf("repository: gagal scan invoice row: %w", err)
		}

		invoices = append(invoices, &domain.Invoice{
			ID:             uuidToString(id),
			TenantID:       uuidToString(tenantID),
			CustomerID:     uuidToString(customerID),
			InvoiceNumber:  invoiceNumber,
			PeriodMonth:    int(periodMonth),
			PeriodYear:     int(periodYear),
			DueDate:        dateToTime(dueDate),
			Subtotal:       subtotal,
			TaxAmount:      taxAmount,
			PenaltyAmount:  penaltyAmount,
			DiscountAmount: discountAmount,
			CreditApplied:  creditApplied,
			TotalAmount:    totalAmount,
			PaidAmount:     paidAmount,
			Status:         domain.InvoiceStatus(status),
			Notes:          textToString(notes),
			IsPrepaid:      isPrepaid,
			PrepaidMonths:  int4ToIntPtr(prepaidMonths),
			Version:        int(version),
			CreatedAt:      timestamptzToTime(createdAt),
			UpdatedAt:      timestamptzToTime(updatedAt),
			CustomerName:   customerName,
			CustomerIDSeq:  textToString(customerIDSeq),
			PackageName:    packageName,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("repository: gagal iterasi invoice rows: %w", err)
	}

	totalPages := int(math.Ceil(float64(total) / float64(params.PageSize)))
	if totalPages < 1 {
		totalPages = 1
	}

	return &domain.InvoiceListResult{
		Data: invoices,
		Pagination: domain.PaginationMeta{
			Total:      total,
			Page:       params.Page,
			PageSize:   params.PageSize,
			TotalPages: totalPages,
		},
	}, nil
}

// ExistsForPeriod mengecek apakah invoice sudah ada untuk customer dan periode tertentu.
// Digunakan untuk idempotency check saat auto-generate invoice.
func (r *InvoiceRepo) ExistsForPeriod(ctx context.Context, customerID string, month, year int) (bool, error) {
	exists, err := r.queries.ExistsForPeriod(ctx, ExistsForPeriodParams{
		CustomerID:  stringToUUID(customerID),
		PeriodMonth: int32(month),
		PeriodYear:  int32(year),
	})
	if err != nil {
		return false, fmt.Errorf("repository: gagal mengecek exists for period: %w", err)
	}
	return exists, nil
}

// ExistsForPeriodPrepaid mengecek apakah invoice prepaid sudah mencakup periode tertentu.
// Invoice prepaid mencakup beberapa bulan mulai dari period_month/period_year.
func (r *InvoiceRepo) ExistsForPeriodPrepaid(ctx context.Context, customerID string, month, year int) (bool, error) {
	exists, err := r.queries.ExistsForPeriodPrepaid(ctx, ExistsForPeriodPrepaidParams{
		CustomerID: stringToUUID(customerID),
		Column2:    int32(month),
		Column3:    int32(year),
	})
	if err != nil {
		return false, fmt.Errorf("repository: gagal mengecek exists for period prepaid: %w", err)
	}
	return exists, nil
}

// FindOverdue mengambil semua invoice yang sudah melewati jatuh tempo (status belum_bayar).
// Digunakan oleh cron job untuk update status ke terlambat.
func (r *InvoiceRepo) FindOverdue(ctx context.Context, currentDate time.Time) ([]*domain.Invoice, error) {
	rows, err := r.queries.FindOverdueInvoices(ctx, timeToDate(currentDate))
	if err != nil {
		return nil, fmt.Errorf("repository: gagal mengambil invoice overdue: %w", err)
	}
	invoices := make([]*domain.Invoice, 0, len(rows))
	for _, row := range rows {
		invoices = append(invoices, mapInvoiceRow(row))
	}
	return invoices, nil
}

// GetSummary mengambil ringkasan invoice per status untuk dashboard.
// Mendukung filter opsional berdasarkan period_month dan period_year.
func (r *InvoiceRepo) GetSummary(ctx context.Context, tenantID string, periodMonth, periodYear *int) (*domain.InvoiceSummary, error) {
	// Konversi parameter opsional — sqlc menggunakan int32 dengan 0 sebagai NULL
	var pm, py int32
	if periodMonth != nil {
		pm = int32(*periodMonth)
	}
	if periodYear != nil {
		py = int32(*periodYear)
	}

	rows, err := r.queries.GetInvoiceSummary(ctx, GetInvoiceSummaryParams{
		TenantID: stringToUUID(tenantID),
		Column2:  pm,
		Column3:  py,
	})
	if err != nil {
		return nil, fmt.Errorf("repository: gagal mengambil summary invoice: %w", err)
	}

	summary := &domain.InvoiceSummary{
		ByStatus: make(map[domain.InvoiceStatus]domain.InvoiceSummaryStat),
	}

	var totalCount int64
	var totalAmount int64

	for _, row := range rows {
		// Konversi total_amount dari interface{} ke int64
		var amount int64
		switch v := row.TotalAmount.(type) {
		case int64:
			amount = v
		case float64:
			amount = int64(v)
		}

		stat := domain.InvoiceSummaryStat{
			Count:       row.Count,
			TotalAmount: amount,
		}
		summary.ByStatus[domain.InvoiceStatus(row.Status)] = stat
		totalCount += row.Count
		totalAmount += amount
	}

	summary.Total = domain.InvoiceSummaryStat{
		Count:       totalCount,
		TotalAmount: totalAmount,
	}

	return summary, nil
}

// GetByIDs mengambil beberapa invoice berdasarkan daftar ID.
// Digunakan untuk bulk actions (reminder, cancel, PDF).
func (r *InvoiceRepo) GetByIDs(ctx context.Context, ids []string) ([]*domain.Invoice, error) {
	uuids := make([]pgtype.UUID, 0, len(ids))
	for _, id := range ids {
		uuids = append(uuids, stringToUUID(id))
	}

	rows, err := r.queries.GetInvoicesByIDs(ctx, uuids)
	if err != nil {
		return nil, fmt.Errorf("repository: gagal mengambil invoice by IDs: %w", err)
	}

	invoices := make([]*domain.Invoice, 0, len(rows))
	for _, row := range rows {
		invoices = append(invoices, mapInvoiceRow(row))
	}
	return invoices, nil
}

// FindOpenByCustomer mengambil semua invoice terbuka untuk customer, urut berdasarkan due_date ASC.
// Terbuka = status in (belum_bayar, terlambat, bayar_sebagian).
func (r *InvoiceRepo) FindOpenByCustomer(ctx context.Context, customerID string) ([]*domain.Invoice, error) {
	rows, err := r.queries.FindOpenInvoicesByCustomer(ctx, stringToUUID(customerID))
	if err != nil {
		return nil, fmt.Errorf("repository: gagal mengambil open invoices by customer: %w", err)
	}
	invoices := make([]*domain.Invoice, 0, len(rows))
	for _, row := range rows {
		invoices = append(invoices, mapInvoiceRow(row))
	}
	return invoices, nil
}

// FindOpenByCustomerForUpdate sama seperti FindOpenByCustomer tapi dengan SELECT FOR UPDATE.
// Harus dipanggil dalam transaksi untuk keamanan konkurensi.
func (r *InvoiceRepo) FindOpenByCustomerForUpdate(ctx context.Context, customerID string) ([]*domain.Invoice, error) {
	rows, err := r.queries.FindOpenInvoicesByCustomerForUpdate(ctx, stringToUUID(customerID))
	if err != nil {
		return nil, fmt.Errorf("repository: gagal mengambil open invoices for update by customer: %w", err)
	}
	invoices := make([]*domain.Invoice, 0, len(rows))
	for _, row := range rows {
		invoices = append(invoices, mapInvoiceRow(row))
	}
	return invoices, nil
}

// GetByIDsForUpdate mengambil invoice berdasarkan ID dengan SELECT FOR UPDATE.
// Harus dipanggil dalam transaksi untuk keamanan konkurensi.
func (r *InvoiceRepo) GetByIDsForUpdate(ctx context.Context, ids []string) ([]*domain.Invoice, error) {
	uuids := make([]pgtype.UUID, 0, len(ids))
	for _, id := range ids {
		uuids = append(uuids, stringToUUID(id))
	}

	rows, err := r.queries.GetInvoicesByIDsForUpdate(ctx, uuids)
	if err != nil {
		return nil, fmt.Errorf("repository: gagal mengambil invoices for update by IDs: %w", err)
	}
	invoices := make([]*domain.Invoice, 0, len(rows))
	for _, row := range rows {
		invoices = append(invoices, mapInvoiceRow(row))
	}
	return invoices, nil
}
