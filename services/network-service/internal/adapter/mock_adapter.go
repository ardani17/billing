// Package adapter — MockAdapter untuk development dan testing tanpa router fisik.
// MockAdapter mengimplementasikan RouterOSAdapter dengan response simulasi.
package adapter

import (
	"context"
	"fmt"
	"strconv"
	"sync"
)

// MockAdapter mengimplementasikan RouterOSAdapter tanpa koneksi ke router fisik.
// Digunakan saat NETWORK_MODE=mock untuk development dan testing.
type MockAdapter struct {
	mu        sync.Mutex
	connected bool
}

// NewMockAdapter membuat instance MockAdapter baru.
func NewMockAdapter() *MockAdapter {
	return &MockAdapter{}
}

// Connect mensimulasikan koneksi ke router (selalu sukses, no-op).
func (m *MockAdapter) Connect(_ context.Context, _ ConnectionConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.connected = true
	return nil
}

// Close mensimulasikan penutupan koneksi (no-op).
func (m *MockAdapter) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.connected = false
	return nil
}

// Execute menjalankan perintah RouterOS dan mengembalikan response simulasi.
// Mendukung command: /system/resource/print, /system/identity/print.
func (m *MockAdapter) Execute(_ context.Context, command string, _ map[string]string) ([]map[string]string, error) {
	switch command {
	case "/system/resource/print":
		return []map[string]string{
			{
				"version":                "6.49.10 (stable)",
				"board-name":             "RB750Gr3",
				"cpu-count":              "2",
				"cpu-load":               "15",
				"total-memory":           strconv.FormatInt(256*1024*1024, 10),
				"free-memory":            strconv.FormatInt(180*1024*1024, 10),
				"uptime":                 "45d00:00:00",
				"architecture-name":      "mmips",
				"total-hdd-space":        strconv.FormatInt(16*1024*1024, 10),
				"free-hdd-space":         strconv.FormatInt(10*1024*1024, 10),
				"write-sect-since-reboot": "1024",
			},
		}, nil

	case "/system/identity/print":
		return []map[string]string{
			{
				"name": "ISPBoss-Router-Mock",
			},
		}, nil

	default:
		return nil, fmt.Errorf("mock: perintah tidak dikenali: %s", command)
	}
}

// GetSystemResource mengembalikan informasi sistem router dengan nilai simulasi.
// Values: version "6.49.10", board "RB750Gr3", CPU 2, RAM 256MB, uptime 3888000s.
func (m *MockAdapter) GetSystemResource(_ context.Context) (*SystemResource, error) {
	return &SystemResource{
		Version:      "6.49.10",
		BoardName:    "RB750Gr3",
		CPUCount:     2,
		CPULoad:      15,
		TotalRAM:     256 * 1024 * 1024, // 256 MB dalam bytes
		FreeRAM:      180 * 1024 * 1024, // 180 MB dalam bytes
		Uptime:       3888000,           // 45 hari dalam detik
		Architecture: "mmips",
		Identity:     "ISPBoss-Router-Mock",
	}, nil
}

// Ping memeriksa koneksi ke router (selalu sukses pada mock).
func (m *MockAdapter) Ping(_ context.Context) error {
	return nil
}
