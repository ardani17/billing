// Package usecase berisi implementasi business logic untuk network-service.
// File ini berisi operasi foto untuk MapNodeManager: upload, list, delete.
package usecase

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// UploadPhoto meng-upload foto ke node dengan validasi tipe file dan batas jumlah.
// File disimpan di uploads/{tenant_id}/map-photos/{node_id}/{photo_id}.{ext}.
func (m *mapNodeManager) UploadPhoto(
	ctx context.Context,
	nodeID string,
	file multipart.File,
	header *multipart.FileHeader,
	caption, uploadedBy string,
) (*domain.NodePhotoResponse, error) {
	// Pastikan node ada
	node, err := m.mapNodeRepo.GetByID(ctx, nodeID)
	if err != nil {
		return nil, err
	}

	// Validasi tipe file
	contentType := header.Header.Get("Content-Type")
	if !domain.IsAllowedPhotoType(contentType) {
		return nil, domain.ErrInvalidFileType
	}

	// Cek batas jumlah foto per node
	count, err := m.nodePhotoRepo.CountByNode(ctx, nodeID)
	if err != nil {
		return nil, fmt.Errorf("gagal menghitung foto: %w", err)
	}
	if count >= domain.MaxPhotosPerNode {
		return nil, domain.ErrPhotoLimitReached
	}

	// Generate ID dan path file
	photoID := uuid.New().String()
	ext := extensionFromMIME(contentType)
	filePath := fmt.Sprintf("uploads/%s/map-photos/%s/%s%s", node.TenantID, nodeID, photoID, ext)

	// Buat direktori jika belum ada
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("gagal membuat direktori foto: %w", err)
	}

	// Simpan file ke disk
	dst, err := os.Create(filePath)
	if err != nil {
		return nil, fmt.Errorf("gagal membuat file foto: %w", err)
	}
	defer dst.Close()

	written, err := io.Copy(dst, file)
	if err != nil {
		return nil, fmt.Errorf("gagal menyimpan file foto: %w", err)
	}

	// Buat record foto di database
	var captionPtr *string
	if caption != "" {
		captionPtr = &caption
	}

	photo := &domain.NodePhoto{
		ID:            photoID,
		TenantID:      node.TenantID,
		MapNodeID:     nodeID,
		FilePath:      filePath,
		FileSizeBytes: int(written),
		Caption:       captionPtr,
		UploadedBy:    uploadedBy,
	}

	created, err := m.nodePhotoRepo.Create(ctx, photo)
	if err != nil {
		// Hapus file jika gagal simpan ke DB
		_ = os.Remove(filePath)
		return nil, fmt.Errorf("gagal menyimpan record foto: %w", err)
	}

	// Catat riwayat perubahan
	m.recordHistory(ctx, node.TenantID, nodeID, domain.ChangeActionPhotoAdded, nil, map[string]string{
		"photo_id": photoID,
		"caption":  caption,
	}, uploadedBy)

	return domain.ToNodePhotoResponse(created), nil
}

// ListPhotos mengambil daftar foto aktif untuk satu node.
func (m *mapNodeManager) ListPhotos(ctx context.Context, nodeID string) ([]*domain.NodePhotoResponse, error) {
	photos, err := m.nodePhotoRepo.ListByNode(ctx, nodeID)
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil daftar foto: %w", err)
	}

	responses := make([]*domain.NodePhotoResponse, 0, len(photos))
	for _, p := range photos {
		responses = append(responses, domain.ToNodePhotoResponse(p))
	}

	return responses, nil
}

// DeletePhoto melakukan soft-delete foto dengan pencatatan riwayat.
func (m *mapNodeManager) DeletePhoto(ctx context.Context, nodeID, photoID, performedBy string) error {
	// Pastikan node ada
	node, err := m.mapNodeRepo.GetByID(ctx, nodeID)
	if err != nil {
		return err
	}

	if err := m.nodePhotoRepo.SoftDelete(ctx, photoID); err != nil {
		return fmt.Errorf("gagal menghapus foto: %w", err)
	}

	// Catat riwayat penghapusan foto
	m.recordHistory(ctx, node.TenantID, nodeID, domain.ChangeActionPhotoRemoved, map[string]string{
		"photo_id": photoID,
	}, nil, performedBy)

	return nil
}

// extensionFromMIME mengembalikan ekstensi file berdasarkan MIME type.
func extensionFromMIME(mimeType string) string {
	switch strings.ToLower(mimeType) {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/webp":
		return ".webp"
	default:
		return ".bin"
	}
}
