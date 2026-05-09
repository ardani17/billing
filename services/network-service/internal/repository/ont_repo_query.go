package repository

import (
	"context"
	"fmt"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// ont_repo_kueri.go berisi method kueri ONTRepo: List, ListByOLTAndStatus,
// GetByCustomerID, SerialNumberExists, PositionExists, UpdateStatus,
// UpdatePortMigration, DeleteUnregisteredByOLT.

// List mengambil daftar ONT dengan paginasi dan filter (tenant-scoped via RLS).
func (r *ONTRepo) List(ctx context.Context, params domain.ONTListParams) (*domain.ONTListResult, error) {
	// Hitung offset dari page dan page_size
	offset := (params.Page - 1) * params.PageSize

	// Ambil total count untuk paginasi
	total, err := r.queries.CountONTs(ctx, CountONTsParams{
		OltID:             stringToNullableUUID(params.OLTID),
		Status:            stringToText(params.Status),
		ProvisioningState: stringToText(params.ProvisioningState),
		CustomerID:        stringToNullableUUID(params.CustomerID),
		Search:            stringToText(params.Search),
	})
	if err != nil {
		return nil, fmt.Errorf("repository: gagal menghitung total ONT: %w", err)
	}

	// Ambil data ONT
	rows, err := r.queries.ListONTs(ctx, ListONTsParams{
		Limit:             int32(params.PageSize),
		Offset:            int32(offset),
		OltID:             stringToNullableUUID(params.OLTID),
		Status:            stringToText(params.Status),
		ProvisioningState: stringToText(params.ProvisioningState),
		CustomerID:        stringToNullableUUID(params.CustomerID),
		Search:            stringToText(params.Search),
	})
	if err != nil {
		return nil, fmt.Errorf("repository: gagal mengambil daftar ONT: %w", err)
	}

	// Konversi ke domain.ONTResponse untuk list
	responses := make([]*domain.ONTResponse, 0, len(rows))
	for _, row := range rows {
		ont := mapONTRow(row)
		responses = append(responses, &domain.ONTResponse{
			ID:                   ont.ID,
			OLTID:                ont.OLTID,
			PONPortIndex:         ont.PONPortIndex,
			ONTIndex:             ont.ONTIndex,
			SerialNumber:         ont.SerialNumber,
			CustomerID:           ont.CustomerID,
			ODPID:                ont.ODPID,
			VLANID:               ont.VLANID,
			ServiceProfileID:     ont.ServiceProfileID,
			Status:               ont.Status,
			ProvisioningState:    ont.ProvisioningState,
			Description:          ont.Description,
			LastProvisionedAt:    ont.LastProvisionedAt,
			LastDecommissionedAt: ont.LastDecommissionedAt,
			CreatedAt:            ont.CreatedAt,
			UpdatedAt:            ont.UpdatedAt,
		})
	}

	// Hitung total pages
	totalPages := int(total) / params.PageSize
	if int(total)%params.PageSize > 0 {
		totalPages++
	}

	return &domain.ONTListResult{
		Data:       responses,
		Total:      total,
		Page:       params.Page,
		PageSize:   params.PageSize,
		TotalPages: totalPages,
	}, nil
}

// ListByOLTAndStatus mengambil ONT berdasarkan olt_id dan status.
func (r *ONTRepo) ListByOLTAndStatus(ctx context.Context, oltID, status string) ([]*domain.ONT, error) {
	rows, err := r.queries.ListONTsByOLTAndStatus(ctx, ListONTsByOLTAndStatusParams{
		OltID:  stringToUUID(oltID),
		Status: status,
	})
	if err != nil {
		return nil, fmt.Errorf("repository: gagal mengambil ONT by OLT dan status: %w", err)
	}

	onts := make([]*domain.ONT, 0, len(rows))
	for _, row := range rows {
		onts = append(onts, mapONTRow(row))
	}
	return onts, nil
}

// GetByCustomerID mengambil ONT aktif berdasarkan customer_id.
func (r *ONTRepo) GetByCustomerID(ctx context.Context, customerID string) (*domain.ONT, error) {
	row, err := r.queries.GetONTByCustomerID(ctx, stringToUUID(customerID))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrONTNotFound
		}
		return nil, fmt.Errorf("repository: gagal mengambil ONT by customer ID: %w", err)
	}
	return mapONTRow(row), nil
}

// SerialNumberExists mengecek apakah serial number sudah ada di tenant.
func (r *ONTRepo) SerialNumberExists(ctx context.Context, tenantID, serialNumber, excludeID string) (bool, error) {
	exID := excludeID
	if exID == "" {
		exID = "00000000-0000-0000-0000-000000000000"
	}

	exists, err := r.queries.ONTSerialNumberExists(ctx, ONTSerialNumberExistsParams{
		TenantID:     stringToUUID(tenantID),
		SerialNumber: serialNumber,
		ID:           stringToUUID(exID),
	})
	if err != nil {
		return false, fmt.Errorf("repository: gagal mengecek serial number ONT: %w", err)
	}
	return exists, nil
}

// PositionExists mengecek apakah posisi (olt_id, pon_port, ont_index) sudah terisi.
func (r *ONTRepo) PositionExists(ctx context.Context, oltID string, ponPort, ontIndex int, excludeID string) (bool, error) {
	exID := excludeID
	if exID == "" {
		exID = "00000000-0000-0000-0000-000000000000"
	}

	exists, err := r.queries.ONTPositionExists(ctx, ONTPositionExistsParams{
		OltID:        stringToUUID(oltID),
		PonPortIndex: int32(ponPort),
		OntIndex:     int32(ontIndex),
		ID:           stringToUUID(exID),
	})
	if err != nil {
		return false, fmt.Errorf("repository: gagal mengecek posisi ONT: %w", err)
	}
	return exists, nil
}

// UpdateStatus memperbarui status dan provisioning_state ONT.
func (r *ONTRepo) UpdateStatus(ctx context.Context, id string, status, provisioningState string) error {
	err := r.queries.UpdateONTStatus(ctx, UpdateONTStatusParams{
		ID:                stringToUUID(id),
		Status:            status,
		ProvisioningState: provisioningState,
	})
	if err != nil {
		return fmt.Errorf("repository: gagal memperbarui status ONT: %w", err)
	}
	return nil
}

// UpdatePortMigration memperbarui pon_port_index dan ont_index setelah migrasi.
func (r *ONTRepo) UpdatePortMigration(ctx context.Context, id string, newPort, newONTIndex int) error {
	err := r.queries.UpdateONTPortMigration(ctx, UpdateONTPortMigrationParams{
		ID:           stringToUUID(id),
		PonPortIndex: int32(newPort),
		OntIndex:     int32(newONTIndex),
	})
	if err != nil {
		return fmt.Errorf("repository: gagal memperbarui port migration ONT: %w", err)
	}
	return nil
}

// DeleteUnregisteredByOLT menghapus ONT unregistered yang tidak lagi terdeteksi.
func (r *ONTRepo) DeleteUnregisteredByOLT(ctx context.Context, oltID string, keepSerialNumbers []string) (int64, error) {
	if keepSerialNumbers == nil {
		keepSerialNumbers = []string{}
	}

	count, err := r.queries.DeleteUnregisteredONTsByOLT(ctx, DeleteUnregisteredONTsByOLTParams{
		OltID:   stringToUUID(oltID),
		Column2: keepSerialNumbers,
	})
	if err != nil {
		return 0, fmt.Errorf("repository: gagal menghapus ONT unregistered: %w", err)
	}
	return count, nil
}

// --- Fungsi bantu untuk nullable UUID filter ---

// stringToNullableUUID mengkonversi string ke pgtype.UUID untuk filter opsional.
// String kosong -> UUID tidak valid (NULL) agar filter diabaikan.
func stringToNullableUUID(s string) pgtype.UUID {
	if s == "" {
		return pgtype.UUID{Valid: false}
	}
	return stringToUUID(s)
}
