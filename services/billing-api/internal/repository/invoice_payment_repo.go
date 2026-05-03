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

// InvoicePaymentRepo mengimplementasikan domain.InvoicePaymentRepository dengan membungkus
// sqlc-generated Queries dan pgxpool.Pool untuk dynamic list query.
type InvoicePaymentRepo struct {
	// queries adalah sqlc-generated Queries untuk operasi invoice payments.
	queries *Queries

	// pool digunakan untuk dynamic list query (raw SQL dengan pgx).
	pool *pgxpool.Pool
}

// NewInvoicePaymentRepo membuat instance baru InvoicePaymentRepo.
func NewInvoicePaymentRepo(queries *Queries, pool *pgxpool.Pool) *InvoicePaymentRepo {
	return &InvoicePaymentRepo{
		queries: queries,
		pool:    pool,
	}
}

// --- Helper function untuk mapping sqlc InvoicePayment → domain.InvoicePayment ---

// mapInvoicePaymentRow memetakan InvoicePayment (sqlc model) ke domain.InvoicePayment.
func mapInvoicePaymentRow(row InvoicePayment) *domain.InvoicePayment {
	return &domain.InvoicePayment{
		ID:              uuidToString(row.ID),
		TenantID:        uuidToString(row.TenantID),
		InvoiceID:       uuidToString(row.InvoiceID),
		Amount:          row.Amount,
		PaymentMethod:   row.PaymentMethod,
		PaymentDate:     dateToTime(row.PaymentDate),
		ReferenceNumber: textToString(row.ReferenceNumber),
		Notes:           textToString(row.Notes),
		RecordedByID:    uuidToString(row.RecordedByID),
		RecordedByName:  row.RecordedByName,
		Voided:          row.Voided,
		VoidedAt:        timestamptzToTimePtr(row.VoidedAt),
		VoidedBy:        textToString(row.VoidedBy),
		VoidReason:      textToString(row.VoidReason),
		CreatedAt:       timestamptzToTime(row.CreatedAt),
	}
}

// --- Implementasi domain.InvoicePaymentRepository ---

// Create membuat catatan pembayaran baru dan mengembalikan pembayaran yang dibuat.
func (r *InvoicePaymentRepo) Create(ctx context.Context, payment *domain.InvoicePayment) (*domain.InvoicePayment, error) {
	row, err := r.queries.CreateInvoicePayment(ctx, CreateInvoicePaymentParams{
		TenantID:        stringToUUID(payment.TenantID),
		InvoiceID:       stringToUUID(payment.InvoiceID),
		Amount:          payment.Amount,
		PaymentMethod:   payment.PaymentMethod,
		PaymentDate:     timeToDate(payment.PaymentDate),
		ReferenceNumber: stringToText(payment.ReferenceNumber),
		Notes:           stringToText(payment.Notes),
		RecordedByID:    stringToUUID(payment.RecordedByID),
		RecordedByName:  payment.RecordedByName,
	})
	if err != nil {
		return nil, fmt.Errorf("repository: gagal membuat invoice payment: %w", err)
	}
	return mapInvoicePaymentRow(row), nil
}

// ListByInvoice mengambil semua pembayaran non-void untuk invoice tertentu.
func (r *InvoicePaymentRepo) ListByInvoice(ctx context.Context, invoiceID string) ([]*domain.InvoicePayment, error) {
	rows, err := r.queries.ListPaymentsByInvoice(ctx, stringToUUID(invoiceID))
	if err != nil {
		return nil, fmt.Errorf("repository: gagal mengambil invoice payments: %w", err)
	}

	result := make([]*domain.InvoicePayment, 0, len(rows))
	for _, row := range rows {
		result = append(result, mapInvoicePaymentRow(row))
	}
	return result, nil
}

// VoidPayment menandai pembayaran sebagai void dengan alasan.
func (r *InvoicePaymentRepo) VoidPayment(ctx context.Context, id string, voidedBy string, reason string) error {
	err := r.queries.VoidPayment(ctx, VoidPaymentParams{
		ID:         stringToUUID(id),
		VoidedBy:   stringToText(voidedBy),
		VoidReason: stringToText(reason),
	})
	if err != nil {
		return fmt.Errorf("repository: gagal void payment: %w", err)
	}
	return nil
}

// GetByID mengambil satu pembayaran berdasarkan ID beserta nomor invoice melalui JOIN.
func (r *InvoicePaymentRepo) GetByID(ctx context.Context, id string) (*domain.InvoicePayment, error) {
	row, err := r.queries.GetPaymentByID(ctx, stringToUUID(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrPaymentNotFound
		}
		return nil, fmt.Errorf("repository: gagal mengambil payment by ID: %w", err)
	}
	return &domain.InvoicePayment{
		ID:              uuidToString(row.ID),
		TenantID:        uuidToString(row.TenantID),
		InvoiceID:       uuidToString(row.InvoiceID),
		Amount:          row.Amount,
		PaymentMethod:   row.PaymentMethod,
		PaymentDate:     dateToTime(row.PaymentDate),
		ReferenceNumber: textToString(row.ReferenceNumber),
		Notes:           textToString(row.Notes),
		RecordedByID:    uuidToString(row.RecordedByID),
		RecordedByName:  row.RecordedByName,
		Voided:          row.Voided,
		VoidedAt:        timestamptzToTimePtr(row.VoidedAt),
		VoidedBy:        textToString(row.VoidedBy),
		VoidReason:      textToString(row.VoidReason),
		CreatedAt:       timestamptzToTime(row.CreatedAt),
	}, nil
}

// ListWithFilters mengambil daftar pembayaran dengan dynamic filtering, search, dan pagination.
// Menggunakan raw SQL karena sqlc tidak mendukung dynamic WHERE clause.
// Query melakukan JOIN ke tabel invoices dan customers untuk mendapatkan
// invoice_number, customer_name, dan customer_id_seq.
func (r *InvoicePaymentRepo) ListWithFilters(ctx context.Context, params domain.PaymentListParams) (*domain.PaymentListResult, error) {
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
	conditions = append(conditions, fmt.Sprintf("ip.tenant_id = $%d", argIdx))
	args = append(args, stringToUUID(params.TenantID))
	argIdx++

	// Include voided filter — default hanya non-voided
	if !params.IncludeVoided {
		conditions = append(conditions, "ip.voided = false")
	}

	// Payment method filter
	if params.PaymentMethod != "" {
		conditions = append(conditions, fmt.Sprintf("ip.payment_method = $%d", argIdx))
		args = append(args, params.PaymentMethod)
		argIdx++
	}

	// Date from filter
	if params.DateFrom != "" {
		dateFrom, err := time.Parse("2006-01-02", params.DateFrom)
		if err == nil {
			conditions = append(conditions, fmt.Sprintf("ip.payment_date >= $%d", argIdx))
			args = append(args, timeToDate(dateFrom))
			argIdx++
		}
	}

	// Date to filter
	if params.DateTo != "" {
		dateTo, err := time.Parse("2006-01-02", params.DateTo)
		if err == nil {
			conditions = append(conditions, fmt.Sprintf("ip.payment_date <= $%d", argIdx))
			args = append(args, timeToDate(dateTo))
			argIdx++
		}
	}

	// Recorded by filter
	if params.RecordedBy != "" {
		conditions = append(conditions, fmt.Sprintf("ip.recorded_by_id = $%d", argIdx))
		args = append(args, stringToUUID(params.RecordedBy))
		argIdx++
	}

	// Search filter (case-insensitive ILIKE pada customer name, customer_id_seq, invoice_number)
	if params.Search != "" {
		searchPattern := "%" + params.Search + "%"
		conditions = append(conditions, fmt.Sprintf(
			"(c.name ILIKE $%d OR c.customer_id_seq ILIKE $%d OR i.invoice_number ILIKE $%d)",
			argIdx, argIdx, argIdx,
		))
		args = append(args, searchPattern)
		argIdx++
	}

	whereClause := "WHERE " + strings.Join(conditions, " AND ")

	// Count total — menggunakan JOIN yang sama untuk filter yang benar
	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM invoice_payments ip
		JOIN invoices i ON i.id = ip.invoice_id
		JOIN customers c ON c.id = i.customer_id
		%s`, whereClause)
	var total int64
	err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("repository: gagal menghitung total payment: %w", err)
	}

	// Build pagination
	offset := (params.Page - 1) * params.PageSize

	// Build data query dengan JOIN ke invoices dan customers
	dataQuery := fmt.Sprintf(`SELECT
		ip.id, ip.invoice_id, i.invoice_number,
		c.name AS customer_name, c.customer_id_seq,
		ip.amount, ip.payment_method, ip.payment_date,
		ip.reference_number, ip.receipt_number,
		ip.recorded_by_name, ip.voided, ip.void_reason,
		ip.proof_image_url, ip.created_at
		FROM invoice_payments ip
		JOIN invoices i ON i.id = ip.invoice_id
		JOIN customers c ON c.id = i.customer_id
		%s
		ORDER BY ip.payment_date DESC, ip.created_at DESC
		LIMIT $%d OFFSET $%d`,
		whereClause, argIdx, argIdx+1,
	)
	args = append(args, params.PageSize, offset)

	rows, err := r.pool.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("repository: gagal mengambil daftar payment: %w", err)
	}
	defer rows.Close()

	items := make([]domain.PaymentListItem, 0)
	for rows.Next() {
		var (
			id              pgtype.UUID
			invoiceID       pgtype.UUID
			invoiceNumber   string
			customerName    string
			customerIDSeq   pgtype.Text
			amount          int64
			paymentMethod   string
			paymentDate     pgtype.Date
			referenceNumber pgtype.Text
			receiptNumber   pgtype.Text
			recordedByName  string
			voided          bool
			voidReason      pgtype.Text
			proofImageURL   pgtype.Text
			createdAt       pgtype.Timestamptz
		)

		if err := rows.Scan(
			&id, &invoiceID, &invoiceNumber,
			&customerName, &customerIDSeq,
			&amount, &paymentMethod, &paymentDate,
			&referenceNumber, &receiptNumber,
			&recordedByName, &voided, &voidReason,
			&proofImageURL, &createdAt,
		); err != nil {
			return nil, fmt.Errorf("repository: gagal scan payment row: %w", err)
		}

		items = append(items, domain.PaymentListItem{
			ID:              uuidToString(id),
			InvoiceID:       uuidToString(invoiceID),
			InvoiceNumber:   invoiceNumber,
			CustomerName:    customerName,
			CustomerIDSeq:   textToString(customerIDSeq),
			Amount:          amount,
			PaymentMethod:   paymentMethod,
			PaymentDate:     dateToTime(paymentDate),
			ReferenceNumber: textToString(referenceNumber),
			ReceiptNumber:   textToString(receiptNumber),
			RecordedByName:  recordedByName,
			Voided:          voided,
			VoidReason:      textToString(voidReason),
			ProofImageURL:   textToString(proofImageURL),
			CreatedAt:       timestamptzToTime(createdAt),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("repository: gagal iterasi payment rows: %w", err)
	}

	totalPages := int(math.Ceil(float64(total) / float64(params.PageSize)))
	if totalPages < 1 {
		totalPages = 1
	}

	return &domain.PaymentListResult{
		Data: items,
		Pagination: domain.PaginationMeta{
			Total:      total,
			Page:       params.Page,
			PageSize:   params.PageSize,
			TotalPages: totalPages,
		},
	}, nil
}

// GetSummary mengambil statistik pembayaran agregat untuk tenant.
// Memanggil sqlc queries untuk today, month, dan by_method lalu merakit PaymentSummary.
func (r *InvoicePaymentRepo) GetSummary(ctx context.Context, tenantID string, timezone string, periodMonth, periodYear *int) (*domain.PaymentSummary, error) {
	// Tentukan bulan dan tahun — default ke bulan/tahun saat ini
	now := time.Now()
	month := int(now.Month())
	year := now.Year()
	if periodMonth != nil {
		month = *periodMonth
	}
	if periodYear != nil {
		year = *periodYear
	}

	// Ambil ringkasan hari ini — timezone dikirim sebagai string ke AT TIME ZONE
	if timezone == "" {
		timezone = "Asia/Jakarta"
	}
	// sqlc meng-generate parameter sebagai pgtype.Interval tapi SQL menggunakan
	// AT TIME ZONE $1 yang menerima string, jadi gunakan raw query untuk today.
	var todayStat domain.PaymentSummaryStat
	todayQuery := `SELECT COUNT(*)::bigint AS count, COALESCE(SUM(amount), 0)::bigint AS total_amount
		FROM invoice_payments
		WHERE voided = false
		  AND (created_at AT TIME ZONE $1)::date = (NOW() AT TIME ZONE $1)::date`
	err := r.pool.QueryRow(ctx, todayQuery, timezone).Scan(&todayStat.Count, &todayStat.TotalAmount)
	if err != nil {
		return nil, fmt.Errorf("repository: gagal mengambil payment summary today: %w", err)
	}

	// Ambil ringkasan bulan ini — sqlc meng-generate parameter sebagai pgtype.Date
	// tapi SQL menggunakan EXTRACT(MONTH/YEAR) yang membandingkan dengan integer,
	// jadi gunakan raw query untuk konsistensi.
	var monthStat domain.PaymentSummaryStat
	monthQuery := `SELECT COUNT(*)::bigint AS count, COALESCE(SUM(amount), 0)::bigint AS total_amount
		FROM invoice_payments
		WHERE voided = false
		  AND EXTRACT(MONTH FROM payment_date) = $1
		  AND EXTRACT(YEAR FROM payment_date) = $2`
	err = r.pool.QueryRow(ctx, monthQuery, month, year).Scan(&monthStat.Count, &monthStat.TotalAmount)
	if err != nil {
		return nil, fmt.Errorf("repository: gagal mengambil payment summary month: %w", err)
	}

	// Ambil ringkasan per metode pembayaran
	byMethodQuery := `SELECT payment_method, COUNT(*)::bigint AS count, COALESCE(SUM(amount), 0)::bigint AS total_amount
		FROM invoice_payments
		WHERE voided = false
		  AND EXTRACT(MONTH FROM payment_date) = $1
		  AND EXTRACT(YEAR FROM payment_date) = $2
		GROUP BY payment_method`
	methodRows, err := r.pool.Query(ctx, byMethodQuery, month, year)
	if err != nil {
		return nil, fmt.Errorf("repository: gagal mengambil payment summary by method: %w", err)
	}
	defer methodRows.Close()

	byMethod := make(map[string]domain.PaymentSummaryStat)
	for methodRows.Next() {
		var (
			paymentMethod string
			count         int64
			totalAmount   int64
		)
		if err := methodRows.Scan(&paymentMethod, &count, &totalAmount); err != nil {
			return nil, fmt.Errorf("repository: gagal scan payment summary by method row: %w", err)
		}
		byMethod[paymentMethod] = domain.PaymentSummaryStat{
			Count:       count,
			TotalAmount: totalAmount,
		}
	}
	if err := methodRows.Err(); err != nil {
		return nil, fmt.Errorf("repository: gagal iterasi payment summary by method rows: %w", err)
	}

	return &domain.PaymentSummary{
		Today:     todayStat,
		ThisMonth: monthStat,
		ByMethod:  byMethod,
	}, nil
}

// FindDuplicate mengecek potensi duplikasi pembayaran dalam 24 jam terakhir.
// Duplikat didefinisikan sebagai pembayaran dengan customer_id, amount, payment_method,
// dan payment_date yang sama, belum di-void.
func (r *InvoicePaymentRepo) FindDuplicate(ctx context.Context, customerID string, amount int64, method string, paymentDate time.Time) (bool, error) {
	exists, err := r.queries.FindDuplicatePayment(ctx, FindDuplicatePaymentParams{
		CustomerID:    stringToUUID(customerID),
		Amount:        amount,
		PaymentMethod: method,
		PaymentDate:   timeToDate(paymentDate),
	})
	if err != nil {
		return false, fmt.Errorf("repository: gagal mengecek duplicate payment: %w", err)
	}
	return exists, nil
}
