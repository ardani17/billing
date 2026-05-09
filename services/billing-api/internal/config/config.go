// Paket config menyediakan konfigurasi aplikasi billing-api.
// Memuat konfigurasi dari environment variables dan file .env menggunakan Viper.
package config

import (
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// AppConfig berisi semua konfigurasi yang dibutuhkan service billing-api.
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

	// Google OAuth
	GoogleClientID string `mapstructure:"GOOGLE_CLIENT_ID"`

	// Auth settings
	JWTRefreshExpiry time.Duration `mapstructure:"JWT_REFRESH_EXPIRY"`
	BcryptCost       int           `mapstructure:"BCRYPT_COST"`

	// Rate limiting
	LoginMaxAttempts      int           `mapstructure:"LOGIN_MAX_ATTEMPTS"`
	LoginLockDuration     time.Duration `mapstructure:"LOGIN_LOCK_DURATION"`
	GlobalRateLimitMax    int           `mapstructure:"GLOBAL_RATE_LIMIT_MAX"`
	GlobalRateLimitWindow time.Duration `mapstructure:"GLOBAL_RATE_LIMIT_WINDOW"`

	// HTTP security
	CORSAllowOrigins string `mapstructure:"CORS_ALLOW_ORIGINS"`

	// Gateway pembayaran
	GatewayMasterKey        string `mapstructure:"GATEWAY_MASTER_KEY"`
	XenditWebhookIPs        string `mapstructure:"XENDIT_WEBHOOK_IPS"`
	MidtransWebhookIPs      string `mapstructure:"MIDTRANS_WEBHOOK_IPS"`
	WebhookLogRetentionDays int    `mapstructure:"WEBHOOK_LOG_RETENTION_DAYS"`

	// Network service URL untuk cross-service calls (laporan jaringan)
	NetworkServiceURL string `mapstructure:"NETWORK_SERVICE_URL"`

	// IsolirAutomationEnabled mengaktifkan cron auto-isolir/suspend harian.
	// Bawaan false untuk mencegah perubahan jaringan otomatis saat integrasi lokal.
	IsolirAutomationEnabled bool `mapstructure:"ISOLIR_AUTOMATION_ENABLED"`

	// IsolirPeriodicSyncEnabled mengaktifkan retry sync jaringan periodik.
	// Bawaan false agar MikroTik tidak menerima dial API berkala.
	IsolirPeriodicSyncEnabled bool `mapstructure:"ISOLIR_PERIODIC_SYNC_ENABLED"`
}

// Muat memuat konfigurasi dari environment variables dan file .env.
// Mengatur nilai bawaan untuk variabel opsional.
func Load() (*AppConfig, error) {
	v := viper.New()

	// Baca file .env jika ada (opsional, tidak error jika tidak ditemukan)
	v.SetConfigFile(".env")
	v.SetConfigType("env")
	_ = v.ReadInConfig()

	// Aktifkan pembacaan dari environment variables
	v.AutomaticEnv()

	// Atur nilai bawaan untuk variabel opsional
	v.SetDefault("APP_ENV", "development")
	v.SetDefault("APP_PORT", 3001)
	v.SetDefault("LOG_LEVEL", "info")
	v.SetDefault("DB_PORT", 5432)
	v.SetDefault("DB_SSL_MODE", "disable")
	v.SetDefault("REDIS_PORT", 6379)
	v.SetDefault("REDIS_PASSWORD", "")
	v.SetDefault("JWT_EXPIRY", "24h")
	v.SetDefault("JWT_REFRESH_EXPIRY", "720h")
	v.SetDefault("BCRYPT_COST", 10)
	v.SetDefault("LOGIN_MAX_ATTEMPTS", 5)
	v.SetDefault("LOGIN_LOCK_DURATION", "15m")
	v.SetDefault("GLOBAL_RATE_LIMIT_MAX", 300)
	v.SetDefault("GLOBAL_RATE_LIMIT_WINDOW", "1m")
	v.SetDefault("CORS_ALLOW_ORIGINS", "http://localhost:3000,http://127.0.0.1:3000")
	v.SetDefault("WEBHOOK_LOG_RETENTION_DAYS", 90)
	v.SetDefault("NETWORK_SERVICE_URL", "http://localhost:3002")
	v.SetDefault("ISOLIR_AUTOMATION_ENABLED", false)
	v.SetDefault("ISOLIR_PERIODIC_SYNC_ENABLED", false)

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
		"GOOGLE_CLIENT_ID",
		"JWT_REFRESH_EXPIRY",
		"BCRYPT_COST",
		"LOGIN_MAX_ATTEMPTS",
		"LOGIN_LOCK_DURATION",
		"GLOBAL_RATE_LIMIT_MAX",
		"GLOBAL_RATE_LIMIT_WINDOW",
		"CORS_ALLOW_ORIGINS",
		"GATEWAY_MASTER_KEY",
		"XENDIT_WEBHOOK_IPS",
		"MIDTRANS_WEBHOOK_IPS",
		"WEBHOOK_LOG_RETENTION_DAYS",
		"NETWORK_SERVICE_URL",
		"ISOLIR_AUTOMATION_ENABLED",
		"ISOLIR_PERIODIC_SYNC_ENABLED",
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

// Validasi memeriksa bahwa semua variabel wajib sudah diisi.
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

	if isProductionEnv(c.AppEnv) {
		var unsafe []string
		if strings.TrimSpace(c.JWTSecret) == developmentJWTSecret || len(strings.TrimSpace(c.JWTSecret)) < 32 {
			unsafe = append(unsafe, "JWT_SECRET harus secret production minimal 32 karakter")
		}
		if strings.TrimSpace(c.DBPassword) == developmentDBPassword {
			unsafe = append(unsafe, "DB_PASSWORD tidak boleh memakai password development")
		}
		if strings.EqualFold(strings.TrimSpace(c.DBSSLMode), "disable") {
			unsafe = append(unsafe, "DB_SSL_MODE production tidak boleh disable")
		}
		if strings.TrimSpace(c.CORSAllowOrigins) == "" || strings.Contains(c.CORSAllowOrigins, "*") {
			unsafe = append(unsafe, "CORS_ALLOW_ORIGINS production harus allowlist domain eksplisit")
		}
		if len(unsafe) > 0 {
			return fmt.Errorf("konfigurasi production tidak aman: %s", strings.Join(unsafe, "; "))
		}
	}

	// Validasi format GATEWAY_MASTER_KEY jika diisi (opsional untuk dev)
	if c.GatewayMasterKey != "" {
		if !isValidHex64(c.GatewayMasterKey) {
			return fmt.Errorf(
				"GATEWAY_MASTER_KEY harus berupa 64 karakter hex (32 bytes)",
			)
		}
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

// ParseWebhookIPs memecah string IP yang dipisahkan koma menjadi slice.
// Mengembalikan slice kosong jika string kosong (whitelist dinonaktifkan).
func (c *AppConfig) ParseWebhookIPs() (xenditIPs, midtransIPs []string) {
	xenditIPs = splitAndTrim(c.XenditWebhookIPs)
	midtransIPs = splitAndTrim(c.MidtransWebhookIPs)
	return xenditIPs, midtransIPs
}

// MasterKeyBytes mendekode hex string GATEWAY_MASTER_KEY menjadi 32 bytes.
// Mengembalikan error jika key bukan hex valid atau bukan 32 bytes.
func (c *AppConfig) MasterKeyBytes() ([]byte, error) {
	if c.GatewayMasterKey == "" {
		return nil, fmt.Errorf("GATEWAY_MASTER_KEY belum diisi")
	}
	key, err := hex.DecodeString(c.GatewayMasterKey)
	if err != nil {
		return nil, fmt.Errorf("GATEWAY_MASTER_KEY bukan hex valid: %w", err)
	}
	if len(key) != 32 {
		return nil, fmt.Errorf(
			"GATEWAY_MASTER_KEY harus 32 bytes, ditemukan %d bytes", len(key),
		)
	}
	return key, nil
}

// hexPattern mencocokkan tepat 64 karakter hex (0-9, a-f, A-F).
var hexPattern = regexp.MustCompile(`^[0-9a-fA-F]{64}$`)

const developmentJWTSecret = "change-me-to-a-strong-secret"
const developmentDBPassword = "ispboss_secret"

func isProductionEnv(env string) bool {
	return strings.EqualFold(strings.TrimSpace(env), "production")
}

// isValidHex64 memeriksa apakah string berupa 64 karakter hex.
func isValidHex64(s string) bool {
	return hexPattern.MatchString(s)
}

// splitAndTrim memecah string berdasarkan koma dan menghapus spasi.
// Mengembalikan slice kosong jika input kosong.
func splitAndTrim(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
