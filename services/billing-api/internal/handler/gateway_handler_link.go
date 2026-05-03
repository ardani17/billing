// gateway_handler_link.go menangani endpoint payment link, webhook query, dan walled garden.
// Termasuk: get customer payment link, regenerate, invoice payment links,
// payment link webhooks, dan walled garden payment info.
package handler

import (
	"github.com/gofiber/fiber/v2"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

// GetCustomerPaymentLink menangani GET /v1/customers/:id/payment-link.
// Mengembalikan payment link aktif untuk customer beserta detail invoice.
func (h *GatewayHandler) GetCustomerPaymentLink(c *fiber.Ctx) error {
	customerID := c.Params("id")
	if customerID == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "customer ID wajib diisi")
	}

	result, err := h.gatewayUsecase.GetCustomerPaymentLink(c.Context(), customerID)
	if err != nil {
		return h.mapGatewayError(c, err)
	}

	// Jika tidak ada payment link aktif, kembalikan null
	if result == nil {
		return domain.SuccessResponse(c, fiber.StatusOK, nil)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, result)
}

// RegeneratePaymentLink menangani POST /v1/customers/:id/payment-link/regenerate.
// Meng-expire link aktif lama dan membuat link baru dengan jumlah terbaru.
func (h *GatewayHandler) RegeneratePaymentLink(c *fiber.Ctx) error {
	customerID := c.Params("id")
	if customerID == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "customer ID wajib diisi")
	}

	link, err := h.gatewayUsecase.RegeneratePaymentLink(c.Context(), customerID)
	if err != nil {
		return h.mapGatewayError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, link)
}

// GetInvoicePaymentLinks menangani GET /v1/invoices/:id/payment-links.
// Mengembalikan semua payment links (active, expired, paid) untuk invoice tertentu.
func (h *GatewayHandler) GetInvoicePaymentLinks(c *fiber.Ctx) error {
	invoiceID := c.Params("id")
	if invoiceID == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "invoice ID wajib diisi")
	}

	links, err := h.gatewayUsecase.GetInvoicePaymentLinks(c.Context(), invoiceID)
	if err != nil {
		return h.mapGatewayError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, links)
}

// GetPaymentLinkWebhooks menangani GET /v1/payment-links/:id/webhooks.
// Mengembalikan semua webhook logs untuk payment link tertentu.
func (h *GatewayHandler) GetPaymentLinkWebhooks(c *fiber.Ctx) error {
	linkID := c.Params("id")
	if linkID == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "payment link ID wajib diisi")
	}

	// Ambil payment link untuk mendapatkan external_id
	link, err := h.linkRepo.GetByID(c.Context(), linkID)
	if err != nil || link == nil {
		return domain.ErrorResponse(c, fiber.StatusNotFound, "PAYMENT_LINK_NOT_FOUND", "payment link tidak ditemukan")
	}

	// Ambil webhook logs berdasarkan external_id payment link
	webhooks, err := h.webhookRepo.ListByPaymentLink(c.Context(), link.ExternalID)
	if err != nil {
		h.logger.Error().Err(err).Str("link_id", linkID).Msg("gagal mengambil webhook logs")
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal mengambil webhook logs")
	}

	return domain.SuccessResponse(c, fiber.StatusOK, webhooks)
}

// WalledGardenPaymentInfo menangani GET /v1/public/walled-garden/:customer_id/payment-info.
// Endpoint publik (tanpa auth) untuk halaman captive portal pelanggan yang diisolir.
// Mengembalikan URL pembayaran, total tunggakan, dan detail invoice.
func (h *GatewayHandler) WalledGardenPaymentInfo(c *fiber.Ctx) error {
	customerID := c.Params("customer_id")
	if customerID == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "customer ID wajib diisi")
	}

	info, err := h.gatewayUsecase.GetWalledGardenPaymentInfo(c.Context(), customerID)
	if err != nil {
		return h.mapGatewayError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, info)
}
