package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ResellerSessionRepo menyimpan refresh token reseller di tabel terpisah dari
// sessions admin agar tidak melanggar foreign key sessions.user_id -> users.id.
type ResellerSessionRepo struct {
	pool *pgxpool.Pool
}

func NewResellerSessionRepo(pool *pgxpool.Pool) *ResellerSessionRepo {
	return &ResellerSessionRepo{pool: pool}
}

func (r *ResellerSessionRepo) CreateSession(ctx context.Context, session *domain.Session) (*domain.Session, error) {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO reseller_sessions (reseller_id, token_hash, device_info, ip_address, expires_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id::text, reseller_id::text, token_hash, COALESCE(device_info, ''), COALESCE(ip_address, ''), expires_at, created_at
	`, session.UserID, session.TokenHash, nullIfEmpty(session.DeviceInfo), nullIfEmpty(session.IPAddress), session.ExpiresAt)

	var created domain.Session
	if err := row.Scan(
		&created.ID,
		&created.UserID,
		&created.TokenHash,
		&created.DeviceInfo,
		&created.IPAddress,
		&created.ExpiresAt,
		&created.CreatedAt,
	); err != nil {
		return nil, fmt.Errorf("repository: gagal membuat session reseller: %w", err)
	}

	return &created, nil
}

func (r *ResellerSessionRepo) GetByTokenHash(ctx context.Context, tokenHash string) (*domain.Session, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id::text, reseller_id::text, token_hash, COALESCE(device_info, ''), COALESCE(ip_address, ''), expires_at, created_at
		FROM reseller_sessions
		WHERE token_hash = $1 AND expires_at > NOW()
	`, tokenHash)

	var session domain.Session
	if err := row.Scan(
		&session.ID,
		&session.UserID,
		&session.TokenHash,
		&session.DeviceInfo,
		&session.IPAddress,
		&session.ExpiresAt,
		&session.CreatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrTokenNotFound
		}
		return nil, fmt.Errorf("repository: gagal mengambil session reseller by token hash: %w", err)
	}

	return &session, nil
}

func (r *ResellerSessionRepo) DeleteByTokenHash(ctx context.Context, tokenHash string) error {
	if _, err := r.pool.Exec(ctx, `DELETE FROM reseller_sessions WHERE token_hash = $1`, tokenHash); err != nil {
		return fmt.Errorf("repository: gagal menghapus session reseller by token hash: %w", err)
	}
	return nil
}

func (r *ResellerSessionRepo) DeleteByUserID(ctx context.Context, resellerID string) error {
	if _, err := r.pool.Exec(ctx, `DELETE FROM reseller_sessions WHERE reseller_id = $1`, resellerID); err != nil {
		return fmt.Errorf("repository: gagal menghapus semua session reseller: %w", err)
	}
	return nil
}

func (r *ResellerSessionRepo) DeleteExpired(ctx context.Context) error {
	if _, err := r.pool.Exec(ctx, `DELETE FROM reseller_sessions WHERE expires_at <= NOW()`); err != nil {
		return fmt.Errorf("repository: gagal menghapus session reseller expired: %w", err)
	}
	return nil
}

func nullIfEmpty(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
