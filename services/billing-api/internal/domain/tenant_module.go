package domain

const (
	ModuleBillingCore  = "billing_core"
	ModuleMikroTik     = "mikrotik"
	ModuleFiberNetwork = "fiber_network"
)

type TenantModuleCapabilities struct {
	BillingCore  bool `json:"billing_core"`
	MikroTik     bool `json:"mikrotik"`
	FiberNetwork bool `json:"fiber_network"`
}

func DefaultTenantModuleCapabilities() TenantModuleCapabilities {
	return TenantModuleCapabilities{
		BillingCore:  true,
		MikroTik:     false,
		FiberNetwork: false,
	}
}
