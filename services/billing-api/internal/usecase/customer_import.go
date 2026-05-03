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
)

// importTemplateHeaders adalah kolom-kolom yang ada di template import CSV.
var importTemplateHeaders = []string{
	"name",
	"phone",
	"email",
	"address",
	"area_id",
	"latitude",
	"longitude",
	"package_id",
	"activation_date",
	"due_date",
	"connection_method",
	"pppoe_username",
	"pppoe_password",
	"mac_address",
	"router_id",
	"odp_port",
	"notes",
}

// importTemplateExample adalah contoh baris data di template import CSV.
var importTemplateExample = []string{
	"Ahmad Rizki",
	"+6281234567890",
	"ahmad@example.com",
	"Jl. Merdeka No. 1, Jakarta",
	"",
	"-6.2088",
	"106.8456",
	"00000000-0000-0000-0000-000000000001",
	"2024-01-15",
	"10",
	"pppoe",
	"",
	"",
	"",
	"",
	"",
	"Pelanggan baru",
}

// ImportCSV memvalidasi file dan mengirim job import ke queue.
// Flow: validate file type → enqueue asynq job (customer.import) → return job_id.
func (uc *CustomerUsecase) ImportCSV(ctx context.Context, tenantID string, file []byte, filename string, actor ActorInfo) (string, error) {
	// Validate file type
	ext := strings.ToLower(filepath.Ext(filename))
	if ext != ".csv" && ext != ".xlsx" && ext != ".xls" {
		return "", fmt.Errorf("usecase: format file tidak didukung, gunakan CSV atau Excel (.csv, .xlsx)")
	}

	if len(file) == 0 {
		return "", fmt.Errorf("usecase: file kosong")
	}

	// Encode file content for the job payload
	payload := map[string]interface{}{
		"filename":  filename,
		"file_size": len(file),
		"actor_id":  actor.ID,
		"actor_name": actor.Name,
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

	// Use the correlation ID as job_id
	// Note: EnqueueTask auto-generates a correlation ID
	// We return a generated job ID for tracking
	jobID := envelope.CorrelationID
	if jobID == "" {
		jobID = "import-" + tenantID
	}

	// Write audit log
	uc.writeAuditLog(ctx, tenantID, "", "customer.import_started", actor, map[string]interface{}{
		"filename":  filename,
		"file_size": len(file),
	})

	return jobID, nil
}

// GetImportTemplate mengembalikan CSV bytes dengan header kolom dan satu baris contoh.
func (uc *CustomerUsecase) GetImportTemplate(_ context.Context) ([]byte, error) {
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)

	// Write header row
	if err := writer.Write(importTemplateHeaders); err != nil {
		return nil, fmt.Errorf("usecase: gagal menulis header template: %w", err)
	}

	// Write example row
	if err := writer.Write(importTemplateExample); err != nil {
		return nil, fmt.Errorf("usecase: gagal menulis contoh template: %w", err)
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, fmt.Errorf("usecase: gagal flush template CSV: %w", err)
	}

	return buf.Bytes(), nil
}
