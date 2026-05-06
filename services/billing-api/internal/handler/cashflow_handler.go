package handler

import (
	"encoding/csv"
	"strconv"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
	"github.com/ispboss/ispboss/services/billing-api/internal/usecase"
	"github.com/rs/zerolog"
)

type CashflowHandler struct {
	usecase  *usecase.CashflowUsecase
	validate *validator.Validate
	logger   zerolog.Logger
}

func NewCashflowHandler(usecase *usecase.CashflowUsecase, logger zerolog.Logger) *CashflowHandler {
	return &CashflowHandler{usecase: usecase, validate: validator.New(), logger: logger}
}

func (h *CashflowHandler) Summary(c *fiber.Ctx) error {
	tenantID, start, end, ok := cashflowParams(c)
	if !ok {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant tidak teridentifikasi")
	}
	data, err := h.usecase.Summary(c.Context(), tenantID, start, end)
	if err != nil {
		h.logger.Error().Err(err).Msg("gagal mengambil ringkasan cashflow")
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal mengambil ringkasan cashflow")
	}
	return domain.SuccessResponse(c, fiber.StatusOK, data)
}

func (h *CashflowHandler) Transactions(c *fiber.Ctx) error {
	tenantID, start, end, ok := cashflowParams(c)
	if !ok {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant tidak teridentifikasi")
	}
	data, err := h.usecase.Transactions(c.Context(), tenantID, start, end, c.Query("direction"), c.Query("source"), c.Query("category"), c.Query("search"))
	if err != nil {
		h.logger.Error().Err(err).Msg("gagal mengambil transaksi cashflow")
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal mengambil transaksi cashflow")
	}
	return domain.SuccessResponse(c, fiber.StatusOK, data)
}

func (h *CashflowHandler) Trend(c *fiber.Ctx) error {
	tenantID, start, end, ok := cashflowParams(c)
	if !ok {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant tidak teridentifikasi")
	}
	data, err := h.usecase.Trend(c.Context(), tenantID, start, end)
	if err != nil {
		h.logger.Error().Err(err).Msg("gagal mengambil trend cashflow")
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal mengambil trend cashflow")
	}
	return domain.SuccessResponse(c, fiber.StatusOK, data)
}

func (h *CashflowHandler) Export(c *fiber.Ctx) error {
	tenantID, start, end, ok := cashflowParams(c)
	if !ok {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant tidak teridentifikasi")
	}
	data, err := h.usecase.Transactions(c.Context(), tenantID, start, end, c.Query("direction"), c.Query("source"), c.Query("category"), c.Query("search"))
	if err != nil {
		h.logger.Error().Err(err).Msg("gagal export cashflow")
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal export cashflow")
	}
	c.Set("Content-Type", "text/csv")
	c.Set("Content-Disposition", "attachment; filename=cashflow.csv")
	writer := csv.NewWriter(c)
	_ = writer.Write([]string{"tanggal", "arah", "sumber", "kategori", "deskripsi", "nominal"})
	for _, tx := range data {
		_ = writer.Write([]string{
			tx.Date.Format("2006-01-02"),
			tx.Direction,
			tx.Source,
			tx.Category,
			tx.Description,
			strconv.FormatInt(tx.Amount, 10),
		})
	}
	writer.Flush()
	return nil
}

func (h *CashflowHandler) CreateManual(c *fiber.Ctx) error {
	tenantID, ok := tenantIDFromCtx(c)
	if !ok {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant tidak teridentifikasi")
	}
	var req domain.CreateManualCashflowRequest
	if err := parseAndValidate(c, h.validate, &req); err != nil {
		return err
	}
	if actorFromCtx(c).ActorID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "aktor tidak teridentifikasi")
	}
	data, err := h.usecase.CreateManualTransaction(c.Context(), tenantID, req, actorFromCtx(c))
	if err != nil {
		h.logger.Error().Err(err).Msg("gagal mencatat kas manual")
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "CASHFLOW_MANUAL_FAILED", err.Error())
	}
	return domain.SuccessResponse(c, fiber.StatusCreated, data)
}

func cashflowParams(c *fiber.Ctx) (string, time.Time, time.Time, bool) {
	tenantID, ok := tenantIDFromCtx(c)
	if !ok {
		return "", time.Time{}, time.Time{}, false
	}
	now := time.Now()
	startText := c.Query("period_start", time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).Format("2006-01-02"))
	endText := c.Query("period_end", now.Format("2006-01-02"))
	start, err := time.Parse("2006-01-02", startText)
	if err != nil {
		start = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local)
	}
	end, err := time.Parse("2006-01-02", endText)
	if err != nil {
		end = now
	}
	return tenantID, start, end, true
}
