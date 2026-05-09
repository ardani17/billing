// map_search_handler.go menangani HTTP permintaan untuk pencarian map node.
// Validasi kueri minimal 2 karakter, kembalikan maksimal 20 hasil.
package handler

import (
	"github.com/gofiber/fiber/v2"

	"github.com/ispboss/ispboss/pkg/tenant"
	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// PencarianHandler menangani HTTP permintaan untuk pencarian di peta.
type SearchHandler struct {
	manager domain.MapNodeManager
}

// NewPencarianHandler membuat instance baru PencarianHandler.
func NewSearchHandler(manager domain.MapNodeManager) *SearchHandler {
	return &SearchHandler{manager: manager}
}

// Pencarian menangani GET /search.
// Validasi kueri minimal 2 karakter, kembalikan maksimal 20 hasil pencarian.
func (h *SearchHandler) Search(c *fiber.Ctx) error {
	tenantID := tenant.FromContext(c.UserContext())
	if tenantID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant tidak teridentifikasi")
	}

	query := c.Query("q")
	if len(query) < 2 {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "query pencarian minimal 2 karakter")
	}

	results, err := h.manager.Search(c.UserContext(), tenantID, query)
	if err != nil {
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "terjadi kesalahan internal")
	}

	// Batasi hasil maksimal 20
	if len(results) > 20 {
		results = results[:20]
	}

	return domain.SuccessResponse(c, fiber.StatusOK, results)
}
