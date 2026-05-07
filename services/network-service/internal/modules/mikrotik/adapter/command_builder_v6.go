// Paket adapter menyediakan implementasi CommandBuilder untuk RouterOS v6.
// File ini mengimplementasikan semua 14 method dari domain.CommandBuilder
// menggunakan API path dan parameter yang kompatibel dengan RouterOS v6.
package adapter

import (
	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// commandBuilderV6 mengimplementasikan domain.CommandBuilder untuk RouterOS v6.
type commandBuilderV6 struct{}

// CreateSecret membangun perintah /ppp/secret/add untuk RouterOS v6.
func (b *commandBuilderV6) CreateSecret(params domain.PPPoESecretParams) (string, map[string]string) {
	args := map[string]string{
		"=name":     params.Name,
		"=password": params.Password,
		"=service":  params.Service,
		"=profile":  params.Profile,
		"=comment":  params.Comment,
	}
	if params.RemoteAddress != "" {
		args["=remote-address"] = params.RemoteAddress
	}
	return "/ppp/secret/add", args
}

// SetSecret membangun perintah /ppp/secret/atur untuk RouterOS v6.
// Parameter =numbers= digunakan untuk mengidentifikasi secret berdasarkan username.
func (b *commandBuilderV6) SetSecret(username string, params map[string]string) (string, map[string]string) {
	args := map[string]string{
		"=numbers": username,
	}
	for k, v := range params {
		args[k] = v
	}
	return "/ppp/secret/set", args
}

// RemoveSecret membangun perintah /ppp/secret/remove untuk RouterOS v6.
func (b *commandBuilderV6) RemoveSecret(username string) (string, map[string]string) {
	args := map[string]string{
		"=numbers": username,
	}
	return "/ppp/secret/remove", args
}

// PrintSecrets membangun perintah /ppp/secret/print untuk RouterOS v6.
func (b *commandBuilderV6) PrintSecrets() (string, map[string]string) {
	return "/ppp/secret/print", map[string]string{}
}

// RemoveActiveSession membangun perintah /ppp/active/remove untuk RouterOS v6.
func (b *commandBuilderV6) RemoveActiveSession(sessionID string) (string, map[string]string) {
	args := map[string]string{
		"=numbers": sessionID,
	}
	return "/ppp/active/remove", args
}

// PrintActiveSessions membangun perintah /ppp/active/print untuk RouterOS v6.
func (b *commandBuilderV6) PrintActiveSessions() (string, map[string]string) {
	return "/ppp/active/print", map[string]string{}
}

// CreateProfile membangun perintah /ppp/profile/add untuk RouterOS v6.
// Burst parameters hanya ditambahkan jika nilainya tidak kosong.
func (b *commandBuilderV6) CreateProfile(params domain.PPPoEProfileParams) (string, map[string]string) {
	args := map[string]string{
		"=name":       params.Name,
		"=rate-limit": params.RateLimit,
		"=only-one":   params.OnlyOne,
	}
	if params.LocalAddress != "" {
		args["=local-address"] = params.LocalAddress
	}
	if params.RemoteAddress != "" {
		args["=remote-address"] = params.RemoteAddress
	}
	if params.BurstLimit != "" {
		args["=burst-limit"] = params.BurstLimit
	}
	if params.BurstThreshold != "" {
		args["=burst-threshold"] = params.BurstThreshold
	}
	if params.BurstTime != "" {
		args["=burst-time"] = params.BurstTime
	}
	return "/ppp/profile/add", args
}

// SetProfile membangun perintah /ppp/profile/atur untuk RouterOS v6.
// Parameter =numbers= digunakan untuk mengidentifikasi profile berdasarkan nama.
func (b *commandBuilderV6) SetProfile(name string, params map[string]string) (string, map[string]string) {
	args := map[string]string{
		"=numbers": name,
	}
	for k, v := range params {
		args[k] = v
	}
	return "/ppp/profile/set", args
}

// CreateNATRule membangun perintah /ip/firewall/nat/add untuk RouterOS v6.
// Parameter to-ports hanya ditambahkan jika nilainya tidak kosong.
func (b *commandBuilderV6) CreateNATRule(params domain.NATRuleParams) (string, map[string]string) {
	args := map[string]string{
		"=chain":        params.Chain,
		"=src-address":  params.SrcAddress,
		"=protocol":     params.Protocol,
		"=dst-port":     params.DstPort,
		"=action":       params.Action,
		"=to-addresses": params.ToAddress,
		"=comment":      params.Comment,
	}
	if params.ToPort != "" {
		args["=to-ports"] = params.ToPort
	}
	return "/ip/firewall/nat/add", args
}

// RemoveNATRuleByComment membangun perintah untuk menghapus NAT rule berdasarkan comment.
// Caller (PPPoE Manager) bertanggung jawab untuk melakukan find-then-remove.
func (b *commandBuilderV6) RemoveNATRuleByComment(comment string) (string, map[string]string) {
	args := map[string]string{
		"=comment": comment,
	}
	return "/ip/firewall/nat/remove", args
}

// CreateSimpleQueue membangun perintah /queue/simple/add untuk RouterOS v6.
// Burst parameters hanya ditambahkan jika nilainya tidak kosong.
func (b *commandBuilderV6) CreateSimpleQueue(params domain.SimpleQueueParams) (string, map[string]string) {
	args := map[string]string{
		"=name":      params.Name,
		"=target":    params.Target,
		"=max-limit": params.MaxLimit,
		"=comment":   params.Comment,
	}
	if params.BurstLimit != "" {
		args["=burst-limit"] = params.BurstLimit
	}
	if params.BurstThreshold != "" {
		args["=burst-threshold"] = params.BurstThreshold
	}
	if params.BurstTime != "" {
		args["=burst-time"] = params.BurstTime
	}
	return "/queue/simple/add", args
}

// SetSimpleQueue membangun perintah /queue/simple/atur untuk RouterOS v6.
// Parameter =numbers= digunakan untuk mengidentifikasi queue berdasarkan nama.
func (b *commandBuilderV6) SetSimpleQueue(name string, params map[string]string) (string, map[string]string) {
	args := map[string]string{
		"=numbers": name,
	}
	for k, v := range params {
		args[k] = v
	}
	return "/queue/simple/set", args
}

// RemoveSimpleQueue membangun perintah /queue/simple/remove untuk RouterOS v6.
func (b *commandBuilderV6) RemoveSimpleQueue(name string) (string, map[string]string) {
	args := map[string]string{
		"=numbers": name,
	}
	return "/queue/simple/remove", args
}

// ResetSimpleQueueCounters membangun perintah /queue/simple/reset-counters untuk RouterOS v6.
// Digunakan saat buka isolir untuk mereset traffic counter.
func (b *commandBuilderV6) ResetSimpleQueueCounters(name string) (string, map[string]string) {
	args := map[string]string{
		"=numbers": name,
	}
	return "/queue/simple/reset-counters", args
}

func (b *commandBuilderV6) AddDHCPLease(params map[string]string) (string, map[string]string) {
	return "/ip/dhcp-server/lease/add", cloneCommandParams(params)
}

func (b *commandBuilderV6) SetDHCPLease(params map[string]string) (string, map[string]string) {
	return "/ip/dhcp-server/lease/set", cloneCommandParams(params)
}

func (b *commandBuilderV6) RemoveDHCPLease(leaseID string) (string, map[string]string) {
	return "/ip/dhcp-server/lease/remove", map[string]string{"=numbers": leaseID}
}

func (b *commandBuilderV6) AddAddressList(params map[string]string) (string, map[string]string) {
	return "/ip/firewall/address-list/add", cloneCommandParams(params)
}

func (b *commandBuilderV6) SetAddressList(params map[string]string) (string, map[string]string) {
	return "/ip/firewall/address-list/set", cloneCommandParams(params)
}

func (b *commandBuilderV6) RemoveAddressList(entryID string) (string, map[string]string) {
	return "/ip/firewall/address-list/remove", map[string]string{"=numbers": entryID}
}

func (b *commandBuilderV6) AddHotspotUser(params map[string]string) (string, map[string]string) {
	return "/ip/hotspot/user/add", cloneCommandParams(params)
}

func (b *commandBuilderV6) SetHotspotUser(params map[string]string) (string, map[string]string) {
	return "/ip/hotspot/user/set", cloneCommandParams(params)
}

func (b *commandBuilderV6) RemoveHotspotUser(userID string) (string, map[string]string) {
	return "/ip/hotspot/user/remove", map[string]string{"=numbers": userID}
}

func (b *commandBuilderV6) AddFirewallRule(kind string, params map[string]string) (string, map[string]string) {
	return "/ip/firewall/" + kind + "/add", cloneCommandParams(params)
}

func (b *commandBuilderV6) SetFirewallRule(kind string, params map[string]string) (string, map[string]string) {
	return "/ip/firewall/" + kind + "/set", cloneCommandParams(params)
}

func (b *commandBuilderV6) RemoveFirewallRule(kind, ruleID string) (string, map[string]string) {
	return "/ip/firewall/" + kind + "/remove", map[string]string{"=numbers": ruleID}
}

func cloneCommandParams(params map[string]string) map[string]string {
	cloned := make(map[string]string, len(params))
	for key, value := range params {
		cloned[key] = value
	}
	return cloned
}
