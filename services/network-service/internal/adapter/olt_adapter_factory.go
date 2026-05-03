// Package adapter — OLTAdapterFactory membuat instance OLTAdapter berdasarkan brand.
// Jika networkMode == "mock", selalu mengembalikan MockOLTAdapter.
// Jika networkMode == "live", memilih adapter sesuai brand OLT.
package adapter

import (
	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// Compile-time check: pastikan oltAdapterFactory mengimplementasikan domain.OLTAdapterFactory.
var _ domain.OLTAdapterFactory = (*oltAdapterFactory)(nil)

// oltAdapterFactory membuat instance OLTAdapter berdasarkan brand dan mode jaringan.
type oltAdapterFactory struct {
	networkMode   string               // "mock" atau "live"
	snmpConnector domain.SNMPConnector // koneksi SNMP untuk adapter live
	cliConnector  domain.CLIConnector  // koneksi CLI untuk adapter live
}

// NewOLTAdapterFactory membuat instance baru OLTAdapterFactory.
// networkMode menentukan apakah menggunakan mock adapter atau adapter live per brand.
// snmpConn dan cliConn digunakan oleh adapter live untuk komunikasi ke OLT.
func NewOLTAdapterFactory(
	networkMode string,
	snmpConn domain.SNMPConnector,
	cliConn domain.CLIConnector,
) domain.OLTAdapterFactory {
	return &oltAdapterFactory{
		networkMode:   networkMode,
		snmpConnector: snmpConn,
		cliConnector:  cliConn,
	}
}

// CreateAdapter membuat adapter sesuai brand OLT dengan konfigurasi SNMP dan CLI.
// Jika networkMode == "mock", selalu mengembalikan MockOLTAdapter tanpa memperhatikan brand.
// Jika networkMode == "live", memilih adapter berdasarkan brand:
//   - BrandZTE → ZTEAdapter (akan diimplementasikan di task 12)
//   - BrandHuawei → HuaweiAdapter (stub, task 13)
//   - BrandFiberHome → FiberHomeAdapter (stub, task 13)
//   - BrandVSOL → VSOLAdapter (stub, task 13)
//   - BrandHSGQ → HSGQAdapter (stub, task 13)
//   - default → ErrUnsupportedBrand
func (f *oltAdapterFactory) CreateAdapter(
	brand domain.OLTBrand,
	snmpCfg domain.SNMPConfig,
	cliCfg domain.CLIConfig,
) (domain.OLTAdapter, error) {
	// Mode mock: selalu kembalikan MockOLTAdapter untuk semua brand.
	if f.networkMode == "mock" {
		return &MockOLTAdapter{}, nil
	}

	// Mode live: pilih adapter berdasarkan brand OLT.
	switch brand {
	case domain.BrandZTE:
		return NewZTEAdapter(f.snmpConnector, f.cliConnector, snmpCfg, cliCfg), nil
	case domain.BrandHuawei:
		// HuaweiAdapter stub — method-method akan mengembalikan ErrUnsupportedBrand saat dipanggil.
		return NewHuaweiAdapter(f.snmpConnector, f.cliConnector, snmpCfg, cliCfg), nil
	case domain.BrandFiberHome:
		// FiberHomeAdapter stub — method-method akan mengembalikan ErrUnsupportedBrand saat dipanggil.
		return NewFiberHomeAdapter(f.snmpConnector, f.cliConnector, snmpCfg, cliCfg), nil
	case domain.BrandVSOL:
		// VSOLAdapter stub — method-method akan mengembalikan ErrUnsupportedBrand saat dipanggil.
		return NewVSOLAdapter(f.snmpConnector, f.cliConnector, snmpCfg, cliCfg), nil
	case domain.BrandHSGQ:
		// HSGQAdapter stub — method-method akan mengembalikan ErrUnsupportedBrand saat dipanggil.
		return NewHSGQAdapter(f.snmpConnector, f.cliConnector, snmpCfg, cliCfg), nil
	default:
		return nil, domain.ErrUnsupportedBrand
	}
}
