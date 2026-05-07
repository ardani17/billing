// Package adapter - Implementasi provisioning methods untuk ZTEAdapter.
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
	commands, buildErr := zteBuildAddONTCommands(params)
	if buildErr != nil {
		return nil, buildErr
	}

	responses, err := a.cliConn.ExecuteMultiple(ctx, a.cliCfg, commands)
	if err != nil {
		return zteProvisioningResult(false, "add_ont", commands, nil, fmt.Sprintf("gagal menambahkan ONT: %v", err), params.ONTIndex), nil
	}

	return zteProvisioningResult(true, "add_ont", commands, responses, "", params.ONTIndex), nil
}

// RemoveONT menghapus ONT dari PON port ZTE via CLI.
// Navigasi ke interface gpon-olt, lalu kirim command no onu.
func (a *ZTEAdapter) RemoveONT(ctx context.Context, params domain.RemoveONTParams) (*domain.ProvisioningResult, error) {
	commands, buildErr := zteBuildRemoveONTCommands(params)
	if buildErr != nil {
		return nil, buildErr
	}

	responses, err := a.cliConn.ExecuteMultiple(ctx, a.cliCfg, commands)
	if err != nil {
		return zteProvisioningResult(false, "remove_ont", commands, nil, fmt.Sprintf("gagal menghapus ONT: %v", err), 0), nil
	}

	return zteProvisioningResult(true, "remove_ont", commands, responses, "", 0), nil
}

// AddServicePort menambahkan service-port dengan VLAN assignment via CLI ZTE.
// Command: service-port add vlan {vlan} gpon {port} ont {id} gemport {gem}.
func (a *ZTEAdapter) AddServicePort(ctx context.Context, params domain.AddServicePortParams) (*domain.ProvisioningResult, error) {
	command, buildErr := zteBuildAddServicePortCommand(params)
	if buildErr != nil {
		return nil, buildErr
	}

	response, err := a.cliConn.Execute(ctx, a.cliCfg, command)
	if err != nil {
		return zteProvisioningResult(false, "add_service_port", []string{command}, nil, fmt.Sprintf("gagal menambahkan service-port: %v", err), 0), nil
	}

	return zteProvisioningResult(true, "add_service_port", []string{command}, []string{response}, "", 0), nil
}

// RemoveServicePort menghapus service-port dari OLT ZTE via CLI.
// Menggunakan command no service-port berdasarkan VLAN, port, dan ONT index.
func (a *ZTEAdapter) RemoveServicePort(ctx context.Context, params domain.RemoveServicePortParams) (*domain.ProvisioningResult, error) {
	command, buildErr := zteBuildRemoveServicePortCommand(params)
	if buildErr != nil {
		return nil, buildErr
	}

	response, err := a.cliConn.Execute(ctx, a.cliCfg, command)
	if err != nil {
		return zteProvisioningResult(false, "remove_service_port", []string{command}, nil, fmt.Sprintf("gagal menghapus service-port: %v", err), 0), nil
	}

	return zteProvisioningResult(true, "remove_service_port", []string{command}, []string{response}, "", 0), nil
}

// RebootONT mengirim perintah reboot ke ONT tertentu via CLI ZTE.
// Navigasi ke interface gpon-olt, lalu kirim command onu reset.
func (a *ZTEAdapter) RebootONT(ctx context.Context, params domain.RebootONTParams) (*domain.ProvisioningResult, error) {
	commands, buildErr := zteBuildRebootONTCommands(params)
	if buildErr != nil {
		return nil, buildErr
	}

	responses, err := a.cliConn.ExecuteMultiple(ctx, a.cliCfg, commands)
	if err != nil {
		return zteProvisioningResult(false, "reboot_ont", commands, nil, fmt.Sprintf("gagal reboot ONT: %v", err), 0), nil
	}

	return zteProvisioningResult(true, "reboot_ont", commands, responses, "", 0), nil
}

// PreviewProvisioningCommands membangun urutan command provisioning tanpa membuka koneksi CLI.
func (a *ZTEAdapter) PreviewProvisioningCommands(_ context.Context, add domain.AddONTParams, service domain.AddServicePortParams) (*domain.ProvisioningResult, error) {
	addCommands, err := zteBuildAddONTCommands(add)
	if err != nil {
		return nil, err
	}
	serviceCommand, err := zteBuildAddServicePortCommand(service)
	if err != nil {
		return nil, err
	}
	commands := append([]string{}, addCommands...)
	commands = append(commands, serviceCommand)
	return zteProvisioningResult(true, "provision_ont_preview", commands, nil, "", add.ONTIndex), nil
}

func zteProvisioningResult(success bool, operation string, commands, responses []string, errMsg string, assignedONTIndex int) *domain.ProvisioningResult {
	return &domain.ProvisioningResult{
		Success:          success,
		CommandsSent:     commands,
		Responses:        responses,
		ErrorMessage:     errMsg,
		AssignedONTIndex: assignedONTIndex,
		Brand:            string(domain.BrandZTE),
		Transport:        "cli",
		Operation:        operation,
	}
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

// zteParseUnregisteredONTs mem-parsing output CLI show gpon onu uncfg.
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
