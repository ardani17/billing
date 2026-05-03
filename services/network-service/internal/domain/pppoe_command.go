package domain

// =============================================================================
// RouterOS Command Parameter Structs — parameter untuk operasi RouterOS API
// =============================================================================

// PPPoESecretParams berisi parameter untuk membuat PPPoE secret di router.
type PPPoESecretParams struct {
	Name          string
	Password      string
	Service       string
	Profile       string
	RemoteAddress string
	Comment       string
}

// PPPoEProfileParams berisi parameter untuk membuat PPPoE profile di router.
type PPPoEProfileParams struct {
	Name           string
	LocalAddress   string
	RemoteAddress  string // address pool name
	RateLimit      string // format: "download/upload" e.g. "50M/25M"
	BurstLimit     string // format: "download/upload"
	BurstThreshold string // format: "download/upload"
	BurstTime      string // format: "download/upload"
	OnlyOne        string // "yes" atau "no"
}

// NATRuleParams berisi parameter untuk membuat NAT rule di router.
type NATRuleParams struct {
	Chain      string // "dstnat"
	SrcAddress string // user remote IP
	Protocol   string // "tcp" atau "udp"
	DstPort    string // "80" atau "53"
	Action     string // "dst-nat"
	ToAddress  string // walled garden IP atau DNS server IP
	ToPort     string // optional
	Comment    string // "ISPBoss:isolir:{customer_id}" atau "ISPBoss:dns-redirect:{customer_id}"
}

// SimpleQueueParams berisi parameter untuk membuat simple queue di router.
type SimpleQueueParams struct {
	Name           string
	Target         string // user remote IP
	MaxLimit       string // format: "download/upload"
	BurstLimit     string
	BurstThreshold string
	BurstTime      string
	Comment        string
}
