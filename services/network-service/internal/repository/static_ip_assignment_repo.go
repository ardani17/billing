package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

type StaticIPAssignmentRepo struct {
	queries *Queries
}

func NewStaticIPAssignmentRepo(queries *Queries) *StaticIPAssignmentRepo {
	return &StaticIPAssignmentRepo{queries: queries}
}

func mapStaticIPAssignment(row StaticIpAssignment) *domain.StaticIPAssignment {
	return &domain.StaticIPAssignment{
		ID:          uuidToString(row.ID),
		TenantID:    uuidToString(row.TenantID),
		RouterID:    uuidToString(row.RouterID),
		CustomerID:  uuidToString(row.CustomerID),
		IPAddress:   row.IpAddress.String(),
		AddressList: row.AddressList,
		QueueName:   textToString(row.QueueName),
		RateLimit:   textToString(row.RateLimit),
		Comment:     row.Comment,
		Status:      row.Status,
		LastSyncAt:  timestamptzToTimePtr(row.LastSyncAt),
		SyncStatus:  row.SyncStatus,
		CreatedAt:   timestamptzToTime(row.CreatedAt),
		UpdatedAt:   timestamptzToTime(row.UpdatedAt),
		DeletedAt:   timestamptzToTimePtr(row.DeletedAt),
	}
}

func (r *StaticIPAssignmentRepo) Create(ctx context.Context, assignment *domain.StaticIPAssignment) (*domain.StaticIPAssignment, error) {
	ip, err := mustIP(assignment.IPAddress)
	if err != nil {
		return nil, err
	}
	row, err := r.queries.CreateStaticIPAssignment(ctx, CreateStaticIPAssignmentParams{
		TenantID:    stringToUUID(assignment.TenantID),
		RouterID:    stringToUUID(assignment.RouterID),
		CustomerID:  optionalUUID(assignment.CustomerID),
		IpAddress:   ip,
		AddressList: assignment.AddressList,
		QueueName:   stringToText(assignment.QueueName),
		RateLimit:   stringToText(assignment.RateLimit),
		Comment:     assignment.Comment,
		Status:      assignment.Status,
		SyncStatus:  assignment.SyncStatus,
	})
	if err != nil {
		return nil, mapStaticIPRepoError("create", err)
	}
	return mapStaticIPAssignment(row), nil
}

func (r *StaticIPAssignmentRepo) GetByID(ctx context.Context, id string) (*domain.StaticIPAssignment, error) {
	row, err := r.queries.GetStaticIPAssignmentByID(ctx, stringToUUID(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrStaticIPAssignmentNotFound
		}
		return nil, fmt.Errorf("repository: gagal mengambil static ip assignment: %w", err)
	}
	return mapStaticIPAssignment(row), nil
}

func (r *StaticIPAssignmentRepo) GetByRouterAndIP(ctx context.Context, routerID, ipAddress string) (*domain.StaticIPAssignment, error) {
	ip, err := mustIP(ipAddress)
	if err != nil {
		return nil, err
	}
	row, err := r.queries.GetStaticIPAssignmentByRouterAndIP(ctx, GetStaticIPAssignmentByRouterAndIPParams{
		RouterID:  stringToUUID(routerID),
		IpAddress: ip,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrStaticIPAssignmentNotFound
		}
		return nil, fmt.Errorf("repository: gagal mengambil static ip by ip: %w", err)
	}
	return mapStaticIPAssignment(row), nil
}

func (r *StaticIPAssignmentRepo) Update(ctx context.Context, assignment *domain.StaticIPAssignment) (*domain.StaticIPAssignment, error) {
	ip, err := mustIP(assignment.IPAddress)
	if err != nil {
		return nil, err
	}
	row, err := r.queries.UpdateStaticIPAssignment(ctx, UpdateStaticIPAssignmentParams{
		ID:          stringToUUID(assignment.ID),
		CustomerID:  optionalUUID(assignment.CustomerID),
		IpAddress:   ip,
		AddressList: assignment.AddressList,
		QueueName:   stringToText(assignment.QueueName),
		RateLimit:   stringToText(assignment.RateLimit),
		Comment:     assignment.Comment,
		Status:      assignment.Status,
		SyncStatus:  assignment.SyncStatus,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrStaticIPAssignmentNotFound
		}
		return nil, mapStaticIPRepoError("update", err)
	}
	return mapStaticIPAssignment(row), nil
}

func (r *StaticIPAssignmentRepo) SoftDelete(ctx context.Context, id string) error {
	if err := r.queries.SoftDeleteStaticIPAssignment(ctx, stringToUUID(id)); err != nil {
		return fmt.Errorf("repository: gagal soft-delete static ip assignment: %w", err)
	}
	return nil
}

func (r *StaticIPAssignmentRepo) List(ctx context.Context, params domain.StaticIPAssignmentListParams) (*domain.StaticIPAssignmentListResult, error) {
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PageSize < 1 {
		params.PageSize = 20
	}
	offset := (params.Page - 1) * params.PageSize
	total, err := r.queries.CountStaticIPAssignments(ctx, CountStaticIPAssignmentsParams{
		RouterID: stringToUUID(params.RouterID),
		Search:   stringToText(params.Search),
	})
	if err != nil {
		return nil, fmt.Errorf("repository: gagal menghitung static ip assignment: %w", err)
	}
	rows, err := r.queries.ListStaticIPAssignments(ctx, ListStaticIPAssignmentsParams{
		RouterID: stringToUUID(params.RouterID),
		Limit:    int32(params.PageSize),
		Offset:   int32(offset),
		Search:   stringToText(params.Search),
	})
	if err != nil {
		return nil, fmt.Errorf("repository: gagal list static ip assignment: %w", err)
	}
	items := make([]*domain.StaticIPAssignmentResponse, 0, len(rows))
	for _, row := range rows {
		items = append(items, mapStaticIPAssignment(row).ToResponse())
	}
	totalPages := int(total) / params.PageSize
	if int(total)%params.PageSize > 0 {
		totalPages++
	}
	return &domain.StaticIPAssignmentListResult{Data: items, Total: total, Page: params.Page, PageSize: params.PageSize, TotalPages: totalPages}, nil
}

func (r *StaticIPAssignmentRepo) UpdateSyncState(ctx context.Context, id, syncStatus string, syncAt *time.Time) error {
	if err := r.queries.UpdateStaticIPAssignmentSyncState(ctx, UpdateStaticIPAssignmentSyncStateParams{
		ID:         stringToUUID(id),
		SyncStatus: syncStatus,
		LastSyncAt: timePtrToTimestamptz(syncAt),
	}); err != nil {
		return fmt.Errorf("repository: gagal update sync state static ip: %w", err)
	}
	return nil
}

func mapStaticIPRepoError(op string, err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrStaticIPAssignmentNotFound
	}
	if strings.Contains(err.Error(), "idx_static_ip_assignments_router_ip") || strings.Contains(err.Error(), "duplicate key") {
		return domain.ErrStaticIPAssignmentExists
	}
	return fmt.Errorf("repository: gagal %s static ip assignment: %w", op, err)
}
