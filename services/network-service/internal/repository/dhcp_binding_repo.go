package repository

import (
	"context"
	"errors"
	"fmt"
	"net/netip"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

type DHCPBindingRepo struct {
	queries *Queries
}

func NewDHCPBindingRepo(queries *Queries) *DHCPBindingRepo {
	return &DHCPBindingRepo{queries: queries}
}

func mapDHCPBinding(row DhcpBinding) *domain.DHCPBinding {
	return &domain.DHCPBinding{
		ID:            uuidToString(row.ID),
		TenantID:      uuidToString(row.TenantID),
		RouterID:      uuidToString(row.RouterID),
		CustomerID:    uuidToString(row.CustomerID),
		RouterLeaseID: textToString(row.RouterLeaseID),
		Server:        row.Server,
		MACAddress:    row.MacAddress,
		IPAddress:     row.IpAddress.String(),
		HostName:      textToString(row.HostName),
		Comment:       row.Comment,
		Disabled:      row.Disabled,
		Status:        row.Status,
		LastSyncAt:    timestamptzToTimePtr(row.LastSyncAt),
		SyncStatus:    row.SyncStatus,
		CreatedAt:     timestamptzToTime(row.CreatedAt),
		UpdatedAt:     timestamptzToTime(row.UpdatedAt),
		DeletedAt:     timestamptzToTimePtr(row.DeletedAt),
	}
}

func optionalUUID(value string) pgtype.UUID {
	if value == "" {
		return pgtype.UUID{Valid: false}
	}
	return stringToUUID(value)
}

func mustIP(value string) (netip.Addr, error) {
	ip, err := netip.ParseAddr(value)
	if err != nil {
		return netip.Addr{}, domain.ErrInvalidIPAddress
	}
	return ip, nil
}

func (r *DHCPBindingRepo) Create(ctx context.Context, binding *domain.DHCPBinding) (*domain.DHCPBinding, error) {
	ip, err := mustIP(binding.IPAddress)
	if err != nil {
		return nil, err
	}
	row, err := r.queries.CreateDHCPBinding(ctx, CreateDHCPBindingParams{
		TenantID:      stringToUUID(binding.TenantID),
		RouterID:      stringToUUID(binding.RouterID),
		CustomerID:    optionalUUID(binding.CustomerID),
		RouterLeaseID: stringToText(binding.RouterLeaseID),
		Server:        binding.Server,
		MacAddress:    binding.MACAddress,
		IpAddress:     ip,
		HostName:      stringToText(binding.HostName),
		Comment:       binding.Comment,
		Disabled:      binding.Disabled,
		Status:        binding.Status,
		SyncStatus:    binding.SyncStatus,
	})
	if err != nil {
		return nil, mapDHCPRepoError("create", err)
	}
	return mapDHCPBinding(row), nil
}

func (r *DHCPBindingRepo) GetByID(ctx context.Context, id string) (*domain.DHCPBinding, error) {
	row, err := r.queries.GetDHCPBindingByID(ctx, stringToUUID(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrDHCPBindingNotFound
		}
		return nil, fmt.Errorf("repository: gagal mengambil dhcp binding: %w", err)
	}
	return mapDHCPBinding(row), nil
}

func (r *DHCPBindingRepo) GetByRouterAndMAC(ctx context.Context, routerID, mac string) (*domain.DHCPBinding, error) {
	row, err := r.queries.GetDHCPBindingByRouterAndMAC(ctx, GetDHCPBindingByRouterAndMACParams{
		RouterID: stringToUUID(routerID),
		Lower:    mac,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrDHCPBindingNotFound
		}
		return nil, fmt.Errorf("repository: gagal mengambil dhcp binding by mac: %w", err)
	}
	return mapDHCPBinding(row), nil
}

func (r *DHCPBindingRepo) GetByRouterAndIP(ctx context.Context, routerID, ipAddress string) (*domain.DHCPBinding, error) {
	ip, err := mustIP(ipAddress)
	if err != nil {
		return nil, err
	}
	row, err := r.queries.GetDHCPBindingByRouterAndIP(ctx, GetDHCPBindingByRouterAndIPParams{
		RouterID:  stringToUUID(routerID),
		IpAddress: ip,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrDHCPBindingNotFound
		}
		return nil, fmt.Errorf("repository: gagal mengambil dhcp binding by ip: %w", err)
	}
	return mapDHCPBinding(row), nil
}

func (r *DHCPBindingRepo) Update(ctx context.Context, binding *domain.DHCPBinding) (*domain.DHCPBinding, error) {
	ip, err := mustIP(binding.IPAddress)
	if err != nil {
		return nil, err
	}
	row, err := r.queries.UpdateDHCPBinding(ctx, UpdateDHCPBindingParams{
		ID:            stringToUUID(binding.ID),
		CustomerID:    optionalUUID(binding.CustomerID),
		RouterLeaseID: stringToText(binding.RouterLeaseID),
		Server:        binding.Server,
		MacAddress:    binding.MACAddress,
		IpAddress:     ip,
		HostName:      stringToText(binding.HostName),
		Comment:       binding.Comment,
		Disabled:      binding.Disabled,
		Status:        binding.Status,
		SyncStatus:    binding.SyncStatus,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrDHCPBindingNotFound
		}
		return nil, mapDHCPRepoError("update", err)
	}
	return mapDHCPBinding(row), nil
}

func (r *DHCPBindingRepo) SoftDelete(ctx context.Context, id string) error {
	if err := r.queries.SoftDeleteDHCPBinding(ctx, stringToUUID(id)); err != nil {
		return fmt.Errorf("repository: gagal soft-delete dhcp binding: %w", err)
	}
	return nil
}

func (r *DHCPBindingRepo) List(ctx context.Context, params domain.DHCPBindingListParams) (*domain.DHCPBindingListResult, error) {
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PageSize < 1 {
		params.PageSize = 20
	}
	offset := (params.Page - 1) * params.PageSize
	total, err := r.queries.CountDHCPBindings(ctx, CountDHCPBindingsParams{
		RouterID: stringToUUID(params.RouterID),
		Search:   stringToText(params.Search),
	})
	if err != nil {
		return nil, fmt.Errorf("repository: gagal menghitung dhcp binding: %w", err)
	}
	rows, err := r.queries.ListDHCPBindings(ctx, ListDHCPBindingsParams{
		RouterID: stringToUUID(params.RouterID),
		Limit:    int32(params.PageSize),
		Offset:   int32(offset),
		Search:   stringToText(params.Search),
	})
	if err != nil {
		return nil, fmt.Errorf("repository: gagal list dhcp binding: %w", err)
	}
	items := make([]*domain.DHCPBindingResponse, 0, len(rows))
	for _, row := range rows {
		items = append(items, mapDHCPBinding(row).ToResponse())
	}
	totalPages := int(total) / params.PageSize
	if int(total)%params.PageSize > 0 {
		totalPages++
	}
	return &domain.DHCPBindingListResult{Data: items, Total: total, Page: params.Page, PageSize: params.PageSize, TotalPages: totalPages}, nil
}

func (r *DHCPBindingRepo) UpdateSyncState(ctx context.Context, id, routerLeaseID, syncStatus string, syncAt *time.Time) error {
	if err := r.queries.UpdateDHCPBindingSyncState(ctx, UpdateDHCPBindingSyncStateParams{
		ID:            stringToUUID(id),
		RouterLeaseID: stringToText(routerLeaseID),
		SyncStatus:    syncStatus,
		LastSyncAt:    timePtrToTimestamptz(syncAt),
	}); err != nil {
		return fmt.Errorf("repository: gagal update sync state dhcp binding: %w", err)
	}
	return nil
}

func mapDHCPRepoError(op string, err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrDHCPBindingNotFound
	}
	msg := err.Error()
	if containsAny(msg, "idx_dhcp_bindings_router_mac", "idx_dhcp_bindings_router_ip", "duplicate key") {
		return domain.ErrDHCPBindingExists
	}
	return fmt.Errorf("repository: gagal %s dhcp binding: %w", op, err)
}

func containsAny(value string, needles ...string) bool {
	for _, needle := range needles {
		if needle != "" && strings.Contains(value, needle) {
			return true
		}
	}
	return false
}
