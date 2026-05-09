package gateway

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

// MidtransAdapter mengimplementasikan PaymentGatewayAdapter untuk Midtrans.
// Menggunakan Midtrans Snap API untuk membuat link pembayaran.
type MidtransAdapter struct {
	serverKey  string
	httpClient *http.Client
	baseURL    string
}

// NewMidtransAdapter membuat instance baru MidtransAdapter.
func NewMidtransAdapter(serverKey string) *MidtransAdapter {
	return &MidtransAdapter{
		serverKey: serverKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL: "https://app.midtrans.com",
	}
}

// Tipe permintaan/respons untuk Midtrans Snap API.
type midtransSnapRequest struct {
	TransactionDetails midtransTransactionDetails `json:"transaction_details"`
	CustomerDetails    midtransCustomerDetails    `json:"customer_details"`
	Expiry             midtransExpiry             `json:"expiry"`
}

type midtransTransactionDetails struct {
	OrderID     string `json:"order_id"`
	GrossAmount int64  `json:"gross_amount"`
}

type midtransCustomerDetails struct {
	FirstName string `json:"first_name"`
	Email     string `json:"email,omitempty"`
}

type midtransExpiry struct {
	StartTime string `json:"start_time"`
	Unit      string `json:"unit"`
	Duration  int    `json:"duration"`
}

type midtransSnapResponse struct {
	Token       string `json:"token"`
	RedirectURL string `json:"redirect_url"`
}

// CreatePaymentLink membuat link pembayaran via Midtrans Snap API (POST /snap/v1/transactions).
func (a *MidtransAdapter) CreatePaymentLink(ctx context.Context, req CreateLinkRequest) (*domain.PaymentLinkResponse, error) {
	durationMin := int(req.ExpiryDuration.Minutes())
	if durationMin <= 0 {
		durationMin = 7 * 24 * 60 // bawaan 7 hari dalam menit
	}

	body := midtransSnapRequest{
		TransactionDetails: midtransTransactionDetails{
			OrderID:     req.ExternalID,
			GrossAmount: req.Amount,
		},
		CustomerDetails: midtransCustomerDetails{
			FirstName: req.CustomerName,
			Email:     req.CustomerEmail,
		},
		Expiry: midtransExpiry{
			StartTime: time.Now().Format("2006-01-02 15:04:05 -0700"),
			Unit:      "minute",
			Duration:  durationMin,
		},
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("gagal marshal request midtrans: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, a.baseURL+"/snap/v1/transactions", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("gagal membuat request midtrans: %w", err)
	}
	a.setAuthHeaders(httpReq)

	resp, err := a.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrGatewayUnavailable, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("gagal membaca response midtrans: %w", err)
	}

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, domain.ErrGatewayInvalidAPIKey
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("%w: status=%d body=%s", domain.ErrGatewayUnavailable, resp.StatusCode, string(respBody))
	}

	var snapResp midtransSnapResponse
	if err := json.Unmarshal(respBody, &snapResp); err != nil {
		return nil, fmt.Errorf("gagal parse response midtrans: %w", err)
	}

	expiresAt := time.Now().Add(req.ExpiryDuration)
	if req.ExpiryDuration <= 0 {
		expiresAt = time.Now().Add(7 * 24 * time.Hour)
	}

	return &domain.PaymentLinkResponse{
		ExternalID: req.ExternalID,
		PaymentURL: snapResp.RedirectURL,
		ExpiresAt:  expiresAt,
	}, nil
}

// ExpirePaymentLink meng-cancel link pembayaran di Midtrans (POST /v2/{order_id}/cancel).
func (a *MidtransAdapter) ExpirePaymentLink(ctx context.Context, externalID string) error {
	url := fmt.Sprintf("%s/v2/%s/cancel", a.baseURL, externalID)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return fmt.Errorf("gagal membuat request cancel midtrans: %w", err)
	}
	a.setAuthHeaders(httpReq)

	resp, err := a.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("%w: %v", domain.ErrGatewayUnavailable, err)
	}
	defer resp.Body.Close()

	// 404 berarti transaksi sudah expired atau tidak ditemukan
	if resp.StatusCode == http.StatusNotFound {
		return nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("%w: cancel status=%d body=%s", domain.ErrGatewayUnavailable, resp.StatusCode, string(body))
	}
	return nil
}

// TestConnection menguji koneksi ke Midtrans via GET /v2/point_of_sales dengan timeout 10 detik.
func (a *MidtransAdapter) TestConnection(ctx context.Context) (*domain.GatewayTestResult, error) {
	testCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	start := time.Now()
	// Gunakan endpoint status untuk verifikasi kredensial
	httpReq, err := http.NewRequestWithContext(testCtx, http.MethodGet, a.baseURL+"/v2/point_of_sales", nil)
	if err != nil {
		return &domain.GatewayTestResult{
			ErrorCode: "request_error", ErrorMessage: err.Error(), LatencyMs: time.Since(start).Milliseconds(),
		}, nil
	}
	a.setAuthHeaders(httpReq)

	resp, err := a.httpClient.Do(httpReq)
	latency := time.Since(start).Milliseconds()
	if err != nil {
		return &domain.GatewayTestResult{
			ErrorCode: "gateway_unavailable", ErrorMessage: err.Error(), LatencyMs: latency,
		}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return &domain.GatewayTestResult{
			ErrorCode: "invalid_api_key", ErrorMessage: "Server key tidak valid", LatencyMs: latency,
		}, nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return &domain.GatewayTestResult{
			ErrorCode:    "gateway_error",
			ErrorMessage: fmt.Sprintf("status=%d body=%s", resp.StatusCode, string(body)),
			LatencyMs:    latency,
		}, nil
	}
	return &domain.GatewayTestResult{Success: true, LatencyMs: latency}, nil
}

// setAuthHeaders menambahkan header autentikasi Midtrans (Basic Auth dengan server key).
func (a *MidtransAdapter) setAuthHeaders(req *http.Request) {
	// Midtrans menggunakan Basic Auth: base64(serverKey + ":")
	auth := base64.StdEncoding.EncodeToString([]byte(a.serverKey + ":"))
	req.Header.Set("Authorization", "Basic "+auth)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
}
