// Package adapter — Provisioning methods untuk MockOLTAdapter.
// File terpisah dari olt_mock_adapter.go karena batas 200 baris per file.
// Semua method mengembalikan simulasi sukses tanpa koneksi jaringan.
package adapter

import (
	"context"
	"fmt"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// AddONT mengembalikan simulasi sukses penambahan ONT.
func (m *MockOLTAdapter) AddONT(_ context.Context, params domain.AddONTParams) (*domain.ProvisioningResult, error) {
	cmds := []string{
		fmt.Sprintf("interface gpon-olt_1/%d", params.PONPortIndex),
		fmt.Sprintf("onu %d type auto sn %s ont-lineprofile-id %d ont-srvprofile-id %d",
			params.ONTIndex, params.SerialNumber, params.LineProfileID, params.ServiceProfileID),
		"exit",
	}
	return &domain.ProvisioningResult{
		Success:      true,
		CommandsSent: cmds,
		Responses:    []string{"OK", "OK", "OK"},
	}, nil
}

// RemoveONT mengembalikan simulasi sukses penghapusan ONT.
func (m *MockOLTAdapter) RemoveONT(_ context.Context, params domain.RemoveONTParams) (*domain.ProvisioningResult, error) {
	cmds := []string{
		fmt.Sprintf("interface gpon-olt_1/%d", params.PONPortIndex),
		fmt.Sprintf("no onu %d", params.ONTIndex),
		"exit",
	}
	return &domain.ProvisioningResult{
		Success:      true,
		CommandsSent: cmds,
		Responses:    []string{"OK", "OK", "OK"},
	}, nil
}

// AddServicePort mengembalikan simulasi sukses penambahan service-port.
func (m *MockOLTAdapter) AddServicePort(_ context.Context, params domain.AddServicePortParams) (*domain.ProvisioningResult, error) {
	gemPort := params.GemPort
	if gemPort <= 0 {
		gemPort = 1
	}
	cmd := fmt.Sprintf("service-port add vlan %d gpon-olt_1/%d ont %d gemport %d",
		params.VLANID, params.PONPortIndex, params.ONTIndex, gemPort)
	return &domain.ProvisioningResult{
		Success:      true,
		CommandsSent: []string{cmd},
		Responses:    []string{"OK"},
	}, nil
}

// RemoveServicePort mengembalikan simulasi sukses penghapusan service-port.
func (m *MockOLTAdapter) RemoveServicePort(_ context.Context, params domain.RemoveServicePortParams) (*domain.ProvisioningResult, error) {
	cmd := fmt.Sprintf("no service-port vlan %d gpon-olt_1/%d ont %d",
		params.VLANID, params.PONPortIndex, params.ONTIndex)
	return &domain.ProvisioningResult{
		Success:      true,
		CommandsSent: []string{cmd},
		Responses:    []string{"OK"},
	}, nil
}

// RebootONT mengembalikan simulasi sukses reboot ONT.
func (m *MockOLTAdapter) RebootONT(_ context.Context, params domain.RebootONTParams) (*domain.ProvisioningResult, error) {
	cmds := []string{
		fmt.Sprintf("interface gpon-olt_1/%d", params.PONPortIndex),
		fmt.Sprintf("onu reset %d", params.ONTIndex),
		"exit",
	}
	return &domain.ProvisioningResult{
		Success:      true,
		CommandsSent: cmds,
		Responses:    []string{"OK", "OK", "OK"},
	}, nil
}

// GetUnregisteredONTs mengembalikan daftar ONT unregistered simulasi.
// Mengembalikan 3 ONT sample pada port 0, 1, 2.
func (m *MockOLTAdapter) GetUnregisteredONTs(_ context.Context) ([]domain.UnregisteredONT, error) {
	return []domain.UnregisteredONT{
		{SerialNumber: "ZTEGMOCK0001", PONPortIndex: 0, ONTIndex: 50},
		{SerialNumber: "ZTEGMOCK0002", PONPortIndex: 1, ONTIndex: 51},
		{SerialNumber: "ZTEGMOCK0003", PONPortIndex: 2, ONTIndex: 52},
	}, nil
}
