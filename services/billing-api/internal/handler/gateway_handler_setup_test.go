// untuk test GatewayHandler.
package handler

import (
	"context"
	"io"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
	"github.com/ispboss/ispboss/services/billing-api/internal/usecase"
)

type mockGatewaySettingsRepo struct{}

func (m *mockGatewaySettingsRepo) GetByTenantID(_ context.Context, _ string) (*domain.BillingSettings, error) {
	return &domain.BillingSettings{}, nil
}
func (m *mockGatewaySettingsRepo) Upsert(_ context.Context, s *domain.BillingSettings) (*domain.BillingSettings, error) {
	return s, nil
}
func (m *mockGatewaySettingsRepo) ListAll(_ context.Context) ([]*domain.BillingSettings, error) {
	return nil, nil
}

// gatewayTestSetup berisi semua komponen yang dibutuhkan untuk test.
type gatewayTestSetup struct {
	app          *fiber.App
	configRepo   *mockGatewayConfigRepo
	linkRepo     *mockPaymentLinkRepo
	webhookRepo  *mockWebhookLogRepo
	invoiceRepo  *mockGatewayInvoiceRepo
	customerRepo *mockGatewayCustomerRepo
}

// testMasterKey adalah 32-byte key untuk test enkripsi.
var testMasterKey = []byte("01234567890123456789012345678901")

func setupGatewayTestApp() *gatewayTestSetup {
	configRepo := newMockGatewayConfigRepo()
	linkRepo := newMockPaymentLinkRepo()
	webhookRepo := newMockWebhookLogRepo()
	invoiceRepo := newMockGatewayInvoiceRepo()
	customerRepo := newMockGatewayCustomerRepo()
	settingsRepo := &mockGatewaySettingsRepo{}
	logger := zerolog.New(io.Discard)

	uc := usecase.NewGatewayUsecase(
		configRepo, linkRepo, invoiceRepo, customerRepo,
		settingsRepo, nil, nil, testMasterKey, logger,
	)
	handler := NewGatewayHandler(uc, webhookRepo, linkRepo, logger)

	app := fiber.New()

	// Middleware untuk atur tenant_id dan user_id di Locals
	setLocals := func(c *fiber.Ctx) error {
		c.Locals("tenant_id", "test-tenant-id")
		c.Locals("user_id", "test-user-id")
		return c.Next()
	}

	// Route konfigurasi gateway
	gw := app.Group("/api/v1/settings/payment-gateways", setLocals)
	gw.Post("/", handler.CreateConfig)
	gw.Get("/", handler.ListConfigs)
	gw.Put("/:id", handler.UpdateConfig)
	gw.Delete("/:id", handler.DeactivateConfig)
	gw.Post("/:id/test", handler.TestConfig)

	// Route tautan pembayaran
	cust := app.Group("/api/v1/customers", setLocals)
	cust.Get("/:id/payment-link", handler.GetCustomerPaymentLink)
	cust.Post("/:id/payment-link/regenerate", handler.RegeneratePaymentLink)

	// Route invoice tautan pembayaran dan tautan pembayaran webhooks
	inv := app.Group("/api/v1/invoices", setLocals)
	inv.Get("/:id/payment-links", handler.GetInvoicePaymentLinks)
	pl := app.Group("/api/v1/payment-links", setLocals)
	pl.Get("/:id/webhooks", handler.GetPaymentLinkWebhooks)

	// Route walled garden (publik, tanpa auth)
	app.Get("/api/v1/public/walled-garden/:customer_id/payment-info",
		handler.WalledGardenPaymentInfo)

	return &gatewayTestSetup{
		app: app, configRepo: configRepo, linkRepo: linkRepo,
		webhookRepo: webhookRepo, invoiceRepo: invoiceRepo, customerRepo: customerRepo,
	}
}
