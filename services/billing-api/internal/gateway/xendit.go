package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

// XenditAdapter mengimplementasikan PaymentGatewayAdapter untuk Xendit.
type XenditAdapter struct {
	apiKey     string
	httpClient *http.Client
	baseURL    string
}

// NewXenditAdapter membuat instance baru XenditAdapter.
func NewXenditAdapter(apiKey string) *XenditAdapter {
	return &XenditAdapter{
		apiKey:     apiKey,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		baseURL:    "https://api.xendit.co",
	}
}

// Tipe permintaan/respons untuk Xendit Invoice API v2.
type xenditInvoiceRequest struct {
	ExternalID      string         `json:"external_id"`
	Amount          int64          `json:"amount"`
	Description     string         `json:"description"`
	Customer        xenditCustomer `json:"customer"`
	InvoiceDuration int            `json:"invoice_duration"`
	PaymentMethods  []string       `json:"payment_methods,omitempty"`
}

type xenditCustomer struct {
	GivenNames string `json:"given_names"`
	Email      string `json:"email,omitempty"`
}

type xenditInvoiceResponse struct {
	ID         string `json:"id"`
	InvoiceURL string `json:"invoice_url"`
	ExpiryDate string `json:"expiry_date"`
}

// xenditMethodMap memetakan metode internal ke format payment_methods Xendit.
var xenditMethodMap = map[string]string{
	"va_bca": "BCA", "va_bni": "BNI", "va_bri": "BRI",
	"va_mandiri": "MANDIRI", "va_permata": "PERMATA", "qris": "QRIS",
	"ewallet_ovo": "OVO", "ewallet_gopay": "GOPAY",
	"ewallet_dana": "DANA", "ewallet_shopeepay": "SHOPEEPAY",
	"credit_card": "CREDIT_CARD",
}

// CreatePaymentLink membuat link pembayaran via Xendit Invoice API v2 (POST /v2/invoices).
func (a *XenditAdapter) CreatePaymentLink(ctx context.Context, req CreateLinkRequest) (*domain.PaymentLinkResponse, error) {
	durationSec := int(req.ExpiryDuration.Seconds())
	if durationSec <= 0 {
		durationSec = 7 * 24 * 3600 // bawaan 7 hari
	}

	// Konversi metode pembayaran ke format Xendit
	var methods []string
	for _, m := range req.EnabledMethods {
		if xm, ok := xenditMethodMap[m]; ok {
			methods = append(methods, xm)
		}
	}

	body := xenditInvoiceRequest{
		ExternalID:      req.ExternalID,
		Amount:          req.Amount,
		Description:     req.Description,
		Customer:        xenditCustomer{GivenNames: req.CustomerName, Email: req.CustomerEmail},
		InvoiceDuration: durationSec,
		PaymentMethods:  methods,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("gagal marshal request xendit: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, a.baseURL+"/v2/invoices", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("gagal membuat request xendit: %w", err)
	}
	a.setAuthHeaders(httpReq)

	resp, err := a.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrGatewayUnavailable, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("gagal membaca response xendit: %w", err)
	}

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, domain.ErrGatewayInvalidAPIKey
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("%w: status=%d body=%s", domain.ErrGatewayUnavailable, resp.StatusCode, string(respBody))
	}

	var invoiceResp xenditInvoiceResponse
	if err := json.Unmarshal(respBody, &invoiceResp); err != nil {
		return nil, fmt.Errorf("gagal parse response xendit: %w", err)
	}

	expiresAt, _ := time.Parse(time.RFC3339, invoiceResp.ExpiryDate)
	return &domain.PaymentLinkResponse{
		ExternalID: invoiceResp.ID,
		PaymentURL: invoiceResp.InvoiceURL,
		ExpiresAt:  expiresAt,
	}, nil
}

// ExpirePaymentLink meng-expire link pembayaran di Xendit (POST /invoices/{id}/expire!).
func (a *XenditAdapter) ExpirePaymentLink(ctx context.Context, externalID string) error {
	url := fmt.Sprintf("%s/invoices/%s/expire!", a.baseURL, externalID)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return fmt.Errorf("gagal membuat request expire xendit: %w", err)
	}
	a.setAuthHeaders(httpReq)

	resp, err := a.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("%w: %v", domain.ErrGatewayUnavailable, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil // invoice sudah expired atau tidak ditemukan
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("%w: expire status=%d body=%s", domain.ErrGatewayUnavailable, resp.StatusCode, string(body))
	}
	return nil
}

// TestConnection menguji koneksi ke Xendit via GET /balance dengan timeout 10 detik.
func (a *XenditAdapter) TestConnection(ctx context.Context) (*domain.GatewayTestResult, error) {
	testCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	start := time.Now()
	httpReq, err := http.NewRequestWithContext(testCtx, http.MethodGet, a.baseURL+"/balance", nil)
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
			ErrorCode: "invalid_api_key", ErrorMessage: "API key tidak valid", LatencyMs: latency,
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

// setAuthHeaders menambahkan header autentikasi Xendit (Basic Auth).
func (a *XenditAdapter) setAuthHeaders(req *http.Request) {
	req.SetBasicAuth(a.apiKey, "")
	req.Header.Set("Content-Type", "application/json")
}
