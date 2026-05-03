package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ispboss/ispboss/services/notification/internal/domain"
)

// =============================================================================
// Test FonnteAdapter — integration test dengan mock HTTP server
// =============================================================================

// newTestFonnteAdapter membuat FonnteAdapter yang mengarah ke mock server URL.
func newTestFonnteAdapter(serverURL, apiToken string) *FonnteAdapter {
	adapter := NewFonnteAdapter(apiToken, 5*time.Second)
	adapter.baseURL = serverURL
	return adapter
}

// TestFonnteAdapter_Send_Success menguji pengiriman berhasil via Fonnte API.
func TestFonnteAdapter_Send_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verifikasi method dan header
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("Authorization") != "test-token-123" {
			t.Errorf("expected Authorization 'test-token-123', got '%s'", r.Header.Get("Authorization"))
		}
		if r.Header.Get("Content-Type") != "application/x-www-form-urlencoded" {
			t.Errorf("expected Content-Type form-urlencoded, got '%s'", r.Header.Get("Content-Type"))
		}

		// Verifikasi form body
		if err := r.ParseForm(); err != nil {
			t.Fatalf("gagal parse form: %v", err)
		}
		if r.FormValue("target") != "628123456789" {
			t.Errorf("expected target '628123456789', got '%s'", r.FormValue("target"))
		}
		if r.FormValue("message") != "Halo, ini pesan test" {
			t.Errorf("expected message 'Halo, ini pesan test', got '%s'", r.FormValue("message"))
		}

		// Kirim response sukses
		resp := fonnteResponse{Status: true, ID: "msg-001", Detail: "sent"}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	adapter := newTestFonnteAdapter(server.URL, "test-token-123")
	result, err := adapter.Send(context.Background(), domain.WhatsAppMessage{
		Recipient: "628123456789",
		Body:      "Halo, ini pesan test",
	})

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if result.Status != "sent" {
		t.Errorf("expected status 'sent', got '%s'", result.Status)
	}
	if result.MessageID != "msg-001" {
		t.Errorf("expected message ID 'msg-001', got '%s'", result.MessageID)
	}
	if result.ErrorDetail != "" {
		t.Errorf("expected empty error detail, got '%s'", result.ErrorDetail)
	}
}

// TestFonnteAdapter_Send_APIFailure menguji response gagal dari Fonnte (status: false).
func TestFonnteAdapter_Send_APIFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := fonnteResponse{Status: false, ID: "", Detail: "nomor tidak valid"}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	adapter := newTestFonnteAdapter(server.URL, "test-token")
	result, err := adapter.Send(context.Background(), domain.WhatsAppMessage{
		Recipient: "invalid-number",
		Body:      "Test",
	})

	// API failure bukan transport error, jadi err harus nil
	if err != nil {
		t.Fatalf("expected no error (API failure bukan transport error), got: %v", err)
	}
	if result.Status != "failed" {
		t.Errorf("expected status 'failed', got '%s'", result.Status)
	}
	if result.ErrorDetail != "nomor tidak valid" {
		t.Errorf("expected error detail 'nomor tidak valid', got '%s'", result.ErrorDetail)
	}
}

// TestFonnteAdapter_Send_HTTPError menguji penanganan status code non-200.
func TestFonnteAdapter_Send_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal server error"))
	}))
	defer server.Close()

	adapter := newTestFonnteAdapter(server.URL, "test-token")
	result, err := adapter.Send(context.Background(), domain.WhatsAppMessage{
		Recipient: "628123456789",
		Body:      "Test",
	})

	if err == nil {
		t.Fatal("expected error for HTTP 500, got nil")
	}
	if result.Status != "failed" {
		t.Errorf("expected status 'failed', got '%s'", result.Status)
	}
}

// TestFonnteAdapter_Send_InvalidJSON menguji penanganan response JSON tidak valid.
func TestFonnteAdapter_Send_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("bukan json yang valid"))
	}))
	defer server.Close()

	adapter := newTestFonnteAdapter(server.URL, "test-token")
	result, err := adapter.Send(context.Background(), domain.WhatsAppMessage{
		Recipient: "628123456789",
		Body:      "Test",
	})

	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
	if result.Status != "failed" {
		t.Errorf("expected status 'failed', got '%s'", result.Status)
	}
}

// TestFonnteAdapter_Send_Timeout menguji penanganan timeout dari server.
func TestFonnteAdapter_Send_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	adapter := newTestFonnteAdapter(server.URL, "test-token")
	adapter.httpClient.Timeout = 100 * time.Millisecond

	result, err := adapter.Send(context.Background(), domain.WhatsAppMessage{
		Recipient: "628123456789",
		Body:      "Test",
	})

	if err == nil {
		t.Fatal("expected error for timeout, got nil")
	}
	if result.Status != "failed" {
		t.Errorf("expected status 'failed', got '%s'", result.Status)
	}
}
