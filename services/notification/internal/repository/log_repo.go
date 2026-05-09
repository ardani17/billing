package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/ispboss/ispboss/services/notification/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// LogRepo mengimplementasikan domain.LogRepository menggunakan sqlc Queries.
type LogRepo struct{ queries *Queries }

// NewLogRepo membuat instance baru LogRepo.
func NewLogRepo(q *Queries) *LogRepo { return &LogRepo{queries: q} }

// marshalMetadata mengkonversi map metadata ke []byte JSONB ('{}' jika nil).
func marshalMetadata(m map[string]interface{}) ([]byte, error) {
	if len(m) == 0 {
		return []byte("{}"), nil
	}
	return json.Marshal(m)
}

// unmarshalMetadata mengkonversi []byte JSONB ke map metadata (nil jika kosong).
func unmarshalMetadata(data []byte) (map[string]interface{}, error) {
	if len(data) == 0 {
		return nil, nil
	}
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("repository: gagal unmarshal metadata: %w", err)
	}
	if len(m) == 0 {
		return nil, nil
	}
	return m, nil
}

func timeToTimestamptz(t *time.Time) pgtype.Timestamptz {
	if t == nil {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: *t, Valid: true}
}

func timestamptzToTimePtr(t pgtype.Timestamptz) *time.Time {
	if !t.Valid {
		return nil
	}
	return &t.Time
}

func parseOptionalUUID(s string) pgtype.UUID {
	if s == "" {
		return pgtype.UUID{}
	}
	return parseUUID(s)
}

// mapLogRow memetakan sqlc NotificationLog ke domain.NotificationLog.
func mapLogRow(row NotificationLog) (*domain.NotificationLog, error) {
	metadata, err := unmarshalMetadata(row.Metadata)
	if err != nil {
		return nil, err
	}
	return &domain.NotificationLog{
		ID: uuidToString(row.ID), TenantID: uuidToString(row.TenantID),
		CustomerID: uuidToString(row.CustomerID), TemplateID: uuidToString(row.TemplateID),
		Channel: domain.Channel(row.Channel), Provider: row.Provider,
		Recipient: row.Recipient, Subject: textToString(row.Subject),
		Body: row.Body, Status: domain.LogStatus(row.Status),
		RetryCount: int(row.RetryCount), MaxRetries: int(row.MaxRetries),
		ErrorMessage: textToString(row.ErrorMessage), DedupKey: textToString(row.DedupKey),
		Metadata: metadata, SentAt: timestamptzToTimePtr(row.SentAt),
		CreatedAt: timestamptzToTime(row.CreatedAt), UpdatedAt: timestamptzToTime(row.UpdatedAt),
	}, nil
}

// Buat membuat catatan log notifikasi baru dan mengembalikan log yang dibuat.
func (r *LogRepo) Create(ctx context.Context, log *domain.NotificationLog) (*domain.NotificationLog, error) {
	metaJSON, err := marshalMetadata(log.Metadata)
	if err != nil {
		return nil, fmt.Errorf("repository: gagal marshal metadata: %w", err)
	}
	row, err := r.queries.CreateLog(ctx, CreateLogParams{
		TenantID: parseUUID(log.TenantID), CustomerID: parseUUID(log.CustomerID),
		TemplateID: parseUUID(log.TemplateID), Channel: string(log.Channel),
		Provider: log.Provider, Recipient: log.Recipient,
		Subject: stringToText(log.Subject), Body: log.Body,
		Status: string(log.Status), RetryCount: int32(log.RetryCount),
		MaxRetries: int32(log.MaxRetries), ErrorMessage: stringToText(log.ErrorMessage),
		DedupKey: stringToText(log.DedupKey), Metadata: metaJSON,
		SentAt: timeToTimestamptz(log.SentAt),
	})
	if err != nil {
		return nil, fmt.Errorf("repository: gagal membuat log: %w", err)
	}
	return mapLogRow(row)
}

// GetByID mengambil log notifikasi berdasarkan ID.
// Mengembalikan domain.ErrLogNotFound jika tidak ditemukan.
func (r *LogRepo) GetByID(ctx context.Context, id string) (*domain.NotificationLog, error) {
	row, err := r.queries.GetLogByID(ctx, parseUUID(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrLogNotFound
		}
		return nil, fmt.Errorf("repository: gagal mengambil log by ID: %w", err)
	}
	return mapLogRow(row)
}

// Perbarui memperbarui status, retry_count, error_message, sent_at, dan metadata.
func (r *LogRepo) Update(ctx context.Context, log *domain.NotificationLog) error {
	metaJSON, err := marshalMetadata(log.Metadata)
	if err != nil {
		return err
	}
	return r.queries.UpdateLog(ctx, UpdateLogParams{
		ID: parseUUID(log.ID), Status: string(log.Status),
		RetryCount: int32(log.RetryCount), ErrorMessage: stringToText(log.ErrorMessage),
		SentAt: timeToTimestamptz(log.SentAt), Metadata: metaJSON,
	})
}

// List mengambil daftar log notifikasi dengan filter dan paginasi.
// Menggunakan ListLogs + CountLogs, lalu menghitung total_pages.
func (r *LogRepo) List(ctx context.Context, params domain.LogListParams) (*domain.LogListResult, error) {
	offset := int32((params.Page - 1) * params.PageSize)
	chF := stringToText(string(params.Channel))
	stF := stringToText(string(params.Status))
	cuF := parseOptionalUUID(params.CustomerID)
	tmF := parseOptionalUUID(params.TemplateID)
	dfF := timeToTimestamptz(params.DateFrom)
	dtF := timeToTimestamptz(params.DateTo)
	tid := parseUUID(params.TenantID)

	rows, err := r.queries.ListLogs(ctx, ListLogsParams{
		TenantID: tid, Limit: int32(params.PageSize), Offset: offset,
		Channel: chF, Status: stF, CustomerID: cuF,
		TemplateID: tmF, DateFrom: dfF, DateTo: dtF,
	})
	if err != nil {
		return nil, fmt.Errorf("repository: gagal mengambil daftar log: %w", err)
	}
	total, err := r.queries.CountLogs(ctx, CountLogsParams{
		TenantID: tid, Channel: chF, Status: stF,
		CustomerID: cuF, TemplateID: tmF, DateFrom: dfF, DateTo: dtF,
	})
	if err != nil {
		return nil, fmt.Errorf("repository: gagal menghitung total log: %w", err)
	}

	data := make([]*domain.NotificationLog, 0, len(rows))
	for _, row := range rows {
		l, err := mapLogRow(row)
		if err != nil {
			return nil, err
		}
		data = append(data, l)
	}
	totalPages := int(math.Ceil(float64(total) / float64(params.PageSize)))
	return &domain.LogListResult{
		Data: data, Total: total, Page: params.Page,
		PageSize: params.PageSize, TotalPages: totalPages,
	}, nil
}

// FindByDedupKey mencari log berdasarkan dedup_key dalam jendela waktu tertentu.
// Mengembalikan nil tanpa error jika tidak ditemukan (bukan duplikat).
func (r *LogRepo) FindByDedupKey(ctx context.Context, dedupKey string, withinHours int) (*domain.NotificationLog, error) {
	row, err := r.queries.FindByDedupKey(ctx, FindByDedupKeyParams{
		DedupKey: stringToText(dedupKey),
		Column2:  pgtype.Text{String: fmt.Sprintf("%d", withinHours), Valid: true},
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("repository: gagal mencari dedup_key: %w", err)
	}
	return mapLogRow(row)
}

// Raw query karena sqlc salah inferensi $3 sebagai pgtype.Interval (seharusnya string timezone).
const countTodaySQL = `SELECT COUNT(*) FROM notification_logs
WHERE tenant_id = $1 AND customer_id = $2 AND status IN ('sent', 'delivered')
  AND created_at >= (NOW() AT TIME ZONE $3)::date AT TIME ZONE $3`

// CountTodayByCustomer menghitung notifikasi ke pelanggan hari ini (timezone tenant).
func (r *LogRepo) CountTodayByCustomer(ctx context.Context, tenantID, customerID, tz string) (int, error) {
	var count int64
	err := r.queries.db.QueryRow(ctx, countTodaySQL, parseUUID(tenantID), parseUUID(customerID), tz).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("repository: gagal menghitung notifikasi hari ini: %w", err)
	}
	return int(count), nil
}

// LastSentToCustomer mengambil waktu pengiriman terakhir ke pelanggan.
// Mengembalikan nil jika belum pernah ada pengiriman (untuk pengecekan cooldown).
func (r *LogRepo) LastSentToCustomer(ctx context.Context, tenantID, customerID string) (*time.Time, error) {
	sentAt, err := r.queries.LastSentToCustomer(ctx, LastSentToCustomerParams{
		TenantID: parseUUID(tenantID), CustomerID: parseUUID(customerID),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("repository: gagal mengambil waktu kirim terakhir: %w", err)
	}
	return timestamptzToTimePtr(sentAt), nil
}
