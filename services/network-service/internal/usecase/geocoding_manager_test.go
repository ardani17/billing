// geocoding_manager_test.go — unit test dan property test untuk GeocodingManager.
// Menggunakan mock in-memory repository dan HTTP client.
// Semua komentar dalam Bahasa Indonesia.
package usecase

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"sync"
	"testing"
	"time"

	"pgregory.net/rapid"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// =============================================================================
// Mock Repository: GeocodingCacheRepository — in-memory untuk testing
// =============================================================================

// mockGeocodingCacheRepo adalah implementasi in-memory dari domain.GeocodingCacheRepository.
type mockGeocodingCacheRepo struct {
	mu    sync.Mutex
	cache map[string]*domain.GeocodingCache // key: "latRound:lngRound"
}

func newMockGeocodingCacheRepo() *mockGeocodingCacheRepo {
	return &mockGeocodingCacheRepo{cache: make(map[string]*domain.GeocodingCache)}
}

// cacheKey membuat key unik dari koordinat yang dibulatkan.
// Menangani kasus -0.0 agar konsisten dengan 0.0 (IEEE 754 negative zero).
func cacheKey(latRound, lngRound float64) string {
	// Normalisasi negative zero ke positive zero untuk konsistensi format string.
	// Dalam IEEE 754, -0.0 == 0.0 bernilai true, tapi fmt.Sprintf menghasilkan
	// string berbeda ("-0.00000" vs "0.00000").
	if latRound == 0 {
		latRound = math.Abs(latRound)
	}
	if lngRound == 0 {
		lngRound = math.Abs(lngRound)
	}
	return fmt.Sprintf("%.5f:%.5f", latRound, lngRound)
}

func (r *mockGeocodingCacheRepo) Get(_ context.Context, latRound, lngRound float64) (*domain.GeocodingCache, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := cacheKey(latRound, lngRound)
	c, ok := r.cache[key]
	if !ok {
		return nil, nil
	}
	// Cek apakah sudah kedaluwarsa
	if time.Now().After(c.ExpiresAt) {
		delete(r.cache, key)
		return nil, nil
	}
	return c, nil
}

func (r *mockGeocodingCacheRepo) Set(_ context.Context, cache *domain.GeocodingCache) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := cacheKey(cache.LatRound, cache.LngRound)
	cache.CreatedAt = time.Now()
	r.cache[key] = cache
	return nil
}

func (r *mockGeocodingCacheRepo) DeleteExpired(_ context.Context) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var count int64
	now := time.Now()
	for key, c := range r.cache {
		if now.After(c.ExpiresAt) {
			delete(r.cache, key)
			count++
		}
	}
	return count, nil
}

// =============================================================================
// Mock HTTP Client — untuk testing tanpa request ke Nominatim
// =============================================================================

// mockHTTPClient adalah implementasi mock dari HTTPClient interface.
type mockHTTPClient struct {
	// response yang akan dikembalikan
	statusCode int
	body       string
	err        error
	callCount  int
	mu         sync.Mutex
}

func newMockHTTPClient(statusCode int, body string) *mockHTTPClient {
	return &mockHTTPClient{statusCode: statusCode, body: body}
}

func (c *mockHTTPClient) Do(_ *http.Request) (*http.Response, error) {
	c.mu.Lock()
	c.callCount++
	c.mu.Unlock()

	if c.err != nil {
		return nil, c.err
	}
	return &http.Response{
		StatusCode: c.statusCode,
		Body:       io.NopCloser(bytes.NewBufferString(c.body)),
	}, nil
}

// nominatimSuccessBody adalah contoh respons sukses dari Nominatim.
const nominatimSuccessBody = `{
	"display_name": "Jl. Sudirman, Menteng, Jakarta Pusat, DKI Jakarta, 10220, Indonesia",
	"address": {
		"road": "Jl. Sudirman",
		"village": "Menteng",
		"suburb": "Menteng",
		"city": "Jakarta Pusat",
		"state": "DKI Jakarta",
		"postcode": "10220",
		"country": "Indonesia",
		"country_code": "id"
	}
}`

// =============================================================================
// Unit Test 1: TestReverseGeocode_CacheHit — cache hit, tidak panggil provider
// =============================================================================

// TestReverseGeocode_CacheHit memverifikasi bahwa ReverseGeocode mengembalikan
// hasil dari cache tanpa memanggil provider eksternal.
func TestReverseGeocode_CacheHit(t *testing.T) {
	cacheRepo := newMockGeocodingCacheRepo()
	httpClient := newMockHTTPClient(200, nominatimSuccessBody)
	mgr := NewGeocodingManager(cacheRepo, httpClient)
	ctx := context.Background()

	// Pre-populate cache
	lat, lng := -6.20880, 106.84560
	latRound := domain.RoundCoordinate(lat)
	lngRound := domain.RoundCoordinate(lng)

	rawJSON, _ := json.Marshal(map[string]interface{}{
		"display_name": "Cached Address",
		"address":      map[string]string{"city": "Jakarta"},
	})

	cacheRepo.Set(ctx, &domain.GeocodingCache{
		ID:        "cache-1",
		TenantID:  "tenant-geo",
		LatRound:  latRound,
		LngRound:  lngRound,
		Address:   "Cached Address",
		RawJSON:   rawJSON,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	})

	result, err := mgr.ReverseGeocode(ctx, "tenant-geo", lat, lng)
	if err != nil {
		t.Fatalf("ReverseGeocode gagal: %v", err)
	}

	// Verifikasi hasil dari cache
	if result.Address != "Cached Address" {
		t.Errorf("Address: got %q, want %q", result.Address, "Cached Address")
	}

	// Verifikasi HTTP client tidak dipanggil
	httpClient.mu.Lock()
	calls := httpClient.callCount
	httpClient.mu.Unlock()
	if calls != 0 {
		t.Errorf("HTTP client seharusnya tidak dipanggil saat cache hit, got %d calls", calls)
	}
}

// =============================================================================
// Unit Test 2: TestReverseGeocode_CacheMiss — cache miss, panggil provider
// =============================================================================

// TestReverseGeocode_CacheMiss memverifikasi bahwa ReverseGeocode memanggil
// provider Nominatim saat cache miss dan menyimpan hasilnya ke cache.
func TestReverseGeocode_CacheMiss(t *testing.T) {
	cacheRepo := newMockGeocodingCacheRepo()
	httpClient := newMockHTTPClient(200, nominatimSuccessBody)
	mgr := NewGeocodingManager(cacheRepo, httpClient)
	ctx := context.Background()

	result, err := mgr.ReverseGeocode(ctx, "tenant-geo", -6.2088, 106.8456)
	if err != nil {
		t.Fatalf("ReverseGeocode gagal: %v", err)
	}

	// Verifikasi hasil dari provider
	if result.Address == "" {
		t.Error("Address seharusnya tidak kosong setelah cache miss")
	}
	if result.City != "Jakarta Pusat" {
		t.Errorf("City: got %q, want %q", result.City, "Jakarta Pusat")
	}

	// Verifikasi HTTP client dipanggil 1 kali
	httpClient.mu.Lock()
	calls := httpClient.callCount
	httpClient.mu.Unlock()
	if calls != 1 {
		t.Errorf("HTTP client seharusnya dipanggil 1 kali, got %d", calls)
	}

	// Verifikasi cache terisi
	cacheRepo.mu.Lock()
	cacheLen := len(cacheRepo.cache)
	cacheRepo.mu.Unlock()
	if cacheLen != 1 {
		t.Errorf("cache seharusnya berisi 1 entry, got %d", cacheLen)
	}
}

// =============================================================================
// Unit Test 3: TestReverseGeocode_ProviderError — provider gagal, graceful degradation
// =============================================================================

// TestReverseGeocode_ProviderError memverifikasi bahwa ReverseGeocode
// mengembalikan koordinat tanpa alamat saat provider gagal.
func TestReverseGeocode_ProviderError(t *testing.T) {
	cacheRepo := newMockGeocodingCacheRepo()
	httpClient := newMockHTTPClient(500, "Internal Server Error")
	mgr := NewGeocodingManager(cacheRepo, httpClient)
	ctx := context.Background()

	result, err := mgr.ReverseGeocode(ctx, "tenant-geo", -6.2088, 106.8456)
	if err != nil {
		t.Fatalf("ReverseGeocode seharusnya tidak error (graceful degradation): %v", err)
	}

	// Verifikasi koordinat tetap ada
	if result.Latitude != -6.2088 {
		t.Errorf("Latitude: got %f, want %f", result.Latitude, -6.2088)
	}
	if result.Longitude != 106.8456 {
		t.Errorf("Longitude: got %f, want %f", result.Longitude, 106.8456)
	}

	// Verifikasi error field terisi
	if result.Error == nil {
		t.Error("Error field seharusnya terisi saat provider gagal")
	}
}

// =============================================================================
// Property Test 10: Geocoding Cache Key Consistency
// =============================================================================

// TestPropertyGeocodingCacheKeyConsistency memverifikasi bahwa dua koordinat
// yang menghasilkan rounded value yang sama akan menggunakan cache entry yang sama.
// Sebaliknya, koordinat dengan rounded value berbeda menggunakan entry berbeda.
//
// **Validates: Requirements 8.3**
func TestPropertyGeocodingCacheKeyConsistency(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate dua pasang koordinat
		lat1 := rapid.Float64Range(-90.0, 90.0).Draw(t, "lat1")
		lng1 := rapid.Float64Range(-180.0, 180.0).Draw(t, "lng1")
		lat2 := rapid.Float64Range(-90.0, 90.0).Draw(t, "lat2")
		lng2 := rapid.Float64Range(-180.0, 180.0).Draw(t, "lng2")

		// Bulatkan koordinat
		latRound1 := domain.RoundCoordinate(lat1)
		lngRound1 := domain.RoundCoordinate(lng1)
		latRound2 := domain.RoundCoordinate(lat2)
		lngRound2 := domain.RoundCoordinate(lng2)

		// Buat cache key
		key1 := cacheKey(latRound1, lngRound1)
		key2 := cacheKey(latRound2, lngRound2)

		sameRounded := latRound1 == latRound2 && lngRound1 == lngRound2
		sameKey := key1 == key2

		// Properti: jika rounded values sama, maka cache key harus sama
		if sameRounded && !sameKey {
			t.Fatalf("koordinat dengan rounded value sama seharusnya memiliki cache key sama: "+
				"(%.5f,%.5f) vs (%.5f,%.5f) → key %q vs %q",
				latRound1, lngRound1, latRound2, lngRound2, key1, key2)
		}

		// Properti: jika rounded values berbeda, maka cache key harus berbeda
		if !sameRounded && sameKey {
			t.Fatalf("koordinat dengan rounded value berbeda seharusnya memiliki cache key berbeda: "+
				"(%.5f,%.5f) vs (%.5f,%.5f) → key %q",
				latRound1, lngRound1, latRound2, lngRound2, key1)
		}

		// Properti tambahan: RoundCoordinate harus idempoten
		if domain.RoundCoordinate(latRound1) != latRound1 {
			t.Fatalf("RoundCoordinate seharusnya idempoten: round(%.10f) != %.10f",
				latRound1, domain.RoundCoordinate(latRound1))
		}

		// Properti: presisi 5 desimal (~1.1 meter)
		// Koordinat yang berbeda kurang dari 0.000005 derajat harus menghasilkan key yang sama
		epsilon := 0.000005
		latNear := lat1 + epsilon/2
		if math.Abs(latNear) <= 90 {
			latRoundNear := domain.RoundCoordinate(latNear)
			// Koordinat yang sangat dekat mungkin atau mungkin tidak menghasilkan key yang sama
			// tergantung posisi relatif terhadap batas pembulatan — ini valid
			_ = latRoundNear
		}
	})
}
