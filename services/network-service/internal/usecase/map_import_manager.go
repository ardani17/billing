// Package usecase berisi implementasi business logic untuk network-service.
// File ini mendefinisikan MapImportManager: import data peta dari file KML/KMZ/GeoJSON.
package usecase

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"mime/multipart"
	"path/filepath"
	"strings"

	"github.com/google/uuid"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// Compile-time check: mapImportManager harus mengimplementasikan domain.MapImportManager.
var _ domain.MapImportManager = (*mapImportManager)(nil)

// mapImportManager mengimplementasikan domain.MapImportManager.
// Mengelola import data peta dari file KML, KMZ, dan GeoJSON.
type mapImportManager struct {
	mapNodeRepo    domain.MapNodeRepository
	cableRouteRepo domain.CableRouteRepository
	// previews menyimpan data preview import (in-memory)
	previews map[string]*importPreviewData
}

// importPreviewData menyimpan data preview import untuk eksekusi nanti.
type importPreviewData struct {
	tenantID string
	items    []domain.ImportPreviewItem
}

// NewMapImportManager membuat instance MapImportManager baru dengan dependensi repository.
func NewMapImportManager(
	mapNodeRepo domain.MapNodeRepository,
	cableRouteRepo domain.CableRouteRepository,
) domain.MapImportManager {
	return &mapImportManager{
		mapNodeRepo:    mapNodeRepo,
		cableRouteRepo: cableRouteRepo,
		previews:       make(map[string]*importPreviewData),
	}
}

// Preview mem-parse file import dan mengembalikan preview item yang terdeteksi.
// Mendukung format KML, KMZ, dan GeoJSON berdasarkan ekstensi file.
func (m *mapImportManager) Preview(_ context.Context, tenantID string, file multipart.File, filename string) (*domain.ImportPreview, error) {
	// Baca seluruh file ke memory
	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("gagal membaca file import: %w", err)
	}

	// Deteksi format berdasarkan ekstensi
	ext := strings.ToLower(filepath.Ext(filename))
	var items []domain.ImportPreviewItem

	switch ext {
	case ".kml":
		items, err = parseKMLImport(data)
	case ".kmz":
		items, err = parseKMZImport(data)
	case ".geojson", ".json":
		items, err = parseGeoJSONImport(data)
	default:
		return nil, domain.ErrUnsupportedFormat
	}

	if err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrInvalidImportFile, err)
	}

	// Hitung statistik
	importID := uuid.New().String()
	points, lines, polygons := countItemTypes(items)

	// Simpan preview untuk eksekusi nanti
	m.previews[importID] = &importPreviewData{
		tenantID: tenantID,
		items:    items,
	}

	return &domain.ImportPreview{
		ImportID: importID,
		FileName: filename,
		Points:   points,
		Lines:    lines,
		Polygons: polygons,
		Items:    items,
	}, nil
}

// Execute mengeksekusi import berdasarkan mapping yang dipilih user.
// Validasi koordinat dan insert node/cable ke database.
func (m *mapImportManager) Execute(ctx context.Context, importID string, mapping domain.ImportMapping) (*domain.ImportSummary, error) {
	preview, ok := m.previews[importID]
	if !ok {
		return nil, domain.ErrImportNotFound
	}

	summary := &domain.ImportSummary{}
	for _, item := range preview.items {
		summary.Total++

		// Tentukan tipe node berdasarkan mapping
		nodeType, mapped := mapping.TypeMapping[item.Type]
		if !mapped {
			summary.Skipped++
			continue
		}

		// Validasi koordinat untuk item bertipe point
		if item.Type == "point" && item.Lat != nil && item.Lng != nil {
			if err := domain.ValidateCoordinate(*item.Lat, *item.Lng); err != nil {
				summary.Errors++
				summary.ErrorDetails = append(summary.ErrorDetails, domain.ImportError{
					ItemName: item.Name,
					Reason:   err.Error(),
				})
				continue
			}

			// Insert node
			node := &domain.MapNode{
				ID:          uuid.New().String(),
				TenantID:    preview.tenantID,
				NodeType:    nodeType,
				ReferenceID: uuid.New().String(),
				Latitude:    *item.Lat,
				Longitude:   *item.Lng,
			}

			if _, err := m.mapNodeRepo.Create(ctx, node); err != nil {
				summary.Errors++
				summary.ErrorDetails = append(summary.ErrorDetails, domain.ImportError{
					ItemName: item.Name,
					Reason:   fmt.Sprintf("gagal insert: %v", err),
				})
				continue
			}
			summary.Success++
		} else {
			summary.Skipped++
		}
	}

	// Hapus preview setelah eksekusi
	delete(m.previews, importID)

	return summary, nil
}

// GetImportStatus mengecek status import async berdasarkan job_id.
func (m *mapImportManager) GetImportStatus(_ context.Context, jobID string) (*domain.ImportStatus, error) {
	// Cek apakah preview masih ada (berarti belum dieksekusi)
	if _, ok := m.previews[jobID]; ok {
		return &domain.ImportStatus{
			JobID:  jobID,
			Status: "pending",
		}, nil
	}
	return nil, domain.ErrImportNotFound
}

// =============================================================================
// Parser KML — parse file KML menjadi ImportPreviewItem
// =============================================================================

// kmlImportDoc adalah struktur untuk parsing KML import.
type kmlImportDoc struct {
	XMLName  xml.Name         `xml:"kml"`
	Document kmlImportDocBody `xml:"Document"`
}

// kmlImportDocBody berisi folder dan placemark di dokumen KML.
type kmlImportDocBody struct {
	Folders    []kmlImportFolder    `xml:"Folder"`
	Placemarks []kmlImportPlacemark `xml:"Placemark"`
}

// kmlImportFolder berisi placemark dalam satu folder KML.
type kmlImportFolder struct {
	Name       string               `xml:"name"`
	Placemarks []kmlImportPlacemark `xml:"Placemark"`
}

// kmlImportPlacemark adalah satu placemark di file KML.
type kmlImportPlacemark struct {
	Name       string          `xml:"name"`
	Point      *kmlImportPoint `xml:"Point"`
	LineString *kmlImportLine  `xml:"LineString"`
}

// kmlImportPoint berisi koordinat titik dari KML.
type kmlImportPoint struct {
	Coordinates string `xml:"coordinates"`
}

// kmlImportLine berisi koordinat garis dari KML.
type kmlImportLine struct {
	Coordinates string `xml:"coordinates"`
}

// parseKMLImport mem-parse data KML menjadi daftar ImportPreviewItem.
func parseKMLImport(data []byte) ([]domain.ImportPreviewItem, error) {
	var doc kmlImportDoc
	if err := xml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("format KML tidak valid: %w", err)
	}

	var items []domain.ImportPreviewItem

	// Parse placemark di dalam folder terlebih dahulu
	for _, folder := range doc.Document.Folders {
		items = append(items, extractPlacemarks(folder.Placemarks)...)
	}

	// Parse placemark di root document
	items = append(items, extractPlacemarks(doc.Document.Placemarks)...)

	return items, nil
}

// extractPlacemarks mengekstrak ImportPreviewItem dari daftar placemark KML.
func extractPlacemarks(placemarks []kmlImportPlacemark) []domain.ImportPreviewItem {
	var items []domain.ImportPreviewItem
	for _, pm := range placemarks {
		if pm.Point != nil {
			lat, lng := parseKMLCoordinate(pm.Point.Coordinates)
			items = append(items, domain.ImportPreviewItem{
				Name: pm.Name,
				Type: "point",
				Lat:  &lat,
				Lng:  &lng,
			})
		} else if pm.LineString != nil {
			items = append(items, domain.ImportPreviewItem{
				Name: pm.Name,
				Type: "line",
			})
		}
	}
	return items
}

// parseKMLCoordinate mem-parse string koordinat KML "lng,lat,alt" menjadi lat, lng.
func parseKMLCoordinate(coordStr string) (float64, float64) {
	coordStr = strings.TrimSpace(coordStr)
	var lng, lat, alt float64
	fmt.Sscanf(coordStr, "%f,%f,%f", &lng, &lat, &alt)
	return lat, lng
}

// =============================================================================
// Parser KMZ — extract KML dari arsip ZIP
// =============================================================================

// parseKMZImport mem-parse data KMZ (ZIP berisi KML) menjadi ImportPreviewItem.
func parseKMZImport(data []byte) ([]domain.ImportPreviewItem, error) {
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("format KMZ tidak valid: %w", err)
	}

	// Cari file KML di dalam arsip
	for _, f := range reader.File {
		if strings.HasSuffix(strings.ToLower(f.Name), ".kml") {
			rc, err := f.Open()
			if err != nil {
				return nil, fmt.Errorf("gagal membuka KML di KMZ: %w", err)
			}
			defer rc.Close()

			kmlData, err := io.ReadAll(rc)
			if err != nil {
				return nil, fmt.Errorf("gagal membaca KML di KMZ: %w", err)
			}

			return parseKMLImport(kmlData)
		}
	}

	return nil, fmt.Errorf("tidak ditemukan file KML di dalam KMZ")
}

// =============================================================================
// Parser GeoJSON — parse file GeoJSON menjadi ImportPreviewItem
// =============================================================================

// parseGeoJSONImport mem-parse data GeoJSON menjadi daftar ImportPreviewItem.
func parseGeoJSONImport(data []byte) ([]domain.ImportPreviewItem, error) {
	var collection geoJSONCollection
	if err := json.Unmarshal(data, &collection); err != nil {
		return nil, fmt.Errorf("format GeoJSON tidak valid: %w", err)
	}

	var items []domain.ImportPreviewItem
	for _, f := range collection.Features {
		name := ""
		if n, ok := f.Properties["name"]; ok {
			name = fmt.Sprintf("%v", n)
		}

		switch f.Geometry.Type {
		case "Point":
			lat, lng := extractPointCoords(f.Geometry.Coordinates)
			items = append(items, domain.ImportPreviewItem{
				Name: name,
				Type: "point",
				Lat:  &lat,
				Lng:  &lng,
			})
		case "LineString":
			items = append(items, domain.ImportPreviewItem{
				Name: name,
				Type: "line",
			})
		}
	}

	return items, nil
}

// extractPointCoords mengekstrak lat, lng dari koordinat GeoJSON Point.
// GeoJSON menggunakan format [lng, lat].
func extractPointCoords(coords interface{}) (float64, float64) {
	switch c := coords.(type) {
	case []interface{}:
		if len(c) >= 2 {
			lng, _ := toFloat64(c[0])
			lat, _ := toFloat64(c[1])
			return lat, lng
		}
	}
	return 0, 0
}

// toFloat64 mengkonversi interface{} ke float64.
func toFloat64(v interface{}) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case int:
		return float64(n), true
	case json.Number:
		f, err := n.Float64()
		return f, err == nil
	}
	return 0, false
}

// countItemTypes menghitung jumlah item per tipe (point, line, polygon).
func countItemTypes(items []domain.ImportPreviewItem) (points, lines, polygons int) {
	for _, item := range items {
		switch item.Type {
		case "point":
			points++
		case "line":
			lines++
		case "polygon":
			polygons++
		}
	}
	return
}
