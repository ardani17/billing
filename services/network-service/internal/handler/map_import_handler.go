// map_import_handler.go menangani HTTP permintaan untuk import peta.
// Mendukung format KML, KMZ, dan GeoJSON.
// Alur: Preview (parsing file) -> Execute (apply mapping).
package handler

import (
	"errors"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"

	"github.com/ispboss/ispboss/pkg/tenant"
	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// ImportHandler menangani HTTP permintaan untuk import peta.
type ImportHandler struct {
	manager  domain.MapImportManager
	validate *validator.Validate
}

// NewImportHandler membuat instance baru ImportHandler.
func NewImportHandler(manager domain.MapImportManager) *ImportHandler {
	return &ImportHandler{
		manager:  manager,
		validate: validator.New(),
	}
}

// Preview menangani POST /import.
// Menerima file via multipart form, parsing, dan kembalikan preview item.
func (h *ImportHandler) Preview(c *fiber.Ctx) error {
	tenantID := tenant.FromContext(c.UserContext())
	if tenantID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant tidak teridentifikasi")
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "file import wajib diisi")
	}

	file, err := fileHeader.Open()
	if err != nil {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "gagal membaca file import")
	}
	defer file.Close()

	preview, err := h.manager.Preview(c.UserContext(), tenantID, file, fileHeader.Filename)
	if err != nil {
		return h.mapError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, preview)
}

// Execute menangani POST /import/execute.
// Parsing body mapping, validasi, lalu eksekusi import.
func (h *ImportHandler) Execute(c *fiber.Ctx) error {
	var req domain.ImportMapping
	if err := c.BodyParser(&req); err != nil {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "request body tidak valid")
	}

	if err := h.validate.Struct(req); err != nil {
		return h.validationError(c, err)
	}

	summary, err := h.manager.Execute(c.UserContext(), req.ImportID, req)
	if err != nil {
		return h.mapError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, summary)
}

// GetImportStatus menangani GET /import/status/:job_id.
// Mengecek status import async berdasarkan job_id.
func (h *ImportHandler) GetImportStatus(c *fiber.Ctx) error {
	jobID := c.Params("job_id")
	if jobID == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "job_id wajib diisi")
	}

	status, err := h.manager.GetImportStatus(c.UserContext(), jobID)
	if err != nil {
		return h.mapError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, status)
}

// mapError memetakan domain error import ke HTTP error respons.
func (h *ImportHandler) mapError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, domain.ErrUnsupportedFormat):
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "UNSUPPORTED_FORMAT", err.Error())
	case errors.Is(err, domain.ErrInvalidImportFile):
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "INVALID_IMPORT_FILE", err.Error())
	case errors.Is(err, domain.ErrImportNotFound):
		return domain.ErrorResponse(c, fiber.StatusNotFound, "IMPORT_NOT_FOUND", err.Error())
	default:
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "terjadi kesalahan internal")
	}
}

// validationError menangani error validasi dari go-playground/validator.
func (h *ImportHandler) validationError(c *fiber.Ctx, err error) error {
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
