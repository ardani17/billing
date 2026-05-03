// network_client_test.go berisi integration tests untuk NetworkServiceClient.
// Test: successful HTTP call, timeout handling, fallback to cache, module_inactive response.
package usecase

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/rs/zerolog"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

// --- Mock Redis client untuk network client tests ---

// mockRedisClient mengimplementasikan operasi Redis minimal untuk testing.
// Menyimpan data di memory map sebagai pengganti Redis.
type mockRedisClient struct {
	data map[string][]byte
}

func newMockRedisClient() *mockRedisClient {
	return &mockRedisClient{data: make(map[string][]byte)}
}

// --- Test: Successful HTTP call ---

func TestNetworkClient_GetUptimeReport_Success(t *testing.T) {
	// Buat mock HTTP server yang mengembalikan uptime report
	expectedReport := domain.UptimeReport{
		Routers: []domain.RouterUptimeItem{
			{
				RouterID:         "router-1",
				RouterName:       "MikroTik-01",
				UptimePercentage: 99.5,
				TotalDowntimeMin: 30,
				RebootCount:      1,
				StatusLabel:      "Excellent",
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verifikasi path dan query params
		if r.URL.Path != "/internal/v1/reports/uptime" {
			t.Errorf("expected path /internal/v1/reports/uptime, got %s", r.URL.Path)
		}
		if r.URL.Query().Get("tenant_id") != "test-tenant" {
			t.Errorf("expected tenant_id=test-tenant, got %s", r.URL.Query().Get("tenant_id"))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(expectedReport)
	}))
	defer server.Close()

	logger := zerolog.New(io.Discard)
	// Buat NetworkClient tanpa Redis (nil) — hanya test HTTP call
	client := &NetworkClient{
		baseURL:    server.URL,
		httpClient: &http.Client{Timeout: 10 * time.Second},
		redis:      nil,
		logger:     logger,
	}

	ctx := context.Background()
	report, err := client.GetUptimeReport(ctx, "test-tenant",
		time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC),
		"")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if report == nil {
		t.Fatal("expected report, got nil")
	}
	if len(report.Routers) != 1 {
		t.Fatalf("expected 1 router, got %d", len(report.Routers))
	}
	if report.Routers[0].RouterName != "MikroTik-01" {
		t.Fatalf("expected router name MikroTik-01, got %s", report.Routers[0].RouterName)
	}
	if report.Routers[0].UptimePercentage != 99.5 {
		t.Fatalf("expected uptime 99.5, got %f", report.Routers[0].UptimePercentage)
	}
	if report.StaleData {
		t.Fatal("expected stale_data=false for fresh data")
	}
}

func TestNetworkClient_GetTrafficReport_Success(t *testing.T) {
	expectedReport := domain.TrafficReport{
		TotalDownloadBytes: 1073741824, // 1 GB
		TotalUploadBytes:   536870912,  // 512 MB
		TotalTrafficBytes:  1610612736,
		ByRouter: []domain.RouterTraffic{
			{RouterID: "router-1", RouterName: "MikroTik-01", DownloadBytes: 1073741824},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(expectedReport)
	}))
	defer server.Close()

	logger := zerolog.New(io.Discard)
	client := &NetworkClient{
		baseURL:    server.URL,
		httpClient: &http.Client{Timeout: 10 * time.Second},
		redis:      nil,
		logger:     logger,
	}

	ctx := context.Background()
	report, err := client.GetTrafficReport(ctx, "test-tenant",
		time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC),
		"")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if report == nil {
		t.Fatal("expected report, got nil")
	}
	if report.TotalTrafficBytes != 1610612736 {
		t.Fatalf("expected total traffic 1610612736, got %d", report.TotalTrafficBytes)
	}
}

// --- Test: Module inactive response (server down, no cache) ---

func TestNetworkClient_GetUptimeReport_ModuleInactive(t *testing.T) {
	// Server yang mengembalikan error 500
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	logger := zerolog.New(io.Discard)
	client := &NetworkClient{
		baseURL:    server.URL,
		httpClient: &http.Client{Timeout: 10 * time.Second},
		redis:      nil, // Tidak ada Redis → tidak ada cache fallback
		logger:     logger,
	}

	ctx := context.Background()
	report, err := client.GetUptimeReport(ctx, "test-tenant",
		time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC),
		"")
	if err != nil {
		t.Fatalf("expected no error (graceful degradation), got: %v", err)
	}
	if report == nil {
		t.Fatal("expected report with module_inactive, got nil")
	}
	if !report.ModuleInactive {
		t.Fatal("expected module_inactive=true when server down and no cache")
	}
}

func TestNetworkClient_GetTrafficReport_ModuleInactive(t *testing.T) {
	// Server yang mengembalikan error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	logger := zerolog.New(io.Discard)
	client := &NetworkClient{
		baseURL:    server.URL,
		httpClient: &http.Client{Timeout: 10 * time.Second},
		redis:      nil,
		logger:     logger,
	}

	ctx := context.Background()
	report, err := client.GetTrafficReport(ctx, "test-tenant",
		time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC),
		"")
	if err != nil {
		t.Fatalf("expected no error (graceful degradation), got: %v", err)
	}
	if report == nil {
		t.Fatal("expected report with module_inactive, got nil")
	}
	if !report.ModuleInactive {
		t.Fatal("expected module_inactive=true")
	}
}

// --- Test: Timeout handling ---

func TestNetworkClient_GetUptimeReport_Timeout(t *testing.T) {
	// Server yang delay lebih lama dari timeout
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	logger := zerolog.New(io.Discard)
	client := &NetworkClient{
		baseURL:    server.URL,
		httpClient: &http.Client{Timeout: 50 * time.Millisecond}, // Timeout sangat pendek
		redis:      nil,
		logger:     logger,
	}

	ctx := context.Background()
	report, err := client.GetUptimeReport(ctx, "test-tenant",
		time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC),
		"")
	if err != nil {
		t.Fatalf("expected graceful degradation, got error: %v", err)
	}
	// Tanpa cache, harus return module_inactive
	if report == nil {
		t.Fatal("expected report with module_inactive, got nil")
	}
	if !report.ModuleInactive {
		t.Fatal("expected module_inactive=true on timeout")
	}
}

// --- Test: Invalid JSON response ---

func TestNetworkClient_GetUptimeReport_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("invalid json"))
	}))
	defer server.Close()

	logger := zerolog.New(io.Discard)
	client := &NetworkClient{
		baseURL:    server.URL,
		httpClient: &http.Client{Timeout: 10 * time.Second},
		redis:      nil,
		logger:     logger,
	}

	ctx := context.Background()
	report, err := client.GetUptimeReport(ctx, "test-tenant",
		time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC),
		"")
	if err != nil {
		t.Fatalf("expected graceful degradation, got error: %v", err)
	}
	// Invalid JSON → fallback ke cache → tidak ada cache → module_inactive
	if report == nil {
		t.Fatal("expected report with module_inactive, got nil")
	}
	if !report.ModuleInactive {
		t.Fatal("expected module_inactive=true on invalid JSON")
	}
}

// --- Test: URL building ---

func TestNetworkClient_BuildURL(t *testing.T) {
	logger := zerolog.New(io.Discard)
	client := &NetworkClient{
		baseURL: "http://localhost:8081",
		logger:  logger,
	}

	url, params := client.buildURL("uptime", "tenant-123",
		time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC),
		map[string]string{"router_id": "router-1"})

	if params.Get("tenant_id") != "tenant-123" {
		t.Fatalf("expected tenant_id=tenant-123, got %s", params.Get("tenant_id"))
	}
	if params.Get("router_id") != "router-1" {
		t.Fatalf("expected router_id=router-1, got %s", params.Get("router_id"))
	}
	if url == "" {
		t.Fatal("expected non-empty URL")
	}
}

// --- Test: Server unreachable ---

func TestNetworkClient_GetUptimeReport_ServerUnreachable(t *testing.T) {
	logger := zerolog.New(io.Discard)
	client := &NetworkClient{
		baseURL:    "http://localhost:99999", // Port tidak valid
		httpClient: &http.Client{Timeout: 1 * time.Second},
		redis:      nil,
		logger:     logger,
	}

	ctx := context.Background()
	report, err := client.GetUptimeReport(ctx, "test-tenant",
		time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC),
		"")
	if err != nil {
		t.Fatalf("expected graceful degradation, got error: %v", err)
	}
	if report == nil {
		t.Fatal("expected report with module_inactive, got nil")
	}
	if !report.ModuleInactive {
		t.Fatal("expected module_inactive=true when server unreachable")
	}
}

// --- Test: Signal Quality Report ---

func TestNetworkClient_GetSignalQualityReport_Success(t *testing.T) {
	expectedReport := domain.SignalQualityReport{
		NormalCount:      80,
		WarningCount:     15,
		WeakCount:        4,
		CriticalCount:    1,
		TotalONTCount:    100,
		AverageSignalDBm: -22.5,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(expectedReport)
	}))
	defer server.Close()

	logger := zerolog.New(io.Discard)
	client := &NetworkClient{
		baseURL:    server.URL,
		httpClient: &http.Client{Timeout: 10 * time.Second},
		redis:      nil,
		logger:     logger,
	}

	ctx := context.Background()
	report, err := client.GetSignalQualityReport(ctx, "test-tenant",
		time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC),
		"")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if report == nil {
		t.Fatal("expected report, got nil")
	}
	if report.TotalONTCount != 100 {
		t.Fatalf("expected 100 ONTs, got %d", report.TotalONTCount)
	}
}

// --- Test: Capacity Report ---

func TestNetworkClient_GetCapacityReport_Success(t *testing.T) {
	expectedReport := domain.CapacityReport{
		RouterCapacity: []domain.RouterCapacity{
			{RouterID: "r-1", RouterName: "MikroTik-01", CurrentCustomers: 50, MaxCapacity: 100, UsagePercentage: 50.0},
		},
		ODPCapacity: []domain.ODPCapacity{
			{ODPID: "odp-1", ODPName: "ODP-01", UsedPorts: 6, TotalPorts: 8, UsagePercentage: 75.0},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(expectedReport)
	}))
	defer server.Close()

	logger := zerolog.New(io.Discard)
	client := &NetworkClient{
		baseURL:    server.URL,
		httpClient: &http.Client{Timeout: 10 * time.Second},
		redis:      nil,
		logger:     logger,
	}

	ctx := context.Background()
	report, err := client.GetCapacityReport(ctx, "test-tenant")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if report == nil {
		t.Fatal("expected report, got nil")
	}
	if len(report.RouterCapacity) != 1 {
		t.Fatalf("expected 1 router capacity, got %d", len(report.RouterCapacity))
	}
}
