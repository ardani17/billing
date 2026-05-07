package usecase

// =============================================================================
// Template strings tambahan untuk protokol PPTP, SSTP, dan OpenVPN.
// =============================================================================

// pptpTplStr adalah template .rsc untuk protokol PPTP.
var pptpTplStr = templateHeader("PPTP") + `
# ⚠️ PPTP kurang aman — gunakan WireGuard atau L2TP/IPSec jika memungkinkan

# 1. Buat PPTP client interface
/interface/pptp-client/add \
    name=ispboss-pptp \
    connect-to={{.PrimaryEndpoint}} \
    user="{{.L2TPUsername}}" \
    password="{{.L2TPPassword}}" \
    profile=default-encryption \
    comment="ISPBoss:vpn:{{.TunnelID}}"

# 2. Assign IP address
/ip/address/add \
    address={{.VPNIP}}/24 \
    interface=ispboss-pptp \
    comment="ISPBoss:vpn:{{.TunnelID}}"

# 3. Route untuk subnet VPN
/ip/route/add \
    dst-address=10.99.0.0/16 \
    gateway=ispboss-pptp \
    comment="ISPBoss:vpn-route:{{.TunnelID}}"

` + firewallTpl("ispboss-pptp")

// sstpTplStr adalah template .rsc untuk protokol SSTP.
var sstpTplStr = templateHeader("SSTP") + `
# 1. Buat SSTP client interface
/interface/sstp-client/add \
    name=ispboss-sstp \
    connect-to={{.PrimaryEndpoint}} \
    user="{{.L2TPUsername}}" \
    password="{{.L2TPPassword}}" \
    profile=default-encryption \
    verify-server-certificate=no \
    tls-version=only-1.2 \
    comment="ISPBoss:vpn:{{.TunnelID}}"

# 2. Assign IP address
/ip/address/add \
    address={{.VPNIP}}/24 \
    interface=ispboss-sstp \
    comment="ISPBoss:vpn:{{.TunnelID}}"

# 3. Route untuk subnet VPN
/ip/route/add \
    dst-address=10.99.0.0/16 \
    gateway=ispboss-sstp \
    comment="ISPBoss:vpn-route:{{.TunnelID}}"

` + firewallTpl("ispboss-sstp")

// openvpnTplStr adalah template .rsc untuk protokol OpenVPN.
var openvpnTplStr = templateHeader("OpenVPN") + `
# 1. Buat OpenVPN client interface
/interface/ovpn-client/add \
    name=ispboss-ovpn \
    connect-to={{.PrimaryEndpoint}} \
    port=1194 \
    mode=ip \
    protocol=tcp \
    user="{{.L2TPUsername}}" \
    password="{{.L2TPPassword}}" \
    auth=sha256 \
    cipher=aes-256-cbc \
    profile=default-encryption \
    comment="ISPBoss:vpn:{{.TunnelID}}"

# 2. Assign IP address
/ip/address/add \
    address={{.VPNIP}}/24 \
    interface=ispboss-ovpn \
    comment="ISPBoss:vpn:{{.TunnelID}}"

# 3. Route untuk subnet VPN
/ip/route/add \
    dst-address=10.99.0.0/16 \
    gateway=ispboss-ovpn \
    comment="ISPBoss:vpn-route:{{.TunnelID}}"

` + firewallTpl("ispboss-ovpn")
