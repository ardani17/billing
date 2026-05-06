package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CashflowRepo struct {
	pool *pgxpool.Pool
}

func NewCashflowRepo(pool *pgxpool.Pool) *CashflowRepo {
	return &CashflowRepo{pool: pool}
}

func (r *CashflowRepo) Summary(ctx context.Context, tenantID string, start, end time.Time) (*domain.CashflowSummary, error) {
	openingIn, openingOut, err := r.totals(ctx, tenantID, time.Time{}, start.AddDate(0, 0, -1))
	if err != nil {
		return nil, err
	}
	cashIn, cashOut, err := r.totals(ctx, tenantID, start, end)
	if err != nil {
		return nil, err
	}
	breakdown, err := r.Breakdown(ctx, tenantID, start, end)
	if err != nil {
		return nil, err
	}
	latest, err := r.Transactions(ctx, tenantID, start, end, "", "", "", "")
	if err != nil {
		return nil, err
	}
	if len(latest) > 10 {
		latest = latest[:10]
	}

	opening := openingIn - openingOut
	return &domain.CashflowSummary{
		OpeningBalance:         opening,
		TotalCashIn:            cashIn,
		TotalCashOut:           cashOut,
		NetCashflow:            cashIn - cashOut,
		ClosingBalanceEstimate: opening + cashIn - cashOut,
		Breakdown:              breakdown,
		LatestTransactions:     latest,
	}, nil
}

func (r *CashflowRepo) totals(ctx context.Context, tenantID string, start, end time.Time) (int64, int64, error) {
	transactions, err := r.Transactions(ctx, tenantID, start, end, "", "", "", "")
	if err != nil {
		return 0, 0, err
	}
	var cashIn, cashOut int64
	for _, tx := range transactions {
		if tx.Direction == "in" {
			cashIn += tx.Amount
		} else {
			cashOut += tx.Amount
		}
	}
	return cashIn, cashOut, nil
}

func (r *CashflowRepo) Breakdown(ctx context.Context, tenantID string, start, end time.Time) ([]domain.CashflowBreakdown, error) {
	transactions, err := r.Transactions(ctx, tenantID, start, end, "", "", "", "")
	if err != nil {
		return nil, err
	}
	byKey := map[string]*domain.CashflowBreakdown{}
	for _, tx := range transactions {
		key := tx.Direction + "|" + tx.Source + "|" + tx.Category
		item, ok := byKey[key]
		if !ok {
			item = &domain.CashflowBreakdown{Direction: tx.Direction, Source: tx.Source, Category: tx.Category}
			byKey[key] = item
		}
		item.Amount += tx.Amount
	}
	items := make([]domain.CashflowBreakdown, 0, len(byKey))
	for _, item := range byKey {
		items = append(items, *item)
	}
	return items, nil
}

func (r *CashflowRepo) Transactions(ctx context.Context, tenantID string, start, end time.Time, direction, source, category, search string) ([]domain.CashflowTransaction, error) {
	rows, err := r.pool.Query(ctx, cashflowUnionQuery(), tenantID, nullableDate(start), nullableDate(end), direction, source, category, "%"+search+"%")
	if err != nil {
		return nil, fmt.Errorf("repository: gagal mengambil transaksi cashflow: %w", err)
	}
	defer rows.Close()

	items := []domain.CashflowTransaction{}
	for rows.Next() {
		item, err := scanCashflowTransaction(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *CashflowRepo) Trend(ctx context.Context, tenantID string, start, end time.Time) ([]domain.CashflowTrendPoint, error) {
	transactions, err := r.Transactions(ctx, tenantID, start, end, "", "", "", "")
	if err != nil {
		return nil, err
	}
	byDate := map[string]*domain.CashflowTrendPoint{}
	for _, tx := range transactions {
		key := tx.Date.Format("2006-01-02")
		point, ok := byDate[key]
		if !ok {
			point = &domain.CashflowTrendPoint{Date: key}
			byDate[key] = point
		}
		if tx.Direction == "in" {
			point.CashIn += tx.Amount
		} else {
			point.CashOut += tx.Amount
		}
		point.Net = point.CashIn - point.CashOut
	}
	points := []domain.CashflowTrendPoint{}
	for day := start; !day.After(end); day = day.AddDate(0, 0, 1) {
		key := day.Format("2006-01-02")
		if point, ok := byDate[key]; ok {
			points = append(points, *point)
		} else {
			points = append(points, domain.CashflowTrendPoint{Date: key})
		}
	}
	return points, nil
}

func cashflowUnionQuery() string {
	return `WITH cashflow AS (
		SELECT id::text, payment_date::timestamptz AS tx_date, 'in' AS direction,
			'pembayaran' AS source, payment_method AS category,
			COALESCE(reference_number, notes, 'Pembayaran invoice') AS description, amount
		FROM invoice_payments
		WHERE tenant_id = $1 AND voided = false
		UNION ALL
		SELECT id::text, created_at AS tx_date,
			CASE WHEN type IN ('deposit','refund') THEN 'in' ELSE 'out' END AS direction,
			'reseller' AS source, type AS category,
			COALESCE(notes, 'Transaksi reseller') AS description, amount
		FROM reseller_transactions
		WHERE tenant_id = $1 AND type IN ('deposit','withdraw','refund')
		UNION ALL
		SELECT v.id::text, COALESCE(v.purchased_at, v.activated_at, v.created_at) AS tx_date,
			'in' AS direction, 'voucher' AS source, 'penjualan langsung' AS category,
			'Penjualan voucher ' || v.code AS description, COALESCE(v.sell_price_snapshot, 0) AS amount
		FROM vouchers v
		WHERE v.tenant_id = $1
			AND v.reseller_id IS NULL
			AND v.status IN ('terjual','aktif','selesai')
			AND v.voided_at IS NULL
			AND COALESCE(v.sell_price_snapshot, 0) > 0
		UNION ALL
		SELECT e.id::text, e.expense_date::timestamptz AS tx_date, 'out' AS direction,
			'pengeluaran' AS source, c.name AS category, e.description, e.amount
		FROM expenses e
		JOIN expense_categories c ON c.id = e.category_id
		WHERE e.tenant_id = $1 AND e.deleted_at IS NULL
		UNION ALL
		SELECT m.id::text, m.created_at AS tx_date, 'out' AS direction,
			'inventaris' AS source, 'pembelian stok' AS category,
			COALESCE(NULLIF(m.notes,''), 'Pembelian inventaris') AS description,
			(ABS(m.quantity) * m.unit_cost)::bigint AS amount
		FROM inventory_movements m
		WHERE m.tenant_id = $1
			AND m.movement_type = 'purchase'
			AND m.expense_id IS NULL
			AND m.unit_cost > 0
		UNION ALL
		SELECT id::text, transaction_date::timestamptz AS tx_date,
			direction, 'manual' AS source, category, description, amount
		FROM cashflow_manual_transactions
		WHERE tenant_id = $1 AND deleted_at IS NULL
	)
	SELECT id, tx_date, direction, source, category, description, amount
	FROM cashflow
	WHERE ($2::date IS NULL OR tx_date::date >= $2::date)
	  AND ($3::date IS NULL OR tx_date::date <= $3::date)
	  AND ($4 = '' OR direction = $4)
	  AND ($5 = '' OR source = $5)
	  AND ($6 = '' OR category = $6)
	  AND ($7 = '%%' OR description ILIKE $7 OR category ILIKE $7 OR source ILIKE $7)
	ORDER BY tx_date DESC, id DESC`
}

func (r *CashflowRepo) CreateManualTransaction(ctx context.Context, tenantID string, req domain.CreateManualCashflowRequest, actor domain.ActorInfo) (domain.CashflowTransaction, error) {
	txDate, err := time.Parse("2006-01-02", req.TransactionDate)
	if err != nil {
		return domain.CashflowTransaction{}, fmt.Errorf("format tanggal tidak valid")
	}
	row := r.pool.QueryRow(ctx, `
		INSERT INTO cashflow_manual_transactions (
			tenant_id, direction, category, description, amount, transaction_date, created_by_id
		) VALUES ($1,$2,$3,$4,$5,$6,$7)
		RETURNING id::text, transaction_date::timestamptz, direction, 'manual'::text, category, description, amount`,
		tenantID, req.Direction, req.Category, req.Description, req.Amount, txDate, actor.ActorID)
	item, err := scanCashflowTransaction(row)
	if err != nil {
		return domain.CashflowTransaction{}, fmt.Errorf("repository: gagal mencatat transaksi kas manual: %w", err)
	}
	return item, nil
}

func scanCashflowTransaction(row pgx.Row) (domain.CashflowTransaction, error) {
	var item domain.CashflowTransaction
	err := row.Scan(&item.ID, &item.Date, &item.Direction, &item.Source, &item.Category, &item.Description, &item.Amount)
	return item, err
}

func nullableDate(value time.Time) interface{} {
	if value.IsZero() {
		return nil
	}
	return value
}
