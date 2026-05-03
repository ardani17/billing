// customer_export.go berisi business logic untuk export pelanggan ke CSV/Excel.
// Mengimplementasikan ExportCSV pada CustomerUsecase.
package usecase

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ispboss/ispboss/pkg/queue"
	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

// ExportCSV mengirim job export ke queue.
// Flow: enqueue asynq job (customer.export) with filter params, format,
// and optional columns list → return job_id.
func (uc *CustomerUsecase) ExportCSV(ctx context.Context, tenantID string, params domain.CustomerListParams, format string, columns []string, actor ActorInfo) (string, error) {
	// Default format to csv
	if format == "" {
		format = "csv"
	}

	// Build export job payload
	payload := map[string]interface{}{
		"format":     format,
		"actor_id":   actor.ID,
		"actor_name": actor.Name,
	}

	// Add filter params
	if params.Search != "" {
		payload["search"] = params.Search
	}
	if params.Status != "" {
		payload["status"] = params.Status
	}
	if params.PackageID != "" {
		payload["package_id"] = params.PackageID
	}
	if params.AreaID != "" {
		payload["area_id"] = params.AreaID
	}
	if params.DueDate != nil {
		payload["due_date"] = *params.DueDate
	}

	// Add columns if specified
	if len(columns) > 0 {
		payload["columns"] = columns
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("usecase: gagal marshal export payload: %w", err)
	}

	// Enqueue asynq job
	envelope := queue.TaskEnvelope{
		EventType: "customer.export",
		TenantID:  tenantID,
		Payload:   payloadJSON,
	}

	if err := queue.EnqueueTask(uc.queueClient, envelope); err != nil {
		return "", fmt.Errorf("usecase: gagal enqueue export job: %w", err)
	}

	// Use the correlation ID as job_id
	jobID := envelope.CorrelationID
	if jobID == "" {
		jobID = "export-" + tenantID
	}

	// Write audit log
	uc.writeAuditLog(ctx, tenantID, "", "customer.export_started", actor, map[string]interface{}{
		"format":  format,
		"columns": columns,
	})

	return jobID, nil
}
