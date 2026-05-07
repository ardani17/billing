// Memverifikasi bahwa ResolveVLAN mengembalikan VLAN yang benar berdasarkan
package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
	"pgregory.net/rapid"
)

// =============================================================================
// **Memvalidasi: Kebutuhan 9.5**
//
// - "single" -> selalu mengembalikan bawaan VLAN (VLAN pertama tipe data)
// - "per_paket" -> mengembalikan VLAN yang di-map ke package_id
// - "per_odp" -> mengembalikan VLAN yang di-map ke odp_id
// - "per_pelanggan" -> mengembalikan VLAN unik per customer_id
// =============================================================================

func validStrategyGen() *rapid.Generator[domain.VLANStrategy] {
	return rapid.SampledFrom([]domain.VLANStrategy{
		domain.VLANStrategySingle,
		domain.VLANStrategyPerPaket,
		domain.VLANStrategyPerODP,
		domain.VLANStrategyPerPelanggan,
	})
}

func invalidStrategyGen() *rapid.Generator[domain.VLANStrategy] {
	return rapid.Custom(func(rt *rapid.T) domain.VLANStrategy {
		s := rapid.StringMatching(`[a-z_]{3,20}`).Draw(rt, "invalidStrategy")
		for s == "single" || s == "per_paket" || s == "per_odp" || s == "per_pelanggan" {
			s = rapid.StringMatching(`[a-z_]{3,20}`).Draw(rt, "invalidStrategy")
		}
		return domain.VLANStrategy(s)
	})
}

// setupVLANStrategyTest menyiapkan VLANManager dengan VLAN untuk semua strategy.
func setupVLANStrategyTest(packageID, odpID, customerID string) (*vlanManager, *mockVLANRepoForManager) {
	vlanRepo := newMockVLANRepoForManager()
	oltRepo := newMockOLTRepo()

	oltRepo.olts["olt-001"] = &domain.OLT{
		ID:       "olt-001",
		TenantID: "tenant-001",
		Name:     "OLT-Test",
		Status:   domain.OLTStatusOnline,
	}

	now := time.Now()

	// Bawaan VLAN (untuk strategy single) - tipe data, tanpa description
	vlanRepo.vlans["vlan-default"] = &domain.VLAN{
		ID:        "vlan-default",
		TenantID:  "tenant-001",
		OLTID:     "olt-001",
		VLANID:    100,
		Name:      "Default-VLAN",
		VLANType:  domain.VLANTypeData,
		CreatedAt: now,
		UpdatedAt: now,
	}
	// Set bawaan VLAN secara eksplisit
	vlanRepo.defaultVLAN = vlanRepo.vlans["vlan-default"]

	// VLAN per paket - tipe voice agar tidak bentrok dengan bawaan
	if packageID != "" {
		vlanRepo.vlans["vlan-paket"] = &domain.VLAN{
			ID:          "vlan-paket",
			TenantID:    "tenant-001",
			OLTID:       "olt-001",
			VLANID:      200,
			Name:        "VLAN-Paket",
			VLANType:    domain.VLANTypeVoice,
			Description: packageID,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
	}

	// VLAN per ODP - tipe voice agar tidak bentrok dengan bawaan
	if odpID != "" {
		vlanRepo.vlans["vlan-odp"] = &domain.VLAN{
			ID:          "vlan-odp",
			TenantID:    "tenant-001",
			OLTID:       "olt-001",
			VLANID:      300,
			Name:        "VLAN-ODP",
			VLANType:    domain.VLANTypeVoice,
			Description: odpID,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
	}

	// VLAN per pelanggan - tipe voice agar tidak bentrok dengan bawaan
	if customerID != "" {
		vlanRepo.vlans["vlan-cust"] = &domain.VLAN{
			ID:          "vlan-cust",
			TenantID:    "tenant-001",
			OLTID:       "olt-001",
			VLANID:      400,
			Name:        "VLAN-Customer",
			VLANType:    domain.VLANTypeVoice,
			Description: customerID,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
	}

	mgr := NewVLANManager(vlanRepo, oltRepo).(*vlanManager)
	return mgr, vlanRepo
}

// TestProperty4_SingleStrategy_AlwaysReturnsDefault memverifikasi bahwa
// strategy "single" selalu mengembalikan bawaan VLAN, terlepas dari context.
//
// **Memvalidasi: Kebutuhan 9.5**
func TestProperty4_SingleStrategy_AlwaysReturnsDefault(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		packageID := rapid.StringMatching(`pkg-[a-z0-9]{4}`).Draw(rt, "packageID")
		odpID := rapid.StringMatching(`odp-[a-z0-9]{4}`).Draw(rt, "odpID")
		customerID := rapid.StringMatching(`cust-[a-z0-9]{4}`).Draw(rt, "customerID")

		mgr, _ := setupVLANStrategyTest(packageID, odpID, customerID)
		ctx := context.Background()

		// Strategy single harus mengembalikan bawaan VLAN, apapun context-nya
		resolveCtx := domain.VLANResolveContext{
			PackageID:  packageID,
			ODPID:      odpID,
			CustomerID: customerID,
		}

		vlan, err := mgr.ResolveVLAN(ctx, "olt-001", domain.VLANStrategySingle, resolveCtx)
		if err != nil {
			t.Fatalf("strategy single gagal: %v", err)
		}
		if vlan.ID != "vlan-default" {
			t.Errorf("strategy single harus mengembalikan default VLAN, got %q", vlan.ID)
		}
	})
}

// TestProperty4_PerPaketStrategy_ReturnsPackageVLAN memverifikasi bahwa
// strategy "per_paket" mengembalikan VLAN yang di-map ke package_id.
//
// **Memvalidasi: Kebutuhan 9.5**
func TestProperty4_PerPaketStrategy_ReturnsPackageVLAN(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		packageID := rapid.StringMatching(`pkg-[a-z0-9]{4}`).Draw(rt, "packageID")

		mgr, _ := setupVLANStrategyTest(packageID, "", "")
		ctx := context.Background()

		resolveCtx := domain.VLANResolveContext{PackageID: packageID}
		vlan, err := mgr.ResolveVLAN(ctx, "olt-001", domain.VLANStrategyPerPaket, resolveCtx)
		if err != nil {
			t.Fatalf("strategy per_paket gagal: %v", err)
		}
		// Harus mengembalikan VLAN yang di-map ke package_id
		if vlan.ID != "vlan-paket" {
			t.Errorf("strategy per_paket harus mengembalikan VLAN paket, got %q", vlan.ID)
		}
	})
}

// TestProperty4_PerODPStrategy_ReturnsODPVLAN memverifikasi bahwa
// strategy "per_odp" mengembalikan VLAN yang di-map ke odp_id.
//
// **Memvalidasi: Kebutuhan 9.5**
func TestProperty4_PerODPStrategy_ReturnsODPVLAN(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		odpID := rapid.StringMatching(`odp-[a-z0-9]{4}`).Draw(rt, "odpID")

		mgr, _ := setupVLANStrategyTest("", odpID, "")
		ctx := context.Background()

		resolveCtx := domain.VLANResolveContext{ODPID: odpID}
		vlan, err := mgr.ResolveVLAN(ctx, "olt-001", domain.VLANStrategyPerODP, resolveCtx)
		if err != nil {
			t.Fatalf("strategy per_odp gagal: %v", err)
		}
		if vlan.ID != "vlan-odp" {
			t.Errorf("strategy per_odp harus mengembalikan VLAN ODP, got %q", vlan.ID)
		}
	})
}

// TestProperty4_PerPelangganStrategy_ReturnsCustomerVLAN memverifikasi bahwa
// strategy "per_pelanggan" mengembalikan VLAN unik per customer_id.
//
// **Memvalidasi: Kebutuhan 9.5**
func TestProperty4_PerPelangganStrategy_ReturnsCustomerVLAN(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		customerID := rapid.StringMatching(`cust-[a-z0-9]{4}`).Draw(rt, "customerID")

		mgr, _ := setupVLANStrategyTest("", "", customerID)
		ctx := context.Background()

		resolveCtx := domain.VLANResolveContext{CustomerID: customerID}
		vlan, err := mgr.ResolveVLAN(ctx, "olt-001", domain.VLANStrategyPerPelanggan, resolveCtx)
		if err != nil {
			t.Fatalf("strategy per_pelanggan gagal: %v", err)
		}
		if vlan.ID != "vlan-cust" {
			t.Errorf("strategy per_pelanggan harus mengembalikan VLAN customer, got %q", vlan.ID)
		}
	})
}

// TestProperty4_InvalidStrategy_AlwaysReturnsError memverifikasi bahwa
//
// **Memvalidasi: Kebutuhan 9.5**
func TestProperty4_InvalidStrategy_AlwaysReturnsError(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		strategy := invalidStrategyGen().Draw(rt, "strategy")

		mgr, _ := setupVLANStrategyTest("pkg-001", "odp-001", "cust-001")
		ctx := context.Background()

		resolveCtx := domain.VLANResolveContext{
			PackageID:  "pkg-001",
			ODPID:      "odp-001",
			CustomerID: "cust-001",
		}

		_, err := mgr.ResolveVLAN(ctx, "olt-001", strategy, resolveCtx)
		if err != domain.ErrInvalidVLANStrategy {
			t.Fatalf("strategy %q: expected ErrInvalidVLANStrategy, got: %v", strategy, err)
		}
	})
}

// TestProperty4_PerPelanggan_MissingCustomer_ReturnsError memverifikasi bahwa
//
// **Memvalidasi: Kebutuhan 9.5**
func TestProperty4_PerPelanggan_MissingCustomer_ReturnsError(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		mgr, _ := setupVLANStrategyTest("", "", "")
		ctx := context.Background()

		resolveCtx := domain.VLANResolveContext{CustomerID: ""}
		_, err := mgr.ResolveVLAN(ctx, "olt-001", domain.VLANStrategyPerPelanggan, resolveCtx)
		if err != domain.ErrVLANResolutionFailed {
			t.Fatalf("per_pelanggan tanpa customer_id: expected ErrVLANResolutionFailed, got: %v", err)
		}
	})
}
