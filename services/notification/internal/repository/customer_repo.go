package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/ispboss/ispboss/services/notification/internal/domain"
	"github.com/jackc/pgx/v5"
)

// =============================================================================
// CustomerDataRepo - implementasi fetcher data pelanggan dan tenant
// menggunakan sqlc Queries untuk keperluan substitusi variabel template.
// =============================================================================

// CustomerData berisi data pelanggan yang dibutuhkan untuk template notifikasi.
type CustomerData struct {
	ID            string
	CustomerIDSeq string
	Name          string
	Phone         string
	Email         string
	PackageID     string
}

// TenantData berisi data tenant yang dibutuhkan untuk template notifikasi.
type TenantData struct {
	ID   string
	Name string
}

// CustomerDataRepo mengimplementasikan CustomerDataFetcher dan TenantDataFetcher
// dengan membungkus sqlc-generated Queries untuk akses tabel shared (customers, tenants).
type CustomerDataRepo struct {
	queries *Queries
}

// NewCustomerDataRepo membuat instance baru CustomerDataRepo.
func NewCustomerDataRepo(queries *Queries) *CustomerDataRepo {
	return &CustomerDataRepo{queries: queries}
}

// GetCustomerByID mengambil data pelanggan berdasarkan ID.
// Mengembalikan domain.ErrCustomerNotFound jika pelanggan tidak ditemukan atau sudah dihapus.
func (r *CustomerDataRepo) GetCustomerByID(ctx context.Context, customerID string) (*CustomerData, error) {
	row, err := r.queries.GetCustomerByID(ctx, parseUUID(customerID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrCustomerNotFound
		}
		return nil, fmt.Errorf("repository: gagal mengambil data pelanggan: %w", err)
	}

	return &CustomerData{
		ID:            uuidToString(row.ID),
		CustomerIDSeq: row.CustomerIDSeq.String,
		Name:          row.Name,
		Phone:         row.Phone,
		Email:         row.Email.String,
		PackageID:     uuidToString(row.PackageID),
	}, nil
}

// GetTenantByID mengambil data tenant berdasarkan ID.
// Mengembalikan error jika tenant tidak ditemukan.
func (r *CustomerDataRepo) GetTenantByID(ctx context.Context, tenantID string) (*TenantData, error) {
	row, err := r.queries.GetTenantByID(ctx, parseUUID(tenantID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("repository: tenant tidak ditemukan: %s", tenantID)
		}
		return nil, fmt.Errorf("repository: gagal mengambil data tenant: %w", err)
	}

	return &TenantData{
		ID:   uuidToString(row.ID),
		Name: row.Name,
	}, nil
}
