// voucher_handler.go menangani HTTP request untuk manajemen voucher (admin).
// Termasuk: generate, list, get, bulk void, bulk assign, export CSV.
package handler

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
	"github.com/ispboss/ispboss/services/billing-api/internal/usecase"
)

// alphanumHyphenRegex memvalidasi format prefix voucher: hanya huruf, angka, dan hyphen.
var alphanumHyphenRegex = regexp.MustCompile(`^[A-Za-z0-9\-]+$`)

// VoucherHandler menangani HTTP request untuk manajemen voucher (admin).
type VoucherHandler struct {
	voucherUsecase       *usecase.VoucherUsecase
	voucherActionUsecase *usecase.VoucherActionUsecase
	validate             *validator.Validate
	logger               zerolog.Logger
}

// NewVoucherHandler membuat instance baru VoucherHandler.
// Mendaftarkan custom validator alphanum_hyphen untuk prefix voucher.
func NewVoucherHandler(
	voucherUsecase *usecase.VoucherUsecase,
	voucherActionUsecase *usecase.VoucherActionUsecase,
	logger zerolog.Logger,
) *VoucherHandler {
	v := validator.New()
	RegisterCustomValidators(v)
	_ = v.RegisterValidation("alphanum_hyphen", validateAlphanumHyphen)
	return &VoucherHandler{
		voucherUsecase:       voucherUsecase,
		voucherActionUsecase: voucherActionUsecase,
		validate:             v,
		logger:               logger,
	}
}

// validateAlphanumHyphen memvalidasi bahwa field hanya berisi huruf, angka, dan hyphen.
func validateAlphanumHyphen(fl validator.FieldLevel) bool {
	return alphanumHyphenRegex.MatchString(fl.Field().String())
}

// Generate menangani POST /v1/vouchers/generate.
// Menghasilkan batch voucher baru. Mengembalikan 201 (sync) atau 202 (async).
func (h *VoucherHandler) Generate(c *fiber.Ctx) error {
	tenantID, ok := c.Locals("tenant_id").(string)
	if !ok || tenantID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant tidak teridentifikasi")
	}

	var req domain.GenerateVoucherRequest
	if err := c.BodyParser(&req); err != nil {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "request body tidak valid")
	}

	if err := h.validate.Struct(req); err != nil {
		var ve validator.ValidationErrors
		if errors.As(err, &ve) {
			return domain.ErrorResponse(c, fiber.StatusBadRequest, "VALIDATION_ERROR", "validasi gagal", mapValidationErrors(ve)...)
		}
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "VALIDATION_ERROR", err.Error())
	}

	actor := h.extractActor(c)

	result, err := h.voucherUsecase.Generate(c.Context(), tenantID, req, actor)
	if err != nil {
		return h.mapVoucherError(c, err)
	}

	// Jika async (ada JobID), kembalikan 202 Accepted
	if result.JobID != "" {
		return domain.SuccessResponse(c, fiber.StatusAccepted, result)
	}

	return domain.SuccessResponse(c, fiber.StatusCreated, result)
}

// List menangani GET /v1/vouchers.
// Mengembalikan daftar voucher dengan paginasi, filter, dan sorting.
func (h *VoucherHandler) List(c *fiber.Ctx) error {
	tenantID, ok := c.Locals("tenant_id").(string)
	if !ok || tenantID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant tidak teridentifikasi")
	}

	var params domain.VoucherListParams
	params.TenantID = tenantID
	params.Page, _ = strconv.Atoi(c.Query("page", "1"))
	params.PageSize, _ = strconv.Atoi(c.Query("page_size", "25"))
	params.Search = c.Query("search")
	params.PackageID = c.Query("package_id")
	params.Status = c.Query("status")
	params.ResellerID = c.Query("reseller_id")
	params.SortBy = c.Query("sort_by")
	params.SortOrder = c.Query("sort_order")

	if err := h.validate.Struct(params); err != nil {
		var ve validator.ValidationErrors
		if errors.As(err, &ve) {
			return domain.ErrorResponse(c, fiber.StatusBadRequest, "VALIDATION_ERROR", "validasi gagal", mapValidationErrors(ve)...)
		}
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "VALIDATION_ERROR", err.Error())
	}

	result, err := h.voucherUsecase.List(c.Context(), params)
	if err != nil {
		h.logger.Error().Err(err).Msg("gagal mengambil daftar voucher")
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal mengambil daftar voucher")
	}

	return domain.SuccessResponse(c, fiber.StatusOK, result)
}

// Get menangani GET /v1/vouchers/:id.
// Mengembalikan detail voucher termasuk audit logs.
func (h *VoucherHandler) Get(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "voucher ID wajib diisi")
	}

	detail, err := h.voucherUsecase.GetByID(c.Context(), id)
	if err != nil {
		return h.mapVoucherError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, detail)
}

// Activate menangani POST /v1/vouchers/activate.
// Mengaktifkan kode voucher dan menerbitkan event untuk provisioning Hotspot.
func (h *VoucherHandler) Activate(c *fiber.Ctx) error {
	tenantID, ok := c.Locals("tenant_id").(string)
	if !ok || tenantID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant tidak teridentifikasi")
	}

	var req domain.ActivateVoucherRequest
	if err := c.BodyParser(&req); err != nil {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "request body tidak valid")
	}
	if err := h.validate.Struct(req); err != nil {
		var ve validator.ValidationErrors
		if errors.As(err, &ve) {
			return domain.ErrorResponse(c, fiber.StatusBadRequest, "VALIDATION_ERROR", "validasi gagal", mapValidationErrors(ve)...)
		}
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "VALIDATION_ERROR", err.Error())
	}

	voucher, err := h.voucherUsecase.Activate(c.Context(), tenantID, req, h.extractActor(c))
	if err != nil {
		return h.mapVoucherError(c, err)
	}
	return domain.SuccessResponse(c, fiber.StatusOK, voucher)
}

// BulkVoid menangani POST /v1/vouchers/bulk/void.
// Mem-void beberapa voucher sekaligus (hanya status tersedia).
func (h *VoucherHandler) BulkVoid(c *fiber.Ctx) error {
	var req domain.BulkVoucherIDsRequest
	if err := c.BodyParser(&req); err != nil {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "request body tidak valid")
	}

	if err := h.validate.Struct(req); err != nil {
		var ve validator.ValidationErrors
		if errors.As(err, &ve) {
			return domain.ErrorResponse(c, fiber.StatusBadRequest, "VALIDATION_ERROR", "validasi gagal", mapValidationErrors(ve)...)
		}
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "VALIDATION_ERROR", err.Error())
	}

	actor := h.extractActor(c)

	result, err := h.voucherActionUsecase.BulkVoid(c.Context(), req.VoucherIDs, actor)
	if err != nil {
		return h.mapVoucherError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, result)
}

// BulkAssign menangani POST /v1/vouchers/bulk/assign.
// Meng-assign beberapa voucher ke reseller (admin assignment, tanpa potong saldo).
func (h *VoucherHandler) BulkAssign(c *fiber.Ctx) error {
	var req domain.BulkAssignRequest
	if err := c.BodyParser(&req); err != nil {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "request body tidak valid")
	}

	if err := h.validate.Struct(req); err != nil {
		var ve validator.ValidationErrors
		if errors.As(err, &ve) {
			return domain.ErrorResponse(c, fiber.StatusBadRequest, "VALIDATION_ERROR", "validasi gagal", mapValidationErrors(ve)...)
		}
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "VALIDATION_ERROR", err.Error())
	}

	actor := h.extractActor(c)

	result, err := h.voucherActionUsecase.BulkAssign(c.Context(), req.VoucherIDs, req.ResellerID, actor)
	if err != nil {
		return h.mapVoucherError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, result)
}

// Export menangani GET /v1/vouchers/export.
// Mengekspor daftar voucher ke format CSV.
func (h *VoucherHandler) Export(c *fiber.Ctx) error {
	tenantID, ok := c.Locals("tenant_id").(string)
	if !ok || tenantID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant tidak teridentifikasi")
	}

	var params domain.VoucherListParams
	params.TenantID = tenantID
	params.Search = c.Query("search")
	params.PackageID = c.Query("package_id")
	params.Status = c.Query("status")
	params.ResellerID = c.Query("reseller_id")

	csvBytes, err := h.voucherActionUsecase.ExportCSV(c.Context(), params)
	if err != nil {
		h.logger.Error().Err(err).Msg("gagal export voucher CSV")
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal export voucher")
	}

	// Set header untuk download file CSV
	filename := fmt.Sprintf("vouchers_%s.csv", time.Now().Format("20060102_150405"))
	c.Set("Content-Type", "text/csv; charset=utf-8")
	c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))

	return c.Send(csvBytes)
}

// extractActor mengambil informasi aktor dari Fiber locals (di-set oleh auth middleware).
func (h *VoucherHandler) extractActor(c *fiber.Ctx) domain.ActorInfo {
	actorID, _ := c.Locals("user_id").(string)
	actorName, _ := c.Locals("user_name").(string)
	return domain.ActorInfo{
		ActorID:   actorID,
		ActorName: actorName,
	}
}

// mapVoucherError memetakan domain error ke HTTP error response untuk voucher.
func (h *VoucherHandler) mapVoucherError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, domain.ErrVoucherNotFound):
		return domain.ErrorResponse(c, fiber.StatusNotFound, "VOUCHER_NOT_FOUND", err.Error())
	case errors.Is(err, domain.ErrInvalidVoucherTransition):
		return domain.ErrorResponse(c, fiber.StatusUnprocessableEntity, "INVALID_STATUS_TRANSITION", err.Error())
	case errors.Is(err, domain.ErrInvalidPackageType):
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "INVALID_PACKAGE_TYPE", err.Error())
	case errors.Is(err, domain.ErrPackageNotFound):
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "PACKAGE_NOT_FOUND", err.Error())
	case errors.Is(err, domain.ErrResellerNotFound):
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "RESELLER_NOT_FOUND", err.Error())
	default:
		h.logger.Error().Err(err).Msg("internal error pada voucher handler")
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "terjadi kesalahan internal")
	}
}
