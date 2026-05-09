// export_handler_test.go berisi integration tests untuk export endpoints.
package handler

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

func setupExportTestApp(mock *mockReportUsecase) *fiber.App {
	logger := zerolog.New(io.Discard)
	handler := NewExportHandler(mock, logger)

	app := fiber.New()

	setLocals := func(c *fiber.Ctx) error {
		c.Locals("tenant_id", "test-tenant-id")
		c.Locals("user_id", "test-user-id")
		c.Locals("user_name", "Test User")
		return c.Next()
	}

	export := app.Group("/api/v1/reports/export", setLocals)
	export.Post("/", handler.RequestExport)
	export.Get("/:job_id", handler.Status)

	return app
}

func TestExportHandler_RequestExport_AsyncAccepted(t *testing.T) {
	mock := &mockReportUsecase{
		exportJobID: "job-123",
	}
	app := setupExportTestApp(mock)

	body, _ := json.Marshal(domain.ExportRequest{
		ReportType: "revenue",
		Format:     "pdf",
	})

	req := httptest.NewRequest("POST", "/api/v1/reports/export", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusAccepted {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 202, got %d: %s", resp.StatusCode, string(respBody))
	}

	var apiResp domain.APIResponse
	json.NewDecoder(resp.Body).Decode(&apiResp)
	if !apiResp.Success {
		t.Fatal("expected success=true")
	}
}

func TestExportHandler_RequestExport_InvalidBody(t *testing.T) {
	mock := &mockReportUsecase{}
	app := setupExportTestApp(mock)

	req := httptest.NewRequest("POST", "/api/v1/reports/export", bytes.NewReader([]byte("invalid")))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestExportHandler_RequestExport_ValidationError(t *testing.T) {
	mock := &mockReportUsecase{}
	app := setupExportTestApp(mock)

	body, _ := json.Marshal(map[string]interface{}{
		"report_type": "revenue",
		"format":      "invalid_format",
	})

	req := httptest.NewRequest("POST", "/api/v1/reports/export", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusBadRequest {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 400, got %d: %s", resp.StatusCode, string(respBody))
	}
}

func TestExportHandler_RequestExport_InvalidReportType(t *testing.T) {
	mock := &mockReportUsecase{
		err: domain.ErrInvalidReportType,
	}
	app := setupExportTestApp(mock)

	body, _ := json.Marshal(domain.ExportRequest{
		ReportType: "nonexistent_type",
		Format:     "pdf",
	})

	req := httptest.NewRequest("POST", "/api/v1/reports/export", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusBadRequest {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 400, got %d: %s", resp.StatusCode, string(respBody))
	}

	var apiResp domain.APIResponse
	json.NewDecoder(resp.Body).Decode(&apiResp)
	if apiResp.Error == nil || apiResp.Error.Code != "INVALID_REPORT_TYPE" {
		t.Fatalf("expected INVALID_REPORT_TYPE, got %v", apiResp.Error)
	}
}

// --- Tes: Job Status ---

func TestExportHandler_Status_Success(t *testing.T) {
	mock := &mockReportUsecase{
		exportJob: &domain.ReportJob{
			ID:          "job-123",
			TenantID:    "test-tenant-id",
			ReportType:  "revenue",
			Format:      "pdf",
			Status:      domain.JobCompleted,
			DownloadURL: "/exports/test-tenant-id/revenue.pdf",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
	}
	app := setupExportTestApp(mock)

	req := httptest.NewRequest("GET", "/api/v1/reports/export/job-123", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(respBody))
	}

	var apiResp domain.APIResponse
	json.NewDecoder(resp.Body).Decode(&apiResp)
	if !apiResp.Success {
		t.Fatal("expected success=true")
	}
}

func TestExportHandler_Status_NotFound(t *testing.T) {
	mock := &mockReportUsecase{
		err: domain.ErrReportJobNotFound,
	}
	app := setupExportTestApp(mock)

	req := httptest.NewRequest("GET", "/api/v1/reports/export/nonexistent", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}

	var apiResp domain.APIResponse
	json.NewDecoder(resp.Body).Decode(&apiResp)
	if apiResp.Error == nil || apiResp.Error.Code != "JOB_NOT_FOUND" {
		t.Fatalf("expected JOB_NOT_FOUND, got %v", apiResp.Error)
	}
}

func TestExportHandler_RequestExport_Unauthorized(t *testing.T) {
	mock := &mockReportUsecase{}
	logger := zerolog.New(io.Discard)
	handler := NewExportHandler(mock, logger)

	app := fiber.New()
	// Tanpa middleware atur tenant_id
	app.Post("/api/v1/reports/export", handler.RequestExport)

	body, _ := json.Marshal(domain.ExportRequest{
		ReportType: "revenue",
		Format:     "pdf",
	})

	req := httptest.NewRequest("POST", "/api/v1/reports/export", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}
