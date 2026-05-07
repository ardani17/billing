// File ini berisi implementasi AutoConfigure dan helper executeVPNCommands.
// Dipisahkan dari vpn_manager_setup.go agar setiap file tetap di bawah 200 baris.
package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// AutoConfigure mengkonfigurasi VPN di router yang sudah online via RouterOS API.
func (m *vpnManager) AutoConfigure(ctx context.Context, id string) error {
	tunnel, err := m.tunnelRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("gagal ambil tunnel %s: %w", id, err)
	}

	// Pastikan tunnel punya router_id
	if tunnel.RouterID == nil {
		return domain.ErrRouterNotOnline
	}

	// Ambil router dan pastikan online
	router, err := m.routerRepo.GetByID(ctx, *tunnel.RouterID)
	if err != nil {
		return fmt.Errorf("gagal ambil router %s: %w", *tunnel.RouterID, err)
	}
	if router.Status != domain.StatusOnline {
		return domain.ErrRouterNotOnline
	}

	// Decrypt password router untuk koneksi
	password, err := m.crypto.Decrypt(router.PasswordEncrypted)
	if err != nil {
		return fmt.Errorf("gagal decrypt password router: %w", err)
	}

	cfg := domain.ConnectionConfig{
		Host:           router.Host,
		Port:           router.Port,
		Username:       router.Username,
		Password:       password,
		UseSSL:         router.UseSSL,
		ConnectTimeout: 10 * time.Second,
		CommandTimeout: 30 * time.Second,
	}
	pool := m.poolManager.GetPool(*tunnel.RouterID, cfg)

	// Bangun dan execute commands sesuai protokol
	cmdBuilder := m.cmdBuilder(router.RouterOSVersion)
	if err := m.executeVPNCommands(ctx, pool, tunnel, cmdBuilder); err != nil {
		m.logger.Error().Err(err).Str("tunnel_id", id).Msg("auto-configure vpn gagal")
		return domain.ErrAutoConfigFailed
	}

	// Perbarui status tunnel ke pending (menunggu verifikasi)
	pendingStatus := domain.TunnelStatusPending
	if err := m.tunnelRepo.UpdateStatus(ctx, id, domain.TunnelHealthUpdate{
		Status: &pendingStatus,
	}); err != nil {
		return fmt.Errorf("gagal update status tunnel: %w", err)
	}

	m.logger.Info().Str("tunnel_id", id).Str("protocol", string(tunnel.Protocol)).
		Msg("auto-configure vpn berhasil, status: pending")

	return nil
}

// executeVPNCommands membangun dan mengeksekusi perintah VPN sesuai protokol.
// Menggunakan VPNCommandBuilder untuk build commands, lalu execute via pool adapter.
func (m *vpnManager) executeVPNCommands(ctx context.Context, pool domain.ConnPool, tunnel *domain.VPNTunnel, cmdBuilder domain.VPNCommandBuilder) error {
	// Ambil adapter dari pool
	adapter, err := pool.Get(ctx, domain.PriorityMedium)
	if err != nil {
		return fmt.Errorf("gagal ambil koneksi dari pool: %w", err)
	}
	defer pool.Put(adapter)

	// Kumpulkan semua commands yang perlu dieksekusi
	commands, err := m.buildVPNCommands(tunnel, cmdBuilder)
	if err != nil {
		return err
	}

	// Eksekusi semua commands secara berurutan
	for _, cmd := range commands {
		if _, err := adapter.Execute(ctx, cmd.path, cmd.args); err != nil {
			return fmt.Errorf("gagal eksekusi %s: %w", cmd.path, err)
		}
	}

	return nil
}

// vpnCommand menyimpan satu perintah RouterOS (path + args).
type vpnCommand struct {
	path string
	args map[string]string
}

// buildVPNCommands membangun daftar perintah RouterOS sesuai protokol tunnel.
func (m *vpnManager) buildVPNCommands(tunnel *domain.VPNTunnel, cmdBuilder domain.VPNCommandBuilder) ([]vpnCommand, error) {
	var cmds []vpnCommand
	interfaceName := fmt.Sprintf("ispboss-vpn-%s", tunnel.TunnelName)

	switch tunnel.Protocol {
	case domain.ProtocolWireGuard:
		cmds = m.buildWireGuardCommands(tunnel, interfaceName, cmdBuilder)
	case domain.ProtocolL2TPIPSec:
		cmds = m.buildL2TPCommands(tunnel, interfaceName, cmdBuilder)
	case domain.ProtocolPPTP:
		cmds = m.buildPPTPCommands(tunnel, interfaceName, cmdBuilder)
	case domain.ProtocolSSTP:
		cmds = m.buildSSTPCommands(tunnel, interfaceName, cmdBuilder)
	case domain.ProtocolOpenVPN:
		cmds = m.buildOpenVPNCommands(tunnel, interfaceName, cmdBuilder)
	default:
		return nil, fmt.Errorf("%w: %s", domain.ErrInvalidVPNProtocol, tunnel.Protocol)
	}

	// Tambahkan IP address dan route untuk semua protokol
	cmds = append(cmds, m.buildCommonCommands(tunnel, interfaceName, cmdBuilder)...)

	return cmds, nil
}

// buildWireGuardCommands membangun perintah WireGuard.
func (m *vpnManager) buildWireGuardCommands(t *domain.VPNTunnel, ifName string, cmdBuilder domain.VPNCommandBuilder) []vpnCommand {
	cmd1Path, cmd1Args := cmdBuilder.CreateWireGuardInterface(domain.WireGuardInterfaceParams{
		Name: ifName, ListenPort: t.ListenPort, PrivateKey: t.ClientPrivateKeyEncrypted,
	})
	endpoint := extractHost(t.ServerEndpoint)
	cmd2Path, cmd2Args := cmdBuilder.AddWireGuardPeer(domain.WireGuardPeerParams{
		Interface: ifName, PublicKey: t.ServerPublicKey, PreSharedKey: t.PreSharedKeyEncrypted,
		EndpointAddress: endpoint, EndpointPort: t.ListenPort,
		AllowedAddress: t.AllowedAddresses, PersistentKeepalive: t.PersistentKeepalive,
	})
	return []vpnCommand{{cmd1Path, cmd1Args}, {cmd2Path, cmd2Args}}
}

// buildL2TPCommands membangun perintah L2TP/IPSec.
func (m *vpnManager) buildL2TPCommands(t *domain.VPNTunnel, ifName string, cmdBuilder domain.VPNCommandBuilder) []vpnCommand {
	endpoint := extractHost(t.ServerEndpoint)
	path, args := cmdBuilder.CreateL2TPClient(domain.L2TPClientParams{
		Name: ifName, ConnectTo: endpoint, User: t.L2TPUsername,
		Password: t.L2TPPasswordEncrypted, UseIPSec: "yes",
		IPSecSecret: t.PreSharedKeyEncrypted, Profile: "default-encryption",
	})
	return []vpnCommand{{path, args}}
}

// buildPPTPCommands membangun perintah PPTP.
func (m *vpnManager) buildPPTPCommands(t *domain.VPNTunnel, ifName string, cmdBuilder domain.VPNCommandBuilder) []vpnCommand {
	endpoint := extractHost(t.ServerEndpoint)
	path, args := cmdBuilder.CreatePPTPClient(domain.PPTPClientParams{
		Name: ifName, ConnectTo: endpoint, User: t.L2TPUsername,
		Password: t.L2TPPasswordEncrypted, Profile: "default-encryption",
	})
	return []vpnCommand{{path, args}}
}

// buildSSTPCommands membangun perintah SSTP.
func (m *vpnManager) buildSSTPCommands(t *domain.VPNTunnel, ifName string, cmdBuilder domain.VPNCommandBuilder) []vpnCommand {
	endpoint := extractHost(t.ServerEndpoint)
	path, args := cmdBuilder.CreateSSTPClient(domain.SSTPClientParams{
		Name: ifName, ConnectTo: endpoint, User: t.L2TPUsername,
		Password: t.L2TPPasswordEncrypted, Profile: "default-encryption",
		CertificateVerify: "no", TLSVersion: "only-1.2",
	})
	return []vpnCommand{{path, args}}
}

// buildOpenVPNCommands membangun perintah OpenVPN.
func (m *vpnManager) buildOpenVPNCommands(t *domain.VPNTunnel, ifName string, cmdBuilder domain.VPNCommandBuilder) []vpnCommand {
	endpoint := extractHost(t.ServerEndpoint)
	path, args := cmdBuilder.CreateOpenVPNClient(domain.OpenVPNClientParams{
		Name: ifName, ConnectTo: endpoint, Port: t.ListenPort,
		User: t.L2TPUsername, Password: t.L2TPPasswordEncrypted,
		Mode: "ip", Protocol: "tcp", Auth: "sha256",
		Cipher: "aes-256-cbc", Profile: "default-encryption",
	})
	return []vpnCommand{{path, args}}
}

// buildCommonCommands membangun perintah IP address dan route.
func (m *vpnManager) buildCommonCommands(t *domain.VPNTunnel, ifName string, cmdBuilder domain.VPNCommandBuilder) []vpnCommand {
	ipPath, ipArgs := cmdBuilder.AddIPAddress(domain.IPAddressParams{
		Address: t.VPNIP + "/24", Interface: ifName,
		Comment: fmt.Sprintf("ISPBoss:vpn:%s", t.ID),
	})
	routePath, routeArgs := cmdBuilder.AddIPRoute(domain.IPRouteParams{
		DstAddress: t.AllowedAddresses, Gateway: ifName,
		Comment: fmt.Sprintf("ISPBoss:vpn-route:%s", t.ID),
	})
	return []vpnCommand{{ipPath, ipArgs}, {routePath, routeArgs}}
}
