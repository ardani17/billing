// Paket adapter menyediakan implementasi VPNCommandBuilder untuk RouterOS.
// File ini mengimplementasikan method WireGuard dan L2TP dari domain.VPNCommandBuilder.
package adapter

import (
	"fmt"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// vpnCommandBuilder mengimplementasikan domain.VPNCommandBuilder.
// Membangun perintah RouterOS API (command path + args map) untuk operasi VPN.
type vpnCommandBuilder struct{}

// --- WireGuard Commands (RouterOS v7+ only) ---

// CreateWireGuardInterface membangun perintah /interface/wireguard/add.
func (b *vpnCommandBuilder) CreateWireGuardInterface(params domain.WireGuardInterfaceParams) (string, map[string]string) {
	args := map[string]string{
		"=name":        params.Name,
		"=listen-port": fmt.Sprintf("%d", params.ListenPort),
		"=private-key": params.PrivateKey,
	}
	return "/interface/wireguard/add", args
}

// AddWireGuardPeer membangun perintah /interface/wireguard/peers/add.
func (b *vpnCommandBuilder) AddWireGuardPeer(params domain.WireGuardPeerParams) (string, map[string]string) {
	args := map[string]string{
		"=interface":            params.Interface,
		"=public-key":           params.PublicKey,
		"=endpoint-address":     params.EndpointAddress,
		"=endpoint-port":        fmt.Sprintf("%d", params.EndpointPort),
		"=allowed-address":      params.AllowedAddress,
		"=persistent-keepalive": fmt.Sprintf("%d", params.PersistentKeepalive),
	}
	if params.PreSharedKey != "" {
		args["=preshared-key"] = params.PreSharedKey
	}
	return "/interface/wireguard/peers/add", args
}

// RemoveWireGuardInterface membangun perintah /interface/wireguard/remove.
func (b *vpnCommandBuilder) RemoveWireGuardInterface(name string) (string, map[string]string) {
	args := map[string]string{
		"=numbers": name,
	}
	return "/interface/wireguard/remove", args
}

// RemoveWireGuardPeer membangun perintah /interface/wireguard/peers/remove.
func (b *vpnCommandBuilder) RemoveWireGuardPeer(interfaceName string) (string, map[string]string) {
	args := map[string]string{
		"=interface": interfaceName,
	}
	return "/interface/wireguard/peers/remove", args
}

// --- L2TP Commands ---

// CreateL2TPClient membangun perintah /interface/l2tp-client/add.
func (b *vpnCommandBuilder) CreateL2TPClient(params domain.L2TPClientParams) (string, map[string]string) {
	args := map[string]string{
		"=name":         params.Name,
		"=connect-to":   params.ConnectTo,
		"=user":         params.User,
		"=password":     params.Password,
		"=use-ipsec":    params.UseIPSec,
		"=ipsec-secret": params.IPSecSecret,
		"=profile":      params.Profile,
	}
	if params.AllowFastPath != "" {
		args["=allow-fast-path"] = params.AllowFastPath
	}
	return "/interface/l2tp-client/add", args
}

// RemoveL2TPClient membangun perintah /interface/l2tp-client/remove.
func (b *vpnCommandBuilder) RemoveL2TPClient(name string) (string, map[string]string) {
	args := map[string]string{
		"=numbers": name,
	}
	return "/interface/l2tp-client/remove", args
}

// CreateIPSecProfile membangun perintah /ip/ipsec/profile/add.
func (b *vpnCommandBuilder) CreateIPSecProfile(params domain.IPSecProfileParams) (string, map[string]string) {
	args := map[string]string{
		"=name":           params.Name,
		"=hash-algorithm": params.HashAlgorithm,
		"=enc-algorithm":  params.EncAlgorithm,
		"=dh-group":       params.DHGroup,
		"=lifetime":       params.Lifetime,
		"=proposal-check": params.ProposalCheck,
	}
	return "/ip/ipsec/profile/add", args
}

// CreateIPSecProposal membangun perintah /ip/ipsec/proposal/add.
func (b *vpnCommandBuilder) CreateIPSecProposal(params domain.IPSecProposalParams) (string, map[string]string) {
	args := map[string]string{
		"=name":            params.Name,
		"=auth-algorithms": params.AuthAlgorithm,
		"=enc-algorithms":  params.EncAlgorithm,
		"=lifetime":        params.Lifetime,
		"=pfs-group":       params.PFSGroup,
	}
	return "/ip/ipsec/proposal/add", args
}
