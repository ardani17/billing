package middleware

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"
)

// RequestLogger membuat Fiber middleware yang mencatat setiap HTTP permintaan.
// Informasi yang dicatat: method, path, status code, dan durasi pemrosesan.
// Menggunakan zerolog untuk output terstruktur.
func RequestLogger(logger zerolog.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Catat waktu mulai pemrosesan permintaan
		start := time.Now()

		// Lanjutkan ke handler berikutnya
		err := c.Next()

		// Hitung durasi pemrosesan
		duration := time.Since(start)

		// Tentukan level log berdasarkan status code
		var event *zerolog.Event
		status := c.Response().StatusCode()

		switch {
		case status >= 500:
			event = logger.Error()
		case status >= 400:
			event = logger.Warn()
		default:
			event = logger.Info()
		}

		// Tulis log entry dengan informasi permintaan
		event.
			Str("method", c.Method()).
			Str("path", c.Path()).
			Int("status", status).
			Dur("duration", duration).
			Str("ip", c.IP()).
			Msg("request selesai")

		return err
	}
}
