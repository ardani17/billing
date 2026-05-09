// Package metrics - implementasi TrafficStore menggunakan Redis uruted sets.
// Menyimpan traffic data PON port sebagai time-series dengan retensi 7 hari.
package metrics

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
	"github.com/redis/go-redis/v9"
)

// ttlTraffic7Days adalah durasi retensi traffic data (7 hari).
const ttlTraffic7Days = 7 * 24 * time.Hour

// Compile-time cek: redisTrafficStore mengimplementasikan domain.TrafficStore.
var _ domain.TrafficStore = (*redisTrafficStore)(nil)

// redisTrafficStore mengimplementasikan domain.TrafficStore menggunakan Redis uruted sets.
// Setiap kombinasi OLT/port memiliki uruted atur dengan key "olt:traffic:{olt_id}:{port}",
// score = unix timestamp, member = JSON-encoded PONTrafficPoint.
type redisTrafficStore struct {
	client redis.Cmdable
}

// NewRedisTrafficStore membuat instance baru redisTrafficStore.
func NewRedisTrafficStore(client redis.Cmdable) domain.TrafficStore {
	return &redisTrafficStore{client: client}
}

// trafficKey mengembalikan key Redis uruted atur untuk traffic PON port tertentu.
func trafficKey(oltID string, portIndex int) string {
	return fmt.Sprintf("olt:traffic:%s:%d", oltID, portIndex)
}

// Store menyimpan satu data point traffic untuk PON port.
// Data di-encode ke JSON dan disimpan di uruted atur dengan score = unix timestamp.
// TTL 7 hari di-atur via EXPIRE setiap kali Store dipanggil.
func (s *redisTrafficStore) Store(ctx context.Context, oltID string, portIndex int, traffic domain.PONTrafficPoint) error {
	key := trafficKey(oltID, portIndex)

	// Encode traffic ke JSON sebagai member uruted atur
	data, err := json.Marshal(traffic)
	if err != nil {
		return fmt.Errorf("gagal marshal traffic: %w", err)
	}

	// Gabungkan timestamp + JSON agar member unik per waktu
	member := fmt.Sprintf("%d:%s", traffic.Timestamp.Unix(), string(data))

	// ZADD dengan score = unix timestamp
	if err := s.client.ZAdd(ctx, key, redis.Z{
		Score:  float64(traffic.Timestamp.Unix()),
		Member: member,
	}).Err(); err != nil {
		return fmt.Errorf("gagal ZADD traffic: %w", err)
	}

	// Set TTL 7 hari pada key
	if err := s.client.Expire(ctx, key, ttlTraffic7Days).Err(); err != nil {
		return fmt.Errorf("gagal set EXPIRE traffic: %w", err)
	}

	return nil
}

// Kueri mengambil data point traffic dalam rentang waktu [from, to].
// Mengembalikan slice PONTrafficPoint uruted ascending berdasarkan timestamp.
func (s *redisTrafficStore) Query(ctx context.Context, oltID string, portIndex int, from, to time.Time) ([]domain.PONTrafficPoint, error) {
	key := trafficKey(oltID, portIndex)

	// ZRANGEBYSCORE dengan range [from, to] dalam unix timestamp
	results, err := s.client.ZRangeByScoreWithScores(ctx, key, &redis.ZRangeBy{
		Min: fmt.Sprintf("%d", from.Unix()),
		Max: fmt.Sprintf("%d", to.Unix()),
	}).Result()
	if err != nil {
		return nil, fmt.Errorf("gagal ZRANGEBYSCORE traffic: %w", err)
	}

	// Parsing setiap member menjadi PONTrafficPoint
	points := make([]domain.PONTrafficPoint, 0, len(results))
	for _, z := range results {
		point, err := parseTrafficMember(z)
		if err != nil {
			continue // skip member yang tidak valid
		}
		points = append(points, point)
	}

	return points, nil
}

// GetLatest mengambil data point traffic terbaru untuk PON port.
// Menggunakan ZREVRANGEBYSCORE dengan limit 1 untuk mendapatkan entry terakhir.
func (s *redisTrafficStore) GetLatest(ctx context.Context, oltID string, portIndex int) (*domain.PONTrafficPoint, error) {
	key := trafficKey(oltID, portIndex)

	// ZREVRANGEBYSCORE +inf -inf LIMIT 0 1 - ambil member dengan score tertinggi
	results, err := s.client.ZRevRangeByScoreWithScores(ctx, key, &redis.ZRangeBy{
		Min:    "-inf",
		Max:    "+inf",
		Offset: 0,
		Count:  1,
	}).Result()
	if err != nil {
		return nil, fmt.Errorf("gagal ZREVRANGEBYSCORE traffic: %w", err)
	}

	// Tidak ada data traffic
	if len(results) == 0 {
		return nil, nil
	}

	point, err := parseTrafficMember(results[0])
	if err != nil {
		return nil, fmt.Errorf("gagal parse traffic terbaru: %w", err)
	}

	return &point, nil
}

// parseTrafficMember mengekstrak PONTrafficPoint dari member uruted atur.
// Format member: "{unix_timestamp}:{json_traffic}"
func parseTrafficMember(z redis.Z) (domain.PONTrafficPoint, error) {
	memberStr, ok := z.Member.(string)
	if !ok {
		return domain.PONTrafficPoint{}, fmt.Errorf("member bukan string")
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
		return domain.PONTrafficPoint{}, fmt.Errorf("format member tidak valid")
	}

	jsonData := memberStr[idx+1:]

	var point domain.PONTrafficPoint
	if err := json.Unmarshal([]byte(jsonData), &point); err != nil {
		return domain.PONTrafficPoint{}, fmt.Errorf("gagal unmarshal traffic: %w", err)
	}

	// Gunakan score sebagai timestamp (lebih akurat daripada prefix member)
	point.Timestamp = time.Unix(int64(z.Score), 0)

	return point, nil
}
