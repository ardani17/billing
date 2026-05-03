package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
	"github.com/jackc/pgx/v5"
)

// CableRouteRepo mengimplementasikan domain.CableRouteRepository dengan membungkus
// DBTX dan memetakan tipe database ke domain.CableRoute.
type CableRouteRepo struct {
	db DBTX
}

// NewCableRouteRepo membuat instance baru CableRouteRepo.
func NewCableRouteRepo(db DBTX) *CableRouteRepo {
	return &CableRouteRepo{db: db}
}

// scanCableRoute memindai satu baris hasil query ke domain.CableRoute.
func scanCableRoute(row pgx.Row) (*domain.CableRoute, error) {
	var cr domain.CableRoute
	err := row.Scan(
		&cr.ID, &cr.TenantID, &cr.FromNodeID, &cr.ToNodeID, &cr.RouteType,
		&cr.Coordinates, &cr.DistanceMeters, &cr.CoreCount, &cr.Description,
		&cr.DeletedAt, &cr.CreatedAt, &cr.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &cr, nil
}

// scanCableRouteRows memindai banyak baris hasil query ke slice domain.CableRoute.
func scanCableRouteRows(rows pgx.Rows) ([]*domain.CableRoute, error) {
	defer rows.Close()
	var results []*domain.CableRoute
	for rows.Next() {
		var cr domain.CableRoute
		err := rows.Scan(
			&cr.ID, &cr.TenantID, &cr.FromNodeID, &cr.ToNodeID, &cr.RouteType,
			&cr.Coordinates, &cr.DistanceMeters, &cr.CoreCount, &cr.Description,
			&cr.DeletedAt, &cr.CreatedAt, &cr.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		results = append(results, &cr)
	}
	return results, rows.Err()
}

// Create membuat cable route baru dan mengembalikan route yang dibuat.
func (r *CableRouteRepo) Create(ctx context.Context, route *domain.CableRoute) (*domain.CableRoute, error) {
	row := r.db.QueryRow(ctx,
		`INSERT INTO cable_routes (tenant_id, from_node_id, to_node_id, route_type, coordinates, distance_meters, core_count, description)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 RETURNING id, tenant_id, from_node_id, to_node_id, route_type, coordinates, distance_meters, core_count, description, deleted_at, created_at, updated_at`,
		route.TenantID, route.FromNodeID, route.ToNodeID, route.RouteType,
		route.Coordinates, route.DistanceMeters, route.CoreCount, route.Description,
	)
	result, err := scanCableRoute(row)
	if err != nil {
		return nil, fmt.Errorf("repository: gagal membuat cable route: %w", err)
	}
	return result, nil
}

// GetByID mengambil cable route berdasarkan ID (tenant-scoped via RLS).
func (r *CableRouteRepo) GetByID(ctx context.Context, id string) (*domain.CableRoute, error) {
	row := r.db.QueryRow(ctx,
		`SELECT id, tenant_id, from_node_id, to_node_id, route_type, coordinates, distance_meters, core_count, description, deleted_at, created_at, updated_at
		 FROM cable_routes WHERE id = $1 AND deleted_at IS NULL`, id,
	)
	result, err := scanCableRoute(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrCableRouteNotFound
		}
		return nil, fmt.Errorf("repository: gagal mengambil cable route by ID: %w", err)
	}
	return result, nil
}

// Update memperbarui data cable route dan mengembalikan route yang diperbarui.
func (r *CableRouteRepo) Update(ctx context.Context, route *domain.CableRoute) (*domain.CableRoute, error) {
	row := r.db.QueryRow(ctx,
		`UPDATE cable_routes SET route_type = $2, coordinates = $3, distance_meters = $4, core_count = $5, description = $6, updated_at = NOW()
		 WHERE id = $1 AND deleted_at IS NULL
		 RETURNING id, tenant_id, from_node_id, to_node_id, route_type, coordinates, distance_meters, core_count, description, deleted_at, created_at, updated_at`,
		route.ID, route.RouteType, route.Coordinates, route.DistanceMeters,
		route.CoreCount, route.Description,
	)
	result, err := scanCableRoute(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrCableRouteNotFound
		}
		return nil, fmt.Errorf("repository: gagal memperbarui cable route: %w", err)
	}
	return result, nil
}

// SoftDelete melakukan soft-delete cable route (set deleted_at).
func (r *CableRouteRepo) SoftDelete(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE cable_routes SET deleted_at = NOW(), updated_at = NOW() WHERE id = $1 AND deleted_at IS NULL`, id,
	)
	if err != nil {
		return fmt.Errorf("repository: gagal soft-delete cable route: %w", err)
	}
	return nil
}

// ListByBounds mengambil daftar cable route berdasarkan bounding box dan filter opsional.
func (r *CableRouteRepo) ListByBounds(ctx context.Context, params domain.CableRouteListParams) ([]*domain.CableRoute, error) {
	rows, err := r.db.Query(ctx,
		`SELECT cr.id, cr.tenant_id, cr.from_node_id, cr.to_node_id, cr.route_type,
			cr.coordinates, cr.distance_meters, cr.core_count, cr.description,
			cr.deleted_at, cr.created_at, cr.updated_at
		 FROM cable_routes cr
		 JOIN map_nodes fn ON cr.from_node_id = fn.id
		 JOIN map_nodes tn ON cr.to_node_id = tn.id
		 WHERE cr.deleted_at IS NULL
		   AND ((fn.latitude BETWEEN $1 AND $2 AND fn.longitude BETWEEN $3 AND $4)
		        OR (tn.latitude BETWEEN $1 AND $2 AND tn.longitude BETWEEN $3 AND $4))
		   AND ($5::varchar IS NULL OR cr.route_type = $5::varchar)
		 ORDER BY cr.created_at DESC`,
		params.MinLat, params.MaxLat, params.MinLng, params.MaxLng,
		nilIfEmpty(params.RouteType),
	)
	if err != nil {
		return nil, fmt.Errorf("repository: gagal mengambil cable routes by bounds: %w", err)
	}
	results, err := scanCableRouteRows(rows)
	if err != nil {
		return nil, fmt.Errorf("repository: gagal scan cable route rows: %w", err)
	}
	return results, nil
}

// ListByNode mengambil daftar cable route yang terhubung ke node tertentu.
func (r *CableRouteRepo) ListByNode(ctx context.Context, nodeID string) ([]*domain.CableRoute, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, tenant_id, from_node_id, to_node_id, route_type, coordinates, distance_meters, core_count, description, deleted_at, created_at, updated_at
		 FROM cable_routes WHERE (from_node_id = $1 OR to_node_id = $1) AND deleted_at IS NULL
		 ORDER BY created_at DESC`, nodeID,
	)
	if err != nil {
		return nil, fmt.Errorf("repository: gagal mengambil cable routes by node: %w", err)
	}
	results, err := scanCableRouteRows(rows)
	if err != nil {
		return nil, fmt.Errorf("repository: gagal scan cable route rows: %w", err)
	}
	return results, nil
}

// Compile-time check: CableRouteRepo mengimplementasikan domain.CableRouteRepository.
var _ domain.CableRouteRepository = (*CableRouteRepo)(nil)
