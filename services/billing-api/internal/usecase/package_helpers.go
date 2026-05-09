// package_helpers.go berisi fungsi standalone helper untuk usecase paket.
// Termasuk: gabungkan perbarui, hitung perubahan, dan perbandingan pointer.
package usecase

import "github.com/ispboss/ispboss/services/billing-api/internal/domain"

// applyPackageUpdates menerapkan perubahan dari UpdatePackageRequest ke Package.
// Hanya field non-zero/non-nil yang diperbarui.
func applyPackageUpdates(existing *domain.Package, req domain.UpdatePackageRequest) *domain.Package {
	pkg := *existing // copy
	if req.Name != "" {
		pkg.Name = req.Name
	}
	if req.Description != "" {
		pkg.Description = req.Description
	}
	if req.DownloadMbps != nil {
		pkg.DownloadMbps = *req.DownloadMbps
	}
	if req.UploadMbps != nil {
		pkg.UploadMbps = *req.UploadMbps
	}
	if req.BandwidthType != "" {
		pkg.BandwidthType = req.BandwidthType
	}
	if req.BurstDownloadMbps != nil {
		pkg.BurstDownloadMbps = req.BurstDownloadMbps
	}
	if req.BurstUploadMbps != nil {
		pkg.BurstUploadMbps = req.BurstUploadMbps
	}
	if req.BurstThresholdMbps != nil {
		pkg.BurstThresholdMbps = req.BurstThresholdMbps
	}
	if req.BurstTimeSeconds != nil {
		pkg.BurstTimeSeconds = req.BurstTimeSeconds
	}
	if req.QuotaType != "" {
		pkg.QuotaType = domain.QuotaType(req.QuotaType)
	}
	if req.QuotaMB != nil {
		pkg.QuotaMB = req.QuotaMB
	}
	if req.QuotaAction != "" {
		pkg.QuotaAction = req.QuotaAction
	}
	if req.ThrottleMbps != nil {
		pkg.ThrottleMbps = req.ThrottleMbps
	}
	if req.MonthlyPrice != nil {
		pkg.MonthlyPrice = req.MonthlyPrice
	}
	if req.InstallationFee != nil {
		pkg.InstallationFee = *req.InstallationFee
	}
	if req.SellPrice != nil {
		pkg.SellPrice = req.SellPrice
	}
	if req.ResellerPrice != nil {
		pkg.ResellerPrice = req.ResellerPrice
	}
	if req.DurationValue != nil {
		pkg.DurationValue = req.DurationValue
	}
	if req.DurationUnit != "" {
		pkg.DurationUnit = req.DurationUnit
	}
	if req.SharedUsers != nil {
		pkg.SharedUsers = *req.SharedUsers
	}
	if req.MikrotikProfileName != "" {
		pkg.MikrotikProfileName = req.MikrotikProfileName
	}
	if req.AddressPool != "" {
		pkg.AddressPool = req.AddressPool
	}
	if req.ParentQueue != "" {
		pkg.ParentQueue = req.ParentQueue
	}
	if req.HotspotProfileName != "" {
		pkg.HotspotProfileName = req.HotspotProfileName
	}
	return &pkg
}

// computePackageChanges menghitung field yang berubah antara old dan updated package.
func computePackageChanges(old, updated *domain.Package) map[string]interface{} {
	changes := make(map[string]interface{})
	if old.Name != updated.Name {
		changes["name"] = map[string]interface{}{"old": old.Name, "new": updated.Name}
	}
	if old.Description != updated.Description {
		changes["description"] = map[string]interface{}{"old": old.Description, "new": updated.Description}
	}
	if old.DownloadMbps != updated.DownloadMbps {
		changes["download_mbps"] = map[string]interface{}{"old": old.DownloadMbps, "new": updated.DownloadMbps}
	}
	if old.UploadMbps != updated.UploadMbps {
		changes["upload_mbps"] = map[string]interface{}{"old": old.UploadMbps, "new": updated.UploadMbps}
	}
	if old.BandwidthType != updated.BandwidthType {
		changes["bandwidth_type"] = map[string]interface{}{"old": old.BandwidthType, "new": updated.BandwidthType}
	}
	if !ptrIntEqual(old.MonthlyPrice, updated.MonthlyPrice) {
		changes["monthly_price"] = map[string]interface{}{"old": old.MonthlyPrice, "new": updated.MonthlyPrice}
	}
	if old.InstallationFee != updated.InstallationFee {
		changes["installation_fee"] = map[string]interface{}{"old": old.InstallationFee, "new": updated.InstallationFee}
	}
	if !ptrIntEqual(old.SellPrice, updated.SellPrice) {
		changes["sell_price"] = map[string]interface{}{"old": old.SellPrice, "new": updated.SellPrice}
	}
	if !ptrIntEqual(old.ResellerPrice, updated.ResellerPrice) {
		changes["reseller_price"] = map[string]interface{}{"old": old.ResellerPrice, "new": updated.ResellerPrice}
	}
	return changes
}

// ptrIntEqual membandingkan dua pointer int64 (nil-safe).
func ptrIntEqual(a, b *int64) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}
