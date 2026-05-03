// Package usecase berisi implementasi business logic untuk network-service.
// File ini mendefinisikan MapExportManager: export data peta ke berbagai format.
package usecase

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// asyncExportThreshold adalah batas jumlah item untuk export sync.
// Jika total node + cable > threshold, export diproses secara async.
const asyncExportThreshold = 500

// Compile-time check: mapExportManager harus mengimplementasikan domain.MapExportManager.
var _ domain.MapExportManager = (*mapExportManager)(nil)

// mapExportManager mengimplementasikan domain.MapExportManager.
// Mengelola export data peta ke format KML, KMZ, GeoJSON, dan CSV.
// Dataset besar (>500 items) diproses secara async via job ID.
type mapExportManager struct {
	mapNodeRepo    domain.MapNodeRepository
	cableRouteRepo domain.CableRouteRepository
	// jobs menyimpan status export async (in-memory untuk saat ini)
	jobs map[string]*domain.ExportStatus
}

// NewMapExportManager membuat instance MapExportManager baru dengan dependensi repository.
func NewMapExportManager(
	mapNodeRepo domain.MapNodeRepository,
	cableRouteRepo domain.CableRouteRepository,
) domain.MapExportManager {
	return &mapExportManager{
		mapNodeRepo:    mapNodeRepo,
		cableRouteRepo: cableRouteRepo,
		jobs:           make(map[string]*domain.ExportStatus),
	}
}

// Export mengekspor data peta ke format yang diminta.
// Jika dataset ≤500 items → sync generate file.
// Jika dataset >500 items → return async job_id.
func (m *mapExportManager) Export(ctx context.Context, tenantID string, req domain.ExportRequest) (*domain.ExportResult, error) {
	// Validasi format export
	if !domain.IsValidExportFormat(req.Format) {
		return nil, domain.ErrUnsupportedFormat
	}

	// Query semua node dan cable route untuk tenant
	nodes, err := m.queryAllNodes(ctx, tenantID, req.Layers)
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil data node: %w", err)
	}

	cables, err := m.queryAllCables(ctx, tenantID, req.Layers)
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil data cable: %w", err)
	}

	totalItems := len(nodes) + len(cables)

	// Jika dataset besar, buat async job
	if totalItems > asyncExportThreshold {
		jobID := uuid.New().String()
		m.jobs[jobID] = &domain.ExportStatus{
			JobID:  jobID,
			Status: "processing",
		}
		return &domain.ExportResult{
			JobID: jobID,
			Async: true,
		}, nil
	}

	// Generate file secara sync
	fileBytes, fileName, contentType, err := m.generateExport(req.Format, nodes, cables, req.Options)
	if err != nil {
		return nil, fmt.Errorf("gagal generate export: %w", err)
	}

	return &domain.ExportResult{
		FileBytes:   fileBytes,
		FileName:    fileName,
		ContentType: contentType,
		Async:       false,
	}, nil
}

// GetExportStatus mengecek status export async berdasarkan job_id.
func (m *mapExportManager) GetExportStatus(_ context.Context, jobID string) (*domain.ExportStatus, error) {
	status, ok := m.jobs[jobID]
	if !ok {
		return nil, domain.ErrExportNotFound
	}
	return status, nil
}

// generateExport men-dispatch ke formatter yang sesuai berdasarkan format.
func (m *mapExportManager) generateExport(
	format string,
	nodes []*domain.MapNodeWithRef,
	cables []*domain.CableRoute,
	opts domain.ExportOptions,
) ([]byte, string, string, error) {
	switch format {
	case domain.ExportFormatKML:
		data, err := exportKML(nodes, cables, opts)
		return data, "map-export.kml", "application/vnd.google-earth.kml+xml", err
	case domain.ExportFormatKMZ:
		data, err := exportKMZ(nodes, cables, opts)
		return data, "map-export.kmz", "application/vnd.google-earth.kmz", err
	case domain.ExportFormatGeoJSON:
		data, err := exportGeoJSON(nodes, cables)
		return data, "map-export.geojson", "application/geo+json", err
	case domain.ExportFormatCSV:
		data, err := exportCSV(nodes, cables)
		return data, "map-export.csv", "text/csv", err
	default:
		return nil, "", "", domain.ErrUnsupportedFormat
	}
}

// queryAllNodes mengambil semua node untuk tenant berdasarkan layer filter.
// Jika layers mengandung node type spesifik, query per tipe lalu gabungkan.
func (m *mapExportManager) queryAllNodes(ctx context.Context, tenantID string, layers []string) ([]*domain.MapNodeWithRef, error) {
	// Kumpulkan node types yang diminta
	var nodeTypes []string
	for _, layer := range layers {
		if domain.IsValidNodeType(layer) {
			nodeTypes = append(nodeTypes, layer)
		}
	}

	// Jika tidak ada node type spesifik, ambil semua
	if len(nodeTypes) == 0 {
		nodeTypes = domain.ValidNodeTypes
	}

	var allNodes []*domain.MapNodeWithRef
	for _, nt := range nodeTypes {
		params := domain.MapNodeListParams{
			TenantID: tenantID,
			NodeType: nt,
			MinLat:   -90,
			MaxLat:   90,
			MinLng:   -180,
			MaxLng:   180,
		}
		nodes, err := m.mapNodeRepo.ListByBounds(ctx, params)
		if err != nil {
			return nil, err
		}
		allNodes = append(allNodes, nodes...)
	}

	return allNodes, nil
}

// queryAllCables mengambil semua cable route untuk tenant berdasarkan layer filter.
// Jika layers mengandung route type spesifik, query per tipe lalu gabungkan.
func (m *mapExportManager) queryAllCables(ctx context.Context, tenantID string, layers []string) ([]*domain.CableRoute, error) {
	var routeTypes []string
	for _, layer := range layers {
		if domain.IsValidRouteType(layer) {
			routeTypes = append(routeTypes, layer)
		}
	}

	// Jika tidak ada route type spesifik, ambil semua
	if len(routeTypes) == 0 {
		routeTypes = domain.ValidRouteTypes
	}

	var allCables []*domain.CableRoute
	for _, rt := range routeTypes {
		params := domain.CableRouteListParams{
			TenantID:  tenantID,
			RouteType: rt,
			MinLat:    -90,
			MaxLat:    90,
			MinLng:    -180,
			MaxLng:    180,
		}
		cables, err := m.cableRouteRepo.ListByBounds(ctx, params)
		if err != nil {
			return nil, err
		}
		allCables = append(allCables, cables...)
	}

	return allCables, nil
}
