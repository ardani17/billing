// Package usecase berisi implementasi business logic untuk network-service.
// File ini mendefinisikan CableRouteManager: manajemen cable route peta FTTH.
package usecase

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// Compile-time check: cableRouteManager harus mengimplementasikan domain.CableRouteManager.
var _ domain.CableRouteManager = (*cableRouteManager)(nil)

// cableRouteManager mengimplementasikan domain.CableRouteManager.
// Mengelola business logic CRUD cable route dengan auto-kalkulasi jarak via Haversine.
type cableRouteManager struct {
	cableRouteRepo domain.CableRouteRepository
	mapNodeRepo    domain.MapNodeRepository
}

// NewCableRouteManager membuat instance CableRouteManager baru dengan dependensi repository.
func NewCableRouteManager(
	cableRouteRepo domain.CableRouteRepository,
	mapNodeRepo domain.MapNodeRepository,
) domain.CableRouteManager {
	return &cableRouteManager{
		cableRouteRepo: cableRouteRepo,
		mapNodeRepo:    mapNodeRepo,
	}
}

// parseCoordinates mem-parse json.RawMessage menjadi [][2]float64.
// Mengembalikan ErrInvalidCoordArray jika format tidak valid atau kurang dari 2 titik.
func parseCoordinates(raw json.RawMessage) ([][2]float64, error) {
	var coords [][2]float64
	if err := json.Unmarshal(raw, &coords); err != nil {
		return nil, fmt.Errorf("%w: format JSON tidak valid", domain.ErrInvalidCoordArray)
	}
	if len(coords) < 2 {
		return nil, fmt.Errorf("%w: minimal 2 titik koordinat diperlukan", domain.ErrInvalidCoordArray)
	}
	return coords, nil
}

// CreateRoute membuat cable route baru dengan validasi node dan kalkulasi jarak otomatis.
// Validasi: from/to node harus ada, route_type valid, koordinat ≥2 titik.
func (m *cableRouteManager) CreateRoute(ctx context.Context, tenantID string, req domain.CreateCableRouteRequest) (*domain.CableRouteResponse, error) {
	// Validasi from_node_id ada
	if _, err := m.mapNodeRepo.GetByID(ctx, req.FromNodeID); err != nil {
		return nil, fmt.Errorf("%w: from_node_id %s", domain.ErrNodeNotFound, req.FromNodeID)
	}

	// Validasi to_node_id ada
	if _, err := m.mapNodeRepo.GetByID(ctx, req.ToNodeID); err != nil {
		return nil, fmt.Errorf("%w: to_node_id %s", domain.ErrNodeNotFound, req.ToNodeID)
	}

	// Validasi route_type
	if !domain.IsValidRouteType(req.RouteType) {
		return nil, domain.ErrInvalidRouteType
	}

	// Parse dan validasi koordinat
	coords, err := parseCoordinates(req.Coordinates)
	if err != nil {
		return nil, err
	}

	// Hitung jarak otomatis dari koordinat via Haversine
	distance := domain.CalculateRouteDistance(coords)

	route := &domain.CableRoute{
		ID:             uuid.New().String(),
		TenantID:       tenantID,
		FromNodeID:     req.FromNodeID,
		ToNodeID:       req.ToNodeID,
		RouteType:      req.RouteType,
		Coordinates:    req.Coordinates,
		DistanceMeters: distance,
		CoreCount:      req.CoreCount,
		Description:    req.Description,
	}

	created, err := m.cableRouteRepo.Create(ctx, route)
	if err != nil {
		return nil, fmt.Errorf("gagal membuat cable route: %w", err)
	}

	return domain.ToCableRouteResponse(created), nil
}

// GetRoute mengambil detail cable route berdasarkan ID.
func (m *cableRouteManager) GetRoute(ctx context.Context, id string) (*domain.CableRouteResponse, error) {
	route, err := m.cableRouteRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return domain.ToCableRouteResponse(route), nil
}

// UpdateRoute memperbarui cable route dengan kalkulasi ulang jarak jika koordinat berubah.
// Hanya field yang dikirim yang akan diupdate.
func (m *cableRouteManager) UpdateRoute(ctx context.Context, id string, req domain.UpdateCableRouteRequest) (*domain.CableRouteResponse, error) {
	existing, err := m.cableRouteRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Update koordinat dan hitung ulang jarak jika diberikan
	if req.Coordinates != nil {
		coords, err := parseCoordinates(req.Coordinates)
		if err != nil {
			return nil, err
		}
		existing.Coordinates = req.Coordinates
		existing.DistanceMeters = domain.CalculateRouteDistance(coords)
	}

	// Update core_count jika diberikan
	if req.CoreCount != nil {
		existing.CoreCount = req.CoreCount
	}

	// Update description jika diberikan
	if req.Description != nil {
		existing.Description = req.Description
	}

	updated, err := m.cableRouteRepo.Update(ctx, existing)
	if err != nil {
		return nil, fmt.Errorf("gagal memperbarui cable route: %w", err)
	}

	return domain.ToCableRouteResponse(updated), nil
}

// DeleteRoute melakukan soft-delete cable route.
func (m *cableRouteManager) DeleteRoute(ctx context.Context, id string) error {
	if err := m.cableRouteRepo.SoftDelete(ctx, id); err != nil {
		return fmt.Errorf("gagal menghapus cable route: %w", err)
	}
	return nil
}

// ListRoutes mengambil daftar cable route berdasarkan bounding box dan filter.
func (m *cableRouteManager) ListRoutes(ctx context.Context, params domain.CableRouteListParams) ([]*domain.CableRouteResponse, error) {
	routes, err := m.cableRouteRepo.ListByBounds(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil daftar cable route: %w", err)
	}

	responses := make([]*domain.CableRouteResponse, 0, len(routes))
	for _, r := range routes {
		responses = append(responses, domain.ToCableRouteResponse(r))
	}

	return responses, nil
}
