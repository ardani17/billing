package pool

import (
	"container/heap"
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"pgregory.net/rapid"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// testAdapter adalah mock adapter sederhana untuk testing pool.
type testAdapter struct {
	mu        sync.Mutex
	connected bool
}

func (a *testAdapter) Connect(_ context.Context, _ domain.ConnectionConfig) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.connected = true
	return nil
}

func (a *testAdapter) Close() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.connected = false
	return nil
}

func (a *testAdapter) Execute(_ context.Context, _ string, _ map[string]string) ([]map[string]string, error) {
	return nil, nil
}

func (a *testAdapter) GetSystemResource(_ context.Context) (*domain.SystemResource, error) {
	return &domain.SystemResource{Version: "6.49.10"}, nil
}

func (a *testAdapter) Ping(_ context.Context) error {
	return nil
}

// defaultTestCfg mengembalikan ConnectionConfig default untuk testing.
func defaultTestCfg() domain.ConnectionConfig {
	return domain.ConnectionConfig{
		Host:           "127.0.0.1",
		Port:           8728,
		Username:       "admin",
		Password:       "test",
		ConnectTimeout: 5 * time.Second,
		CommandTimeout: 5 * time.Second,
	}
}

// =============================================================================
// Feature: mikrotik-router, Property 2: Pool capacity invariant
// =============================================================================

// TestProperty_PoolCapacityInvariant memverifikasi bahwa pool TIDAK PERNAH
// memiliki lebih dari 5 koneksi aktif secara bersamaan.
//
// **Validates: Requirements 4.1, 4.6**
func TestProperty_PoolCapacityInvariant(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Range kecil (1-8) agar test cepat
		n := rapid.IntRange(1, 8).Draw(t, "concurrentRequests")

		var activeConns atomic.Int32
		var maxObserved atomic.Int32

		factory := func() domain.RouterOSAdapter {
			return &testAdapter{}
		}

		p := NewConnPool(defaultTestCfg(), factory)
		defer p.Close()

		var wg sync.WaitGroup
		wg.Add(n)

		for i := 0; i < n; i++ {
			go func() {
				defer wg.Done()

				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()

				conn, err := p.Get(ctx, domain.PriorityMedium)
				if err != nil {
					return
				}

				current := activeConns.Add(1)
				for {
					old := maxObserved.Load()
					if current <= old || maxObserved.CompareAndSwap(old, current) {
						break
					}
				}

				// Hold singkat tanpa sleep — cukup yield
				activeConns.Add(-1)
				p.Put(conn)
			}()
		}

		wg.Wait()

		observed := maxObserved.Load()
		if observed > int32(maxConns) {
			t.Errorf(
				"Pool capacity invariant dilanggar: max=%d, batas=%d (N=%d)",
				observed, maxConns, n,
			)
		}
	})
}

// =============================================================================
// Feature: mikrotik-router, Property 3: Rate limiting enforcement
// =============================================================================

// TestProperty_RateLimitingEnforcement memverifikasi bahwa rate eksekusi
// TIDAK melebihi 10 commands per detik. Menggunakan satu skenario deterministik
// dengan 12 commands (2 di atas burst) untuk memastikan rate limiter bekerja.
//
// **Validates: Requirements 4.8**
func TestProperty_RateLimitingEnforcement(t *testing.T) {
	// 12 commands: 10 burst langsung, 2 harus menunggu token refill
	n := 12

	factory := func() domain.RouterOSAdapter {
		return &testAdapter{}
	}

	p := NewConnPool(defaultTestCfg(), factory)
	defer p.Close()

	var mu sync.Mutex
	timestamps := make([]time.Time, 0, n)

	var wg sync.WaitGroup
	wg.Add(n)

	start := time.Now()

	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			conn, err := p.Get(ctx, domain.PriorityMedium)
			if err != nil {
				return
			}

			mu.Lock()
			timestamps = append(timestamps, time.Now())
			mu.Unlock()

			p.Put(conn)
		}()
	}

	wg.Wait()
	totalDuration := time.Since(start)

	mu.Lock()
	completedCount := len(timestamps)
	mu.Unlock()

	if completedCount < n {
		t.Errorf("hanya %d dari %d commands selesai", completedCount, n)
	}

	// 2 commands di atas burst harus menunggu ~200ms total (2/10 detik).
	// Dengan toleransi 50%, minimal 100ms.
	minExpected := 100 * time.Millisecond
	if totalDuration < minExpected {
		t.Errorf(
			"Rate limit mungkin dilanggar: %d commands dalam %v, minimal %v",
			completedCount, totalDuration, minExpected,
		)
	}
}

// =============================================================================
// Feature: mikrotik-router, Property 14: Priority queue ordering
// =============================================================================

// TestProperty_PriorityQueueOrdering memverifikasi bahwa waiterHeap selalu
// mengeluarkan elemen dengan prioritas tertinggi duluan, dan FIFO dalam
// prioritas yang sama.
//
// **Validates: Requirements 4.10**
func TestProperty_PriorityQueueOrdering(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		numHigh := rapid.IntRange(1, 5).Draw(t, "numHigh")
		numMedium := rapid.IntRange(1, 5).Draw(t, "numMedium")
		numLow := rapid.IntRange(1, 5).Draw(t, "numLow")
		total := numHigh + numMedium + numLow

		// Verifikasi heap ordering: push campuran, pop harus terurut
		var h waiterHeap
		heap.Init(&h)
		var seq uint64

		for i := 0; i < numLow; i++ {
			heap.Push(&h, &waiter{priority: domain.PriorityLow, seq: seq})
			seq++
		}
		for i := 0; i < numMedium; i++ {
			heap.Push(&h, &waiter{priority: domain.PriorityMedium, seq: seq})
			seq++
		}
		for i := 0; i < numHigh; i++ {
			heap.Push(&h, &waiter{priority: domain.PriorityHigh, seq: seq})
			seq++
		}

		if h.Len() != total {
			t.Fatalf("heap size %d != total %d", h.Len(), total)
		}

		popOrder := make([]domain.CommandPriority, 0, total)
		for h.Len() > 0 {
			w := heap.Pop(&h).(*waiter)
			popOrder = append(popOrder, w.priority)
		}

		// Verifikasi: High duluan, lalu Medium, lalu Low
		idx := 0
		for i := 0; i < numHigh; i++ {
			if popOrder[idx] != domain.PriorityHigh {
				t.Errorf("posisi %d: dapat %d, harapkan High(%d)", idx, popOrder[idx], domain.PriorityHigh)
			}
			idx++
		}
		for i := 0; i < numMedium; i++ {
			if popOrder[idx] != domain.PriorityMedium {
				t.Errorf("posisi %d: dapat %d, harapkan Medium(%d)", idx, popOrder[idx], domain.PriorityMedium)
			}
			idx++
		}
		for i := 0; i < numLow; i++ {
			if popOrder[idx] != domain.PriorityLow {
				t.Errorf("posisi %d: dapat %d, harapkan Low(%d)", idx, popOrder[idx], domain.PriorityLow)
			}
			idx++
		}

		// Verifikasi FIFO dalam prioritas yang sama
		var h2 waiterHeap
		heap.Init(&h2)
		numSame := rapid.IntRange(2, 8).Draw(t, "numSamePriority")
		prio := rapid.SampledFrom([]domain.CommandPriority{
			domain.PriorityHigh, domain.PriorityMedium, domain.PriorityLow,
		}).Draw(t, "samePriority")

		for i := 0; i < numSame; i++ {
			heap.Push(&h2, &waiter{priority: prio, seq: uint64(i)})
		}

		var prevSeq uint64
		for i := 0; h2.Len() > 0; i++ {
			w := heap.Pop(&h2).(*waiter)
			if i > 0 && w.seq <= prevSeq {
				t.Errorf("FIFO dilanggar: seq %d setelah seq %d", w.seq, prevSeq)
			}
			prevSeq = w.seq
		}
	})
}
