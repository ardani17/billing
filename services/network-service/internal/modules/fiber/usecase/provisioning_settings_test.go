// Memverifikasi bahwa tenant tanpa settings record mendapat bawaan values
// yang konsisten: auto_provisioning=false, auto_port_migration=false,
// vlan_strategy="single".
package usecase

import (
	"testing"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
	"pgregory.net/rapid"
)

// =============================================================================
// **Memvalidasi: Kebutuhan 15.5**
//
// Untuk sembarang tenant_id yang tidak memiliki record provisioning settings
// auto_provisioning_enabled=false, auto_port_migration_enabled=false,
// vlan_strategy="single".
// =============================================================================

// tenantIDGen menghasilkan tenant ID acak dalam format UUID.
func tenantIDGen() *rapid.Generator[string] {
	return rapid.StringMatching(
		`[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}`,
	)
}

// TestProperty8_DefaultProvisioningSettings memverifikasi bahwa untuk
// sembarang tenant_id, DefaultProvisioningSettings selalu mengembalikan
// nilai bawaan yang benar dan konsisten.
//
// **Memvalidasi: Kebutuhan 15.5**
func TestProperty8_DefaultProvisioningSettings(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		tenantID := tenantIDGen().Draw(rt, "tenantID")

		settings := domain.DefaultProvisioningSettings(tenantID)

		// Settings tidak boleh nil
		if settings == nil {
			t.Fatal("DefaultProvisioningSettings mengembalikan nil")
		}

		if settings.TenantID != tenantID {
			t.Errorf("TenantID=%q, want=%q", settings.TenantID, tenantID)
		}

		// auto_provisioning_enabled harus false (bawaan)
		if settings.AutoProvisioningEnabled {
			t.Error("AutoProvisioningEnabled harus false (default), got true")
		}

		// auto_port_migration_enabled harus false (bawaan)
		if settings.AutoPortMigrationEnabled {
			t.Error("AutoPortMigrationEnabled harus false (default), got true")
		}

		// vlan_strategy harus "single" (bawaan)
		if settings.VLANStrategy != domain.VLANStrategySingle {
			t.Errorf("VLANStrategy=%q, want=%q",
				settings.VLANStrategy, domain.VLANStrategySingle)
		}

		// CreatedAt dan UpdatedAt harus terisi (non-zero)
		if settings.CreatedAt.IsZero() {
			t.Error("CreatedAt tidak boleh zero")
		}
		if settings.UpdatedAt.IsZero() {
			t.Error("UpdatedAt tidak boleh zero")
		}
	})
}

// TestProperty8_DefaultSettings_Consistency memverifikasi bahwa dua panggilan
// DefaultProvisioningSettings dengan tenant_id yang sama menghasilkan
// nilai bawaan yang identik (kecuali timestamp).
//
// **Memvalidasi: Kebutuhan 15.5**
func TestProperty8_DefaultSettings_Consistency(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		tenantID := tenantIDGen().Draw(rt, "tenantID")

		s1 := domain.DefaultProvisioningSettings(tenantID)
		s2 := domain.DefaultProvisioningSettings(tenantID)

		// Kedua panggilan harus menghasilkan bawaan values yang sama
		if s1.TenantID != s2.TenantID {
			t.Errorf("TenantID tidak konsisten: %q vs %q", s1.TenantID, s2.TenantID)
		}
		if s1.AutoProvisioningEnabled != s2.AutoProvisioningEnabled {
			t.Error("AutoProvisioningEnabled tidak konsisten antar panggilan")
		}
		if s1.AutoPortMigrationEnabled != s2.AutoPortMigrationEnabled {
			t.Error("AutoPortMigrationEnabled tidak konsisten antar panggilan")
		}
		if s1.VLANStrategy != s2.VLANStrategy {
			t.Errorf("VLANStrategy tidak konsisten: %q vs %q",
				s1.VLANStrategy, s2.VLANStrategy)
		}
	})
}

// TestProperty8_DefaultSettings_DifferentTenants memverifikasi bahwa
// untuk tenant_id yang berbeda, bawaan values tetap sama
// (hanya TenantID yang berbeda).
//
// **Memvalidasi: Kebutuhan 15.5**
func TestProperty8_DefaultSettings_DifferentTenants(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		tenantA := tenantIDGen().Draw(rt, "tenantA")
		tenantB := tenantIDGen().Draw(rt, "tenantB")

		sA := domain.DefaultProvisioningSettings(tenantA)
		sB := domain.DefaultProvisioningSettings(tenantB)

		// Bawaan values harus sama untuk semua tenant
		if sA.AutoProvisioningEnabled != sB.AutoProvisioningEnabled {
			t.Error("AutoProvisioningEnabled berbeda antar tenant, seharusnya sama")
		}
		if sA.AutoPortMigrationEnabled != sB.AutoPortMigrationEnabled {
			t.Error("AutoPortMigrationEnabled berbeda antar tenant, seharusnya sama")
		}
		if sA.VLANStrategy != sB.VLANStrategy {
			t.Errorf("VLANStrategy berbeda: tenant_a=%q, tenant_b=%q",
				sA.VLANStrategy, sB.VLANStrategy)
		}

		// TenantID harus sesuai masing-masing
		if sA.TenantID != tenantA {
			t.Errorf("tenant_a: TenantID=%q, want=%q", sA.TenantID, tenantA)
		}
		if sB.TenantID != tenantB {
			t.Errorf("tenant_b: TenantID=%q, want=%q", sB.TenantID, tenantB)
		}
	})
}
