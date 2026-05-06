// Package config menyediakan konfigurasi aplikasi network-service.
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

// AppConfig berisi semua konfigurasi yang dibutuhkan service network-service.
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

	// NetworkMode menentukan mode operasi jaringan: "mock" atau "live"
	NetworkMode string `mapstructure:"NETWORK_MODE"`

	// EncryptionKey adalah master key untuk enkripsi credential router (64 hex chars = 32 bytes)
	EncryptionKey string `mapstructure:"ENCRYPTION_KEY"`

	// SyncIntervalMinutes adalah interval periodic sync PPPoE user (default 15 menit)
	SyncIntervalMinutes int `mapstructure:"SYNC_INTERVAL_MINUTES"`

	// RouterHealthCheckEnabled mengaktifkan health check periodik ke RouterOS API.
	// Default false agar router tidak menerima login API berulang saat idle.
	RouterHealthCheckEnabled bool `mapstructure:"ROUTER_HEALTH_CHECK_ENABLED"`

	// PPPoESyncSchedulerEnabled mengaktifkan sync PPPoE periodik ke RouterOS API.
	// Default false; sync tetap bisa dipicu manual atau dari event operasional.
	PPPoESyncSchedulerEnabled bool `mapstructure:"PPPOE_SYNC_SCHEDULER_ENABLED"`

	// DefaultIsolirMethod adalah metode isolir default: "firewall_nat_redirect" atau "dns_redirect"
	DefaultIsolirMethod string `mapstructure:"DEFAULT_ISOLIR_METHOD"`

	// WalledGardenIP adalah IP address walled garden untuk redirect pelanggan terisolir
	WalledGardenIP string `mapstructure:"WALLED_GARDEN_IP"`

	// DNSServerIP adalah IP address DNS server ISPBoss untuk DNS redirect isolir
	DNSServerIP string `mapstructure:"DNS_SERVER_IP"`

	// VPN Server Configuration
	VPNServerEndpoint           string `mapstructure:"VPN_SERVER_ENDPOINT"`
	VPNSecondaryEndpoint        string `mapstructure:"VPN_SECONDARY_ENDPOINT"`
	VPNServerPublicKey          string `mapstructure:"VPN_SERVER_PUBLIC_KEY"`
	VPNSecondaryServerPublicKey string `mapstructure:"VPN_SECONDARY_SERVER_PUBLIC_KEY"`
	VPNListenPort               int    `mapstructure:"VPN_LISTEN_PORT"`
	VPNHealthCheckInterval      int    `mapstructure:"VPN_HEALTH_CHECK_INTERVAL"`
	VPNBandwidthCollectInterval int    `mapstructure:"VPN_BANDWIDTH_COLLECT_INTERVAL"`
	VPNServerCapacityMbps       int64  `mapstructure:"VPN_SERVER_CAPACITY_MBPS"`

	// OLT Configuration
	// OLTHealthCheckInterval adalah interval health check OLT dalam detik (default 300 = 5 menit)
	OLTHealthCheckInterval int `mapstructure:"OLT_HEALTH_CHECK_INTERVAL"`
	// OLTSyncInterval adalah interval periodic sync OLT dalam detik (default 1800 = 30 menit)
	OLTSyncInterval int `mapstructure:"OLT_SYNC_INTERVAL"`
	// SNMPTrapPort adalah port untuk SNMP trap receiver (default 162)
	SNMPTrapPort int `mapstructure:"SNMP_TRAP_PORT"`
	// MaxONTPerPort adalah jumlah maksimum ONT per PON port untuk capacity planning (default 64)
	MaxONTPerPort int `mapstructure:"MAX_ONT_PER_PORT"`
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
	v.SetDefault("APP_NAME", "network-service")
	v.SetDefault("APP_ENV", "development")
	v.SetDefault("APP_PORT", 3002)
	v.SetDefault("LOG_LEVEL", "info")
	v.SetDefault("DB_PORT", 5432)
	v.SetDefault("DB_SSL_MODE", "disable")
	v.SetDefault("REDIS_PORT", 6379)
	v.SetDefault("REDIS_PASSWORD", "")
	v.SetDefault("JWT_EXPIRY", "24h")
	v.SetDefault("NETWORK_MODE", "mock")
	v.SetDefault("SYNC_INTERVAL_MINUTES", 15)
	v.SetDefault("ROUTER_HEALTH_CHECK_ENABLED", false)
	v.SetDefault("PPPOE_SYNC_SCHEDULER_ENABLED", false)
	v.SetDefault("DEFAULT_ISOLIR_METHOD", "firewall_nat_redirect")
	v.SetDefault("WALLED_GARDEN_IP", "")
	v.SetDefault("DNS_SERVER_IP", "")
	v.SetDefault("VPN_SERVER_ENDPOINT", "vpn1.ispboss.id")
	v.SetDefault("VPN_SECONDARY_ENDPOINT", "vpn2.ispboss.id")
	v.SetDefault("VPN_SERVER_PUBLIC_KEY", "")
	v.SetDefault("VPN_SECONDARY_SERVER_PUBLIC_KEY", "")
	v.SetDefault("VPN_LISTEN_PORT", 51820)
	v.SetDefault("VPN_HEALTH_CHECK_INTERVAL", 30)
	v.SetDefault("VPN_BANDWIDTH_COLLECT_INTERVAL", 30)
	v.SetDefault("VPN_SERVER_CAPACITY_MBPS", 1000)
	v.SetDefault("OLT_HEALTH_CHECK_INTERVAL", 300)
	v.SetDefault("OLT_SYNC_INTERVAL", 1800)
	v.SetDefault("SNMP_TRAP_PORT", 162)
	v.SetDefault("MAX_ONT_PER_PORT", 64)

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
		"NETWORK_MODE",
		"ENCRYPTION_KEY",
		"SYNC_INTERVAL_MINUTES",
		"ROUTER_HEALTH_CHECK_ENABLED",
		"PPPOE_SYNC_SCHEDULER_ENABLED",
		"DEFAULT_ISOLIR_METHOD",
		"WALLED_GARDEN_IP",
		"DNS_SERVER_IP",
		"VPN_SERVER_ENDPOINT",
		"VPN_SECONDARY_ENDPOINT",
		"VPN_SERVER_PUBLIC_KEY",
		"VPN_SECONDARY_SERVER_PUBLIC_KEY",
		"VPN_LISTEN_PORT",
		"VPN_HEALTH_CHECK_INTERVAL",
		"VPN_BANDWIDTH_COLLECT_INTERVAL",
		"VPN_SERVER_CAPACITY_MBPS",
		"OLT_HEALTH_CHECK_INTERVAL",
		"OLT_SYNC_INTERVAL",
		"SNMP_TRAP_PORT",
		"MAX_ONT_PER_PORT",
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
	if c.EncryptionKey == "" {
		missing = append(missing, "ENCRYPTION_KEY")
	}

	if len(missing) > 0 {
		return fmt.Errorf(
			"variabel wajib belum diisi: %s",
			strings.Join(missing, ", "),
		)
	}

	// Validasi format ENCRYPTION_KEY: harus 64 karakter hex (32 bytes)
	if !isValidHex64(c.EncryptionKey) {
		return fmt.Errorf(
			"ENCRYPTION_KEY harus berupa 64 karakter hex (32 bytes)",
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
		if strings.TrimSpace(c.EncryptionKey) == developmentEncryptionKey {
			unsafe = append(unsafe, "ENCRYPTION_KEY tidak boleh memakai key development")
		}
		if strings.EqualFold(strings.TrimSpace(c.DBSSLMode), "disable") {
			unsafe = append(unsafe, "DB_SSL_MODE production tidak boleh disable")
		}
		if len(unsafe) > 0 {
			return fmt.Errorf("konfigurasi production tidak aman: %s", strings.Join(unsafe, "; "))
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

// EncryptionKeyBytes mendekode hex string ENCRYPTION_KEY menjadi 32 bytes.
// Mengembalikan error jika key bukan hex valid atau bukan 32 bytes.
func (c *AppConfig) EncryptionKeyBytes() ([]byte, error) {
	if c.EncryptionKey == "" {
		return nil, fmt.Errorf("ENCRYPTION_KEY belum diisi")
	}
	key, err := hex.DecodeString(c.EncryptionKey)
	if err != nil {
		return nil, fmt.Errorf("ENCRYPTION_KEY bukan hex valid: %w", err)
	}
	if len(key) != 32 {
		return nil, fmt.Errorf(
			"ENCRYPTION_KEY harus 32 bytes, ditemukan %d bytes", len(key),
		)
	}
	return key, nil
}

// hexPattern mencocokkan tepat 64 karakter hex (0-9, a-f, A-F).
var hexPattern = regexp.MustCompile(`^[0-9a-fA-F]{64}$`)

const developmentJWTSecret = "change-me-to-a-strong-secret"
const developmentDBPassword = "ispboss_secret"
const developmentEncryptionKey = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

func isProductionEnv(env string) bool {
	return strings.EqualFold(strings.TrimSpace(env), "production")
}

// isValidHex64 memeriksa apakah string berupa 64 karakter hex.
func isValidHex64(s string) bool {
	return hexPattern.MatchString(s)
}
