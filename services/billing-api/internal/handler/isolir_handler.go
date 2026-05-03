// isolir_handler.go menangani HTTP request untuk modul isolir.
// Termasuk: manual sync, pending syncs, summary, waive penalty, dan reactivate.
package handler

import (
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
	"github.com/ispboss/ispboss/services/billing-api/internal/usecase"
)

// IsolirHandler menangani HTTP request untuk modul isolir.
type IsolirHandler struct {
	isolirUsecase *usecase.IsolirUsecase
	logger        zerolog.Logger
}

// NewIsolirHandler membuat instance baru IsolirHandler.
func NewIsolirHandler(isolirUsecase *usecase.IsolirUsecase, logger zerolog.Logger) *IsolirHandler {
	return &IsolirHandler{isolirUsecase: isolirUsecase, logger: logger}
}

// ManualSync menangani POST /v1/isolir/sync/:customer_id.
// Re-publish pending sync event untuk satu pelanggan dan reset retry_count.
func (h *IsolirHandler) ManualSync(c *fiber.Ctx) error {
	customerID := c.Params("customer_id")
	if _, err := uuid.Parse(customerID); err != nil {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "customer_id tidak valid")
	}

	tenantID, ok := c.Locals("tenant_id").(string)
	if !ok || tenantID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant tidak teridentifikasi")
	}
	actor := h.extractActor(c)

	if err := h.isolirUsecase.ManualSync(c.Context(), customerID, actor.ActorID); err != nil {
		return h.mapIsolirError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, fiber.Map{
		"message": "sinkronisasi manual berhasil dipicu",
	})
}

// ManualSyncAll menangani POST /v1/isolir/sync-all.
// Re-publish semua pending/failed sync untuk tenant dan kembalikan jumlahnya.
func (h *IsolirHandler) ManualSyncAll(c *fiber.Ctx) error {
	tenantID, ok := c.Locals("tenant_id").(string)
	if !ok || tenantID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant tidak teridentifikasi")
	}
	actor := h.extractActor(c)

	count, err := h.isolirUsecase.ManualSyncAll(c.Context(), tenantID, actor.ActorID)
	if err != nil {
		h.logger.Error().Err(err).Msg("gagal manual sync all")
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal memproses sinkronisasi")
	}

	return domain.SuccessResponse(c, fiber.StatusOK, fiber.Map{
		"message": "sinkronisasi manual berhasil dipicu",
		"count":   count,
	})
}

// ListPendingSyncs menangani GET /v1/isolir/pending-syncs.
// Mengembalikan daftar pending sync dengan paginasi dan filter status.
func (h *IsolirHandler) ListPendingSyncs(c *fiber.Ctx) error {
	tenantID, ok := c.Locals("tenant_id").(string)
	if !ok || tenantID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant tidak teridentifikasi")
	}

	// Parse query params
	page, _ := strconv.Atoi(c.Query("page", "1"))
	pageSize, _ := strconv.Atoi(c.Query("page_size", "25"))

	// Validasi page_size: hanya 10, 25, 50 yang diizinkan
	if pageSize != 10 && pageSize != 25 && pageSize != 50 {
		pageSize = 25
	}

	// Filter status opsional
	var status *domain.SyncStatus
	if s := c.Query("status"); s != "" {
		ss := domain.SyncStatus(s)
		status = &ss
	}

	result, err := h.isolirUsecase.GetPendingSyncs(c.Context(), tenantID, status, page, pageSize)
	if err != nil {
		h.logger.Error().Err(err).Msg("gagal mengambil daftar pending syncs")
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal mengambil daftar pending syncs")
	}

	return domain.SuccessResponse(c, fiber.StatusOK, result)
}

// Summary menangani GET /v1/isolir/summary.
// Mengembalikan ringkasan statistik isolir untuk dashboard.
func (h *IsolirHandler) Summary(c *fiber.Ctx) error {
	tenantID, ok := c.Locals("tenant_id").(string)
	if !ok || tenantID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant tidak teridentifikasi")
	}

	summary, err := h.isolirUsecase.GetDashboardSummary(c.Context(), tenantID)
	if err != nil {
		h.logger.Error().Err(err).Msg("gagal mengambil ringkasan isolir")
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal mengambil ringkasan isolir")
	}

	return domain.SuccessResponse(c, fiber.StatusOK, summary)
}

// WaivePenalty menangani POST /v1/invoices/:id/waive-penalty.
// Menghapus denda dari invoice dan menghitung ulang total.
func (h *IsolirHandler) WaivePenalty(c *fiber.Ctx) error {
	id := c.Params("id")
	if _, err := uuid.Parse(id); err != nil {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "invoice ID tidak valid")
	}

	actor := h.extractActor(c)

	if err := h.isolirUsecase.WaivePenalty(c.Context(), id, actor.ActorID, actor.ActorName); err != nil {
		return h.mapIsolirError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, fiber.Map{
		"message": "denda berhasil dihapus",
	})
}

// Reactivate menangani POST /v1/customers/:id/reactivate.
// Mengaktifkan kembali pelanggan yang di-suspend setelah semua invoice lunas.
func (h *IsolirHandler) Reactivate(c *fiber.Ctx) error {
	id := c.Params("id")
	if _, err := uuid.Parse(id); err != nil {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "customer ID tidak valid")
	}

	actor := h.extractActor(c)

	if err := h.isolirUsecase.ProcessReactivate(c.Context(), id, actor.ActorID, actor.ActorName); err != nil {
		return h.mapReactivateError(c, err, id)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, fiber.Map{
		"message": "pelanggan berhasil diaktifkan kembali",
	})
}

// extractActor mengambil informasi aktor dari Fiber locals (di-set oleh auth middleware).
func (h *IsolirHandler) extractActor(c *fiber.Ctx) domain.ActorInfo {
	actorID, _ := c.Locals("user_id").(string)
	actorName, _ := c.Locals("user_name").(string)
	return domain.ActorInfo{ActorID: actorID, ActorName: actorName}
}

// mapIsolirError memetakan domain error ke HTTP error response.
func (h *IsolirHandler) mapIsolirError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, domain.ErrNoPendingSync):
		return domain.ErrorResponse(c, fiber.StatusNotFound, "NO_PENDING_SYNC", err.Error())
	case errors.Is(err, domain.ErrInvoiceNotFound):
		return domain.ErrorResponse(c, fiber.StatusNotFound, "INVOICE_NOT_FOUND", err.Error())
	case errors.Is(err, domain.ErrNoPenaltyToWaive):
		return domain.ErrorResponse(c, fiber.StatusUnprocessableEntity, "NO_PENALTY_TO_WAIVE", err.Error())
	case errors.Is(err, domain.ErrInvoiceNotEditable):
		return domain.ErrorResponse(c, fiber.StatusUnprocessableEntity, "INVOICE_NOT_EDITABLE", err.Error())
	default:
		h.logger.Error().Err(err).Msg("internal error pada isolir handler")
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "terjadi kesalahan internal")
	}
}

// mapReactivateError memetakan error reactivate ke HTTP response.
// Untuk ErrOutstandingInvoicesExist, menyertakan jumlah dan total tagihan.
func (h *IsolirHandler) mapReactivateError(c *fiber.Ctx, err error, customerID string) error {
	switch {
	case errors.Is(err, domain.ErrCustomerNotFound):
		return domain.ErrorResponse(c, fiber.StatusNotFound, "CUSTOMER_NOT_FOUND", err.Error())
	case errors.Is(err, domain.ErrOutstandingInvoicesExist):
		// Ambil detail outstanding untuk respons yang informatif
		count, _ := h.isolirUsecase.CountOutstandingInvoices(c.Context(), customerID)
		total, _ := h.isolirUsecase.SumOutstandingAmount(c.Context(), customerID)
		return c.Status(fiber.StatusUnprocessableEntity).JSON(domain.APIResponse{
			Success: false,
			Error: &domain.APIError{
				Code:    "OUTSTANDING_INVOICES_EXIST",
				Message: err.Error(),
			},
			Data: fiber.Map{
				"outstanding_count": count,
				"outstanding_total": total,
			},
		})
	case errors.Is(err, domain.ErrInvalidStatusTransition):
		return domain.ErrorResponse(c, fiber.StatusUnprocessableEntity, "INVALID_STATUS_TRANSITION", err.Error())
	default:
		h.logger.Error().Err(err).Msg("internal error pada reactivate handler")
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "terjadi kesalahan internal")
	}
}
