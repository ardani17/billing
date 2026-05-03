// package_handler.go menangani HTTP request untuk manajemen paket.
// Termasuk: list, get, create, update, dan delete.
package handler

import (
	"errors"
	"strconv"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

// PackageHandler menangani HTTP request untuk manajemen paket.
type PackageHandler struct {
	packageUsecase domain.PackageUsecase
	validate       *validator.Validate
	logger         zerolog.Logger
}

// NewPackageHandler membuat instance baru PackageHandler.
// Mendaftarkan custom validator untuk validasi type-conditional.
func NewPackageHandler(packageUsecase domain.PackageUsecase, logger zerolog.Logger) *PackageHandler {
	v := validator.New()
	v.RegisterStructValidation(validatePackageCreate, domain.CreatePackageRequest{})
	v.RegisterStructValidation(validatePackageUpdate, domain.UpdatePackageRequest{})
	return &PackageHandler{
		packageUsecase: packageUsecase,
		validate:       v,
		logger:         logger,
	}
}

// List menangani GET /v1/packages.
// Mengembalikan daftar paket dengan paginasi, filter, dan sorting.
func (h *PackageHandler) List(c *fiber.Ctx) error {
	tenantID, ok := c.Locals("tenant_id").(string)
	if !ok || tenantID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant tidak teridentifikasi")
	}

	var params domain.PackageListParams
	params.TenantID = tenantID
	params.Page, _ = strconv.Atoi(c.Query("page", "1"))
	params.PageSize, _ = strconv.Atoi(c.Query("page_size", "25"))
	params.Search = c.Query("search")
	params.Type = c.Query("type")
	params.SortBy = c.Query("sort_by")
	params.SortOrder = c.Query("sort_order")

	if isActiveStr := c.Query("is_active"); isActiveStr != "" {
		val := isActiveStr == "true"
		params.IsActive = &val
	}

	if err := h.validate.Struct(params); err != nil {
		var ve validator.ValidationErrors
		if errors.As(err, &ve) {
			return domain.ErrorResponse(c, fiber.StatusBadRequest, "VALIDATION_ERROR", "validasi gagal", mapValidationErrors(ve)...)
		}
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "VALIDATION_ERROR", err.Error())
	}

	result, err := h.packageUsecase.List(c.Context(), params)
	if err != nil {
		h.logger.Error().Err(err).Msg("gagal mengambil daftar paket")
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal mengambil daftar paket")
	}

	return domain.SuccessResponse(c, fiber.StatusOK, result)
}

// Get menangani GET /v1/packages/:id.
// Mengembalikan detail paket, opsional termasuk audit logs.
func (h *PackageHandler) Get(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "package ID wajib diisi")
	}

	includeAudit := strings.Contains(c.Query("include"), "audit_logs")

	detail, err := h.packageUsecase.GetByID(c.Context(), id, includeAudit)
	if err != nil {
		return h.mapPackageError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, detail)
}

// Create menangani POST /v1/packages.
// Membuat paket baru (PPPoE atau Voucher).
func (h *PackageHandler) Create(c *fiber.Ctx) error {
	tenantID, ok := c.Locals("tenant_id").(string)
	if !ok || tenantID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant tidak teridentifikasi")
	}

	var req domain.CreatePackageRequest
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

	pkg, err := h.packageUsecase.Create(c.Context(), tenantID, req, actor)
	if err != nil {
		return h.mapPackageError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusCreated, pkg)
}

// Update menangani PUT /v1/packages/:id.
// Memperbarui data paket.
func (h *PackageHandler) Update(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "package ID wajib diisi")
	}

	var req domain.UpdatePackageRequest
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

	pkg, err := h.packageUsecase.Update(c.Context(), id, req, actor)
	if err != nil {
		return h.mapPackageError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, pkg)
}

// Delete menangani DELETE /v1/packages/:id.
// Hard delete paket dengan konfirmasi nama.
func (h *PackageHandler) Delete(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "package ID wajib diisi")
	}

	var req domain.DeletePackageRequest
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

	err := h.packageUsecase.Delete(c.Context(), id, req.ConfirmationName, actor)
	if err != nil {
		return h.mapPackageError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, fiber.Map{
		"message": "paket berhasil dihapus",
	})
}

// extractActor mengambil informasi aktor dari Fiber locals (di-set oleh auth middleware).
func (h *PackageHandler) extractActor(c *fiber.Ctx) domain.ActorInfo {
	actorID, _ := c.Locals("user_id").(string)
	actorName, _ := c.Locals("user_name").(string)
	return domain.ActorInfo{
		ActorID:   actorID,
		ActorName: actorName,
	}
}

// mapPackageError memetakan domain error ke HTTP error response untuk paket.
func (h *PackageHandler) mapPackageError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, domain.ErrPackageNotFound):
		return domain.ErrorResponse(c, fiber.StatusNotFound, "PACKAGE_NOT_FOUND", err.Error())
	case errors.Is(err, domain.ErrPackageNameDuplicate):
		return domain.ErrorResponse(c, fiber.StatusConflict, "PACKAGE_NAME_DUPLICATE", err.Error())
	case errors.Is(err, domain.ErrPackageHasCustomers):
		return domain.ErrorResponse(c, fiber.StatusConflict, "PACKAGE_HAS_CUSTOMERS", err.Error())
	case errors.Is(err, domain.ErrPackageAlreadyActive):
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "PACKAGE_ALREADY_ACTIVE", err.Error())
	case errors.Is(err, domain.ErrPackageAlreadyInactive):
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "PACKAGE_ALREADY_INACTIVE", err.Error())
	case errors.Is(err, domain.ErrInsufficientMargin):
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "INSUFFICIENT_MARGIN", err.Error())
	case errors.Is(err, domain.ErrTypeChangeNotAllowed):
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "TYPE_CHANGE_NOT_ALLOWED", err.Error())
	case errors.Is(err, domain.ErrBurstFieldsIncomplete):
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BURST_FIELDS_INCOMPLETE", err.Error())
	case errors.Is(err, domain.ErrConfirmationMismatch):
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "CONFIRMATION_MISMATCH", err.Error())
	default:
		h.logger.Error().Err(err).Msg("internal error pada package handler")
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "terjadi kesalahan internal")
	}
}

// validatePackageCreate melakukan validasi struct-level untuk CreatePackageRequest.
// Memeriksa field wajib berdasarkan tipe paket (PPPoE vs Voucher).
func validatePackageCreate(sl validator.StructLevel) {
	req := sl.Current().Interface().(domain.CreatePackageRequest)

	if req.Type == "pppoe" {
		if req.MonthlyPrice == nil {
			sl.ReportError(req.MonthlyPrice, "monthly_price", "MonthlyPrice", "required_for_pppoe", "")
		}
		if req.BandwidthType == "" {
			sl.ReportError(req.BandwidthType, "bandwidth_type", "BandwidthType", "required_for_pppoe", "")
		}
		// Validasi quota_type hanya boleh unlimited/monthly_quota/fup untuk PPPoE
		if req.QuotaType != "" && req.QuotaType != "unlimited" && req.QuotaType != "monthly_quota" && req.QuotaType != "fup" {
			sl.ReportError(req.QuotaType, "quota_type", "QuotaType", "invalid_quota_type_for_pppoe", "")
		}
	} else if req.Type == "voucher" {
		if req.SellPrice == nil {
			sl.ReportError(req.SellPrice, "sell_price", "SellPrice", "required_for_voucher", "")
		}
		if req.ResellerPrice == nil {
			sl.ReportError(req.ResellerPrice, "reseller_price", "ResellerPrice", "required_for_voucher", "")
		}
		if req.DurationValue == nil {
			sl.ReportError(req.DurationValue, "duration_value", "DurationValue", "required_for_voucher", "")
		}
		if req.DurationUnit == "" {
			sl.ReportError(req.DurationUnit, "duration_unit", "DurationUnit", "required_for_voucher", "")
		}
		// Validasi quota_type hanya boleh unlimited/quota untuk Voucher
		if req.QuotaType != "" && req.QuotaType != "unlimited" && req.QuotaType != "quota" {
			sl.ReportError(req.QuotaType, "quota_type", "QuotaType", "invalid_quota_type_for_voucher", "")
		}
	}
}

// validatePackageUpdate melakukan validasi struct-level untuk UpdatePackageRequest.
// Sama seperti create, tapi semua field opsional — hanya validasi jika field diisi.
func validatePackageUpdate(sl validator.StructLevel) {
	// Update tidak memerlukan validasi struct-level tambahan karena semua field opsional.
	// Validasi type-conditional dilakukan di usecase layer setelah merge dengan data existing.
	_ = sl.Current().Interface().(domain.UpdatePackageRequest)
}
