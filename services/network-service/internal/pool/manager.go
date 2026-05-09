// Package pool - Pool manager untuk mengelola connection pool per router.
// Menyediakan akses thread-safe ke pool koneksi menggunakan sync.RWMutex.
package pool

import (
	"sync"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// poolManager mengelola pool koneksi untuk semua router.
// Thread-safe: menggunakan sync.RWMutex untuk akses concurrent ke map pools.
type poolManager struct {
	mu      sync.RWMutex
	pools   map[string]domain.ConnPool
	factory AdapterFactory
}

// NewPoolManager membuat instance PoolManager baru.
// factory digunakan untuk membuat adapter baru saat pool membuat koneksi.
func NewPoolManager(factory AdapterFactory) domain.PoolManager {
	return &poolManager{
		pools:   make(map[string]domain.ConnPool),
		factory: factory,
	}
}

// GetPool mengembalikan pool untuk router tertentu.
// Jika pool belum ada, buat baru secara thread-safe.
func (m *poolManager) GetPool(routerID string, cfg domain.ConnectionConfig) domain.ConnPool {
	// Coba baca dulu dengan read lock (fast path)
	m.mu.RLock()
	p, ok := m.pools[routerID]
	m.mu.RUnlock()

	if ok {
		return p
	}

	// Pool belum ada - ambil write lock untuk membuat baru
	m.mu.Lock()
	defer m.mu.Unlock()

	// Double-cek setelah upgrade lock (goroutine lain mungkin sudah buat)
	if p, ok = m.pools[routerID]; ok {
		return p
	}

	p = NewConnPool(cfg, m.factory)
	m.pools[routerID] = p
	return p
}

// ClosePool menutup pool untuk router tertentu dan hapus dari map.
func (m *poolManager) ClosePool(routerID string) {
	m.mu.Lock()
	p, ok := m.pools[routerID]
	if ok {
		delete(m.pools, routerID)
	}
	m.mu.Unlock()

	// Tutup pool di luar lock untuk menghindari deadlock
	if ok {
		_ = p.Close()
	}
}

// CloseAll menutup semua pool dan mengosongkan map.
func (m *poolManager) CloseAll() {
	m.mu.Lock()
	pools := m.pools
	m.pools = make(map[string]domain.ConnPool)
	m.mu.Unlock()

	// Tutup semua pool di luar lock untuk menghindari deadlock
	for _, p := range pools {
		_ = p.Close()
	}
}
