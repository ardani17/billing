// map_node_handler_photo.go menangani HTTP permintaan untuk foto dan riwayat map node.
// Termasuk: list foto, upload foto, hapus foto, dan riwayat perubahan.
// Dipisah dari map_node_handler.go agar tidak melebihi 200 baris.
package handler

import (
	"strconv"

	"github.com/gofiber/fiber/v2"

	"github.com/ispboss/ispboss/pkg/tenant"
	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// ListPhotos menangani GET /nodes/:id/photos.
// Mengambil daftar foto aktif untuk satu node.
func (h *MapNodeHandler) ListPhotos(c *fiber.Ctx) error {
	nodeID := c.Params("id")
	if nodeID == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "node ID wajib diisi")
	}

	result, err := h.manager.ListPhotos(c.UserContext(), nodeID)
	if err != nil {
		return h.mapError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, result)
}

// UploadPhoto menangani POST /nodes/:id/photos.
// Menerima file foto via multipart form, validasi, dan simpan.
func (h *MapNodeHandler) UploadPhoto(c *fiber.Ctx) error {
	nodeID := c.Params("id")
	if nodeID == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "node ID wajib diisi")
	}

	tenantID := tenant.FromContext(c.UserContext())
	if tenantID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant tidak teridentifikasi")
	}

	// Ambil file dari multipart form
	fileHeader, err := c.FormFile("photo")
	if err != nil {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "file foto wajib diisi")
	}

	file, err := fileHeader.Open()
	if err != nil {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "gagal membaca file foto")
	}
	defer file.Close()

	caption := c.FormValue("caption")

	resp, err := h.manager.UploadPhoto(c.UserContext(), nodeID, file, fileHeader, caption, tenantID)
	if err != nil {
		return h.mapError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusCreated, resp)
}

// DeletePhoto menangani DELETE /nodes/:id/photos/:photo_id.
// Soft-hapus foto dan catat riwayat perubahan.
func (h *MapNodeHandler) DeletePhoto(c *fiber.Ctx) error {
	nodeID := c.Params("id")
	if nodeID == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "node ID wajib diisi")
	}

	photoID := c.Params("photo_id")
	if photoID == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "photo ID wajib diisi")
	}

	tenantID := tenant.FromContext(c.UserContext())
	if err := h.manager.DeletePhoto(c.UserContext(), nodeID, photoID, tenantID); err != nil {
		return h.mapError(c, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// GetHistory menangani GET /nodes/:id/history.
// Mengambil riwayat perubahan node dengan paginasi.
func (h *MapNodeHandler) GetHistory(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "node ID wajib diisi")
	}

	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))
	if limit < 1 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	result, err := h.manager.GetHistory(c.UserContext(), id, limit, offset)
	if err != nil {
		return h.mapError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, result)
}
