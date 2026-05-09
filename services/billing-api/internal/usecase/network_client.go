// Package usecase berisi business logic untuk billing-api.
// NetworkClient mengimplementasikan domain.NetworkServiceClient untuk
// komunikasi HTTP dengan network-service. Mendukung graceful degradation:
// jika network-service down, data diambil dari Redis cache (stale).
// Jika tidak ada cache, respons dikembalikan dengan module_inactive=true.
package usecase

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

// networkCacheTTL adalah TTL cache Redis untuk data network (1 jam).
const networkCacheTTL = 1 * time.Hour

// networkHTTPTimeout adalah timeout per HTTP permintaan ke network-service (10 detik).
const networkHTTPTimeout = 10 * time.Second

// Compile-time cek: NetworkClient harus mengimplementasikan domain.NetworkServiceClient.
var _ domain.NetworkServiceClient = (*NetworkClient)(nil)

// NetworkClient mengimplementasikan domain.NetworkServiceClient.
// Melakukan HTTP GET ke network-service dan menyimpan respons di Redis cache.
type NetworkClient struct {
	baseURL    string
	httpClient *http.Client
	redis      *redis.Client
	logger     zerolog.Logger
}

// NewNetworkClient membuat instance baru NetworkClient.
func NewNetworkClient(baseURL string, redisClient *redis.Client, logger zerolog.Logger) *NetworkClient {
	return &NetworkClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: networkHTTPTimeout,
		},
		redis:  redisClient,
		logger: logger.With().Str("component", "network_client").Logger(),
	}
}

// cacheKey menghasilkan Redis key untuk cache network report.
// Format: report:network:{reportType}:{tenantID}:{filterHash}
func cacheKey(reportType, tenantID, filterHash string) string {
	return fmt.Sprintf("report:network:%s:%s:%s", reportType, tenantID, filterHash)
}

// filterHash menghasilkan MD5 hash dari parameter filter untuk cache key.
func filterHash(params url.Values) string {
	return fmt.Sprintf("%x", md5.Sum([]byte(params.Encode())))
}

// buildURL membangun URL lengkap untuk permintaan ke network-service.
func (nc *NetworkClient) buildURL(reportType, tenantID string, periodStart, periodEnd time.Time, extra map[string]string) (string, url.Values) {
	endpoint := fmt.Sprintf("%s/internal/v1/reports/%s", nc.baseURL, reportType)
	params := url.Values{}
	params.Set("tenant_id", tenantID)
	if !periodStart.IsZero() {
		params.Set("period_start", periodStart.Format(time.RFC3339))
	}
	if !periodEnd.IsZero() {
		params.Set("period_end", periodEnd.Format(time.RFC3339))
	}
	for k, v := range extra {
		if v != "" {
			params.Set(k, v)
		}
	}
	return endpoint + "?" + params.Encode(), params
}

// cachedResponse menyimpan data respons beserta timestamp untuk cache.
type cachedResponse struct {
	Data     json.RawMessage `json:"data"`
	CachedAt time.Time       `json:"cached_at"`
}

// Jika gagal, coba ambil dari cache (stale). Jika tidak ada cache, mengembalikan nil.
// Parameter result harus pointer ke struct tujuan (e.g. *domain.UptimeReport).
func (nc *NetworkClient) fetchAndCache(ctx context.Context, reportType, tenantID string, reqURL string, params url.Values, result interface{}) (stale bool, lastUpdated *time.Time, err error) {
	key := cacheKey(reportType, tenantID, filterHash(params))

	// Coba HTTP GET ke network-service
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		nc.logger.Error().Err(err).Str("report_type", reportType).Msg("gagal membuat HTTP request")
		return nc.fallbackFromCache(ctx, key, result)
	}

	resp, err := nc.httpClient.Do(req)
	if err != nil {
		nc.logger.Warn().Err(err).Str("report_type", reportType).Msg("network-service tidak tersedia, coba cache")
		return nc.fallbackFromCache(ctx, key, result)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		nc.logger.Warn().Int("status", resp.StatusCode).Str("report_type", reportType).Msg("network-service response error, coba cache")
		return nc.fallbackFromCache(ctx, key, result)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		nc.logger.Error().Err(err).Str("report_type", reportType).Msg("gagal membaca response body")
		return nc.fallbackFromCache(ctx, key, result)
	}

	// Parsing respons ke struct tujuan
	if err := json.Unmarshal(body, result); err != nil {
		nc.logger.Error().Err(err).Str("report_type", reportType).Msg("gagal parse JSON response")
		return nc.fallbackFromCache(ctx, key, result)
	}

	// Simpan ke Redis cache
	cached := cachedResponse{
		Data:     body,
		CachedAt: time.Now(),
	}
	cacheData, _ := json.Marshal(cached)
	if nc.redis != nil {
		if err := nc.redis.Set(ctx, key, cacheData, networkCacheTTL).Err(); err != nil {
			nc.logger.Warn().Err(err).Str("key", key).Msg("gagal menyimpan cache network report")
		}
	}

	return false, nil, nil
}

// cadanganFromCache mengambil data dari Redis cache sebagai cadangan.
// Mengembalikan stale=true dan lastUpdated jika cache ditemukan.
// Mengembalikan stale=false, nil, nil jika tidak ada cache (module_inactive).
func (nc *NetworkClient) fallbackFromCache(ctx context.Context, key string, result interface{}) (stale bool, lastUpdated *time.Time, err error) {
	if nc.redis == nil {
		// Tidak ada Redis client - module dianggap inactive
		nc.logger.Info().Str("key", key).Msg("redis tidak tersedia, module dianggap inactive")
		return false, nil, nil
	}

	data, err := nc.redis.Get(ctx, key).Bytes()
	if err != nil {
		// Tidak ada cache - module dianggap inactive
		nc.logger.Info().Str("key", key).Msg("tidak ada cache, module dianggap inactive")
		return false, nil, nil
	}

	var cached cachedResponse
	if err := json.Unmarshal(data, &cached); err != nil {
		nc.logger.Error().Err(err).Str("key", key).Msg("gagal parse cached data")
		return false, nil, nil
	}

	// Parsing cached data ke struct tujuan
	if err := json.Unmarshal(cached.Data, result); err != nil {
		nc.logger.Error().Err(err).Str("key", key).Msg("gagal parse cached JSON ke struct")
		return false, nil, nil
	}

	return true, &cached.CachedAt, nil
}
