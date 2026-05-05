package handler

import (
	"context"
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/swagger"
	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
	"github.com/ispboss/ispboss/services/billing-api/internal/middleware"
	"github.com/rs/zerolog"
)

// RouterConfig berisi dependensi yang dibutuhkan untuk registrasi route.
type RouterConfig struct {
	// App adalah instance Fiber application
	App *fiber.App

	// HealthHandler adalah handler untuk health check endpoint
	HealthHandler *HealthHandler

	// AuthHandler adalah handler untuk endpoint autentikasi
	AuthHandler *AuthHandler

	// UserHandler adalah handler untuk manajemen user
	UserHandler *UserHandler

	// SessionHandler adalah handler untuk manajemen session
	SessionHandler *SessionHandler

	// AdminHandler adalah handler untuk fitur super admin (impersonation)
	AdminHandler *AdminHandler

	// CustomerHandler adalah handler untuk manajemen pelanggan
	CustomerHandler *CustomerHandler

	// AreaHandler adalah handler untuk manajemen area
	AreaHandler *AreaHandler

	// PackageHandler adalah handler untuk manajemen paket
	PackageHandler *PackageHandler

	// ResellerHandler adalah handler untuk manajemen reseller (admin CRUD)
	ResellerHandler *ResellerHandler

	// ResellerActionHandler adalah handler untuk aksi reseller (admin: suspend, activate, dll)
	ResellerActionHandler *ResellerActionHandler

	// VoucherHandler adalah handler untuk manajemen voucher (admin)
	VoucherHandler *VoucherHandler

	// VoucherPrintHandler adalah handler untuk generate PDF voucher
	VoucherPrintHandler *VoucherPrintHandler

	// ResellerAuthHandler adalah handler untuk autentikasi reseller
	ResellerAuthHandler *ResellerAuthHandler

	// ResellerDashboardHandler adalah handler untuk dashboard reseller
	ResellerDashboardHandler *ResellerDashboardHandler

	// InvoiceHandler adalah handler untuk manajemen invoice (CRUD)
	InvoiceHandler *InvoiceHandler

	// InvoiceActionHandler adalah handler untuk aksi invoice (cancel, payment, bulk, export)
	InvoiceActionHandler *InvoiceActionHandler

	// RecurringItemHandler adalah handler untuk recurring items pelanggan
	RecurringItemHandler *RecurringItemHandler

	// CreditNoteHandler adalah handler untuk credit notes
	CreditNoteHandler *CreditNoteHandler

	// DebitNoteHandler adalah handler untuk debit notes
	DebitNoteHandler *DebitNoteHandler

	// PaymentHandler adalah handler untuk modul pembayaran manual
	PaymentHandler *PaymentHandler

	// GatewayHandler adalah handler untuk konfigurasi payment gateway dan payment link
	GatewayHandler *GatewayHandler

	// WebhookHandler adalah handler untuk endpoint webhook publik (Xendit, Midtrans)
	WebhookHandler *WebhookHandler

	// IsolirHandler adalah handler untuk modul isolir (sync, pending syncs, summary, waive penalty, reactivate)
	IsolirHandler *IsolirHandler

	// ReportHandler adalah handler untuk laporan (financial, customer, network, operational)
	ReportHandler *ReportHandler

	// ExpenseHandler adalah handler untuk manajemen pengeluaran
	ExpenseHandler *ExpenseHandler

	// ExportHandler adalah handler untuk export laporan (PDF, Excel, CSV)
	ExportHandler *ExportHandler

	// ScheduleHandler adalah handler untuk jadwal laporan otomatis
	ScheduleHandler *ScheduleHandler

	// KPIHandler adalah handler untuk target KPI
	KPIHandler *KPIHandler

	// ForecastHandler adalah handler untuk proyeksi bisnis
	ForecastHandler *ForecastHandler

	// ComparisonHandler adalah handler untuk perbandingan antar periode
	ComparisonHandler *ComparisonHandler

	// CustomReportHandler adalah handler untuk laporan custom
	CustomReportHandler *CustomReportHandler

	// DashboardHandler adalah handler untuk dashboard widget
	DashboardHandler *DashboardHandler

	// TenantModuleHandler adalah handler untuk entitlement modul tenant
	TenantModuleHandler *TenantModuleHandler

	// RateLimiter adalah rate limiter untuk login endpoint (admin)
	RateLimiter *middleware.LoginRateLimiter

	// ResellerRateLimiter adalah rate limiter untuk login endpoint reseller
	ResellerRateLimiter *middleware.LoginRateLimiter

	// JWTSecret adalah secret key untuk validasi JWT token
	JWTSecret string

	// Logger adalah instance zerolog untuk request logging
	Logger zerolog.Logger
}

// loginRateLimiterMiddleware membuat Fiber middleware wrapper untuk LoginRateLimiter.
// Middleware ini memeriksa rate limit berdasarkan email dari request body.
// Jika email terkunci, mengembalikan HTTP 429 dengan sisa waktu lock.
func loginRateLimiterMiddleware(rateLimiter *middleware.LoginRateLimiter) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Parse email dari request body tanpa mengkonsumsi body
		var body struct {
			Email string `json:"email"`
		}
		if err := c.BodyParser(&body); err != nil || body.Email == "" {
			// Jika tidak bisa parse email, lanjutkan ke handler (handler akan validasi)
			return c.Next()
		}

		// Cek rate limit
		allowed, remainingSec, err := rateLimiter.Check(context.Background(), body.Email)
		if err != nil {
			// Jika error, lanjutkan ke handler (jangan block user karena error internal)
			return c.Next()
		}

		if !allowed {
			return domain.ErrorResponse(c, fiber.StatusTooManyRequests, "ACCOUNT_LOCKED",
				fmt.Sprintf("akun terkunci sementara, coba lagi dalam %d detik", remainingSec))
		}

		return c.Next()
	}
}

// RegisterRoutes mendaftarkan semua route pada Fiber app.
// Health check dan Swagger bersifat publik (tanpa auth).
// Auth routes dibagi menjadi public (tanpa auth) dan protected (dengan auth).
// Settings dan admin routes dilindungi oleh auth + RBAC middleware.
func RegisterRoutes(cfg RouterConfig) {
	// Middleware logging untuk semua request
	cfg.App.Use(middleware.RequestLogger(cfg.Logger))

	// Route publik — health check (tanpa autentikasi)
	cfg.App.Get("/healthz", cfg.HealthHandler.Healthz)
	cfg.App.Get("/readyz", cfg.HealthHandler.Readyz)

	// Swagger UI — dokumentasi API otomatis
	cfg.App.Get("/swagger/*", swagger.HandlerDefault)

	// --- Public webhook routes (tanpa auth, keamanan via IP whitelist + signature) ---
	cfg.App.Post("/webhooks/xendit", cfg.WebhookHandler.HandleXendit)
	cfg.App.Post("/webhooks/midtrans", cfg.WebhookHandler.HandleMidtrans)

	// --- Public walled garden route (tanpa auth, untuk captive portal pelanggan isolir) ---
	cfg.App.Get("/api/v1/public/walled-garden/:customer_id/payment-info", cfg.GatewayHandler.WalledGardenPaymentInfo)

	// --- Public auth routes (tanpa auth middleware) ---
	authPublic := cfg.App.Group("/api/v1/auth")
	authPublic.Post("/register", cfg.AuthHandler.Register)
	authPublic.Post("/login", loginRateLimiterMiddleware(cfg.RateLimiter), cfg.AuthHandler.Login)
	authPublic.Post("/google", cfg.AuthHandler.LoginWithGoogle)
	authPublic.Post("/verify-email", cfg.AuthHandler.VerifyEmail)
	authPublic.Post("/resend-verification", cfg.AuthHandler.ResendVerification)
	authPublic.Post("/forgot-password", cfg.AuthHandler.ForgotPassword)
	authPublic.Post("/reset-password", cfg.AuthHandler.ResetPassword)
	authPublic.Post("/refresh", cfg.AuthHandler.RefreshToken)

	// --- Protected auth routes (auth middleware only, tanpa tenant/RBAC) ---
	authProtected := cfg.App.Group("/api/v1/auth")
	authProtected.Use(middleware.Auth(cfg.JWTSecret))
	authProtected.Get("/me", cfg.AuthHandler.GetMe)
	authProtected.Post("/logout", cfg.AuthHandler.Logout)
	authProtected.Get("/sessions", cfg.SessionHandler.List)
	authProtected.Delete("/sessions/:id", cfg.SessionHandler.Revoke)
	authProtected.Delete("/sessions", cfg.SessionHandler.RevokeOthers)

	// --- Reseller auth routes (publik, tanpa auth middleware) ---
	// Login dan refresh token reseller menggunakan phone+password, terpisah dari admin auth.
	resellerAuthPublic := cfg.App.Group("/api/v1/reseller/auth")
	resellerAuthPublic.Post("/login",
		middleware.ResellerLoginRateLimiterMiddleware(cfg.ResellerRateLimiter),
		cfg.ResellerAuthHandler.Login,
	)
	resellerAuthPublic.Post("/refresh", cfg.ResellerAuthHandler.Refresh)

	// --- Reseller auth protected routes (reseller JWT, tanpa tenant context) ---
	// Logout memerlukan token reseller yang valid.
	resellerAuthProtected := cfg.App.Group("/api/v1/reseller/auth")
	resellerAuthProtected.Use(middleware.ResellerAuth(cfg.JWTSecret))
	resellerAuthProtected.Post("/logout", cfg.ResellerAuthHandler.Logout)

	// --- Reseller dashboard routes (reseller JWT + tenant context) ---
	// Route ini didaftarkan eksplisit agar tidak bentrok dengan admin /api/v1/resellers.
	resellerAuth := middleware.ResellerAuth(cfg.JWTSecret)
	resellerTenant := middleware.TenantContext(cfg.JWTSecret)
	cfg.App.Get("/api/v1/reseller/dashboard", resellerAuth, resellerTenant, cfg.ResellerDashboardHandler.Summary)
	cfg.App.Post("/api/v1/reseller/vouchers/buy", resellerAuth, resellerTenant, cfg.ResellerDashboardHandler.Buy)
	cfg.App.Get("/api/v1/reseller/vouchers", resellerAuth, resellerTenant, cfg.ResellerDashboardHandler.MyVouchers)
	cfg.App.Post("/api/v1/reseller/vouchers/print", resellerAuth, resellerTenant, cfg.ResellerDashboardHandler.Print)
	cfg.App.Get("/api/v1/reseller/deposit", resellerAuth, resellerTenant, cfg.ResellerDashboardHandler.DepositHistory)
	cfg.App.Get("/api/v1/reseller/history", resellerAuth, resellerTenant, cfg.ResellerDashboardHandler.TransactionHistory)

	// --- Settings routes (auth + RBAC middleware) ---
	settings := cfg.App.Group("/api/v1/settings")
	settings.Use(middleware.Auth(cfg.JWTSecret))

	// Change password — semua authenticated user boleh akses
	settings.Post("/security/change-password", cfg.AuthHandler.ChangePassword)

	// User management — hanya tenant_admin (dan super_admin via bypass)
	users := settings.Group("/users")
	users.Use(middleware.RBAC(domain.RBACConfig{
		AllowedRoles: []domain.UserRole{domain.RoleTenantAdmin},
	}))
	users.Get("/", cfg.UserHandler.List)
	users.Post("/", cfg.UserHandler.Create)
	users.Get("/:id", cfg.UserHandler.Get)
	users.Put("/:id", cfg.UserHandler.Update)
	users.Delete("/:id", cfg.UserHandler.Delete)
	users.Post("/:id/deactivate", cfg.UserHandler.Deactivate)
	users.Post("/:id/activate", cfg.UserHandler.Activate)
	users.Post("/:id/reset-password", cfg.UserHandler.ResetPassword)

	// --- Payment gateway config routes (auth + RBAC, tenant_admin only) ---
	paymentGateways := settings.Group("/payment-gateways")
	paymentGateways.Use(middleware.RBAC(domain.RBACConfig{
		AllowedRoles: []domain.UserRole{domain.RoleTenantAdmin},
	}))
	paymentGateways.Post("/", cfg.GatewayHandler.CreateConfig)
	paymentGateways.Get("/", cfg.GatewayHandler.ListConfigs)
	paymentGateways.Put("/:id", cfg.GatewayHandler.UpdateConfig)
	paymentGateways.Delete("/:id", cfg.GatewayHandler.DeactivateConfig)
	paymentGateways.Post("/:id/test", cfg.GatewayHandler.TestConfig)

	// --- Admin routes (auth + RBAC middleware, super_admin only) ---
	admin := cfg.App.Group("/api/v1/admin")
	admin.Use(middleware.Auth(cfg.JWTSecret))
	admin.Use(middleware.RBAC(domain.RBACConfig{
		AllowedRoles: []domain.UserRole{domain.RoleSuperAdmin},
	}))
	admin.Post("/impersonate", cfg.AdminHandler.Start)
	admin.Post("/stop-impersonate", cfg.AdminHandler.Stop)
	admin.Get("/platform/overview", cfg.AdminHandler.PlatformOverview)
	admin.Get("/platform/tenants", cfg.AdminHandler.PlatformTenants)
	admin.Get("/platform/tenants/:id", cfg.AdminHandler.PlatformTenantDetail)
	admin.Get("/platform/subscriptions", cfg.AdminHandler.PlatformSubscriptions)
	admin.Get("/platform/support", cfg.AdminHandler.PlatformSupport)
	admin.Get("/platform/health", cfg.AdminHandler.PlatformHealth)
	admin.Get("/platform/audit", cfg.AdminHandler.PlatformAudit)

	// --- Protected business routes (auth + tenant middleware) ---
	// Grup route yang dilindungi oleh auth dan tenant middleware untuk endpoint bisnis
	api := cfg.App.Group("/api/v1")
	api.Use(middleware.Auth(cfg.JWTSecret))
	api.Use(middleware.TenantContext(cfg.JWTSecret))

	// --- Tenant module entitlement routes ---
	api.Get("/tenant/modules", cfg.TenantModuleHandler.Current)

	// --- Customer routes (auth + tenant + RBAC) ---
	customerHandler := cfg.CustomerHandler
	areaHandler := cfg.AreaHandler

	customers := api.Group("/customers")

	// Routes accessible by admin, operator, kasir(GET only)
	customersRead := customers.Group("")
	customersRead.Use(middleware.RBAC(domain.RBACConfig{
		AllowedRoles: []domain.UserRole{
			domain.RoleTenantAdmin, domain.RoleOperator, domain.RoleKasir,
		},
		MethodRestrictions: map[domain.UserRole][]string{
			domain.RoleKasir: {"GET"},
		},
	}))
	customersRead.Get("/", customerHandler.List)
	customersRead.Get("/stats", customerHandler.Stats)
	customersRead.Get("/:id", customerHandler.Get)

	// Payment link read — admin + kasir (GET only)
	customersRead.Get("/:id/payment-link", cfg.GatewayHandler.GetCustomerPaymentLink)

	// Routes accessible by admin, operator (write operations)
	customersWrite := customers.Group("")
	customersWrite.Use(middleware.RBAC(domain.RBACConfig{
		AllowedRoles: []domain.UserRole{
			domain.RoleTenantAdmin, domain.RoleOperator,
		},
	}))
	customersWrite.Post("/", customerHandler.Create)
	customersWrite.Put("/:id", customerHandler.Update)
	customersWrite.Delete("/:id", customerHandler.Delete)
	customersWrite.Post("/:id/isolir", customerHandler.Isolir)
	customersWrite.Post("/:id/activate", customerHandler.Activate)
	customersWrite.Post("/:id/change-package", customerHandler.ChangePackage)
	customersWrite.Post("/bulk/isolir", customerHandler.BulkIsolir)
	customersWrite.Post("/bulk/activate", customerHandler.BulkActivate)
	customersWrite.Post("/bulk/notification", customerHandler.BulkNotify)
	customersWrite.Post("/bulk/change-package", customerHandler.BulkChangePackage)
	customersWrite.Post("/bulk/edit", customerHandler.BulkEdit)

	// Regenerate payment link — admin + operator (write)
	customersWrite.Post("/:id/payment-link/regenerate", cfg.GatewayHandler.RegeneratePaymentLink)

	// Routes accessible by tenant_admin only (import, export, bulk delete)
	customersAdmin := customers.Group("")
	customersAdmin.Use(middleware.RBAC(domain.RBACConfig{
		AllowedRoles: []domain.UserRole{domain.RoleTenantAdmin},
	}))
	customersAdmin.Get("/export", customerHandler.Export)
	customersAdmin.Get("/import/template", customerHandler.ImportTemplate)
	customersAdmin.Post("/import", customerHandler.Import)
	customersAdmin.Delete("/bulk", customerHandler.BulkDelete)
	customersAdmin.Post("/:id/reactivate", cfg.IsolirHandler.Reactivate)

	// --- Area routes (auth + tenant + RBAC) ---
	areas := api.Group("/areas")
	areas.Use(middleware.RBAC(domain.RBACConfig{
		AllowedRoles: []domain.UserRole{
			domain.RoleTenantAdmin, domain.RoleOperator,
		},
	}))
	areas.Get("/", areaHandler.List)
	areas.Post("/", areaHandler.Create)
	areas.Get("/:id", areaHandler.Get)
	areas.Put("/:id", areaHandler.Update)
	areas.Delete("/:id", areaHandler.Delete)

	// --- Package routes (auth + tenant + RBAC) ---
	packageHandler := cfg.PackageHandler
	packages := api.Group("/packages")

	// Routes accessible by admin, operator, kasir(GET only)
	packagesRead := packages.Group("")
	packagesRead.Use(middleware.RBAC(domain.RBACConfig{
		AllowedRoles: []domain.UserRole{
			domain.RoleTenantAdmin, domain.RoleOperator, domain.RoleKasir,
		},
		MethodRestrictions: map[domain.UserRole][]string{
			domain.RoleKasir: {"GET"},
		},
	}))
	packagesRead.Get("/", packageHandler.List)
	packagesRead.Get("/:id", packageHandler.Get)

	// Routes accessible by tenant_admin only (write operations)
	packagesAdmin := packages.Group("")
	packagesAdmin.Use(middleware.RBAC(domain.RBACConfig{
		AllowedRoles: []domain.UserRole{domain.RoleTenantAdmin},
	}))
	packagesAdmin.Post("/", packageHandler.Create)
	packagesAdmin.Put("/:id", packageHandler.Update)
	packagesAdmin.Delete("/:id", packageHandler.Delete)
	packagesAdmin.Post("/:id/activate", packageHandler.Activate)
	packagesAdmin.Post("/:id/deactivate", packageHandler.Deactivate)
	packagesAdmin.Post("/:id/duplicate", packageHandler.Duplicate)

	// --- Admin reseller routes (auth + tenant + RBAC) ---
	resellerHandler := cfg.ResellerHandler
	resellerActionHandler := cfg.ResellerActionHandler
	resellers := api.Group("/resellers")

	// Routes accessible by admin + operator (GET-only untuk operator)
	resellersRead := resellers.Group("")
	resellersRead.Use(middleware.RBAC(domain.RBACConfig{
		AllowedRoles: []domain.UserRole{
			domain.RoleTenantAdmin, domain.RoleOperator,
		},
		MethodRestrictions: map[domain.UserRole][]string{
			domain.RoleOperator: {"GET"},
		},
	}))
	resellersRead.Get("/", resellerHandler.List)
	resellersRead.Get("/:id", resellerHandler.Get)

	// Routes accessible by tenant_admin only (write operations)
	resellersAdmin := resellers.Group("")
	resellersAdmin.Use(middleware.RBAC(domain.RBACConfig{
		AllowedRoles: []domain.UserRole{domain.RoleTenantAdmin},
	}))
	resellersAdmin.Post("/", resellerHandler.Create)
	resellersAdmin.Put("/:id", resellerHandler.Update)
	resellersAdmin.Post("/:id/suspend", resellerActionHandler.Suspend)
	resellersAdmin.Post("/:id/activate", resellerActionHandler.Activate)
	resellersAdmin.Post("/:id/deactivate", resellerActionHandler.Deactivate)
	resellersAdmin.Post("/:id/reset-password", resellerActionHandler.ResetPassword)
	resellersAdmin.Post("/:id/deposit", resellerActionHandler.Deposit)
	resellersAdmin.Post("/:id/withdraw", resellerActionHandler.Withdraw)

	// --- Admin voucher routes (auth + tenant + RBAC) ---
	voucherHandler := cfg.VoucherHandler
	voucherPrintHandler := cfg.VoucherPrintHandler
	vouchers := api.Group("/vouchers")

	// Routes accessible by admin + operator (GET-only untuk operator)
	vouchersRead := vouchers.Group("")
	vouchersRead.Use(middleware.RBAC(domain.RBACConfig{
		AllowedRoles: []domain.UserRole{
			domain.RoleTenantAdmin, domain.RoleOperator,
		},
		MethodRestrictions: map[domain.UserRole][]string{
			domain.RoleOperator: {"GET"},
		},
	}))
	vouchersRead.Get("/", voucherHandler.List)
	vouchersRead.Get("/:id", voucherHandler.Get)

	// Routes accessible by tenant_admin only (write operations + export)
	vouchersAdmin := vouchers.Group("")
	vouchersAdmin.Use(middleware.RBAC(domain.RBACConfig{
		AllowedRoles: []domain.UserRole{domain.RoleTenantAdmin},
	}))
	vouchersAdmin.Post("/generate", voucherHandler.Generate)
	vouchersAdmin.Post("/activate", voucherHandler.Activate)
	vouchersAdmin.Post("/bulk/print", voucherPrintHandler.BulkPrint)
	vouchersAdmin.Post("/bulk/void", voucherHandler.BulkVoid)
	vouchersAdmin.Post("/bulk/assign", voucherHandler.BulkAssign)
	vouchersAdmin.Get("/export", voucherHandler.Export)

	// --- Invoice routes (auth + tenant + RBAC) ---
	invoiceHandler := cfg.InvoiceHandler
	invoiceActionHandler := cfg.InvoiceActionHandler
	invoices := api.Group("/invoices")

	// Routes accessible by admin, operator, kasir (GET-only)
	invoicesRead := invoices.Group("")
	invoicesRead.Use(middleware.RBAC(domain.RBACConfig{
		AllowedRoles: []domain.UserRole{
			domain.RoleTenantAdmin, domain.RoleOperator, domain.RoleKasir,
		},
		MethodRestrictions: map[domain.UserRole][]string{
			domain.RoleKasir:    {"GET"},
			domain.RoleOperator: {"GET"},
		},
	}))
	invoicesRead.Get("/", invoiceHandler.List)
	invoicesRead.Get("/summary", invoiceHandler.Summary)
	invoicesRead.Get("/:id", invoiceHandler.Get)
	invoicesRead.Get("/:id/pdf", invoiceHandler.PDF)
	invoicesRead.Get("/:id/audit-logs", invoiceHandler.AuditLogs)

	// Payment links untuk invoice — menggunakan group invoicesRead yang sudah ada
	invoicesRead.Get("/:id/payment-links", cfg.GatewayHandler.GetInvoicePaymentLinks)

	// Routes accessible by admin + kasir (write: record payment)
	invoicesWrite := invoices.Group("")
	invoicesWrite.Use(middleware.RBAC(domain.RBACConfig{
		AllowedRoles: []domain.UserRole{
			domain.RoleTenantAdmin, domain.RoleKasir,
		},
	}))
	invoicesWrite.Post("/:id/payment", invoiceActionHandler.RecordPayment)

	// Routes accessible by tenant_admin only (write operations)
	invoicesAdmin := invoices.Group("")
	invoicesAdmin.Use(middleware.RBAC(domain.RBACConfig{
		AllowedRoles: []domain.UserRole{domain.RoleTenantAdmin},
	}))
	invoicesAdmin.Post("/", invoiceHandler.Create)
	invoicesAdmin.Post("/prepaid", invoiceHandler.CreatePrepaid)
	invoicesAdmin.Put("/:id", invoiceHandler.Edit)
	invoicesAdmin.Post("/:id/cancel", invoiceActionHandler.Cancel)
	invoicesAdmin.Post("/:id/waive-penalty", cfg.IsolirHandler.WaivePenalty)
	invoicesAdmin.Post("/bulk/reminder", invoiceActionHandler.BulkReminder)
	invoicesAdmin.Post("/bulk/cancel", invoiceActionHandler.BulkCancel)
	invoicesAdmin.Post("/bulk/pdf", invoiceActionHandler.BulkPDF)
	invoicesAdmin.Get("/export", invoiceActionHandler.ExportCSV)

	// --- Recurring item routes (admin-only, nested under customers) ---
	recurringItemHandler := cfg.RecurringItemHandler
	recurringItems := customers.Group("/:id/recurring-items")
	recurringItems.Use(middleware.RBAC(domain.RBACConfig{
		AllowedRoles: []domain.UserRole{domain.RoleTenantAdmin},
	}))
	recurringItems.Get("/", recurringItemHandler.List)
	recurringItems.Post("/", recurringItemHandler.Create)
	recurringItems.Put("/:item_id", recurringItemHandler.Update)
	recurringItems.Delete("/:item_id", recurringItemHandler.Delete)

	// --- Credit/Debit note routes (admin-only) ---
	creditNoteHandler := cfg.CreditNoteHandler
	debitNoteHandler := cfg.DebitNoteHandler

	creditNotes := api.Group("/credit-notes")
	creditNotes.Use(middleware.RBAC(domain.RBACConfig{
		AllowedRoles: []domain.UserRole{domain.RoleTenantAdmin},
	}))
	creditNotes.Post("/", creditNoteHandler.Create)

	debitNotes := api.Group("/debit-notes")
	debitNotes.Use(middleware.RBAC(domain.RBACConfig{
		AllowedRoles: []domain.UserRole{domain.RoleTenantAdmin},
	}))
	debitNotes.Post("/", debitNoteHandler.Create)

	// --- Payment routes (auth + tenant + RBAC) ---
	paymentHandler := cfg.PaymentHandler
	payments := api.Group("/payments")

	// Routes accessible by admin + kasir (read + record payment)
	paymentsReadWrite := payments.Group("")
	paymentsReadWrite.Use(middleware.RBAC(domain.RBACConfig{
		AllowedRoles: []domain.UserRole{
			domain.RoleTenantAdmin, domain.RoleKasir,
		},
	}))
	paymentsReadWrite.Get("/", paymentHandler.List)
	paymentsReadWrite.Get("/summary", paymentHandler.Summary)
	paymentsReadWrite.Get("/quick/customers", paymentHandler.SearchCustomers)
	paymentsReadWrite.Get("/quick/customers/:customer_id/invoices", paymentHandler.GetOpenInvoices)
	paymentsReadWrite.Post("/multi", paymentHandler.RecordMultiPayment)
	paymentsReadWrite.Post("/pay-all", paymentHandler.PayAll)
	paymentsReadWrite.Get("/:payment_id/receipt", paymentHandler.GetReceipt)
	paymentsReadWrite.Post("/:payment_id/proof", paymentHandler.UploadProof)
	paymentsReadWrite.Get("/:payment_id/proof", paymentHandler.GetProof)

	// Routes accessible by tenant_admin only (void, bulk import)
	paymentsAdmin := payments.Group("")
	paymentsAdmin.Use(middleware.RBAC(domain.RBACConfig{
		AllowedRoles: []domain.UserRole{domain.RoleTenantAdmin},
	}))
	paymentsAdmin.Post("/:payment_id/void", paymentHandler.VoidPayment)
	paymentsAdmin.Post("/import", paymentHandler.BulkImport)

	// --- Payment link webhook query routes (auth + tenant + RBAC, admin + kasir) ---
	paymentLinks := api.Group("/payment-links")
	paymentLinks.Use(middleware.RBAC(domain.RBACConfig{
		AllowedRoles: []domain.UserRole{
			domain.RoleTenantAdmin, domain.RoleKasir,
		},
	}))
	paymentLinks.Get("/:id/webhooks", cfg.GatewayHandler.GetPaymentLinkWebhooks)

	// --- Isolir routes (auth + tenant + RBAC, tenant_admin + operator) ---
	isolir := api.Group("/isolir")
	isolir.Use(middleware.RBAC(domain.RBACConfig{
		AllowedRoles: []domain.UserRole{
			domain.RoleTenantAdmin, domain.RoleOperator,
		},
	}))
	isolir.Post("/sync/:customer_id", cfg.IsolirHandler.ManualSync)
	isolir.Post("/sync-all", cfg.IsolirHandler.ManualSyncAll)
	isolir.Get("/pending-syncs", cfg.IsolirHandler.ListPendingSyncs)
	isolir.Get("/summary", cfg.IsolirHandler.Summary)

	// --- Report routes (auth + tenant + RBAC) ---
	reportHandler := cfg.ReportHandler
	exportHandler := cfg.ExportHandler
	scheduleHandler := cfg.ScheduleHandler
	kpiHandler := cfg.KPIHandler
	forecastHandler := cfg.ForecastHandler
	comparisonHandler := cfg.ComparisonHandler
	customReportHandler := cfg.CustomReportHandler
	dashboardHandler := cfg.DashboardHandler

	reports := api.Group("/reports")

	// Reports read — admin + operator + kasir (GET only)
	reportsRead := reports.Group("")
	reportsRead.Use(middleware.RBAC(domain.RBACConfig{
		AllowedRoles: []domain.UserRole{
			domain.RoleTenantAdmin, domain.RoleOperator, domain.RoleKasir,
		},
		MethodRestrictions: map[domain.UserRole][]string{
			domain.RoleKasir:    {"GET"},
			domain.RoleOperator: {"GET"},
		},
	}))

	// Financial reports
	reportsRead.Get("/financial/revenue", reportHandler.Revenue)
	reportsRead.Get("/financial/aging", reportHandler.Aging)
	reportsRead.Get("/financial/payments", reportHandler.Payments)
	reportsRead.Get("/financial/vouchers", reportHandler.Vouchers)
	reportsRead.Get("/financial/profit-loss", reportHandler.ProfitLoss)
	reportsRead.Get("/financial/revenue-by-area", reportHandler.RevenueByArea)

	// Customer reports
	reportsRead.Get("/customers/growth", reportHandler.CustomerGrowth)
	reportsRead.Get("/customers/distribution", reportHandler.CustomerDistribution)
	reportsRead.Get("/customers/churn", reportHandler.ChurnAnalysis)

	// Network reports
	reportsRead.Get("/network/uptime", reportHandler.Uptime)
	reportsRead.Get("/network/traffic", reportHandler.Traffic)
	reportsRead.Get("/network/signal-quality", reportHandler.SignalQuality)
	reportsRead.Get("/network/capacity", reportHandler.Capacity)

	// Operational reports
	reportsRead.Get("/operational/activity", reportHandler.Activity)
	reportsRead.Get("/operational/notifications", reportHandler.Notifications)
	reportsRead.Get("/operational/sync", reportHandler.Sync)

	// Comparison, forecast, dashboard
	reportsRead.Get("/comparison", comparisonHandler.Compare)
	reportsRead.Get("/forecast", forecastHandler.Forecast)
	reportsRead.Get("/dashboard", dashboardHandler.Dashboard)

	// Export status (GET — read-only)
	reportsRead.Get("/export/:job_id", exportHandler.Status)

	// Reports admin — tenant_admin only (export, schedules, KPI, custom reports)
	reportsAdmin := reports.Group("")
	reportsAdmin.Use(middleware.RBAC(domain.RBACConfig{
		AllowedRoles: []domain.UserRole{domain.RoleTenantAdmin},
	}))

	// Export (async PDF/XLSX, sync CSV)
	reportsAdmin.Post("/export", exportHandler.RequestExport)

	// Schedules CRUD
	reportsAdmin.Get("/schedules", scheduleHandler.List)
	reportsAdmin.Post("/schedules", scheduleHandler.Create)
	reportsAdmin.Put("/schedules/:id", scheduleHandler.Update)
	reportsAdmin.Delete("/schedules/:id", scheduleHandler.Delete)

	// KPI targets
	reportsAdmin.Get("/kpi-targets", kpiHandler.Get)
	reportsAdmin.Put("/kpi-targets", kpiHandler.Update)

	// Custom reports
	reportsAdmin.Post("/custom/preview", customReportHandler.Preview)
	reportsAdmin.Get("/custom/templates", customReportHandler.ListTemplates)
	reportsAdmin.Post("/custom/templates", customReportHandler.CreateTemplate)
	reportsAdmin.Delete("/custom/templates/:id", customReportHandler.DeleteTemplate)

	// --- Expense routes (auth + tenant + RBAC, tenant_admin only) ---
	expenseHandler := cfg.ExpenseHandler
	expenses := api.Group("/expenses")
	expenses.Use(middleware.RBAC(domain.RBACConfig{
		AllowedRoles: []domain.UserRole{domain.RoleTenantAdmin},
	}))

	// Expenses CRUD
	expenses.Get("/", expenseHandler.List)
	expenses.Post("/", expenseHandler.Create)
	expenses.Put("/:id", expenseHandler.Update)
	expenses.Delete("/:id", expenseHandler.Delete)

	// Expense categories CRUD
	expenses.Get("/categories", expenseHandler.ListCategories)
	expenses.Post("/categories", expenseHandler.CreateCategory)
	expenses.Put("/categories/:id", expenseHandler.UpdateCategory)
	expenses.Delete("/categories/:id", expenseHandler.DeleteCategory)
}
