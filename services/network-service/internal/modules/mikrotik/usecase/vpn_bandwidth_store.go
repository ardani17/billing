// Package usecase berisi implementasi business logic untuk network-service.
// File ini mengimplementasikan VPNBandwidthStore menggunakan Redis uruted sets.
// Bandwidth metrics per tunnel disimpan sebagai time-series dengan retensi 24 jam.
package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// ttl24Hours adalah durasi retensi bandwidth metrics (24 jam dalam detik).
const ttl24Hours = 24 * 60 * 60 // 86400 detik

// vpnBandwidthStore mengimplementasikan domain.VPNBandwidthStore menggunakan Redis uruted sets.
// Setiap tunnel memiliki uruted atur dengan key "vpn:bw:{tunnel_id}",
// score = unix timestamp, member = "{unix_timestamp}:{json_metrics}".
type vpnBandwidthStore struct {
	client redis.Cmdable
	logger zerolog.Logger
}

// NewVPNBandwidthStore membuat instance baru vpnBandwidthStore.
func NewVPNBandwidthStore(client redis.Cmdable, logger zerolog.Logger) domain.VPNBandwidthStore {
	return &vpnBandwidthStore{
		client: client,
		logger: logger,
	}
}

// bwKey mengembalikan key Redis uruted atur untuk tunnel tertentu.
func bwKey(tunnelID string) string {
	return fmt.Sprintf("vpn:bw:%s", tunnelID)
}

// Store menyimpan satu data point bandwidth untuk tunnel.
// Metrik di-encode ke JSON dan disimpan di uruted atur dengan score = unix timestamp.
// Setelah ZADD, data lebih tua dari 24 jam dihapus via ZREMRANGEBYSCORE.
func (s *vpnBandwidthStore) Store(ctx context.Context, tunnelID string, metrics domain.VPNBandwidthMetrics) error {
	now := time.Now()
	key := bwKey(tunnelID)

	// Encode metrik ke JSON sebagai member uruted atur
	data, err := json.Marshal(metrics)
	if err != nil {
		return fmt.Errorf("gagal marshal bandwidth metrik: %w", err)
	}

	// Gabungkan timestamp + JSON agar member unik per waktu
	member := fmt.Sprintf("%d:%s", now.Unix(), string(data))

	// ZADD dengan score = unix timestamp
	if err := s.client.ZAdd(ctx, key, redis.Z{
		Score:  float64(now.Unix()),
		Member: member,
	}).Err(); err != nil {
		return fmt.Errorf("gagal ZADD bandwidth metrik: %w", err)
	}

	// Hapus data lebih tua dari 24 jam (enforce TTL)
	cutoff := now.Unix() - ttl24Hours
	if err := s.client.ZRemRangeByScore(ctx, key, "-inf", fmt.Sprintf("%d", cutoff)).Err(); err != nil {
		s.logger.Warn().Err(err).Str("tunnel_id", tunnelID).
			Msg("gagal ZREMRANGEBYSCORE bandwidth metrik")
	}

	return nil
}

// Kueri mengambil data point bandwidth dalam rentang waktu [from, to].
// Mengembalikan slice VPNBandwidthPoint yang sudah uruted ascending berdasarkan timestamp.
func (s *vpnBandwidthStore) Query(ctx context.Context, tunnelID string, from, to time.Time) ([]domain.VPNBandwidthPoint, error) {
	key := bwKey(tunnelID)

	// ZRANGEBYSCORE dengan range [from, to] dalam unix timestamp
	results, err := s.client.ZRangeByScoreWithScores(ctx, key, &redis.ZRangeBy{
		Min: fmt.Sprintf("%d", from.Unix()),
		Max: fmt.Sprintf("%d", to.Unix()),
	}).Result()
	if err != nil {
		return nil, fmt.Errorf("gagal ZRANGEBYSCORE bandwidth metrik: %w", err)
	}

	// Parsing setiap member menjadi VPNBandwidthPoint
	points := make([]domain.VPNBandwidthPoint, 0, len(results))
	for _, z := range results {
		point, err := parseBWMember(z)
		if err != nil {
			continue // skip member yang tidak valid
		}
		points = append(points, point)
	}

	return points, nil
}

// GetLatest mengambil data point bandwidth terbaru untuk tunnel.
// Menggunakan ZREVRANGEBYSCORE dengan limit 1 untuk mendapatkan entry terakhir.
func (s *vpnBandwidthStore) GetLatest(ctx context.Context, tunnelID string) (*domain.VPNBandwidthPoint, error) {
	key := bwKey(tunnelID)

	// ZREVRANGEBYSCORE +inf -inf LIMIT 0 1 - ambil member dengan score tertinggi
	results, err := s.client.ZRevRangeByScoreWithScores(ctx, key, &redis.ZRangeBy{
		Min:    "-inf",
		Max:    "+inf",
		Offset: 0,
		Count:  1,
	}).Result()
	if err != nil {
		return nil, fmt.Errorf("gagal ZREVRANGEBYSCORE bandwidth metrik: %w", err)
	}

	// Tidak ada data bandwidth
	if len(results) == 0 {
		return nil, nil
	}

	point, err := parseBWMember(results[0])
	if err != nil {
		return nil, fmt.Errorf("gagal parse bandwidth metrik terbaru: %w", err)
	}

	return &point, nil
}

// parseBWMember mengekstrak VPNBandwidthPoint dari member uruted atur.
// Format member: "{unix_timestamp}:{json_metrics}"
func parseBWMember(z redis.Z) (domain.VPNBandwidthPoint, error) {
	memberStr, ok := z.Member.(string)
	if !ok {
		return domain.VPNBandwidthPoint{}, fmt.Errorf("member bukan string")
	}

	// Cari posisi ':' pertama untuk memisahkan timestamp dari JSON
	idx := 0
	for i, c := range memberStr {
		if c == ':' {
			idx = i
			break
		}
	}
	if idx == 0 || idx >= len(memberStr)-1 {
		return domain.VPNBandwidthPoint{}, fmt.Errorf("format member tidak valid")
	}

	jsonData := memberStr[idx+1:]

	var metrics domain.VPNBandwidthMetrics
	if err := json.Unmarshal([]byte(jsonData), &metrics); err != nil {
		return domain.VPNBandwidthPoint{}, fmt.Errorf("gagal unmarshal bandwidth metrik: %w", err)
	}

	// Gunakan score sebagai timestamp (lebih akurat daripada prefix member)
	ts := time.Unix(int64(z.Score), 0)

	return domain.VPNBandwidthPoint{
		Timestamp: ts,
		Metrics:   metrics,
	}, nil
}
