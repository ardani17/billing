package usecase

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
	"pgregory.net/rapid"
)

// =============================================================================
// **Memvalidasi: Kebutuhan 5.2, 5.6**
//
// Tidak ada baris yang hilang atau terduplikasi.
// =============================================================================

// buildCSV membangun CSV bytes dari daftar baris.
func buildCSV(rows [][]string) []byte {
	var sb strings.Builder
	sb.WriteString("sn_ont,pelanggan_id,pon_port,vlan,odp,deskripsi\n")
	for _, row := range rows {
		sb.WriteString(strings.Join(row, ","))
		sb.WriteString("\n")
	}
	return []byte(sb.String())
}

// TestProperty2_BulkCountInvariant memverifikasi bahwa setelah ExecuteBulk,
//
// **Memvalidasi: Kebutuhan 5.2, 5.6**
func TestProperty2_BulkCountInvariant(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		// Buat 1-10 baris CSV
		numRows := rapid.IntRange(1, 10).Draw(rt, "numRows")

		var rows [][]string
		for i := 0; i < numRows; i++ {
			sn := fmt.Sprintf("ZTEG%08d", i+1)
			customerID := fmt.Sprintf("cust-%04d", i+1)
			ponPort := fmt.Sprintf("%d", rapid.IntRange(0, 7).Draw(rt, fmt.Sprintf("port_%d", i)))
			rows = append(rows, []string{sn, customerID, ponPort, "vlan-001", "", "test"})
		}

		csvData := buildCSV(rows)

		mgr, _, _, _, _, _, _ := newTestProvisioningManager()
		ctx := context.Background()

		// ValidateBulk
		preview, err := mgr.ValidateBulk(ctx, "tenant-001", "olt-001", csvData)
		if err != nil {
			t.Fatalf("ValidateBulk gagal: %v", err)
		}

		// Verifikasi total rows
		if preview.TotalRows != numRows {
			t.Fatalf("TotalRows=%d, want=%d", preview.TotalRows, numRows)
		}

		if preview.ValidCount+preview.ErrorCount != preview.TotalRows {
			t.Fatalf("ValidCount(%d) + ErrorCount(%d) != TotalRows(%d)",
				preview.ValidCount, preview.ErrorCount, preview.TotalRows)
		}

		// ExecuteBulk
		result, err := mgr.ExecuteBulk(ctx, preview.BulkID, "admin@test.com")
		if err != nil {
			t.Fatalf("ExecuteBulk gagal: %v", err)
		}

		if result.SuccessCount+result.FailureCount != result.Total {
			t.Fatalf("INVARIANT VIOLATED: SuccessCount(%d) + FailureCount(%d) != Total(%d)",
				result.SuccessCount, result.FailureCount, result.Total)
		}

		// Verifikasi jumlah row results == total
		if len(result.Rows) != result.Total {
			t.Fatalf("len(Rows)=%d != Total=%d", len(result.Rows), result.Total)
		}
	})
}

// =============================================================================
// Unit Tests - Bulk Provisioning
// =============================================================================

func TestValidateBulk_ValidCSV(t *testing.T) {
	mgr, _, _, _, _, _, _ := newTestProvisioningManager()
	ctx := context.Background()

	csvData := buildCSV([][]string{
		{"ZTEG00000001", "customer-001", "0", "vlan-001", "", "ONT 1"},
		{"ZTEG00000002", "customer-002", "1", "vlan-001", "", "ONT 2"},
	})

	preview, err := mgr.ValidateBulk(ctx, "tenant-001", "olt-001", csvData)
	if err != nil {
		t.Fatalf("ValidateBulk gagal: %v", err)
	}

	if preview.TotalRows != 2 {
		t.Errorf("TotalRows=%d, want 2", preview.TotalRows)
	}
}

func TestValidateBulk_InvalidCSV(t *testing.T) {
	mgr, _, _, _, _, _, _ := newTestProvisioningManager()
	ctx := context.Background()

	// CSV tanpa header yang cukup
	csvData := []byte("a,b\n1,2\n")

	_, err := mgr.ValidateBulk(ctx, "tenant-001", "olt-001", csvData)
	if err != domain.ErrInvalidCSVFormat {
		t.Errorf("expected ErrInvalidCSVFormat, got: %v", err)
	}
}

// TestValidateBulk_EmptySN memverifikasi validasi serial number kosong.
func TestValidateBulk_EmptySN(t *testing.T) {
	mgr, _, _, _, _, _, _ := newTestProvisioningManager()
	ctx := context.Background()

	csvData := buildCSV([][]string{
		{"", "customer-001", "0", "vlan-001", "", "ONT tanpa SN"},
	})

	preview, err := mgr.ValidateBulk(ctx, "tenant-001", "olt-001", csvData)
	if err != nil {
		t.Fatalf("ValidateBulk gagal: %v", err)
	}

	if preview.ErrorCount != 1 {
		t.Errorf("ErrorCount=%d, want 1", preview.ErrorCount)
	}
	if preview.Rows[0].Valid {
		t.Error("baris dengan SN kosong harus invalid")
	}
}

func TestExecuteBulk_NotFound(t *testing.T) {
	mgr, _, _, _, _, _, _ := newTestProvisioningManager()
	ctx := context.Background()

	_, err := mgr.ExecuteBulk(ctx, "nonexistent-bulk-id", "admin@test.com")
	if err != domain.ErrBulkNotFound {
		t.Errorf("expected ErrBulkNotFound, got: %v", err)
	}
}

// TestGetBulkTemplate memverifikasi template CSV dikembalikan dengan benar.
func TestGetBulkTemplate(t *testing.T) {
	mgr, _, _, _, _, _, _ := newTestProvisioningManager()

	template := mgr.GetBulkTemplate()
	if len(template) == 0 {
		t.Error("template CSV kosong")
	}

	content := string(template)
	if !strings.Contains(content, "sn_ont") {
		t.Error("template harus mengandung header sn_ont")
	}
	if !strings.Contains(content, "pelanggan_id") {
		t.Error("template harus mengandung header pelanggan_id")
	}
}
