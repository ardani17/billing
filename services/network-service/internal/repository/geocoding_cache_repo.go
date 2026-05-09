package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
	"github.com/jackc/pgx/v5"
)

// GeocodingCacheRepo mengimplementasikan domain.GeocodingCacheRepository dengan membungkus
// DBTX dan memetakan tipe database ke domain.GeocodingCache.
type GeocodingCacheRepo struct {
	db DBTX
}

// NewGeocodingCacheRepo membuat instance baru GeocodingCacheRepo.
func NewGeocodingCacheRepo(db DBTX) *GeocodingCacheRepo {
	return &GeocodingCacheRepo{db: db}
}

// Get mengambil cache geocoding berdasarkan koordinat yang sudah dibulatkan.
// Mengembalikan nil jika cache tidak ditemukan atau sudah kedaluwarsa.
// tenantID diambil dari context RLS, tapi kueri tetap menerima parameter eksplisit
// sesuai pola sqlc query yang sudah didefinisikan.
func (r *GeocodingCacheRepo) Get(ctx context.Context, latRound, lngRound float64) (*domain.GeocodingCache, error) {
	var c domain.GeocodingCache
	err := r.db.QueryRow(ctx,
		`SELECT id, tenant_id, lat_round, lng_round, address, raw_json, expires_at, created_at
		 FROM geocoding_cache
		 WHERE lat_round = $1 AND lng_round = $2 AND expires_at > NOW()`,
		latRound, lngRound,
	).Scan(
		&c.ID, &c.TenantID, &c.LatRound, &c.LngRound,
		&c.Address, &c.RawJSON, &c.ExpiresAt, &c.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("repository: gagal mengambil geocoding cache: %w", err)
	}
	return &c, nil
}

// Set menyimpan atau memperbarui cache geocoding (upsert).
func (r *GeocodingCacheRepo) Set(ctx context.Context, cache *domain.GeocodingCache) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO geocoding_cache (tenant_id, lat_round, lng_round, address, raw_json, expires_at)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 ON CONFLICT (tenant_id, lat_round, lng_round) DO UPDATE SET
			address = EXCLUDED.address,
			raw_json = EXCLUDED.raw_json,
			expires_at = EXCLUDED.expires_at`,
		cache.TenantID, cache.LatRound, cache.LngRound,
		cache.Address, cache.RawJSON, cache.ExpiresAt,
	)
	if err != nil {
		return fmt.Errorf("repository: gagal menyimpan geocoding cache: %w", err)
	}
	return nil
}

// DeleteExpired menghapus cache yang sudah kedaluwarsa (expires_at < now).
func (r *GeocodingCacheRepo) DeleteExpired(ctx context.Context) (int64, error) {
	tag, err := r.db.Exec(ctx,
		`DELETE FROM geocoding_cache WHERE expires_at < NOW()`,
	)
	if err != nil {
		return 0, fmt.Errorf("repository: gagal menghapus expired geocoding cache: %w", err)
	}
	return tag.RowsAffected(), nil
}

// Compile-time cek: GeocodingCacheRepo mengimplementasikan domain.GeocodingCacheRepository.
var _ domain.GeocodingCacheRepository = (*GeocodingCacheRepo)(nil)
