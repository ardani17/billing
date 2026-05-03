package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
	"github.com/jackc/pgx/v5"
)

// ShareLinkRepo mengimplementasikan domain.ShareLinkRepository dengan membungkus
// DBTX dan memetakan tipe database ke domain.MapShareLink.
type ShareLinkRepo struct {
	db DBTX
}

// NewShareLinkRepo membuat instance baru ShareLinkRepo.
func NewShareLinkRepo(db DBTX) *ShareLinkRepo {
	return &ShareLinkRepo{db: db}
}

// scanShareLink memindai satu baris hasil query ke domain.MapShareLink.
func scanShareLink(row pgx.Row) (*domain.MapShareLink, error) {
	var s domain.MapShareLink
	err := row.Scan(
		&s.ID, &s.TenantID, &s.Token, &s.VisibleLayers,
		&s.ExpiresAt, &s.PasswordHash, &s.AccessCount,
		&s.CreatedBy, &s.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// Create membuat share link baru dan mengembalikan link yang dibuat.
func (r *ShareLinkRepo) Create(ctx context.Context, link *domain.MapShareLink) (*domain.MapShareLink, error) {
	row := r.db.QueryRow(ctx,
		`INSERT INTO map_share_links (tenant_id, token, visible_layers, expires_at, password_hash, created_by)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, tenant_id, token, visible_layers, expires_at, password_hash, access_count, created_by, created_at`,
		link.TenantID, link.Token, link.VisibleLayers,
		link.ExpiresAt, link.PasswordHash, link.CreatedBy,
	)
	result, err := scanShareLink(row)
	if err != nil {
		return nil, fmt.Errorf("repository: gagal membuat share link: %w", err)
	}
	return result, nil
}

// GetByToken mengambil share link berdasarkan token unik.
func (r *ShareLinkRepo) GetByToken(ctx context.Context, token string) (*domain.MapShareLink, error) {
	row := r.db.QueryRow(ctx,
		`SELECT id, tenant_id, token, visible_layers, expires_at, password_hash, access_count, created_by, created_at
		 FROM map_share_links WHERE token = $1`, token,
	)
	result, err := scanShareLink(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrShareLinkNotFound
		}
		return nil, fmt.Errorf("repository: gagal mengambil share link by token: %w", err)
	}
	return result, nil
}

// Delete menghapus share link berdasarkan token.
func (r *ShareLinkRepo) Delete(ctx context.Context, token string) error {
	_, err := r.db.Exec(ctx,
		`DELETE FROM map_share_links WHERE token = $1`, token,
	)
	if err != nil {
		return fmt.Errorf("repository: gagal menghapus share link: %w", err)
	}
	return nil
}

// ListByTenant mengambil daftar share link untuk satu tenant.
func (r *ShareLinkRepo) ListByTenant(ctx context.Context, tenantID string) ([]*domain.MapShareLink, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, tenant_id, token, visible_layers, expires_at, password_hash, access_count, created_by, created_at
		 FROM map_share_links WHERE tenant_id = $1 ORDER BY created_at DESC`, tenantID,
	)
	if err != nil {
		return nil, fmt.Errorf("repository: gagal mengambil share links: %w", err)
	}
	defer rows.Close()

	var results []*domain.MapShareLink
	for rows.Next() {
		var s domain.MapShareLink
		err := rows.Scan(
			&s.ID, &s.TenantID, &s.Token, &s.VisibleLayers,
			&s.ExpiresAt, &s.PasswordHash, &s.AccessCount,
			&s.CreatedBy, &s.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("repository: gagal scan share link: %w", err)
		}
		results = append(results, &s)
	}
	return results, rows.Err()
}

// IncrementAccessCount menaikkan access_count share link saat diakses.
func (r *ShareLinkRepo) IncrementAccessCount(ctx context.Context, token string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE map_share_links SET access_count = access_count + 1 WHERE token = $1`, token,
	)
	if err != nil {
		return fmt.Errorf("repository: gagal increment access count: %w", err)
	}
	return nil
}

// Compile-time check: ShareLinkRepo mengimplementasikan domain.ShareLinkRepository.
var _ domain.ShareLinkRepository = (*ShareLinkRepo)(nil)
