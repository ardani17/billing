// report_handler_test.go berisi integration tests untuk report endpoints.
// Test: HTTP status codes, response shape, filter validation, graceful degradation.
package handler

import (
	"context"
	"encoding/json"
	"io"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

// --- Mock ReportUsecase untuk report handler tests ---

// mockReportUsecase mengimplementasikan domain.ReportUsecase untuk testing.
type mockReportUsecase struct {
	revenueReport      *domain.RevenueReport
	agingReport        *domain.AgingReport
	paymentReport      *domain.PaymentReport
	voucherReport      *domain.VoucherRevenueReport
	profitLossReport   *domain.ProfitLossReport
	revenueByArea      *domain.RevenueByAreaReport
	customerGrowth     *domain.CustomerGrowthReport
	customerDist       *domain.CustomerDistributionReport
	churnAnalysis      *domain.ChurnAnalysisReport
	uptimeReport       *domain.UptimeReport
	trafficReport      *domain.TrafficReport
	signalReport       *domain.SignalQualityReport
	capacityReport     *domain.CapacityReport
	activityReport     *domain.ActivityReport
	notificationReport *domain.NotificationReport
	syncReport         *domain.SyncReport
	dashboardData      *domain.DashboardData
	exportJobID        string
	exportJob          *domain.ReportJob
	err                error
}

func (m *mockReportUsecase) GetRevenueReport(_ context.Context, _ string, _ domain.ReportFilter) (*domain.RevenueReport, error) {
	return m.revenueReport, m.err
}
func (m *mockReportUsecase) GetAgingReport(_ context.Context, _ string, _ time.Time, _, _ string) (*domain.AgingReport, error) {
	return m.agingReport, m.err
}
func (m *mockReportUsecase) GetPaymentReport(_ context.Context, _ string, _ domain.ReportFilter) (*domain.PaymentReport, error) {
	return m.paymentReport, m.err
}
func (m *mockReportUsecase) GetVoucherRevenueReport(_ context.Context, _ string, _ domain.ReportFilter) (*domain.VoucherRevenueReport, error) {
	return m.voucherReport, m.err
}
func (m *mockReportUsecase) GetProfitLossReport(_ context.Context, _ string, _ domain.ReportFilter) (*domain.ProfitLossReport, error) {
	return m.profitLossReport, m.err
}
func (m *mockReportUsecase) GetRevenueByAreaReport(_ context.Context, _ string, _ domain.ReportFilter) (*domain.RevenueByAreaReport, error) {
	return m.revenueByArea, m.err
}
func (m *mockReportUsecase) GetCustomerGrowthReport(_ context.Context, _ string, _ domain.ReportFilter) (*domain.CustomerGrowthReport, error) {
	return m.customerGrowth, m.err
}
func (m *mockReportUsecase) GetCustomerDistributionReport(_ context.Context, _ string, _ time.Time) (*domain.CustomerDistributionReport, error) {
	return m.customerDist, m.err
}
func (m *mockReportUsecase) GetChurnAnalysisReport(_ context.Context, _ string, _ domain.ReportFilter) (*domain.ChurnAnalysisReport, error) {
	return m.churnAnalysis, m.err
}
func (m *mockReportUsecase) GetUptimeReport(_ context.Context, _ string, _ domain.ReportFilter) (*domain.UptimeReport, error) {
	return m.uptimeReport, m.err
}
func (m *mockReportUsecase) GetTrafficReport(_ context.Context, _ string, _ domain.ReportFilter) (*domain.TrafficReport, error) {
	return m.trafficReport, m.err
}
func (m *mockReportUsecase) GetSignalQualityReport(_ context.Context, _ string, _ domain.ReportFilter) (*domain.SignalQualityReport, error) {
	return m.signalReport, m.err
}
func (m *mockReportUsecase) GetCapacityReport(_ context.Context, _ string) (*domain.CapacityReport, error) {
	return m.capacityReport, m.err
}
func (m *mockReportUsecase) GetActivityReport(_ context.Context, _ string, _ domain.ReportFilter) (*domain.ActivityReport, error) {
	return m.activityReport, m.err
}
func (m *mockReportUsecase) GetNotificationReport(_ context.Context, _ string, _ domain.ReportFilter) (*domain.NotificationReport, error) {
	return m.notificationReport, m.err
}
func (m *mockReportUsecase) GetSyncReport(_ context.Context, _ string, _ domain.ReportFilter) (*domain.SyncReport, error) {
	return m.syncReport, m.err
}
func (m *mockReportUsecase) GetDashboardData(_ context.Context, _ string) (*domain.DashboardData, error) {
	return m.dashboardData, m.err
}
func (m *mockReportUsecase) RequestExport(_ context.Context, _, _, _, _ string, _ domain.ReportFilter) (string, error) {
	return m.exportJobID, m.err
}
func (m *mockReportUsecase) GetExportStatus(_ context.Context, _ string) (*domain.ReportJob, error) {
	return m.exportJob, m.err
}

// --- Setup helper ---

// setupReportTestApp membuat Fiber app dengan mock ReportUsecase untuk testing.
func setupReportTestApp(mock *mockReportUsecase) *fiber.App {
	logger := zerolog.New(io.Discard)
	handler := NewReportHandler(mock, logger)

	app := fiber.New()

	// Middleware untuk set tenant_id
	setLocals := func(c *fiber.Ctx) error {
		c.Locals("tenant_id", "test-tenant-id")
		c.Locals("user_id", "test-user-id")
		c.Locals("user_name", "Test User")
		return c.Next()
	}

	// Daftarkan route laporan keuangan
	financial := app.Group("/api/v1/reports/financial", setLocals)
	financial.Get("/revenue", handler.Revenue)
	financial.Get("/aging", handler.Aging)
	financial.Get("/payments", handler.Payments)
	financial.Get("/vouchers", handler.Vouchers)
	financial.Get("/profit-loss", handler.ProfitLoss)
	financial.Get("/revenue-by-area", handler.RevenueByArea)

	// Daftarkan route laporan pelanggan
	customers := app.Group("/api/v1/reports/customers", setLocals)
	customers.Get("/growth", handler.CustomerGrowth)
	customers.Get("/distribution", handler.CustomerDistribution)
	customers.Get("/churn", handler.ChurnAnalysis)

	return app
}

// --- Test: Filter validation ---

func TestReportHandler_Revenue_MissingPeriod(t *testing.T) {
	mock := &mockReportUsecase{}
	app := setupReportTestApp(mock)

	// Tanpa period_start dan period_end → 400
	req := httptest.NewRequest("GET", "/api/v1/reports/financial/revenue", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusBadRequest {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 400, got %d: %s", resp.StatusCode, string(body))
	}

	var apiResp domain.APIResponse
	json.NewDecoder(resp.Body).Decode(&apiResp)
	if apiResp.Error == nil || apiResp.Error.Code != "BAD_REQUEST" {
		t.Fatalf("expected BAD_REQUEST, got %v", apiResp.Error)
	}
}

func TestReportHandler_Revenue_PeriodStartAfterEnd(t *testing.T) {
	mock := &mockReportUsecase{}
	app := setupReportTestApp(mock)

	// period_start > period_end → 400
	req := httptest.NewRequest("GET",
		"/api/v1/reports/financial/revenue?period_start=2024-12-31&period_end=2024-01-01", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusBadRequest {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 400, got %d: %s", resp.StatusCode, string(body))
	}

	var apiResp domain.APIResponse
	json.NewDecoder(resp.Body).Decode(&apiResp)
	if apiResp.Error == nil || apiResp.Error.Code != "BAD_REQUEST" {
		t.Fatalf("expected BAD_REQUEST, got %v", apiResp.Error)
	}
}

func TestReportHandler_Revenue_InvalidDateFormat(t *testing.T) {
	mock := &mockReportUsecase{}
	app := setupReportTestApp(mock)

	// Format tanggal tidak valid → 400
	req := httptest.NewRequest("GET",
		"/api/v1/reports/financial/revenue?period_start=invalid&period_end=2024-01-31", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

// --- Test: Successful responses ---

func TestReportHandler_Revenue_Success(t *testing.T) {
	mock := &mockReportUsecase{
		revenueReport: &domain.RevenueReport{
			Current: domain.RevenueSource{
				MonthlySubscription: 10000000,
				VoucherSales:        5000000,
				Total:               15000000,
			},
			Trend: []domain.MonthlyRevenueTrend{},
		},
	}
	app := setupReportTestApp(mock)

	req := httptest.NewRequest("GET",
		"/api/v1/reports/financial/revenue?period_start=2024-01-01&period_end=2024-01-31", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(body))
	}

	var apiResp domain.APIResponse
	json.NewDecoder(resp.Body).Decode(&apiResp)
	if !apiResp.Success {
		t.Fatal("expected success=true")
	}
}

func TestReportHandler_Aging_Success(t *testing.T) {
	mock := &mockReportUsecase{
		agingReport: &domain.AgingReport{
			Buckets:          []domain.AgingBucket{},
			TotalOutstanding: 5000000,
			CollectionRate:   85.5,
		},
	}
	app := setupReportTestApp(mock)

	req := httptest.NewRequest("GET",
		"/api/v1/reports/financial/aging?period_end=2024-01-31", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(body))
	}
}

func TestReportHandler_Aging_MissingPeriodEnd(t *testing.T) {
	mock := &mockReportUsecase{}
	app := setupReportTestApp(mock)

	req := httptest.NewRequest("GET", "/api/v1/reports/financial/aging", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestReportHandler_Payments_Success(t *testing.T) {
	mock := &mockReportUsecase{
		paymentReport: &domain.PaymentReport{
			Methods:       []domain.PaymentMethodBreakdown{},
			DailyPayments: []domain.DailyPayment{},
		},
	}
	app := setupReportTestApp(mock)

	req := httptest.NewRequest("GET",
		"/api/v1/reports/financial/payments?period_start=2024-01-01&period_end=2024-01-31", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestReportHandler_ProfitLoss_Success(t *testing.T) {
	mock := &mockReportUsecase{
		profitLossReport: &domain.ProfitLossReport{
			TotalRevenue:  20000000,
			TotalExpenses: 8000000,
			NetProfit:     12000000,
			ProfitMargin:  60.0,
		},
	}
	app := setupReportTestApp(mock)

	req := httptest.NewRequest("GET",
		"/api/v1/reports/financial/profit-loss?period_start=2024-01-01&period_end=2024-01-31", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

// --- Test: Customer report endpoints ---

func TestReportHandler_CustomerGrowth_Success(t *testing.T) {
	mock := &mockReportUsecase{
		customerGrowth: &domain.CustomerGrowthReport{},
	}
	app := setupReportTestApp(mock)

	req := httptest.NewRequest("GET",
		"/api/v1/reports/customers/growth?period_start=2024-01-01&period_end=2024-01-31", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(body))
	}
}

func TestReportHandler_CustomerDistribution_Success(t *testing.T) {
	mock := &mockReportUsecase{
		customerDist: &domain.CustomerDistributionReport{},
	}
	app := setupReportTestApp(mock)

	req := httptest.NewRequest("GET",
		"/api/v1/reports/customers/distribution?period_end=2024-01-31", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestReportHandler_CustomerDistribution_MissingPeriodEnd(t *testing.T) {
	mock := &mockReportUsecase{}
	app := setupReportTestApp(mock)

	req := httptest.NewRequest("GET", "/api/v1/reports/customers/distribution", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestReportHandler_ChurnAnalysis_Success(t *testing.T) {
	mock := &mockReportUsecase{
		churnAnalysis: &domain.ChurnAnalysisReport{},
	}
	app := setupReportTestApp(mock)

	req := httptest.NewRequest("GET",
		"/api/v1/reports/customers/churn?period_start=2024-01-01&period_end=2024-01-31", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

// --- Test: Usecase error propagation ---

func TestReportHandler_Revenue_InternalError(t *testing.T) {
	mock := &mockReportUsecase{
		err: domain.ErrInsufficientData,
	}
	app := setupReportTestApp(mock)

	req := httptest.NewRequest("GET",
		"/api/v1/reports/financial/revenue?period_start=2024-01-01&period_end=2024-01-31", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusUnprocessableEntity {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 422, got %d: %s", resp.StatusCode, string(body))
	}

	var apiResp domain.APIResponse
	json.NewDecoder(resp.Body).Decode(&apiResp)
	if apiResp.Error == nil || apiResp.Error.Code != "INSUFFICIENT_DATA" {
		t.Fatalf("expected INSUFFICIENT_DATA, got %v", apiResp.Error)
	}
}

// --- Test: Response shape ---

func TestReportHandler_Revenue_ResponseShape(t *testing.T) {
	mock := &mockReportUsecase{
		revenueReport: &domain.RevenueReport{
			Current: domain.RevenueSource{
				MonthlySubscription: 10000000,
				VoucherSales:        5000000,
				InstallationFees:    1000000,
				LateFees:            500000,
				Other:               200000,
				Total:               16700000,
			},
			Trend: []domain.MonthlyRevenueTrend{
				{Month: "2024-01", TotalRevenue: 16700000},
			},
		},
	}
	app := setupReportTestApp(mock)

	req := httptest.NewRequest("GET",
		"/api/v1/reports/financial/revenue?period_start=2024-01-01&period_end=2024-01-31", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}

	body, _ := io.ReadAll(resp.Body)
	var raw map[string]interface{}
	if err := json.Unmarshal(body, &raw); err != nil {
		t.Fatalf("gagal parse response: %v", err)
	}

	// Verifikasi shape: success, data.current, data.trend
	if raw["success"] != true {
		t.Fatal("expected success=true")
	}
	data, ok := raw["data"].(map[string]interface{})
	if !ok {
		t.Fatal("expected data object in response")
	}
	if _, ok := data["current"]; !ok {
		t.Fatal("expected 'current' field in data")
	}
	if _, ok := data["trend"]; !ok {
		t.Fatal("expected 'trend' field in data")
	}
}

// --- Test: Unauthorized (tanpa tenant_id) ---

func TestReportHandler_Revenue_Unauthorized(t *testing.T) {
	mock := &mockReportUsecase{}
	logger := zerolog.New(io.Discard)
	handler := NewReportHandler(mock, logger)

	app := fiber.New()
	// Tanpa middleware set tenant_id
	app.Get("/api/v1/reports/financial/revenue", handler.Revenue)

	req := httptest.NewRequest("GET",
		"/api/v1/reports/financial/revenue?period_start=2024-01-01&period_end=2024-01-31", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}
