package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// ListTrashed, PermanentDeleteExpired, CountPhotosByNode.

// ListByBounds mengambil daftar map node dengan join data referensi berdasarkan bounding box.
func (r *MapNodeRepo) ListByBounds(ctx context.Context, params domain.MapNodeListParams) ([]*domain.MapNodeWithRef, error) {
	rows, err := r.db.Query(ctx,
		`SELECT mn.id, mn.tenant_id, mn.node_type, mn.reference_id,
			mn.latitude, mn.longitude, mn.custom_fields,
			mn.created_at, mn.updated_at,
			COALESCE(o.name, odp.name, '') AS name,
			COALESCE(o.status, ont.status, '') AS status,
			ont.serial_number, odp.splitter_type,
			odp.capacity, odp.used_ports, odp.address
		 FROM map_nodes mn
		 LEFT JOIN olts o ON mn.node_type = 'olt' AND mn.reference_id = o.id
		 LEFT JOIN odps odp ON mn.node_type = 'odp' AND mn.reference_id = odp.id
		 LEFT JOIN onts ont ON mn.node_type = 'ont' AND mn.reference_id = ont.id
		 WHERE mn.deleted_at IS NULL
		   AND mn.latitude BETWEEN $1 AND $2
		   AND mn.longitude BETWEEN $3 AND $4
		   AND ($5::varchar IS NULL OR mn.node_type = $5::varchar)
		 ORDER BY mn.created_at DESC`,
		params.MinLat, params.MaxLat, params.MinLng, params.MaxLng,
		nilIfEmpty(params.NodeType),
	)
	if err != nil {
		return nil, fmt.Errorf("repository: gagal mengambil map nodes by bounds: %w", err)
	}
	defer rows.Close()

	var results []*domain.MapNodeWithRef
	for rows.Next() {
		var n domain.MapNodeWithRef
		err := rows.Scan(
			&n.ID, &n.TenantID, &n.NodeType, &n.ReferenceID,
			&n.Latitude, &n.Longitude, &n.CustomFields,
			&n.CreatedAt, &n.UpdatedAt,
			&n.Name, &n.Status, &n.SerialNumber, &n.SplitterType,
			&n.Capacity, &n.UsedPorts, &n.Address,
		)
		if err != nil {
			return nil, fmt.Errorf("repository: gagal scan map node with ref: %w", err)
		}
		results = append(results, &n)
	}
	return results, rows.Err()
}

// Pencarian melakukan pencarian full-text di map node dan entitas referensi.
func (r *MapNodeRepo) Search(ctx context.Context, tenantID, query string, limit int) ([]*domain.MapSearchResult, error) {
	rows, err := r.db.Query(ctx,
		`SELECT mn.id, mn.node_type, mn.latitude, mn.longitude,
			COALESCE(o.name, odp.name, '') AS name,
			COALESCE(o.status, ont.status, '') AS status,
			COALESCE(ont.serial_number, '') AS serial_number,
			COALESCE(odp.address, '') AS address
		 FROM map_nodes mn
		 LEFT JOIN olts o ON mn.node_type = 'olt' AND mn.reference_id = o.id
		 LEFT JOIN odps odp ON mn.node_type = 'odp' AND mn.reference_id = odp.id
		 LEFT JOIN onts ont ON mn.node_type = 'ont' AND mn.reference_id = ont.id
		 WHERE mn.deleted_at IS NULL AND mn.tenant_id = $1
		   AND (o.name ILIKE '%' || $2 || '%'
		        OR odp.name ILIKE '%' || $2 || '%'
		        OR odp.address ILIKE '%' || $2 || '%'
		        OR ont.serial_number ILIKE '%' || $2 || '%')
		 ORDER BY mn.created_at DESC LIMIT $3`,
		tenantID, query, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("repository: gagal search map nodes: %w", err)
	}
	defer rows.Close()

	var results []*domain.MapSearchResult
	for rows.Next() {
		var sr domain.MapSearchResult
		var status, serial, address string
		err := rows.Scan(
			&sr.Identifier, &sr.Type, &sr.Latitude, &sr.Longitude,
			&sr.Name, &status, &serial, &address,
		)
		if err != nil {
			return nil, fmt.Errorf("repository: gagal scan search result: %w", err)
		}
		sr.Description = address
		results = append(results, &sr)
	}
	return results, rows.Err()
}

// ListTrashed mengambil daftar map node yang sudah di-hapus lunak.
func (r *MapNodeRepo) ListTrashed(ctx context.Context, tenantID string) ([]*domain.MapNode, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, tenant_id, node_type, reference_id, latitude, longitude,
		   custom_fields, deleted_at, created_at, updated_at
		 FROM map_nodes
		 WHERE deleted_at IS NOT NULL AND tenant_id = $1
		 ORDER BY deleted_at DESC`, tenantID,
	)
	if err != nil {
		return nil, fmt.Errorf("repository: gagal mengambil trashed map nodes: %w", err)
	}
	defer rows.Close()

	var results []*domain.MapNode
	for rows.Next() {
		var n domain.MapNode
		err := rows.Scan(
			&n.ID, &n.TenantID, &n.NodeType, &n.ReferenceID,
			&n.Latitude, &n.Longitude, &n.CustomFields,
			&n.DeletedAt, &n.CreatedAt, &n.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("repository: gagal scan trashed map node: %w", err)
		}
		results = append(results, &n)
	}
	return results, rows.Err()
}

// PermanentDeleteExpired menghapus permanen map node yang deleted_at lebih tua dari olderThan.
func (r *MapNodeRepo) PermanentDeleteExpired(ctx context.Context, olderThan time.Time) (int64, error) {
	tag, err := r.db.Exec(ctx,
		`DELETE FROM map_nodes WHERE deleted_at IS NOT NULL AND deleted_at < $1`, olderThan,
	)
	if err != nil {
		return 0, fmt.Errorf("repository: gagal permanent delete expired map nodes: %w", err)
	}
	return tag.RowsAffected(), nil
}

// CountPhotosByNode menghitung jumlah foto aktif untuk satu node.
func (r *MapNodeRepo) CountPhotosByNode(ctx context.Context, nodeID string) (int, error) {
	var count int
	err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM node_photos WHERE map_node_id = $1 AND deleted_at IS NULL`, nodeID,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("repository: gagal menghitung foto node: %w", err)
	}
	return count, nil
}
