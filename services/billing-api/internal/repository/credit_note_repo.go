package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// =============================================================================
// CreditNoteRepo - implementasi domain.CreditNoteRepository
// =============================================================================

// CreditNoteRepo mengimplementasikan domain.CreditNoteRepository menggunakan raw SQL
// karena tabel credit_notes belum memiliki sqlc queries.
type CreditNoteRepo struct {
	// pool digunakan untuk raw SQL queries dengan pgx.
	pool *pgxpool.Pool
}

// NewCreditNoteRepo membuat instance baru CreditNoteRepo.
func NewCreditNoteRepo(pool *pgxpool.Pool) *CreditNoteRepo {
	return &CreditNoteRepo{
		pool: pool,
	}
}

// Buat membuat credit note baru dan mengembalikan credit note yang dibuat.
func (r *CreditNoteRepo) Create(ctx context.Context, cn *domain.CreditNote) (*domain.CreditNote, error) {
	query := `INSERT INTO credit_notes (
		tenant_id, credit_note_number, invoice_id, amount, reason,
		apply_to_credit, created_by_id, created_by_name
	) VALUES (
		$1, $2, $3, $4, $5,
		$6, $7, $8
	)
	RETURNING id, tenant_id, credit_note_number, invoice_id, amount, reason,
		apply_to_credit, created_by_id, created_by_name, created_at`

	var (
		id               pgtype.UUID
		tenantID         pgtype.UUID
		creditNoteNumber string
		invoiceID        pgtype.UUID
		amount           int64
		reason           string
		applyToCredit    bool
		createdByID      string
		createdByName    string
		createdAt        pgtype.Timestamptz
	)

	err := r.pool.QueryRow(ctx, query,
		stringToUUID(cn.TenantID),
		cn.CreditNoteNumber,
		stringToUUID(cn.InvoiceID),
		cn.Amount,
		cn.Reason,
		cn.ApplyToCredit,
		cn.CreatedByID,
		cn.CreatedByName,
	).Scan(
		&id, &tenantID, &creditNoteNumber, &invoiceID, &amount, &reason,
		&applyToCredit, &createdByID, &createdByName, &createdAt,
	)
	if err != nil {
		return nil, fmt.Errorf("repository: gagal membuat credit note: %w", err)
	}

	return &domain.CreditNote{
		ID:               uuidToString(id),
		TenantID:         uuidToString(tenantID),
		CreditNoteNumber: creditNoteNumber,
		InvoiceID:        uuidToString(invoiceID),
		Amount:           amount,
		Reason:           reason,
		ApplyToCredit:    applyToCredit,
		CreatedByID:      createdByID,
		CreatedByName:    createdByName,
		CreatedAt:        timestamptzToTime(createdAt),
	}, nil
}

// GetByID mengambil credit note berdasarkan ID.
// Mengembalikan ErrCreditNoteNotFound jika tidak ditemukan.
func (r *CreditNoteRepo) GetByID(ctx context.Context, id string) (*domain.CreditNote, error) {
	query := `SELECT id, tenant_id, credit_note_number, invoice_id, amount, reason,
		apply_to_credit, created_by_id, created_by_name, created_at
	FROM credit_notes
	WHERE id = $1`

	return r.scanCreditNote(r.pool.QueryRow(ctx, query, stringToUUID(id)))
}

// ListByInvoice mengambil semua credit note untuk invoice tertentu.
func (r *CreditNoteRepo) ListByInvoice(ctx context.Context, invoiceID string) ([]*domain.CreditNote, error) {
	query := `SELECT id, tenant_id, credit_note_number, invoice_id, amount, reason,
		apply_to_credit, created_by_id, created_by_name, created_at
	FROM credit_notes
	WHERE invoice_id = $1
	ORDER BY created_at ASC`

	rows, err := r.pool.Query(ctx, query, stringToUUID(invoiceID))
	if err != nil {
		return nil, fmt.Errorf("repository: gagal mengambil credit notes by invoice: %w", err)
	}
	defer rows.Close()

	result := make([]*domain.CreditNote, 0)
	for rows.Next() {
		cn, err := r.scanCreditNoteFromRows(rows)
		if err != nil {
			return nil, fmt.Errorf("repository: gagal scan credit note row: %w", err)
		}
		result = append(result, cn)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("repository: gagal iterasi credit note rows: %w", err)
	}
	return result, nil
}

// scanCreditNote memindai satu baris credit note dari pgx.Row.
func (r *CreditNoteRepo) scanCreditNote(row pgx.Row) (*domain.CreditNote, error) {
	var (
		id               pgtype.UUID
		tenantID         pgtype.UUID
		creditNoteNumber string
		invoiceID        pgtype.UUID
		amount           int64
		reason           string
		applyToCredit    bool
		createdByID      string
		createdByName    string
		createdAt        pgtype.Timestamptz
	)

	err := row.Scan(
		&id, &tenantID, &creditNoteNumber, &invoiceID, &amount, &reason,
		&applyToCredit, &createdByID, &createdByName, &createdAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrCreditNoteNotFound
		}
		return nil, fmt.Errorf("repository: gagal mengambil credit note: %w", err)
	}

	return &domain.CreditNote{
		ID:               uuidToString(id),
		TenantID:         uuidToString(tenantID),
		CreditNoteNumber: creditNoteNumber,
		InvoiceID:        uuidToString(invoiceID),
		Amount:           amount,
		Reason:           reason,
		ApplyToCredit:    applyToCredit,
		CreatedByID:      createdByID,
		CreatedByName:    createdByName,
		CreatedAt:        timestamptzToTime(createdAt),
	}, nil
}

// scanCreditNoteFromRows memindai satu baris credit note dari pgx.Rows.
func (r *CreditNoteRepo) scanCreditNoteFromRows(rows pgx.Rows) (*domain.CreditNote, error) {
	var (
		id               pgtype.UUID
		tenantID         pgtype.UUID
		creditNoteNumber string
		invoiceID        pgtype.UUID
		amount           int64
		reason           string
		applyToCredit    bool
		createdByID      string
		createdByName    string
		createdAt        pgtype.Timestamptz
	)

	err := rows.Scan(
		&id, &tenantID, &creditNoteNumber, &invoiceID, &amount, &reason,
		&applyToCredit, &createdByID, &createdByName, &createdAt,
	)
	if err != nil {
		return nil, err
	}

	return &domain.CreditNote{
		ID:               uuidToString(id),
		TenantID:         uuidToString(tenantID),
		CreditNoteNumber: creditNoteNumber,
		InvoiceID:        uuidToString(invoiceID),
		Amount:           amount,
		Reason:           reason,
		ApplyToCredit:    applyToCredit,
		CreatedByID:      createdByID,
		CreatedByName:    createdByName,
		CreatedAt:        timestamptzToTime(createdAt),
	}, nil
}

// =============================================================================
// DebitNoteRepo - implementasi domain.DebitNoteRepository
// =============================================================================

// DebitNoteRepo mengimplementasikan domain.DebitNoteRepository menggunakan raw SQL
// karena tabel debit_notes belum memiliki sqlc queries.
type DebitNoteRepo struct {
	// pool digunakan untuk raw SQL queries dengan pgx.
	pool *pgxpool.Pool
}

// NewDebitNoteRepo membuat instance baru DebitNoteRepo.
func NewDebitNoteRepo(pool *pgxpool.Pool) *DebitNoteRepo {
	return &DebitNoteRepo{
		pool: pool,
	}
}

// Buat membuat debit note baru beserta items dan mengembalikan debit note yang dibuat.
// Operasi ini menggunakan transaksi untuk memastikan konsistensi antara debit_notes dan debit_note_items.
func (r *DebitNoteRepo) Create(ctx context.Context, dn *domain.DebitNote) (*domain.DebitNote, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("repository: gagal memulai transaksi debit note: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	// Insert debit note
	dnQuery := `INSERT INTO debit_notes (
		tenant_id, debit_note_number, customer_id, due_date,
		total_amount, invoice_id, created_by_id, created_by_name
	) VALUES (
		$1, $2, $3, $4,
		$5, $6, $7, $8
	)
	RETURNING id, tenant_id, debit_note_number, customer_id, due_date,
		total_amount, invoice_id, created_by_id, created_by_name, created_at`

	var (
		id              pgtype.UUID
		tenantID        pgtype.UUID
		debitNoteNumber string
		customerID      pgtype.UUID
		dueDate         pgtype.Date
		totalAmount     int64
		invoiceID       pgtype.UUID
		createdByID     string
		createdByName   string
		createdAt       pgtype.Timestamptz
	)

	// Konversi invoice_id opsional
	var invoiceIDParam pgtype.UUID
	if dn.InvoiceID != nil {
		invoiceIDParam = stringToUUID(*dn.InvoiceID)
	}

	err = tx.QueryRow(ctx, dnQuery,
		stringToUUID(dn.TenantID),
		dn.DebitNoteNumber,
		stringToUUID(dn.CustomerID),
		timeToDate(dn.DueDate),
		dn.TotalAmount,
		invoiceIDParam,
		dn.CreatedByID,
		dn.CreatedByName,
	).Scan(
		&id, &tenantID, &debitNoteNumber, &customerID, &dueDate,
		&totalAmount, &invoiceID, &createdByID, &createdByName, &createdAt,
	)
	if err != nil {
		return nil, fmt.Errorf("repository: gagal membuat debit note: %w", err)
	}

	debitNoteID := uuidToString(id)

	// Insert debit note items
	itemQuery := `INSERT INTO debit_note_items (
		debit_note_id, description, amount
	) VALUES ($1, $2, $3)
	RETURNING id, debit_note_id, description, amount`

	items := make([]domain.DebitNoteItem, 0, len(dn.Items))
	for _, item := range dn.Items {
		var (
			itemID          pgtype.UUID
			itemDebitNoteID pgtype.UUID
			itemDescription string
			itemAmount      int64
		)

		err = tx.QueryRow(ctx, itemQuery,
			stringToUUID(debitNoteID),
			item.Description,
			item.Amount,
		).Scan(&itemID, &itemDebitNoteID, &itemDescription, &itemAmount)
		if err != nil {
			return nil, fmt.Errorf("repository: gagal membuat debit note item: %w", err)
		}

		items = append(items, domain.DebitNoteItem{
			ID:          uuidToString(itemID),
			DebitNoteID: uuidToString(itemDebitNoteID),
			Description: itemDescription,
			Amount:      itemAmount,
		})
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("repository: gagal commit transaksi debit note: %w", err)
	}

	// Konversi invoice_id opsional untuk respons
	var resultInvoiceID *string
	if invoiceID.Valid {
		s := uuidToString(invoiceID)
		resultInvoiceID = &s
	}

	return &domain.DebitNote{
		ID:              debitNoteID,
		TenantID:        uuidToString(tenantID),
		DebitNoteNumber: debitNoteNumber,
		CustomerID:      uuidToString(customerID),
		DueDate:         dateToTime(dueDate),
		Items:           items,
		TotalAmount:     totalAmount,
		InvoiceID:       resultInvoiceID,
		CreatedByID:     createdByID,
		CreatedByName:   createdByName,
		CreatedAt:       timestamptzToTime(createdAt),
	}, nil
}

// GetByID mengambil debit note berdasarkan ID beserta items-nya.
// Mengembalikan ErrDebitNoteNotFound jika tidak ditemukan.
func (r *DebitNoteRepo) GetByID(ctx context.Context, id string) (*domain.DebitNote, error) {
	// Ambil debit note
	dnQuery := `SELECT id, tenant_id, debit_note_number, customer_id, due_date,
		total_amount, invoice_id, created_by_id, created_by_name, created_at
	FROM debit_notes
	WHERE id = $1`

	dn, err := r.scanDebitNote(r.pool.QueryRow(ctx, dnQuery, stringToUUID(id)))
	if err != nil {
		return nil, err
	}

	// Ambil items untuk debit note
	items, err := r.listItemsByDebitNote(ctx, id)
	if err != nil {
		return nil, err
	}
	dn.Items = items

	return dn, nil
}

// ListByCustomer mengambil semua debit note untuk customer tertentu beserta items-nya.
func (r *DebitNoteRepo) ListByCustomer(ctx context.Context, customerID string) ([]*domain.DebitNote, error) {
	dnQuery := `SELECT id, tenant_id, debit_note_number, customer_id, due_date,
		total_amount, invoice_id, created_by_id, created_by_name, created_at
	FROM debit_notes
	WHERE customer_id = $1
	ORDER BY created_at DESC`

	rows, err := r.pool.Query(ctx, dnQuery, stringToUUID(customerID))
	if err != nil {
		return nil, fmt.Errorf("repository: gagal mengambil debit notes by customer: %w", err)
	}
	defer rows.Close()

	result := make([]*domain.DebitNote, 0)
	for rows.Next() {
		dn, err := r.scanDebitNoteFromRows(rows)
		if err != nil {
			return nil, fmt.Errorf("repository: gagal scan debit note row: %w", err)
		}
		result = append(result, dn)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("repository: gagal iterasi debit note rows: %w", err)
	}

	// Ambil items untuk setiap debit note
	for _, dn := range result {
		items, err := r.listItemsByDebitNote(ctx, dn.ID)
		if err != nil {
			return nil, err
		}
		dn.Items = items
	}

	return result, nil
}

// listItemsByDebitNote mengambil semua items untuk debit note tertentu.
func (r *DebitNoteRepo) listItemsByDebitNote(ctx context.Context, debitNoteID string) ([]domain.DebitNoteItem, error) {
	query := `SELECT id, debit_note_id, description, amount
	FROM debit_note_items
	WHERE debit_note_id = $1
	ORDER BY id ASC`

	rows, err := r.pool.Query(ctx, query, stringToUUID(debitNoteID))
	if err != nil {
		return nil, fmt.Errorf("repository: gagal mengambil debit note items: %w", err)
	}
	defer rows.Close()

	items := make([]domain.DebitNoteItem, 0)
	for rows.Next() {
		var (
			itemID      pgtype.UUID
			dnID        pgtype.UUID
			description string
			amount      int64
		)
		if err := rows.Scan(&itemID, &dnID, &description, &amount); err != nil {
			return nil, fmt.Errorf("repository: gagal scan debit note item row: %w", err)
		}
		items = append(items, domain.DebitNoteItem{
			ID:          uuidToString(itemID),
			DebitNoteID: uuidToString(dnID),
			Description: description,
			Amount:      amount,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("repository: gagal iterasi debit note item rows: %w", err)
	}
	return items, nil
}

// scanDebitNote memindai satu baris debit note dari pgx.Row.
func (r *DebitNoteRepo) scanDebitNote(row pgx.Row) (*domain.DebitNote, error) {
	var (
		id              pgtype.UUID
		tenantID        pgtype.UUID
		debitNoteNumber string
		customerID      pgtype.UUID
		dueDate         pgtype.Date
		totalAmount     int64
		invoiceID       pgtype.UUID
		createdByID     string
		createdByName   string
		createdAt       pgtype.Timestamptz
	)

	err := row.Scan(
		&id, &tenantID, &debitNoteNumber, &customerID, &dueDate,
		&totalAmount, &invoiceID, &createdByID, &createdByName, &createdAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrDebitNoteNotFound
		}
		return nil, fmt.Errorf("repository: gagal mengambil debit note: %w", err)
	}

	// Konversi invoice_id opsional
	var resultInvoiceID *string
	if invoiceID.Valid {
		s := uuidToString(invoiceID)
		resultInvoiceID = &s
	}

	return &domain.DebitNote{
		ID:              uuidToString(id),
		TenantID:        uuidToString(tenantID),
		DebitNoteNumber: debitNoteNumber,
		CustomerID:      uuidToString(customerID),
		DueDate:         dateToTime(dueDate),
		TotalAmount:     totalAmount,
		InvoiceID:       resultInvoiceID,
		CreatedByID:     createdByID,
		CreatedByName:   createdByName,
		CreatedAt:       timestamptzToTime(createdAt),
	}, nil
}

// scanDebitNoteFromRows memindai satu baris debit note dari pgx.Rows.
func (r *DebitNoteRepo) scanDebitNoteFromRows(rows pgx.Rows) (*domain.DebitNote, error) {
	var (
		id              pgtype.UUID
		tenantID        pgtype.UUID
		debitNoteNumber string
		customerID      pgtype.UUID
		dueDate         pgtype.Date
		totalAmount     int64
		invoiceID       pgtype.UUID
		createdByID     string
		createdByName   string
		createdAt       pgtype.Timestamptz
	)

	err := rows.Scan(
		&id, &tenantID, &debitNoteNumber, &customerID, &dueDate,
		&totalAmount, &invoiceID, &createdByID, &createdByName, &createdAt,
	)
	if err != nil {
		return nil, err
	}

	// Konversi invoice_id opsional
	var resultInvoiceID *string
	if invoiceID.Valid {
		s := uuidToString(invoiceID)
		resultInvoiceID = &s
	}

	return &domain.DebitNote{
		ID:              uuidToString(id),
		TenantID:        uuidToString(tenantID),
		DebitNoteNumber: debitNoteNumber,
		CustomerID:      uuidToString(customerID),
		DueDate:         dateToTime(dueDate),
		TotalAmount:     totalAmount,
		InvoiceID:       resultInvoiceID,
		CreatedByID:     createdByID,
		CreatedByName:   createdByName,
		CreatedAt:       timestamptzToTime(createdAt),
	}, nil
}
