package adapter

import (
	"fmt"
	"testing"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// =============================================================================
// Unit tests untuk VPNCommandBuilder - memverifikasi setiap method menghasilkan
// command path dan args map yang benar untuk operasi VPN RouterOS.
// =============================================================================

func newTestVPNBuilder() domain.VPNCommandBuilder {
	return NewVPNCommandBuilder()
}

// --- WireGuard Tests ---

// TestCreateWireGuardInterface memverifikasi command path dan parameter lengkap.
func TestCreateWireGuardInterface(t *testing.T) {
	builder := newTestVPNBuilder()
	params := domain.WireGuardInterfaceParams{
		Name:       "ispboss-vpn",
		ListenPort: 51820,
		PrivateKey: "cGVtdWxhLXByaXZhdGUta2V5LWJhc2U2NA==",
	}

	cmd, args := builder.CreateWireGuardInterface(params)

	// Verifikasi command path
	if cmd != "/interface/wireguard/add" {
		t.Errorf("command = %q, want /interface/wireguard/add", cmd)
	}

	// Verifikasi semua parameter wajib ada
	requiredKeys := []string{"=name", "=listen-port", "=private-key"}
	for _, key := range requiredKeys {
		if _, ok := args[key]; !ok {
			t.Errorf("args tidak memiliki key wajib %q", key)
		}
	}

	if args["=name"] != "ispboss-vpn" {
		t.Errorf("args[=name] = %q, want %q", args["=name"], "ispboss-vpn")
	}
	if args["=listen-port"] != "51820" {
		t.Errorf("args[=listen-port] = %q, want %q", args["=listen-port"], "51820")
	}
	if args["=private-key"] != params.PrivateKey {
		t.Errorf("args[=private-key] = %q, want %q", args["=private-key"], params.PrivateKey)
	}
}

// TestAddWireGuardPeer memverifikasi command path dan parameter peer lengkap.
func TestAddWireGuardPeer(t *testing.T) {
	builder := newTestVPNBuilder()
	params := domain.WireGuardPeerParams{
		Interface:           "ispboss-vpn",
		PublicKey:           "c2VydmVyLXB1YmxpYy1rZXktYmFzZTY0",
		PreSharedKey:        "cHNrLWtleS1iYXNlNjQ=",
		EndpointAddress:     "vpn.ispboss.id",
		EndpointPort:        51820,
		AllowedAddress:      "10.99.0.0/16",
		PersistentKeepalive: 25,
	}

	cmd, args := builder.AddWireGuardPeer(params)

	if cmd != "/interface/wireguard/peers/add" {
		t.Errorf("command = %q, want /interface/wireguard/peers/add", cmd)
	}

	// Verifikasi semua parameter wajib (termasuk PSK karena non-empty)
	requiredKeys := []string{
		"=interface", "=public-key", "=preshared-key",
		"=endpoint-address", "=endpoint-port",
		"=allowed-address", "=persistent-keepalive",
	}
	for _, key := range requiredKeys {
		if _, ok := args[key]; !ok {
			t.Errorf("args tidak memiliki key wajib %q", key)
		}
	}

	if args["=interface"] != "ispboss-vpn" {
		t.Errorf("args[=interface] = %q, want %q", args["=interface"], "ispboss-vpn")
	}
	if args["=public-key"] != params.PublicKey {
		t.Errorf("args[=public-key] = %q, want %q", args["=public-key"], params.PublicKey)
	}
	if args["=endpoint-port"] != "51820" {
		t.Errorf("args[=endpoint-port] = %q, want %q", args["=endpoint-port"], "51820")
	}
	if args["=persistent-keepalive"] != "25" {
		t.Errorf("args[=persistent-keepalive] = %q, want %q", args["=persistent-keepalive"], "25")
	}
}

// TestAddWireGuardPeerTanpaPSK memverifikasi PSK tidak ditambahkan jika kosong.
func TestAddWireGuardPeerTanpaPSK(t *testing.T) {
	builder := newTestVPNBuilder()
	params := domain.WireGuardPeerParams{
		Interface:           "ispboss-vpn",
		PublicKey:           "c2VydmVyLXB1YmxpYy1rZXk=",
		PreSharedKey:        "", // kosong
		EndpointAddress:     "vpn.ispboss.id",
		EndpointPort:        51820,
		AllowedAddress:      "10.99.0.0/16",
		PersistentKeepalive: 25,
	}

	_, args := builder.AddWireGuardPeer(params)

	if _, ok := args["=preshared-key"]; ok {
		t.Error("args seharusnya tidak memiliki =preshared-key saat PSK kosong")
	}
}

// TestRemoveWireGuardInterface memverifikasi command path dan parameter remove.
func TestRemoveWireGuardInterface(t *testing.T) {
	builder := newTestVPNBuilder()
	cmd, args := builder.RemoveWireGuardInterface("ispboss-vpn")

	if cmd != "/interface/wireguard/remove" {
		t.Errorf("command = %q, want /interface/wireguard/remove", cmd)
	}
	if args["=numbers"] != "ispboss-vpn" {
		t.Errorf("args[=numbers] = %q, want %q", args["=numbers"], "ispboss-vpn")
	}
}

// TestRemoveWireGuardPeer memverifikasi command path dan parameter remove peer.
func TestRemoveWireGuardPeer(t *testing.T) {
	builder := newTestVPNBuilder()
	cmd, args := builder.RemoveWireGuardPeer("ispboss-vpn")

	if cmd != "/interface/wireguard/peers/remove" {
		t.Errorf("command = %q, want /interface/wireguard/peers/remove", cmd)
	}
	if args["=interface"] != "ispboss-vpn" {
		t.Errorf("args[=interface] = %q, want %q", args["=interface"], "ispboss-vpn")
	}
}

// --- L2TP Tests ---

// TestCreateL2TPClient memverifikasi command path dan parameter L2TP client.
func TestCreateL2TPClient(t *testing.T) {
	builder := newTestVPNBuilder()
	params := domain.L2TPClientParams{
		Name:          "ispboss-l2tp",
		ConnectTo:     "vpn.ispboss.id",
		User:          "tunnel-user",
		Password:      "tunnel-pass",
		UseIPSec:      "yes",
		IPSecSecret:   "ipsec-psk-secret",
		AllowFastPath: "yes",
		Profile:       "default-encryption",
	}

	cmd, args := builder.CreateL2TPClient(params)

	if cmd != "/interface/l2tp-client/add" {
		t.Errorf("command = %q, want /interface/l2tp-client/add", cmd)
	}

	requiredKeys := []string{
		"=name", "=connect-to", "=user", "=password",
		"=use-ipsec", "=ipsec-secret", "=profile", "=allow-fast-path",
	}
	for _, key := range requiredKeys {
		if _, ok := args[key]; !ok {
			t.Errorf("args tidak memiliki key wajib %q", key)
		}
	}

	if args["=name"] != "ispboss-l2tp" {
		t.Errorf("args[=name] = %q, want %q", args["=name"], "ispboss-l2tp")
	}
	if args["=connect-to"] != "vpn.ispboss.id" {
		t.Errorf("args[=connect-to] = %q, want %q", args["=connect-to"], "vpn.ispboss.id")
	}
	if args["=user"] != "tunnel-user" {
		t.Errorf("args[=user] = %q, want %q", args["=user"], "tunnel-user")
	}
	if args["=use-ipsec"] != "yes" {
		t.Errorf("args[=use-ipsec] = %q, want %q", args["=use-ipsec"], "yes")
	}
}

// TestCreateL2TPClientTanpaFastPath memverifikasi allow-fast-path tidak ada jika kosong.
func TestCreateL2TPClientTanpaFastPath(t *testing.T) {
	builder := newTestVPNBuilder()
	params := domain.L2TPClientParams{
		Name:        "ispboss-l2tp",
		ConnectTo:   "vpn.ispboss.id",
		User:        "user",
		Password:    "pass",
		UseIPSec:    "yes",
		IPSecSecret: "secret",
		Profile:     "default-encryption",
	}

	_, args := builder.CreateL2TPClient(params)

	if _, ok := args["=allow-fast-path"]; ok {
		t.Error("args seharusnya tidak memiliki =allow-fast-path saat kosong")
	}
}

// TestRemoveL2TPClient memverifikasi command path dan parameter remove.
func TestRemoveL2TPClient(t *testing.T) {
	builder := newTestVPNBuilder()
	cmd, args := builder.RemoveL2TPClient("ispboss-l2tp")

	if cmd != "/interface/l2tp-client/remove" {
		t.Errorf("command = %q, want /interface/l2tp-client/remove", cmd)
	}
	if args["=numbers"] != "ispboss-l2tp" {
		t.Errorf("args[=numbers] = %q, want %q", args["=numbers"], "ispboss-l2tp")
	}
}

// TestCreateIPSecProfile memverifikasi command path dan parameter IPSec profile.
func TestCreateIPSecProfile(t *testing.T) {
	builder := newTestVPNBuilder()
	params := domain.IPSecProfileParams{
		Name:          "ispboss-ipsec",
		HashAlgorithm: "sha256",
		EncAlgorithm:  "aes-256",
		DHGroup:       "modp2048",
		Lifetime:      "1d",
		ProposalCheck: "obey",
	}

	cmd, args := builder.CreateIPSecProfile(params)

	if cmd != "/ip/ipsec/profile/add" {
		t.Errorf("command = %q, want /ip/ipsec/profile/add", cmd)
	}

	requiredKeys := []string{
		"=name", "=hash-algorithm", "=enc-algorithm",
		"=dh-group", "=lifetime", "=proposal-check",
	}
	for _, key := range requiredKeys {
		if _, ok := args[key]; !ok {
			t.Errorf("args tidak memiliki key wajib %q", key)
		}
	}

	if args["=name"] != "ispboss-ipsec" {
		t.Errorf("args[=name] = %q, want %q", args["=name"], "ispboss-ipsec")
	}
	if args["=hash-algorithm"] != "sha256" {
		t.Errorf("args[=hash-algorithm] = %q, want %q", args["=hash-algorithm"], "sha256")
	}
	if args["=enc-algorithm"] != "aes-256" {
		t.Errorf("args[=enc-algorithm] = %q, want %q", args["=enc-algorithm"], "aes-256")
	}
}

// TestCreateIPSecProposal memverifikasi command path dan parameter IPSec proposal.
func TestCreateIPSecProposal(t *testing.T) {
	builder := newTestVPNBuilder()
	params := domain.IPSecProposalParams{
		Name:          "ispboss-proposal",
		AuthAlgorithm: "sha256",
		EncAlgorithm:  "aes-256-cbc",
		Lifetime:      "30m",
		PFSGroup:      "modp2048",
	}

	cmd, args := builder.CreateIPSecProposal(params)

	if cmd != "/ip/ipsec/proposal/add" {
		t.Errorf("command = %q, want /ip/ipsec/proposal/add", cmd)
	}

	requiredKeys := []string{
		"=name", "=auth-algorithms", "=enc-algorithms",
		"=lifetime", "=pfs-group",
	}
	for _, key := range requiredKeys {
		if _, ok := args[key]; !ok {
			t.Errorf("args tidak memiliki key wajib %q", key)
		}
	}

	if args["=auth-algorithms"] != "sha256" {
		t.Errorf("args[=auth-algorithms] = %q, want %q", args["=auth-algorithms"], "sha256")
	}
	if args["=pfs-group"] != "modp2048" {
		t.Errorf("args[=pfs-group] = %q, want %q", args["=pfs-group"], "modp2048")
	}
}

// --- PPTP Tests ---

// TestCreatePPTPClient memverifikasi command path dan parameter PPTP client.
func TestCreatePPTPClient(t *testing.T) {
	builder := newTestVPNBuilder()
	params := domain.PPTPClientParams{
		Name:      "ispboss-pptp",
		ConnectTo: "vpn.ispboss.id",
		User:      "pptp-user",
		Password:  "pptp-pass",
		Profile:   "default-encryption",
	}

	cmd, args := builder.CreatePPTPClient(params)

	if cmd != "/interface/pptp-client/add" {
		t.Errorf("command = %q, want /interface/pptp-client/add", cmd)
	}

	requiredKeys := []string{"=name", "=connect-to", "=user", "=password", "=profile"}
	for _, key := range requiredKeys {
		if _, ok := args[key]; !ok {
			t.Errorf("args tidak memiliki key wajib %q", key)
		}
	}

	if args["=name"] != "ispboss-pptp" {
		t.Errorf("args[=name] = %q, want %q", args["=name"], "ispboss-pptp")
	}
	if args["=connect-to"] != "vpn.ispboss.id" {
		t.Errorf("args[=connect-to] = %q, want %q", args["=connect-to"], "vpn.ispboss.id")
	}
}

// TestRemovePPTPClient memverifikasi command path dan parameter remove.
func TestRemovePPTPClient(t *testing.T) {
	builder := newTestVPNBuilder()
	cmd, args := builder.RemovePPTPClient("ispboss-pptp")

	if cmd != "/interface/pptp-client/remove" {
		t.Errorf("command = %q, want /interface/pptp-client/remove", cmd)
	}
	if args["=numbers"] != "ispboss-pptp" {
		t.Errorf("args[=numbers] = %q, want %q", args["=numbers"], "ispboss-pptp")
	}
}

// --- SSTP Tests ---

// TestCreateSSTPClient memverifikasi command path dan parameter SSTP client.
func TestCreateSSTPClient(t *testing.T) {
	builder := newTestVPNBuilder()
	params := domain.SSTPClientParams{
		Name:              "ispboss-sstp",
		ConnectTo:         "vpn.ispboss.id",
		User:              "sstp-user",
		Password:          "sstp-pass",
		Profile:           "default-encryption",
		CertificateVerify: "no",
		TLSVersion:        "only-1.2",
	}

	cmd, args := builder.CreateSSTPClient(params)

	if cmd != "/interface/sstp-client/add" {
		t.Errorf("command = %q, want /interface/sstp-client/add", cmd)
	}

	requiredKeys := []string{
		"=name", "=connect-to", "=user", "=password",
		"=profile", "=certificate-verify", "=tls-version",
	}
	for _, key := range requiredKeys {
		if _, ok := args[key]; !ok {
			t.Errorf("args tidak memiliki key wajib %q", key)
		}
	}

	if args["=certificate-verify"] != "no" {
		t.Errorf("args[=certificate-verify] = %q, want %q", args["=certificate-verify"], "no")
	}
	if args["=tls-version"] != "only-1.2" {
		t.Errorf("args[=tls-version] = %q, want %q", args["=tls-version"], "only-1.2")
	}
}

// TestRemoveSSTPClient memverifikasi command path dan parameter remove.
func TestRemoveSSTPClient(t *testing.T) {
	builder := newTestVPNBuilder()
	cmd, args := builder.RemoveSSTPClient("ispboss-sstp")

	if cmd != "/interface/sstp-client/remove" {
		t.Errorf("command = %q, want /interface/sstp-client/remove", cmd)
	}
	if args["=numbers"] != "ispboss-sstp" {
		t.Errorf("args[=numbers] = %q, want %q", args["=numbers"], "ispboss-sstp")
	}
}

// --- OpenVPN Tests ---

// TestCreateOpenVPNClient memverifikasi command path dan parameter OpenVPN client.
func TestCreateOpenVPNClient(t *testing.T) {
	builder := newTestVPNBuilder()
	params := domain.OpenVPNClientParams{
		Name:        "ispboss-ovpn",
		ConnectTo:   "vpn.ispboss.id",
		Port:        1194,
		User:        "ovpn-user",
		Password:    "ovpn-pass",
		Mode:        "ip",
		Protocol:    "tcp",
		Certificate: "ispboss-cert",
		Auth:        "sha256",
		Cipher:      "aes-256-cbc",
		Profile:     "default-encryption",
	}

	cmd, args := builder.CreateOpenVPNClient(params)

	if cmd != "/interface/ovpn-client/add" {
		t.Errorf("command = %q, want /interface/ovpn-client/add", cmd)
	}

	requiredKeys := []string{
		"=name", "=connect-to", "=port", "=user", "=password",
		"=mode", "=protocol", "=certificate", "=auth", "=cipher", "=profile",
	}
	for _, key := range requiredKeys {
		if _, ok := args[key]; !ok {
			t.Errorf("args tidak memiliki key wajib %q", key)
		}
	}

	if args["=port"] != "1194" {
		t.Errorf("args[=port] = %q, want %q", args["=port"], "1194")
	}
	if args["=mode"] != "ip" {
		t.Errorf("args[=mode] = %q, want %q", args["=mode"], "ip")
	}
	if args["=protocol"] != "tcp" {
		t.Errorf("args[=protocol] = %q, want %q", args["=protocol"], "tcp")
	}
	if args["=cipher"] != "aes-256-cbc" {
		t.Errorf("args[=cipher] = %q, want %q", args["=cipher"], "aes-256-cbc")
	}
}

// TestCreateOpenVPNClientTanpaCertificate memverifikasi certificate tidak ada jika kosong.
func TestCreateOpenVPNClientTanpaCertificate(t *testing.T) {
	builder := newTestVPNBuilder()
	params := domain.OpenVPNClientParams{
		Name:      "ispboss-ovpn",
		ConnectTo: "vpn.ispboss.id",
		Port:      1194,
		User:      "ovpn-user",
		Password:  "ovpn-pass",
		Mode:      "ip",
		Protocol:  "tcp",
		Auth:      "sha256",
		Cipher:    "aes-256-cbc",
		Profile:   "default-encryption",
	}

	_, args := builder.CreateOpenVPNClient(params)

	if _, ok := args["=certificate"]; ok {
		t.Error("args seharusnya tidak memiliki =certificate saat kosong")
	}
}

// TestRemoveOpenVPNClient memverifikasi command path dan parameter remove.
func TestRemoveOpenVPNClient(t *testing.T) {
	builder := newTestVPNBuilder()
	cmd, args := builder.RemoveOpenVPNClient("ispboss-ovpn")

	if cmd != "/interface/ovpn-client/remove" {
		t.Errorf("command = %q, want /interface/ovpn-client/remove", cmd)
	}
	if args["=numbers"] != "ispboss-ovpn" {
		t.Errorf("args[=numbers] = %q, want %q", args["=numbers"], "ispboss-ovpn")
	}
}

// --- Common Commands Tests ---

// TestAddIPAddress memverifikasi command path dan parameter IP address.
func TestAddIPAddress(t *testing.T) {
	builder := newTestVPNBuilder()
	params := domain.IPAddressParams{
		Address:   "10.99.1.2/24",
		Interface: "ispboss-vpn",
		Comment:   "ISPBoss:vpn:tunnel-123",
	}

	cmd, args := builder.AddIPAddress(params)

	if cmd != "/ip/address/add" {
		t.Errorf("command = %q, want /ip/address/add", cmd)
	}

	requiredKeys := []string{"=address", "=interface", "=comment"}
	for _, key := range requiredKeys {
		if _, ok := args[key]; !ok {
			t.Errorf("args tidak memiliki key wajib %q", key)
		}
	}

	if args["=address"] != "10.99.1.2/24" {
		t.Errorf("args[=address] = %q, want %q", args["=address"], "10.99.1.2/24")
	}
	if args["=interface"] != "ispboss-vpn" {
		t.Errorf("args[=interface] = %q, want %q", args["=interface"], "ispboss-vpn")
	}
}

// TestRemoveIPAddressByInterface memverifikasi command path dan parameter remove.
func TestRemoveIPAddressByInterface(t *testing.T) {
	builder := newTestVPNBuilder()
	cmd, args := builder.RemoveIPAddressByInterface("ispboss-vpn")

	if cmd != "/ip/address/remove" {
		t.Errorf("command = %q, want /ip/address/remove", cmd)
	}
	if args["=interface"] != "ispboss-vpn" {
		t.Errorf("args[=interface] = %q, want %q", args["=interface"], "ispboss-vpn")
	}
}

// TestAddIPRoute memverifikasi command path dan parameter IP route.
func TestAddIPRoute(t *testing.T) {
	builder := newTestVPNBuilder()
	params := domain.IPRouteParams{
		DstAddress: "10.99.0.0/16",
		Gateway:    "ispboss-vpn",
		Comment:    "ISPBoss:vpn-route:tunnel-123",
	}

	cmd, args := builder.AddIPRoute(params)

	if cmd != "/ip/route/add" {
		t.Errorf("command = %q, want /ip/route/add", cmd)
	}

	requiredKeys := []string{"=dst-address", "=gateway", "=comment"}
	for _, key := range requiredKeys {
		if _, ok := args[key]; !ok {
			t.Errorf("args tidak memiliki key wajib %q", key)
		}
	}

	if args["=dst-address"] != "10.99.0.0/16" {
		t.Errorf("args[=dst-address] = %q, want %q", args["=dst-address"], "10.99.0.0/16")
	}
	if args["=gateway"] != "ispboss-vpn" {
		t.Errorf("args[=gateway] = %q, want %q", args["=gateway"], "ispboss-vpn")
	}
}

// TestAddFirewallFilter memverifikasi command path dan parameter firewall filter.
func TestAddFirewallFilter(t *testing.T) {
	builder := newTestVPNBuilder()
	params := domain.FirewallFilterParams{
		Chain:       "input",
		InInterface: "ispboss-vpn",
		Protocol:    "tcp",
		DstPort:     "8728,8729,161",
		Action:      "accept",
		Comment:     "ISPBoss:vpn-firewall:tunnel-123",
	}

	cmd, args := builder.AddFirewallFilter(params)

	if cmd != "/ip/firewall/filter/add" {
		t.Errorf("command = %q, want /ip/firewall/filter/add", cmd)
	}

	requiredKeys := []string{
		"=chain", "=in-interface", "=protocol",
		"=dst-port", "=action", "=comment",
	}
	for _, key := range requiredKeys {
		if _, ok := args[key]; !ok {
			t.Errorf("args tidak memiliki key wajib %q", key)
		}
	}

	if args["=chain"] != "input" {
		t.Errorf("args[=chain] = %q, want %q", args["=chain"], "input")
	}
	if args["=in-interface"] != "ispboss-vpn" {
		t.Errorf("args[=in-interface] = %q, want %q", args["=in-interface"], "ispboss-vpn")
	}
	if args["=dst-port"] != "8728,8729,161" {
		t.Errorf("args[=dst-port] = %q, want %q", args["=dst-port"], "8728,8729,161")
	}
	if args["=action"] != "accept" {
		t.Errorf("args[=action] = %q, want %q", args["=action"], "accept")
	}
}

// --- Table-driven test: semua remove commands menggunakan pola yang konsisten ---

// TestRemoveCommandsKonsisten memverifikasi semua remove command menggunakan
// pola yang sama: command path berakhiran /remove dan args berisi identifier.
func TestRemoveCommandsKonsisten(t *testing.T) {
	builder := newTestVPNBuilder()

	tests := []struct {
		nama        string
		cmdFunc     func() (string, map[string]string)
		expectedCmd string
		expectedKey string
		expectedVal string
	}{
		{
			nama:        "RemoveWireGuardInterface",
			cmdFunc:     func() (string, map[string]string) { return builder.RemoveWireGuardInterface("wg-test") },
			expectedCmd: "/interface/wireguard/remove",
			expectedKey: "=numbers",
			expectedVal: "wg-test",
		},
		{
			nama:        "RemoveWireGuardPeer",
			cmdFunc:     func() (string, map[string]string) { return builder.RemoveWireGuardPeer("wg-test") },
			expectedCmd: "/interface/wireguard/peers/remove",
			expectedKey: "=interface",
			expectedVal: "wg-test",
		},
		{
			nama:        "RemoveL2TPClient",
			cmdFunc:     func() (string, map[string]string) { return builder.RemoveL2TPClient("l2tp-test") },
			expectedCmd: "/interface/l2tp-client/remove",
			expectedKey: "=numbers",
			expectedVal: "l2tp-test",
		},
		{
			nama:        "RemovePPTPClient",
			cmdFunc:     func() (string, map[string]string) { return builder.RemovePPTPClient("pptp-test") },
			expectedCmd: "/interface/pptp-client/remove",
			expectedKey: "=numbers",
			expectedVal: "pptp-test",
		},
		{
			nama:        "RemoveSSTPClient",
			cmdFunc:     func() (string, map[string]string) { return builder.RemoveSSTPClient("sstp-test") },
			expectedCmd: "/interface/sstp-client/remove",
			expectedKey: "=numbers",
			expectedVal: "sstp-test",
		},
		{
			nama:        "RemoveOpenVPNClient",
			cmdFunc:     func() (string, map[string]string) { return builder.RemoveOpenVPNClient("ovpn-test") },
			expectedCmd: "/interface/ovpn-client/remove",
			expectedKey: "=numbers",
			expectedVal: "ovpn-test",
		},
		{
			nama:        "RemoveIPAddressByInterface",
			cmdFunc:     func() (string, map[string]string) { return builder.RemoveIPAddressByInterface("iface-test") },
			expectedCmd: "/ip/address/remove",
			expectedKey: "=interface",
			expectedVal: "iface-test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.nama, func(t *testing.T) {
			cmd, args := tt.cmdFunc()
			if cmd != tt.expectedCmd {
				t.Errorf("command = %q, want %q", cmd, tt.expectedCmd)
			}
			if args[tt.expectedKey] != tt.expectedVal {
				t.Errorf("args[%s] = %q, want %q", tt.expectedKey, args[tt.expectedKey], tt.expectedVal)
			}
		})
	}
}

// TestPortConversionDenganFmtSprintf memverifikasi konversi port integer ke string
// menggunakan fmt.Sprintf untuk WireGuard dan OpenVPN.
func TestPortConversionDenganFmtSprintf(t *testing.T) {
	builder := newTestVPNBuilder()

	// WireGuard listen port
	wgParams := domain.WireGuardInterfaceParams{
		Name:       "wg-test",
		ListenPort: 13231,
		PrivateKey: "key",
	}
	_, wgArgs := builder.CreateWireGuardInterface(wgParams)
	if wgArgs["=listen-port"] != fmt.Sprintf("%d", 13231) {
		t.Errorf("WireGuard listen-port = %q, want %q", wgArgs["=listen-port"], "13231")
	}

	// WireGuard peer endpoint port
	peerParams := domain.WireGuardPeerParams{
		Interface:           "wg-test",
		PublicKey:           "key",
		EndpointAddress:     "vpn.ispboss.id",
		EndpointPort:        51821,
		AllowedAddress:      "10.99.0.0/16",
		PersistentKeepalive: 25,
	}
	_, peerArgs := builder.AddWireGuardPeer(peerParams)
	if peerArgs["=endpoint-port"] != "51821" {
		t.Errorf("WireGuard endpoint-port = %q, want %q", peerArgs["=endpoint-port"], "51821")
	}

	// OpenVPN port
	ovpnParams := domain.OpenVPNClientParams{
		Name:      "ovpn-test",
		ConnectTo: "vpn.ispboss.id",
		Port:      443,
		User:      "user",
		Password:  "pass",
		Mode:      "ip",
		Protocol:  "tcp",
		Auth:      "sha256",
		Cipher:    "aes-256-cbc",
		Profile:   "default",
	}
	_, ovpnArgs := builder.CreateOpenVPNClient(ovpnParams)
	if ovpnArgs["=port"] != "443" {
		t.Errorf("OpenVPN port = %q, want %q", ovpnArgs["=port"], "443")
	}
}
