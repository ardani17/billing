package metrics

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/ispboss/ispboss/services/network-service/internal/domain"
	"github.com/redis/go-redis/v9"
	"pgregory.net/rapid"
)

// =============================================================================
// =============================================================================

// metricsGen menghasilkan RouterMetrics acak dengan nilai realistis.
func metricsGen(t *rapid.T, label string) domain.RouterMetrics {
	return domain.RouterMetrics{
		CPULoad:         rapid.IntRange(0, 100).Draw(t, label+"_cpu"),
		RAMUsagePercent: rapid.IntRange(0, 100).Draw(t, label+"_ram"),
		UptimeSeconds:   int64(rapid.IntRange(0, 31536000).Draw(t, label+"_uptime")),
		ActiveSessions:  rapid.IntRange(0, 5000).Draw(t, label+"_sessions"),
	}
}

// storeWithTimestamp menyimpan metrik ke Redis sorted atur dengan timestamp yang dikontrol.
// Menggunakan format member yang sama dengan redisMetricsStore.Store:
// "{unix_timestamp}:{json_metrics}"
func storeWithTimestamp(ctx context.Context, client *redis.Client, routerID string, ts time.Time, m domain.RouterMetrics) error {
	key := metricsKey(routerID)

	// Encode metrik ke JSON - sama seperti store.go
	data, err := json.Marshal(m)
	if err != nil {
		return fmt.Errorf("gagal marshal metrik: %w", err)
	}

	// Format member: "{unix_timestamp}:{json_metrics}" - konsisten dengan Store()
	member := fmt.Sprintf("%d:%s", ts.Unix(), string(data))

	return client.ZAdd(ctx, key, redis.Z{
		Score:  float64(ts.Unix()),
		Member: member,
	}).Err()
}

// TestProperty_MetricsStoreRoundTripWithOrdering memverifikasi bahwa untuk
// sembarang atur data point RouterMetrics yang disimpan untuk sebuah router,
// query dengan rentang waktu [from, to] hanya mengembalikan data point
// dengan timestamp dalam rentang tersebut, sorted ascending berdasarkan
// timestamp. Setiap data point yang dikembalikan memiliki field values
// yang sama dengan data point yang disimpan.
//
// **Memvalidasi: Kebutuhan 9.3, 9.4**
func TestProperty_MetricsStoreRoundTripWithOrdering(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Siapkan miniredis sebagai in-memory Redis
		mr, err := miniredis.Run()
		if err != nil {
			t.Fatalf("gagal memulai miniredis: %v", err)
		}
		defer mr.Close()

		redisClient := redis.NewClient(&redis.Options{
			Addr: mr.Addr(),
		})
		defer redisClient.Close()

		store := NewRedisMetricsStore(redisClient)
		ctx := context.Background()

		// Buat router ID acak (format UUID)
		routerID := rapid.StringMatching(
			`[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}`,
		).Draw(t, "routerID")

		// Buat jumlah data point acak (1-20)
		numPoints := rapid.IntRange(1, 20).Draw(t, "numPoints")

		// Base time: 1 jam yang lalu, agar semua timestamp dalam 7 hari terakhir
		baseTime := time.Now().Add(-1 * time.Hour).Truncate(time.Second)

		// Simpan data point yang di-buat
		type storedPoint struct {
			ts      time.Time
			metrics domain.RouterMetrics
		}
		points := make([]storedPoint, numPoints)

		// Buat offset unik untuk setiap data point (dalam detik)
		usedOffsets := make(map[int]bool)
		for i := 0; i < numPoints; i++ {
			var offset int
			for {
				offset = rapid.IntRange(0, 3600).Draw(t, fmt.Sprintf("offset_%d", i))
				if !usedOffsets[offset] {
					usedOffsets[offset] = true
					break
				}
			}

			ts := baseTime.Add(time.Duration(offset) * time.Second)
			m := metricsGen(t, fmt.Sprintf("m_%d", i))
			points[i] = storedPoint{ts: ts, metrics: m}

			// Simpan ke Redis dengan timestamp yang dikontrol
			if err := storeWithTimestamp(ctx, redisClient, routerID, ts, m); err != nil {
				t.Fatalf("gagal menyimpan metrik[%d]: %v", i, err)
			}
		}

		// Tentukan query range: pilih from dan to dari rentang offset
		fromOffset := rapid.IntRange(0, 1800).Draw(t, "fromOffset")
		toOffset := rapid.IntRange(fromOffset, 3600).Draw(t, "toOffset")
		from := baseTime.Add(time.Duration(fromOffset) * time.Second)
		to := baseTime.Add(time.Duration(toOffset) * time.Second)

		// Query data point dalam rentang [from, to]
		results, err := store.Query(ctx, routerID, from, to)
		if err != nil {
			t.Fatalf("gagal query metrik: %v", err)
		}

		// Hitung data point yang seharusnya ada dalam rentang [from, to]
		var expected []storedPoint
		for _, p := range points {
			if p.ts.Unix() >= from.Unix() && p.ts.Unix() <= to.Unix() {
				expected = append(expected, p)
			}
		}

		sort.Slice(expected, func(i, j int) bool {
			return expected[i].ts.Unix() < expected[j].ts.Unix()
		})

		// Verifikasi jumlah hasil sama dengan yang diharapkan
		if len(results) != len(expected) {
			t.Fatalf(
				"jumlah hasil query=%d, diharapkan=%d (range [%d, %d])",
				len(results), len(expected), from.Unix(), to.Unix(),
			)
		}

		// Verifikasi urutan ascending dan field values cocok
		for i, result := range results {
			exp := expected[i]

			// Verifikasi timestamp dalam rentang [from, to]
			if result.Timestamp.Unix() < from.Unix() || result.Timestamp.Unix() > to.Unix() {
				t.Errorf(
					"hasil[%d] timestamp %d di luar rentang [%d, %d]",
					i, result.Timestamp.Unix(), from.Unix(), to.Unix(),
				)
			}

			// Verifikasi sorted ascending
			if i > 0 && result.Timestamp.Before(results[i-1].Timestamp) {
				t.Errorf(
					"hasil tidak sorted ascending: [%d]=%v > [%d]=%v",
					i-1, results[i-1].Timestamp, i, result.Timestamp,
				)
			}

			// Verifikasi field values cocok dengan data yang disimpan
			if result.Metrics.CPULoad != exp.metrics.CPULoad {
				t.Errorf("hasil[%d] CPULoad=%d, diharapkan=%d",
					i, result.Metrics.CPULoad, exp.metrics.CPULoad)
			}
			if result.Metrics.RAMUsagePercent != exp.metrics.RAMUsagePercent {
				t.Errorf("hasil[%d] RAMUsagePercent=%d, diharapkan=%d",
					i, result.Metrics.RAMUsagePercent, exp.metrics.RAMUsagePercent)
			}
			if result.Metrics.UptimeSeconds != exp.metrics.UptimeSeconds {
				t.Errorf("hasil[%d] UptimeSeconds=%d, diharapkan=%d",
					i, result.Metrics.UptimeSeconds, exp.metrics.UptimeSeconds)
			}
			if result.Metrics.ActiveSessions != exp.metrics.ActiveSessions {
				t.Errorf("hasil[%d] ActiveSessions=%d, diharapkan=%d",
					i, result.Metrics.ActiveSessions, exp.metrics.ActiveSessions)
			}
		}
	})
}
