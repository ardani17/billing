// reseller_dashboard.go menangani HTTP permintaan untuk dashboard reseller.
// Termasuk: summary, buy voucher, my vouchers, print, deposit history, transaction history.
package handler

import (
	"errors"
	"strconv"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
	"github.com/ispboss/ispboss/services/billing-api/internal/usecase"
)

// ResellerDashboardHandler menangani HTTP permintaan untuk dashboard reseller.
type ResellerDashboardHandler struct {
	resellerUsecase        *usecase.ResellerUsecase
	packageUsecase         domain.PackageUsecase
	voucherPurchaseUsecase *usecase.VoucherPurchaseUsecase
	voucherUsecase         *usecase.VoucherUsecase
	voucherPrintUsecase    *usecase.VoucherPrintUsecase
	resellerTxRepo         domain.ResellerTransactionRepository
	validate               *validator.Validate
	logger                 zerolog.Logger
}

// NewResellerDashboardHandler membuat instance baru ResellerDashboardHandler.
func NewResellerDashboardHandler(
	resellerUsecase *usecase.ResellerUsecase,
	packageUsecase domain.PackageUsecase,
	voucherPurchaseUsecase *usecase.VoucherPurchaseUsecase,
	voucherUsecase *usecase.VoucherUsecase,
	voucherPrintUsecase *usecase.VoucherPrintUsecase,
	resellerTxRepo domain.ResellerTransactionRepository,
	logger zerolog.Logger,
) *ResellerDashboardHandler {
	v := validator.New()
	RegisterCustomValidators(v)
	return &ResellerDashboardHandler{
		resellerUsecase:        resellerUsecase,
		packageUsecase:         packageUsecase,
		voucherPurchaseUsecase: voucherPurchaseUsecase,
		voucherUsecase:         voucherUsecase,
		voucherPrintUsecase:    voucherPrintUsecase,
		resellerTxRepo:         resellerTxRepo,
		validate:               v,
		logger:                 logger,
	}
}

// VoucherPackages menangani GET /v1/reseller/packages.
// Mengembalikan paket voucher aktif milik tenant reseller yang login.
func (h *ResellerDashboardHandler) VoucherPackages(c *fiber.Ctx) error {
	tenantID, ok := c.Locals("tenant_id").(string)
	if !ok || tenantID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant reseller tidak teridentifikasi")
	}

	active := true
	params := domain.PackageListParams{
		TenantID:  tenantID,
		Type:      string(domain.PackageTypeVoucher),
		IsActive:  &active,
		Page:      1,
		PageSize:  50,
		SortBy:    "name",
		SortOrder: "asc",
	}

	if page := c.Query("page"); page != "" {
		params.Page, _ = strconv.Atoi(page)
	}
	if pageSize := c.Query("page_size"); pageSize != "" {
		params.PageSize, _ = strconv.Atoi(pageSize)
	}
	if sortBy := c.Query("sort_by"); sortBy != "" {
		params.SortBy = sortBy
	}
	if sortOrder := c.Query("sort_order"); sortOrder != "" {
		params.SortOrder = sortOrder
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
		h.logger.Error().Err(err).Msg("gagal mengambil paket voucher reseller")
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal mengambil paket voucher")
	}

	return domain.SuccessResponse(c, fiber.StatusOK, result)
}

// Summary menangani GET /v1/reseller/dashboard.
// Mengembalikan ringkasan dashboard: saldo, voucher terjual hari ini, voucher tersedia.
func (h *ResellerDashboardHandler) Summary(c *fiber.Ctx) error {
	resellerID, ok := c.Locals("user_id").(string)
	if !ok || resellerID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "reseller tidak terautentikasi")
	}

	// Ambil data reseller untuk saldo
	detail, err := h.resellerUsecase.GetByID(c.Context(), resellerID, false)
	if err != nil {
		return h.mapDashboardError(c, err)
	}

	// Ambil jumlah voucher terjual hari ini
	soldToday, err := h.voucherUsecase.CountSoldToday(c.Context(), resellerID)
	if err != nil {
		h.logger.Error().Err(err).Str("reseller_id", resellerID).Msg("gagal menghitung voucher terjual hari ini")
		soldToday = 0
	}

	// Ambil jumlah voucher tersedia (status terjual, milik reseller ini)
	availableVouchers, err := h.voucherUsecase.CountAvailableByReseller(c.Context(), resellerID)
	if err != nil {
		h.logger.Error().Err(err).Str("reseller_id", resellerID).Msg("gagal menghitung voucher tersedia")
		availableVouchers = 0
	}

	summary := domain.DashboardSummary{
		Balance:           detail.Reseller.Balance,
		SoldToday:         soldToday,
		AvailableVouchers: availableVouchers,
	}

	return domain.SuccessResponse(c, fiber.StatusOK, summary)
}

// Buy menangani POST /v1/reseller/vouchers/buy.
// Melakukan pembelian voucher oleh reseller.
func (h *ResellerDashboardHandler) Buy(c *fiber.Ctx) error {
	resellerID, ok := c.Locals("user_id").(string)
	if !ok || resellerID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "reseller tidak terautentikasi")
	}

	var req domain.BuyVoucherRequest
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

	result, err := h.voucherPurchaseUsecase.Buy(c.Context(), resellerID, req)
	if err != nil {
		return h.mapDashboardError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, result)
}

// MyVouchers menangani GET /v1/reseller/vouchers.
// Mengembalikan daftar voucher milik reseller dengan paginasi.
func (h *ResellerDashboardHandler) MyVouchers(c *fiber.Ctx) error {
	resellerID, ok := c.Locals("user_id").(string)
	if !ok || resellerID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "reseller tidak terautentikasi")
	}

	tenantID, _ := c.Locals("tenant_id").(string)

	var params domain.ResellerVoucherListParams
	params.ResellerID = resellerID
	params.TenantID = tenantID
	params.Page, _ = strconv.Atoi(c.Query("page", "1"))
	params.PageSize, _ = strconv.Atoi(c.Query("page_size", "25"))
	params.Status = c.Query("status")
	params.PackageID = c.Query("package_id")
	params.SortBy = c.Query("sort_by")
	params.SortOrder = c.Query("sort_order")

	if err := h.validate.Struct(params); err != nil {
		var ve validator.ValidationErrors
		if errors.As(err, &ve) {
			return domain.ErrorResponse(c, fiber.StatusBadRequest, "VALIDATION_ERROR", "validasi gagal", mapValidationErrors(ve)...)
		}
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "VALIDATION_ERROR", err.Error())
	}

	result, err := h.voucherUsecase.ListByReseller(c.Context(), params)
	if err != nil {
		h.logger.Error().Err(err).Msg("gagal mengambil daftar voucher reseller")
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal mengambil daftar voucher")
	}

	return domain.SuccessResponse(c, fiber.StatusOK, result)
}

// Print menangani POST /v1/reseller/vouchers/print.
// Menghasilkan PDF voucher milik reseller yang terautentikasi.
// Memverifikasi bahwa semua voucher milik reseller yang login.
func (h *ResellerDashboardHandler) Print(c *fiber.Ctx) error {
	resellerID, ok := c.Locals("user_id").(string)
	if !ok || resellerID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "reseller tidak terautentikasi")
	}

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

	// Verifikasi semua voucher milik reseller yang terautentikasi
	if err := h.voucherUsecase.VerifyOwnership(c.Context(), req.VoucherIDs, resellerID); err != nil {
		if errors.Is(err, domain.ErrVoucherForbidden) {
			return domain.ErrorResponse(c, fiber.StatusForbidden, "VOUCHER_FORBIDDEN", err.Error())
		}
		return h.mapDashboardError(c, err)
	}

	// Ambil informasi tenant dari JWT locals
	tenantName, _ := c.Locals("tenant_name").(string)
	tenantPhone, _ := c.Locals("tenant_phone").(string)

	pdfBytes, err := h.voucherPrintUsecase.GeneratePDF(c.Context(), req.VoucherIDs, tenantName, tenantPhone)
	if err != nil {
		h.logger.Error().Err(err).Msg("gagal generate PDF voucher reseller")
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal generate PDF voucher")
	}

	// Set header untuk respons PDF
	c.Set("Content-Type", "application/pdf")
	c.Set("Content-Disposition", "attachment; filename=my-vouchers.pdf")

	return c.Send(pdfBytes)
}

// DepositHistory menangani GET /v1/reseller/deposit.
// Mengembalikan riwayat deposit reseller dengan paginasi.
func (h *ResellerDashboardHandler) DepositHistory(c *fiber.Ctx) error {
	resellerID, ok := c.Locals("user_id").(string)
	if !ok || resellerID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "reseller tidak terautentikasi")
	}

	tenantID, _ := c.Locals("tenant_id").(string)

	var params domain.ResellerTxListParams
	params.ResellerID = resellerID
	params.TenantID = tenantID
	params.Page, _ = strconv.Atoi(c.Query("page", "1"))
	params.PageSize, _ = strconv.Atoi(c.Query("page_size", "25"))
	params.SortOrder = c.Query("sort_order")

	if err := h.validate.Struct(params); err != nil {
		var ve validator.ValidationErrors
		if errors.As(err, &ve) {
			return domain.ErrorResponse(c, fiber.StatusBadRequest, "VALIDATION_ERROR", "validasi gagal", mapValidationErrors(ve)...)
		}
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "VALIDATION_ERROR", err.Error())
	}

	result, err := h.resellerTxRepo.ListDepositsByReseller(c.Context(), params)
	if err != nil {
		h.logger.Error().Err(err).Msg("gagal mengambil riwayat deposit reseller")
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal mengambil riwayat deposit")
	}

	return domain.SuccessResponse(c, fiber.StatusOK, result)
}

// TransactionHistory menangani GET /v1/reseller/history.
// Mengembalikan riwayat transaksi reseller dengan paginasi.
func (h *ResellerDashboardHandler) TransactionHistory(c *fiber.Ctx) error {
	resellerID, ok := c.Locals("user_id").(string)
	if !ok || resellerID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "reseller tidak terautentikasi")
	}

	tenantID, _ := c.Locals("tenant_id").(string)

	var params domain.ResellerTxListParams
	params.ResellerID = resellerID
	params.TenantID = tenantID
	params.Page, _ = strconv.Atoi(c.Query("page", "1"))
	params.PageSize, _ = strconv.Atoi(c.Query("page_size", "25"))
	params.Type = c.Query("type")
	params.SortOrder = c.Query("sort_order")

	if err := h.validate.Struct(params); err != nil {
		var ve validator.ValidationErrors
		if errors.As(err, &ve) {
			return domain.ErrorResponse(c, fiber.StatusBadRequest, "VALIDATION_ERROR", "validasi gagal", mapValidationErrors(ve)...)
		}
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "VALIDATION_ERROR", err.Error())
	}

	result, err := h.resellerTxRepo.ListByReseller(c.Context(), params)
	if err != nil {
		h.logger.Error().Err(err).Msg("gagal mengambil riwayat transaksi reseller")
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal mengambil riwayat transaksi")
	}

	return domain.SuccessResponse(c, fiber.StatusOK, result)
}

// mapDashboardError memetakan domain error ke HTTP error respons untuk dashboard reseller.
func (h *ResellerDashboardHandler) mapDashboardError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, domain.ErrResellerNotFound):
		return domain.ErrorResponse(c, fiber.StatusNotFound, "RESELLER_NOT_FOUND", err.Error())
	case errors.Is(err, domain.ErrResellerAccountDisabled):
		return domain.ErrorResponse(c, fiber.StatusForbidden, "ACCOUNT_DISABLED", err.Error())
	case errors.Is(err, domain.ErrInsufficientBalance):
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "INSUFFICIENT_BALANCE", err.Error())
	case errors.Is(err, domain.ErrDailyLimitExceeded):
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "DAILY_LIMIT_EXCEEDED", err.Error())
	case errors.Is(err, domain.ErrInvalidPackageType):
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "INVALID_PACKAGE_TYPE", err.Error())
	case errors.Is(err, domain.ErrPackageNotActive):
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "PACKAGE_NOT_ACTIVE", err.Error())
	case errors.Is(err, domain.ErrVoucherPackagePriceInvalid):
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "VOUCHER_PACKAGE_PRICE_INVALID", err.Error())
	case errors.Is(err, domain.ErrVoucherStockInsufficient):
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "VOUCHER_STOCK_INSUFFICIENT", err.Error())
	case errors.Is(err, domain.ErrVoucherForbidden):
		return domain.ErrorResponse(c, fiber.StatusForbidden, "VOUCHER_FORBIDDEN", err.Error())
	case errors.Is(err, domain.ErrVoucherNotFound):
		return domain.ErrorResponse(c, fiber.StatusNotFound, "VOUCHER_NOT_FOUND", err.Error())
	default:
		h.logger.Error().Err(err).Msg("internal error pada reseller dashboard handler")
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "terjadi kesalahan internal")
	}
}
