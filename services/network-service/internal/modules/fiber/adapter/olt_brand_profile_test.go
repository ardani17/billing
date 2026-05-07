package adapter

import (
	"testing"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

func TestDefaultOLTProfileRegistry_ZTEC320Capabilities(t *testing.T) {
	registry := NewDefaultOLTProfileRegistry()

	model, ok := registry.GetModel(domain.BrandZTE, "C320")
	if !ok {
		t.Fatal("profile ZTE C320 tidak ditemukan")
	}
	if model.Brand != domain.BrandZTE {
		t.Fatalf("brand = %q, want %q", model.Brand, domain.BrandZTE)
	}
	for _, capability := range []OLTCapability{
		CapabilitySNMPSystemProbe,
		CapabilityPONMonitoring,
		CapabilityONTList,
		CapabilityONTSignal,
		CapabilitySFPMonitoring,
		CapabilityTrafficStats,
		CapabilityAlarmPolling,
		CapabilityAlarmTrap,
		CapabilityUnregisteredONT,
		CapabilityONTProvisioning,
		CapabilityServicePort,
		CapabilityONTReboot,
	} {
		if !model.Capabilities.Supports(capability) {
			t.Fatalf("ZTE C320 harus support capability %q", capability)
		}
	}
}

func TestDefaultOLTProfileRegistry_UnknownBrandUnsupported(t *testing.T) {
	registry := NewDefaultOLTProfileRegistry()

	if registry.IsKnownBrand(domain.OLTBrand("unknown-brand")) {
		t.Fatal("unknown-brand tidak boleh dianggap known")
	}
	if _, ok := registry.GetModel(domain.OLTBrand("unknown-brand"), "X1"); ok {
		t.Fatal("unknown-brand tidak boleh punya model profile")
	}
}
