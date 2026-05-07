package adapter

import (
	"strings"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// OLTAddressingProfile mendeskripsikan cara model OLT memetakan port fisik ke OID/CLI.
type OLTAddressingProfile struct {
	DefaultShelf int
	DefaultSlot  int
	PONBase      int
	ONTBase      int
	MaxPONPorts  int
	MaxONTPerPON int
}

// OLTCLIProfile menyimpan karakteristik CLI per model.
type OLTCLIProfile struct {
	DefaultProtocol domain.CLIProtocol
	PromptSuffixes  []string
	PagerDisable    []string
	ConfigMode      string
}

// OLTModelProfile menyimpan metadata model spesifik di bawah brand.
type OLTModelProfile struct {
	Brand        domain.OLTBrand
	Model        string
	DisplayName  string
	Aliases      []string
	Capabilities CapabilitySet
	Addressing   OLTAddressingProfile
	CLI          OLTCLIProfile
}

// OLTBrandProfile menyimpan metadata brand dan daftar model.
type OLTBrandProfile struct {
	Brand       domain.OLTBrand
	DisplayName string
	Models      map[string]OLTModelProfile
}

// OLTProfileRegistry menyimpan semua brand/model yang diketahui adapter layer.
type OLTProfileRegistry struct {
	brands map[domain.OLTBrand]OLTBrandProfile
}

// NewDefaultOLTProfileRegistry membuat registry awal. Hanya ZTE C320 yang ditandai production-capable.
func NewDefaultOLTProfileRegistry() *OLTProfileRegistry {
	zteC320 := OLTModelProfile{
		Brand:        domain.BrandZTE,
		Model:        "C320",
		DisplayName:  "ZTE ZXA10 C320",
		Aliases:      []string{"C320", "ZXA10 C320"},
		Capabilities: zteC320Capabilities(),
		Addressing: OLTAddressingProfile{
			DefaultShelf: 1,
			DefaultSlot:  1,
			PONBase:      1,
			ONTBase:      1,
			MaxPONPorts:  16,
			MaxONTPerPON: 128,
		},
		CLI: OLTCLIProfile{
			DefaultProtocol: domain.CLIProtocolTelnet,
			PromptSuffixes:  []string{"#", ">", ")#"},
			PagerDisable:    []string{"terminal length 0"},
			ConfigMode:      "configure terminal",
		},
	}

	return &OLTProfileRegistry{
		brands: map[domain.OLTBrand]OLTBrandProfile{
			domain.BrandZTE: {
				Brand:       domain.BrandZTE,
				DisplayName: "ZTE",
				Models: map[string]OLTModelProfile{
					"C320": zteC320,
				},
			},
			domain.BrandHuawei: {
				Brand:       domain.BrandHuawei,
				DisplayName: "Huawei",
				Models:      map[string]OLTModelProfile{},
			},
			domain.BrandFiberHome: {
				Brand:       domain.BrandFiberHome,
				DisplayName: "FiberHome",
				Models:      map[string]OLTModelProfile{},
			},
			domain.BrandVSOL: {
				Brand:       domain.BrandVSOL,
				DisplayName: "VSOL",
				Models:      map[string]OLTModelProfile{},
			},
			domain.BrandHSGQ: {
				Brand:       domain.BrandHSGQ,
				DisplayName: "HSGQ",
				Models:      map[string]OLTModelProfile{},
			},
		},
	}
}

// IsKnownBrand mengembalikan true jika brand ada di registry.
func (r *OLTProfileRegistry) IsKnownBrand(brand domain.OLTBrand) bool {
	if r == nil {
		return false
	}
	_, ok := r.brands[brand]
	return ok
}

// DetectModel mencocokkan sysDescr ke model profile untuk brand tertentu.
func (r *OLTProfileRegistry) DetectModel(brand domain.OLTBrand, sysDescr string) (OLTModelProfile, bool) {
	if r == nil {
		return OLTModelProfile{}, false
	}
	brandProfile, ok := r.brands[brand]
	if !ok {
		return OLTModelProfile{}, false
	}
	upper := strings.ToUpper(sysDescr)
	for _, model := range brandProfile.Models {
		if strings.Contains(upper, strings.ToUpper(model.Model)) {
			return model, true
		}
		for _, alias := range model.Aliases {
			if strings.Contains(upper, strings.ToUpper(alias)) {
				return model, true
			}
		}
	}
	return OLTModelProfile{}, false
}

// GetModel mengambil profile model berdasarkan brand dan model string.
func (r *OLTProfileRegistry) GetModel(brand domain.OLTBrand, model string) (OLTModelProfile, bool) {
	if r == nil {
		return OLTModelProfile{}, false
	}
	brandProfile, ok := r.brands[brand]
	if !ok {
		return OLTModelProfile{}, false
	}
	profile, ok := brandProfile.Models[strings.ToUpper(strings.TrimSpace(model))]
	return profile, ok
}
