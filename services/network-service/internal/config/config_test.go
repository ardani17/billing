package config

import "testing"

func setRequiredConfigEnv(t *testing.T) {
	t.Helper()
	t.Setenv("DB_HOST", "localhost")
	t.Setenv("DB_USER", "postgres")
	t.Setenv("DB_PASSWORD", "postgres_password")
	t.Setenv("DB_NAME", "billing")
	t.Setenv("REDIS_HOST", "localhost")
	t.Setenv("JWT_SECRET", "development-secret-for-test-only")
	t.Setenv("ENCRYPTION_KEY", "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
}

func TestLoad_OLTGuardDefaultsDisabled(t *testing.T) {
	setRequiredConfigEnv(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.OLTHealthCheckEnabled {
		t.Fatal("OLT_HEALTH_CHECK_ENABLED default harus false")
	}
	if cfg.OLTSyncEnabled {
		t.Fatal("OLT_SYNC_ENABLED default harus false")
	}
	if cfg.OLTSyncImmediateEnabled {
		t.Fatal("OLT_SYNC_IMMEDIATE_ENABLED default harus false")
	}
	if cfg.OLTTrapEnabled {
		t.Fatal("OLT_TRAP_ENABLED default harus false")
	}
	if cfg.OLTProvisioningWriteEnabled {
		t.Fatal("OLT_PROVISIONING_WRITE_ENABLED default harus false")
	}
}

func TestLoad_OLTGuardEnvOverride(t *testing.T) {
	setRequiredConfigEnv(t)
	t.Setenv("OLT_HEALTH_CHECK_ENABLED", "true")
	t.Setenv("OLT_SYNC_ENABLED", "true")
	t.Setenv("OLT_SYNC_IMMEDIATE_ENABLED", "true")
	t.Setenv("OLT_TRAP_ENABLED", "true")
	t.Setenv("OLT_PROVISIONING_WRITE_ENABLED", "true")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if !cfg.OLTHealthCheckEnabled || !cfg.OLTSyncEnabled || !cfg.OLTSyncImmediateEnabled || !cfg.OLTTrapEnabled || !cfg.OLTProvisioningWriteEnabled {
		t.Fatalf("OLT guard env override tidak terbaca: %+v", cfg)
	}
}
