package domain

const (
	WalledGardenMethodDNSRedirect       = "dns_redirect"
	WalledGardenMethodHTTPRedirect      = "http_redirect"
	WalledGardenMethodBlockAllWhitelist = "block_all_whitelist"
)

type WalledGardenConfig struct {
	Method              string   `json:"method"`
	WalledGardenIP      string   `json:"walled_garden_ip"`
	DNSServerIP         string   `json:"dns_server_ip"`
	IsolatedAddressList string   `json:"isolated_address_list"`
	AllowedAddressList  string   `json:"allowed_address_list"`
	AllowedDestinations []string `json:"allowed_destinations"`
}

type WalledGardenStatus struct {
	Config        WalledGardenConfig   `json:"config"`
	Rules         []RouterFirewallRule `json:"rules"`
	IsolatedCount int                  `json:"isolated_count"`
	AllowedCount  int                  `json:"allowed_count"`
	Applied       bool                 `json:"applied"`
}

type ApplyWalledGardenRequest struct {
	Method              string   `json:"method"`
	WalledGardenIP      string   `json:"walled_garden_ip"`
	DNSServerIP         string   `json:"dns_server_ip"`
	IsolatedAddressList string   `json:"isolated_address_list"`
	AllowedAddressList  string   `json:"allowed_address_list"`
	AllowedDestinations []string `json:"allowed_destinations"`
}
