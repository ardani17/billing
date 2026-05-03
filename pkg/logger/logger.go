// Package logger menyediakan factory untuk membuat zerolog logger
// yang sudah dikonfigurasi dengan output JSON, timestamp, dan nama service.
package logger

import (
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

// Config berisi konfigurasi untuk membuat logger baru.
type Config struct {
	// Level menentukan level log minimum: debug, info, warn, error, fatal.
	// Jika tidak valid, default ke info.
	Level string

	// ServiceName adalah nama service yang akan ditambahkan ke setiap log entry.
	ServiceName string

	// Pretty mengaktifkan ConsoleWriter untuk output yang mudah dibaca saat development.
	Pretty bool
}

// New membuat zerolog.Logger baru dengan konfigurasi yang diberikan.
// Output dalam format JSON dengan timestamp dan nama service.
// Jika Pretty=true, menggunakan ConsoleWriter untuk tampilan development.
func New(cfg Config) zerolog.Logger {
	level := parseLevel(cfg.Level)
	zerolog.SetGlobalLevel(level)

	// Gunakan format timestamp RFC3339 untuk konsistensi
	zerolog.TimeFieldFormat = time.RFC3339

	var logger zerolog.Logger

	if cfg.Pretty {
		// Mode development: output berwarna dan mudah dibaca
		writer := zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: time.RFC3339,
		}
		logger = zerolog.New(writer).
			With().
			Timestamp().
			Str("service", cfg.ServiceName).
			Logger().
			Level(level)
	} else {
		// Mode production: output JSON terstruktur
		logger = zerolog.New(os.Stdout).
			With().
			Timestamp().
			Str("service", cfg.ServiceName).
			Logger().
			Level(level)
	}

	return logger
}

// NewDefault membuat logger dengan konfigurasi default.
// Menggunakan level info dan output JSON (tanpa pretty mode).
func NewDefault(serviceName string) zerolog.Logger {
	return New(Config{
		Level:       "info",
		ServiceName: serviceName,
		Pretty:      false,
	})
}

// parseLevel mengkonversi string level ke zerolog.Level.
// Jika string tidak valid, mengembalikan zerolog.InfoLevel sebagai default.
func parseLevel(level string) zerolog.Level {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		return zerolog.DebugLevel
	case "info":
		return zerolog.InfoLevel
	case "warn":
		return zerolog.WarnLevel
	case "error":
		return zerolog.ErrorLevel
	case "fatal":
		return zerolog.FatalLevel
	default:
		// Default ke info jika level tidak dikenali
		return zerolog.InfoLevel
	}
}
