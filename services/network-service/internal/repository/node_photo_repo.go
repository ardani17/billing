package repository

import (
	"context"
	"fmt"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
	"github.com/jackc/pgx/v5"
)

// NodePhotoRepo mengimplementasikan domain.NodePhotoRepository dengan membungkus
// DBTX dan memetakan tipe database ke domain.NodePhoto.
type NodePhotoRepo struct {
	db DBTX
}

// NewNodePhotoRepo membuat instance baru NodePhotoRepo.
func NewNodePhotoRepo(db DBTX) *NodePhotoRepo {
	return &NodePhotoRepo{db: db}
}

// scanNodePhoto memindai satu baris hasil kueri ke domain.NodePhoto.
func scanNodePhoto(row pgx.Row) (*domain.NodePhoto, error) {
	var p domain.NodePhoto
	err := row.Scan(
		&p.ID, &p.TenantID, &p.MapNodeID, &p.FilePath,
		&p.FileSizeBytes, &p.Caption, &p.UploadedBy,
		&p.DeletedAt, &p.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// Buat membuat record foto baru dan mengembalikan foto yang dibuat.
func (r *NodePhotoRepo) Create(ctx context.Context, photo *domain.NodePhoto) (*domain.NodePhoto, error) {
	row := r.db.QueryRow(ctx,
		`INSERT INTO node_photos (tenant_id, map_node_id, file_path, file_size_bytes, caption, uploaded_by)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, tenant_id, map_node_id, file_path, file_size_bytes, caption, uploaded_by, deleted_at, created_at`,
		photo.TenantID, photo.MapNodeID, photo.FilePath,
		photo.FileSizeBytes, photo.Caption, photo.UploadedBy,
	)
	result, err := scanNodePhoto(row)
	if err != nil {
		return nil, fmt.Errorf("repository: gagal membuat node photo: %w", err)
	}
	return result, nil
}

// ListByNode mengambil daftar foto aktif (non-deleted) untuk satu node.
func (r *NodePhotoRepo) ListByNode(ctx context.Context, nodeID string) ([]*domain.NodePhoto, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, tenant_id, map_node_id, file_path, file_size_bytes, caption, uploaded_by, deleted_at, created_at
		 FROM node_photos WHERE map_node_id = $1 AND deleted_at IS NULL
		 ORDER BY created_at DESC`, nodeID,
	)
	if err != nil {
		return nil, fmt.Errorf("repository: gagal mengambil foto node: %w", err)
	}
	defer rows.Close()

	var results []*domain.NodePhoto
	for rows.Next() {
		var p domain.NodePhoto
		err := rows.Scan(
			&p.ID, &p.TenantID, &p.MapNodeID, &p.FilePath,
			&p.FileSizeBytes, &p.Caption, &p.UploadedBy,
			&p.DeletedAt, &p.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("repository: gagal scan node photo: %w", err)
		}
		results = append(results, &p)
	}
	return results, rows.Err()
}

// SoftDelete melakukan hapus lunak foto (atur deleted_at).
func (r *NodePhotoRepo) SoftDelete(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE node_photos SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL`, id,
	)
	if err != nil {
		return fmt.Errorf("repository: gagal soft-delete node photo: %w", err)
	}
	return nil
}

// CountByNode menghitung jumlah foto aktif (non-deleted) untuk satu node.
func (r *NodePhotoRepo) CountByNode(ctx context.Context, nodeID string) (int, error) {
	var count int
	err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM node_photos WHERE map_node_id = $1 AND deleted_at IS NULL`, nodeID,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("repository: gagal menghitung foto node: %w", err)
	}
	return count, nil
}

// Compile-time cek: NodePhotoRepo mengimplementasikan domain.NodePhotoRepository.
var _ domain.NodePhotoRepository = (*NodePhotoRepo)(nil)
