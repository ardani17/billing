package usecase

// =============================================================================
// Template strings untuk RouterOS script (.rsc) per protokol VPN.
// Setiap template menghasilkan script lengkap dengan komentar dalam bahasa Indonesia.
// =============================================================================

// firewallTpl menghasilkan blok firewall rules untuk interface VPN tertentu.
// Membatasi traffic hanya ke port RouterOS API (8728, 8729) dan SNMP (161).
func firewallTpl(iface string) string {
	return `# Firewall — hanya izinkan traffic API dan SNMP via VPN
/ip/firewall/filter/add \
    chain=input \
    in-interface=` + iface + ` \
    protocol=tcp \
    dst-port=8728,8729 \
    action=accept \
    comment="ISPBoss:vpn-allow-api:{{.TunnelID}}"

/ip/firewall/filter/add \
    chain=input \
    in-interface=` + iface + ` \
    protocol=udp \
    dst-port=161 \
    action=accept \
    comment="ISPBoss:vpn-allow-snmp:{{.TunnelID}}"

/ip/firewall/filter/add \
    chain=input \
    in-interface=` + iface + ` \
    action=drop \
    comment="ISPBoss:vpn-drop-other:{{.TunnelID}}"
`
}

// templateHeader menghasilkan header komentar untuk script .rsc.
func templateHeader(protocol string) string {
	return `# ISPBoss VPN Configuration — ` + protocol + `
# Tunnel ID: {{.TunnelID}}
# Tunnel: {{.TunnelName}}
# Generated: {{.GeneratedAt}}
# JANGAN edit manual — gunakan ISPBoss dashboard untuk perubahan
`
}

// wireguardTplStr adalah template .rsc untuk protokol WireGuard.
var wireguardTplStr = templateHeader("WireGuard") + `
# 1. Buat interface WireGuard
/interface/wireguard/add \
    name=ispboss-vpn \
    listen-port={{.ListenPort}} \
    private-key="{{.ClientPrivateKey}}"

# 2. Tambah peer ISPBoss VPN Server (Primary)
/interface/wireguard/peers/add \
    interface=ispboss-vpn \
    public-key="{{.ServerPublicKey}}" \
{{- if .PreSharedKey}}
    preshared-key="{{.PreSharedKey}}" \
{{- end}}
    endpoint-address={{.PrimaryEndpoint}} \
    endpoint-port={{.ListenPort}} \
    allowed-address={{.AllowedAddresses}} \
    persistent-keepalive={{.PersistentKeepalive}}

# 3. Tambah peer ISPBoss VPN Server (Secondary/Failover)
/interface/wireguard/peers/add \
    interface=ispboss-vpn \
    public-key="{{.SecondaryServerPublicKey}}" \
    endpoint-address={{.SecondaryEndpoint}} \
    endpoint-port={{.ListenPort}} \
    allowed-address={{.AllowedAddresses}} \
    persistent-keepalive={{.PersistentKeepalive}}

# 4. Assign IP address ke interface VPN
/ip/address/add \
    address={{.VPNIP}}/24 \
    interface=ispboss-vpn \
    comment="ISPBoss:vpn:{{.TunnelID}}"

` + firewallTpl("ispboss-vpn")

// l2tpTplStr adalah template .rsc untuk protokol L2TP/IPSec.
var l2tpTplStr = templateHeader("L2TP/IPSec") + `
# 1. Buat IPSec profile
/ip/ipsec/profile/add \
    name=ispboss-ipsec \
    hash-algorithm=sha256 \
    enc-algorithm=aes-256 \
    dh-group=modp2048 \
    lifetime=1d \
    proposal-check=obey

# 2. Buat IPSec proposal
/ip/ipsec/proposal/add \
    name=ispboss-proposal \
    auth-algorithms=sha256 \
    enc-algorithms=aes-256-cbc \
    lifetime=30m \
    pfs-group=modp2048

# 3. Buat L2TP client interface
/interface/l2tp-client/add \
    name=ispboss-l2tp \
    connect-to={{.PrimaryEndpoint}} \
    user="{{.L2TPUsername}}" \
    password="{{.L2TPPassword}}" \
    use-ipsec=yes \
    ipsec-secret="{{.IPSecPSK}}" \
    profile=default-encryption \
    comment="ISPBoss:vpn:{{.TunnelID}}"

# 4. Assign IP address
/ip/address/add \
    address={{.VPNIP}}/24 \
    interface=ispboss-l2tp \
    comment="ISPBoss:vpn:{{.TunnelID}}"

# 5. Route untuk subnet VPN
/ip/route/add \
    dst-address=10.99.0.0/16 \
    gateway=ispboss-l2tp \
    comment="ISPBoss:vpn-route:{{.TunnelID}}"

# 6. Failover script — switch ke secondary jika primary unreachable
/system/scheduler/add \
    name=ispboss-vpn-failover \
    interval=30s \
    on-event="/tool fetch url=\"https://{{.PrimaryEndpoint}}\" mode=https keep-result=no\r\n\
    :if (\$error = true) do={\r\n\
        /interface l2tp-client set ispboss-l2tp connect-to={{.SecondaryEndpoint}}\r\n\
    } else={\r\n\
        /interface l2tp-client set ispboss-l2tp connect-to={{.PrimaryEndpoint}}\r\n\
    }" \
    comment="ISPBoss:vpn-failover:{{.TunnelID}}"

` + firewallTpl("ispboss-l2tp")
