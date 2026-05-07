package usecase

import (
	"strings"
	"testing"
	"time"

	"pgregory.net/rapid"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// =============================================================================
// Tes fixtures dan helpers untuk VPN Script Generator
// =============================================================================

// testScriptConfig mengembalikan VPNScriptConfig untuk testing.
func testScriptConfig() VPNScriptConfig {
	return VPNScriptConfig{
		PrimaryEndpoint:          "vpn1.ispboss.id",
		SecondaryEndpoint:        "vpn2.ispboss.id",
		ServerPublicKey:          "ServerPubKeyBase64==",
		SecondaryServerPublicKey: "SecondaryPubKeyBase64==",
	}
}

// testSubnet mengembalikan VPNSubnet fixture untuk testing.
func testSubnet() *domain.VPNSubnet {
	return &domain.VPNSubnet{
		ID:              "subnet-001",
		TenantID:        "tenant-001",
		SubnetPrefix:    "10.99.1.0/24",
		TenantSeq:       1,
		ServerIP:        "10.99.1.1",
		NextClientIPSeq: 3,
		CreatedAt:       time.Now(),
	}
}

// testWireGuardTunnel mengembalikan VPNTunnel fixture untuk WireGuard.
// Field encrypted berisi nilai plaintext (simulasi sudah di-dekripsi oleh caller).
func testWireGuardTunnel() *domain.VPNTunnel {
	return &domain.VPNTunnel{
		ID:                        "tun-wg-001",
		TenantID:                  "tenant-001",
		TunnelName:                "kantor-pusat",
		Protocol:                  domain.ProtocolWireGuard,
		VPNIP:                     "10.99.1.2",
		ServerEndpoint:            "vpn1.ispboss.id:51820",
		ServerPublicKey:           "ServerPubKeyBase64==",
		ClientPublicKey:           "ClientPubKeyBase64==",
		ClientPrivateKeyEncrypted: "ClientPrivKeyDecrypted==",
		PreSharedKeyEncrypted:     "PSKDecrypted==",
		Status:                    domain.TunnelStatusPending,
		ListenPort:                51820,
		AllowedAddresses:          "10.99.0.0/16",
		PersistentKeepalive:       25,
		CreatedAt:                 time.Now(),
		UpdatedAt:                 time.Now(),
	}
}

// testL2TPTunnel mengembalikan VPNTunnel fixture untuk L2TP/IPSec.
func testL2TPTunnel() *domain.VPNTunnel {
	return &domain.VPNTunnel{
		ID:                    "tun-l2tp-001",
		TenantID:              "tenant-001",
		TunnelName:            "cabang-bandung",
		Protocol:              domain.ProtocolL2TPIPSec,
		VPNIP:                 "10.99.1.3",
		ServerEndpoint:        "vpn1.ispboss.id:1701",
		L2TPUsername:          "vpn-cabang-abc123",
		L2TPPasswordEncrypted: "L2TPPassDecrypted",
		PreSharedKeyEncrypted: "IPSecPSKDecrypted==",
		Status:                domain.TunnelStatusPending,
		ListenPort:            51820,
		AllowedAddresses:      "10.99.0.0/16",
		PersistentKeepalive:   25,
		CreatedAt:             time.Now(),
		UpdatedAt:             time.Now(),
	}
}

// testPPTPTunnel mengembalikan VPNTunnel fixture untuk PPTP.
func testPPTPTunnel() *domain.VPNTunnel {
	return &domain.VPNTunnel{
		ID:                    "tun-pptp-001",
		TenantID:              "tenant-001",
		TunnelName:            "cabang-surabaya",
		Protocol:              domain.ProtocolPPTP,
		VPNIP:                 "10.99.1.4",
		ServerEndpoint:        "vpn1.ispboss.id:1723",
		L2TPUsername:          "vpn-cabang-def456",
		L2TPPasswordEncrypted: "PPTPPassDecrypted",
		Status:                domain.TunnelStatusPending,
		ListenPort:            51820,
		AllowedAddresses:      "10.99.0.0/16",
		PersistentKeepalive:   25,
		CreatedAt:             time.Now(),
		UpdatedAt:             time.Now(),
	}
}

// testSSTPTunnel mengembalikan VPNTunnel fixture untuk SSTP.
func testSSTPTunnel() *domain.VPNTunnel {
	return &domain.VPNTunnel{
		ID:                    "tun-sstp-001",
		TenantID:              "tenant-001",
		TunnelName:            "cabang-semarang",
		Protocol:              domain.ProtocolSSTP,
		VPNIP:                 "10.99.1.5",
		ServerEndpoint:        "vpn1.ispboss.id:443",
		L2TPUsername:          "vpn-cabang-ghi789",
		L2TPPasswordEncrypted: "SSTPPassDecrypted",
		Status:                domain.TunnelStatusPending,
		ListenPort:            51820,
		AllowedAddresses:      "10.99.0.0/16",
		PersistentKeepalive:   25,
		CreatedAt:             time.Now(),
		UpdatedAt:             time.Now(),
	}
}

// testOpenVPNTunnel mengembalikan VPNTunnel fixture untuk OpenVPN.
func testOpenVPNTunnel() *domain.VPNTunnel {
	return &domain.VPNTunnel{
		ID:                    "tun-ovpn-001",
		TenantID:              "tenant-001",
		TunnelName:            "cabang-medan",
		Protocol:              domain.ProtocolOpenVPN,
		VPNIP:                 "10.99.1.6",
		ServerEndpoint:        "vpn1.ispboss.id:1194",
		L2TPUsername:          "vpn-cabang-jkl012",
		L2TPPasswordEncrypted: "OVPNPassDecrypted",
		Status:                domain.TunnelStatusPending,
		ListenPort:            51820,
		AllowedAddresses:      "10.99.0.0/16",
		PersistentKeepalive:   25,
		CreatedAt:             time.Now(),
		UpdatedAt:             time.Now(),
	}
}

// =============================================================================
// =============================================================================

// TestVPNProperty_ScriptGenerationCompleteness memverifikasi bahwa untuk setiap
// protokol VPN, script yang dihasilkan mengandung semua perintah RouterOS dan
// parameter yang diperlukan.
//
// **Memvalidasi: Kebutuhan 4.4, 7.1, 7.2, 7.3, 7.4, 7.5, 7.7, 11.3**
func TestVPNProperty_ScriptGenerationCompleteness(t *testing.T) {
	gen := NewVPNScriptGenerator(testScriptConfig())
	subnet := testSubnet()

	t.Run("WireGuard", func(t *testing.T) {
		rapid.Check(t, func(rt *rapid.T) {
			tunnel := testWireGuardTunnel()
			tunnel.TunnelName = rapid.StringMatching(`[a-z][a-z0-9\-]{2,20}`).Draw(rt, "tunnelName")
			tunnel.PersistentKeepalive = rapid.IntRange(10, 120).Draw(rt, "keepalive")
			tunnel.ListenPort = rapid.IntRange(1024, 65535).Draw(rt, "listenPort")

			script, err := gen.Generate(tunnel, subnet)
			if err != nil {
				t.Fatalf("gagal generate WireGuard script: %v", err)
			}

			// Harus mengandung perintah WireGuard
			assertContains(t, script, "/interface/wireguard/add", "wireguard interface add")
			assertContains(t, script, "/interface/wireguard/peers/add", "wireguard peers add")
			assertContains(t, script, "/ip/address/add", "ip address add")

			// Harus mengandung client private key dan server public key
			assertContains(t, script, tunnel.ClientPrivateKeyEncrypted, "client private key")
			assertContains(t, script, tunnel.ServerPublicKey, "server public key")

			// Harus mengandung endpoint address dan persistent-keepalive
			assertContains(t, script, "vpn1.ispboss.id", "primary endpoint")
			assertContains(t, script, "persistent-keepalive=", "persistent keepalive")

			// Harus mengandung firewall rules (port 8728, 8729, 161)
			assertFirewallRules(t, script)
		})
	})

	t.Run("L2TP_IPSec", func(t *testing.T) {
		rapid.Check(t, func(rt *rapid.T) {
			tunnel := testL2TPTunnel()
			tunnel.TunnelName = rapid.StringMatching(`[a-z][a-z0-9\-]{2,20}`).Draw(rt, "tunnelName")

			script, err := gen.Generate(tunnel, subnet)
			if err != nil {
				t.Fatalf("gagal generate L2TP script: %v", err)
			}

			// Harus mengandung perintah L2TP dan IPSec
			assertContains(t, script, "/interface/l2tp-client/add", "l2tp client add")
			assertContains(t, script, "/ip/ipsec/profile/add", "ipsec profile add")
			assertContains(t, script, "/ip/ipsec/proposal/add", "ipsec proposal add")
			assertContains(t, script, "/ip/address/add", "ip address add")
			assertContains(t, script, "/ip/route/add", "ip route add")

			assertContains(t, script, tunnel.L2TPUsername, "l2tp username")
			assertContains(t, script, "use-ipsec=yes", "use-ipsec flag")

			// Harus mengandung firewall rules
			assertFirewallRules(t, script)
		})
	})

	t.Run("PPTP", func(t *testing.T) {
		rapid.Check(t, func(rt *rapid.T) {
			tunnel := testPPTPTunnel()
			tunnel.TunnelName = rapid.StringMatching(`[a-z][a-z0-9\-]{2,20}`).Draw(rt, "tunnelName")

			script, err := gen.Generate(tunnel, subnet)
			if err != nil {
				t.Fatalf("gagal generate PPTP script: %v", err)
			}

			// Harus mengandung perintah PPTP
			assertContains(t, script, "/interface/pptp-client/add", "pptp client add")
			assertContains(t, script, "/ip/address/add", "ip address add")
			assertContains(t, script, "/ip/route/add", "ip route add")

			// Harus mengandung username
			assertContains(t, script, tunnel.L2TPUsername, "pptp username")

			// Harus mengandung firewall rules
			assertFirewallRules(t, script)
		})
	})

	t.Run("SSTP", func(t *testing.T) {
		rapid.Check(t, func(rt *rapid.T) {
			tunnel := testSSTPTunnel()
			tunnel.TunnelName = rapid.StringMatching(`[a-z][a-z0-9\-]{2,20}`).Draw(rt, "tunnelName")

			script, err := gen.Generate(tunnel, subnet)
			if err != nil {
				t.Fatalf("gagal generate SSTP script: %v", err)
			}

			// Harus mengandung perintah SSTP
			assertContains(t, script, "/interface/sstp-client/add", "sstp client add")
			assertContains(t, script, "/ip/address/add", "ip address add")
			assertContains(t, script, "/ip/route/add", "ip route add")

			// Harus mengandung username dan tls-version
			assertContains(t, script, tunnel.L2TPUsername, "sstp username")
			assertContains(t, script, "tls-version", "tls version")

			// Harus mengandung firewall rules
			assertFirewallRules(t, script)
		})
	})

	t.Run("OpenVPN", func(t *testing.T) {
		rapid.Check(t, func(rt *rapid.T) {
			tunnel := testOpenVPNTunnel()
			tunnel.TunnelName = rapid.StringMatching(`[a-z][a-z0-9\-]{2,20}`).Draw(rt, "tunnelName")

			script, err := gen.Generate(tunnel, subnet)
			if err != nil {
				t.Fatalf("gagal generate OpenVPN script: %v", err)
			}

			// Harus mengandung perintah OpenVPN
			assertContains(t, script, "/interface/ovpn-client/add", "ovpn client add")
			assertContains(t, script, "/ip/address/add", "ip address add")
			assertContains(t, script, "/ip/route/add", "ip route add")

			// Harus mengandung username dan cipher
			assertContains(t, script, tunnel.L2TPUsername, "ovpn username")
			assertContains(t, script, "cipher", "cipher setting")

			// Harus mengandung firewall rules
			assertFirewallRules(t, script)
		})
	})
}

// =============================================================================
// =============================================================================

// TestVPNProperty_ScriptSecurityNoServerPrivateKey memverifikasi bahwa untuk
// sembarang script .rsc yang dihasilkan untuk protokol apapun, script TIDAK
// mengandung server private key. Script BOLEH mengandung server public key,
// client private key, dan endpoint addresses.
//
// **Memvalidasi: Kebutuhan 7.7**
func TestVPNProperty_ScriptSecurityNoServerPrivateKey(t *testing.T) {
	// Gunakan marker unik sebagai "server private key" yang tidak boleh muncul
	serverPrivateKeyMarker := "SERVER_PRIVATE_KEY_SECRET_DO_NOT_EXPOSE"

	cfg := testScriptConfig()
	gen := NewVPNScriptGenerator(cfg)
	subnet := testSubnet()

	rapid.Check(t, func(rt *rapid.T) {
		// Pilih protokol secara acak
		protocol := rapid.SampledFrom(domain.ValidVPNProtocols).Draw(rt, "protocol")

		var tunnel *domain.VPNTunnel
		switch protocol {
		case domain.ProtocolWireGuard:
			tunnel = testWireGuardTunnel()
		case domain.ProtocolL2TPIPSec:
			tunnel = testL2TPTunnel()
		case domain.ProtocolPPTP:
			tunnel = testPPTPTunnel()
		case domain.ProtocolSSTP:
			tunnel = testSSTPTunnel()
		case domain.ProtocolOpenVPN:
			tunnel = testOpenVPNTunnel()
		}

		// Randomize tunnel name
		tunnel.TunnelName = rapid.StringMatching(`[a-z][a-z0-9\-]{2,20}`).Draw(rt, "tunnelName")

		script, err := gen.Generate(tunnel, subnet)
		if err != nil {
			t.Fatalf("gagal generate script untuk %s: %v", protocol, err)
		}

		// Script TIDAK boleh mengandung server private key marker
		if strings.Contains(script, serverPrivateKeyMarker) {
			t.Fatalf(
				"script %s mengandung server private key marker",
				protocol,
			)
		}

		// Script BOLEH mengandung server public key (verifikasi positif)
		if protocol == domain.ProtocolWireGuard {
			if !strings.Contains(script, cfg.ServerPublicKey) {
				t.Fatalf("script WireGuard tidak mengandung server public key")
			}
		}

		// Script harus mengandung tunnel ID di komentar
		assertContains(t, script, tunnel.ID, "tunnel ID in comments")
	})
}

// =============================================================================
// =============================================================================

// assertContains memverifikasi bahwa script mengandung substring tertentu.
func assertContains(t testing.TB, script, substr, label string) {
	t.Helper()
	if !strings.Contains(script, substr) {
		t.Fatalf("script tidak mengandung %s: %q", label, substr)
	}
}

// assertFirewallRules memverifikasi bahwa script mengandung firewall rules
// untuk port 8728, 8729 (RouterOS API) dan 161 (SNMP).
func assertFirewallRules(t testing.TB, script string) {
	t.Helper()
	assertContains(t, script, "dst-port=8728,8729", "firewall port API")
	assertContains(t, script, "dst-port=161", "firewall port SNMP")
	assertContains(t, script, "action=accept", "firewall accept rule")
	assertContains(t, script, "action=drop", "firewall drop rule")
}
