// Package adapter - HuaweiAdapter adalah stub implementasi domain.OLTAdapter
// untuk OLT brand Huawei (MA5608T, MA5683T, MA5800).
// Saat ini semua method mengembalikan ErrUnsupportedBrand.
// Implementasi penuh akan ditambahkan di masa depan ketika
// OID dan CLI command Huawei sudah didokumentasikan.
package adapter

import (
	"context"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// Compile-time cek: pastikan HuaweiAdapter mengimplementasikan domain.OLTAdapter.
var _ domain.OLTAdapter = (*HuaweiAdapter)(nil)

// HuaweiAdapter adalah stub adapter untuk brand Huawei.
// Placeholder untuk implementasi masa depan.
type HuaweiAdapter struct {
	snmpConn domain.SNMPConnector
	cliConn  domain.CLIConnector
	snmpCfg  domain.SNMPConfig
	cliCfg   domain.CLIConfig
}

// NewHuaweiAdapter membuat instance baru HuaweiAdapter.
func NewHuaweiAdapter(
	snmpConn domain.SNMPConnector,
	cliConn domain.CLIConnector,
	snmpCfg domain.SNMPConfig,
	cliCfg domain.CLIConfig,
) *HuaweiAdapter {
	return &HuaweiAdapter{
		snmpConn: snmpConn,
		cliConn:  cliConn,
		snmpCfg:  snmpCfg,
		cliCfg:   cliCfg,
	}
}

// GetSystemInfo - belum diimplementasikan untuk Huawei.
func (a *HuaweiAdapter) GetSystemInfo(_ context.Context) (*domain.OLTSystemInfo, error) {
	return nil, domain.ErrUnsupportedBrand
}

// Ping - belum diimplementasikan untuk Huawei.
func (a *HuaweiAdapter) Ping(_ context.Context) error {
	return domain.ErrUnsupportedBrand
}

// GetPONPortStatus - belum diimplementasikan untuk Huawei.
func (a *HuaweiAdapter) GetPONPortStatus(_ context.Context, _ int) (*domain.PONPortStatus, error) {
	return nil, domain.ErrUnsupportedBrand
}

// GetAllPONPorts - belum diimplementasikan untuk Huawei.
func (a *HuaweiAdapter) GetAllPONPorts(_ context.Context) ([]domain.PONPortStatus, error) {
	return nil, domain.ErrUnsupportedBrand
}

// GetONTList - belum diimplementasikan untuk Huawei.
func (a *HuaweiAdapter) GetONTList(_ context.Context, _ int) ([]domain.ONTPortStatus, error) {
	return nil, domain.ErrUnsupportedBrand
}

// GetONTSignal - belum diimplementasikan untuk Huawei.
func (a *HuaweiAdapter) GetONTSignal(_ context.Context, _ int, _ int) (*domain.ONTSignalInfo, error) {
	return nil, domain.ErrUnsupportedBrand
}

// GetAlarms - belum diimplementasikan untuk Huawei.
func (a *HuaweiAdapter) GetAlarms(_ context.Context) ([]domain.OLTAlarm, error) {
	return nil, domain.ErrUnsupportedBrand
}

// GetSFPInfo - belum diimplementasikan untuk Huawei.
func (a *HuaweiAdapter) GetSFPInfo(_ context.Context, _ int) (*domain.SFPInfo, error) {
	return nil, domain.ErrUnsupportedBrand
}

// GetTrafficStats - belum diimplementasikan untuk Huawei.
func (a *HuaweiAdapter) GetTrafficStats(_ context.Context, _ int) (*domain.PONTrafficStats, error) {
	return nil, domain.ErrUnsupportedBrand
}

// --- Provisioning methods - belum diimplementasikan untuk Huawei ---

// AddONT - belum diimplementasikan untuk Huawei.
func (a *HuaweiAdapter) AddONT(_ context.Context, _ domain.AddONTParams) (*domain.ProvisioningResult, error) {
	return nil, domain.ErrUnsupportedBrand
}

// RemoveONT - belum diimplementasikan untuk Huawei.
func (a *HuaweiAdapter) RemoveONT(_ context.Context, _ domain.RemoveONTParams) (*domain.ProvisioningResult, error) {
	return nil, domain.ErrUnsupportedBrand
}

// AddServicePort - belum diimplementasikan untuk Huawei.
func (a *HuaweiAdapter) AddServicePort(_ context.Context, _ domain.AddServicePortParams) (*domain.ProvisioningResult, error) {
	return nil, domain.ErrUnsupportedBrand
}

// RemoveServicePort - belum diimplementasikan untuk Huawei.
func (a *HuaweiAdapter) RemoveServicePort(_ context.Context, _ domain.RemoveServicePortParams) (*domain.ProvisioningResult, error) {
	return nil, domain.ErrUnsupportedBrand
}

// RebootONT - belum diimplementasikan untuk Huawei.
func (a *HuaweiAdapter) RebootONT(_ context.Context, _ domain.RebootONTParams) (*domain.ProvisioningResult, error) {
	return nil, domain.ErrUnsupportedBrand
}

// GetUnregisteredONTs - belum diimplementasikan untuk Huawei.
func (a *HuaweiAdapter) GetUnregisteredONTs(_ context.Context) ([]domain.UnregisteredONT, error) {
	return nil, domain.ErrUnsupportedBrand
}
