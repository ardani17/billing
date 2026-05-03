package repository

import (
	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
	"github.com/jackc/pgx/v5/pgtype"
)

// --- Helper functions untuk konversi pgtype.Int4 ↔ *int ---

// int4ToIntPtr mengkonversi pgtype.Int4 ke *int.
// Mengembalikan nil jika Int4 tidak valid (NULL).
func int4ToIntPtr(i pgtype.Int4) *int {
	if !i.Valid {
		return nil
	}
	v := int(i.Int32)
	return &v
}

// intPtrToInt4 mengkonversi *int ke pgtype.Int4.
// Mengembalikan Int4 tidak valid jika pointer nil.
func intPtrToInt4(i *int) pgtype.Int4 {
	if i == nil {
		return pgtype.Int4{Valid: false}
	}
	return pgtype.Int4{Int32: int32(*i), Valid: true}
}

// --- Helper functions untuk konversi pgtype.Int8 ↔ *int64 ---

// int8ToInt64Ptr mengkonversi pgtype.Int8 ke *int64.
// Mengembalikan nil jika Int8 tidak valid (NULL).
func int8ToInt64Ptr(i pgtype.Int8) *int64 {
	if !i.Valid {
		return nil
	}
	v := i.Int64
	return &v
}

// int64PtrToInt8 mengkonversi *int64 ke pgtype.Int8.
// Mengembalikan Int8 tidak valid jika pointer nil.
func int64PtrToInt8(i *int64) pgtype.Int8 {
	if i == nil {
		return pgtype.Int8{Valid: false}
	}
	return pgtype.Int8{Int64: *i, Valid: true}
}

// --- Helper functions untuk mapping sqlc Package → domain.Package ---

// mapPackageRow memetakan Package (sqlc model) ke domain.Package.
func mapPackageRow(row Package) *domain.Package {
	return &domain.Package{
		ID:                  uuidToString(row.ID),
		TenantID:            uuidToString(row.TenantID),
		Type:                domain.PackageType(row.Type),
		Name:                row.Name,
		Description:         textToString(row.Description),
		IsActive:            row.IsActive,
		DownloadMbps:        int(row.DownloadMbps),
		UploadMbps:          int(row.UploadMbps),
		BandwidthType:       textToString(row.BandwidthType),
		BurstDownloadMbps:   int4ToIntPtr(row.BurstDownloadMbps),
		BurstUploadMbps:     int4ToIntPtr(row.BurstUploadMbps),
		BurstThresholdMbps:  int4ToIntPtr(row.BurstThresholdMbps),
		BurstTimeSeconds:    int4ToIntPtr(row.BurstTimeSeconds),
		QuotaType:           domain.QuotaType(row.QuotaType),
		QuotaMB:             int4ToIntPtr(row.QuotaMb),
		QuotaAction:         textToString(row.QuotaAction),
		ThrottleMbps:        int4ToIntPtr(row.ThrottleMbps),
		MonthlyPrice:        int8ToInt64Ptr(row.MonthlyPrice),
		InstallationFee:     row.InstallationFee,
		SellPrice:           int8ToInt64Ptr(row.SellPrice),
		ResellerPrice:       int8ToInt64Ptr(row.ResellerPrice),
		DurationValue:       int4ToIntPtr(row.DurationValue),
		DurationUnit:        textToString(row.DurationUnit),
		SharedUsers:         int(row.SharedUsers),
		MikrotikProfileName: textToString(row.MikrotikProfileName),
		AddressPool:         textToString(row.AddressPool),
		ParentQueue:         textToString(row.ParentQueue),
		HotspotProfileName:  textToString(row.HotspotProfileName),
		CreatedAt:           timestamptzToTime(row.CreatedAt),
		UpdatedAt:           timestamptzToTime(row.UpdatedAt),
	}
}

// mapGetPackageByIDRow memetakan GetPackageByIDRow (sqlc model) ke domain.Package.
// Termasuk field komputasi customer_count.
func mapGetPackageByIDRow(row GetPackageByIDRow) *domain.Package {
	return &domain.Package{
		ID:                  uuidToString(row.ID),
		TenantID:            uuidToString(row.TenantID),
		Type:                domain.PackageType(row.Type),
		Name:                row.Name,
		Description:         textToString(row.Description),
		IsActive:            row.IsActive,
		DownloadMbps:        int(row.DownloadMbps),
		UploadMbps:          int(row.UploadMbps),
		BandwidthType:       textToString(row.BandwidthType),
		BurstDownloadMbps:   int4ToIntPtr(row.BurstDownloadMbps),
		BurstUploadMbps:     int4ToIntPtr(row.BurstUploadMbps),
		BurstThresholdMbps:  int4ToIntPtr(row.BurstThresholdMbps),
		BurstTimeSeconds:    int4ToIntPtr(row.BurstTimeSeconds),
		QuotaType:           domain.QuotaType(row.QuotaType),
		QuotaMB:             int4ToIntPtr(row.QuotaMb),
		QuotaAction:         textToString(row.QuotaAction),
		ThrottleMbps:        int4ToIntPtr(row.ThrottleMbps),
		MonthlyPrice:        int8ToInt64Ptr(row.MonthlyPrice),
		InstallationFee:     row.InstallationFee,
		SellPrice:           int8ToInt64Ptr(row.SellPrice),
		ResellerPrice:       int8ToInt64Ptr(row.ResellerPrice),
		DurationValue:       int4ToIntPtr(row.DurationValue),
		DurationUnit:        textToString(row.DurationUnit),
		SharedUsers:         int(row.SharedUsers),
		MikrotikProfileName: textToString(row.MikrotikProfileName),
		AddressPool:         textToString(row.AddressPool),
		ParentQueue:         textToString(row.ParentQueue),
		HotspotProfileName:  textToString(row.HotspotProfileName),
		CustomerCount:       int(row.CustomerCount),
		CreatedAt:           timestamptzToTime(row.CreatedAt),
		UpdatedAt:           timestamptzToTime(row.UpdatedAt),
	}
}

// domainPkgToCreateParams mengkonversi domain.Package ke CreatePackageParams (sqlc).
func domainPkgToCreateParams(pkg *domain.Package) CreatePackageParams {
	return CreatePackageParams{
		TenantID:            stringToUUID(pkg.TenantID),
		Type:                string(pkg.Type),
		Name:                pkg.Name,
		Description:         stringToText(pkg.Description),
		IsActive:            pkg.IsActive,
		DownloadMbps:        int32(pkg.DownloadMbps),
		UploadMbps:          int32(pkg.UploadMbps),
		BandwidthType:       stringToText(pkg.BandwidthType),
		BurstDownloadMbps:   intPtrToInt4(pkg.BurstDownloadMbps),
		BurstUploadMbps:     intPtrToInt4(pkg.BurstUploadMbps),
		BurstThresholdMbps:  intPtrToInt4(pkg.BurstThresholdMbps),
		BurstTimeSeconds:    intPtrToInt4(pkg.BurstTimeSeconds),
		QuotaType:           string(pkg.QuotaType),
		QuotaMb:             intPtrToInt4(pkg.QuotaMB),
		QuotaAction:         stringToText(pkg.QuotaAction),
		ThrottleMbps:        intPtrToInt4(pkg.ThrottleMbps),
		MonthlyPrice:        int64PtrToInt8(pkg.MonthlyPrice),
		InstallationFee:     pkg.InstallationFee,
		SellPrice:           int64PtrToInt8(pkg.SellPrice),
		ResellerPrice:       int64PtrToInt8(pkg.ResellerPrice),
		DurationValue:       intPtrToInt4(pkg.DurationValue),
		DurationUnit:        stringToText(pkg.DurationUnit),
		SharedUsers:         int32(pkg.SharedUsers),
		MikrotikProfileName: stringToText(pkg.MikrotikProfileName),
		AddressPool:         stringToText(pkg.AddressPool),
		ParentQueue:         stringToText(pkg.ParentQueue),
		HotspotProfileName:  stringToText(pkg.HotspotProfileName),
	}
}

// domainPkgToUpdateParams mengkonversi domain.Package ke UpdatePackageParams (sqlc).
func domainPkgToUpdateParams(pkg *domain.Package) UpdatePackageParams {
	return UpdatePackageParams{
		ID:                  stringToUUID(pkg.ID),
		Name:                pkg.Name,
		Description:         stringToText(pkg.Description),
		DownloadMbps:        int32(pkg.DownloadMbps),
		UploadMbps:          int32(pkg.UploadMbps),
		BandwidthType:       stringToText(pkg.BandwidthType),
		BurstDownloadMbps:   intPtrToInt4(pkg.BurstDownloadMbps),
		BurstUploadMbps:     intPtrToInt4(pkg.BurstUploadMbps),
		BurstThresholdMbps:  intPtrToInt4(pkg.BurstThresholdMbps),
		BurstTimeSeconds:    intPtrToInt4(pkg.BurstTimeSeconds),
		QuotaType:           string(pkg.QuotaType),
		QuotaMb:             intPtrToInt4(pkg.QuotaMB),
		QuotaAction:         stringToText(pkg.QuotaAction),
		ThrottleMbps:        intPtrToInt4(pkg.ThrottleMbps),
		MonthlyPrice:        int64PtrToInt8(pkg.MonthlyPrice),
		InstallationFee:     pkg.InstallationFee,
		SellPrice:           int64PtrToInt8(pkg.SellPrice),
		ResellerPrice:       int64PtrToInt8(pkg.ResellerPrice),
		DurationValue:       intPtrToInt4(pkg.DurationValue),
		DurationUnit:        stringToText(pkg.DurationUnit),
		SharedUsers:         int32(pkg.SharedUsers),
		MikrotikProfileName: stringToText(pkg.MikrotikProfileName),
		AddressPool:         stringToText(pkg.AddressPool),
		ParentQueue:         stringToText(pkg.ParentQueue),
		HotspotProfileName:  stringToText(pkg.HotspotProfileName),
	}
}
