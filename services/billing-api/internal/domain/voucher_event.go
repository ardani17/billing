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
// Dikirim saat batch voucher berhasil di-generate oleh admin.
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
