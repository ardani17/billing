// provisioning_handler.go menangani HTTP permintaan untuk provisioning ONT.
// Termasuk: provision, list, get, decommission, reboot, confirm migration, unregistered ONTs.
// Bulk dan settings handler ada di file terpisah.
package handler

import (
	"strconv"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"

	"github.com/ispboss/ispboss/pkg/tenant"
	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// ProvisioningHandler menangani HTTP permintaan untuk operasi provisioning ONT.
type ProvisioningHandler struct {
	manager  domain.ProvisioningManager
	validate *validator.Validate
}

// NewProvisioningHandler membuat instance baru ProvisioningHandler.
func NewProvisioningHandler(manager domain.ProvisioningManager) *ProvisioningHandler {
	return &ProvisioningHandler{
		manager:  manager,
		validate: validator.New(),
	}
}

// ProvisionONT menangani POST /provisioning/ont.
// Parsing body, validasi, extract tenant_id, lalu provision ONT baru.
func (h *ProvisioningHandler) ProvisionONT(c *fiber.Ctx) error {
	tenantID := tenant.FromContext(c.UserContext())
	if tenantID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant tidak teridentifikasi")
	}

	var req domain.ProvisionONTRequest
	if err := c.BodyParser(&req); err != nil {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "request body tidak valid")
	}
	if err := h.validate.Struct(req); err != nil {
		return h.validationError(c, err)
	}

	resp, err := h.manager.ProvisionONT(c.UserContext(), tenantID, req)
	if err != nil {
		return h.mapError(c, err)
	}
	return domain.SuccessResponse(c, fiber.StatusCreated, resp)
}

// PreviewProvisionONT menangani POST /provisioning/ont/preview.
// Endpoint ini hanya membangun dry-run command dan tidak mengeksekusi write ke OLT.
func (h *ProvisioningHandler) PreviewProvisionONT(c *fiber.Ctx) error {
	tenantID := tenant.FromContext(c.UserContext())
	if tenantID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant tidak teridentifikasi")
	}

	var req domain.ProvisionONTRequest
	if err := c.BodyParser(&req); err != nil {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "request body tidak valid")
	}
	if err := h.validate.Struct(req); err != nil {
		return h.validationError(c, err)
	}

	resp, err := h.manager.PreviewProvisionONT(c.UserContext(), tenantID, req)
	if err != nil {
		return h.mapError(c, err)
	}
	return domain.SuccessResponse(c, fiber.StatusOK, resp)
}

// ListONTs menangani GET /provisioning/onts.
func (h *ProvisioningHandler) ListONTs(c *fiber.Ctx) error {
	tenantID := tenant.FromContext(c.UserContext())
	if tenantID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant tidak teridentifikasi")
	}

	page, _ := strconv.Atoi(c.Query("page", "1"))
	pageSize, _ := strconv.Atoi(c.Query("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}

	params := domain.ONTListParams{
		TenantID:          tenantID,
		Page:              page,
		PageSize:          pageSize,
		OLTID:             c.Query("olt_id"),
		Status:            c.Query("status"),
		ProvisioningState: c.Query("provisioning_state"),
		CustomerID:        c.Query("customer_id"),
		Search:            c.Query("search"),
	}

	result, err := h.manager.ListONTs(c.UserContext(), params)
	if err != nil {
		return h.mapError(c, err)
	}
	return domain.PaginatedResponse(c, fiber.StatusOK, result.Data, result.Total, result.Page, result.PageSize, result.TotalPages)
}

// GetONT menangani GET /provisioning/onts/:id.
// Mengambil detail ONT termasuk audit logs.
func (h *ProvisioningHandler) GetONT(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "ONT ID wajib diisi")
	}

	resp, err := h.manager.GetONTByID(c.UserContext(), id)
	if err != nil {
		return h.mapError(c, err)
	}
	return domain.SuccessResponse(c, fiber.StatusOK, resp)
}

// DecommissionONT menangani POST /provisioning/ont/:id/decommission.
// Menghapus ONT dari OLT dan perbarui status di DB.
func (h *ProvisioningHandler) DecommissionONT(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "ONT ID wajib diisi")
	}

	performedBy, _ := c.Locals("username").(string)
	if performedBy == "" {
		performedBy = "system"
	}

	if err := h.manager.DecommissionONT(c.UserContext(), id, performedBy); err != nil {
		return h.mapError(c, err)
	}
	return domain.SuccessResponse(c, fiber.StatusOK, fiber.Map{"message": "ont berhasil di-decommission"})
}

// RebootONT menangani POST /provisioning/ont/:id/reboot.
// Mengirim perintah reboot ke ONT via OLT CLI.
func (h *ProvisioningHandler) RebootONT(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "ONT ID wajib diisi")
	}

	performedBy, _ := c.Locals("username").(string)
	if performedBy == "" {
		performedBy = "system"
	}

	result, err := h.manager.RebootONT(c.UserContext(), id, performedBy)
	if err != nil {
		return h.mapError(c, err)
	}
	return domain.SuccessResponse(c, fiber.StatusOK, result)
}

// ConfirmMigration menangani POST /provisioning/ont/:id/confirm-migration.
// Mengkonfirmasi port migration dan perbarui posisi ONT di DB.
func (h *ProvisioningHandler) ConfirmMigration(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "ONT ID wajib diisi")
	}

	if err := h.manager.ConfirmMigration(c.UserContext(), id); err != nil {
		return h.mapError(c, err)
	}
	return domain.SuccessResponse(c, fiber.StatusOK, fiber.Map{"message": "migrasi port dikonfirmasi"})
}

// GetUnregisteredONTs menangani GET /devices/:id/unregistered-onts.
// Mengambil daftar ONT unregistered untuk satu OLT.
func (h *ProvisioningHandler) GetUnregisteredONTs(c *fiber.Ctx) error {
	oltID := c.Params("id")
	if oltID == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "OLT ID wajib diisi")
	}

	result, err := h.manager.GetUnregisteredONTs(c.UserContext(), oltID)
	if err != nil {
		return h.mapError(c, err)
	}
	return domain.SuccessResponse(c, fiber.StatusOK, result)
}

// GetAuditLogs menangani GET /provisioning/audit-logs.
func (h *ProvisioningHandler) GetAuditLogs(c *fiber.Ctx) error {
	tenantID := tenant.FromContext(c.UserContext())
	if tenantID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant tidak teridentifikasi")
	}

	page, _ := strconv.Atoi(c.Query("page", "1"))
	pageSize, _ := strconv.Atoi(c.Query("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}

	params := domain.AuditLogListParams{
		TenantID: tenantID,
		Page:     page,
		PageSize: pageSize,
		OLTID:    c.Query("olt_id"),
		ONTID:    c.Query("ont_id"),
		Action:   c.Query("action"),
	}

	// Parsing date_from dan date_to (RFC3339)
	if dateFrom := c.Query("date_from"); dateFrom != "" {
		t, err := time.Parse(time.RFC3339, dateFrom)
		if err != nil {
			return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "format date_from tidak valid, gunakan RFC3339")
		}
		params.DateFrom = &t
	}
	if dateTo := c.Query("date_to"); dateTo != "" {
		t, err := time.Parse(time.RFC3339, dateTo)
		if err != nil {
			return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "format date_to tidak valid, gunakan RFC3339")
		}
		params.DateTo = &t
	}

	result, err := h.manager.GetAuditLogs(c.UserContext(), params)
	if err != nil {
		return h.mapError(c, err)
	}
	return domain.PaginatedResponse(c, fiber.StatusOK, result.Data, result.Total, result.Page, result.PageSize, result.TotalPages)
}
