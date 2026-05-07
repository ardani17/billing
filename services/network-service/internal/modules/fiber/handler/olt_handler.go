// olt_handler.go menangani HTTP permintaan untuk manajemen OLT device.
// Termasuk: CRUD, test SNMP/CLI, dan status summary.
// Endpoint pemantauan (PON ports, ONT, traffic, alarm, SFP, capacity) ada di olt_handler_monitoring.go.
package handler

import (
	"strconv"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"

	"github.com/ispboss/ispboss/pkg/tenant"
	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// OLTHandler menangani HTTP permintaan untuk operasi OLT device.
type OLTHandler struct {
	oltManager   domain.OLTManager
	alarmManager domain.AlarmManager
	validate     *validator.Validate
}

// NewOLTHandler membuat instance baru OLTHandler.
func NewOLTHandler(oltManager domain.OLTManager, alarmManager domain.AlarmManager) *OLTHandler {
	return &OLTHandler{
		oltManager:   oltManager,
		alarmManager: alarmManager,
		validate:     validator.New(),
	}
}

// CreateOLT menangani POST /devices.
// Parsing body, validasi, extract tenant_id, lalu buat OLT baru.
func (h *OLTHandler) CreateOLT(c *fiber.Ctx) error {
	tenantID := tenant.FromContext(c.UserContext())
	if tenantID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant tidak teridentifikasi")
	}

	var req domain.CreateOLTRequest
	if err := c.BodyParser(&req); err != nil {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "request body tidak valid")
	}

	if err := h.validate.Struct(req); err != nil {
		return h.validationError(c, err)
	}

	resp, err := h.oltManager.Create(c.UserContext(), tenantID, req)
	if err != nil {
		return h.mapError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusCreated, resp)
}

// ListOLTs menangani GET /devices.
func (h *OLTHandler) ListOLTs(c *fiber.Ctx) error {
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

	params := domain.OLTListParams{
		TenantID: tenantID,
		Page:     page,
		PageSize: pageSize,
		Status:   c.Query("status"),
		Brand:    c.Query("brand"),
		Search:   c.Query("search"),
	}

	result, err := h.oltManager.List(c.UserContext(), params)
	if err != nil {
		return h.mapError(c, err)
	}

	return domain.PaginatedResponse(c, fiber.StatusOK, result.Data, result.Total, result.Page, result.PageSize, result.TotalPages)
}

// GetOLT menangani GET /devices/:id.
// Mengambil detail OLT berdasarkan ID.
func (h *OLTHandler) GetOLT(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "OLT ID wajib diisi")
	}

	resp, err := h.oltManager.GetByID(c.UserContext(), id)
	if err != nil {
		return h.mapError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, resp)
}

// UpdateOLT menangani PUT /devices/:id.
// Parsing body, validasi, lalu perbarui data OLT.
func (h *OLTHandler) UpdateOLT(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "OLT ID wajib diisi")
	}

	var req domain.UpdateOLTRequest
	if err := c.BodyParser(&req); err != nil {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "request body tidak valid")
	}

	if err := h.validate.Struct(req); err != nil {
		return h.validationError(c, err)
	}

	resp, err := h.oltManager.Update(c.UserContext(), id, req)
	if err != nil {
		return h.mapError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, resp)
}

// DeleteOLT menangani DELETE /devices/:id.
// Soft-hapus OLT dan stop health cek pemantauan.
func (h *OLTHandler) DeleteOLT(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "OLT ID wajib diisi")
	}

	if err := h.oltManager.Delete(c.UserContext(), id); err != nil {
		return h.mapError(c, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// TestSNMP menangani POST /devices/:id/test-snmp.
// Menguji koneksi SNMP dan mengembalikan system info.
func (h *OLTHandler) TestSNMP(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "OLT ID wajib diisi")
	}

	result, err := h.oltManager.TestSNMP(c.UserContext(), id)
	if err != nil {
		return h.mapError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, result)
}

// TestCLI menangani POST /devices/:id/test-cli.
// Menguji koneksi CLI (SSH/Telnet) dan mengembalikan hasil.
func (h *OLTHandler) TestCLI(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "OLT ID wajib diisi")
	}

	result, err := h.oltManager.TestCLI(c.UserContext(), id)
	if err != nil {
		return h.mapError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, result)
}

// GetSummary menangani GET /summary.
// Mengembalikan ringkasan status semua OLT tenant.
func (h *OLTHandler) GetSummary(c *fiber.Ctx) error {
	tenantID := tenant.FromContext(c.UserContext())
	if tenantID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant tidak teridentifikasi")
	}

	summary, err := h.oltManager.GetStatusSummary(c.UserContext())
	if err != nil {
		return h.mapError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, summary)
}
