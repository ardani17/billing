// webhook_handler_test.go berisi unit test untuk endpoint webhook publik:
// HandleXendit dan HandleMidtrans.
package handler

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/hibiken/asynq"
	"github.com/rs/zerolog"
)

// setupWebhookTestApp membuat Fiber app dengan WebhookHandler untuk testing.
func setupWebhookTestApp(xenditIPs, midtransIPs []string) (*fiber.App, *mockWebhookLogRepo) {
	mr, _ := miniredis.Run()
	client := asynq.NewClient(asynq.RedisClientOpt{Addr: mr.Addr()})
	repo := newMockWebhookLogRepo()
	logger := zerolog.New(io.Discard)
	h := NewWebhookHandler(repo, client, xenditIPs, midtransIPs, logger)
	app := fiber.New()
	app.Post("/webhooks/xendit", h.HandleXendit)
	app.Post("/webhooks/midtrans", h.HandleMidtrans)
	return app, repo
}

func postWebhook(app *fiber.App, path string, payload map[string]interface{}) (*fiber.Response, int) {
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", path, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req, -1)
	return nil, resp.StatusCode
}

func findLog(repo *mockWebhookLogRepo, extID string) (string, bool) {
	for _, l := range repo.logs {
		if l.ExternalID == extID {
			return l.EventType, true
		}
	}
	return "", false
}

func TestWebhookHandler_HandleXendit_ValidIP(t *testing.T) {
	app, repo := setupWebhookTestApp([]string{"0.0.0.0"}, nil)
	body, _ := json.Marshal(map[string]interface{}{
		"external_id": "pl-abc-123", "status": "PAID", "amount": 150000,
	})
	req := httptest.NewRequest("POST", "/webhooks/xendit", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-callback-token", "secret-token")
	resp, _ := app.Test(req, -1)
	if resp.StatusCode != fiber.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(b))
	}
	// Verifikasi webhook log tersimpan dengan field yang benar
	evt, ok := findLog(repo, "pl-abc-123")
	if !ok {
		t.Fatal("webhook log dengan external_id=pl-abc-123 tidak ditemukan")
	}
	if evt != "payment.paid" {
		t.Errorf("expected event_type=payment.paid, got %s", evt)
	}
}

func TestWebhookHandler_HandleXendit_InvalidIP(t *testing.T) {
	app, repo := setupWebhookTestApp([]string{"10.0.0.1"}, nil)
	_, code := postWebhook(app, "/webhooks/xendit", map[string]interface{}{
		"external_id": "pl-xyz", "status": "PAID",
	})
	if code != fiber.StatusForbidden {
		t.Fatalf("expected 403, got %d", code)
	}
	// Verifikasi log blocked tersimpan
	for _, l := range repo.logs {
		if l.ErrorMessage == "ip_not_whitelisted" {
			return
		}
	}
	t.Fatal("expected log blocked dengan error_message=ip_not_whitelisted")
}

func TestWebhookHandler_HandleMidtrans_ValidIP(t *testing.T) {
	app, repo := setupWebhookTestApp(nil, []string{"0.0.0.0"})
	_, code := postWebhook(app, "/webhooks/midtrans", map[string]interface{}{
		"order_id": "pl-mid-001", "transaction_status": "settlement",
	})
	if code != fiber.StatusOK {
		t.Fatalf("expected 200, got %d", code)
	}
	evt, ok := findLog(repo, "pl-mid-001")
	if !ok {
		t.Fatal("webhook log midtrans tidak ditemukan")
	}
	if evt != "payment.paid" {
		t.Errorf("expected payment.paid, got %s", evt)
	}
}

func TestWebhookHandler_HandleMidtrans_InvalidIP(t *testing.T) {
	app, _ := setupWebhookTestApp(nil, []string{"10.0.0.1"})
	_, code := postWebhook(app, "/webhooks/midtrans", map[string]interface{}{
		"order_id": "pl-mid-002", "transaction_status": "settlement",
	})
	if code != fiber.StatusForbidden {
		t.Fatalf("expected 403, got %d", code)
	}
}

func TestWebhookHandler_EmptyWhitelist_SkipsIPCheck(t *testing.T) {
	app, _ := setupWebhookTestApp(nil, nil)
	// Xendit tanpa whitelist -> harus 200
	_, code := postWebhook(app, "/webhooks/xendit", map[string]interface{}{
		"external_id": "pl-no-wl-1", "status": "EXPIRED",
	})
	if code != fiber.StatusOK {
		t.Fatalf("xendit: expected 200, got %d", code)
	}
	// Midtrans tanpa whitelist -> harus 200
	_, code2 := postWebhook(app, "/webhooks/midtrans", map[string]interface{}{
		"order_id": "pl-no-wl-2", "transaction_status": "capture",
	})
	if code2 != fiber.StatusOK {
		t.Fatalf("midtrans: expected 200, got %d", code2)
	}
}

func TestWebhookHandler_Xendit_ExtractsFields(t *testing.T) {
	app, repo := setupWebhookTestApp(nil, nil)
	cases := []struct {
		status, wantEvent string
	}{
		{"PAID", "payment.paid"},
		{"EXPIRED", "payment.expired"},
		{"FAILED", "payment.failed"},
	}
	for _, tc := range cases {
		postWebhook(app, "/webhooks/xendit", map[string]interface{}{
			"external_id": "ext-" + tc.status, "status": tc.status,
		})
	}
	for _, tc := range cases {
		evt, ok := findLog(repo, "ext-"+tc.status)
		if !ok {
			t.Errorf("xendit %s: log tidak ditemukan", tc.status)
			continue
		}
		if evt != tc.wantEvent {
			t.Errorf("xendit %s: expected %s, got %s", tc.status, tc.wantEvent, evt)
		}
	}
}

func TestWebhookHandler_Midtrans_ExtractsFields(t *testing.T) {
	app, repo := setupWebhookTestApp(nil, nil)
	cases := []struct {
		txStatus, wantEvent string
	}{
		{"settlement", "payment.paid"},
		{"capture", "payment.paid"},
		{"expire", "payment.expired"},
		{"deny", "payment.failed"},
		{"cancel", "payment.failed"},
	}
	for _, tc := range cases {
		postWebhook(app, "/webhooks/midtrans", map[string]interface{}{
			"order_id": "mid-" + tc.txStatus, "transaction_status": tc.txStatus,
		})
	}
	for _, tc := range cases {
		evt, ok := findLog(repo, "mid-"+tc.txStatus)
		if !ok {
			t.Errorf("midtrans %s: log tidak ditemukan", tc.txStatus)
			continue
		}
		if evt != tc.wantEvent {
			t.Errorf("midtrans %s: expected %s, got %s", tc.txStatus, tc.wantEvent, evt)
		}
	}
}
