package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ModuleEntitlementRepo struct {
	db *pgxpool.Pool
}

func NewModuleEntitlementRepo(db *pgxpool.Pool) *ModuleEntitlementRepo {
	return &ModuleEntitlementRepo{db: db}
}

func (r *ModuleEntitlementRepo) IsEnabled(ctx context.Context, tenantID, moduleCode string) (bool, error) {
	if moduleCode == "billing_core" {
		return true, nil
	}

	var enabled bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM tenant_modules
			WHERE tenant_id = $1
				AND module_code = $2
				AND status = 'active'
				AND (expires_at IS NULL OR expires_at > NOW())
		)
	`, tenantID, moduleCode).Scan(&enabled)
	if err != nil {
		if pgErr, ok := err.(*pgconn.PgError); ok && pgErr.Code == "42P01" {
			return false, nil
		}
		return false, err
	}

	return enabled, nil
}
