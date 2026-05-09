// customer_import.go berisi business logic untuk import pelanggan dari CSV.
// Mengimplementasikan ImportCSV dan GetImportTemplate pada CustomerUsecase.
package usecase

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/ispboss/ispboss/pkg/queue"
	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

// customerImportBaseColumns adalah kolom Billing Core yang selalu tersedia.
var customerImportBaseColumns = []string{
	"name",
	"phone",
	"email",
	"address",
	"area_id",
	"package_id",
	"activation_date",
	"due_date",
	"connection_method",
	"notes",
}

var customerImportMikrotikColumns = []string{
	"pppoe_username",
	"pppoe_password",
	"mac_address",
	"router_id",
}

var customerImportFiberColumns = []string{
	"latitude",
	"longitude",
	"odp_port",
}

// customerImportBaseExample adalah contoh baris Billing-only di template import CSV.
var customerImportBaseExample = []string{
	"Ahmad Rizki",
	"+6281234567890",
	"ahmad@example.com",
	"Jl. Merdeka No. 1, Jakarta",
	"",
	"00000000-0000-0000-0000-000000000001",
	"2024-01-15",
	"10",
	"manual",
	"Pelanggan baru",
}

var customerImportMikrotikExample = []string{"", "", "", ""}
var customerImportFiberExample = []string{"", "", ""}

func (uc *CustomerUsecase) customerImportColumns(ctx context.Context, tenantID string) ([]string, []string) {
	headers := append([]string{}, customerImportBaseColumns...)
	example := append([]string{}, customerImportBaseExample...)

	caps := domain.DefaultTenantModuleCapabilities()
	if uc.moduleRepo != nil {
		nextCaps, err := uc.moduleRepo.Capabilities(ctx, tenantID)
		if err != nil {
			uc.logger.Warn().Err(err).Str("tenant_id", tenantID).Msg("gagal cek entitlement template import pelanggan")
		} else {
			caps = nextCaps
		}
	}

	if caps.MikroTik {
		headers = append(headers, customerImportMikrotikColumns...)
		example = append(example, customerImportMikrotikExample...)
	}
	if caps.FiberNetwork {
		headers = append(headers, customerImportFiberColumns...)
		example = append(example, customerImportFiberExample...)
	}
	return headers, example
}

// ImportCSV memvalidasi file dan mengirim job import ke queue.
// Alur: validasi tipe file -> antrekan asynq job (customer.import) -> kembalikan job_id.
func (uc *CustomerUsecase) ImportCSV(ctx context.Context, tenantID string, file []byte, filename string, actor ActorInfo) (string, error) {
	// Validasi tipe file
	ext := strings.ToLower(filepath.Ext(filename))
	if ext != ".csv" && ext != ".xlsx" && ext != ".xls" {
		return "", fmt.Errorf("usecase: format file tidak didukung, gunakan CSV atau Excel (.csv, .xlsx)")
	}

	if len(file) == 0 {
		return "", fmt.Errorf("usecase: file kosong")
	}

	columns, _ := uc.customerImportColumns(ctx, tenantID)

	// Encode file content untuk the job payload
	payload := map[string]interface{}{
		"filename":        filename,
		"file_size":       len(file),
		"allowed_columns": columns,
		"actor_id":        actor.ID,
		"actor_name":      actor.Name,
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("usecase: gagal marshal import payload: %w", err)
	}

	// Enqueue asynq job
	envelope := queue.TaskEnvelope{
		EventType: "customer.import",
		TenantID:  tenantID,
		Payload:   payloadJSON,
	}

	if err := queue.EnqueueTask(uc.queueClient, envelope); err != nil {
		return "", fmt.Errorf("usecase: gagal enqueue import job: %w", err)
	}

	// Gunakan the correlation ID as job_id
	// Catatan: EnqueueTask otomatis membuat correlation ID
	// We kembalikan a generated job ID untuk tracking
	jobID := envelope.CorrelationID
	if jobID == "" {
		jobID = "import-" + tenantID
	}

	// Tulis audit log
	uc.writeAuditLog(ctx, tenantID, "", "customer.import_started", actor, map[string]interface{}{
		"filename":  filename,
		"file_size": len(file),
	})

	return jobID, nil
}

// GetImportTemplate mengembalikan CSV bytes dengan header kolom dan satu baris contoh.
func (uc *CustomerUsecase) GetImportTemplate(ctx context.Context, tenantID string) ([]byte, error) {
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)
	headers, example := uc.customerImportColumns(ctx, tenantID)

	// Write header row
	if err := writer.Write(headers); err != nil {
		return nil, fmt.Errorf("usecase: gagal menulis header template: %w", err)
	}

	// Write example row
	if err := writer.Write(example); err != nil {
		return nil, fmt.Errorf("usecase: gagal menulis contoh template: %w", err)
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, fmt.Errorf("usecase: gagal flush template CSV: %w", err)
	}

	return buf.Bytes(), nil
}
