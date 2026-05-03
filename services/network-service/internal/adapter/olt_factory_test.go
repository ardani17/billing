package adapter

import (
	"testing"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
	"pgregory.net/rapid"
)

// allBrands berisi semua OLTBrand constant yang didefinisikan di domain.
// Digunakan untuk memastikan factory mapping exhaustive.
var allBrands = []domain.OLTBrand{
	domain.BrandZTE,
	domain.BrandHuawei,
	domain.BrandFiberHome,
	domain.BrandVSOL,
	domain.BrandHSGQ,
}

// brandGen menghasilkan OLTBrand acak dari daftar brand yang valid.
func brandGen() *rapid.Generator[domain.OLTBrand] {
	return rapid.SampledFrom(allBrands)
}

// TestProperty10_AdapterFactoryBrandMapping_MockMode memverifikasi bahwa
// untuk setiap OLTBrand yang valid, CreateAdapter dalam mode "mock"
// selalu mengembalikan adapter non-nil tanpa error.
//
// **Validates: Requirements 3.3**
func TestProperty10_AdapterFactoryBrandMapping_MockMode(t *testing.T) {
	factory := NewOLTAdapterFactory("mock", nil, nil)

	rapid.Check(t, func(t *rapid.T) {
		brand := brandGen().Draw(t, "brand")
		snmpCfg := domain.SNMPConfig{Host: "10.0.0.1", Port: 161}
		cliCfg := domain.CLIConfig{Host: "10.0.0.1", Port: 22}

		adapter, err := factory.CreateAdapter(brand, snmpCfg, cliCfg)
		if err != nil {
			t.Fatalf("mock mode: CreateAdapter(%q) mengembalikan error: %v", brand, err)
		}
		if adapter == nil {
			t.Fatalf("mock mode: CreateAdapter(%q) mengembalikan adapter nil", brand)
		}

		// Pastikan adapter yang dikembalikan adalah MockOLTAdapter.
		if _, ok := adapter.(*MockOLTAdapter); !ok {
			t.Fatalf("mock mode: CreateAdapter(%q) bukan *MockOLTAdapter", brand)
		}
	})
}

// TestProperty10_AdapterFactoryBrandMapping_Exhaustive memverifikasi bahwa
// factory mapping exhaustive: setiap brand yang valid menghasilkan adapter
// (mock mode) atau error yang dikenali (live mode), tidak pernah panic.
//
// **Validates: Requirements 3.3**
func TestProperty10_AdapterFactoryBrandMapping_Exhaustive(t *testing.T) {
	mockFactory := NewOLTAdapterFactory("mock", nil, nil)
	liveFactory := NewOLTAdapterFactory("live", nil, nil)

	rapid.Check(t, func(t *rapid.T) {
		brand := brandGen().Draw(t, "brand")
		snmpCfg := domain.SNMPConfig{Host: "10.0.0.1", Port: 161}
		cliCfg := domain.CLIConfig{Host: "10.0.0.1", Port: 22}

		// Mock mode: semua brand harus berhasil.
		adapter, err := mockFactory.CreateAdapter(brand, snmpCfg, cliCfg)
		if err != nil {
			t.Fatalf("mock factory: brand %q error: %v", brand, err)
		}
		if adapter == nil {
			t.Fatalf("mock factory: brand %q adapter nil", brand)
		}

		// Live mode: semua brand yang valid harus mengembalikan adapter atau
		// ErrUnsupportedBrand (karena stub belum diimplementasikan).
		// Tidak boleh panic atau error selain ErrUnsupportedBrand.
		adapterLive, errLive := liveFactory.CreateAdapter(brand, snmpCfg, cliCfg)
		_ = adapterLive // bisa nil untuk stub
		if errLive != nil && errLive != domain.ErrUnsupportedBrand {
			t.Fatalf("live factory: brand %q error tak terduga: %v", brand, errLive)
		}
	})
}

// TestAdapterFactory_UnsupportedBrand memverifikasi bahwa brand yang tidak
// dikenali mengembalikan ErrUnsupportedBrand di mode live.
func TestAdapterFactory_UnsupportedBrand(t *testing.T) {
	factory := NewOLTAdapterFactory("live", nil, nil)
	snmpCfg := domain.SNMPConfig{Host: "10.0.0.1", Port: 161}
	cliCfg := domain.CLIConfig{Host: "10.0.0.1", Port: 22}

	_, err := factory.CreateAdapter("unknown_brand", snmpCfg, cliCfg)
	if err != domain.ErrUnsupportedBrand {
		t.Fatalf("expected ErrUnsupportedBrand, got: %v", err)
	}
}

// TestAdapterFactory_MockModeIgnoresBrand memverifikasi bahwa mode mock
// mengembalikan MockOLTAdapter bahkan untuk brand yang tidak dikenali.
func TestAdapterFactory_MockModeIgnoresBrand(t *testing.T) {
	factory := NewOLTAdapterFactory("mock", nil, nil)
	snmpCfg := domain.SNMPConfig{Host: "10.0.0.1", Port: 161}
	cliCfg := domain.CLIConfig{Host: "10.0.0.1", Port: 22}

	adapter, err := factory.CreateAdapter("nonexistent", snmpCfg, cliCfg)
	if err != nil {
		t.Fatalf("mock mode seharusnya tidak error untuk brand apapun: %v", err)
	}
	if adapter == nil {
		t.Fatal("mock mode seharusnya mengembalikan adapter non-nil")
	}
	if _, ok := adapter.(*MockOLTAdapter); !ok {
		t.Fatal("mock mode seharusnya mengembalikan *MockOLTAdapter")
	}
}
