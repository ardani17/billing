package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// PPPoEUserRepo mengimplementasikan domain.PPPoEUserRepository dengan membungkus
// sqlc-generated Queries dan memetakan tipe database ke domain.PPPoEUser.
type PPPoEUserRepo struct {
	queries *Queries
}

// NewPPPoEUserRepo membuat instance baru PPPoEUserRepo.
func NewPPPoEUserRepo(queries *Queries) *PPPoEUserRepo {
	return &PPPoEUserRepo{queries: queries}
}

// --- Mapping sqlc PppoeUser → domain.PPPoEUser ---

// mapPPPoEUserRow memetakan PppoeUser (sqlc model) ke domain.PPPoEUser.
func mapPPPoEUserRow(row PppoeUser) *domain.PPPoEUser {
	return &domain.PPPoEUser{
		ID:                uuidToString(row.ID),
		TenantID:          uuidToString(row.TenantID),
		CustomerID:        uuidToString(row.CustomerID),
		RouterID:          uuidToString(row.RouterID),
		Username:          row.Username,
		PasswordEncrypted: row.PasswordEncrypted,
		ProfileName:       row.ProfileName,
		Service:           row.Service,
		RemoteAddress:     textToString(row.RemoteAddress),
		Comment:           row.Comment,
		Disabled:          row.Disabled,
		UseSimpleQueue:    row.UseSimpleQueue,
		Status:            row.Status,
		LastSyncAt:        timestamptzToTimePtr(row.LastSyncAt),
		SyncStatus:        domain.SyncStatus(row.SyncStatus),
		CreatedAt:         timestamptzToTime(row.CreatedAt),
		UpdatedAt:         timestamptzToTime(row.UpdatedAt),
		DeletedAt:         timestamptzToTimePtr(row.DeletedAt),
	}
}

// isUniqueViolation memeriksa apakah error adalah unique constraint violation (kode 23505).
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

// --- Implementasi domain.PPPoEUserRepository ---

// Create membuat record PPPoE user baru.
func (r *PPPoEUserRepo) Create(ctx context.Context, user *domain.PPPoEUser) (*domain.PPPoEUser, error) {
	row, err := r.queries.CreatePPPoEUser(ctx, CreatePPPoEUserParams{
		TenantID:          stringToUUID(user.TenantID),
		CustomerID:        stringToUUID(user.CustomerID),
		RouterID:          stringToUUID(user.RouterID),
		Username:          user.Username,
		PasswordEncrypted: user.PasswordEncrypted,
		ProfileName:       user.ProfileName,
		Service:           user.Service,
		RemoteAddress:     stringToText(user.RemoteAddress),
		Comment:           user.Comment,
		Disabled:          user.Disabled,
		UseSimpleQueue:    user.UseSimpleQueue,
		Status:            user.Status,
		SyncStatus:        string(user.SyncStatus),
	})
	if err != nil {
		if isUniqueViolation(err) {
			return nil, domain.ErrPPPoEUsernameExists
		}
		return nil, fmt.Errorf("repository: gagal membuat pppoe user: %w", err)
	}
	return mapPPPoEUserRow(row), nil
}

// GetByID mengambil PPPoE user berdasarkan ID.
func (r *PPPoEUserRepo) GetByID(ctx context.Context, id string) (*domain.PPPoEUser, error) {
	row, err := r.queries.GetPPPoEUserByID(ctx, stringToUUID(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrPPPoEUserNotFound
		}
		return nil, fmt.Errorf("repository: gagal mengambil pppoe user by ID: %w", err)
	}
	return mapPPPoEUserRow(row), nil
}

// GetByUsername mengambil PPPoE user berdasarkan router_id dan username.
func (r *PPPoEUserRepo) GetByUsername(ctx context.Context, routerID, username string) (*domain.PPPoEUser, error) {
	row, err := r.queries.GetPPPoEUserByUsername(ctx, GetPPPoEUserByUsernameParams{
		RouterID: stringToUUID(routerID),
		Username: username,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrPPPoEUserNotFound
		}
		return nil, fmt.Errorf("repository: gagal mengambil pppoe user by username: %w", err)
	}
	return mapPPPoEUserRow(row), nil
}

// GetByCustomerID mengambil PPPoE user berdasarkan customer_id.
func (r *PPPoEUserRepo) GetByCustomerID(ctx context.Context, customerID string) (*domain.PPPoEUser, error) {
	row, err := r.queries.GetPPPoEUserByCustomerID(ctx, stringToUUID(customerID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrPPPoEUserNotFound
		}
		return nil, fmt.Errorf("repository: gagal mengambil pppoe user by customer ID: %w", err)
	}
	return mapPPPoEUserRow(row), nil
}

// Update memperbarui record PPPoE user.
func (r *PPPoEUserRepo) Update(ctx context.Context, user *domain.PPPoEUser) (*domain.PPPoEUser, error) {
	row, err := r.queries.UpdatePPPoEUser(ctx, UpdatePPPoEUserParams{
		ID:                stringToUUID(user.ID),
		Username:          user.Username,
		PasswordEncrypted: user.PasswordEncrypted,
		ProfileName:       user.ProfileName,
		Service:           user.Service,
		RemoteAddress:     stringToText(user.RemoteAddress),
		Comment:           user.Comment,
		Disabled:          user.Disabled,
		UseSimpleQueue:    user.UseSimpleQueue,
		Status:            user.Status,
		SyncStatus:        string(user.SyncStatus),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrPPPoEUserNotFound
		}
		if isUniqueViolation(err) {
			return nil, domain.ErrPPPoEUsernameExists
		}
		return nil, fmt.Errorf("repository: gagal memperbarui pppoe user: %w", err)
	}
	return mapPPPoEUserRow(row), nil
}

// SoftDelete melakukan soft-delete PPPoE user.
func (r *PPPoEUserRepo) SoftDelete(ctx context.Context, id string) error {
	err := r.queries.SoftDeletePPPoEUser(ctx, stringToUUID(id))
	if err != nil {
		return fmt.Errorf("repository: gagal soft-delete pppoe user: %w", err)
	}
	return nil
}

// List mengambil daftar PPPoE user dengan paginasi per router.
func (r *PPPoEUserRepo) List(ctx context.Context, params domain.PPPoEUserListParams) (*domain.PPPoEUserListResult, error) {
	offset := (params.Page - 1) * params.PageSize

	total, err := r.queries.CountPPPoEUsers(ctx, CountPPPoEUsersParams{
		RouterID:   stringToUUID(params.RouterID),
		SyncStatus: stringToText(params.SyncStatus),
		Search:     stringToText(params.Search),
	})
	if err != nil {
		return nil, fmt.Errorf("repository: gagal menghitung total pppoe users: %w", err)
	}

	rows, err := r.queries.ListPPPoEUsers(ctx, ListPPPoEUsersParams{
		RouterID:   stringToUUID(params.RouterID),
		Limit:      int32(params.PageSize),
		Offset:     int32(offset),
		SyncStatus: stringToText(params.SyncStatus),
		Search:     stringToText(params.Search),
	})
	if err != nil {
		return nil, fmt.Errorf("repository: gagal mengambil daftar pppoe users: %w", err)
	}

	users := make([]*domain.PPPoEUser, 0, len(rows))
	for _, row := range rows {
		users = append(users, mapPPPoEUserRow(row))
	}

	totalPages := int(total) / params.PageSize
	if int(total)%params.PageSize > 0 {
		totalPages++
	}

	return &domain.PPPoEUserListResult{
		Data:       users,
		Total:      total,
		Page:       params.Page,
		PageSize:   params.PageSize,
		TotalPages: totalPages,
	}, nil
}
