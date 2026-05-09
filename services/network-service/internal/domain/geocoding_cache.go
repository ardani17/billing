package domain

import (
	"encoding/json"
	"math"
	"time"
)

// =============================================================================
// Konstanta Geocoding Cache - konfigurasi TTL dan presisi koordinat
// =============================================================================

// CacheTTLDays adalah durasi cache hasil reverse geocoding dalam hari.
// Setelah 30 hari, cache dianggap kedaluwarsa dan akan di-refresh
// dari provider geocoding (Nominatim/Google).
const CacheTTLDays = 30

// coordinatePrecision adalah jumlah desimal untuk pembulatan koordinat.
// 5 desimal memberikan presisi ~1.1 meter, cukup untuk geocoding cache.
const coordinatePrecision = 5

// =============================================================================
// GeocodingCache Entitas - cache hasil reverse geocoding per koordinat
// =============================================================================

// GeocodingCache merepresentasikan cache hasil reverse geocoding.
// Koordinat dibulatkan ke 5 desimal sebagai cache key untuk mengurangi
// permintaan ke provider eksternal (Nominatim/Google Geocoding).
// Data diisolasi per tenant via RLS di PostgreSQL.
type GeocodingCache struct {
	ID        string          `json:"id"`
	TenantID  string          `json:"tenant_id"`
	LatRound  float64         `json:"lat_round"`
	LngRound  float64         `json:"lng_round"`
	Address   string          `json:"address"`
	RawJSON   json.RawMessage `json:"raw_json"`
	ExpiresAt time.Time       `json:"expires_at"`
	CreatedAt time.Time       `json:"created_at"`
}

// =============================================================================
// Fungsi bantu Functions - utilitas untuk geocoding cache
// =============================================================================

// RoundCoordinate membulatkan koordinat ke 5 desimal.
// Presisi 5 desimal (~1.1 meter) digunakan sebagai cache key
// agar koordinat yang sangat berdekatan menghasilkan cache hit yang sama.
func RoundCoordinate(coord float64) float64 {
	factor := math.Pow(10, float64(coordinatePrecision))
	return math.Round(coord*factor) / factor
}

// CacheExpiresAt mengembalikan waktu kedaluwarsa cache berdasarkan CacheTTLDays.
// Digunakan saat menyimpan entry baru ke geocoding cache.
func CacheExpiresAt() time.Time {
	return time.Now().AddDate(0, 0, CacheTTLDays)
}
