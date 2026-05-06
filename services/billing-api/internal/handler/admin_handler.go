// admin_handler.go menangani HTTP request untuk fitur super admin.
// Termasuk: start impersonation dan stop impersonation.
package handler

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
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
	h.auditOwnerAction(c, "impersonation", req.UserID, "impersonate-start", req.Reason, map[string]interface{}{
		"tenant_id":       req.TenantID,
		"target_user_id":  req.UserID,
		"impersonator_id": impersonatorID,
	})

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
	targetTenantID, targetUserID := "", ""
	if claims, ok := c.Locals("claims").(*auth.Claims); ok && claims != nil {
		targetTenantID = claims.TenantID
		targetUserID = claims.UserID
	}
	h.auditOwnerAction(c, "impersonation", impersonatorID, "impersonate-stop", "Stop impersonate", map[string]interface{}{
		"tenant_id":       targetTenantID,
		"target_user_id":  targetUserID,
		"impersonator_id": impersonatorID,
	})

	return domain.SuccessResponse(c, fiber.StatusOK, tokenPair)
}

type platformTenantRow struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	OwnerName      string    `json:"owner_name"`
	OwnerEmail     string    `json:"owner_email"`
	Domain         string    `json:"domain"`
	DomainStatus   string    `json:"domain_status"`
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
	Modules        []string  `json:"modules"`
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
	Service     string `json:"service"`
	Region      string `json:"region"`
	LatencyMs   int64  `json:"latency_ms"`
	Uptime      string `json:"uptime"`
	Status      string `json:"status"`
	LastChecked string `json:"last_checked,omitempty"`
	LastError   string `json:"last_error,omitempty"`
}

type platformSubscriptionRow struct {
	TenantID         string     `json:"tenant_id"`
	Tenant           string     `json:"tenant"`
	Plan             string     `json:"plan"`
	Amount           int64      `json:"amount"`
	Currency         string     `json:"currency"`
	Status           string     `json:"status"`
	TrialEndsAt      *time.Time `json:"trial_ends_at,omitempty"`
	CurrentPeriodEnd time.Time  `json:"current_period_end"`
	DueDate          time.Time  `json:"due_date"`
	Modules          []string   `json:"modules"`
	CustomerCount    int64      `json:"customer_count"`
	OpenInvoiceCount int64      `json:"open_invoice_count"`
	MonthlyRevenue   int64      `json:"monthly_revenue"`
}

type platformUpgradeRequestRow struct {
	ID               string     `json:"id"`
	TenantID         string     `json:"tenant_id"`
	TenantName       string     `json:"tenant_name"`
	RequestedPlan    string     `json:"requested_plan"`
	RequestedModules []string   `json:"requested_modules"`
	Message          string     `json:"message"`
	Status           string     `json:"status"`
	ProcessedBy      string     `json:"processed_by"`
	ProcessedReason  string     `json:"processed_reason"`
	ProcessedAt      *time.Time `json:"processed_at,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

type platformSupportTicketRow struct {
	ID           string    `json:"id"`
	TenantID     string    `json:"tenant_id"`
	TenantName   string    `json:"tenant_name"`
	Subject      string    `json:"subject"`
	Description  string    `json:"description"`
	Priority     string    `json:"priority"`
	Status       string    `json:"status"`
	AssigneeID   string    `json:"assignee_id"`
	CreatedBy    string    `json:"created_by"`
	CommentCount int64     `json:"comment_count"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type platformSupportCommentRow struct {
	ID         string    `json:"id"`
	TicketID   string    `json:"ticket_id"`
	AuthorID   string    `json:"author_id"`
	AuthorRole string    `json:"author_role"`
	Body       string    `json:"body"`
	IsInternal bool      `json:"is_internal"`
	CreatedAt  time.Time `json:"created_at"`
}

type platformTenantActionRequest struct {
	Reason string `json:"reason" validate:"required"`
}

type platformTenantUpdateRequest struct {
	Name   string `json:"name" validate:"required"`
	Domain string `json:"domain"`
	Plan   string `json:"plan" validate:"required"`
	Status string `json:"status" validate:"required,oneof=trial active suspended cancelled"`
	Reason string `json:"reason" validate:"required"`
}

type platformTenantCreateRequest struct {
	Name       string `json:"name" validate:"required"`
	Domain     string `json:"domain"`
	Plan       string `json:"plan" validate:"required"`
	Status     string `json:"status" validate:"required,oneof=trial active suspended cancelled"`
	OwnerName  string `json:"owner_name" validate:"required"`
	OwnerEmail string `json:"owner_email" validate:"required,email"`
	Reason     string `json:"reason" validate:"required"`
}

type platformOwnerResetRequest struct {
	OwnerName  string `json:"owner_name" validate:"required"`
	OwnerEmail string `json:"owner_email" validate:"required,email"`
	Reason     string `json:"reason" validate:"required"`
}

type platformTenantModulesRequest struct {
	Modules []string `json:"modules"`
	Reason  string   `json:"reason" validate:"required"`
}

type platformUpgradeDecisionRequest struct {
	Reason string `json:"reason" validate:"required"`
}

type tenantUpgradeRequestCreate struct {
	RequestedPlan    string   `json:"requested_plan"`
	RequestedModules []string `json:"requested_modules"`
	Message          string   `json:"message"`
}

type platformSupportCreateRequest struct {
	TenantID    string `json:"tenant_id"`
	Subject     string `json:"subject" validate:"required"`
	Description string `json:"description"`
	Priority    string `json:"priority" validate:"required,oneof=low normal high urgent"`
}

type platformSupportCommentRequest struct {
	Body       string `json:"body" validate:"required"`
	IsInternal bool   `json:"is_internal"`
}

type platformSupportStatusRequest struct {
	Status string `json:"status" validate:"required,oneof=open in_progress waiting_tenant resolved closed"`
	Reason string `json:"reason"`
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

func platformPlanAmount(plan string) int64 {
	return planMonthlyRevenue(plan)
}

func (h *AdminHandler) actorID(c *fiber.Ctx) string {
	if userID, ok := c.Locals("user_id").(string); ok && userID != "" {
		return userID
	}
	return "22222222-2222-4222-8222-222222222222"
}

func (h *AdminHandler) actorName(c *fiber.Ctx) string {
	if claims, ok := c.Locals("claims").(*auth.Claims); ok && claims != nil && claims.Role != "" {
		return "Super Admin"
	}
	return "Super Admin"
}

func uniqueModules(modules []string) []string {
	allowed := map[string]bool{
		domain.ModuleBillingCore:  true,
		domain.ModuleMikroTik:     true,
		domain.ModuleFiberNetwork: true,
	}
	seen := map[string]bool{}
	result := []string{domain.ModuleBillingCore}
	seen[domain.ModuleBillingCore] = true
	for _, module := range modules {
		module = strings.TrimSpace(module)
		if !allowed[module] || seen[module] {
			continue
		}
		seen[module] = true
		result = append(result, module)
	}
	return result
}

func parseStringArray(raw []byte) []string {
	var values []string
	if len(raw) == 0 {
		return values
	}
	_ = json.Unmarshal(raw, &values)
	return values
}

func jsonStringArray(values []string) []byte {
	encoded, _ := json.Marshal(values)
	return encoded
}

func (h *AdminHandler) auditPlatformAction(c *fiber.Ctx, tenantID, entityType, entityID, action, reason string, changes map[string]interface{}) {
	if tenantID == "" || entityID == "" {
		return
	}
	changesJSON, _ := json.Marshal(changes)
	metadataJSON, _ := json.Marshal(map[string]interface{}{
		"reason": reason,
		"scope":  "super_admin",
	})
	_, err := h.db.Exec(c.Context(), `
		INSERT INTO audit_logs (tenant_id, entity_type, entity_id, action, actor_id, actor_name, changes, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb, $8::jsonb)
	`, tenantID, entityType, entityID, action, h.actorID(c), h.actorName(c), string(changesJSON), string(metadataJSON))
	if err != nil {
		h.logger.Warn().Err(err).Str("action", action).Str("tenant_id", tenantID).Msg("gagal menulis audit super admin")
	}
}

func (h *AdminHandler) auditOwnerAction(c *fiber.Ctx, entityType, entityID, action, reason string, changes map[string]interface{}) {
	if entityID == "" {
		return
	}
	changesJSON, _ := json.Marshal(redactPlatformPayload(changes))
	metadataJSON, _ := json.Marshal(map[string]interface{}{
		"reason": reason,
		"scope":  "platform",
	})
	_, err := h.db.Exec(c.Context(), `
		INSERT INTO platform_audit_logs (entity_type, entity_id, action, actor_id, actor_name, changes, metadata)
		VALUES ($1, $2, $3, $4, $5, $6::jsonb, $7::jsonb)
	`, entityType, entityID, action, h.actorID(c), h.actorName(c), string(changesJSON), string(metadataJSON))
	if err != nil {
		h.logger.Warn().Err(err).Str("action", action).Str("entity_type", entityType).Msg("gagal menulis audit platform")
	}
}

func validatePlatformTenantStatus(status string) string {
	switch status {
	case "trial", "active", "suspended", "cancelled":
		return status
	default:
		return "active"
	}
}

func subscriptionStatusForTenantStatus(status string) string {
	switch status {
	case "trial":
		return "trial"
	case "suspended":
		return "suspended"
	case "cancelled":
		return "cancelled"
	default:
		return "active"
	}
}

func scanPlatformTenant(rows pgx.Rows) (*platformTenantRow, error) {
	var tenant platformTenantRow
	var lastActivity *time.Time
	if err := rows.Scan(
		&tenant.ID,
		&tenant.Name,
		&tenant.Domain,
		&tenant.DomainStatus,
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
			COALESCE(t.domain_status, 'unverified'),
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
		GROUP BY t.id, t.name, t.domain, t.domain_status, t.plan, t.status, t.created_at, t.updated_at, owner.name, owner.email
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
		tenant.Modules, _ = h.platformTenantModules(ctx, tenant.ID)
		tenants = append(tenants, *tenant)
	}

	return tenants, rows.Err()
}

func (h *AdminHandler) platformTenantModules(c *fiber.Ctx, tenantID string) ([]string, error) {
	rows, err := h.db.Query(c.Context(), `
		SELECT module_code
		FROM tenant_modules
		WHERE tenant_id = $1 AND status = 'active'
		ORDER BY CASE module_code
			WHEN 'billing_core' THEN 1
			WHEN 'mikrotik' THEN 2
			WHEN 'fiber_network' THEN 3
			ELSE 4
		END
	`, tenantID)
	if err != nil {
		return []string{domain.ModuleBillingCore}, err
	}
	defer rows.Close()

	modules := []string{}
	for rows.Next() {
		var module string
		if err := rows.Scan(&module); err != nil {
			return []string{domain.ModuleBillingCore}, err
		}
		modules = append(modules, module)
	}
	return uniqueModules(modules), rows.Err()
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
	var pendingUpgrades, supportOpen, overdueSubscriptions int64
	_ = h.db.QueryRow(c.Context(), `SELECT COUNT(*)::bigint FROM tenant_upgrade_requests WHERE status = 'pending'`).Scan(&pendingUpgrades)
	_ = h.db.QueryRow(c.Context(), `SELECT COUNT(*)::bigint FROM support_tickets WHERE status IN ('open','in_progress','waiting_tenant')`).Scan(&supportOpen)
	_ = h.db.QueryRow(c.Context(), `SELECT COUNT(*)::bigint FROM platform_subscriptions WHERE status = 'overdue' OR (status IN ('trial','active') AND current_period_end < NOW())`).Scan(&overdueSubscriptions)

	return domain.SuccessResponse(c, fiber.StatusOK, fiber.Map{
		"stats": fiber.Map{
			"tenant_total":         len(tenants),
			"tenant_active":        activeTenants,
			"tenant_trial":         trialTenants,
			"tenant_suspended":     suspendedTenants,
			"customer_total":       totalCustomers,
			"monthly_recurring":    mrr,
			"upgrade_pending":      pendingUpgrades,
			"support_open":         supportOpen,
			"subscription_overdue": overdueSubscriptions,
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
	tenant.Modules, _ = h.platformTenantModules(c, tenantID)

	audit, _ := h.platformAuditByTenant(c, tenantID, 10)
	admins, _ := h.platformTenantAdmins(c, tenantID)
	subscription, _ := h.platformSubscription(c, tenantID)
	tickets, _ := h.platformSupportTickets(c, tenantID, "", "", "")
	return domain.SuccessResponse(c, fiber.StatusOK, fiber.Map{
		"tenant":       tenant,
		"audit":        audit,
		"admins":       admins,
		"subscription": subscription,
		"tickets":      tickets,
	})
}

func (h *AdminHandler) platformTenantAdmins(c *fiber.Ctx, tenantID string) ([]fiber.Map, error) {
	rows, err := h.db.Query(c.Context(), `
		SELECT id::text, name, email, status, last_login
		FROM users
		WHERE tenant_id = $1 AND role = 'tenant_admin'
		ORDER BY created_at ASC
	`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	admins := []fiber.Map{}
	for rows.Next() {
		var id, name, email, status string
		var lastLogin *time.Time
		if err := rows.Scan(&id, &name, &email, &status, &lastLogin); err != nil {
			return nil, err
		}
		admins = append(admins, fiber.Map{
			"id":         id,
			"name":       name,
			"email":      email,
			"status":     status,
			"last_login": lastLogin,
		})
	}
	return admins, rows.Err()
}

func (h *AdminHandler) platformSubscription(c *fiber.Ctx, tenantID string) (*platformSubscriptionRow, error) {
	row := h.db.QueryRow(c.Context(), `
		SELECT
			t.id::text,
			t.name,
			COALESCE(ps.plan_code, t.plan),
			COALESCE(ps.amount, CASE WHEN t.plan IN ('growth', 'pro') THEN 799000 WHEN t.plan IN ('scale', 'enterprise') THEN 1499000 ELSE 299000 END)::bigint,
			COALESCE(ps.currency, 'IDR'),
			COALESCE(ps.status, CASE WHEN t.status = 'trial' THEN 'trial' WHEN t.status = 'suspended' THEN 'suspended' WHEN t.status = 'cancelled' THEN 'cancelled' ELSE 'active' END),
			ps.trial_ends_at,
			COALESCE(ps.current_period_end, t.created_at + INTERVAL '1 month'),
			COUNT(DISTINCT c.id)::bigint,
			COUNT(DISTINCT CASE WHEN i.status IN ('belum_bayar', 'terlambat', 'bayar_sebagian') THEN i.id END)::bigint
		FROM tenants t
		LEFT JOIN platform_subscriptions ps ON ps.tenant_id = t.id
		LEFT JOIN customers c ON c.tenant_id = t.id
		LEFT JOIN invoices i ON i.tenant_id = t.id
		WHERE t.id = $1
		GROUP BY t.id, t.name, t.plan, t.status, t.created_at, ps.plan_code, ps.amount, ps.currency, ps.status, ps.trial_ends_at, ps.current_period_end
	`, tenantID)

	var item platformSubscriptionRow
	var trialEndsAt *time.Time
	if err := row.Scan(
		&item.TenantID,
		&item.Tenant,
		&item.Plan,
		&item.Amount,
		&item.Currency,
		&item.Status,
		&trialEndsAt,
		&item.CurrentPeriodEnd,
		&item.CustomerCount,
		&item.OpenInvoiceCount,
	); err != nil {
		return nil, err
	}
	item.TrialEndsAt = trialEndsAt
	item.DueDate = item.CurrentPeriodEnd
	item.Modules, _ = h.platformTenantModules(c, tenantID)
	item.MonthlyRevenue = item.Amount
	return &item, nil
}

// PlatformTenantCreate menangani POST /v1/admin/platform/tenants.
func (h *AdminHandler) PlatformTenantCreate(c *fiber.Ctx) error {
	var req platformTenantCreateRequest
	if err := c.BodyParser(&req); err != nil {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "request body tidak valid")
	}
	if err := h.validate.Struct(req); err != nil {
		return domain.ErrorResponse(c, fiber.StatusUnprocessableEntity, "VALIDATION_ERROR", "validasi gagal")
	}
	status := validatePlatformTenantStatus(req.Status)
	tx, err := h.db.Begin(c.Context())
	if err != nil {
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal memulai transaksi")
	}
	defer tx.Rollback(c.Context())

	var tenantID string
	if err := tx.QueryRow(c.Context(), `
		INSERT INTO tenants (name, domain, plan, status)
		VALUES ($1, NULLIF($2, ''), $3, $4)
		RETURNING id::text
	`, req.Name, req.Domain, req.Plan, status).Scan(&tenantID); err != nil {
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal membuat tenant")
	}
	if _, err := tx.Exec(c.Context(), `
		INSERT INTO users (tenant_id, name, email, role, email_verified, status)
		VALUES ($1, $2, $3, 'tenant_admin', true, 'active')
		ON CONFLICT (tenant_id, email) DO UPDATE
		SET name = EXCLUDED.name, role = 'tenant_admin', status = 'active', updated_at = NOW()
	`, tenantID, req.OwnerName, req.OwnerEmail); err != nil {
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal membuat owner tenant")
	}
	if _, err := tx.Exec(c.Context(), `
		INSERT INTO tenant_modules (tenant_id, module_code, status, activated_at)
		VALUES ($1, 'billing_core', 'active', NOW())
		ON CONFLICT (tenant_id, module_code) DO UPDATE SET status = 'active', updated_at = NOW()
	`, tenantID); err != nil {
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal menyiapkan modul tenant")
	}
	if _, err := tx.Exec(c.Context(), `
		INSERT INTO platform_subscriptions (tenant_id, plan_code, status, amount, trial_ends_at, current_period_start, current_period_end)
		VALUES ($1, $2, $3, $4, CASE WHEN $3 = 'trial' THEN NOW() + INTERVAL '14 days' ELSE NULL END, NOW(), NOW() + INTERVAL '1 month')
		ON CONFLICT (tenant_id) DO UPDATE
		SET plan_code = EXCLUDED.plan_code, status = EXCLUDED.status, amount = EXCLUDED.amount, updated_at = NOW()
	`, tenantID, req.Plan, subscriptionStatusForTenantStatus(status), platformPlanAmount(req.Plan)); err != nil {
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal membuat subscription")
	}
	if err := tx.Commit(c.Context()); err != nil {
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal menyimpan tenant")
	}

	h.auditPlatformAction(c, tenantID, "tenant", tenantID, "tenant.created_by_super_admin", req.Reason, map[string]interface{}{
		"name": req.Name, "plan": req.Plan, "status": status, "owner_email": req.OwnerEmail,
	})
	return domain.SuccessResponse(c, fiber.StatusCreated, fiber.Map{"id": tenantID})
}

// PlatformTenantUpdate menangani PUT /v1/admin/platform/tenants/:id.
func (h *AdminHandler) PlatformTenantUpdate(c *fiber.Ctx) error {
	tenantID := c.Params("id")
	var req platformTenantUpdateRequest
	if err := c.BodyParser(&req); err != nil {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "request body tidak valid")
	}
	if err := h.validate.Struct(req); err != nil {
		return domain.ErrorResponse(c, fiber.StatusUnprocessableEntity, "VALIDATION_ERROR", "validasi gagal")
	}
	status := validatePlatformTenantStatus(req.Status)
	var oldName, oldDomain, oldPlan, oldStatus string
	_ = h.db.QueryRow(c.Context(), `SELECT name, COALESCE(domain, ''), plan, status FROM tenants WHERE id = $1`, tenantID).Scan(&oldName, &oldDomain, &oldPlan, &oldStatus)
	if _, err := h.db.Exec(c.Context(), `
		UPDATE tenants
		SET name = $2, domain = NULLIF($3, ''), plan = $4, status = $5, updated_at = NOW()
		WHERE id = $1
	`, tenantID, req.Name, req.Domain, req.Plan, status); err != nil {
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal memperbarui tenant")
	}
	if _, err := h.db.Exec(c.Context(), `
		INSERT INTO platform_subscriptions (tenant_id, plan_code, status, amount, current_period_start, current_period_end)
		VALUES ($1, $2, $3, $4, NOW(), NOW() + INTERVAL '1 month')
		ON CONFLICT (tenant_id) DO UPDATE
		SET plan_code = EXCLUDED.plan_code, status = EXCLUDED.status, amount = EXCLUDED.amount, updated_at = NOW()
	`, tenantID, req.Plan, subscriptionStatusForTenantStatus(status), platformPlanAmount(req.Plan)); err != nil {
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal memperbarui subscription")
	}
	h.auditPlatformAction(c, tenantID, "tenant", tenantID, "tenant.updated_by_super_admin", req.Reason, map[string]interface{}{
		"old": map[string]string{"name": oldName, "domain": oldDomain, "plan": oldPlan, "status": oldStatus},
		"new": map[string]string{"name": req.Name, "domain": req.Domain, "plan": req.Plan, "status": status},
	})
	return h.PlatformTenantDetail(c)
}

func (h *AdminHandler) updateTenantStatus(c *fiber.Ctx, status string, action string) error {
	tenantID := c.Params("id")
	var req platformTenantActionRequest
	if err := c.BodyParser(&req); err != nil {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "request body tidak valid")
	}
	if err := h.validate.Struct(req); err != nil {
		return domain.ErrorResponse(c, fiber.StatusUnprocessableEntity, "VALIDATION_ERROR", "alasan wajib diisi")
	}
	var oldStatus string
	_ = h.db.QueryRow(c.Context(), `SELECT status FROM tenants WHERE id = $1`, tenantID).Scan(&oldStatus)
	if _, err := h.db.Exec(c.Context(), `UPDATE tenants SET status = $2, updated_at = NOW() WHERE id = $1`, tenantID, status); err != nil {
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal mengubah status tenant")
	}
	if _, err := h.db.Exec(c.Context(), `
		UPDATE platform_subscriptions
		SET status = $2, cancelled_at = CASE WHEN $2 = 'cancelled' THEN NOW() ELSE cancelled_at END, updated_at = NOW()
		WHERE tenant_id = $1
	`, tenantID, subscriptionStatusForTenantStatus(status)); err != nil {
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal mengubah status subscription")
	}
	h.auditPlatformAction(c, tenantID, "tenant", tenantID, action, req.Reason, map[string]interface{}{"old_status": oldStatus, "new_status": status})
	return h.PlatformTenantDetail(c)
}

func (h *AdminHandler) PlatformTenantActivate(c *fiber.Ctx) error {
	return h.updateTenantStatus(c, "active", "tenant.activated_by_super_admin")
}

func (h *AdminHandler) PlatformTenantSuspend(c *fiber.Ctx) error {
	return h.updateTenantStatus(c, "suspended", "tenant.suspended_by_super_admin")
}

func (h *AdminHandler) PlatformTenantResume(c *fiber.Ctx) error {
	return h.updateTenantStatus(c, "active", "tenant.resumed_by_super_admin")
}

func (h *AdminHandler) PlatformTenantCancel(c *fiber.Ctx) error {
	return h.updateTenantStatus(c, "cancelled", "tenant.cancelled_by_super_admin")
}

// PlatformTenantResetOwner menangani POST /v1/admin/platform/tenants/:id/reset-owner.
func (h *AdminHandler) PlatformTenantResetOwner(c *fiber.Ctx) error {
	tenantID := c.Params("id")
	var req platformOwnerResetRequest
	if err := c.BodyParser(&req); err != nil {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "request body tidak valid")
	}
	if err := h.validate.Struct(req); err != nil {
		return domain.ErrorResponse(c, fiber.StatusUnprocessableEntity, "VALIDATION_ERROR", "validasi gagal")
	}
	var userID string
	if err := h.db.QueryRow(c.Context(), `
		INSERT INTO users (tenant_id, name, email, role, email_verified, status)
		VALUES ($1, $2, $3, 'tenant_admin', true, 'active')
		ON CONFLICT (tenant_id, email) DO UPDATE
		SET name = EXCLUDED.name, role = 'tenant_admin', status = 'active', updated_at = NOW()
		RETURNING id::text
	`, tenantID, req.OwnerName, req.OwnerEmail).Scan(&userID); err != nil {
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal reset owner tenant")
	}
	h.auditPlatformAction(c, tenantID, "user", userID, "tenant.owner_reset_by_super_admin", req.Reason, map[string]interface{}{
		"owner_name": req.OwnerName, "owner_email": req.OwnerEmail,
	})
	return h.PlatformTenantDetail(c)
}

// PlatformTenantModules menangani GET /v1/admin/platform/tenants/:id/modules.
func (h *AdminHandler) PlatformTenantModules(c *fiber.Ctx) error {
	tenantID := c.Params("id")
	modules, err := h.platformTenantModules(c, tenantID)
	if err != nil {
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal mengambil modul tenant")
	}
	return domain.SuccessResponse(c, fiber.StatusOK, fiber.Map{"data": modules})
}

// PlatformTenantModulesUpdate menangani PUT /v1/admin/platform/tenants/:id/modules.
func (h *AdminHandler) PlatformTenantModulesUpdate(c *fiber.Ctx) error {
	tenantID := c.Params("id")
	var req platformTenantModulesRequest
	if err := c.BodyParser(&req); err != nil {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "request body tidak valid")
	}
	if err := h.validate.Struct(req); err != nil {
		return domain.ErrorResponse(c, fiber.StatusUnprocessableEntity, "VALIDATION_ERROR", "alasan wajib diisi")
	}
	oldModules, _ := h.platformTenantModules(c, tenantID)
	newModules := uniqueModules(req.Modules)
	active := map[string]bool{}
	for _, module := range newModules {
		active[module] = true
	}
	for _, module := range []string{domain.ModuleBillingCore, domain.ModuleMikroTik, domain.ModuleFiberNetwork} {
		status := "inactive"
		var activatedAt interface{}
		if active[module] {
			status = "active"
			activatedAt = time.Now()
		}
		if module == domain.ModuleBillingCore {
			status = "active"
			activatedAt = time.Now()
		}
		if _, err := h.db.Exec(c.Context(), `
			INSERT INTO tenant_modules (tenant_id, module_code, status, activated_at)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (tenant_id, module_code) DO UPDATE
			SET status = EXCLUDED.status,
			    activated_at = CASE WHEN EXCLUDED.status = 'active' THEN COALESCE(tenant_modules.activated_at, NOW()) ELSE tenant_modules.activated_at END,
			    updated_at = NOW()
		`, tenantID, module, status, activatedAt); err != nil {
			return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal memperbarui modul tenant")
		}
	}
	h.auditPlatformAction(c, tenantID, "tenant", tenantID, "tenant.modules_updated_by_super_admin", req.Reason, map[string]interface{}{
		"old_modules": oldModules, "new_modules": newModules,
	})
	return h.PlatformTenantModules(c)
}

// PlatformAudit menangani GET /v1/admin/platform/audit.
func (h *AdminHandler) PlatformAudit(c *fiber.Ctx) error {
	audit, err := h.platformAuditFiltered(c, 100)
	if err != nil {
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal mengambil audit global")
	}
	return domain.SuccessResponse(c, fiber.StatusOK, fiber.Map{"data": audit})
}

func (h *AdminHandler) platformAudit(c *fiber.Ctx, limit int) ([]platformAuditRow, error) {
	return h.platformAuditFiltered(c, limit)
}

func (h *AdminHandler) platformAuditFiltered(c *fiber.Ctx, limit int) ([]platformAuditRow, error) {
	clauses := []string{}
	args := []interface{}{}
	add := func(clause string, value interface{}) {
		args = append(args, value)
		clauses = append(clauses, fmt.Sprintf(clause, len(args)))
	}
	if tenantID := c.Query("tenant_id"); tenantID != "" {
		add("a.tenant_id = $%d", tenantID)
	}
	if actor := c.Query("actor"); actor != "" {
		add("(a.actor_name ILIKE '%%' || $%d || '%%' OR a.actor_id::text = $%d)", actor)
	}
	if action := c.Query("action"); action != "" {
		add("a.action ILIKE '%%' || $%d || '%%'", action)
	}
	if entity := c.Query("entity"); entity != "" {
		add("(a.entity_type ILIKE '%%' || $%d || '%%' OR a.entity_id::text ILIKE '%%' || $%d || '%%')", entity)
	}
	if from := c.Query("from"); from != "" {
		add("a.created_at >= $%d::timestamptz", from)
	}
	if to := c.Query("to"); to != "" {
		add("a.created_at <= $%d::timestamptz", to)
	}
	where := ""
	if len(clauses) > 0 {
		where = "WHERE " + strings.Join(clauses, " AND ")
	}
	args = append(args, limit)
	limitArg := len(args)
	rows, err := h.db.Query(c.Context(), `
		WITH platform_events AS (
			SELECT
				a.id::text AS id,
				a.tenant_id::text AS tenant_id,
				t.name AS tenant_name,
				COALESCE(a.actor_id::text, '') AS actor_id,
				a.actor_name AS actor_name,
				a.action AS action,
				a.entity_type AS entity_type,
				a.entity_id::text AS entity_id,
				'success' AS status,
				a.created_at AS created_at
			FROM audit_logs a
			JOIN tenants t ON t.id = a.tenant_id
			UNION ALL
			SELECT
				pa.id::text AS id,
				'' AS tenant_id,
				'Platform' AS tenant_name,
				COALESCE(pa.actor_id::text, '') AS actor_id,
				pa.actor_name AS actor_name,
				pa.action AS action,
				pa.entity_type AS entity_type,
				pa.entity_id AS entity_id,
				'success' AS status,
				pa.created_at AS created_at
			FROM platform_audit_logs pa
		)
		SELECT
			a.id::text,
			a.tenant_id,
			a.tenant_name,
			a.actor_name,
			a.action,
			a.entity_type,
			a.entity_id,
			a.status,
			a.created_at
		FROM platform_events a
		`+where+`
		ORDER BY a.created_at DESC
		LIMIT $`+fmt.Sprint(limitArg), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanPlatformAuditRows(rows)
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

	return scanPlatformAuditRows(rows)
}

func scanPlatformAuditRows(rows pgx.Rows) ([]platformAuditRow, error) {
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

// PlatformAuditDetail menangani GET /v1/admin/platform/audit/:id.
func (h *AdminHandler) PlatformAuditDetail(c *fiber.Ctx) error {
	id := c.Params("id")
	var item platformAuditRow
	var changesRaw, metadataRaw []byte
	var changes, metadata map[string]interface{}
	err := h.db.QueryRow(c.Context(), `
		WITH platform_events AS (
			SELECT
				a.id::text AS id,
				a.tenant_id::text AS tenant_id,
				t.name AS tenant_name,
				a.actor_name AS actor_name,
				a.action AS action,
				a.entity_type AS entity_type,
				a.entity_id::text AS entity_id,
				'success' AS status,
				a.created_at AS created_at,
				COALESCE(a.changes, '{}'::jsonb) AS changes,
				COALESCE(a.metadata, '{}'::jsonb) AS metadata
			FROM audit_logs a
			JOIN tenants t ON t.id = a.tenant_id
			UNION ALL
			SELECT
				pa.id::text AS id,
				'' AS tenant_id,
				'Platform' AS tenant_name,
				pa.actor_name AS actor_name,
				pa.action AS action,
				pa.entity_type AS entity_type,
				pa.entity_id AS entity_id,
				'success' AS status,
				pa.created_at AS created_at,
				COALESCE(pa.changes, '{}'::jsonb) AS changes,
				COALESCE(pa.metadata, '{}'::jsonb) AS metadata
			FROM platform_audit_logs pa
		)
		SELECT id, tenant_id, tenant_name, actor_name, action, entity_type, entity_id, status, created_at, changes, metadata
		FROM platform_events
		WHERE id = $1
	`, id).Scan(&item.ID, &item.TenantID, &item.TenantName, &item.ActorName, &item.Action, &item.EntityType, &item.EntityID, &item.Status, &item.CreatedAt, &changesRaw, &metadataRaw)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrorResponse(c, fiber.StatusNotFound, "AUDIT_NOT_FOUND", "audit tidak ditemukan")
		}
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal mengambil audit")
	}
	_ = json.Unmarshal(changesRaw, &changes)
	_ = json.Unmarshal(metadataRaw, &metadata)
	return domain.SuccessResponse(c, fiber.StatusOK, fiber.Map{
		"audit":    item,
		"changes":  redactPlatformPayload(changes),
		"metadata": redactPlatformPayload(metadata),
	})
}

// PlatformAuditExport menangani GET /v1/admin/platform/audit/export.
func (h *AdminHandler) PlatformAuditExport(c *fiber.Ctx) error {
	audit, err := h.platformAuditFiltered(c, 1000)
	if err != nil {
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal export audit")
	}
	var builder strings.Builder
	writer := csv.NewWriter(&builder)
	_ = writer.Write([]string{"waktu", "tenant", "actor", "action", "entity_type", "entity_id", "status"})
	for _, item := range audit {
		_ = writer.Write([]string{
			item.CreatedAt.Format(time.RFC3339),
			item.TenantName,
			item.ActorName,
			item.Action,
			item.EntityType,
			item.EntityID,
			item.Status,
		})
	}
	writer.Flush()
	c.Set("Content-Type", "text/csv; charset=utf-8")
	c.Set("Content-Disposition", `attachment; filename="audit-global.csv"`)
	return c.SendString(builder.String())
}

func redactPlatformPayload(value map[string]interface{}) map[string]interface{} {
	if value == nil {
		return map[string]interface{}{}
	}
	redacted := map[string]interface{}{}
	for key, raw := range value {
		lower := strings.ToLower(key)
		if strings.Contains(lower, "password") || strings.Contains(lower, "token") || strings.Contains(lower, "secret") || strings.Contains(lower, "credential") {
			redacted[key] = "[redacted]"
			continue
		}
		redacted[key] = raw
	}
	return redacted
}

// PlatformHealth menangani GET /v1/admin/platform/health.
func (h *AdminHandler) PlatformHealth(c *fiber.Ctx) error {
	return domain.SuccessResponse(c, fiber.StatusOK, fiber.Map{"data": h.platformHealth(c)})
}

func (h *AdminHandler) platformHealth(c *fiber.Ctx) []platformHealthRow {
	start := time.Now()
	status := "online"
	lastError := ""
	if err := h.db.Ping(c.Context()); err != nil {
		status = "offline"
		lastError = err.Error()
	}
	latency := time.Since(start).Milliseconds()
	checkedAt := time.Now().Format(time.RFC3339)
	network := h.probeHTTPService("Network Service", getenvDefault("NETWORK_SERVICE_URL", "http://network-service:3002")+"/healthz")
	notification := h.probeHTTPService("Notification Service", getenvDefault("NOTIFICATION_SERVICE_URL", "http://notification:3003")+"/healthz")
	redis := h.probeTCPService("Redis", getenvDefault("REDIS_ADDR", "redis:6379"))
	var pendingJobs int64
	_ = h.db.QueryRow(c.Context(), `SELECT COUNT(*)::bigint FROM asynq_tasks WHERE state IN (1,2,3)`).Scan(&pendingJobs)

	return []platformHealthRow{
		{Service: "Billing API", Region: "Local", LatencyMs: latency, Uptime: "live", Status: "online", LastChecked: checkedAt},
		{Service: "PostgreSQL", Region: "Local", LatencyMs: latency, Uptime: "live", Status: status, LastChecked: checkedAt, LastError: lastError},
		redis,
		network,
		notification,
		{Service: "Queue Worker", Region: "Local", LatencyMs: 0, Uptime: fmt.Sprintf("%d pending", pendingJobs), Status: "online", LastChecked: checkedAt},
		{Service: "Payment Gateway", Region: "Tenant config", LatencyMs: 0, Uptime: "manual check", Status: "unknown", LastChecked: checkedAt},
		{Service: "Notification Error Rate", Region: "Tenant config", LatencyMs: 0, Uptime: "provider logs", Status: "unknown", LastChecked: checkedAt},
	}
}

func (h *AdminHandler) probeTCPService(name, address string) platformHealthRow {
	start := time.Now()
	status := "online"
	lastError := ""
	conn, err := net.DialTimeout("tcp", address, 2*time.Second)
	if err != nil {
		status = "offline"
		lastError = err.Error()
	} else {
		_ = conn.Close()
	}
	return platformHealthRow{
		Service:     name,
		Region:      "Local",
		LatencyMs:   time.Since(start).Milliseconds(),
		Uptime:      "live",
		Status:      status,
		LastChecked: time.Now().Format(time.RFC3339),
		LastError:   lastError,
	}
}

func (h *AdminHandler) probeHTTPService(name, url string) platformHealthRow {
	start := time.Now()
	client := http.Client{Timeout: 2 * time.Second}
	status := "online"
	lastError := ""
	resp, err := client.Get(url)
	if err != nil {
		status = "offline"
		lastError = err.Error()
	} else {
		defer resp.Body.Close()
		if resp.StatusCode >= 400 {
			status = "warning"
			lastError = fmt.Sprintf("HTTP %d", resp.StatusCode)
		}
	}
	return platformHealthRow{
		Service:     name,
		Region:      "Local",
		LatencyMs:   time.Since(start).Milliseconds(),
		Uptime:      "live",
		Status:      status,
		LastChecked: time.Now().Format(time.RFC3339),
		LastError:   lastError,
	}
}

func getenvDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return strings.TrimRight(value, "/")
	}
	return fallback
}

// PlatformSubscriptions menangani GET /v1/admin/platform/subscriptions.
func (h *AdminHandler) PlatformSubscriptions(c *fiber.Ctx) error {
	rows, err := h.db.Query(c.Context(), `
		SELECT
			t.id::text,
			t.name,
			COALESCE(ps.plan_code, t.plan),
			COALESCE(ps.amount, CASE WHEN t.plan IN ('growth', 'pro') THEN 799000 WHEN t.plan IN ('scale', 'enterprise') THEN 1499000 ELSE 299000 END)::bigint,
			COALESCE(ps.currency, 'IDR'),
			COALESCE(ps.status, CASE WHEN t.status = 'trial' THEN 'trial' WHEN t.status = 'suspended' THEN 'suspended' WHEN t.status = 'cancelled' THEN 'cancelled' ELSE 'active' END),
			ps.trial_ends_at,
			COALESCE(ps.current_period_end, t.created_at + INTERVAL '1 month'),
			COUNT(DISTINCT c.id)::bigint,
			COUNT(DISTINCT CASE WHEN i.status IN ('belum_bayar', 'terlambat', 'bayar_sebagian') THEN i.id END)::bigint
		FROM tenants t
		LEFT JOIN platform_subscriptions ps ON ps.tenant_id = t.id
		LEFT JOIN customers c ON c.tenant_id = t.id
		LEFT JOIN invoices i ON i.tenant_id = t.id
		GROUP BY t.id, t.name, t.plan, t.status, t.created_at, ps.plan_code, ps.amount, ps.currency, ps.status, ps.trial_ends_at, ps.current_period_end
		ORDER BY COALESCE(ps.current_period_end, t.created_at + INTERVAL '1 month') ASC
	`)
	if err != nil {
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal mengambil subscription")
	}
	defer rows.Close()

	items := make([]platformSubscriptionRow, 0)
	for rows.Next() {
		var item platformSubscriptionRow
		var trialEndsAt *time.Time
		if err := rows.Scan(&item.TenantID, &item.Tenant, &item.Plan, &item.Amount, &item.Currency, &item.Status, &trialEndsAt, &item.CurrentPeriodEnd, &item.CustomerCount, &item.OpenInvoiceCount); err != nil {
			return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal membaca subscription")
		}
		item.TrialEndsAt = trialEndsAt
		item.DueDate = item.CurrentPeriodEnd
		item.Modules, _ = h.platformTenantModules(c, item.TenantID)
		item.MonthlyRevenue = item.Amount
		items = append(items, item)
	}
	return domain.SuccessResponse(c, fiber.StatusOK, fiber.Map{"data": items})
}

// PlatformSubscriptionUpdate menangani PUT /v1/admin/platform/subscriptions/:tenant_id.
func (h *AdminHandler) PlatformSubscriptionUpdate(c *fiber.Ctx) error {
	tenantID := c.Params("tenant_id")
	var req struct {
		Plan   string `json:"plan" validate:"required"`
		Status string `json:"status" validate:"required,oneof=trial active overdue suspended cancelled"`
		Amount int64  `json:"amount"`
		Reason string `json:"reason" validate:"required"`
	}
	if err := c.BodyParser(&req); err != nil {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "request body tidak valid")
	}
	if err := h.validate.Struct(req); err != nil {
		return domain.ErrorResponse(c, fiber.StatusUnprocessableEntity, "VALIDATION_ERROR", "validasi gagal")
	}
	if req.Amount <= 0 {
		req.Amount = platformPlanAmount(req.Plan)
	}
	if _, err := h.db.Exec(c.Context(), `
		INSERT INTO platform_subscriptions (tenant_id, plan_code, status, amount, current_period_start, current_period_end)
		VALUES ($1, $2, $3, $4, NOW(), NOW() + INTERVAL '1 month')
		ON CONFLICT (tenant_id) DO UPDATE
		SET plan_code = EXCLUDED.plan_code, status = EXCLUDED.status, amount = EXCLUDED.amount, updated_at = NOW()
	`, tenantID, req.Plan, req.Status, req.Amount); err != nil {
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal memperbarui subscription")
	}
	_, _ = h.db.Exec(c.Context(), `UPDATE tenants SET plan = $2, status = CASE WHEN $3 IN ('suspended','cancelled','trial') THEN $3 ELSE 'active' END, updated_at = NOW() WHERE id = $1`, tenantID, req.Plan, req.Status)
	h.auditPlatformAction(c, tenantID, "tenant", tenantID, "subscription.updated_by_super_admin", req.Reason, map[string]interface{}{
		"plan": req.Plan, "status": req.Status, "amount": req.Amount,
	})
	subscription, err := h.platformSubscription(c, tenantID)
	if err != nil {
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal mengambil subscription")
	}
	return domain.SuccessResponse(c, fiber.StatusOK, subscription)
}

// PlatformSupport menangani GET /v1/admin/platform/support.
func (h *AdminHandler) PlatformSupport(c *fiber.Ctx) error {
	tickets, err := h.platformSupportTickets(c, c.Query("tenant_id"), c.Query("status"), c.Query("priority"), c.Query("assignee_id"))
	if err != nil {
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal mengambil tiket support")
	}
	return domain.SuccessResponse(c, fiber.StatusOK, fiber.Map{"data": tickets})
}

func (h *AdminHandler) platformSupportTickets(c *fiber.Ctx, tenantID, status, priority, assigneeID string) ([]platformSupportTicketRow, error) {
	clauses := []string{}
	args := []interface{}{}
	add := func(clause string, value interface{}) {
		args = append(args, value)
		clauses = append(clauses, fmt.Sprintf(clause, len(args)))
	}
	if tenantID != "" {
		add("st.tenant_id = $%d", tenantID)
	}
	if status != "" {
		add("st.status = $%d", status)
	}
	if priority != "" {
		add("st.priority = $%d", priority)
	}
	if assigneeID != "" {
		add("st.assignee_id = $%d", assigneeID)
	}
	where := ""
	if len(clauses) > 0 {
		where = "WHERE " + strings.Join(clauses, " AND ")
	}
	rows, err := h.db.Query(c.Context(), `
		SELECT
			st.id::text,
			COALESCE(st.tenant_id::text, ''),
			COALESCE(t.name, ''),
			st.subject,
			COALESCE(st.description, ''),
			st.priority,
			st.status,
			COALESCE(st.assignee_id::text, ''),
			COALESCE(st.created_by::text, ''),
			COUNT(cmt.id)::bigint,
			st.created_at,
			st.updated_at
		FROM support_tickets st
		LEFT JOIN tenants t ON t.id = st.tenant_id
		LEFT JOIN support_ticket_comments cmt ON cmt.ticket_id = st.id
		`+where+`
		GROUP BY st.id, t.name
		ORDER BY
			CASE st.priority WHEN 'urgent' THEN 1 WHEN 'high' THEN 2 WHEN 'normal' THEN 3 ELSE 4 END,
			st.updated_at DESC
		LIMIT 100
	`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []platformSupportTicketRow{}
	for rows.Next() {
		var item platformSupportTicketRow
		if err := rows.Scan(&item.ID, &item.TenantID, &item.TenantName, &item.Subject, &item.Description, &item.Priority, &item.Status, &item.AssigneeID, &item.CreatedBy, &item.CommentCount, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// PlatformSupportCreate menangani POST /v1/admin/platform/support.
func (h *AdminHandler) PlatformSupportCreate(c *fiber.Ctx) error {
	var req platformSupportCreateRequest
	if err := c.BodyParser(&req); err != nil {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "request body tidak valid")
	}
	if err := h.validate.Struct(req); err != nil {
		return domain.ErrorResponse(c, fiber.StatusUnprocessableEntity, "VALIDATION_ERROR", "validasi gagal")
	}
	var ticketID string
	if err := h.db.QueryRow(c.Context(), `
		INSERT INTO support_tickets (tenant_id, subject, description, priority, status, created_by)
		VALUES (NULLIF($1, '')::uuid, $2, $3, $4, 'open', $5)
		RETURNING id::text
	`, req.TenantID, req.Subject, req.Description, req.Priority, h.actorID(c)).Scan(&ticketID); err != nil {
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal membuat tiket support")
	}
	if req.TenantID != "" {
		h.auditPlatformAction(c, req.TenantID, "support_ticket", ticketID, "support.ticket_created_by_super_admin", req.Subject, map[string]interface{}{"priority": req.Priority})
	}
	return domain.SuccessResponse(c, fiber.StatusCreated, fiber.Map{"id": ticketID})
}

// PlatformSupportDetail menangani GET /v1/admin/platform/support/:id.
func (h *AdminHandler) PlatformSupportDetail(c *fiber.Ctx) error {
	id := c.Params("id")
	rows, err := h.db.Query(c.Context(), `
		SELECT
			st.id::text, COALESCE(st.tenant_id::text, ''), COALESCE(t.name, ''),
			st.subject, COALESCE(st.description, ''), st.priority, st.status,
			COALESCE(st.assignee_id::text, ''), COALESCE(st.created_by::text, ''),
			(SELECT COUNT(*)::bigint FROM support_ticket_comments WHERE ticket_id = st.id),
			st.created_at, st.updated_at
		FROM support_tickets st
		LEFT JOIN tenants t ON t.id = st.tenant_id
		WHERE st.id = $1
	`, id)
	if err != nil {
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal mengambil tiket")
	}
	defer rows.Close()
	tickets := []platformSupportTicketRow{}
	for rows.Next() {
		var item platformSupportTicketRow
		if err := rows.Scan(&item.ID, &item.TenantID, &item.TenantName, &item.Subject, &item.Description, &item.Priority, &item.Status, &item.AssigneeID, &item.CreatedBy, &item.CommentCount, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal membaca tiket")
		}
		tickets = append(tickets, item)
	}
	if len(tickets) == 0 {
		return domain.ErrorResponse(c, fiber.StatusNotFound, "SUPPORT_TICKET_NOT_FOUND", "tiket tidak ditemukan")
	}
	comments, _ := h.platformSupportComments(c, id)
	return domain.SuccessResponse(c, fiber.StatusOK, fiber.Map{"ticket": tickets[0], "comments": comments})
}

func (h *AdminHandler) platformSupportComments(c *fiber.Ctx, ticketID string) ([]platformSupportCommentRow, error) {
	rows, err := h.db.Query(c.Context(), `
		SELECT id::text, ticket_id::text, COALESCE(author_id::text, ''), author_role, body, is_internal, created_at
		FROM support_ticket_comments
		WHERE ticket_id = $1
		ORDER BY created_at ASC
	`, ticketID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []platformSupportCommentRow{}
	for rows.Next() {
		var item platformSupportCommentRow
		if err := rows.Scan(&item.ID, &item.TicketID, &item.AuthorID, &item.AuthorRole, &item.Body, &item.IsInternal, &item.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// PlatformSupportComment menangani POST /v1/admin/platform/support/:id/comments.
func (h *AdminHandler) PlatformSupportComment(c *fiber.Ctx) error {
	ticketID := c.Params("id")
	var req platformSupportCommentRequest
	if err := c.BodyParser(&req); err != nil {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "request body tidak valid")
	}
	if err := h.validate.Struct(req); err != nil {
		return domain.ErrorResponse(c, fiber.StatusUnprocessableEntity, "VALIDATION_ERROR", "komentar wajib diisi")
	}
	if _, err := h.db.Exec(c.Context(), `
		INSERT INTO support_ticket_comments (ticket_id, author_id, author_role, body, is_internal)
		VALUES ($1, $2, 'super_admin', $3, $4)
	`, ticketID, h.actorID(c), req.Body, req.IsInternal); err != nil {
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal menambah komentar")
	}
	_, _ = h.db.Exec(c.Context(), `UPDATE support_tickets SET updated_at = NOW() WHERE id = $1`, ticketID)
	return h.PlatformSupportDetail(c)
}

// PlatformSupportStatus menangani PUT /v1/admin/platform/support/:id/status.
func (h *AdminHandler) PlatformSupportStatus(c *fiber.Ctx) error {
	ticketID := c.Params("id")
	var req platformSupportStatusRequest
	if err := c.BodyParser(&req); err != nil {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "request body tidak valid")
	}
	if err := h.validate.Struct(req); err != nil {
		return domain.ErrorResponse(c, fiber.StatusUnprocessableEntity, "VALIDATION_ERROR", "status tidak valid")
	}
	var tenantID, oldStatus string
	_ = h.db.QueryRow(c.Context(), `SELECT COALESCE(tenant_id::text, ''), status FROM support_tickets WHERE id = $1`, ticketID).Scan(&tenantID, &oldStatus)
	if _, err := h.db.Exec(c.Context(), `UPDATE support_tickets SET status = $2, updated_at = NOW() WHERE id = $1`, ticketID, req.Status); err != nil {
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal mengubah status tiket")
	}
	if tenantID != "" {
		h.auditPlatformAction(c, tenantID, "support_ticket", ticketID, "support.ticket_status_changed", req.Reason, map[string]interface{}{"old_status": oldStatus, "new_status": req.Status})
	}
	return h.PlatformSupportDetail(c)
}

// PlatformUpgradeRequests menangani GET /v1/admin/platform/upgrade-requests.
func (h *AdminHandler) PlatformUpgradeRequests(c *fiber.Ctx) error {
	rows, err := h.db.Query(c.Context(), `
		SELECT
			ur.id::text, ur.tenant_id::text, t.name, COALESCE(ur.requested_plan, ''),
			ur.requested_modules, COALESCE(ur.message, ''), ur.status,
			COALESCE(ur.processed_by::text, ''), COALESCE(ur.processed_reason, ''),
			ur.processed_at, ur.created_at, ur.updated_at
		FROM tenant_upgrade_requests ur
		JOIN tenants t ON t.id = ur.tenant_id
		ORDER BY CASE ur.status WHEN 'pending' THEN 1 ELSE 2 END, ur.created_at DESC
		LIMIT 100
	`)
	if err != nil {
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal mengambil upgrade request")
	}
	defer rows.Close()
	items := []platformUpgradeRequestRow{}
	for rows.Next() {
		var item platformUpgradeRequestRow
		var modulesRaw []byte
		if err := rows.Scan(&item.ID, &item.TenantID, &item.TenantName, &item.RequestedPlan, &modulesRaw, &item.Message, &item.Status, &item.ProcessedBy, &item.ProcessedReason, &item.ProcessedAt, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal membaca upgrade request")
		}
		item.RequestedModules = parseStringArray(modulesRaw)
		items = append(items, item)
	}
	return domain.SuccessResponse(c, fiber.StatusOK, fiber.Map{"data": items})
}

func (h *AdminHandler) processUpgradeRequest(c *fiber.Ctx, status string, action string) error {
	id := c.Params("id")
	var req platformUpgradeDecisionRequest
	if err := c.BodyParser(&req); err != nil {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "request body tidak valid")
	}
	if err := h.validate.Struct(req); err != nil {
		return domain.ErrorResponse(c, fiber.StatusUnprocessableEntity, "VALIDATION_ERROR", "alasan wajib diisi")
	}
	var tenantID, requestedPlan string
	var modulesRaw []byte
	if err := h.db.QueryRow(c.Context(), `SELECT tenant_id::text, COALESCE(requested_plan, ''), requested_modules FROM tenant_upgrade_requests WHERE id = $1`, id).Scan(&tenantID, &requestedPlan, &modulesRaw); err != nil {
		return domain.ErrorResponse(c, fiber.StatusNotFound, "UPGRADE_REQUEST_NOT_FOUND", "upgrade request tidak ditemukan")
	}
	modules := parseStringArray(modulesRaw)
	tx, err := h.db.Begin(c.Context())
	if err != nil {
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal memulai transaksi")
	}
	defer tx.Rollback(c.Context())
	if _, err := tx.Exec(c.Context(), `
		UPDATE tenant_upgrade_requests
		SET status = $2, processed_by = $3, processed_reason = $4, processed_at = NOW(), updated_at = NOW()
		WHERE id = $1
	`, id, status, h.actorID(c), req.Reason); err != nil {
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal memproses upgrade request")
	}
	if status == "approved" {
		if requestedPlan != "" {
			if _, err := tx.Exec(c.Context(), `UPDATE tenants SET plan = $2, status = 'active', updated_at = NOW() WHERE id = $1`, tenantID, requestedPlan); err != nil {
				return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal update tenant")
			}
			if _, err := tx.Exec(c.Context(), `
				INSERT INTO platform_subscriptions (tenant_id, plan_code, status, amount, current_period_start, current_period_end)
				VALUES ($1, $2, 'active', $3, NOW(), NOW() + INTERVAL '1 month')
				ON CONFLICT (tenant_id) DO UPDATE
				SET plan_code = EXCLUDED.plan_code, status = 'active', amount = EXCLUDED.amount, updated_at = NOW()
			`, tenantID, requestedPlan, platformPlanAmount(requestedPlan)); err != nil {
				return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal update subscription")
			}
		}
		for _, module := range uniqueModules(modules) {
			if _, err := tx.Exec(c.Context(), `
				INSERT INTO tenant_modules (tenant_id, module_code, status, activated_at)
				VALUES ($1, $2, 'active', NOW())
				ON CONFLICT (tenant_id, module_code) DO UPDATE
				SET status = 'active', activated_at = COALESCE(tenant_modules.activated_at, NOW()), updated_at = NOW()
			`, tenantID, module); err != nil {
				return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal update modul")
			}
		}
	}
	if err := tx.Commit(c.Context()); err != nil {
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal menyimpan upgrade request")
	}
	h.auditPlatformAction(c, tenantID, "tenant_upgrade_request", id, action, req.Reason, map[string]interface{}{
		"status": status, "requested_plan": requestedPlan, "requested_modules": modules,
	})
	return h.PlatformUpgradeRequests(c)
}

func (h *AdminHandler) PlatformUpgradeApprove(c *fiber.Ctx) error {
	return h.processUpgradeRequest(c, "approved", "upgrade_request.approved_by_super_admin")
}

func (h *AdminHandler) PlatformUpgradeReject(c *fiber.Ctx) error {
	return h.processUpgradeRequest(c, "rejected", "upgrade_request.rejected_by_super_admin")
}

func (h *AdminHandler) PlatformUpgradeCancel(c *fiber.Ctx) error {
	return h.processUpgradeRequest(c, "cancelled", "upgrade_request.cancelled_by_super_admin")
}

// TenantUpgradeRequests menangani GET /v1/tenant/upgrade-requests.
func (h *AdminHandler) TenantUpgradeRequests(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	if tenantID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant tidak teridentifikasi")
	}
	rows, err := h.db.Query(c.Context(), `
		SELECT
			ur.id::text, ur.tenant_id::text, t.name, COALESCE(ur.requested_plan, ''),
			ur.requested_modules, COALESCE(ur.message, ''), ur.status,
			COALESCE(ur.processed_by::text, ''), COALESCE(ur.processed_reason, ''),
			ur.processed_at, ur.created_at, ur.updated_at
		FROM tenant_upgrade_requests ur
		JOIN tenants t ON t.id = ur.tenant_id
		WHERE ur.tenant_id = $1
		ORDER BY ur.created_at DESC
		LIMIT 25
	`, tenantID)
	if err != nil {
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal mengambil upgrade request")
	}
	defer rows.Close()
	items := []platformUpgradeRequestRow{}
	for rows.Next() {
		var item platformUpgradeRequestRow
		var modulesRaw []byte
		if err := rows.Scan(&item.ID, &item.TenantID, &item.TenantName, &item.RequestedPlan, &modulesRaw, &item.Message, &item.Status, &item.ProcessedBy, &item.ProcessedReason, &item.ProcessedAt, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal membaca upgrade request")
		}
		item.RequestedModules = parseStringArray(modulesRaw)
		items = append(items, item)
	}
	return domain.SuccessResponse(c, fiber.StatusOK, fiber.Map{"data": items})
}

// TenantUpgradeRequestCreate menangani POST /v1/tenant/upgrade-requests.
func (h *AdminHandler) TenantUpgradeRequestCreate(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	if tenantID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant tidak teridentifikasi")
	}
	var req tenantUpgradeRequestCreate
	if err := c.BodyParser(&req); err != nil {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "request body tidak valid")
	}
	modules := uniqueModules(req.RequestedModules)
	var id string
	if err := h.db.QueryRow(c.Context(), `
		INSERT INTO tenant_upgrade_requests (tenant_id, requested_plan, requested_modules, message, status)
		VALUES ($1, NULLIF($2, ''), $3::jsonb, $4, 'pending')
		RETURNING id::text
	`, tenantID, req.RequestedPlan, string(jsonStringArray(modules)), req.Message).Scan(&id); err != nil {
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal membuat upgrade request")
	}
	h.auditPlatformAction(c, tenantID, "tenant_upgrade_request", id, "upgrade_request.created_by_tenant", req.Message, map[string]interface{}{
		"requested_plan": req.RequestedPlan, "requested_modules": modules,
	})
	return domain.SuccessResponse(c, fiber.StatusCreated, fiber.Map{"id": id})
}

// PlatformSettings menangani GET /v1/admin/platform/settings.
func (h *AdminHandler) PlatformSettings(c *fiber.Ctx) error {
	rows, err := h.db.Query(c.Context(), `SELECT key, value_json, updated_at FROM platform_settings ORDER BY key`)
	if err != nil {
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal mengambil settings platform")
	}
	defer rows.Close()
	settings := fiber.Map{}
	updated := fiber.Map{}
	for rows.Next() {
		var key string
		var raw []byte
		var updatedAt time.Time
		if err := rows.Scan(&key, &raw, &updatedAt); err != nil {
			return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal membaca settings platform")
		}
		var value interface{}
		_ = json.Unmarshal(raw, &value)
		settings[key] = value
		updated[key] = updatedAt
	}
	return domain.SuccessResponse(c, fiber.StatusOK, fiber.Map{"settings": settings, "updated_at": updated})
}

// PlatformSettingsUpdate menangani PUT /v1/admin/platform/settings.
func (h *AdminHandler) PlatformSettingsUpdate(c *fiber.Ctx) error {
	var payload map[string]interface{}
	if err := json.Unmarshal(c.Body(), &payload); err != nil {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "request body tidak valid")
	}
	reason, _ := payload["reason"].(string)
	if strings.TrimSpace(reason) == "" {
		return domain.ErrorResponse(c, fiber.StatusUnprocessableEntity, "VALIDATION_ERROR", "alasan wajib diisi")
	}
	delete(payload, "reason")
	for key, value := range payload {
		encoded, _ := json.Marshal(value)
		if _, err := h.db.Exec(c.Context(), `
			INSERT INTO platform_settings (key, value_json, updated_by, updated_at)
			VALUES ($1, $2::jsonb, $3, NOW())
			ON CONFLICT (key) DO UPDATE
			SET value_json = EXCLUDED.value_json, updated_by = EXCLUDED.updated_by, updated_at = NOW()
		`, key, string(encoded), h.actorID(c)); err != nil {
			return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal menyimpan settings platform")
		}
		h.auditOwnerAction(c, "platform_settings", key, "platform.settings_updated", reason, map[string]interface{}{
			"key":   key,
			"value": value,
		})
	}
	h.logger.Info().Str("actor_id", h.actorID(c)).Msg("platform settings updated")
	return h.PlatformSettings(c)
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
