// map_export_handler.go menangani HTTP permintaan untuk export peta.
// Mendukung format KML, KMZ, GeoJSON, dan CSV.
// Dataset besar (>500 items) diproses async, kembalikan job_id.
package handler

import (
	"errors"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"

	"github.com/ispboss/ispboss/pkg/tenant"
	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// ExportHandler menangani HTTP permintaan untuk export peta.
type ExportHandler struct {
	manager  domain.MapExportManager
	validate *validator.Validate
}

// NewExportHandler membuat instance baru ExportHandler.
func NewExportHandler(manager domain.MapExportManager) *ExportHandler {
	return &ExportHandler{
		manager:  manager,
		validate: validator.New(),
	}
}

// Export menangani POST /export.
// Parsing body, validasi format dan layers, lalu export data peta.
// Jika dataset kecil -> kembalikan file langsung. Jika besar -> kembalikan job_id.
func (h *ExportHandler) Export(c *fiber.Ctx) error {
	tenantID := tenant.FromContext(c.UserContext())
	if tenantID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant tidak teridentifikasi")
	}

	var req domain.ExportRequest
	if err := c.BodyParser(&req); err != nil {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "request body tidak valid")
	}

	if err := h.validate.Struct(req); err != nil {
		return h.validationError(c, err)
	}

	result, err := h.manager.Export(c.UserContext(), tenantID, req)
	if err != nil {
		return h.mapError(c, err)
	}

	// Jika async, kembalikan 202 Accepted dengan job_id
	if result.Async {
		return c.Status(fiber.StatusAccepted).JSON(domain.APIResponse{
			Success: true,
			Data:    result,
		})
	}

	// Jika sync, kembalikan file langsung
	c.Set("Content-Type", result.ContentType)
	c.Set("Content-Disposition", "attachment; filename="+result.FileName)
	return c.Send(result.FileBytes)
}

// GetExportStatus menangani GET /export/status/:job_id.
// Mengecek status export async berdasarkan job_id.
func (h *ExportHandler) GetExportStatus(c *fiber.Ctx) error {
	jobID := c.Params("job_id")
	if jobID == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "job_id wajib diisi")
	}

	status, err := h.manager.GetExportStatus(c.UserContext(), jobID)
	if err != nil {
		return h.mapError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, status)
}

// mapError memetakan domain error export ke HTTP error respons.
func (h *ExportHandler) mapError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, domain.ErrUnsupportedFormat):
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "UNSUPPORTED_FORMAT", err.Error())
	case errors.Is(err, domain.ErrExportNotFound):
		return domain.ErrorResponse(c, fiber.StatusNotFound, "EXPORT_NOT_FOUND", err.Error())
	default:
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "terjadi kesalahan internal")
	}
}

// validationError menangani error validasi dari go-playground/validator.
func (h *ExportHandler) validationError(c *fiber.Ctx, err error) error {
	var ve validator.ValidationErrors
	if errors.As(err, &ve) {
		fields := make([]domain.FieldError, 0, len(ve))
		for _, fe := range ve {
			fields = append(fields, domain.FieldError{
				Field:   toSnakeCase(fe.Field()),
				Message: validationMessage(fe),
			})
		}
		return domain.ErrorResponse(c, fiber.StatusUnprocessableEntity, "VALIDATION_ERROR", "validasi gagal", fields...)
	}
	return domain.ErrorResponse(c, fiber.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
}
