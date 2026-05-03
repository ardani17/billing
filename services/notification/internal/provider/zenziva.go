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
// ZenzivaAdapter — adapter untuk pengiriman pesan SMS via Zenziva API
// =============================================================================

// zenzivaDefaultURL adalah URL default endpoint Zenziva untuk pengiriman SMS reguler.
const zenzivaDefaultURL = "https://console.zenziva.net/reguler/api/sendsms/"

// ZenzivaAdapter mengimplementasikan domain.SMSProvider menggunakan Zenziva HTTP API.
// Mengirim pesan SMS melalui endpoint POST https://console.zenziva.net/reguler/api/sendsms/.
type ZenzivaAdapter struct {
	httpClient *http.Client
	apiKey     string
	userKey    string
	baseURL    string
}

// NewZenzivaAdapter membuat instance baru ZenzivaAdapter dengan API key, user key, dan timeout.
func NewZenzivaAdapter(apiKey, userKey string, timeout time.Duration) *ZenzivaAdapter {
	return &ZenzivaAdapter{
		httpClient: &http.Client{
			Timeout: timeout,
		},
		apiKey:  apiKey,
		userKey: userKey,
		baseURL: zenzivaDefaultURL,
	}
}

// zenzivaResponse merepresentasikan struktur response JSON dari Zenziva API.
type zenzivaResponse struct {
	MessageID string `json:"messageId"`
	Status    int    `json:"status"`
	Text      string `json:"text"`
}

// Send mengirim pesan SMS ke penerima melalui Zenziva API.
// Mengembalikan SendResult dengan status "sent" jika berhasil atau "failed" jika gagal.
func (a *ZenzivaAdapter) Send(ctx context.Context, req domain.SMSMessage) (domain.SendResult, error) {
	// Siapkan form body: userkey, passkey, to, message
	formData := url.Values{}
	formData.Set("userkey", a.userKey)
	formData.Set("passkey", a.apiKey)
	formData.Set("to", req.Recipient)
	formData.Set("message", req.Body)

	// Buat HTTP request dengan context untuk mendukung timeout dan cancellation
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
		}, fmt.Errorf("gagal membuat request zenziva: %w", err)
	}

	// Set header Content-Type untuk form data
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Kirim request ke Zenziva API
	resp, err := a.httpClient.Do(httpReq)
	if err != nil {
		return domain.SendResult{
			Status:      "failed",
			ErrorDetail: fmt.Sprintf("gagal mengirim request: %v", err),
		}, fmt.Errorf("gagal mengirim request zenziva: %w", err)
	}
	defer resp.Body.Close()

	// Baca response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return domain.SendResult{
			Status:      "failed",
			ErrorDetail: fmt.Sprintf("gagal membaca response: %v", err),
		}, fmt.Errorf("gagal membaca response zenziva: %w", err)
	}

	// Tangani status code non-200
	if resp.StatusCode != http.StatusOK {
		detail := fmt.Sprintf("status HTTP %d: %s", resp.StatusCode, string(body))
		return domain.SendResult{
			Status:      "failed",
			ErrorDetail: detail,
		}, fmt.Errorf("zenziva API error: %s", detail)
	}

	// Parse response JSON dari Zenziva
	var zenzivaResp zenzivaResponse
	if err := json.Unmarshal(body, &zenzivaResp); err != nil {
		detail := fmt.Sprintf("gagal parse response JSON: %v", err)
		return domain.SendResult{
			Status:      "failed",
			ErrorDetail: detail,
		}, fmt.Errorf("gagal parse response zenziva: %w", err)
	}

	// Evaluasi status dari response Zenziva (status 1 = sukses, 0 = gagal)
	if zenzivaResp.Status != 1 {
		return domain.SendResult{
			MessageID:   zenzivaResp.MessageID,
			Status:      "failed",
			ErrorDetail: zenzivaResp.Text,
		}, nil
	}

	return domain.SendResult{
		MessageID: zenzivaResp.MessageID,
		Status:    "sent",
	}, nil
}
