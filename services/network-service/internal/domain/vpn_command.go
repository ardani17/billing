package domain

// =============================================================================
// RouterOS VPN Command Parameter Structs - parameter untuk operasi VPN via
// RouterOS API. Setiap struct merepresentasikan satu perintah RouterOS.
// =============================================================================

// WireGuardInterfaceParams berisi parameter untuk /interface/wireguard/add.
// Digunakan saat membuat interface WireGuard baru di router (RouterOS v7+).
type WireGuardInterfaceParams struct {
	Name       string // nama interface, e.g. "ispboss-vpn"
	ListenPort int    // port listen, e.g. 51820
	PrivateKey string // client private key (WireGuard)
}

// WireGuardPeerParams berisi parameter untuk /interface/wireguard/peers/add.
// Digunakan saat menambahkan peer ISPBoss VPN server ke interface WireGuard.
type WireGuardPeerParams struct {
	Interface           string // nama interface WireGuard
	PublicKey           string // server public key
	PreSharedKey        string // PSK opsional untuk keamanan ekstra
	EndpointAddress     string // alamat server, e.g. "vpn.ispboss.id"
	EndpointPort        int    // port server, e.g. 51820
	AllowedAddress      string // allowed addresses, e.g. "10.99.0.0/16"
	PersistentKeepalive int    // interval keepalive dalam detik, e.g. 25
}

// L2TPClientParams berisi parameter untuk /interface/l2tp-client/add.
// Digunakan saat membuat koneksi L2TP client ke VPN server ISPBoss.
type L2TPClientParams struct {
	Name          string // nama interface, e.g. "ispboss-l2tp"
	ConnectTo     string // alamat server VPN
	User          string // L2TP username
	Password      string // L2TP password
	UseIPSec      string // "yes" atau "no"
	IPSecSecret   string // IPSec pre-shared key
	AllowFastPath string // "yes" (v7) atau kosong (v6)
	Profile       string // profil enkripsi, e.g. "bawaan-encryption"
}

// IPSecProfileParams berisi parameter untuk /ip/ipsec/profile/add.
// Digunakan saat membuat profil IPSec untuk koneksi L2TP/IPSec.
type IPSecProfileParams struct {
	Name          string // nama profil, e.g. "ispboss-ipsec"
	HashAlgorithm string // algoritma hash, e.g. "sha256"
	EncAlgorithm  string // algoritma enkripsi, e.g. "aes-256"
	DHGroup       string // Diffie-Hellman group, e.g. "modp2048"
	Lifetime      string // masa berlaku, e.g. "1d"
	ProposalCheck string // mode pengecekan proposal, e.g. "obey"
}

// IPSecProposalParams berisi parameter untuk /ip/ipsec/proposal/add.
// Digunakan saat membuat proposal IPSec untuk negosiasi enkripsi.
type IPSecProposalParams struct {
	Name          string // nama proposal, e.g. "ispboss-proposal"
	AuthAlgorithm string // algoritma autentikasi, e.g. "sha256"
	EncAlgorithm  string // algoritma enkripsi, e.g. "aes-256-cbc"
	Lifetime      string // masa berlaku, e.g. "30m"
	PFSGroup      string // Perfect Forward Secrecy group, e.g. "modp2048"
}

// PPTPClientParams berisi parameter untuk /interface/pptp-client/add.
// Digunakan saat membuat koneksi PPTP client (legacy, kurang aman).
type PPTPClientParams struct {
	Name      string // nama interface, e.g. "ispboss-pptp"
	ConnectTo string // alamat server VPN
	User      string // PPTP username
	Password  string // PPTP password
	Profile   string // profil enkripsi, e.g. "bawaan-encryption"
}

// SSTPClientParams berisi parameter untuk /interface/sstp-client/add.
// Digunakan saat membuat koneksi SSTP client (melewati firewall/NAT ketat).
type SSTPClientParams struct {
	Name              string // nama interface, e.g. "ispboss-sstp"
	ConnectTo         string // alamat server VPN
	User              string // SSTP username
	Password          string // SSTP password
	Profile           string // profil enkripsi, e.g. "bawaan-encryption"
	CertificateVerify string // "no" (self-signed) atau "yes"
	TLSVersion        string // versi TLS, e.g. "only-1.2"
}

// OpenVPNClientParams berisi parameter untuk /interface/ovpn-client/add.
// Digunakan saat membuat koneksi OpenVPN client sebagai alternatif WireGuard.
type OpenVPNClientParams struct {
	Name        string // nama interface, e.g. "ispboss-ovpn"
	ConnectTo   string // alamat server VPN
	Port        int    // port server, e.g. 1194
	User        string // OpenVPN username
	Password    string // OpenVPN password
	Mode        string // mode operasi: "ip" atau "ethernet"
	Protocol    string // protokol transport: "tcp" atau "udp"
	Certificate string // nama sertifikat (jika menggunakan cert auth)
	Auth        string // algoritma autentikasi, e.g. "sha256"
	Cipher      string // algoritma cipher, e.g. "aes-256-cbc"
	Profile     string // profil enkripsi, e.g. "bawaan-encryption"
}

// IPAddressParams berisi parameter untuk /ip/address/add.
// Digunakan saat menambahkan IP address ke interface VPN di router.
type IPAddressParams struct {
	Address   string // alamat IP dengan prefix, e.g. "10.99.1.2/24"
	Interface string // nama interface VPN tujuan
	Comment   string // komentar identifikasi, e.g. "ISPBoss:vpn:{tunnel_id}"
}

// IPRouteParams berisi parameter untuk /ip/route/add.
// Digunakan saat menambahkan route ke subnet VPN melalui interface VPN.
type IPRouteParams struct {
	DstAddress string // alamat tujuan, e.g. "10.99.0.0/16"
	Gateway    string // gateway (nama interface VPN)
	Comment    string // komentar identifikasi, e.g. "ISPBoss:vpn-route:{tunnel_id}"
}

// FirewallFilterParams berisi parameter untuk /ip/firewall/filter/add.
// Digunakan saat menambahkan rule firewall untuk mengizinkan traffic VPN.
type FirewallFilterParams struct {
	Chain       string // chain firewall: "input" atau "forward"
	InInterface string // nama interface VPN masuk
	Protocol    string // protokol: "tcp", "udp", dll
	DstPort     string // port tujuan, e.g. "8728,8729,161"
	Action      string // aksi: "accept", "drop", dll
	Comment     string // komentar identifikasi, e.g. "ISPBoss:vpn-firewall:{tunnel_id}"
}
