// Paket adapter menyediakan implementasi RouterOS adapter (mock dan live).
// File ini mendefinisikan type alias dari domain untuk kemudahan akses di package adapter.
package adapter

import (
	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// RouterOSAdapter adalah alias dari domain.RouterOSAdapter.
// Interface ini mendefinisikan kontrak komunikasi dengan RouterOS API.
type RouterOSAdapter = domain.RouterOSAdapter

// ConnectionConfig adalah alias dari domain.ConnectionConfig.
// Berisi konfigurasi koneksi ke router MikroTik (host, port, credential, timeout).
type ConnectionConfig = domain.ConnectionConfig

// SystemResource adalah alias dari domain.SystemResource.
// Berisi informasi sistem yang diambil dari router (CPU, RAM, uptime, dll).
type SystemResource = domain.SystemResource
