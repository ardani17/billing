// Package config menyediakan konfigurasi aplikasi notification.
// Memuat konfigurasi dari environment variables dan file .env menggunakan Viper.
package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// AppConfig berisi semua konfigurasi yang dibutuhkan service notification.
type AppConfig struct {
	AppName  string `mapstructure:"APP_NAME"`
	AppPort  int    `mapstructure:"APP_PORT"`
	AppEnv   string `mapstructure:"APP_ENV"`
	LogLevel string `mapstructure:"LOG_LEVEL"`

	DBHost     string `mapstructure:"DB_HOST"`
	DBPort     int    `mapstructure:"DB_PORT"`
	DBUser     string `mapstructure:"DB_USER"`
	DBPassword string `mapstructure:"DB_PASSWORD"`
	DBName     string `mapstructure:"DB_NAME"`
	DBSSLMode  string `mapstructure:"DB_SSL_MODE"`

	RedisHost     string `mapstructure:"REDIS_HOST"`
	RedisPort     int    `mapstructure:"REDIS_PORT"`
	RedisPassword string `mapstructure:"REDIS_PASSWORD"`

	JWTSecret string        `mapstructure:"JWT_SECRET"`
	JWTExpiry time.Duration `mapstructure:"JWT_EXPIRY"`

	// Worker configuration
	WorkerConcurrency int `mapstructure:"WORKER_CONCURRENCY"`

	// Provider timeout configuration
	FonnteTimeout  time.Duration `mapstructure:"FONNTE_TIMEOUT"`
	ZenzivaTimeout time.Duration `mapstructure:"ZENZIVA_TIMEOUT"`
	SMTPTimeout    time.Duration `mapstructure:"SMTP_TIMEOUT"`
}

// Load memuat konfigurasi dari environment variables dan file .env.
// Mengatur nilai default untuk variabel opsional.
func Load() (*AppConfig, error) {
	v := viper.New()

	// Baca file .env jika ada (opsional, tidak error jika tidak ditemukan)
	v.SetConfigFile(".env")
	v.SetConfigType("env")
	_ = v.ReadInConfig()

	// Aktifkan pembacaan dari environment variables
	v.AutomaticEnv()

	// Atur nilai default untuk variabel opsional
	v.SetDefault("APP_NAME", "notification")
	v.SetDefault("APP_ENV", "development")
	v.SetDefault("APP_PORT", 3003)
	v.SetDefault("LOG_LEVEL", "info")
	v.SetDefault("DB_PORT", 5432)
	v.SetDefault("DB_SSL_MODE", "disable")
	v.SetDefault("REDIS_PORT", 6379)
	v.SetDefault("REDIS_PASSWORD", "")
	v.SetDefault("JWT_EXPIRY", "24h")
	v.SetDefault("WORKER_CONCURRENCY", 10)
	v.SetDefault("FONNTE_TIMEOUT", "30s")
	v.SetDefault("ZENZIVA_TIMEOUT", "30s")
	v.SetDefault("SMTP_TIMEOUT", "30s")

	for _, key := range []string{
		"APP_NAME",
		"APP_PORT",
		"APP_ENV",
		"LOG_LEVEL",
		"DB_HOST",
		"DB_PORT",
		"DB_USER",
		"DB_PASSWORD",
		"DB_NAME",
		"DB_SSL_MODE",
		"REDIS_HOST",
		"REDIS_PORT",
		"REDIS_PASSWORD",
		"JWT_SECRET",
		"JWT_EXPIRY",
		"WORKER_CONCURRENCY",
		"FONNTE_TIMEOUT",
		"ZENZIVA_TIMEOUT",
		"SMTP_TIMEOUT",
	} {
		_ = v.BindEnv(key)
	}

	var cfg AppConfig
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("gagal unmarshal konfigurasi: %w", err)
	}

	// Validasi variabel wajib
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// Validate memeriksa bahwa semua variabel wajib sudah diisi.
// Mengembalikan error dengan daftar variabel yang hilang.
func (c *AppConfig) Validate() error {
	var missing []string

	if c.AppName == "" {
		missing = append(missing, "APP_NAME")
	}
	if c.DBHost == "" {
		missing = append(missing, "DB_HOST")
	}
	if c.DBUser == "" {
		missing = append(missing, "DB_USER")
	}
	if c.DBPassword == "" {
		missing = append(missing, "DB_PASSWORD")
	}
	if c.DBName == "" {
		missing = append(missing, "DB_NAME")
	}
	if c.RedisHost == "" {
		missing = append(missing, "REDIS_HOST")
	}
	if c.JWTSecret == "" {
		missing = append(missing, "JWT_SECRET")
	}

	if len(missing) > 0 {
		return fmt.Errorf(
			"variabel wajib belum diisi: %s",
			strings.Join(missing, ", "),
		)
	}

	return nil
}

// DSN mengembalikan PostgreSQL connection string dalam format URL.
func (c *AppConfig) DSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		c.DBUser, c.DBPassword, c.DBHost, c.DBPort, c.DBName, c.DBSSLMode,
	)
}

// QueuePriorities mengembalikan prioritas queue untuk asynq worker.
// critical: 6, default: 3, low: 1.
func (c *AppConfig) QueuePriorities() map[string]int {
	return map[string]int{
		"critical": 6,
		"default":  3,
		"low":      1,
	}
}
