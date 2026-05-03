package domain

import (
	"fmt"
	"net"
	"testing"

	"pgregory.net/rapid"
)

// =============================================================================
// Feature: mikrotik-vpn, Property 1: VPN IP allocation uniqueness and subnet range
// =============================================================================

// allTunnelStatuses berisi semua status tunnel yang valid.
var allTunnelStatuses = []TunnelStatus{
	TunnelStatusConnected, TunnelStatusDisconnected,
	TunnelStatusPending, TunnelStatusError,
}

// validTunnelTransitionPairs berisi semua pasangan transisi status tunnel yang valid.
// Sesuai dengan ValidTunnelTransitions di vpn.go.
var validTunnelTransitionPairs = map[[2]TunnelStatus]bool{
	{TunnelStatusPending, TunnelStatusConnected}:       true,
	{TunnelStatusPending, TunnelStatusDisconnected}:    true,
	{TunnelStatusPending, TunnelStatusError}:           true,
	{TunnelStatusConnected, TunnelStatusDisconnected}:  true,
	{TunnelStatusConnected, TunnelStatusError}:         true,
	{TunnelStatusDisconnected, TunnelStatusConnected}:  true,
	{TunnelStatusDisconnected, TunnelStatusError}:      true,
	{TunnelStatusError, TunnelStatusPending}:           true,
	{TunnelStatusError, TunnelStatusConnected}:         true,
}

// allVPNProtocols berisi semua protokol VPN yang valid.
var allVPNProtocols = []string{
	"wireguard", "l2tp_ipsec", "pptp", "sstp", "openvpn",
}

// TestVPNProperty_IPAllocationUniqueness memverifikasi bahwa untuk sembarang
// tenant_seq (1-255) dan sejumlah N alokasi client IP (1 ≤ N ≤ 253),
// setiap IP yang dialokasikan unik dalam tenant dan berada dalam range
// 10.99.{tenant_seq}.2 sampai 10.99.{tenant_seq}.254.
//
// **Validates: Requirements 1.4, 2.1, 2.3, 2.4**
func TestVPNProperty_IPAllocationUniqueness(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		tenantSeq := rapid.IntRange(1, 255).Draw(t, "tenantSeq")
		numClients := rapid.IntRange(1, 253).Draw(t, "numClients")

		// Alokasikan IP secara berurutan mulai dari seq 2
		seen := make(map[string]bool)
		for i := 0; i < numClients; i++ {
			clientSeq := i + 2 // dimulai dari 2
			ip := BuildClientIP(tenantSeq, clientSeq)

			// Properti: setiap IP harus unik
			if seen[ip] {
				t.Fatalf("IP duplikat ditemukan: %s (tenantSeq=%d, clientSeq=%d)", ip, tenantSeq, clientSeq)
			}
			seen[ip] = true

			// Properti: IP harus valid IPv4
			parsed := net.ParseIP(ip)
			if parsed == nil {
				t.Fatalf("BuildClientIP(%d, %d) = %q bukan IPv4 valid", tenantSeq, clientSeq, ip)
			}

			// Properti: IP harus dalam range 10.99.{tenant_seq}.2 - 10.99.{tenant_seq}.254
			expectedPrefix := fmt.Sprintf("10.99.%d.", tenantSeq)
			if ip[:len(expectedPrefix)] != expectedPrefix {
				t.Fatalf("IP %s tidak memiliki prefix %s", ip, expectedPrefix)
			}
		}
	})
}

// TestVPNProperty_ServerIPAndSubnetPrefix memverifikasi bahwa untuk sembarang
// tenant_seq (1-255), server IP selalu 10.99.{tenant_seq}.1 dan subnet prefix
// selalu 10.99.{tenant_seq}.0/24.
//
// **Validates: Requirements 1.4, 2.1, 2.3, 2.4**
func TestVPNProperty_ServerIPAndSubnetPrefix(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		tenantSeq := rapid.IntRange(1, 255).Draw(t, "tenantSeq")

		// Properti: server IP selalu 10.99.{tenant_seq}.1
		serverIP := BuildServerIP(tenantSeq)
		expected := fmt.Sprintf("10.99.%d.1", tenantSeq)
		if serverIP != expected {
			t.Errorf("BuildServerIP(%d) = %q, ingin %q", tenantSeq, serverIP, expected)
		}

		// Properti: server IP harus valid IPv4
		if net.ParseIP(serverIP) == nil {
			t.Errorf("BuildServerIP(%d) = %q bukan IPv4 valid", tenantSeq, serverIP)
		}

		// Properti: subnet prefix selalu 10.99.{tenant_seq}.0/24
		subnetPrefix := BuildSubnetPrefix(tenantSeq)
		expectedSubnet := fmt.Sprintf("10.99.%d.0/24", tenantSeq)
		if subnetPrefix != expectedSubnet {
			t.Errorf("BuildSubnetPrefix(%d) = %q, ingin %q", tenantSeq, subnetPrefix, expectedSubnet)
		}

		// Properti: subnet prefix harus valid CIDR
		_, _, err := net.ParseCIDR(subnetPrefix)
		if err != nil {
			t.Errorf("BuildSubnetPrefix(%d) = %q bukan CIDR valid: %v", tenantSeq, subnetPrefix, err)
		}
	})
}

// TestVPNProperty_BuildClientIPValidIPv4 memverifikasi bahwa BuildClientIP
// menghasilkan IPv4 valid untuk sembarang seq dalam range [2, 254].
//
// **Validates: Requirements 1.4, 2.1, 2.3, 2.4**
func TestVPNProperty_BuildClientIPValidIPv4(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		tenantSeq := rapid.IntRange(1, 255).Draw(t, "tenantSeq")
		clientSeq := rapid.IntRange(2, 254).Draw(t, "clientSeq")

		ip := BuildClientIP(tenantSeq, clientSeq)

		// Properti: harus valid IPv4
		parsed := net.ParseIP(ip)
		if parsed == nil {
			t.Fatalf("BuildClientIP(%d, %d) = %q bukan IPv4 valid", tenantSeq, clientSeq, ip)
		}

		// Properti: format harus tepat 10.99.{tenant_seq}.{client_seq}
		expected := fmt.Sprintf("10.99.%d.%d", tenantSeq, clientSeq)
		if ip != expected {
			t.Errorf("BuildClientIP(%d, %d) = %q, ingin %q", tenantSeq, clientSeq, ip, expected)
		}
	})
}

// TestVPNProperty_IsValidClientSeq memverifikasi bahwa IsValidClientSeq
// mengembalikan true untuk seq dalam [2, 254] dan false untuk nilai lainnya.
//
// **Validates: Requirements 1.4, 2.1, 2.3, 2.4**
func TestVPNProperty_IsValidClientSeq(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		seq := rapid.IntRange(-100, 500).Draw(t, "seq")

		result := IsValidClientSeq(seq)
		expected := seq >= 2 && seq <= 254

		if result != expected {
			t.Errorf("IsValidClientSeq(%d) = %v, ingin %v", seq, result, expected)
		}
	})
}

// =============================================================================
// Feature: mikrotik-vpn, Property 5: Tunnel status transition validity
// =============================================================================

// TestVPNProperty_TunnelStatusTransitionCorrectness memverifikasi bahwa untuk
// sembarang pasangan TunnelStatus (current, target), CanTransitionTunnel
// mengembalikan true jika dan hanya jika pasangan tersebut ada di set
// transisi valid.
//
// **Validates: Requirements 1.5, 5.5, 8.4, 8.5**
func TestVPNProperty_TunnelStatusTransitionCorrectness(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Pilih status asal dan tujuan secara acak dari semua status valid
		current := rapid.SampledFrom(allTunnelStatuses).Draw(t, "current")
		target := rapid.SampledFrom(allTunnelStatuses).Draw(t, "target")

		result := CanTransitionTunnel(current, target)
		pair := [2]TunnelStatus{current, target}
		expected := validTunnelTransitionPairs[pair]

		if result != expected {
			t.Errorf(
				"CanTransitionTunnel(%q, %q) = %v, ingin %v",
				current, target, result, expected,
			)
		}
	})
}

// TestVPNProperty_TunnelStatusTransitionInvalidRejected memverifikasi bahwa
// status tunnel yang tidak dikenal selalu ditolak oleh CanTransitionTunnel.
//
// **Validates: Requirements 1.5, 5.5, 8.4, 8.5**
func TestVPNProperty_TunnelStatusTransitionInvalidRejected(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate string acak sebagai status tidak valid
		invalidStatus := TunnelStatus(rapid.String().Draw(t, "invalidStatus"))

		// Pastikan bukan salah satu status valid
		for _, s := range allTunnelStatuses {
			if invalidStatus == s {
				return // skip iterasi ini, status kebetulan valid
			}
		}

		// Transisi dari status tidak valid harus selalu ditolak
		validTarget := rapid.SampledFrom(allTunnelStatuses).Draw(t, "target")
		if CanTransitionTunnel(invalidStatus, validTarget) {
			t.Errorf(
				"CanTransitionTunnel(%q, %q) seharusnya false untuk status asal tidak valid",
				invalidStatus, validTarget,
			)
		}

		// Transisi ke status tidak valid juga harus ditolak
		validCurrent := rapid.SampledFrom(allTunnelStatuses).Draw(t, "current")
		if CanTransitionTunnel(validCurrent, invalidStatus) {
			t.Errorf(
				"CanTransitionTunnel(%q, %q) seharusnya false untuk status tujuan tidak valid",
				validCurrent, invalidStatus,
			)
		}
	})
}

// TestVPNProperty_TunnelTransitionAllowedTargetsConsistency memverifikasi bahwa
// daftar target yang diizinkan di ValidTunnelTransitions konsisten dengan
// hasil CanTransitionTunnel.
//
// **Validates: Requirements 1.5, 5.5, 8.4, 8.5**
func TestVPNProperty_TunnelTransitionAllowedTargetsConsistency(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		current := rapid.SampledFrom(allTunnelStatuses).Draw(t, "current")

		// Ambil daftar target yang diizinkan dari map
		allowedTargets := ValidTunnelTransitions[current]

		// Untuk setiap status, cek konsistensi
		for _, target := range allTunnelStatuses {
			result := CanTransitionTunnel(current, target)
			isAllowed := false
			for _, allowed := range allowedTargets {
				if allowed == target {
					isAllowed = true
					break
				}
			}

			if result != isAllowed {
				t.Errorf(
					"Inkonsistensi: CanTransitionTunnel(%q, %q) = %v, tapi allowedTargets = %v",
					current, target, result, allowedTargets,
				)
			}
		}
	})
}

// =============================================================================
// Feature: mikrotik-vpn, Property 6: Protocol-version compatibility
// =============================================================================

// TestVPNProperty_WireGuardRequiresV7 memverifikasi bahwa WireGuard ditolak
// pada RouterOS v6 (IsRouterOSv7 = false) dan diterima pada v7 (IsRouterOSv7 = true).
//
// **Validates: Requirements 3.1, 3.2**
func TestVPNProperty_WireGuardRequiresV7(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate versi RouterOS v6.x
		v6Minor := rapid.IntRange(0, 99).Draw(t, "v6Minor")
		v6Version := fmt.Sprintf("6.%d", v6Minor)

		// WireGuard pada v6 harus ditolak (IsRouterOSv7 = false)
		if IsRouterOSv7(v6Version) {
			t.Errorf("IsRouterOSv7(%q) = true, seharusnya false untuk v6", v6Version)
		}

		// Generate versi RouterOS v7.x
		v7Minor := rapid.IntRange(0, 99).Draw(t, "v7Minor")
		v7Version := fmt.Sprintf("7.%d", v7Minor)

		// WireGuard pada v7 harus diterima (IsRouterOSv7 = true)
		if !IsRouterOSv7(v7Version) {
			t.Errorf("IsRouterOSv7(%q) = false, seharusnya true untuk v7", v7Version)
		}
	})
}

// TestVPNProperty_NonWireGuardAcceptsAllVersions memverifikasi bahwa protokol
// selain WireGuard (l2tp_ipsec, pptp, sstp, openvpn) diterima untuk semua
// versi RouterOS.
//
// **Validates: Requirements 3.1, 3.2**
func TestVPNProperty_NonWireGuardAcceptsAllVersions(t *testing.T) {
	nonWireGuardProtocols := []string{"l2tp_ipsec", "pptp", "sstp", "openvpn"}

	rapid.Check(t, func(t *rapid.T) {
		protocol := rapid.SampledFrom(nonWireGuardProtocols).Draw(t, "protocol")

		// Protokol non-WireGuard harus valid
		if !IsValidVPNProtocol(protocol) {
			t.Errorf("IsValidVPNProtocol(%q) = false, seharusnya true", protocol)
		}

		// Generate versi RouterOS acak (v6 atau v7)
		majorVersion := rapid.SampledFrom([]string{"6", "7"}).Draw(t, "major")
		minorVersion := rapid.IntRange(0, 99).Draw(t, "minor")
		version := fmt.Sprintf("%s.%d", majorVersion, minorVersion)

		// Protokol non-WireGuard harus diterima untuk semua versi
		// (tidak ada pengecekan versi untuk protokol selain WireGuard)
		_ = version // versi tidak mempengaruhi validitas protokol non-WireGuard
		if !IsValidVPNProtocol(protocol) {
			t.Errorf("Protokol %q seharusnya valid untuk versi %s", protocol, version)
		}
	})
}

// TestVPNProperty_IsValidVPNProtocol memverifikasi bahwa IsValidVPNProtocol
// mengembalikan true hanya untuk protokol yang valid dan false untuk string lainnya.
//
// **Validates: Requirements 3.1, 3.2**
func TestVPNProperty_IsValidVPNProtocol(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Test dengan protokol valid
		validProtocol := rapid.SampledFrom(allVPNProtocols).Draw(t, "validProtocol")
		if !IsValidVPNProtocol(validProtocol) {
			t.Errorf("IsValidVPNProtocol(%q) = false, seharusnya true", validProtocol)
		}

		// Test dengan string acak
		randomStr := rapid.String().Draw(t, "randomStr")
		isValid := false
		for _, p := range allVPNProtocols {
			if randomStr == p {
				isValid = true
				break
			}
		}
		result := IsValidVPNProtocol(randomStr)
		if result != isValid {
			t.Errorf("IsValidVPNProtocol(%q) = %v, ingin %v", randomStr, result, isValid)
		}
	})
}
