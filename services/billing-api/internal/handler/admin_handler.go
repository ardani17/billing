// admin_handler.go menangani HTTP request untuk fitur super admin.
// Termasuk: start impersonation dan stop impersonation.
package handler

import (
	"errors"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/ispboss/ispboss/pkg/auth"
	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
	"github.com/ispboss/ispboss/services/billing-api/internal/usecase"
)

// AdminHandler menangani HTTP request untuk fitur super admin (impersonation).
type AdminHandler struct {
	impersonationUsecase *usecase.ImpersonationUsecase
	db                   *pgxpool.Pool
	validate             *validator.Validate
	logger               zerolog.Logger
}

// NewAdminHandler membuat instance baru AdminHandler.
func NewAdminHandler(impersonationUsecase *usecase.ImpersonationUsecase, db *pgxpool.Pool, logger zerolog.Logger) *AdminHandler {
	return &AdminHandler{
		impersonationUsecase: impersonationUsecase,
		db:                   db,
		validate:             validator.New(),
		logger:               logger,
	}
}

// Start menangani POST /v1/admin/impersonate.
// Memulai impersonasi user target oleh super admin.
func (h *AdminHandler) Start(c *fiber.Ctx) error {
	impersonatorID, ok := c.Locals("user_id").(string)
	if !ok || impersonatorID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "user tidak terautentikasi")
	}

	var req domain.ImpersonateRequest
	if err := c.BodyParser(&req); err != nil {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "request body tidak valid")
	}

	if err := h.validate.Struct(req); err != nil {
		var ve validator.ValidationErrors
		if errors.As(err, &ve) {
			return domain.ErrorResponse(c, fiber.StatusUnprocessableEntity, "VALIDATION_ERROR", "validasi gagal", mapValidationErrors(ve)...)
		}
		return domain.ErrorResponse(c, fiber.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
	}

	tokenPair, err := h.impersonationUsecase.StartImpersonation(c.Context(), impersonatorID, req)
	if err != nil {
		return h.mapAdminError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, tokenPair)
}

// Stop menangani POST /v1/admin/stop-impersonate.
// Menghentikan impersonasi dan mengembalikan JWT ke claims super admin asli.
// JWT saat impersonasi berisi impersonator_id yang digunakan untuk mengambil data super admin.
func (h *AdminHandler) Stop(c *fiber.Ctx) error {
	// Ambil impersonator_id dari JWT claims yang disimpan oleh auth middleware.
	// Saat impersonasi aktif, JWT berisi claims target user + impersonator_id.
	impersonatorID := ""

	if claims, ok := c.Locals("claims").(*auth.Claims); ok && claims != nil {
		impersonatorID = claims.ImpersonatorID
	}

	if impersonatorID == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "impersonator_id tidak ditemukan, pastikan sedang dalam mode impersonasi")
	}

	tokenPair, err := h.impersonationUsecase.StopImpersonation(c.Context(), impersonatorID)
	if err != nil {
		return h.mapAdminError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, tokenPair)
}

type platformTenantRow struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	OwnerName      string    `json:"owner_name"`
	OwnerEmail     string    `json:"owner_email"`
	Domain         string    `json:"domain"`
	Plan           string    `json:"plan"`
	Status         string    `json:"status"`
	Health         string    `json:"health"`
	MonthlyRevenue int64     `json:"monthly_revenue"`
	CustomerCount  int64     `json:"customer_count"`
	OpenInvoice    int64     `json:"open_invoice_count"`
	RouterCount    int64     `json:"router_count"`
	OltCount       int64     `json:"olt_count"`
	ResellerCount  int64     `json:"reseller_count"`
	LastActivityAt time.Time `json:"last_activity_at"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type platformAuditRow struct {
	ID         string    `json:"id"`
	TenantID   string    `json:"tenant_id"`
	TenantName string    `json:"tenant_name"`
	ActorName  string    `json:"actor_name"`
	Action     string    `json:"action"`
	EntityType string    `json:"entity_type"`
	EntityID   string    `json:"entity_id"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
}

type platformHealthRow struct {
	Service   string `json:"service"`
	Region    string `json:"region"`
	LatencyMs int64  `json:"latency_ms"`
	Uptime    string `json:"uptime"`
	Status    string `json:"status"`
}

func planMonthlyRevenue(plan string) int64 {
	switch plan {
	case "growth", "pro":
		return 799000
	case "scale", "enterprise":
		return 1499000
	default:
		return 299000
	}
}

func tenantHealth(status string, openInvoices, routers, olts int64) string {
	if status == "suspended" || status == "cancelled" {
		return "blocked"
	}
	if openInvoices > 0 || routers == 0 {
		return "warning"
	}
	if olts == 0 {
		return "normal"
	}
	return "normal"
}

func scanPlatformTenant(rows pgx.Rows) (*platformTenantRow, error) {
	var tenant platformTenantRow
	var lastActivity *time.Time
	if err := rows.Scan(
		&tenant.ID,
		&tenant.Name,
		&tenant.Domain,
		&tenant.Plan,
		&tenant.Status,
		&tenant.CreatedAt,
		&tenant.UpdatedAt,
		&tenant.OwnerName,
		&tenant.OwnerEmail,
		&tenant.CustomerCount,
		&tenant.OpenInvoice,
		&tenant.RouterCount,
		&tenant.OltCount,
		&tenant.ResellerCount,
		&lastActivity,
	); err != nil {
		return nil, err
	}
	tenant.MonthlyRevenue = planMonthlyRevenue(tenant.Plan)
	tenant.Health = tenantHealth(tenant.Status, tenant.OpenInvoice, tenant.RouterCount, tenant.OltCount)
	if lastActivity != nil {
		tenant.LastActivityAt = *lastActivity
	} else {
		tenant.LastActivityAt = tenant.UpdatedAt
	}
	return &tenant, nil
}

func platformTenantQuery(where string) string {
	return `
		SELECT
			t.id::text,
			t.name,
			COALESCE(t.domain, ''),
			t.plan,
			t.status,
			t.created_at,
			t.updated_at,
			COALESCE(owner.name, ''),
			COALESCE(owner.email, ''),
			COUNT(DISTINCT c.id)::bigint,
			COUNT(DISTINCT CASE WHEN i.status IN ('belum_bayar', 'terlambat', 'bayar_sebagian') THEN i.id END)::bigint,
			COUNT(DISTINCT r.id)::bigint,
			COUNT(DISTINCT o.id)::bigint,
			COUNT(DISTINCT rs.id)::bigint,
			MAX(a.created_at)
		FROM tenants t
		LEFT JOIN LATERAL (
			SELECT name, email
			FROM users
			WHERE tenant_id = t.id AND role = 'tenant_admin'
			ORDER BY created_at ASC
			LIMIT 1
		) owner ON TRUE
		LEFT JOIN customers c ON c.tenant_id = t.id
		LEFT JOIN invoices i ON i.tenant_id = t.id
		LEFT JOIN routers r ON r.tenant_id = t.id
		LEFT JOIN olts o ON o.tenant_id = t.id
		LEFT JOIN resellers rs ON rs.tenant_id = t.id
		LEFT JOIN audit_logs a ON a.tenant_id = t.id
		` + where + `
		GROUP BY t.id, t.name, t.domain, t.plan, t.status, t.created_at, t.updated_at, owner.name, owner.email
		ORDER BY t.created_at DESC
	`
}

func (h *AdminHandler) listPlatformTenants(ctx *fiber.Ctx) ([]platformTenantRow, error) {
	rows, err := h.db.Query(ctx.Context(), platformTenantQuery(""))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tenants := make([]platformTenantRow, 0)
	for rows.Next() {
		tenant, err := scanPlatformTenant(rows)
		if err != nil {
			return nil, err
		}
		tenants = append(tenants, *tenant)
	}

	return tenants, rows.Err()
}

// PlatformOverview menangani GET /v1/admin/platform/overview.
func (h *AdminHandler) PlatformOverview(c *fiber.Ctx) error {
	tenants, err := h.listPlatformTenants(c)
	if err != nil {
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal mengambil overview platform")
	}

	var activeTenants, trialTenants, suspendedTenants, totalCustomers, mrr int64
	for _, tenant := range tenants {
		if tenant.Status == "active" {
			activeTenants++
		}
		if tenant.Status == "trial" {
			trialTenants++
		}
		if tenant.Status == "suspended" {
			suspendedTenants++
		}
		totalCustomers += tenant.CustomerCount
		if tenant.Status == "active" {
			mrr += tenant.MonthlyRevenue
		}
	}

	audit, _ := h.platformAudit(c, 8)
	health := h.platformHealth(c)

	return domain.SuccessResponse(c, fiber.StatusOK, fiber.Map{
		"stats": fiber.Map{
			"tenant_total":      len(tenants),
			"tenant_active":     activeTenants,
			"tenant_trial":      trialTenants,
			"tenant_suspended":  suspendedTenants,
			"customer_total":    totalCustomers,
			"monthly_recurring": mrr,
		},
		"tenants": tenants,
		"health":  health,
		"audit":   audit,
	})
}

// PlatformTenants menangani GET /v1/admin/platform/tenants.
func (h *AdminHandler) PlatformTenants(c *fiber.Ctx) error {
	tenants, err := h.listPlatformTenants(c)
	if err != nil {
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal mengambil daftar tenant")
	}
	return domain.SuccessResponse(c, fiber.StatusOK, fiber.Map{"data": tenants})
}

// PlatformTenantDetail menangani GET /v1/admin/platform/tenants/:id.
func (h *AdminHandler) PlatformTenantDetail(c *fiber.Ctx) error {
	tenantID := c.Params("id")
	rows, err := h.db.Query(c.Context(), platformTenantQuery("WHERE t.id = $1"), tenantID)
	if err != nil {
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal mengambil tenant")
	}
	defer rows.Close()

	if !rows.Next() {
		return domain.ErrorResponse(c, fiber.StatusNotFound, "TENANT_NOT_FOUND", "tenant tidak ditemukan")
	}

	tenant, err := scanPlatformTenant(rows)
	if err != nil {
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal membaca tenant")
	}

	audit, _ := h.platformAuditByTenant(c, tenantID, 10)
	return domain.SuccessResponse(c, fiber.StatusOK, fiber.Map{
		"tenant": tenant,
		"audit":  audit,
	})
}

// PlatformAudit menangani GET /v1/admin/platform/audit.
func (h *AdminHandler) PlatformAudit(c *fiber.Ctx) error {
	audit, err := h.platformAudit(c, 50)
	if err != nil {
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal mengambil audit global")
	}
	return domain.SuccessResponse(c, fiber.StatusOK, fiber.Map{"data": audit})
}

func (h *AdminHandler) platformAudit(c *fiber.Ctx, limit int) ([]platformAuditRow, error) {
	return h.platformAuditByTenant(c, "", limit)
}

func (h *AdminHandler) platformAuditByTenant(c *fiber.Ctx, tenantID string, limit int) ([]platformAuditRow, error) {
	where := ""
	args := []interface{}{limit}
	if tenantID != "" {
		where = "WHERE a.tenant_id = $2"
		args = append(args, tenantID)
	}
	rows, err := h.db.Query(c.Context(), `
		SELECT
			a.id::text,
			a.tenant_id::text,
			t.name,
			a.actor_name,
			a.action,
			a.entity_type,
			a.entity_id::text,
			'success',
			a.created_at
		FROM audit_logs a
		JOIN tenants t ON t.id = a.tenant_id
		`+where+`
		ORDER BY a.created_at DESC
		LIMIT $1
	`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	audit := make([]platformAuditRow, 0)
	for rows.Next() {
		var item platformAuditRow
		if err := rows.Scan(&item.ID, &item.TenantID, &item.TenantName, &item.ActorName, &item.Action, &item.EntityType, &item.EntityID, &item.Status, &item.CreatedAt); err != nil {
			return nil, err
		}
		audit = append(audit, item)
	}
	return audit, rows.Err()
}

// PlatformHealth menangani GET /v1/admin/platform/health.
func (h *AdminHandler) PlatformHealth(c *fiber.Ctx) error {
	return domain.SuccessResponse(c, fiber.StatusOK, fiber.Map{"data": h.platformHealth(c)})
}

func (h *AdminHandler) platformHealth(c *fiber.Ctx) []platformHealthRow {
	start := time.Now()
	status := "online"
	if err := h.db.Ping(c.Context()); err != nil {
		status = "offline"
	}
	latency := time.Since(start).Milliseconds()

	return []platformHealthRow{
		{Service: "Billing API", Region: "Local", LatencyMs: latency, Uptime: "live", Status: status},
		{Service: "PostgreSQL", Region: "Local", LatencyMs: latency, Uptime: "live", Status: status},
		{Service: "Network Service", Region: "Local", LatencyMs: 0, Uptime: "docker", Status: "online"},
		{Service: "Notification Service", Region: "Local", LatencyMs: 0, Uptime: "docker", Status: "online"},
	}
}

// PlatformSubscriptions menangani GET /v1/admin/platform/subscriptions.
func (h *AdminHandler) PlatformSubscriptions(c *fiber.Ctx) error {
	tenants, err := h.listPlatformTenants(c)
	if err != nil {
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal mengambil subscription")
	}
	items := make([]fiber.Map, 0, len(tenants))
	for _, tenant := range tenants {
		items = append(items, fiber.Map{
			"tenant_id": tenant.ID,
			"tenant":    tenant.Name,
			"plan":      tenant.Plan,
			"amount":    tenant.MonthlyRevenue,
			"status":    tenant.Status,
			"due_date":  tenant.CreatedAt.AddDate(0, 1, 0),
		})
	}
	return domain.SuccessResponse(c, fiber.StatusOK, fiber.Map{"data": items})
}

// PlatformSupport menangani GET /v1/admin/platform/support.
func (h *AdminHandler) PlatformSupport(c *fiber.Ctx) error {
	return domain.SuccessResponse(c, fiber.StatusOK, fiber.Map{"data": []fiber.Map{}})
}

// mapAdminError memetakan domain error ke HTTP error response untuk admin operations.
func (h *AdminHandler) mapAdminError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, domain.ErrForbidden):
		return domain.ErrorResponse(c, fiber.StatusForbidden, "FORBIDDEN", err.Error())
	case errors.Is(err, domain.ErrUserNotFound):
		return domain.ErrorResponse(c, fiber.StatusNotFound, "USER_NOT_FOUND", err.Error())
	default:
		h.logger.Error().Err(err).Msg("internal error pada admin handler")
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "terjadi kesalahan internal")
	}
}
