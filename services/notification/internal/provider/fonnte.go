package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ispboss/ispboss/services/notification/internal/domain"
)

// =============================================================================
// FonnteAdapter - adapter untuk pengiriman pesan WhatsApp via Fonnte API
// =============================================================================

// fonnteDefaultURL adalah URL bawaan endpoint Fonnte untuk pengiriman pesan.
const fonnteDefaultURL = "https://api.fonnte.com/send"

// FonnteAdapter mengimplementasikan domain.WhatsAppProvider menggunakan Fonnte HTTP API.
// Mengirim pesan WhatsApp melalui endpoint POST https://api.fonnte.com/send.
type FonnteAdapter struct {
	httpClient *http.Client
	apiToken   string
	baseURL    string
}

// NewFonnteAdapter membuat instance baru FonnteAdapter dengan token API dan timeout.
func NewFonnteAdapter(apiToken string, timeout time.Duration) *FonnteAdapter {
	return &FonnteAdapter{
		httpClient: &http.Client{
			Timeout: timeout,
		},
		apiToken: apiToken,
		baseURL:  fonnteDefaultURL,
	}
}

// fonnteResponse merepresentasikan struktur respons JSON dari Fonnte API.
type fonnteResponse struct {
	Status bool   `json:"status"`
	ID     string `json:"id"`
	Detail string `json:"detail"`
}

// Send mengirim pesan WhatsApp ke penerima melalui Fonnte API.
// Mengembalikan SendResult dengan status "sent" jika berhasil atau "failed" jika gagal.
func (a *FonnteAdapter) Send(ctx context.Context, req domain.WhatsAppMessage) (domain.SendResult, error) {
	// Siapkan form body: target dan message
	formData := url.Values{}
	formData.Set("target", req.Recipient)
	formData.Set("message", req.Body)

	// Buat HTTP permintaan dengan context untuk mendukung timeout dan cancellation
	httpReq, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		a.baseURL,
		strings.NewReader(formData.Encode()),
	)
	if err != nil {
		return domain.SendResult{
			Status:      "failed",
			ErrorDetail: fmt.Sprintf("gagal membuat request: %v", err),
		}, fmt.Errorf("gagal membuat request fonnte: %w", err)
	}

	// Set header Authorization dan Content-Type
	httpReq.Header.Set("Authorization", a.apiToken)
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Kirim permintaan ke Fonnte API
	resp, err := a.httpClient.Do(httpReq)
	if err != nil {
		return domain.SendResult{
			Status:      "failed",
			ErrorDetail: fmt.Sprintf("gagal mengirim request: %v", err),
		}, fmt.Errorf("gagal mengirim request fonnte: %w", err)
	}
	defer resp.Body.Close()

	// Baca respons body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return domain.SendResult{
			Status:      "failed",
			ErrorDetail: fmt.Sprintf("gagal membaca response: %v", err),
		}, fmt.Errorf("gagal membaca response fonnte: %w", err)
	}

	// Tangani status code non-200
	if resp.StatusCode != http.StatusOK {
		detail := fmt.Sprintf("status HTTP %d: %s", resp.StatusCode, string(body))
		return domain.SendResult{
			Status:      "failed",
			ErrorDetail: detail,
		}, fmt.Errorf("fonnte API error: %s", detail)
	}

	// Parsing respons JSON dari Fonnte
	var fonnteResp fonnteResponse
	if err := json.Unmarshal(body, &fonnteResp); err != nil {
		detail := fmt.Sprintf("gagal parse response JSON: %v", err)
		return domain.SendResult{
			Status:      "failed",
			ErrorDetail: detail,
		}, fmt.Errorf("gagal parse response fonnte: %w", err)
	}

	// Evaluasi status dari respons Fonnte
	if !fonnteResp.Status {
		return domain.SendResult{
			MessageID:   fonnteResp.ID,
			Status:      "failed",
			ErrorDetail: fonnteResp.Detail,
		}, nil
	}

	return domain.SendResult{
		MessageID: fonnteResp.ID,
		Status:    "sent",
	}, nil
}
