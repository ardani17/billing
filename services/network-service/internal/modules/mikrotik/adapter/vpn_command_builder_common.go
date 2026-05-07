// Paket adapter menyediakan implementasi VPNCommandBuilder untuk RouterOS.
// File ini mengimplementasikan method PPTP, SSTP, OpenVPN, dan Common commands.
package adapter

import (
	"fmt"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// --- PPTP Commands ---

// CreatePPTPClient membangun perintah /interface/pptp-client/add.
func (b *vpnCommandBuilder) CreatePPTPClient(params domain.PPTPClientParams) (string, map[string]string) {
	args := map[string]string{
		"=name":       params.Name,
		"=connect-to": params.ConnectTo,
		"=user":       params.User,
		"=password":   params.Password,
		"=profile":    params.Profile,
	}
	return "/interface/pptp-client/add", args
}

// RemovePPTPClient membangun perintah /interface/pptp-client/remove.
func (b *vpnCommandBuilder) RemovePPTPClient(name string) (string, map[string]string) {
	args := map[string]string{
		"=numbers": name,
	}
	return "/interface/pptp-client/remove", args
}

// --- SSTP Commands ---

// CreateSSTPClient membangun perintah /interface/sstp-client/add.
func (b *vpnCommandBuilder) CreateSSTPClient(params domain.SSTPClientParams) (string, map[string]string) {
	args := map[string]string{
		"=name":               params.Name,
		"=connect-to":         params.ConnectTo,
		"=user":               params.User,
		"=password":           params.Password,
		"=profile":            params.Profile,
		"=certificate-verify": params.CertificateVerify,
		"=tls-version":        params.TLSVersion,
	}
	return "/interface/sstp-client/add", args
}

// RemoveSSTPClient membangun perintah /interface/sstp-client/remove.
func (b *vpnCommandBuilder) RemoveSSTPClient(name string) (string, map[string]string) {
	args := map[string]string{
		"=numbers": name,
	}
	return "/interface/sstp-client/remove", args
}

// --- OpenVPN Commands ---

// CreateOpenVPNClient membangun perintah /interface/ovpn-client/add.
func (b *vpnCommandBuilder) CreateOpenVPNClient(params domain.OpenVPNClientParams) (string, map[string]string) {
	args := map[string]string{
		"=name":       params.Name,
		"=connect-to": params.ConnectTo,
		"=port":       fmt.Sprintf("%d", params.Port),
		"=user":       params.User,
		"=password":   params.Password,
		"=mode":       params.Mode,
		"=protocol":   params.Protocol,
		"=auth":       params.Auth,
		"=cipher":     params.Cipher,
		"=profile":    params.Profile,
	}
	if params.Certificate != "" {
		args["=certificate"] = params.Certificate
	}
	return "/interface/ovpn-client/add", args
}

// RemoveOpenVPNClient membangun perintah /interface/ovpn-client/remove.
func (b *vpnCommandBuilder) RemoveOpenVPNClient(name string) (string, map[string]string) {
	args := map[string]string{
		"=numbers": name,
	}
	return "/interface/ovpn-client/remove", args
}

// --- Common Commands ---

// AddIPAddress membangun perintah /ip/address/add.
func (b *vpnCommandBuilder) AddIPAddress(params domain.IPAddressParams) (string, map[string]string) {
	args := map[string]string{
		"=address":   params.Address,
		"=interface": params.Interface,
		"=comment":   params.Comment,
	}
	return "/ip/address/add", args
}

// RemoveIPAddressByInterface membangun perintah /ip/address/remove by interface.
// Caller bertanggung jawab untuk melakukan find-then-remove berdasarkan interface.
func (b *vpnCommandBuilder) RemoveIPAddressByInterface(interfaceName string) (string, map[string]string) {
	args := map[string]string{
		"=interface": interfaceName,
	}
	return "/ip/address/remove", args
}

// AddIPRoute membangun perintah /ip/route/add.
func (b *vpnCommandBuilder) AddIPRoute(params domain.IPRouteParams) (string, map[string]string) {
	args := map[string]string{
		"=dst-address": params.DstAddress,
		"=gateway":     params.Gateway,
		"=comment":     params.Comment,
	}
	return "/ip/route/add", args
}

// AddFirewallFilter membangun perintah /ip/firewall/filter/add.
func (b *vpnCommandBuilder) AddFirewallFilter(params domain.FirewallFilterParams) (string, map[string]string) {
	args := map[string]string{
		"=chain":        params.Chain,
		"=in-interface": params.InInterface,
		"=protocol":     params.Protocol,
		"=dst-port":     params.DstPort,
		"=action":       params.Action,
		"=comment":      params.Comment,
	}
	return "/ip/firewall/filter/add", args
}
