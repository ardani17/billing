package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
	"github.com/jackc/pgx/v5"
)

// ONTRepo mengimplementasikan domain.ONTRepository dengan membungkus
// sqlc-generated Queries dan memetakan tipe database ke domain.ONT.
type ONTRepo struct {
	queries *Queries
}

// NewONTRepo membuat instance baru ONTRepo.
func NewONTRepo(queries *Queries) *ONTRepo {
	return &ONTRepo{queries: queries}
}

// --- Mapping sqlc Ont -> domain.ONT ---

// mapONTRow memetakan Ont (sqlc model) ke domain.ONT.
func mapONTRow(row Ont) *domain.ONT {
	return &domain.ONT{
		ID:                   uuidToString(row.ID),
		TenantID:             uuidToString(row.TenantID),
		OLTID:                uuidToString(row.OltID),
		PONPortIndex:         int(row.PonPortIndex),
		ONTIndex:             int(row.OntIndex),
		SerialNumber:         row.SerialNumber,
		CustomerID:           uuidToStringPtr(row.CustomerID),
		ODPID:                uuidToStringPtr(row.OdpID),
		VLANID:               uuidToStringPtr(row.VlanID),
		ServiceProfileID:     uuidToStringPtr(row.ServiceProfileID),
		Status:               domain.ONTStatus(row.Status),
		ProvisioningState:    domain.ProvisioningState(row.ProvisioningState),
		Description:          textToString(row.Description),
		LastProvisionedAt:    timestamptzToTimePtr(row.LastProvisionedAt),
		LastDecommissionedAt: timestamptzToTimePtr(row.LastDecommissionedAt),
		DeletedAt:            timestamptzToTimePtr(row.DeletedAt),
		CreatedAt:            timestamptzToTime(row.CreatedAt),
		UpdatedAt:            timestamptzToTime(row.UpdatedAt),
	}
}

// --- Implementasi CRUD domain.ONTRepository ---

// Buat membuat ONT baru dan mengembalikan ONT yang dibuat.
func (r *ONTRepo) Create(ctx context.Context, ont *domain.ONT) (*domain.ONT, error) {
	row, err := r.queries.CreateONT(ctx, CreateONTParams{
		TenantID:          stringToUUID(ont.TenantID),
		OltID:             stringToUUID(ont.OLTID),
		PonPortIndex:      int32(ont.PONPortIndex),
		OntIndex:          int32(ont.ONTIndex),
		SerialNumber:      ont.SerialNumber,
		CustomerID:        stringPtrToUUID(ont.CustomerID),
		OdpID:             stringPtrToUUID(ont.ODPID),
		VlanID:            stringPtrToUUID(ont.VLANID),
		ServiceProfileID:  stringPtrToUUID(ont.ServiceProfileID),
		Status:            string(ont.Status),
		ProvisioningState: string(ont.ProvisioningState),
		Description:       stringToText(ont.Description),
	})
	if err != nil {
		return nil, fmt.Errorf("repository: gagal membuat ONT: %w", err)
	}
	return mapONTRow(row), nil
}

// GetByID mengambil ONT berdasarkan ID (tenant-scoped via RLS).
func (r *ONTRepo) GetByID(ctx context.Context, id string) (*domain.ONT, error) {
	row, err := r.queries.GetONTByID(ctx, stringToUUID(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrONTNotFound
		}
		return nil, fmt.Errorf("repository: gagal mengambil ONT by ID: %w", err)
	}
	return mapONTRow(row), nil
}

// GetBySerialNumber mengambil ONT berdasarkan tenant_id dan serial_number.
func (r *ONTRepo) GetBySerialNumber(ctx context.Context, tenantID, serialNumber string) (*domain.ONT, error) {
	row, err := r.queries.GetONTBySerialNumber(ctx, GetONTBySerialNumberParams{
		TenantID:     stringToUUID(tenantID),
		SerialNumber: serialNumber,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrONTNotFound
		}
		return nil, fmt.Errorf("repository: gagal mengambil ONT by serial number: %w", err)
	}
	return mapONTRow(row), nil
}

// Perbarui memperbarui record ONT dan mengembalikan ONT yang diperbarui.
func (r *ONTRepo) Update(ctx context.Context, ont *domain.ONT) (*domain.ONT, error) {
	row, err := r.queries.UpdateONT(ctx, UpdateONTParams{
		ID:                   stringToUUID(ont.ID),
		PonPortIndex:         int32(ont.PONPortIndex),
		OntIndex:             int32(ont.ONTIndex),
		SerialNumber:         ont.SerialNumber,
		CustomerID:           stringPtrToUUID(ont.CustomerID),
		OdpID:                stringPtrToUUID(ont.ODPID),
		VlanID:               stringPtrToUUID(ont.VLANID),
		ServiceProfileID:     stringPtrToUUID(ont.ServiceProfileID),
		Status:               string(ont.Status),
		ProvisioningState:    string(ont.ProvisioningState),
		Description:          stringToText(ont.Description),
		LastProvisionedAt:    timePtrToTimestamptz(ont.LastProvisionedAt),
		LastDecommissionedAt: timePtrToTimestamptz(ont.LastDecommissionedAt),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrONTNotFound
		}
		return nil, fmt.Errorf("repository: gagal memperbarui ONT: %w", err)
	}
	return mapONTRow(row), nil
}

// SoftDelete melakukan hapus lunak ONT (atur deleted_at).
func (r *ONTRepo) SoftDelete(ctx context.Context, id string) error {
	err := r.queries.SoftDeleteONT(ctx, stringToUUID(id))
	if err != nil {
		return fmt.Errorf("repository: gagal soft-delete ONT: %w", err)
	}
	return nil
}

// Compile-time cek: ONTRepo mengimplementasikan domain.ONTRepository.
var _ domain.ONTRepository = (*ONTRepo)(nil)
