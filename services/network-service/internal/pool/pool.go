// Package pool - Connection pool per router MikroTik.
// Mengelola koneksi TCP ke satu router dengan lazy connect, priority queue,
// rate limiting, idle timeout, max lifetime, dan health ping.
package pool

import (
	"container/heap"
	"context"
	"sync"
	"time"

	"golang.org/x/time/rate"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// Konstanta konfigurasi pool bawaan.
const (
	maxConns        = 5                // Maksimum koneksi per pool
	idleTimeout     = 5 * time.Minute  // Timeout koneksi idle
	maxLifetime     = 1 * time.Hour    // Lifetime maksimum koneksi
	healthInterval  = 30 * time.Second // Interval health ping pada idle connections
	rateLimit       = 10               // Maksimum commands per detik
	warmUpThreshold = 10               // Threshold antrian untuk warm-up
)

// AdapterFactory adalah fungsi factory untuk membuat instance RouterOSAdapter baru.
type AdapterFactory func() domain.RouterOSAdapter

// poolConn membungkus koneksi adapter dengan metadata waktu.
type poolConn struct {
	adapter   domain.RouterOSAdapter
	createdAt time.Time
	lastUsed  time.Time
}

// waiter merepresentasikan goroutine yang menunggu koneksi dari pool.
type waiter struct {
	priority domain.CommandPriority
	seq      uint64        // Nomor urut untuk FIFO dalam prioritas yang sama
	ch       chan struct{} // Channel untuk notifikasi saat koneksi tersedia
	index    int           // Index di heap (dikelola oleh container/heap)
}

// =============================================================================
// Priority Antrean - implementasi container/heap untuk waiter
// =============================================================================

// waiterHeap mengimplementasikan heap.Interface untuk priority queue waiter.
// Prioritas lebih tinggi di-dequeue duluan. FIFO untuk prioritas sama.
type waiterHeap []*waiter

func (h waiterHeap) Len() int { return len(h) }

func (h waiterHeap) Less(i, j int) bool {
	// Prioritas lebih tinggi duluan (descending)
	if h[i].priority != h[j].priority {
		return h[i].priority > h[j].priority
	}
	// FIFO untuk prioritas sama (ascending seq)
	return h[i].seq < h[j].seq
}

func (h waiterHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].index = i
	h[j].index = j
}

func (h *waiterHeap) Push(x any) {
	w := x.(*waiter)
	w.index = len(*h)
	*h = append(*h, w)
}

func (h *waiterHeap) Pop() any {
	old := *h
	n := len(old)
	w := old[n-1]
	old[n-1] = nil // hindari memory leak
	w.index = -1
	*h = old[:n-1]
	return w
}

// =============================================================================
// connPool - implementasi ConnPool interface
// =============================================================================

// connPool mengelola pool koneksi TCP ke satu router MikroTik.
// Mendukung lazy connect, priority queue, rate limiting, dan warm-up.
type connPool struct {
	mu      sync.Mutex
	cfg     domain.ConnectionConfig
	factory AdapterFactory

	// Koneksi idle yang tersedia
	idle []*poolConn

	// Jumlah koneksi aktif (sedang digunakan + idle)
	totalConns int

	// Priority queue untuk waiter yang menunggu koneksi
	waiters waiterHeap
	seq     uint64 // Counter untuk FIFO ordering

	// Rate limiter: token bucket 10 commands/detik
	limiter *rate.Limiter

	// Lifecycle
	closed   bool
	stopOnce sync.Once
	stopCh   chan struct{}

	// Map adapter -> poolConn untuk tracking saat Put
	connMap map[domain.RouterOSAdapter]*poolConn
}

// NewConnPool membuat instance ConnPool baru untuk satu router.
// factory digunakan untuk membuat adapter baru saat lazy connect.
func NewConnPool(cfg domain.ConnectionConfig, factory AdapterFactory) domain.ConnPool {
	p := &connPool{
		cfg:     cfg,
		factory: factory,
		idle:    make([]*poolConn, 0, maxConns),
		limiter: rate.NewLimiter(rate.Limit(rateLimit), rateLimit),
		stopCh:  make(chan struct{}),
		connMap: make(map[domain.RouterOSAdapter]*poolConn),
	}
	heap.Init(&p.waiters)

	// Jalankan goroutine untuk health ping dan cleanup idle connections
	go p.maintenance()

	return p
}

// Get mengambil koneksi idle atau membuat koneksi baru (lazy connect).
// Memblokir jika pool penuh sampai koneksi tersedia atau context dibatalkan.
// Priority menentukan urutan dequeue saat pool penuh.
func (p *connPool) Get(ctx context.Context, priority domain.CommandPriority) (domain.RouterOSAdapter, error) {
	// Cek rate limit terlebih dahulu
	if !p.limiter.Allow() {
		// Tunggu token tersedia atau context dibatalkan
		if err := p.limiter.Wait(ctx); err != nil {
			return nil, domain.ErrRateLimited
		}
	}

	p.mu.Lock()

	if p.closed {
		p.mu.Unlock()
		return nil, domain.ErrPoolExhausted
	}

	// Coba ambil koneksi idle yang masih valid
	conn := p.getIdleConn()
	if conn != nil {
		conn.lastUsed = time.Now()
		p.mu.Unlock()
		return conn.adapter, nil
	}

	// Jika belum mencapai max, buat koneksi baru (lazy connect)
	if p.totalConns < maxConns {
		p.totalConns++
		p.mu.Unlock()
		return p.createConn(ctx)
	}

	// Pool penuh - masuk ke priority queue dan tunggu
	w := &waiter{
		priority: priority,
		seq:      p.seq,
		ch:       make(chan struct{}, 1),
	}
	p.seq++
	heap.Push(&p.waiters, w)

	// Trigger warm-up jika antrian melebihi threshold
	needWarmUp := p.waiters.Len() > warmUpThreshold
	p.mu.Unlock()

	if needWarmUp {
		go func() { _ = p.WarmUp(context.Background()) }()
	}

	// Tunggu notifikasi atau context dibatalkan
	select {
	case <-w.ch:
		p.mu.Lock()
		conn := p.getIdleConn()
		if conn != nil {
			conn.lastUsed = time.Now()
			p.mu.Unlock()
			return conn.adapter, nil
		}
		// Koneksi sudah diambil waiter lain, buat baru jika bisa
		if p.totalConns < maxConns {
			p.totalConns++
			p.mu.Unlock()
			return p.createConn(ctx)
		}
		p.mu.Unlock()
		return nil, domain.ErrPoolExhausted

	case <-ctx.Done():
		// Hapus waiter dari queue
		p.mu.Lock()
		p.removeWaiter(w)
		p.mu.Unlock()
		return nil, domain.ErrPoolExhausted

	case <-p.stopCh:
		return nil, domain.ErrPoolExhausted
	}
}

// Put mengembalikan koneksi ke pool setelah selesai digunakan.
func (p *connPool) Put(conn domain.RouterOSAdapter) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		_ = conn.Close()
		return
	}

	pc, ok := p.connMap[conn]
	if !ok {
		// Koneksi tidak dikenal, tutup saja
		_ = conn.Close()
		return
	}

	// Cek max lifetime
	if time.Since(pc.createdAt) > maxLifetime {
		p.removeConn(pc)
		// Notifikasi waiter agar buat koneksi baru
		p.notifyWaiter()
		return
	}

	pc.lastUsed = time.Now()

	// Jika ada waiter yang menunggu, notifikasi langsung
	if p.waiters.Len() > 0 {
		p.idle = append(p.idle, pc)
		p.notifyWaiter()
		return
	}

	// Kembalikan ke idle pool
	p.idle = append(p.idle, pc)
}

// Close menutup semua koneksi di pool.
func (p *connPool) Close() error {
	p.stopOnce.Do(func() {
		p.mu.Lock()
		defer p.mu.Unlock()

		p.closed = true
		close(p.stopCh)

		// Tutup semua koneksi idle
		for _, pc := range p.idle {
			_ = pc.adapter.Close()
			delete(p.connMap, pc.adapter)
		}
		p.idle = nil

		// Notifikasi semua waiter agar tidak menunggu selamanya
		for p.waiters.Len() > 0 {
			w := heap.Pop(&p.waiters).(*waiter)
			close(w.ch)
		}
	})
	return nil
}

// Stats mengembalikan statistik pool (active, idle, total).
func (p *connPool) Stats() domain.PoolStats {
	p.mu.Lock()
	defer p.mu.Unlock()

	idleCount := len(p.idle)
	return domain.PoolStats{
		Active: p.totalConns - idleCount,
		Idle:   idleCount,
		Total:  p.totalConns,
	}
}

// WarmUp membuka koneksi hingga max capacity secara paralel.
// Dipanggil saat antrian perintah melebihi warm-up threshold.
func (p *connPool) WarmUp(ctx context.Context) error {
	p.mu.Lock()
	need := maxConns - p.totalConns
	if need <= 0 || p.closed {
		p.mu.Unlock()
		return nil
	}
	// Reserve slot untuk koneksi yang akan dibuat
	p.totalConns += need
	p.mu.Unlock()

	var wg sync.WaitGroup
	var mu sync.Mutex
	var firstErr error

	for i := 0; i < need; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			adapter := p.factory()
			if err := adapter.Connect(ctx, p.cfg); err != nil {
				_ = adapter.Close()
				mu.Lock()
				if firstErr == nil {
					firstErr = err
				}
				mu.Unlock()
				// Kembalikan slot yang gagal
				p.mu.Lock()
				p.totalConns--
				p.mu.Unlock()
				return
			}

			pc := &poolConn{
				adapter:   adapter,
				createdAt: time.Now(),
				lastUsed:  time.Now(),
			}

			p.mu.Lock()
			p.connMap[adapter] = pc
			p.idle = append(p.idle, pc)
			// Notifikasi waiter jika ada
			p.notifyWaiter()
			p.mu.Unlock()
		}()
	}

	wg.Wait()
	return firstErr
}

// =============================================================================
// Fungsi bantu methods (internal)
// =============================================================================

// getIdleConn mengambil koneksi idle yang masih valid (belum expired).
// Harus dipanggil dengan p.mu sudah di-lock.
func (p *connPool) getIdleConn() *poolConn {
	for len(p.idle) > 0 {
		// Ambil dari belakang (LIFO - koneksi terbaru lebih mungkin sehat)
		pc := p.idle[len(p.idle)-1]
		p.idle = p.idle[:len(p.idle)-1]

		// Cek idle timeout
		if time.Since(pc.lastUsed) > idleTimeout {
			p.removeConn(pc)
			continue
		}

		// Cek max lifetime
		if time.Since(pc.createdAt) > maxLifetime {
			p.removeConn(pc)
			continue
		}

		return pc
	}
	return nil
}

// createConn membuat koneksi baru ke router.
// totalConns sudah di-increment sebelum pemanggilan.
func (p *connPool) createConn(ctx context.Context) (domain.RouterOSAdapter, error) {
	adapter := p.factory()
	if err := adapter.Connect(ctx, p.cfg); err != nil {
		_ = adapter.Close()
		p.mu.Lock()
		p.totalConns--
		p.mu.Unlock()
		return nil, domain.ErrConnectionFailed
	}

	pc := &poolConn{
		adapter:   adapter,
		createdAt: time.Now(),
		lastUsed:  time.Now(),
	}

	p.mu.Lock()
	p.connMap[adapter] = pc
	p.mu.Unlock()

	return adapter, nil
}

// removeConn menutup koneksi dan mengurangi counter.
// Harus dipanggil dengan p.mu sudah di-lock.
func (p *connPool) removeConn(pc *poolConn) {
	_ = pc.adapter.Close()
	delete(p.connMap, pc.adapter)
	p.totalConns--
}

// notifyWaiter mengirim notifikasi ke waiter dengan prioritas tertinggi.
// Harus dipanggil dengan p.mu sudah di-lock.
func (p *connPool) notifyWaiter() {
	if p.waiters.Len() > 0 {
		w := heap.Pop(&p.waiters).(*waiter)
		w.ch <- struct{}{}
	}
}

// removeWaiter menghapus waiter dari priority queue.
// Harus dipanggil dengan p.mu sudah di-lock.
func (p *connPool) removeWaiter(w *waiter) {
	if w.index >= 0 && w.index < p.waiters.Len() {
		heap.Remove(&p.waiters, w.index)
	}
}

// maintenance menjalankan goroutine periodik untuk:
// 1. Health ping pada idle connections setiap 30 detik
// 2. Pembersihan koneksi yang melebihi idle timeout atau max lifetime
func (p *connPool) maintenance() {
	ticker := time.NewTicker(healthInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			p.cleanupAndPing()
		case <-p.stopCh:
			return
		}
	}
}

// cleanupAndPing melakukan health ping dan cleanup pada idle connections.
func (p *connPool) cleanupAndPing() {
	p.mu.Lock()
	if p.closed || len(p.idle) == 0 {
		p.mu.Unlock()
		return
	}

	// Salin idle connections untuk di-ping di luar lock
	toCheck := make([]*poolConn, len(p.idle))
	copy(toCheck, p.idle)
	p.mu.Unlock()

	// Ping setiap koneksi idle
	var toRemove []*poolConn
	for _, pc := range toCheck {
		// Cek idle timeout dan max lifetime
		if time.Since(pc.lastUsed) > idleTimeout || time.Since(pc.createdAt) > maxLifetime {
			toRemove = append(toRemove, pc)
			continue
		}

		// Health ping dengan timeout singkat
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		if err := pc.adapter.Ping(ctx); err != nil {
			toRemove = append(toRemove, pc)
		}
		cancel()
	}

	// Hapus koneksi yang gagal atau expired
	if len(toRemove) > 0 {
		p.mu.Lock()
		for _, pc := range toRemove {
			// Pastikan masih ada di idle list
			for i, idle := range p.idle {
				if idle == pc {
					p.idle = append(p.idle[:i], p.idle[i+1:]...)
					p.removeConn(pc)
					break
				}
			}
		}
		p.mu.Unlock()
	}
}
