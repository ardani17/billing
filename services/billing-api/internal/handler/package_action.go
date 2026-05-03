// package_action.go menangani HTTP request untuk aksi paket.
// Termasuk: activate, deactivate, dan duplicate.
package handler

import (
	"github.com/gofiber/fiber/v2"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

// Activate menangani POST /v1/packages/:id/activate.
// Mengaktifkan paket yang sedang nonaktif.
func (h *PackageHandler) Activate(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "package ID wajib diisi")
	}

	actor := h.extractActor(c)

	pkg, err := h.packageUsecase.Activate(c.Context(), id, actor)
	if err != nil {
		return h.mapPackageError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, pkg)
}

// Deactivate menangani POST /v1/packages/:id/deactivate.
// Menonaktifkan paket yang sedang aktif.
func (h *PackageHandler) Deactivate(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "package ID wajib diisi")
	}

	actor := h.extractActor(c)

	pkg, err := h.packageUsecase.Deactivate(c.Context(), id, actor)
	if err != nil {
		return h.mapPackageError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, pkg)
}

// Duplicate menangani POST /v1/packages/:id/duplicate.
// Menduplikasi paket yang sudah ada dengan nama unik.
func (h *PackageHandler) Duplicate(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "package ID wajib diisi")
	}

	actor := h.extractActor(c)

	pkg, err := h.packageUsecase.Duplicate(c.Context(), id, actor)
	if err != nil {
		return h.mapPackageError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusCreated, pkg)
}
