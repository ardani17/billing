package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
	"github.com/jackc/pgx/v5"
)

// MapNodeRepo mengimplementasikan domain.MapNodeRepository dengan membungkus
// DBTX dan memetakan tipe database ke domain.MapNode.
type MapNodeRepo struct {
	db DBTX
}

// NewMapNodeRepo membuat instance baru MapNodeRepo.
func NewMapNodeRepo(db DBTX) *MapNodeRepo {
	return &MapNodeRepo{db: db}
}

// scanMapNode memindai satu baris hasil query ke domain.MapNode.
func scanMapNode(row pgx.Row) (*domain.MapNode, error) {
	var n domain.MapNode
	err := row.Scan(
		&n.ID, &n.TenantID, &n.NodeType, &n.ReferenceID,
		&n.Latitude, &n.Longitude, &n.CustomFields,
		&n.DeletedAt, &n.CreatedAt, &n.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &n, nil
}

// Create membuat map node baru dan mengembalikan node yang dibuat.
func (r *MapNodeRepo) Create(ctx context.Context, node *domain.MapNode) (*domain.MapNode, error) {
	row := r.db.QueryRow(ctx,
		`INSERT INTO map_nodes (tenant_id, node_type, reference_id, latitude, longitude, custom_fields)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, tenant_id, node_type, reference_id, latitude, longitude,
		   custom_fields, deleted_at, created_at, updated_at`,
		node.TenantID, node.NodeType, node.ReferenceID,
		node.Latitude, node.Longitude, node.CustomFields,
	)
	result, err := scanMapNode(row)
	if err != nil {
		return nil, fmt.Errorf("repository: gagal membuat map node: %w", err)
	}
	return result, nil
}

// GetByID mengambil map node berdasarkan ID (tenant-scoped via RLS).
func (r *MapNodeRepo) GetByID(ctx context.Context, id string) (*domain.MapNode, error) {
	row := r.db.QueryRow(ctx,
		`SELECT id, tenant_id, node_type, reference_id, latitude, longitude,
		   custom_fields, deleted_at, created_at, updated_at
		 FROM map_nodes WHERE id = $1 AND deleted_at IS NULL`, id,
	)
	result, err := scanMapNode(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrMapNodeNotFound
		}
		return nil, fmt.Errorf("repository: gagal mengambil map node by ID: %w", err)
	}
	return result, nil
}

// Update memperbarui data map node dan mengembalikan node yang diperbarui.
func (r *MapNodeRepo) Update(ctx context.Context, node *domain.MapNode) (*domain.MapNode, error) {
	row := r.db.QueryRow(ctx,
		`UPDATE map_nodes SET latitude = $2, longitude = $3, custom_fields = $4, updated_at = NOW()
		 WHERE id = $1 AND deleted_at IS NULL
		 RETURNING id, tenant_id, node_type, reference_id, latitude, longitude,
		   custom_fields, deleted_at, created_at, updated_at`,
		node.ID, node.Latitude, node.Longitude, node.CustomFields,
	)
	result, err := scanMapNode(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrMapNodeNotFound
		}
		return nil, fmt.Errorf("repository: gagal memperbarui map node: %w", err)
	}
	return result, nil
}

// SoftDelete melakukan soft-delete map node (set deleted_at).
func (r *MapNodeRepo) SoftDelete(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE map_nodes SET deleted_at = NOW(), updated_at = NOW()
		 WHERE id = $1 AND deleted_at IS NULL`, id,
	)
	if err != nil {
		return fmt.Errorf("repository: gagal soft-delete map node: %w", err)
	}
	return nil
}

// Restore mengembalikan map node yang sudah di-soft-delete (clear deleted_at).
func (r *MapNodeRepo) Restore(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE map_nodes SET deleted_at = NULL, updated_at = NOW()
		 WHERE id = $1 AND deleted_at IS NOT NULL`, id,
	)
	if err != nil {
		return fmt.Errorf("repository: gagal restore map node: %w", err)
	}
	return nil
}

// GetByReference mengambil map node berdasarkan tenant_id, node_type, dan reference_id.
// Digunakan untuk cek duplikasi sebelum create.
func (r *MapNodeRepo) GetByReference(ctx context.Context, tenantID, nodeType, referenceID string) (*domain.MapNode, error) {
	row := r.db.QueryRow(ctx,
		`SELECT id, tenant_id, node_type, reference_id, latitude, longitude,
		   custom_fields, deleted_at, created_at, updated_at
		 FROM map_nodes
		 WHERE tenant_id = $1 AND node_type = $2 AND reference_id = $3
		   AND deleted_at IS NULL`,
		tenantID, nodeType, referenceID,
	)
	result, err := scanMapNode(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrMapNodeNotFound
		}
		return nil, fmt.Errorf("repository: gagal mengambil map node by reference: %w", err)
	}
	return result, nil
}

// nilIfEmpty mengembalikan nil jika string kosong, digunakan untuk filter opsional.
func nilIfEmpty(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

// Compile-time check: MapNodeRepo mengimplementasikan domain.MapNodeRepository.
var _ domain.MapNodeRepository = (*MapNodeRepo)(nil)
