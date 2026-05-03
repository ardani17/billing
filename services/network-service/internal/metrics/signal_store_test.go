package metrics

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/ispboss/ispboss/services/network-service/internal/domain"
	"github.com/redis/go-redis/v9"
)

// setupSignalTest membuat miniredis dan SignalStore untuk testing.
func setupSignalTest(t *testing.T) (*miniredis.Miniredis, domain.SignalStore, *redis.Client) {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("gagal memulai miniredis: %v", err)
	}
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	store := NewRedisSignalStore(client)
	return mr, store, client
}

// TestSignalStore_Store memverifikasi penyimpanan data point signal ke Redis.
func TestSignalStore_Store(t *testing.T) {
	mr, store, client := setupSignalTest(t)
	defer mr.Close()
	defer client.Close()

	ctx := context.Background()
	now := time.Now().Truncate(time.Second)

	signal := domain.ONTSignalPoint{
		Timestamp:   now,
		RxPowerDBm:  -18.5,
		SignalLevel: domain.SignalNormal,
	}

	// Simpan signal
	if err := store.Store(ctx, "olt-1", 0, 1, signal); err != nil {
		t.Fatalf("Store gagal: %v", err)
	}

	// Verifikasi data tersimpan di sorted set
	key := signalKey("olt-1", 0, 1)
	count, err := client.ZCard(ctx, key).Result()
	if err != nil {
		t.Fatalf("ZCard gagal: %v", err)
	}
	if count != 1 {
		t.Errorf("jumlah member=%d, diharapkan=1", count)
	}
}

// TestSignalStore_Query memverifikasi query data point signal berdasarkan rentang waktu.
func TestSignalStore_Query(t *testing.T) {
	mr, store, client := setupSignalTest(t)
	defer mr.Close()
	defer client.Close()

	ctx := context.Background()
	base := time.Now().Add(-1 * time.Hour).Truncate(time.Second)

	// Simpan 3 data point dengan timestamp berbeda
	signals := []domain.ONTSignalPoint{
		{Timestamp: base, RxPowerDBm: -18.0, SignalLevel: domain.SignalNormal},
		{Timestamp: base.Add(10 * time.Minute), RxPowerDBm: -22.0, SignalLevel: domain.SignalNormal},
		{Timestamp: base.Add(30 * time.Minute), RxPowerDBm: -26.0, SignalLevel: domain.SignalWarning},
	}
	for _, s := range signals {
		if err := store.Store(ctx, "olt-2", 1, 3, s); err != nil {
			t.Fatalf("Store gagal: %v", err)
		}
	}

	// Query rentang yang mencakup 2 data point pertama
	from := base
	to := base.Add(15 * time.Minute)
	results, err := store.Query(ctx, "olt-2", 1, 3, from, to)
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

	// Verifikasi nilai RxPowerDBm
	if results[0].RxPowerDBm != -18.0 {
		t.Errorf("results[0].RxPowerDBm=%.1f, diharapkan=-18.0", results[0].RxPowerDBm)
	}
	if results[1].RxPowerDBm != -22.0 {
		t.Errorf("results[1].RxPowerDBm=%.1f, diharapkan=-22.0", results[1].RxPowerDBm)
	}
}

// TestSignalStore_GetLatest memverifikasi pengambilan data point signal terbaru.
func TestSignalStore_GetLatest(t *testing.T) {
	mr, store, client := setupSignalTest(t)
	defer mr.Close()
	defer client.Close()

	ctx := context.Background()
	base := time.Now().Add(-1 * time.Hour).Truncate(time.Second)

	// Simpan 3 data point
	signals := []domain.ONTSignalPoint{
		{Timestamp: base, RxPowerDBm: -18.0, SignalLevel: domain.SignalNormal},
		{Timestamp: base.Add(10 * time.Minute), RxPowerDBm: -22.0, SignalLevel: domain.SignalNormal},
		{Timestamp: base.Add(20 * time.Minute), RxPowerDBm: -28.0, SignalLevel: domain.SignalWeak},
	}
	for _, s := range signals {
		if err := store.Store(ctx, "olt-3", 2, 5, s); err != nil {
			t.Fatalf("Store gagal: %v", err)
		}
	}

	// GetLatest harus mengembalikan data point terakhir
	latest, err := store.GetLatest(ctx, "olt-3", 2, 5)
	if err != nil {
		t.Fatalf("GetLatest gagal: %v", err)
	}
	if latest == nil {
		t.Fatal("GetLatest mengembalikan nil, diharapkan data point")
	}

	if latest.RxPowerDBm != -28.0 {
		t.Errorf("latest.RxPowerDBm=%.1f, diharapkan=-28.0", latest.RxPowerDBm)
	}
	if latest.SignalLevel != domain.SignalWeak {
		t.Errorf("latest.SignalLevel=%s, diharapkan=%s", latest.SignalLevel, domain.SignalWeak)
	}
}

// TestSignalStore_GetLatest_Empty memverifikasi GetLatest mengembalikan nil jika tidak ada data.
func TestSignalStore_GetLatest_Empty(t *testing.T) {
	mr, store, client := setupSignalTest(t)
	defer mr.Close()
	defer client.Close()

	ctx := context.Background()

	latest, err := store.GetLatest(ctx, "olt-nonexistent", 0, 0)
	if err != nil {
		t.Fatalf("GetLatest gagal: %v", err)
	}
	if latest != nil {
		t.Errorf("GetLatest mengembalikan data, diharapkan nil")
	}
}

// TestSignalStore_Query_Empty memverifikasi Query mengembalikan slice kosong jika tidak ada data.
func TestSignalStore_Query_Empty(t *testing.T) {
	mr, store, client := setupSignalTest(t)
	defer mr.Close()
	defer client.Close()

	ctx := context.Background()
	from := time.Now().Add(-1 * time.Hour)
	to := time.Now()

	results, err := store.Query(ctx, "olt-nonexistent", 0, 0, from, to)
	if err != nil {
		t.Fatalf("Query gagal: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("jumlah hasil=%d, diharapkan=0", len(results))
	}
}

// TestSignalStore_TTL memverifikasi bahwa TTL 30 hari di-set pada key setelah Store.
func TestSignalStore_TTL(t *testing.T) {
	mr, store, client := setupSignalTest(t)
	defer mr.Close()
	defer client.Close()

	ctx := context.Background()
	now := time.Now().Truncate(time.Second)

	signal := domain.ONTSignalPoint{
		Timestamp:   now,
		RxPowerDBm:  -20.0,
		SignalLevel: domain.SignalNormal,
	}

	if err := store.Store(ctx, "olt-ttl", 0, 1, signal); err != nil {
		t.Fatalf("Store gagal: %v", err)
	}

	// Verifikasi TTL di-set pada key
	key := signalKey("olt-ttl", 0, 1)
	ttl := mr.TTL(key)
	if ttl == 0 {
		t.Fatal("TTL tidak di-set pada key")
	}

	// TTL harus sekitar 30 hari (toleransi 1 menit)
	expected := 30 * 24 * time.Hour
	diff := ttl - expected
	if diff < 0 {
		diff = -diff
	}
	if diff > time.Minute {
		t.Errorf("TTL=%v, diharapkan sekitar %v", ttl, expected)
	}
}
