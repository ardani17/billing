package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
	"github.com/jackc/pgx/v5"
)

// OLTRepo mengimplementasikan domain.OLTRepository dengan membungkus
// sqlc-generated Queries dan memetakan tipe database ke domain.OLT.
type OLTRepo struct {
	queries *Queries
}

// NewOLTRepo membuat instance baru OLTRepo.
func NewOLTRepo(queries *Queries) *OLTRepo {
	return &OLTRepo{queries: queries}
}

// --- Mapping sqlc Olt -> domain.OLT ---

// mapOLTRow memetakan Olt (sqlc model) ke domain.OLT.
func mapOLTRow(row Olt) *domain.OLT {
	return &domain.OLT{
		ID:                         uuidToString(row.ID),
		TenantID:                   uuidToString(row.TenantID),
		Name:                       row.Name,
		Host:                       row.Host,
		SNMPVersion:                domain.SNMPVersion(row.SnmpVersion),
		SNMPPort:                   int(row.SnmpPort),
		SNMPCommunityEncrypted:     textToString(row.SnmpCommunityEncrypted),
		SNMPUsername:               textToString(row.SnmpUsername),
		SNMPAuthProtocol:           textToString(row.SnmpAuthProtocol),
		SNMPAuthPasswordEncrypted:  textToString(row.SnmpAuthPasswordEncrypted),
		SNMPPrivProtocol:           textToString(row.SnmpPrivProtocol),
		SNMPPrivPasswordEncrypted:  textToString(row.SnmpPrivPasswordEncrypted),
		CLIProtocol:                domain.CLIProtocol(row.CliProtocol),
		CLIPort:                    int(row.CliPort),
		CLIUsername:                row.CliUsername,
		CLIPasswordEncrypted:       row.CliPasswordEncrypted,
		CLIEnablePasswordEncrypted: textToString(row.CliEnablePasswordEncrypted),
		Brand:                      domain.OLTBrand(textToString(row.Brand)),
		Model:                      textToString(row.Model),
		FirmwareVersion:            textToString(row.FirmwareVersion),
		PONPortCount:               int(row.PonPortCount),
		TotalONTCount:              int(row.TotalOntCount),
		Status:                     domain.OLTStatus(row.Status),
		HealthCheckIntervalSec:     int(row.HealthCheckIntervalSec),
		LastOnlineAt:               timestamptzToTimePtr(row.LastOnlineAt),
		LastCheckedAt:              timestamptzToTimePtr(row.LastCheckedAt),
		FailureCount:               int(row.FailureCount),
		Notes:                      textToString(row.Notes),
		DeletedAt:                  timestamptzToTimePtr(row.DeletedAt),
		CreatedAt:                  timestamptzToTime(row.CreatedAt),
		UpdatedAt:                  timestamptzToTime(row.UpdatedAt),
	}
}

// --- Implementasi CRUD domain.OLTRepository ---

// Buat membuat OLT baru dan mengembalikan OLT yang dibuat.
func (r *OLTRepo) Create(ctx context.Context, olt *domain.OLT) (*domain.OLT, error) {
	row, err := r.queries.CreateOLT(ctx, CreateOLTParams{
		TenantID:                   stringToUUID(olt.TenantID),
		Name:                       olt.Name,
		Host:                       olt.Host,
		SnmpVersion:                string(olt.SNMPVersion),
		SnmpPort:                   int32(olt.SNMPPort),
		SnmpCommunityEncrypted:     stringToText(olt.SNMPCommunityEncrypted),
		SnmpUsername:               stringToText(olt.SNMPUsername),
		SnmpAuthProtocol:           stringToText(olt.SNMPAuthProtocol),
		SnmpAuthPasswordEncrypted:  stringToText(olt.SNMPAuthPasswordEncrypted),
		SnmpPrivProtocol:           stringToText(olt.SNMPPrivProtocol),
		SnmpPrivPasswordEncrypted:  stringToText(olt.SNMPPrivPasswordEncrypted),
		CliProtocol:                string(olt.CLIProtocol),
		CliPort:                    int32(olt.CLIPort),
		CliUsername:                olt.CLIUsername,
		CliPasswordEncrypted:       olt.CLIPasswordEncrypted,
		CliEnablePasswordEncrypted: stringToText(olt.CLIEnablePasswordEncrypted),
		Brand:                      stringToText(string(olt.Brand)),
		Model:                      stringToText(olt.Model),
		FirmwareVersion:            stringToText(olt.FirmwareVersion),
		PonPortCount:               int32(olt.PONPortCount),
		TotalOntCount:              int32(olt.TotalONTCount),
		Status:                     string(olt.Status),
		HealthCheckIntervalSec:     int32(olt.HealthCheckIntervalSec),
		Notes:                      stringToText(olt.Notes),
	})
	if err != nil {
		return nil, fmt.Errorf("repository: gagal membuat OLT: %w", err)
	}
	return mapOLTRow(row), nil
}

// GetByID mengambil OLT berdasarkan ID (tenant-scoped via RLS).
func (r *OLTRepo) GetByID(ctx context.Context, id string) (*domain.OLT, error) {
	row, err := r.queries.GetOLTByID(ctx, stringToUUID(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrOLTNotFound
		}
		return nil, fmt.Errorf("repository: gagal mengambil OLT by ID: %w", err)
	}
	return mapOLTRow(row), nil
}

// Perbarui memperbarui data OLT dan mengembalikan OLT yang diperbarui.
func (r *OLTRepo) Update(ctx context.Context, olt *domain.OLT) (*domain.OLT, error) {
	row, err := r.queries.UpdateOLT(ctx, UpdateOLTParams{
		ID:                         stringToUUID(olt.ID),
		Name:                       olt.Name,
		Host:                       olt.Host,
		SnmpVersion:                string(olt.SNMPVersion),
		SnmpPort:                   int32(olt.SNMPPort),
		SnmpCommunityEncrypted:     stringToText(olt.SNMPCommunityEncrypted),
		SnmpUsername:               stringToText(olt.SNMPUsername),
		SnmpAuthProtocol:           stringToText(olt.SNMPAuthProtocol),
		SnmpAuthPasswordEncrypted:  stringToText(olt.SNMPAuthPasswordEncrypted),
		SnmpPrivProtocol:           stringToText(olt.SNMPPrivProtocol),
		SnmpPrivPasswordEncrypted:  stringToText(olt.SNMPPrivPasswordEncrypted),
		CliProtocol:                string(olt.CLIProtocol),
		CliPort:                    int32(olt.CLIPort),
		CliUsername:                olt.CLIUsername,
		CliPasswordEncrypted:       olt.CLIPasswordEncrypted,
		CliEnablePasswordEncrypted: stringToText(olt.CLIEnablePasswordEncrypted),
		Brand:                      stringToText(string(olt.Brand)),
		Model:                      stringToText(olt.Model),
		FirmwareVersion:            stringToText(olt.FirmwareVersion),
		PonPortCount:               int32(olt.PONPortCount),
		Status:                     string(olt.Status),
		HealthCheckIntervalSec:     int32(olt.HealthCheckIntervalSec),
		Notes:                      stringToText(olt.Notes),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrOLTNotFound
		}
		return nil, fmt.Errorf("repository: gagal memperbarui OLT: %w", err)
	}
	return mapOLTRow(row), nil
}

// SoftDelete melakukan hapus lunak OLT (atur deleted_at).
func (r *OLTRepo) SoftDelete(ctx context.Context, id string) error {
	err := r.queries.SoftDeleteOLT(ctx, stringToUUID(id))
	if err != nil {
		return fmt.Errorf("repository: gagal soft-delete OLT: %w", err)
	}
	return nil
}

// Compile-time cek: OLTRepo mengimplementasikan domain.OLTRepository.
var _ domain.OLTRepository = (*OLTRepo)(nil)
