// vpn_handler_ops.go menangani HTTP request untuk operasi VPN tunnel.
// Termasuk: test connection, auto-configure, script generation, bandwidth, summary, maintenance.
package handler

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// TestConnection menangani POST /api/v1/mikrotik/vpn/tunnels/:id/test.
// Menguji koneksi VPN dengan ping ke client VPN IP.
func (h *VPNHandler) TestConnection(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "tunnel ID wajib diisi")
	}

	result, err := h.manager.TestConnection(c.UserContext(), id)
	if err != nil {
		return h.mapError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, result)
}

// AutoConfigure menangani POST /api/v1/mikrotik/vpn/tunnels/:id/auto-configure.
// Mengkonfigurasi VPN di router yang sudah online via RouterOS API.
func (h *VPNHandler) AutoConfigure(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "tunnel ID wajib diisi")
	}

	if err := h.manager.AutoConfigure(c.UserContext(), id); err != nil {
		return h.mapError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, fiber.Map{
		"message": "auto-configure vpn berhasil dikirim",
	})
}

// GenerateScript menangani GET /api/v1/mikrotik/vpn/tunnels/:id/script.
// Menghasilkan RouterOS script (.rsc) dan return sebagai text/plain dengan Content-Disposition.
func (h *VPNHandler) GenerateScript(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "tunnel ID wajib diisi")
	}

	script, err := h.manager.GenerateScript(c.UserContext(), id)
	if err != nil {
		return h.mapError(c, err)
	}

	c.Set("Content-Type", "text/plain; charset=utf-8")
	c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"vpn-tunnel-%s.rsc\"", id))
	return c.SendString(script)
}

// GetBandwidth menangani GET /api/v1/mikrotik/vpn/tunnels/:id/bandwidth.
// Parse from/to query params dan return statistik bandwidth.
func (h *VPNHandler) GetBandwidth(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "tunnel ID wajib diisi")
	}

	// Parse from/to query params, default 24 jam terakhir
	now := time.Now()
	from := now.Add(-24 * time.Hour)
	to := now

	if fromStr := c.Query("from"); fromStr != "" {
		parsed, err := time.Parse(time.RFC3339, fromStr)
		if err != nil {
			return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "format 'from' tidak valid, gunakan RFC3339")
		}
		from = parsed
	}

	if toStr := c.Query("to"); toStr != "" {
		parsed, err := time.Parse(time.RFC3339, toStr)
		if err != nil {
			return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "format 'to' tidak valid, gunakan RFC3339")
		}
		to = parsed
	}

	result, err := h.manager.GetBandwidth(c.UserContext(), id, from, to)
	if err != nil {
		return h.mapError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, result)
}

// GetSummary menangani GET /api/v1/mikrotik/vpn/summary.
// Mengambil ringkasan status tunnel untuk dashboard.
func (h *VPNHandler) GetSummary(c *fiber.Ctx) error {
	summary, err := h.manager.GetSummary(c.UserContext())
	if err != nil {
		return h.mapError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, summary)
}

// ScheduleMaintenance menangani POST /admin/vpn/maintenance.
// Stub — akan diimplementasikan di task selanjutnya.
func (h *VPNHandler) ScheduleMaintenance(c *fiber.Ctx) error {
	return domain.SuccessResponse(c, fiber.StatusOK, fiber.Map{
		"message": "fitur maintenance scheduling belum diimplementasikan",
	})
}

// GetUpcomingMaintenance menangani GET /vpn/maintenance.
// Stub — akan diimplementasikan di task selanjutnya.
func (h *VPNHandler) GetUpcomingMaintenance(c *fiber.Ctx) error {
	return domain.SuccessResponse(c, fiber.StatusOK, fiber.Map{
		"upcoming": []interface{}{},
	})
}
