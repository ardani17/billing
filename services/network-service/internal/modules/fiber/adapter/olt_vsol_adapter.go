// Package adapter - VSOLAdapter adalah stub implementasi domain.OLTAdapter
// untuk OLT brand VSOL (V1600G, V1600D).
// Saat ini semua method mengembalikan ErrUnsupportedBrand.
// Implementasi penuh akan ditambahkan di masa depan ketika
// OID dan CLI command VSOL sudah didokumentasikan.
package adapter

import (
	"context"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// Compile-time cek: pastikan VSOLAdapter mengimplementasikan domain.OLTAdapter.
var _ domain.OLTAdapter = (*VSOLAdapter)(nil)

// VSOLAdapter adalah stub adapter untuk brand VSOL.
// Placeholder untuk implementasi masa depan.
type VSOLAdapter struct {
	snmpConn domain.SNMPConnector
	cliConn  domain.CLIConnector
	snmpCfg  domain.SNMPConfig
	cliCfg   domain.CLIConfig
}

// NewVSOLAdapter membuat instance baru VSOLAdapter.
func NewVSOLAdapter(
	snmpConn domain.SNMPConnector,
	cliConn domain.CLIConnector,
	snmpCfg domain.SNMPConfig,
	cliCfg domain.CLIConfig,
) *VSOLAdapter {
	return &VSOLAdapter{
		snmpConn: snmpConn,
		cliConn:  cliConn,
		snmpCfg:  snmpCfg,
		cliCfg:   cliCfg,
	}
}

// GetSystemInfo - belum diimplementasikan untuk VSOL.
func (a *VSOLAdapter) GetSystemInfo(_ context.Context) (*domain.OLTSystemInfo, error) {
	return nil, domain.ErrUnsupportedBrand
}

// Ping - belum diimplementasikan untuk VSOL.
func (a *VSOLAdapter) Ping(_ context.Context) error {
	return domain.ErrUnsupportedBrand
}

// GetPONPortStatus - belum diimplementasikan untuk VSOL.
func (a *VSOLAdapter) GetPONPortStatus(_ context.Context, _ int) (*domain.PONPortStatus, error) {
	return nil, domain.ErrUnsupportedBrand
}

// GetAllPONPorts - belum diimplementasikan untuk VSOL.
func (a *VSOLAdapter) GetAllPONPorts(_ context.Context) ([]domain.PONPortStatus, error) {
	return nil, domain.ErrUnsupportedBrand
}

// GetONTList - belum diimplementasikan untuk VSOL.
func (a *VSOLAdapter) GetONTList(_ context.Context, _ int) ([]domain.ONTPortStatus, error) {
	return nil, domain.ErrUnsupportedBrand
}

// GetONTSignal - belum diimplementasikan untuk VSOL.
func (a *VSOLAdapter) GetONTSignal(_ context.Context, _ int, _ int) (*domain.ONTSignalInfo, error) {
	return nil, domain.ErrUnsupportedBrand
}

// GetAlarms - belum diimplementasikan untuk VSOL.
func (a *VSOLAdapter) GetAlarms(_ context.Context) ([]domain.OLTAlarm, error) {
	return nil, domain.ErrUnsupportedBrand
}

// GetSFPInfo - belum diimplementasikan untuk VSOL.
func (a *VSOLAdapter) GetSFPInfo(_ context.Context, _ int) (*domain.SFPInfo, error) {
	return nil, domain.ErrUnsupportedBrand
}

// GetTrafficStats - belum diimplementasikan untuk VSOL.
func (a *VSOLAdapter) GetTrafficStats(_ context.Context, _ int) (*domain.PONTrafficStats, error) {
	return nil, domain.ErrUnsupportedBrand
}

// --- Provisioning methods - belum diimplementasikan untuk VSOL ---

// AddONT - belum diimplementasikan untuk VSOL.
func (a *VSOLAdapter) AddONT(_ context.Context, _ domain.AddONTParams) (*domain.ProvisioningResult, error) {
	return nil, domain.ErrUnsupportedBrand
}

// RemoveONT - belum diimplementasikan untuk VSOL.
func (a *VSOLAdapter) RemoveONT(_ context.Context, _ domain.RemoveONTParams) (*domain.ProvisioningResult, error) {
	return nil, domain.ErrUnsupportedBrand
}

// AddServicePort - belum diimplementasikan untuk VSOL.
func (a *VSOLAdapter) AddServicePort(_ context.Context, _ domain.AddServicePortParams) (*domain.ProvisioningResult, error) {
	return nil, domain.ErrUnsupportedBrand
}

// RemoveServicePort - belum diimplementasikan untuk VSOL.
func (a *VSOLAdapter) RemoveServicePort(_ context.Context, _ domain.RemoveServicePortParams) (*domain.ProvisioningResult, error) {
	return nil, domain.ErrUnsupportedBrand
}

// RebootONT - belum diimplementasikan untuk VSOL.
func (a *VSOLAdapter) RebootONT(_ context.Context, _ domain.RebootONTParams) (*domain.ProvisioningResult, error) {
	return nil, domain.ErrUnsupportedBrand
}

// GetUnregisteredONTs - belum diimplementasikan untuk VSOL.
func (a *VSOLAdapter) GetUnregisteredONTs(_ context.Context) ([]domain.UnregisteredONT, error) {
	return nil, domain.ErrUnsupportedBrand
}
