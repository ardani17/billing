package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
	"github.com/jackc/pgx/v5"
)

// VPNSubnetRepo mengimplementasikan domain.VPNSubnetRepository dengan membungkus
// sqlc-generated Queries dan memetakan tipe database ke domain.VPNSubnet.
type VPNSubnetRepo struct {
	queries *Queries
}

// NewVPNSubnetRepo membuat instance baru VPNSubnetRepo.
func NewVPNSubnetRepo(queries *Queries) *VPNSubnetRepo {
	return &VPNSubnetRepo{queries: queries}
}

// --- Mapping sqlc VpnSubnet → domain.VPNSubnet ---

// mapVPNSubnetRow memetakan VpnSubnet (sqlc model) ke domain.VPNSubnet.
func mapVPNSubnetRow(row VpnSubnet) *domain.VPNSubnet {
	return &domain.VPNSubnet{
		ID:              uuidToString(row.ID),
		TenantID:        uuidToString(row.TenantID),
		SubnetPrefix:    row.SubnetPrefix,
		TenantSeq:       int(row.TenantSeq),
		ServerIP:        row.ServerIp,
		NextClientIPSeq: int(row.NextClientIpSeq),
		CreatedAt:       timestamptzToTime(row.CreatedAt),
	}
}

// --- Implementasi domain.VPNSubnetRepository ---

// GetByTenantID mengambil subnet allocation untuk tenant.
func (r *VPNSubnetRepo) GetByTenantID(ctx context.Context, tenantID string) (*domain.VPNSubnet, error) {
	row, err := r.queries.GetVPNSubnetByTenantID(ctx, stringToUUID(tenantID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // belum ada subnet untuk tenant ini
		}
		return nil, fmt.Errorf("repository: gagal mengambil vpn subnet by tenant: %w", err)
	}
	return mapVPNSubnetRow(row), nil
}

// Create membuat subnet allocation baru untuk tenant.
func (r *VPNSubnetRepo) Create(ctx context.Context, subnet *domain.VPNSubnet) (*domain.VPNSubnet, error) {
	row, err := r.queries.CreateVPNSubnet(ctx, CreateVPNSubnetParams{
		TenantID:        stringToUUID(subnet.TenantID),
		SubnetPrefix:    subnet.SubnetPrefix,
		TenantSeq:       int32(subnet.TenantSeq),
		ServerIp:        subnet.ServerIP,
		NextClientIpSeq: int32(subnet.NextClientIPSeq),
	})
	if err != nil {
		if isUniqueViolation(err) {
			return nil, fmt.Errorf("repository: subnet untuk tenant sudah ada: %w", err)
		}
		return nil, fmt.Errorf("repository: gagal membuat vpn subnet: %w", err)
	}
	return mapVPNSubnetRow(row), nil
}

// GetNextTenantSeq mengambil tenant_seq berikutnya yang tersedia.
func (r *VPNSubnetRepo) GetNextTenantSeq(ctx context.Context) (int, error) {
	seq, err := r.queries.GetNextTenantSeq(ctx)
	if err != nil {
		return 0, fmt.Errorf("repository: gagal mengambil next tenant seq: %w", err)
	}
	return int(seq), nil
}

// IncrementNextClientIPSeq menaikkan next_client_ip_seq dan mengembalikan nilai sebelumnya.
func (r *VPNSubnetRepo) IncrementNextClientIPSeq(ctx context.Context, tenantID string) (int, error) {
	seq, err := r.queries.IncrementNextClientIPSeq(ctx, stringToUUID(tenantID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, fmt.Errorf("repository: subnet untuk tenant tidak ditemukan")
		}
		return 0, fmt.Errorf("repository: gagal increment client ip seq: %w", err)
	}
	return int(seq), nil
}
