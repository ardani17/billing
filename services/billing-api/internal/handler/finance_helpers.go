package handler

import (
	"errors"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

func tenantIDFromCtx(c *fiber.Ctx) (string, bool) {
	tenantID, ok := c.Locals("tenant_id").(string)
	return tenantID, ok && tenantID != ""
}

func actorFromCtx(c *fiber.Ctx) domain.ActorInfo {
	actorID, _ := c.Locals("user_id").(string)
	actorName, _ := c.Locals("user_name").(string)
	return domain.ActorInfo{ActorID: actorID, ActorName: actorName}
}

func parseAndValidate(c *fiber.Ctx, validate *validator.Validate, req interface{}) error {
	if err := c.BodyParser(req); err != nil {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "request body tidak valid")
	}
	if err := validate.Struct(req); err != nil {
		var ve validator.ValidationErrors
		if errors.As(err, &ve) {
			return domain.ErrorResponse(c, fiber.StatusBadRequest, "VALIDATION_ERROR", "validasi gagal", mapValidationErrors(ve)...)
		}
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "VALIDATION_ERROR", err.Error())
	}
	return nil
}
