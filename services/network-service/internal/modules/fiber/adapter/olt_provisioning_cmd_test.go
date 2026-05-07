// untuk setiap operasi provisioning (AddONT, AddServicePort, dll).
package adapter

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
	"pgregory.net/rapid"
)

// =============================================================================
// **Memvalidasi: Kebutuhan 4.1, 4.2, 4.3, 4.4, 4.5, 4.7**
//
// CommandsSent non-empty, dan command mengandung serial number.
// Untuk AddServicePortParams (VLANID > 0), command mengandung VLAN ID.
// =============================================================================

// serialNumberGen menghasilkan serial number ONT acak yang realistis.
// Format: 4 huruf besar + 8 hex digit (contoh: ZTEG01234567).
func serialNumberGen() *rapid.Generator[string] {
	return rapid.StringMatching(`[A-Z]{4}[0-9a-fA-F]{8}`)
}

func ponPortGen() *rapid.Generator[int] {
	return rapid.IntRange(0, 15)
}

func ontIndexGen() *rapid.Generator[int] {
	return rapid.IntRange(1, 128)
}

func profileIDGen() *rapid.Generator[int] {
	return rapid.IntRange(1, 64)
}

func vlanIDGen() *rapid.Generator[int] {
	return rapid.IntRange(1, 4094)
}

func gemPortGen() *rapid.Generator[int] {
	return rapid.IntRange(1, 8)
}

// Menyimpan command yang dikirim tanpa koneksi jaringan.
type provMockCLIConnector struct {
	lastCommands []string
}

func (m *provMockCLIConnector) Execute(_ context.Context, _ domain.CLIConfig, command string) (string, error) {
	m.lastCommands = append(m.lastCommands, command)
	return fmt.Sprintf("OK: %s", command), nil
}

func (m *provMockCLIConnector) ExecuteMultiple(_ context.Context, _ domain.CLIConfig, commands []string) ([]string, error) {
	m.lastCommands = append(m.lastCommands, commands...)
	responses := make([]string, len(commands))
	for i := range commands {
		responses[i] = fmt.Sprintf("OK: %s", commands[i])
	}
	return responses, nil
}

// TestConnection mengembalikan banner simulasi.
func (m *provMockCLIConnector) TestConnection(_ context.Context, _ domain.CLIConfig) (string, error) {
	return "ZTE C320>", nil
}

// TestProperty1_AddONT_ProducesValidCommands memverifikasi bahwa untuk
// non-empty yang mengandung serial number.
//
// **Memvalidasi: Kebutuhan 4.1, 4.7**
func TestProperty1_AddONT_ProducesValidCommands(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		sn := serialNumberGen().Draw(rt, "serialNumber")
		ponPort := ponPortGen().Draw(rt, "ponPort")
		ontIdx := ontIndexGen().Draw(rt, "ontIndex")
		lineProfile := profileIDGen().Draw(rt, "lineProfile")
		srvProfile := profileIDGen().Draw(rt, "srvProfile")

		params := domain.AddONTParams{
			PONPortIndex:     ponPort,
			ONTIndex:         ontIdx,
			SerialNumber:     sn,
			LineProfileID:    lineProfile,
			ServiceProfileID: srvProfile,
			Description:      "test ONT",
		}

		cliMock := &provMockCLIConnector{}
		adapter := NewZTEAdapter(nil, cliMock, domain.SNMPConfig{}, domain.CLIConfig{})

		result, err := adapter.AddONT(context.Background(), params)
		if err != nil {
			t.Fatalf("AddONT error: %v", err)
		}

		// CommandsSent harus non-empty
		if len(result.CommandsSent) == 0 {
			t.Fatal("AddONT: CommandsSent kosong, seharusnya berisi CLI commands")
		}

		// Gabungkan semua command untuk pengecekan
		allCmds := strings.Join(result.CommandsSent, " ")

		// Command harus mengandung serial number
		if !strings.Contains(allCmds, sn) {
			t.Errorf("AddONT: command tidak mengandung serial number %q\nCommands: %v",
				sn, result.CommandsSent)
		}

		if !result.Success {
			t.Errorf("AddONT: result.Success=false, error=%s", result.ErrorMessage)
		}
	})
}

// TestProperty1_AddServicePort_ProducesValidCommands memverifikasi bahwa
// command non-empty yang mengandung VLAN ID.
//
// **Memvalidasi: Kebutuhan 4.2, 4.7**
func TestProperty1_AddServicePort_ProducesValidCommands(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		ponPort := ponPortGen().Draw(rt, "ponPort")
		ontIdx := ontIndexGen().Draw(rt, "ontIndex")
		vlanID := vlanIDGen().Draw(rt, "vlanID")
		gemPort := gemPortGen().Draw(rt, "gemPort")

		params := domain.AddServicePortParams{
			PONPortIndex: ponPort,
			ONTIndex:     ontIdx,
			VLANID:       vlanID,
			GemPort:      gemPort,
		}

		cliMock := &provMockCLIConnector{}
		adapter := NewZTEAdapter(nil, cliMock, domain.SNMPConfig{}, domain.CLIConfig{})

		result, err := adapter.AddServicePort(context.Background(), params)
		if err != nil {
			t.Fatalf("AddServicePort error: %v", err)
		}

		// CommandsSent harus non-empty
		if len(result.CommandsSent) == 0 {
			t.Fatal("AddServicePort: CommandsSent kosong")
		}

		// Gabungkan semua command untuk pengecekan
		allCmds := strings.Join(result.CommandsSent, " ")

		// Command harus mengandung VLAN ID
		vlanStr := fmt.Sprintf("%d", vlanID)
		if !strings.Contains(allCmds, vlanStr) {
			t.Errorf("AddServicePort: command tidak mengandung VLAN ID %d\nCommands: %v",
				vlanID, result.CommandsSent)
		}

		if !result.Success {
			t.Errorf("AddServicePort: result.Success=false, error=%s", result.ErrorMessage)
		}
	})
}

// TestProperty1_RemoveONT_ProducesValidCommands memverifikasi bahwa
// RemoveONT menghasilkan command non-empty untuk ZTE adapter.
//
// **Memvalidasi: Kebutuhan 4.3, 4.7**
func TestProperty1_RemoveONT_ProducesValidCommands(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		ponPort := ponPortGen().Draw(rt, "ponPort")
		ontIdx := ontIndexGen().Draw(rt, "ontIndex")

		params := domain.RemoveONTParams{
			PONPortIndex: ponPort,
			ONTIndex:     ontIdx,
		}

		cliMock := &provMockCLIConnector{}
		adapter := NewZTEAdapter(nil, cliMock, domain.SNMPConfig{}, domain.CLIConfig{})

		result, err := adapter.RemoveONT(context.Background(), params)
		if err != nil {
			t.Fatalf("RemoveONT error: %v", err)
		}

		if len(result.CommandsSent) == 0 {
			t.Fatal("RemoveONT: CommandsSent kosong")
		}

		if !result.Success {
			t.Errorf("RemoveONT: result.Success=false, error=%s", result.ErrorMessage)
		}
	})
}

// TestProperty1_RebootONT_ProducesValidCommands memverifikasi bahwa
// RebootONT menghasilkan command non-empty untuk ZTE adapter.
//
// **Memvalidasi: Kebutuhan 4.5, 4.7**
func TestProperty1_RebootONT_ProducesValidCommands(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		ponPort := ponPortGen().Draw(rt, "ponPort")
		ontIdx := ontIndexGen().Draw(rt, "ontIndex")

		params := domain.RebootONTParams{
			PONPortIndex: ponPort,
			ONTIndex:     ontIdx,
		}

		cliMock := &provMockCLIConnector{}
		adapter := NewZTEAdapter(nil, cliMock, domain.SNMPConfig{}, domain.CLIConfig{})

		result, err := adapter.RebootONT(context.Background(), params)
		if err != nil {
			t.Fatalf("RebootONT error: %v", err)
		}

		if len(result.CommandsSent) == 0 {
			t.Fatal("RebootONT: CommandsSent kosong")
		}

		if !result.Success {
			t.Errorf("RebootONT: result.Success=false, error=%s", result.ErrorMessage)
		}
	})
}
