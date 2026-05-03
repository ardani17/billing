// Package usecase berisi implementasi business logic untuk network-service.
// File ini mendefinisikan MapNodeManager: manajemen node peta FTTH.
package usecase

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// Compile-time check: mapNodeManager harus mengimplementasikan domain.MapNodeManager.
var _ domain.MapNodeManager = (*mapNodeManager)(nil)

// mapNodeManager mengimplementasikan domain.MapNodeManager.
// Mengelola business logic CRUD node peta, foto, riwayat, trash, dan label settings.
type mapNodeManager struct {
	mapNodeRepo       domain.MapNodeRepository
	nodePhotoRepo     domain.NodePhotoRepository
	changeHistoryRepo domain.ChangeHistoryRepository
	labelSettingsRepo domain.LabelSettingsRepository
}

// NewMapNodeManager membuat instance MapNodeManager baru dengan dependensi repository.
func NewMapNodeManager(
	mapNodeRepo domain.MapNodeRepository,
	nodePhotoRepo domain.NodePhotoRepository,
	changeHistoryRepo domain.ChangeHistoryRepository,
	labelSettingsRepo domain.LabelSettingsRepository,
) domain.MapNodeManager {
	return &mapNodeManager{
		mapNodeRepo:       mapNodeRepo,
		nodePhotoRepo:     nodePhotoRepo,
		changeHistoryRepo: changeHistoryRepo,
		labelSettingsRepo: labelSettingsRepo,
	}
}

// CreateNode membuat map node baru dengan validasi input dan pencatatan riwayat.
// Validasi: node_type harus valid, koordinat dalam range, reference unik.
func (m *mapNodeManager) CreateNode(ctx context.Context, tenantID string, req domain.CreateMapNodeRequest) (*domain.MapNodeResponse, error) {
	// Validasi node_type
	if !domain.IsValidNodeType(req.NodeType) {
		return nil, domain.ErrInvalidNodeType
	}

	// Validasi koordinat
	if err := domain.ValidateCoordinate(req.Latitude, req.Longitude); err != nil {
		return nil, err
	}

	// Cek duplikasi (tenant_id, node_type, reference_id)
	existing, err := m.mapNodeRepo.GetByReference(ctx, tenantID, req.NodeType, req.ReferenceID)
	if err != nil && err.Error() != domain.ErrMapNodeNotFound.Error() {
		return nil, fmt.Errorf("gagal cek duplikasi node: %w", err)
	}
	if existing != nil {
		return nil, domain.ErrMapNodeDuplicate
	}

	// Buat node baru
	node := &domain.MapNode{
		ID:           uuid.New().String(),
		TenantID:     tenantID,
		NodeType:     req.NodeType,
		ReferenceID:  req.ReferenceID,
		Latitude:     req.Latitude,
		Longitude:    req.Longitude,
		CustomFields: req.CustomFields,
	}

	created, err := m.mapNodeRepo.Create(ctx, node)
	if err != nil {
		return nil, fmt.Errorf("gagal membuat node: %w", err)
	}

	// Catat riwayat perubahan
	m.recordHistory(ctx, created.TenantID, created.ID, domain.ChangeActionCreated, nil, created, "system")

	return domain.ToMapNodeResponse(created), nil
}

// GetNode mengambil detail lengkap map node termasuk foto, riwayat, dan data referensi.
func (m *mapNodeManager) GetNode(ctx context.Context, id string) (*domain.MapNodeDetailResponse, error) {
	node, err := m.mapNodeRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Ambil foto node
	photos, err := m.nodePhotoRepo.ListByNode(ctx, id)
	if err != nil {
		log.Warn().Err(err).Str("node_id", id).Msg("gagal mengambil foto node")
		photos = []*domain.NodePhoto{}
	}

	// Ambil riwayat perubahan (limit 10)
	history, err := m.changeHistoryRepo.ListByNode(ctx, id, 10, 0)
	if err != nil {
		log.Warn().Err(err).Str("node_id", id).Msg("gagal mengambil riwayat node")
		history = []*domain.MapChangeHistory{}
	}

	// Konversi foto ke response
	photoResponses := make([]domain.NodePhotoResponse, 0, len(photos))
	for _, p := range photos {
		photoResponses = append(photoResponses, *domain.ToNodePhotoResponse(p))
	}

	// Konversi riwayat ke response
	historyResponses := make([]domain.MapChangeHistoryResponse, 0, len(history))
	for _, h := range history {
		historyResponses = append(historyResponses, *domain.ToMapChangeHistoryResponse(h))
	}

	// Bangun RefData dari node_type dan reference_id
	refData := map[string]interface{}{
		"node_type":    node.NodeType,
		"reference_id": node.ReferenceID,
	}

	return &domain.MapNodeDetailResponse{
		MapNodeResponse: *domain.ToMapNodeResponse(node),
		Photos:          photoResponses,
		History:         historyResponses,
		RefData:         refData,
	}, nil
}

// UpdateNode memperbarui lokasi dan/atau custom fields node dengan pencatatan riwayat.
func (m *mapNodeManager) UpdateNode(ctx context.Context, id string, req domain.UpdateMapNodeRequest) (*domain.MapNodeResponse, error) {
	existing, err := m.mapNodeRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Simpan nilai lama untuk riwayat
	oldLat := existing.Latitude
	oldLng := existing.Longitude

	// Validasi koordinat baru jika diberikan
	newLat := existing.Latitude
	newLng := existing.Longitude
	if req.Latitude != nil {
		newLat = *req.Latitude
	}
	if req.Longitude != nil {
		newLng = *req.Longitude
	}
	if req.Latitude != nil || req.Longitude != nil {
		if err := domain.ValidateCoordinate(newLat, newLng); err != nil {
			return nil, err
		}
	}

	// Deteksi perubahan untuk riwayat
	locationChanged := newLat != oldLat || newLng != oldLng
	customFieldsChanged := req.CustomFields != nil

	// Update field
	existing.Latitude = newLat
	existing.Longitude = newLng
	if req.CustomFields != nil {
		existing.CustomFields = req.CustomFields
	}

	updated, err := m.mapNodeRepo.Update(ctx, existing)
	if err != nil {
		return nil, fmt.Errorf("gagal memperbarui node: %w", err)
	}

	// Catat riwayat perubahan sesuai field yang berubah
	if locationChanged {
		oldVal := map[string]float64{"latitude": oldLat, "longitude": oldLng}
		newVal := map[string]float64{"latitude": newLat, "longitude": newLng}
		m.recordHistory(ctx, updated.TenantID, updated.ID, domain.ChangeActionLocationMoved, oldVal, newVal, "system")
	}
	if customFieldsChanged {
		m.recordHistory(ctx, updated.TenantID, updated.ID, domain.ChangeActionCustomFieldsUpdated, nil, req.CustomFields, "system")
	}

	return domain.ToMapNodeResponse(updated), nil
}

// recordHistory mencatat entri riwayat perubahan ke tabel map_change_history.
// Error dicatat ke log tapi tidak menggagalkan operasi utama.
func (m *mapNodeManager) recordHistory(ctx context.Context, tenantID, nodeID, action string, oldVal, newVal interface{}, performedBy string) {
	var oldJSON, newJSON json.RawMessage
	if oldVal != nil {
		data, _ := json.Marshal(oldVal)
		oldJSON = data
	}
	if newVal != nil {
		data, _ := json.Marshal(newVal)
		newJSON = data
	}

	entry := &domain.MapChangeHistory{
		ID:          uuid.New().String(),
		TenantID:    tenantID,
		MapNodeID:   nodeID,
		Action:      action,
		OldValue:    oldJSON,
		NewValue:    newJSON,
		PerformedBy: performedBy,
	}

	if _, err := m.changeHistoryRepo.Create(ctx, entry); err != nil {
		log.Error().Err(err).
			Str("node_id", nodeID).
			Str("action", action).
			Msg("gagal mencatat riwayat perubahan")
	}
}
