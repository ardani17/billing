// provisioning_handler_bulk.go menangani HTTP request untuk bulk provisioning ONT.
// Termasuk: upload CSV, execute bulk, dan download template.
package handler

import (
	"io"

	"github.com/gofiber/fiber/v2"

	"github.com/ispboss/ispboss/pkg/tenant"
	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// BulkUpload menangani POST /provisioning/bulk.
// Menerima file CSV via multipart form dan OLT ID, return preview validasi.
func (h *ProvisioningHandler) BulkUpload(c *fiber.Ctx) error {
	tenantID := tenant.FromContext(c.UserContext())
	if tenantID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant tidak teridentifikasi")
	}

	oltID := c.FormValue("olt_id")
	if oltID == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "olt_id wajib diisi")
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "file CSV wajib diupload")
	}

	file, err := fileHeader.Open()
	if err != nil {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "gagal membaca file CSV")
	}
	defer file.Close()

	csvData, err := io.ReadAll(file)
	if err != nil {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "gagal membaca isi file CSV")
	}

	preview, err := h.manager.ValidateBulk(c.UserContext(), tenantID, oltID, csvData)
	if err != nil {
		return h.mapError(c, err)
	}
	return domain.SuccessResponse(c, fiber.StatusOK, preview)
}

// BulkExecute menangani POST /provisioning/bulk/execute.
// Mengeksekusi bulk provisioning berdasarkan bulk_id dari preview sebelumnya.
func (h *ProvisioningHandler) BulkExecute(c *fiber.Ctx) error {
	var req struct {
		BulkID string `json:"bulk_id" validate:"required"`
	}
	if err := c.BodyParser(&req); err != nil {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "request body tidak valid")
	}
	if req.BulkID == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "bulk_id wajib diisi")
	}

	performedBy, _ := c.Locals("username").(string)
	if performedBy == "" {
		performedBy = "system"
	}

	result, err := h.manager.ExecuteBulk(c.UserContext(), req.BulkID, performedBy)
	if err != nil {
		return h.mapError(c, err)
	}
	return domain.SuccessResponse(c, fiber.StatusOK, result)
}

// BulkTemplate menangani GET /provisioning/bulk/template.
// Mengembalikan file CSV template untuk bulk provisioning.
func (h *ProvisioningHandler) BulkTemplate(c *fiber.Ctx) error {
	template := h.manager.GetBulkTemplate()

	c.Set("Content-Type", "text/csv")
	c.Set("Content-Disposition", "attachment; filename=bulk_provisioning_template.csv")
	return c.Send(template)
}
