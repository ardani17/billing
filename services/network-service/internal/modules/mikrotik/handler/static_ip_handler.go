package handler

import (
	"errors"
	"strconv"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"

	"github.com/ispboss/ispboss/pkg/tenant"
	"github.com/ispboss/ispboss/services/network-service/internal/domain"
	"github.com/ispboss/ispboss/services/network-service/internal/modules/mikrotik/usecase"
)

type StaticIPHandler struct {
	manager  usecase.StaticIPManager
	logger   zerolog.Logger
	validate *validator.Validate
}

func NewStaticIPHandler(manager usecase.StaticIPManager, logger zerolog.Logger) *StaticIPHandler {
	return &StaticIPHandler{manager: manager, logger: logger, validate: validator.New()}
}

func (h *StaticIPHandler) ListAssignments(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	pageSize, _ := strconv.Atoi(c.Query("page_size", "20"))
	result, err := h.manager.ListAssignments(c.UserContext(), c.Params("id"), domain.StaticIPAssignmentListParams{
		Page: page, PageSize: pageSize, Search: c.Query("search"),
	})
	if err != nil {
		return h.mapError(c, err)
	}
	return domain.PaginatedResponse(c, fiber.StatusOK, result.Data, result.Total, result.Page, result.PageSize, result.TotalPages)
}

func (h *StaticIPHandler) CreateAssignment(c *fiber.Ctx) error {
	tenantID := tenant.FromContext(c.UserContext())
	if tenantID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant tidak teridentifikasi")
	}
	var req domain.CreateStaticIPAssignmentRequest
	if err := c.BodyParser(&req); err != nil {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "request body tidak valid")
	}
	if err := h.validate.Struct(req); err != nil {
		return h.validationError(c, err)
	}
	ctx := usecase.WithMikroTikAuditActor(c.UserContext(), fiberLocalsString(c, "user_id"), c.IP())
	item, err := h.manager.CreateAssignment(ctx, tenantID, c.Params("id"), req)
	if err != nil {
		return h.mapError(c, err)
	}
	return domain.SuccessResponse(c, fiber.StatusCreated, item)
}

func (h *StaticIPHandler) UpdateAssignment(c *fiber.Ctx) error {
	var req domain.UpdateStaticIPAssignmentRequest
	if err := c.BodyParser(&req); err != nil {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "request body tidak valid")
	}
	ctx := usecase.WithMikroTikAuditActor(c.UserContext(), fiberLocalsString(c, "user_id"), c.IP())
	item, err := h.manager.UpdateAssignment(ctx, c.Params("id"), c.Params("assignment_id"), req)
	if err != nil {
		return h.mapError(c, err)
	}
	return domain.SuccessResponse(c, fiber.StatusOK, item)
}

func (h *StaticIPHandler) DeleteAssignment(c *fiber.Ctx) error {
	var req domain.DeleteStaticIPAssignmentRequest
	if err := c.BodyParser(&req); err != nil {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "request body tidak valid")
	}
	if err := h.validate.Struct(req); err != nil {
		return h.validationError(c, err)
	}
	ctx := usecase.WithMikroTikAuditActor(c.UserContext(), fiberLocalsString(c, "user_id"), c.IP())
	if err := h.manager.DeleteAssignment(ctx, c.Params("id"), c.Params("assignment_id"), req); err != nil {
		return h.mapError(c, err)
	}
	return domain.SuccessResponse(c, fiber.StatusOK, fiber.Map{"message": "static ip berhasil dihapus"})
}

func (h *StaticIPHandler) IsolateAssignment(c *fiber.Ctx) error {
	ctx := usecase.WithMikroTikAuditActor(c.UserContext(), fiberLocalsString(c, "user_id"), c.IP())
	item, err := h.manager.IsolateAssignment(ctx, c.Params("id"), c.Params("assignment_id"))
	if err != nil {
		return h.mapError(c, err)
	}
	return domain.SuccessResponse(c, fiber.StatusOK, item)
}

func (h *StaticIPHandler) UnisolateAssignment(c *fiber.Ctx) error {
	ctx := usecase.WithMikroTikAuditActor(c.UserContext(), fiberLocalsString(c, "user_id"), c.IP())
	item, err := h.manager.UnisolateAssignment(ctx, c.Params("id"), c.Params("assignment_id"))
	if err != nil {
		return h.mapError(c, err)
	}
	return domain.SuccessResponse(c, fiber.StatusOK, item)
}

func (h *StaticIPHandler) mapError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, domain.ErrStaticIPAssignmentNotFound):
		return domain.ErrorResponse(c, fiber.StatusNotFound, "STATIC_IP_NOT_FOUND", err.Error())
	case errors.Is(err, domain.ErrStaticIPAssignmentExists):
		return domain.ErrorResponse(c, fiber.StatusConflict, "STATIC_IP_EXISTS", err.Error())
	case errors.Is(err, domain.ErrInvalidIPAddress), errors.Is(err, domain.ErrConfirmationMismatch):
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", err.Error())
	case errors.Is(err, domain.ErrConnectionFailed):
		return domain.ErrorResponse(c, fiber.StatusBadGateway, "CONNECTION_FAILED", err.Error())
	case errors.Is(err, domain.ErrConnectionTimeout):
		return domain.ErrorResponse(c, fiber.StatusGatewayTimeout, "CONNECTION_TIMEOUT", err.Error())
	default:
		h.logger.Error().Err(err).Msg("unhandled error di static ip handler")
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "terjadi kesalahan internal")
	}
}

func (h *StaticIPHandler) validationError(c *fiber.Ctx, err error) error {
	var ve validator.ValidationErrors
	if errors.As(err, &ve) {
		fields := make([]domain.FieldError, 0, len(ve))
		for _, fe := range ve {
			fields = append(fields, domain.FieldError{Field: toSnakeCase(fe.Field()), Message: validationMessage(fe)})
		}
		return domain.ErrorResponse(c, fiber.StatusUnprocessableEntity, "VALIDATION_ERROR", "validasi gagal", fields...)
	}
	return domain.ErrorResponse(c, fiber.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
}
