// Package usecase berisi implementasi business logic untuk network-service.
// File ini mendefinisikan GeocodingManager: reverse geocoding dengan cache.
package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// nominatimBaseURL adalah URL dasar untuk Nominatim reverse geocoding API.
const nominatimBaseURL = "https://nominatim.openstreetmap.org/reverse"

// nominatimUserAgent adalah User-Agent yang digunakan untuk request ke Nominatim.
// Nominatim memerlukan User-Agent yang valid sesuai kebijakan penggunaan.
const nominatimUserAgent = "ISPBoss/1.0"

// Compile-time check: geocodingManager harus mengimplementasikan domain.GeocodingManager.
var _ domain.GeocodingManager = (*geocodingManager)(nil)

// HTTPClient mendefinisikan interface untuk HTTP client (untuk testing).
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// geocodingManager mengimplementasikan domain.GeocodingManager.
// Menggunakan cache PostgreSQL (TTL 30 hari) dan provider Nominatim.
type geocodingManager struct {
	cacheRepo domain.GeocodingCacheRepository
	client    HTTPClient
}

// NewGeocodingManager membuat instance GeocodingManager baru.
// Parameter client bisa nil, akan menggunakan http.DefaultClient.
func NewGeocodingManager(
	cacheRepo domain.GeocodingCacheRepository,
	client HTTPClient,
) domain.GeocodingManager {
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	return &geocodingManager{
		cacheRepo: cacheRepo,
		client:    client,
	}
}

// ReverseGeocode mengkonversi koordinat GPS menjadi alamat lengkap.
// Langkah: round koordinat → cek cache → jika miss, panggil Nominatim → simpan cache.
// Jika provider gagal, mengembalikan koordinat tanpa alamat (graceful degradation).
func (m *geocodingManager) ReverseGeocode(ctx context.Context, tenantID string, lat, lng float64) (*domain.GeocodingResult, error) {
	// Validasi koordinat
	if err := domain.ValidateCoordinate(lat, lng); err != nil {
		return nil, err
	}

	// Bulatkan koordinat ke 5 desimal untuk cache key
	latRound := domain.RoundCoordinate(lat)
	lngRound := domain.RoundCoordinate(lng)

	// Cek cache
	cached, err := m.cacheRepo.Get(ctx, latRound, lngRound)
	if err != nil {
		log.Warn().Err(err).Msg("gagal mengambil cache geocoding")
	}

	if cached != nil {
		// Cache hit — kembalikan hasil dari cache
		return buildResultFromCache(cached, lat, lng), nil
	}

	// Cache miss — panggil provider Nominatim
	result, rawJSON, err := m.callNominatim(lat, lng)
	if err != nil {
		log.Warn().Err(err).
			Float64("lat", lat).Float64("lng", lng).
			Msg("reverse geocoding gagal, mengembalikan koordinat tanpa alamat")

		// Graceful degradation: kembalikan koordinat tanpa alamat
		errMsg := err.Error()
		return &domain.GeocodingResult{
			Latitude:  lat,
			Longitude: lng,
			Error:     &errMsg,
		}, nil
	}

	// Simpan ke cache
	cache := &domain.GeocodingCache{
		ID:        uuid.New().String(),
		TenantID:  tenantID,
		LatRound:  latRound,
		LngRound:  lngRound,
		Address:   result.Address,
		RawJSON:   rawJSON,
		ExpiresAt: domain.CacheExpiresAt(),
	}

	if err := m.cacheRepo.Set(ctx, cache); err != nil {
		log.Warn().Err(err).Msg("gagal menyimpan cache geocoding")
	}

	return result, nil
}

// nominatimResponse adalah struktur respons dari Nominatim API.
type nominatimResponse struct {
	DisplayName string           `json:"display_name"`
	Address     nominatimAddress `json:"address"`
}

// nominatimAddress berisi detail alamat dari Nominatim.
type nominatimAddress struct {
	Road        string `json:"road"`
	Village     string `json:"village"`
	Suburb      string `json:"suburb"`
	City        string `json:"city"`
	County      string `json:"county"`
	State       string `json:"state"`
	PostCode    string `json:"postcode"`
	Country     string `json:"country"`
	CountryCode string `json:"country_code"`
}

// callNominatim memanggil Nominatim reverse geocoding API.
// Mengembalikan GeocodingResult, raw JSON response, dan error.
func (m *geocodingManager) callNominatim(lat, lng float64) (*domain.GeocodingResult, json.RawMessage, error) {
	// Bangun URL request
	params := url.Values{}
	params.Set("format", "json")
	params.Set("lat", fmt.Sprintf("%.6f", lat))
	params.Set("lon", fmt.Sprintf("%.6f", lng))
	params.Set("addressdetails", "1")
	params.Set("accept-language", "id")

	reqURL := nominatimBaseURL + "?" + params.Encode()

	req, err := http.NewRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("gagal membuat request: %w", err)
	}
	req.Header.Set("User-Agent", nominatimUserAgent)

	resp, err := m.client.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: %v", domain.ErrGeocodingFailed, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, nil, domain.ErrGeocodingRateLimit
	}

	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("%w: status %d", domain.ErrGeocodingFailed, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("gagal membaca response: %w", err)
	}

	var nomResp nominatimResponse
	if err := json.Unmarshal(body, &nomResp); err != nil {
		return nil, nil, fmt.Errorf("gagal parse response Nominatim: %w", err)
	}

	result := &domain.GeocodingResult{
		Address:    nomResp.DisplayName,
		Street:     nomResp.Address.Road,
		Kelurahan:  nomResp.Address.Village,
		Kecamatan:  nomResp.Address.Suburb,
		City:       nomResp.Address.City,
		Province:   nomResp.Address.State,
		PostalCode: nomResp.Address.PostCode,
		Latitude:   lat,
		Longitude:  lng,
	}

	return result, body, nil
}

// buildResultFromCache membangun GeocodingResult dari data cache.
func buildResultFromCache(cache *domain.GeocodingCache, lat, lng float64) *domain.GeocodingResult {
	result := &domain.GeocodingResult{
		Address:   cache.Address,
		Latitude:  lat,
		Longitude: lng,
	}

	// Parse raw JSON untuk detail alamat jika tersedia
	if cache.RawJSON != nil {
		var nomResp nominatimResponse
		if err := json.Unmarshal(cache.RawJSON, &nomResp); err == nil {
			result.Street = nomResp.Address.Road
			result.Kelurahan = nomResp.Address.Village
			result.Kecamatan = nomResp.Address.Suburb
			result.City = nomResp.Address.City
			result.Province = nomResp.Address.State
			result.PostalCode = nomResp.Address.PostCode
		}
	}

	return result
}
