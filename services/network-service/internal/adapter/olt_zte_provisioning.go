// Package adapter — Implementasi provisioning methods untuk ZTEAdapter.
// File terpisah dari olt_zte_adapter.go karena batas 200 baris per file.
// Menggunakan CLI command ZTE: onu add, service-port add, no onu, no service-port, onu reset.
package adapter

import (
	"context"
	"fmt"
	"strings"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// AddONT menambahkan ONT ke PON port ZTE via CLI.
// Navigasi ke interface gpon-olt, lalu kirim command onu add sn.
func (a *ZTEAdapter) AddONT(ctx context.Context, params domain.AddONTParams) (*domain.ProvisioningResult, error) {
	commands := []string{
		fmt.Sprintf("interface gpon-olt_1/%d", params.PONPortIndex),
		fmt.Sprintf("onu %d type auto sn %s ont-lineprofile-id %d ont-srvprofile-id %d",
			params.ONTIndex, params.SerialNumber, params.LineProfileID, params.ServiceProfileID),
		"exit",
	}

	responses, err := a.cliConn.ExecuteMultiple(ctx, a.cliCfg, commands)
	if err != nil {
		return &domain.ProvisioningResult{
			Success:      false,
			CommandsSent: commands,
			ErrorMessage: fmt.Sprintf("gagal menambahkan ONT: %v", err),
		}, nil
	}

	return &domain.ProvisioningResult{
		Success:      true,
		CommandsSent: commands,
		Responses:    responses,
	}, nil
}

// RemoveONT menghapus ONT dari PON port ZTE via CLI.
// Navigasi ke interface gpon-olt, lalu kirim command no onu.
func (a *ZTEAdapter) RemoveONT(ctx context.Context, params domain.RemoveONTParams) (*domain.ProvisioningResult, error) {
	commands := []string{
		fmt.Sprintf("interface gpon-olt_1/%d", params.PONPortIndex),
		fmt.Sprintf("no onu %d", params.ONTIndex),
		"exit",
	}

	responses, err := a.cliConn.ExecuteMultiple(ctx, a.cliCfg, commands)
	if err != nil {
		return &domain.ProvisioningResult{
			Success:      false,
			CommandsSent: commands,
			ErrorMessage: fmt.Sprintf("gagal menghapus ONT: %v", err),
		}, nil
	}

	return &domain.ProvisioningResult{
		Success:      true,
		CommandsSent: commands,
		Responses:    responses,
	}, nil
}

// AddServicePort menambahkan service-port dengan VLAN assignment via CLI ZTE.
// Command: service-port add vlan {vlan} gpon {port} ont {id} gemport {gem}.
func (a *ZTEAdapter) AddServicePort(ctx context.Context, params domain.AddServicePortParams) (*domain.ProvisioningResult, error) {
	gemPort := params.GemPort
	if gemPort <= 0 {
		gemPort = 1 // default GEM port
	}

	command := fmt.Sprintf(
		"service-port add vlan %d gpon-olt_1/%d ont %d gemport %d",
		params.VLANID, params.PONPortIndex, params.ONTIndex, gemPort,
	)

	response, err := a.cliConn.Execute(ctx, a.cliCfg, command)
	if err != nil {
		return &domain.ProvisioningResult{
			Success:      false,
			CommandsSent: []string{command},
			ErrorMessage: fmt.Sprintf("gagal menambahkan service-port: %v", err),
		}, nil
	}

	return &domain.ProvisioningResult{
		Success:      true,
		CommandsSent: []string{command},
		Responses:    []string{response},
	}, nil
}

// RemoveServicePort menghapus service-port dari OLT ZTE via CLI.
// Menggunakan command no service-port berdasarkan VLAN, port, dan ONT index.
func (a *ZTEAdapter) RemoveServicePort(ctx context.Context, params domain.RemoveServicePortParams) (*domain.ProvisioningResult, error) {
	command := fmt.Sprintf(
		"no service-port vlan %d gpon-olt_1/%d ont %d",
		params.VLANID, params.PONPortIndex, params.ONTIndex,
	)

	response, err := a.cliConn.Execute(ctx, a.cliCfg, command)
	if err != nil {
		return &domain.ProvisioningResult{
			Success:      false,
			CommandsSent: []string{command},
			ErrorMessage: fmt.Sprintf("gagal menghapus service-port: %v", err),
		}, nil
	}

	return &domain.ProvisioningResult{
		Success:      true,
		CommandsSent: []string{command},
		Responses:    []string{response},
	}, nil
}

// RebootONT mengirim perintah reboot ke ONT tertentu via CLI ZTE.
// Navigasi ke interface gpon-olt, lalu kirim command onu reset.
func (a *ZTEAdapter) RebootONT(ctx context.Context, params domain.RebootONTParams) (*domain.ProvisioningResult, error) {
	commands := []string{
		fmt.Sprintf("interface gpon-olt_1/%d", params.PONPortIndex),
		fmt.Sprintf("onu reset %d", params.ONTIndex),
		"exit",
	}

	responses, err := a.cliConn.ExecuteMultiple(ctx, a.cliCfg, commands)
	if err != nil {
		return &domain.ProvisioningResult{
			Success:      false,
			CommandsSent: commands,
			ErrorMessage: fmt.Sprintf("gagal reboot ONT: %v", err),
		}, nil
	}

	return &domain.ProvisioningResult{
		Success:      true,
		CommandsSent: commands,
		Responses:    responses,
	}, nil
}

// GetUnregisteredONTs mengambil daftar ONT yang terdeteksi tapi belum terdaftar via CLI ZTE.
// Menggunakan command show gpon onu uncfg untuk mendapatkan ONT unregistered.
func (a *ZTEAdapter) GetUnregisteredONTs(ctx context.Context) ([]domain.UnregisteredONT, error) {
	response, err := a.cliConn.Execute(ctx, a.cliCfg, "show gpon onu uncfg")
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil ONT unregistered: %w", err)
	}

	return zteParseUnregisteredONTs(response), nil
}

// zteParseUnregisteredONTs mem-parse output CLI show gpon onu uncfg.
// Format output ZTE: "gpon-olt_1/{port}:{ontIdx}  {serialNumber}"
func zteParseUnregisteredONTs(output string) []domain.UnregisteredONT {
	var result []domain.UnregisteredONT
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || !strings.Contains(line, "gpon-olt_1/") {
			continue
		}

		var port, ontIdx int
		var sn string
		_, err := fmt.Sscanf(line, "gpon-olt_1/%d:%d %s", &port, &ontIdx, &sn)
		if err != nil {
			continue
		}

		result = append(result, domain.UnregisteredONT{
			SerialNumber: sn,
			PONPortIndex: port,
			ONTIndex:     ontIdx,
		})
	}
	return result
}
