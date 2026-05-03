// Package repository menyediakan implementasi repository yang membungkus
// kode sqlc-generated dan memetakan tipe database ke domain entities.
package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// UserRepo mengimplementasikan domain.UserRepository dengan membungkus
// sqlc-generated Queries dan pgxpool.Pool untuk query lintas tenant (RLS bypass).
type UserRepo struct {
	// queries adalah sqlc-generated Queries yang beroperasi dalam konteks tenant (RLS aktif).
	queries *Queries

	// pool digunakan untuk query yang membutuhkan akses lintas tenant (bypass RLS),
	// seperti EmailExistsGlobal dan GetByEmail.
	pool *pgxpool.Pool
}

// NewUserRepo membuat instance baru UserRepo.
// queries digunakan untuk operasi tenant-scoped (RLS aktif).
// pool digunakan untuk operasi lintas tenant (bypass RLS).
func NewUserRepo(queries *Queries, pool *pgxpool.Pool) *UserRepo {
	return &UserRepo{
		queries: queries,
		pool:    pool,
	}
}

// --- Helper functions untuk konversi tipe pgtype ↔ domain ---

// uuidToString mengkonversi pgtype.UUID ke string.
// Mengembalikan string kosong jika UUID tidak valid.
func uuidToString(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	// Format UUID: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
	return fmt.Sprintf("%x-%x-%x-%x-%x", u.Bytes[0:4], u.Bytes[4:6], u.Bytes[6:8], u.Bytes[8:10], u.Bytes[10:16])
}

// stringToUUID mengkonversi string UUID ke pgtype.UUID.
func stringToUUID(s string) pgtype.UUID {
	var u pgtype.UUID
	_ = u.Scan(s)
	return u
}

// textToString mengkonversi pgtype.Text ke string.
// Mengembalikan string kosong jika Text tidak valid (NULL).
func textToString(t pgtype.Text) string {
	if !t.Valid {
		return ""
	}
	return t.String
}

// stringToText mengkonversi string ke pgtype.Text.
// String kosong dikonversi ke NULL (Valid=false).
func stringToText(s string) pgtype.Text {
	if s == "" {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: s, Valid: true}
}

// timestamptzToTimePtr mengkonversi pgtype.Timestamptz ke *time.Time.
// Mengembalikan nil jika Timestamptz tidak valid (NULL).
func timestamptzToTimePtr(t pgtype.Timestamptz) *time.Time {
	if !t.Valid {
		return nil
	}
	return &t.Time
}

// timestamptzToTime mengkonversi pgtype.Timestamptz ke time.Time.
// Mengembalikan zero time jika Timestamptz tidak valid (NULL).
func timestamptzToTime(t pgtype.Timestamptz) time.Time {
	if !t.Valid {
		return time.Time{}
	}
	return t.Time
}

// timeToTimestamptz mengkonversi time.Time ke pgtype.Timestamptz.
func timeToTimestamptz(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: !t.IsZero()}
}

// mapCreateUserRow memetakan User (sqlc model) ke domain.User.
func mapCreateUserRow(row User) *domain.User {
	return &domain.User{
		ID:            uuidToString(row.ID),
		TenantID:      uuidToString(row.TenantID),
		Name:          row.Name,
		Email:         row.Email,
		Phone:         textToString(row.Phone),
		PasswordHash:  textToString(row.PasswordHash),
		Role:          domain.UserRole(row.Role),
		EmailVerified: row.EmailVerified,
		GoogleID:      textToString(row.GoogleID),
		Status:        domain.UserStatus(row.Status),
		LastLogin:     timestamptzToTimePtr(row.LastLogin),
		CreatedAt:     timestamptzToTime(row.CreatedAt),
		UpdatedAt:     timestamptzToTime(row.UpdatedAt),
	}
}

// mapGetUserByIDRow memetakan User (sqlc model) ke domain.User.
func mapGetUserByIDRow(row User) *domain.User {
	return mapCreateUserRow(row)
}

// mapGetUserByEmailRow memetakan User (sqlc model) ke domain.User.
func mapGetUserByEmailRow(row User) *domain.User {
	return mapCreateUserRow(row)
}

// mapGetUserByTenantAndEmailRow memetakan User (sqlc model) ke domain.User.
func mapGetUserByTenantAndEmailRow(row User) *domain.User {
	return mapCreateUserRow(row)
}

// mapGetUserByGoogleIDRow memetakan User (sqlc model) ke domain.User.
func mapGetUserByGoogleIDRow(row User) *domain.User {
	return mapCreateUserRow(row)
}

// mapUpdateUserRow memetakan User (sqlc model) ke domain.User.
func mapUpdateUserRow(row User) *domain.User {
	return mapCreateUserRow(row)
}

// mapListUsersByTenantRow memetakan User (sqlc model) ke domain.User.
func mapListUsersByTenantRow(row User) *domain.User {
	return mapCreateUserRow(row)
}

// --- Implementasi domain.UserRepository ---

// CreateUser membuat user baru dan mengembalikan user yang dibuat.
func (r *UserRepo) CreateUser(ctx context.Context, user *domain.User) (*domain.User, error) {
	row, err := r.queries.CreateUser(ctx, CreateUserParams{
		TenantID:      stringToUUID(user.TenantID),
		Name:          user.Name,
		Email:         user.Email,
		Phone:         stringToText(user.Phone),
		PasswordHash:  stringToText(user.PasswordHash),
		Role:          string(user.Role),
		EmailVerified: user.EmailVerified,
		GoogleID:      stringToText(user.GoogleID),
		Status:        string(user.Status),
	})
	if err != nil {
		return nil, fmt.Errorf("repository: gagal membuat user: %w", err)
	}
	return mapCreateUserRow(row), nil
}

// GetByID mengambil user berdasarkan ID (tenant-scoped via RLS).
func (r *UserRepo) GetByID(ctx context.Context, id string) (*domain.User, error) {
	row, err := r.queries.GetUserByID(ctx, stringToUUID(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("repository: gagal mengambil user by ID: %w", err)
	}
	return mapGetUserByIDRow(row), nil
}

// GetByEmail mengambil user berdasarkan email (lintas tenant, bypass RLS).
// Menggunakan pool langsung tanpa konteks tenant untuk akses global.
func (r *UserRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	// Gunakan pool langsung untuk bypass RLS
	q := New(r.pool)
	row, err := q.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("repository: gagal mengambil user by email: %w", err)
	}
	return mapGetUserByEmailRow(row), nil
}

// GetByTenantAndEmail mengambil user berdasarkan tenant_id dan email.
func (r *UserRepo) GetByTenantAndEmail(ctx context.Context, tenantID, email string) (*domain.User, error) {
	row, err := r.queries.GetUserByTenantAndEmail(ctx, GetUserByTenantAndEmailParams{
		TenantID: stringToUUID(tenantID),
		Email:    email,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("repository: gagal mengambil user by tenant dan email: %w", err)
	}
	return mapGetUserByTenantAndEmailRow(row), nil
}

// GetByGoogleID mengambil user berdasarkan google_id (lintas tenant, bypass RLS).
func (r *UserRepo) GetByGoogleID(ctx context.Context, googleID string) (*domain.User, error) {
	// Gunakan pool langsung untuk bypass RLS karena Google ID bersifat global
	q := New(r.pool)
	row, err := q.GetUserByGoogleID(ctx, stringToText(googleID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("repository: gagal mengambil user by Google ID: %w", err)
	}
	return mapGetUserByGoogleIDRow(row), nil
}

// UpdateUser memperbarui data user (name, phone, role).
func (r *UserRepo) UpdateUser(ctx context.Context, user *domain.User) (*domain.User, error) {
	row, err := r.queries.UpdateUser(ctx, UpdateUserParams{
		ID:    stringToUUID(user.ID),
		Name:  user.Name,
		Phone: stringToText(user.Phone),
		Role:  string(user.Role),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("repository: gagal memperbarui user: %w", err)
	}
	return mapUpdateUserRow(row), nil
}

// UpdateLastLogin memperbarui timestamp last_login user.
func (r *UserRepo) UpdateLastLogin(ctx context.Context, userID string) error {
	err := r.queries.UpdateLastLogin(ctx, stringToUUID(userID))
	if err != nil {
		return fmt.Errorf("repository: gagal memperbarui last_login: %w", err)
	}
	return nil
}

// UpdatePasswordHash memperbarui password_hash user.
func (r *UserRepo) UpdatePasswordHash(ctx context.Context, userID, hash string) error {
	err := r.queries.UpdatePasswordHash(ctx, UpdatePasswordHashParams{
		ID:           stringToUUID(userID),
		PasswordHash: stringToText(hash),
	})
	if err != nil {
		return fmt.Errorf("repository: gagal memperbarui password_hash: %w", err)
	}
	return nil
}

// UpdateStatus memperbarui status user (active/inactive).
func (r *UserRepo) UpdateStatus(ctx context.Context, userID string, status domain.UserStatus) error {
	err := r.queries.UpdateUserStatus(ctx, UpdateUserStatusParams{
		ID:     stringToUUID(userID),
		Status: string(status),
	})
	if err != nil {
		return fmt.Errorf("repository: gagal memperbarui status user: %w", err)
	}
	return nil
}

// LinkGoogleID menambahkan google_id ke user yang sudah ada.
func (r *UserRepo) LinkGoogleID(ctx context.Context, userID, googleID string) error {
	err := r.queries.LinkGoogleID(ctx, LinkGoogleIDParams{
		ID:       stringToUUID(userID),
		GoogleID: stringToText(googleID),
	})
	if err != nil {
		return fmt.Errorf("repository: gagal menautkan Google ID: %w", err)
	}
	return nil
}

// SetEmailVerified mengatur email_verified menjadi true.
func (r *UserRepo) SetEmailVerified(ctx context.Context, userID string) error {
	err := r.queries.SetEmailVerified(ctx, stringToUUID(userID))
	if err != nil {
		return fmt.Errorf("repository: gagal mengatur email_verified: %w", err)
	}
	return nil
}

// DeleteUser menghapus user secara permanen.
func (r *UserRepo) DeleteUser(ctx context.Context, userID string) error {
	err := r.queries.DeleteUser(ctx, stringToUUID(userID))
	if err != nil {
		return fmt.Errorf("repository: gagal menghapus user: %w", err)
	}
	return nil
}

// ListByTenant mengambil semua user dalam satu tenant.
func (r *UserRepo) ListByTenant(ctx context.Context, tenantID string) ([]*domain.User, error) {
	rows, err := r.queries.ListUsersByTenant(ctx, stringToUUID(tenantID))
	if err != nil {
		return nil, fmt.Errorf("repository: gagal mengambil daftar user: %w", err)
	}

	users := make([]*domain.User, 0, len(rows))
	for _, row := range rows {
		users = append(users, mapListUsersByTenantRow(row))
	}
	return users, nil
}

// EmailExistsGlobal mengecek apakah email sudah terdaftar di tenant manapun (bypass RLS).
// Menggunakan pool langsung tanpa konteks tenant untuk akses global.
func (r *UserRepo) EmailExistsGlobal(ctx context.Context, email string) (bool, error) {
	// Gunakan pool langsung untuk bypass RLS
	q := New(r.pool)
	exists, err := q.EmailExistsGlobal(ctx, email)
	if err != nil {
		return false, fmt.Errorf("repository: gagal mengecek email global: %w", err)
	}
	return exists, nil
}
