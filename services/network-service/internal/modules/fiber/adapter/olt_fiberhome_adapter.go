// Package adapter - FiberHomeAdapter adalah stub implementasi domain.OLTAdapter
// untuk OLT brand FiberHome (AN5516, AN5006).
// Saat ini semua method mengembalikan ErrUnsupportedBrand.
// Implementasi penuh akan ditambahkan di masa depan ketika
// OID dan CLI command FiberHome sudah didokumentasikan.
package adapter

import (
	"context"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// Compile-time cek: pastikan FiberHomeAdapter mengimplementasikan domain.OLTAdapter.
var _ domain.OLTAdapter = (*FiberHomeAdapter)(nil)

// FiberHomeAdapter adalah stub adapter untuk brand FiberHome.
// Placeholder untuk implementasi masa depan.
type FiberHomeAdapter struct {
	snmpConn domain.SNMPConnector
	cliConn  domain.CLIConnector
	snmpCfg  domain.SNMPConfig
	cliCfg   domain.CLIConfig
}

// NewFiberHomeAdapter membuat instance baru FiberHomeAdapter.
func NewFiberHomeAdapter(
	snmpConn domain.SNMPConnector,
	cliConn domain.CLIConnector,
	snmpCfg domain.SNMPConfig,
	cliCfg domain.CLIConfig,
) *FiberHomeAdapter {
	return &FiberHomeAdapter{
		snmpConn: snmpConn,
		cliConn:  cliConn,
		snmpCfg:  snmpCfg,
		cliCfg:   cliCfg,
	}
}

// GetSystemInfo - belum diimplementasikan untuk FiberHome.
func (a *FiberHomeAdapter) GetSystemInfo(_ context.Context) (*domain.OLTSystemInfo, error) {
	return nil, domain.ErrUnsupportedBrand
}

// Ping - belum diimplementasikan untuk FiberHome.
func (a *FiberHomeAdapter) Ping(_ context.Context) error {
	return domain.ErrUnsupportedBrand
}

// GetPONPortStatus - belum diimplementasikan untuk FiberHome.
func (a *FiberHomeAdapter) GetPONPortStatus(_ context.Context, _ int) (*domain.PONPortStatus, error) {
	return nil, domain.ErrUnsupportedBrand
}

// GetAllPONPorts - belum diimplementasikan untuk FiberHome.
func (a *FiberHomeAdapter) GetAllPONPorts(_ context.Context) ([]domain.PONPortStatus, error) {
	return nil, domain.ErrUnsupportedBrand
}

// GetONTList - belum diimplementasikan untuk FiberHome.
func (a *FiberHomeAdapter) GetONTList(_ context.Context, _ int) ([]domain.ONTPortStatus, error) {
	return nil, domain.ErrUnsupportedBrand
}

// GetONTSignal - belum diimplementasikan untuk FiberHome.
func (a *FiberHomeAdapter) GetONTSignal(_ context.Context, _ int, _ int) (*domain.ONTSignalInfo, error) {
	return nil, domain.ErrUnsupportedBrand
}

// GetAlarms - belum diimplementasikan untuk FiberHome.
func (a *FiberHomeAdapter) GetAlarms(_ context.Context) ([]domain.OLTAlarm, error) {
	return nil, domain.ErrUnsupportedBrand
}

// GetSFPInfo - belum diimplementasikan untuk FiberHome.
func (a *FiberHomeAdapter) GetSFPInfo(_ context.Context, _ int) (*domain.SFPInfo, error) {
	return nil, domain.ErrUnsupportedBrand
}

// GetTrafficStats - belum diimplementasikan untuk FiberHome.
func (a *FiberHomeAdapter) GetTrafficStats(_ context.Context, _ int) (*domain.PONTrafficStats, error) {
	return nil, domain.ErrUnsupportedBrand
}

// --- Provisioning methods - belum diimplementasikan untuk FiberHome ---

// AddONT - belum diimplementasikan untuk FiberHome.
func (a *FiberHomeAdapter) AddONT(_ context.Context, _ domain.AddONTParams) (*domain.ProvisioningResult, error) {
	return nil, domain.ErrUnsupportedBrand
}

// RemoveONT - belum diimplementasikan untuk FiberHome.
func (a *FiberHomeAdapter) RemoveONT(_ context.Context, _ domain.RemoveONTParams) (*domain.ProvisioningResult, error) {
	return nil, domain.ErrUnsupportedBrand
}

// AddServicePort - belum diimplementasikan untuk FiberHome.
func (a *FiberHomeAdapter) AddServicePort(_ context.Context, _ domain.AddServicePortParams) (*domain.ProvisioningResult, error) {
	return nil, domain.ErrUnsupportedBrand
}

// RemoveServicePort - belum diimplementasikan untuk FiberHome.
func (a *FiberHomeAdapter) RemoveServicePort(_ context.Context, _ domain.RemoveServicePortParams) (*domain.ProvisioningResult, error) {
	return nil, domain.ErrUnsupportedBrand
}

// RebootONT - belum diimplementasikan untuk FiberHome.
func (a *FiberHomeAdapter) RebootONT(_ context.Context, _ domain.RebootONTParams) (*domain.ProvisioningResult, error) {
	return nil, domain.ErrUnsupportedBrand
}

// GetUnregisteredONTs - belum diimplementasikan untuk FiberHome.
func (a *FiberHomeAdapter) GetUnregisteredONTs(_ context.Context) ([]domain.UnregisteredONT, error) {
	return nil, domain.ErrUnsupportedBrand
}
