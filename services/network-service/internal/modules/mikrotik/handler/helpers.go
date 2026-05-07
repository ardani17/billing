package handler

import (
	"strings"

	"github.com/gofiber/fiber/v2"
)

func fiberLocalsString(c *fiber.Ctx, key string) string {
	value, _ := c.Locals(key).(string)
	return value
}

func canUseMikroTikTerminal(c *fiber.Ctx) bool {
	switch strings.ToLower(fiberLocalsString(c, "role")) {
	case "tenant_admin", "admin", "network_admin", "super_admin", "owner":
		return true
	default:
		return false
	}
}
