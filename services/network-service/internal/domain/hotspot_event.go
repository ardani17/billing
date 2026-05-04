package domain

// VoucherActivatedPayload adalah payload event voucher.activated dari billing-api.
type VoucherActivatedPayload struct {
	TenantID           string `json:"tenant_id"`
	VoucherID          string `json:"voucher_id"`
	Code               string `json:"code"`
	PackageID          string `json:"package_id"`
	PackageName        string `json:"package_name,omitempty"`
	RouterID           string `json:"router_id,omitempty"`
	HotspotProfileName string `json:"hotspot_profile_name,omitempty"`
	LimitUptime        string `json:"limit_uptime,omitempty"`
	MACAddress         string `json:"mac_address,omitempty"`
}
