package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// RouterRepo mengimplementasikan domain.RouterRepository dengan membungkus
// sqlc-generated Queries dan memetakan tipe database ke domain.Router.
type RouterRepo struct {
	queries *Queries
}

// NewRouterRepo membuat instance baru RouterRepo.
func NewRouterRepo(queries *Queries) *RouterRepo {
	return &RouterRepo{queries: queries}
}

// --- Helper functions untuk konversi pgtype ↔ domain types ---

// uuidToString mengkonversi pgtype.UUID ke string.
func uuidToString(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	return fmt.Sprintf("%x-%x-%x-%x-%x",
		u.Bytes[0:4], u.Bytes[4:6], u.Bytes[6:8], u.Bytes[8:10], u.Bytes[10:16])
}

// stringToUUID mengkonversi string UUID ke pgtype.UUID.
func stringToUUID(s string) pgtype.UUID {
	var u pgtype.UUID
	_ = u.Scan(s)
	return u
}

// textToString mengkonversi pgtype.Text ke string.
func textToString(t pgtype.Text) string {
	if !t.Valid {
		return ""
	}
	return t.String
}

// stringToText mengkonversi string ke pgtype.Text. String kosong → NULL.
func stringToText(s string) pgtype.Text {
	if s == "" {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: s, Valid: true}
}

// int4ToInt mengkonversi pgtype.Int4 ke int. Mengembalikan 0 jika NULL.
func int4ToInt(i pgtype.Int4) int {
	if !i.Valid {
		return 0
	}
	return int(i.Int32)
}

// intToInt4 mengkonversi int ke pgtype.Int4. Nilai 0 → NULL.
func intToInt4(i int) pgtype.Int4 {
	if i == 0 {
		return pgtype.Int4{Valid: false}
	}
	return pgtype.Int4{Int32: int32(i), Valid: true}
}

// int8ToInt64Ptr mengkonversi pgtype.Int8 ke *int64. NULL → nil.
func int8ToInt64Ptr(i pgtype.Int8) *int64 {
	if !i.Valid {
		return nil
	}
	v := i.Int64
	return &v
}

// int64PtrToInt8 mengkonversi *int64 ke pgtype.Int8. nil → NULL.
func int64PtrToInt8(i *int64) pgtype.Int8 {
	if i == nil {
		return pgtype.Int8{Valid: false}
	}
	return pgtype.Int8{Int64: *i, Valid: true}
}

// timestamptzToTime mengkonversi pgtype.Timestamptz ke time.Time.
func timestamptzToTime(t pgtype.Timestamptz) time.Time {
	if !t.Valid {
		return time.Time{}
	}
	return t.Time
}

// timestamptzToTimePtr mengkonversi pgtype.Timestamptz ke *time.Time. NULL → nil.
func timestamptzToTimePtr(t pgtype.Timestamptz) *time.Time {
	if !t.Valid {
		return nil
	}
	return &t.Time
}

// timePtrToTimestamptz mengkonversi *time.Time ke pgtype.Timestamptz. nil → NULL.
func timePtrToTimestamptz(t *time.Time) pgtype.Timestamptz {
	if t == nil {
		return pgtype.Timestamptz{Valid: false}
	}
	return pgtype.Timestamptz{Time: *t, Valid: true}
}

// serviceTypesToJSON mengkonversi []string ke []byte (JSON) untuk JSONB column.
func serviceTypesToJSON(types []string) ([]byte, error) {
	if types == nil {
		types = []string{"pppoe"}
	}
	return json.Marshal(types)
}

// jsonToServiceTypes mengkonversi []byte (JSON) ke []string dari JSONB column.
func jsonToServiceTypes(data []byte) []string {
	var types []string
	if err := json.Unmarshal(data, &types); err != nil {
		return []string{}
	}
	return types
}

// --- Mapping sqlc Router → domain.Router ---

// mapRouterRow memetakan Router (sqlc model) ke domain.Router.
func mapRouterRow(row Router) *domain.Router {
	return &domain.Router{
		ID:                     uuidToString(row.ID),
		TenantID:               uuidToString(row.TenantID),
		Name:                   row.Name,
		Host:                   row.Host,
		Port:                   int(row.Port),
		Username:               row.Username,
		PasswordEncrypted:      row.PasswordEncrypted,
		UseSSL:                 row.UseSsl,
		ServiceTypes:           jsonToServiceTypes(row.ServiceTypes),
		RouterOSVersion:        textToString(row.RouterOsVersion),
		BoardName:              textToString(row.BoardName),
		CPUCount:               int4ToInt(row.CpuCount),
		TotalRAMMB:             int4ToInt(row.TotalRamMb),
		Identity:               textToString(row.Identity),
		Status:                 domain.RouterStatus(row.Status),
		HealthCheckIntervalSec: int(row.HealthCheckIntervalSec),
		LastOnlineAt:           timestamptzToTimePtr(row.LastOnlineAt),
		LastCheckedAt:          timestamptzToTimePtr(row.LastCheckedAt),
		LastUptimeSec:          int8ToInt64Ptr(row.LastUptimeSec),
		FailureCount:           int(row.FailureCount),
		Notes:                  textToString(row.Notes),
		DeletedAt:              timestamptzToTimePtr(row.DeletedAt),
		CreatedAt:              timestamptzToTime(row.CreatedAt),
		UpdatedAt:              timestamptzToTime(row.UpdatedAt),
	}
}

// --- Implementasi domain.RouterRepository ---

// Create membuat router baru dan mengembalikan router yang dibuat.
func (r *RouterRepo) Create(ctx context.Context, router *domain.Router) (*domain.Router, error) {
	serviceTypesJSON, err := serviceTypesToJSON(router.ServiceTypes)
	if err != nil {
		return nil, fmt.Errorf("repository: gagal marshal service_types: %w", err)
	}

	row, err := r.queries.CreateRouter(ctx, CreateRouterParams{
		TenantID:               stringToUUID(router.TenantID),
		Name:                   router.Name,
		Host:                   router.Host,
		Port:                   int32(router.Port),
		Username:               router.Username,
		PasswordEncrypted:      router.PasswordEncrypted,
		UseSsl:                 router.UseSSL,
		ServiceTypes:           serviceTypesJSON,
		RouterOsVersion:        stringToText(router.RouterOSVersion),
		BoardName:              stringToText(router.BoardName),
		CpuCount:               intToInt4(router.CPUCount),
		TotalRamMb:             intToInt4(router.TotalRAMMB),
		Identity:               stringToText(router.Identity),
		Status:                 string(router.Status),
		HealthCheckIntervalSec: int32(router.HealthCheckIntervalSec),
		Notes:                  stringToText(router.Notes),
	})
	if err != nil {
		return nil, fmt.Errorf("repository: gagal membuat router: %w", err)
	}
	return mapRouterRow(row), nil
}

// GetByID mengambil router berdasarkan ID (tenant-scoped via RLS).
func (r *RouterRepo) GetByID(ctx context.Context, id string) (*domain.Router, error) {
	row, err := r.queries.GetRouterByID(ctx, stringToUUID(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrRouterNotFound
		}
		return nil, fmt.Errorf("repository: gagal mengambil router by ID: %w", err)
	}
	return mapRouterRow(row), nil
}

// Update memperbarui data router dan mengembalikan router yang diperbarui.
func (r *RouterRepo) Update(ctx context.Context, router *domain.Router) (*domain.Router, error) {
	serviceTypesJSON, err := serviceTypesToJSON(router.ServiceTypes)
	if err != nil {
		return nil, fmt.Errorf("repository: gagal marshal service_types: %w", err)
	}

	row, err := r.queries.UpdateRouter(ctx, UpdateRouterParams{
		ID:                     stringToUUID(router.ID),
		Name:                   router.Name,
		Host:                   router.Host,
		Port:                   int32(router.Port),
		Username:               router.Username,
		PasswordEncrypted:      router.PasswordEncrypted,
		UseSsl:                 router.UseSSL,
		ServiceTypes:           serviceTypesJSON,
		RouterOsVersion:        stringToText(router.RouterOSVersion),
		BoardName:              stringToText(router.BoardName),
		CpuCount:               intToInt4(router.CPUCount),
		TotalRamMb:             intToInt4(router.TotalRAMMB),
		Identity:               stringToText(router.Identity),
		Status:                 string(router.Status),
		HealthCheckIntervalSec: int32(router.HealthCheckIntervalSec),
		LastOnlineAt:           timePtrToTimestamptz(router.LastOnlineAt),
		Notes:                  stringToText(router.Notes),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrRouterNotFound
		}
		return nil, fmt.Errorf("repository: gagal memperbarui router: %w", err)
	}
	return mapRouterRow(row), nil
}

// SoftDelete melakukan soft-delete router (set deleted_at).
func (r *RouterRepo) SoftDelete(ctx context.Context, id string) error {
	err := r.queries.SoftDeleteRouter(ctx, stringToUUID(id))
	if err != nil {
		return fmt.Errorf("repository: gagal soft-delete router: %w", err)
	}
	return nil
}

// List mengambil daftar router dengan paginasi (tenant-scoped via RLS).
func (r *RouterRepo) List(ctx context.Context, params domain.RouterListParams) (*domain.RouterListResult, error) {
	// Hitung offset dari page dan page_size
	offset := (params.Page - 1) * params.PageSize

	// Ambil total count untuk paginasi
	total, err := r.queries.CountRouters(ctx, CountRoutersParams{
		Status: stringToText(params.Status),
		Search: stringToText(params.Search),
	})
	if err != nil {
		return nil, fmt.Errorf("repository: gagal menghitung total router: %w", err)
	}

	// Ambil data router
	rows, err := r.queries.ListRouters(ctx, ListRoutersParams{
		Limit:  int32(params.PageSize),
		Offset: int32(offset),
		Status: stringToText(params.Status),
		Search: stringToText(params.Search),
	})
	if err != nil {
		return nil, fmt.Errorf("repository: gagal mengambil daftar router: %w", err)
	}

	routers := make([]*domain.Router, 0, len(rows))
	for _, row := range rows {
		routers = append(routers, mapRouterRow(row))
	}

	// Hitung total pages
	totalPages := int(total) / params.PageSize
	if int(total)%params.PageSize > 0 {
		totalPages++
	}

	return &domain.RouterListResult{
		Data:       routers,
		Total:      total,
		Page:       params.Page,
		PageSize:   params.PageSize,
		TotalPages: totalPages,
	}, nil
}

// CountByStatus menghitung jumlah router per status untuk tenant.
func (r *RouterRepo) CountByStatus(ctx context.Context) (map[domain.RouterStatus]int64, error) {
	rows, err := r.queries.CountByStatus(ctx)
	if err != nil {
		return nil, fmt.Errorf("repository: gagal menghitung router per status: %w", err)
	}

	result := make(map[domain.RouterStatus]int64)
	for _, row := range rows {
		result[domain.RouterStatus(row.Status)] = row.Count
	}
	return result, nil
}

// GetActiveRouters mengambil semua router aktif (bukan maintenance, bukan deleted).
func (r *RouterRepo) GetActiveRouters(ctx context.Context) ([]*domain.Router, error) {
	rows, err := r.queries.GetActiveRouters(ctx)
	if err != nil {
		return nil, fmt.Errorf("repository: gagal mengambil router aktif: %w", err)
	}

	routers := make([]*domain.Router, 0, len(rows))
	for _, row := range rows {
		routers = append(routers, mapRouterRow(row))
	}
	return routers, nil
}

// NameExists mengecek apakah nama router sudah ada di tenant.
// excludeID digunakan untuk mengecualikan router tertentu (saat update).
func (r *RouterRepo) NameExists(ctx context.Context, tenantID, name, excludeID string) (bool, error) {
	// Jika excludeID kosong, gunakan UUID nil agar tidak mengecualikan siapapun
	exID := excludeID
	if exID == "" {
		exID = "00000000-0000-0000-0000-000000000000"
	}

	exists, err := r.queries.NameExists(ctx, NameExistsParams{
		TenantID: stringToUUID(tenantID),
		Name:     name,
		ID:       stringToUUID(exID),
	})
	if err != nil {
		return false, fmt.Errorf("repository: gagal mengecek nama router: %w", err)
	}
	return exists, nil
}

// UpdateHealthCheck memperbarui field health check router.
func (r *RouterRepo) UpdateHealthCheck(ctx context.Context, id string, params domain.HealthCheckUpdate) error {
	// Tentukan status string, gunakan empty string jika nil
	status := ""
	if params.Status != nil {
		status = string(*params.Status)
	}

	err := r.queries.UpdateHealthCheck(ctx, UpdateHealthCheckParams{
		ID:            stringToUUID(id),
		LastCheckedAt: timePtrToTimestamptz(params.LastCheckedAt),
		LastOnlineAt:  timePtrToTimestamptz(params.LastOnlineAt),
		LastUptimeSec: int64PtrToInt8(params.LastUptimeSec),
		FailureCount:  int32(params.FailureCount),
		Status:        status,
	})
	if err != nil {
		return fmt.Errorf("repository: gagal memperbarui health check: %w", err)
	}
	return nil
}
