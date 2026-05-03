package domain

// PackagePriceChangedPayload adalah payload event package.price_changed.
// Dikirim saat monthly_price (PPPoE) atau sell_price (Voucher) berubah.
type PackagePriceChangedPayload struct {
	TenantID    string `json:"tenant_id"`
	PackageID   string `json:"package_id"`
	PackageName string `json:"package_name"`
	PackageType string `json:"package_type"`
	OldPrice    int64  `json:"old_price"`
	NewPrice    int64  `json:"new_price"`
}
