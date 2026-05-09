package domain

// ResellerCreatedPayload adalah payload event reseller.created.
// Dikirim saat reseller baru berhasil dibuat oleh admin tenant.
type ResellerCreatedPayload struct {
	ResellerID string `json:"reseller_id"`
	TenantID   string `json:"tenant_id"`
	Name       string `json:"name"`
}

// ResellerStatusChangedPayload adalah payload event reseller.status_changed.
// Dikirim saat status reseller berubah (aktif/suspended/nonaktif).
type ResellerStatusChangedPayload struct {
	ResellerID string `json:"reseller_id"`
	TenantID   string `json:"tenant_id"`
	OldStatus  string `json:"old_status"`
	NewStatus  string `json:"new_status"`
}

// VoucherBatchGeneratedPayload adalah payload event voucher.batch_generated.
// Dikirim saat batch voucher berhasil di-buat oleh admin.
type VoucherBatchGeneratedPayload struct {
	TenantID    string `json:"tenant_id"`
	PackageID   string `json:"package_id"`
	Quantity    int    `json:"quantity"`
	GeneratedBy string `json:"generated_by"`
}

// VoucherPurchasedPayload adalah payload event voucher.purchased.
// Dikirim saat reseller berhasil membeli voucher.
type VoucherPurchasedPayload struct {
	TenantID   string `json:"tenant_id"`
	ResellerID string `json:"reseller_id"`
	PackageID  string `json:"package_id"`
	Quantity   int    `json:"quantity"`
	TotalCost  int64  `json:"total_cost"`
}

// VoucherActivatedPayload adalah payload event voucher.activated.
// Dipakai network-service untuk membuat user Hotspot di RouterOS secara async.
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
