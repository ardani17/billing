// Paket database menyediakan koneksi pool PostgreSQL menggunakan pgx
// dan helper untuk mengelola konteks tenant dalam sistem multi-tenant.
package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Nilai bawaan untuk konfigurasi connection pool.
const (
	defaultMaxConns        = 10
	defaultMinConns        = 2
	defaultMaxConnLifetime = 30 * time.Minute
	defaultMaxConnIdleTime = 5 * time.Minute
	defaultConnTimeout     = 5 * time.Second
)

// PoolConfig berisi konfigurasi connection pool PostgreSQL.
// Jika nilai tidak diisi (zero value), akan menggunakan bawaan.
type PoolConfig struct {
	// DSN adalah connection string PostgreSQL, contoh:
	// "postgres://user:pass@localhost:5432/dbname?sslmode=disable"
	DSN string

	// MaxConns adalah jumlah maksimum koneksi dalam pool.
	MaxConns int32

	// MinConns adalah jumlah minimum koneksi yang dijaga tetap terbuka.
	MinConns int32

	// MaxConnLifetime adalah durasi maksimum sebuah koneksi bisa digunakan.
	MaxConnLifetime time.Duration

	// MaxConnIdleTime adalah durasi maksimum koneksi idle sebelum ditutup.
	MaxConnIdleTime time.Duration

	// ConnTimeout adalah batas waktu untuk mendapatkan koneksi dari pool.
	ConnTimeout time.Duration
}

// NewPool membuat connection pool baru menggunakan pgxpool.
// Melakukan parsing DSN, mengatur parameter pool, dan memverifikasi
// koneksi ke database dengan Ping.
func NewPool(ctx context.Context, cfg PoolConfig) (*pgxpool.Pool, error) {
	if cfg.DSN == "" {
		return nil, fmt.Errorf("database: DSN tidak boleh kosong")
	}

	// Parsing DSN menjadi konfigurasi pgxpool
	poolCfg, err := pgxpool.ParseConfig(cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("database: gagal parsing DSN: %w", err)
	}

	// Terapkan konfigurasi pool dengan cadangan ke bawaan
	poolCfg.MaxConns = withDefaultInt32(cfg.MaxConns, defaultMaxConns)
	poolCfg.MinConns = withDefaultInt32(cfg.MinConns, defaultMinConns)
	poolCfg.MaxConnLifetime = withDefaultDuration(cfg.MaxConnLifetime, defaultMaxConnLifetime)
	poolCfg.MaxConnIdleTime = withDefaultDuration(cfg.MaxConnIdleTime, defaultMaxConnIdleTime)
	poolCfg.HealthCheckPeriod = defaultConnTimeout

	// Buat connection pool
	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("database: gagal membuat pool: %w", err)
	}

	// Verifikasi koneksi ke database
	pingCtx, cancel := context.WithTimeout(ctx, withDefaultDuration(cfg.ConnTimeout, defaultConnTimeout))
	defer cancel()

	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("database: gagal ping ke database: %w", err)
	}

	return pool, nil
}

// SetTenantID mengatur session variable app.tenant_id di PostgreSQL
// untuk mengaktifkan RLS filtering pada koneksi saat ini.
// Harus dipanggil sebelum menjalankan kueri tenant-scoped.
func SetTenantID(ctx context.Context, pool *pgxpool.Pool, tenantID string) error {
	if tenantID == "" {
		return fmt.Errorf("database: tenant_id tidak boleh kosong")
	}

	// Gunakan Exec untuk menjalankan SET pada koneksi dari pool
	_, err := pool.Exec(ctx, "SET app.tenant_id = $1", tenantID)
	if err != nil {
		return fmt.Errorf("database: gagal mengatur tenant_id: %w", err)
	}

	return nil
}

// WithTenant menjalankan fungsi fn dalam konteks tenant tertentu.
// Mengambil koneksi dari pool, mengatur app.tenant_id, menjalankan fn,
// lalu mengembalikan koneksi ke pool.
// Koneksi akan selalu dikembalikan ke pool meskipun fn mengembalikan error.
func WithTenant(
	ctx context.Context,
	pool *pgxpool.Pool,
	tenantID string,
	fn func(conn *pgxpool.Conn) error,
) error {
	if tenantID == "" {
		return fmt.Errorf("database: tenant_id tidak boleh kosong")
	}

	// Ambil koneksi dedicated dari pool
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("database: gagal mengambil koneksi: %w", err)
	}
	defer conn.Release()

	// Atur tenant_id pada koneksi ini
	_, err = conn.Exec(ctx, "SET app.tenant_id = $1", tenantID)
	if err != nil {
		return fmt.Errorf("database: gagal mengatur tenant_id: %w", err)
	}

	// Jalankan fungsi yang diberikan dengan koneksi tenant
	return fn(conn)
}

// withDefaultInt32 mengembalikan nilai val jika > 0, atau defaultVal jika tidak.
func withDefaultInt32(val, defaultVal int32) int32 {
	if val > 0 {
		return val
	}
	return defaultVal
}

// withDefaultDuration mengembalikan val jika > 0, atau defaultVal jika tidak.
func withDefaultDuration(val, defaultVal time.Duration) time.Duration {
	if val > 0 {
		return val
	}
	return defaultVal
}
