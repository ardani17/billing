// Package adapter - HSGQAdapter adalah stub implementasi domain.OLTAdapter
// untuk OLT brand HSGQ.
// Saat ini semua method mengembalikan ErrUnsupportedBrand.
// Implementasi penuh akan ditambahkan di masa depan ketika
// OID dan CLI command HSGQ sudah didokumentasikan.
package adapter

import (
	"context"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// Compile-time cek: pastikan HSGQAdapter mengimplementasikan domain.OLTAdapter.
var _ domain.OLTAdapter = (*HSGQAdapter)(nil)

// HSGQAdapter adalah stub adapter untuk brand HSGQ.
// Placeholder untuk implementasi masa depan.
type HSGQAdapter struct {
	snmpConn domain.SNMPConnector
	cliConn  domain.CLIConnector
	snmpCfg  domain.SNMPConfig
	cliCfg   domain.CLIConfig
}

// NewHSGQAdapter membuat instance baru HSGQAdapter.
func NewHSGQAdapter(
	snmpConn domain.SNMPConnector,
	cliConn domain.CLIConnector,
	snmpCfg domain.SNMPConfig,
	cliCfg domain.CLIConfig,
) *HSGQAdapter {
	return &HSGQAdapter{
		snmpConn: snmpConn,
		cliConn:  cliConn,
		snmpCfg:  snmpCfg,
		cliCfg:   cliCfg,
	}
}

// GetSystemInfo - belum diimplementasikan untuk HSGQ.
func (a *HSGQAdapter) GetSystemInfo(_ context.Context) (*domain.OLTSystemInfo, error) {
	return nil, domain.ErrUnsupportedBrand
}

// Ping - belum diimplementasikan untuk HSGQ.
func (a *HSGQAdapter) Ping(_ context.Context) error {
	return domain.ErrUnsupportedBrand
}

// GetPONPortStatus - belum diimplementasikan untuk HSGQ.
func (a *HSGQAdapter) GetPONPortStatus(_ context.Context, _ int) (*domain.PONPortStatus, error) {
	return nil, domain.ErrUnsupportedBrand
}

// GetAllPONPorts - belum diimplementasikan untuk HSGQ.
func (a *HSGQAdapter) GetAllPONPorts(_ context.Context) ([]domain.PONPortStatus, error) {
	return nil, domain.ErrUnsupportedBrand
}

// GetONTList - belum diimplementasikan untuk HSGQ.
func (a *HSGQAdapter) GetONTList(_ context.Context, _ int) ([]domain.ONTPortStatus, error) {
	return nil, domain.ErrUnsupportedBrand
}

// GetONTSignal - belum diimplementasikan untuk HSGQ.
func (a *HSGQAdapter) GetONTSignal(_ context.Context, _ int, _ int) (*domain.ONTSignalInfo, error) {
	return nil, domain.ErrUnsupportedBrand
}

// GetAlarms - belum diimplementasikan untuk HSGQ.
func (a *HSGQAdapter) GetAlarms(_ context.Context) ([]domain.OLTAlarm, error) {
	return nil, domain.ErrUnsupportedBrand
}

// GetSFPInfo - belum diimplementasikan untuk HSGQ.
func (a *HSGQAdapter) GetSFPInfo(_ context.Context, _ int) (*domain.SFPInfo, error) {
	return nil, domain.ErrUnsupportedBrand
}

// GetTrafficStats - belum diimplementasikan untuk HSGQ.
func (a *HSGQAdapter) GetTrafficStats(_ context.Context, _ int) (*domain.PONTrafficStats, error) {
	return nil, domain.ErrUnsupportedBrand
}

// --- Provisioning methods - belum diimplementasikan untuk HSGQ ---

// AddONT - belum diimplementasikan untuk HSGQ.
func (a *HSGQAdapter) AddONT(_ context.Context, _ domain.AddONTParams) (*domain.ProvisioningResult, error) {
	return nil, domain.ErrUnsupportedBrand
}

// RemoveONT - belum diimplementasikan untuk HSGQ.
func (a *HSGQAdapter) RemoveONT(_ context.Context, _ domain.RemoveONTParams) (*domain.ProvisioningResult, error) {
	return nil, domain.ErrUnsupportedBrand
}

// AddServicePort - belum diimplementasikan untuk HSGQ.
func (a *HSGQAdapter) AddServicePort(_ context.Context, _ domain.AddServicePortParams) (*domain.ProvisioningResult, error) {
	return nil, domain.ErrUnsupportedBrand
}

// RemoveServicePort - belum diimplementasikan untuk HSGQ.
func (a *HSGQAdapter) RemoveServicePort(_ context.Context, _ domain.RemoveServicePortParams) (*domain.ProvisioningResult, error) {
	return nil, domain.ErrUnsupportedBrand
}

// RebootONT - belum diimplementasikan untuk HSGQ.
func (a *HSGQAdapter) RebootONT(_ context.Context, _ domain.RebootONTParams) (*domain.ProvisioningResult, error) {
	return nil, domain.ErrUnsupportedBrand
}

// GetUnregisteredONTs - belum diimplementasikan untuk HSGQ.
func (a *HSGQAdapter) GetUnregisteredONTs(_ context.Context) ([]domain.UnregisteredONT, error) {
	return nil, domain.ErrUnsupportedBrand
}
