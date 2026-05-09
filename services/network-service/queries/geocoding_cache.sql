-- Kueri SQL untuk operasi pada tabel geocoding_cache.
-- Digunakan oleh sqlc untuk menghasilkan kode Go yang type-safe.
-- Tabel geocoding_cache dilindungi RLS, kueri hanya mengembalikan baris milik tenant aktif.
-- Cache entries memiliki TTL (expires_at) dan dibersihkan secara periodik.

-- name: GetGeocodingCache :one
SELECT id, tenant_id, lat_round, lng_round,
    address, raw_json, expires_at, created_at
FROM geocoding_cache
WHERE tenant_id = $1 AND lat_round = $2 AND lng_round = $3
  AND expires_at > NOW();

-- name: SetGeocodingCache :exec
INSERT INTO geocoding_cache (
    tenant_id, lat_round, lng_round,
    address, raw_json, expires_at
) VALUES (
    $1, $2, $3,
    $4, $5, $6
)
ON CONFLICT (tenant_id, lat_round, lng_round) DO UPDATE SET
    address = EXCLUDED.address,
    raw_json = EXCLUDED.raw_json,
    expires_at = EXCLUDED.expires_at;

-- name: DeleteExpiredGeocodingCache :execrows
DELETE FROM geocoding_cache
WHERE expires_at < NOW();
