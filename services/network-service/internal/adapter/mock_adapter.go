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
func (m *MockAdapter) Execute(_ context.Context, command string, _ map[string]string) ([]map[string]string, error) {
	switch command {
	case "/system/resource/print":
		return []map[string]string{
			{
				"version":                 "6.49.10 (stable)",
				"board-name":              "RB750Gr3",
				"cpu-count":               "2",
				"cpu-load":                "15",
				"total-memory":            strconv.FormatInt(256*1024*1024, 10),
				"free-memory":             strconv.FormatInt(180*1024*1024, 10),
				"uptime":                  "45d00:00:00",
				"architecture-name":       "mmips",
				"total-hdd-space":         strconv.FormatInt(16*1024*1024, 10),
				"free-hdd-space":          strconv.FormatInt(10*1024*1024, 10),
				"write-sect-since-reboot": "1024",
			},
		}, nil

	case "/system/identity/print":
		return []map[string]string{
			{
				"name": "ISPBoss-Router-Mock",
			},
		}, nil

	case "/interface/print":
		return []map[string]string{
			{
				".id":         "*1",
				"name":        "ether1-wan",
				"type":        "ether",
				"mtu":         "1500",
				"mac-address": "AA:BB:CC:00:01:01",
				"running":     "true",
				"disabled":    "false",
				"rx-byte":     "928122344",
				"tx-byte":     "321780011",
				"rx-packet":   "782110",
				"tx-packet":   "512009",
				"comment":     "ISPBoss: uplink utama",
			},
			{
				".id":         "*2",
				"name":        "pppoe-bridge",
				"type":        "bridge",
				"mtu":         "1500",
				"mac-address": "AA:BB:CC:00:01:02",
				"running":     "true",
				"disabled":    "false",
				"rx-byte":     "2211334455",
				"tx-byte":     "1988776655",
				"rx-packet":   "1822110",
				"tx-packet":   "1512009",
			},
		}, nil

	case "/interface/monitor-traffic":
		return []map[string]string{
			{
				"name":                  "ether1-wan",
				"rx-bits-per-second":    "12800000",
				"tx-bits-per-second":    "7400000",
				"rx-packets-per-second": "940",
				"tx-packets-per-second": "611",
			},
		}, nil

	case "/ip/pool/print":
		return []map[string]string{
			{".id": "*10", "name": "pool-pppoe-reguler", "ranges": "10.10.10.2-10.10.10.254"},
			{".id": "*11", "name": "pool-pppoe-isolir", "ranges": "10.99.0.2-10.99.0.100"},
		}, nil

	case "/ip/pool/used/print":
		return []map[string]string{
			{"pool": "pool-pppoe-reguler", "address": "10.10.10.2"},
			{"pool": "pool-pppoe-reguler", "address": "10.10.10.3"},
			{"pool": "pool-pppoe-isolir", "address": "10.99.0.2"},
		}, nil

	case "/ip/firewall/nat/print":
		return []map[string]string{
			{".id": "*20", "chain": "dstnat", "action": "redirect", "disabled": "false", "comment": "ISPBoss: isolir walled garden"},
		}, nil

	case "/ip/firewall/filter/print":
		return []map[string]string{
			{".id": "*21", "chain": "forward", "action": "drop", "disabled": "false", "comment": "ISPBoss: block isolated customer"},
		}, nil

	case "/ip/firewall/address-list/print":
		return []map[string]string{
			{".id": "*22", "list": "isolated-customers", "address": "10.99.0.2", "disabled": "false", "comment": "ISPBoss: customer overdue"},
			{".id": "*23", "list": "walled-garden-allow", "address": "payment.ispboss.local", "disabled": "false", "comment": "ISPBoss: payment portal"},
		}, nil

	case "/log/print":
		return []map[string]string{
			{".id": "*30", "time": "may/04/2026 09:00:00", "topics": "system,info,account", "message": "user api logged in from mock"},
			{".id": "*31", "time": "may/04/2026 09:02:10", "topics": "pppoe,info", "message": "ISPBoss: customer pppoe-test connected"},
			{".id": "*32", "time": "may/04/2026 09:05:42", "topics": "firewall,info", "message": "ISPBoss: isolated customer redirected"},
		}, nil

	case "/ip/dhcp-server/print":
		return []map[string]string{
			{".id": "*40", "name": "dhcp-lan", "interface": "pppoe-bridge", "address-pool": "pool-dhcp-lan", "lease-time": "1d", "authoritative": "yes", "disabled": "false", "comment": "LAN subscribers"},
		}, nil

	case "/ip/dhcp-server/lease/print":
		return []map[string]string{
			{".id": "*41", "server": "dhcp-lan", "address": "10.20.0.10", "mac-address": "02:00:00:00:00:10", "host-name": "cpe-001", "status": "bound", "dynamic": "true", "disabled": "false", "last-seen": "5m", "comment": ""},
			{".id": "*42", "server": "dhcp-lan", "address": "10.20.0.20", "mac-address": "02:00:00:00:00:20", "host-name": "static-cpe", "status": "bound", "dynamic": "false", "disabled": "false", "comment": "ISPBoss:dhcp:mock managed static binding"},
		}, nil

	case "/ip/dhcp-server/network/print":
		return []map[string]string{
			{".id": "*43", "address": "10.20.0.0/24", "gateway": "10.20.0.1", "dns-server": "8.8.8.8,1.1.1.1", "domain": "ispboss.local", "comment": "LAN DHCP"},
		}, nil

	case "/ip/dhcp-server/lease/add", "/ip/dhcp-server/lease/set", "/ip/dhcp-server/lease/remove":
		return []map[string]string{}, nil

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
