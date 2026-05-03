// customer_io.go menangani HTTP request untuk import/export pelanggan.
// Termasuk: import CSV/Excel, export CSV/Excel, dan download template import.
package handler

import (
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

// Import menangani POST /v1/customers/import.
// Menerima file CSV/Excel dan mengirim job import ke queue.
func (h *CustomerHandler) Import(c *fiber.Ctx) error {
	tenantID, ok := c.Locals("tenant_id").(string)
	if !ok || tenantID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant tidak teridentifikasi")
	}

	// Parse multipart file
	fileHeader, err := c.FormFile("file")
	if err != nil {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "file wajib diunggah")
	}

	// Read file content
	file, err := fileHeader.Open()
	if err != nil {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "gagal membaca file")
	}
	defer file.Close()

	fileBytes := make([]byte, fileHeader.Size)
	if _, err := file.Read(fileBytes); err != nil {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "gagal membaca isi file")
	}

	actor := h.extractActor(c)

	jobID, err := h.customerUsecase.ImportCSV(c.Context(), tenantID, fileBytes, fileHeader.Filename, actor)
	if err != nil {
		h.logger.Error().Err(err).Msg("gagal import pelanggan")
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal memproses import")
	}

	return domain.SuccessResponse(c, fiber.StatusAccepted, fiber.Map{
		"job_id":  jobID,
		"message": "import sedang diproses",
	})
}

// Export menangani GET /v1/customers/export.
// Mengirim job export ke queue dan mengembalikan job_id.
func (h *CustomerHandler) Export(c *fiber.Ctx) error {
	tenantID, ok := c.Locals("tenant_id").(string)
	if !ok || tenantID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant tidak teridentifikasi")
	}

	format := c.Query("format", "csv")
	if format != "csv" && format != "xlsx" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "format harus csv atau xlsx")
	}

	// Parse filter params (same as list)
	var params domain.CustomerListParams
	params.TenantID = tenantID
	params.Search = c.Query("search")
	params.Status = c.Query("status")
	params.PackageID = c.Query("package_id")
	params.AreaID = c.Query("area_id")

	// Parse columns
	var columns []string
	if columnsStr := c.Query("columns"); columnsStr != "" {
		columns = strings.Split(columnsStr, ",")
		for i := range columns {
			columns[i] = strings.TrimSpace(columns[i])
		}
	}

	actor := h.extractActor(c)

	jobID, err := h.customerUsecase.ExportCSV(c.Context(), tenantID, params, format, columns, actor)
	if err != nil {
		h.logger.Error().Err(err).Msg("gagal export pelanggan")
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal memproses export")
	}

	return domain.SuccessResponse(c, fiber.StatusAccepted, fiber.Map{
		"job_id":  jobID,
		"message": "export sedang diproses",
	})
}

// ImportTemplate menangani GET /v1/customers/import/template.
// Mengembalikan file CSV template untuk import.
func (h *CustomerHandler) ImportTemplate(c *fiber.Ctx) error {
	templateBytes, err := h.customerUsecase.GetImportTemplate(c.Context())
	if err != nil {
		h.logger.Error().Err(err).Msg("gagal membuat template import")
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal membuat template import")
	}

	c.Set("Content-Type", "text/csv")
	c.Set("Content-Disposition", "attachment; filename=import_template.csv")
	return c.Send(templateBytes)
}
