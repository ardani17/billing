package metrics

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/ispboss/ispboss/services/network-service/internal/domain"
	"github.com/redis/go-redis/v9"
)

// setupTrafficTest membuat miniredis dan TrafficStore untuk testing.
func setupTrafficTest(t *testing.T) (*miniredis.Miniredis, domain.TrafficStore, *redis.Client) {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("gagal memulai miniredis: %v", err)
	}
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	store := NewRedisTrafficStore(client)
	return mr, store, client
}

// TestTrafficStore_Store memverifikasi penyimpanan data point traffic ke Redis.
func TestTrafficStore_Store(t *testing.T) {
	mr, store, client := setupTrafficTest(t)
	defer mr.Close()
	defer client.Close()

	ctx := context.Background()
	now := time.Now().Truncate(time.Second)

	traffic := domain.PONTrafficPoint{
		Timestamp: now,
		RxBytes:   1024000,
		RxPackets: 500,
		TxBytes:   2048000,
		TxPackets: 1000,
	}

	// Simpan traffic
	if err := store.Store(ctx, "olt-1", 0, traffic); err != nil {
		t.Fatalf("Store gagal: %v", err)
	}

	// Verifikasi data tersimpan di sorted set
	key := trafficKey("olt-1", 0)
	count, err := client.ZCard(ctx, key).Result()
	if err != nil {
		t.Fatalf("ZCard gagal: %v", err)
	}
	if count != 1 {
		t.Errorf("jumlah member=%d, diharapkan=1", count)
	}
}

// TestTrafficStore_Query memverifikasi query data point traffic berdasarkan rentang waktu.
func TestTrafficStore_Query(t *testing.T) {
	mr, store, client := setupTrafficTest(t)
	defer mr.Close()
	defer client.Close()

	ctx := context.Background()
	base := time.Now().Add(-1 * time.Hour).Truncate(time.Second)

	// Simpan 3 data point dengan timestamp berbeda
	points := []domain.PONTrafficPoint{
		{Timestamp: base, RxBytes: 100, RxPackets: 10, TxBytes: 200, TxPackets: 20},
		{Timestamp: base.Add(10 * time.Minute), RxBytes: 300, RxPackets: 30, TxBytes: 400, TxPackets: 40},
		{Timestamp: base.Add(30 * time.Minute), RxBytes: 500, RxPackets: 50, TxBytes: 600, TxPackets: 60},
	}
	for _, p := range points {
		if err := store.Store(ctx, "olt-2", 1, p); err != nil {
			t.Fatalf("Store gagal: %v", err)
		}
	}

	// Query rentang yang mencakup 2 data point pertama
	from := base
	to := base.Add(15 * time.Minute)
	results, err := store.Query(ctx, "olt-2", 1, from, to)
	if err != nil {
		t.Fatalf("Query gagal: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("jumlah hasil=%d, diharapkan=2", len(results))
	}

	// Verifikasi urutan ascending
	if results[0].Timestamp.After(results[1].Timestamp) {
		t.Error("hasil tidak sorted ascending berdasarkan timestamp")
	}

	// Verifikasi nilai traffic
	if results[0].RxBytes != 100 {
		t.Errorf("results[0].RxBytes=%d, diharapkan=100", results[0].RxBytes)
	}
	if results[1].TxBytes != 400 {
		t.Errorf("results[1].TxBytes=%d, diharapkan=400", results[1].TxBytes)
	}
}

// TestTrafficStore_GetLatest memverifikasi pengambilan data point traffic terbaru.
func TestTrafficStore_GetLatest(t *testing.T) {
	mr, store, client := setupTrafficTest(t)
	defer mr.Close()
	defer client.Close()

	ctx := context.Background()
	base := time.Now().Add(-1 * time.Hour).Truncate(time.Second)

	// Simpan 3 data point
	points := []domain.PONTrafficPoint{
		{Timestamp: base, RxBytes: 100, RxPackets: 10, TxBytes: 200, TxPackets: 20},
		{Timestamp: base.Add(10 * time.Minute), RxBytes: 300, RxPackets: 30, TxBytes: 400, TxPackets: 40},
		{Timestamp: base.Add(20 * time.Minute), RxBytes: 500, RxPackets: 50, TxBytes: 600, TxPackets: 60},
	}
	for _, p := range points {
		if err := store.Store(ctx, "olt-3", 2, p); err != nil {
			t.Fatalf("Store gagal: %v", err)
		}
	}

	// GetLatest harus mengembalikan data point terakhir
	latest, err := store.GetLatest(ctx, "olt-3", 2)
	if err != nil {
		t.Fatalf("GetLatest gagal: %v", err)
	}
	if latest == nil {
		t.Fatal("GetLatest mengembalikan nil, diharapkan data point")
	}

	if latest.RxBytes != 500 {
		t.Errorf("latest.RxBytes=%d, diharapkan=500", latest.RxBytes)
	}
	if latest.TxPackets != 60 {
		t.Errorf("latest.TxPackets=%d, diharapkan=60", latest.TxPackets)
	}
}

// TestTrafficStore_GetLatest_Empty memverifikasi GetLatest mengembalikan nil jika tidak ada data.
func TestTrafficStore_GetLatest_Empty(t *testing.T) {
	mr, store, client := setupTrafficTest(t)
	defer mr.Close()
	defer client.Close()

	ctx := context.Background()

	latest, err := store.GetLatest(ctx, "olt-nonexistent", 0)
	if err != nil {
		t.Fatalf("GetLatest gagal: %v", err)
	}
	if latest != nil {
		t.Errorf("GetLatest mengembalikan data, diharapkan nil")
	}
}

// TestTrafficStore_Query_Empty memverifikasi Query mengembalikan slice kosong jika tidak ada data.
func TestTrafficStore_Query_Empty(t *testing.T) {
	mr, store, client := setupTrafficTest(t)
	defer mr.Close()
	defer client.Close()

	ctx := context.Background()
	from := time.Now().Add(-1 * time.Hour)
	to := time.Now()

	results, err := store.Query(ctx, "olt-nonexistent", 0, from, to)
	if err != nil {
		t.Fatalf("Query gagal: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("jumlah hasil=%d, diharapkan=0", len(results))
	}
}

// TestTrafficStore_TTL memverifikasi bahwa TTL 7 hari di-set pada key setelah Store.
func TestTrafficStore_TTL(t *testing.T) {
	mr, store, client := setupTrafficTest(t)
	defer mr.Close()
	defer client.Close()

	ctx := context.Background()
	now := time.Now().Truncate(time.Second)

	traffic := domain.PONTrafficPoint{
		Timestamp: now,
		RxBytes:   1024,
		RxPackets: 10,
		TxBytes:   2048,
		TxPackets: 20,
	}

	if err := store.Store(ctx, "olt-ttl", 0, traffic); err != nil {
		t.Fatalf("Store gagal: %v", err)
	}

	// Verifikasi TTL di-set pada key
	key := trafficKey("olt-ttl", 0)
	ttl := mr.TTL(key)
	if ttl == 0 {
		t.Fatal("TTL tidak di-set pada key")
	}

	// TTL harus sekitar 7 hari (toleransi 1 menit)
	expected := 7 * 24 * time.Hour
	diff := ttl - expected
	if diff < 0 {
		diff = -diff
	}
	if diff > time.Minute {
		t.Errorf("TTL=%v, diharapkan sekitar %v", ttl, expected)
	}
}
