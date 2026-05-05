package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

type TenantModuleRepo struct {
	db *pgxpool.Pool
}

func NewTenantModuleRepo(db *pgxpool.Pool) *TenantModuleRepo {
	return &TenantModuleRepo{db: db}
}

func (r *TenantModuleRepo) Capabilities(ctx context.Context, tenantID string) (domain.TenantModuleCapabilities, error) {
	caps := domain.DefaultTenantModuleCapabilities()

	rows, err := r.db.Query(ctx, `
		SELECT module_code, status
		FROM tenant_modules
		WHERE tenant_id = $1
	`, tenantID)
	if err != nil {
		if pgErr, ok := err.(*pgconn.PgError); ok && pgErr.Code == "42P01" {
			return caps, nil
		}
		return caps, err
	}
	defer rows.Close()

	for rows.Next() {
		var code, status string
		if err := rows.Scan(&code, &status); err != nil {
			return caps, err
		}
		enabled := status == "active"
		switch code {
		case domain.ModuleBillingCore:
			caps.BillingCore = enabled
		case domain.ModuleMikroTik:
			caps.MikroTik = enabled
		case domain.ModuleFiberNetwork:
			caps.FiberNetwork = enabled
		}
	}

	return caps, rows.Err()
}
