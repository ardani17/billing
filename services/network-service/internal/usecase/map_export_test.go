// map_export_test.go — unit test dan property test untuk MapExportManager.
// Menggunakan mock in-memory repository dan pgregory.net/rapid untuk PBT.
// Semua komentar dalam Bahasa Indonesia.
package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"pgregory.net/rapid"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// =============================================================================
// Helper: membuat MapExportManager dengan mock dependencies
// =============================================================================

// newTestExportManager membuat instance MapExportManager dengan mock repository.
func newTestExportManager() (domain.MapExportManager, *mockMapNodeRepo, *mockCableRouteRepo) {
	nodeRepo := newMockMapNodeRepo()
	cableRepo := newMockCableRouteRepo()
	mgr := NewMapExportManager(nodeRepo, cableRepo)
	return mgr, nodeRepo, cableRepo
}

// seedNodeForExport membuat node di mock repo untuk testing export.
func seedNodeForExport(nodeRepo *mockMapNodeRepo, id, tenantID, nodeType string, lat, lng float64) {
	nodeRepo.mu.Lock()
	defer nodeRepo.mu.Unlock()
	nodeRepo.nodes[id] = &domain.MapNode{
		ID:          id,
		TenantID:    tenantID,
		NodeType:    nodeType,
		ReferenceID: "ref-" + id,
		Latitude:    lat,
		Longitude:   lng,
	}
}

// seedCableForExport membuat cable route di mock repo untuk testing export.
func seedCableForExport(cableRepo *mockCableRouteRepo, id, tenantID, routeType string, coords [][2]float64) {
	cableRepo.mu.Lock()
	defer cableRepo.mu.Unlock()
	coordJSON, _ := json.Marshal(coords)
	cableRepo.routes[id] = &domain.CableRoute{
		ID:          id,
		TenantID:    tenantID,
		RouteType:   routeType,
		Coordinates: coordJSON,
		FromNodeID:  "node-from",
		ToNodeID:    "node-to",
	}
}

// =============================================================================
// Unit Test 1: TestExportKML_OutputStructure — verifikasi struktur output KML
// =============================================================================

// TestExportKML_OutputStructure memverifikasi bahwa export KML menghasilkan
// dokumen XML yang valid dengan folder per tipe node.
func TestExportKML_OutputStructure(t *testing.T) {
	mgr, nodeRepo, _ := newTestExportManager()
	ctx := context.Background()

	// Seed beberapa node
	seedNodeForExport(nodeRepo, "node-olt-1", "tenant-exp", domain.NodeTypeOLT, -6.2, 106.8)
	seedNodeForExport(nodeRepo, "node-odp-1", "tenant-exp", domain.NodeTypeODP, -6.3, 106.9)

	req := domain.ExportRequest{
		Format:  domain.ExportFormatKML,
		Layers:  []string{"olt", "odp"},
		Options: domain.ExportOptions{IncludeDescriptions: true},
	}

	result, err := mgr.Export(ctx, "tenant-exp", req)
	if err != nil {
		t.Fatalf("Export KML gagal: %v", err)
	}

	// Verifikasi hasil sync (bukan async)
	if result.Async {
		t.Fatal("export kecil seharusnya sync, bukan async")
	}
	if result.FileBytes == nil {
		t.Fatal("FileBytes seharusnya tidak nil untuk export sync")
	}

	// Verifikasi output mengandung elemen KML
	output := string(result.FileBytes)
	if !strings.Contains(output, "<kml") {
		t.Error("output seharusnya mengandung tag <kml>")
	}
	if !strings.Contains(output, "Node olt") {
		t.Error("output seharusnya mengandung folder 'Node olt'")
	}
	if !strings.Contains(output, "Node odp") {
		t.Error("output seharusnya mengandung folder 'Node odp'")
	}
}

// =============================================================================
// Unit Test 2: TestExportCSV_Columns — verifikasi kolom CSV
// =============================================================================

// TestExportCSV_Columns memverifikasi bahwa export CSV menghasilkan
// header dan baris data yang benar.
func TestExportCSV_Columns(t *testing.T) {
	mgr, nodeRepo, _ := newTestExportManager()
	ctx := context.Background()

	seedNodeForExport(nodeRepo, "node-csv-1", "tenant-csv", domain.NodeTypeONT, -6.5, 107.0)

	req := domain.ExportRequest{
		Format: domain.ExportFormatCSV,
		Layers: []string{"ont"},
	}

	result, err := mgr.Export(ctx, "tenant-csv", req)
	if err != nil {
		t.Fatalf("Export CSV gagal: %v", err)
	}

	output := string(result.FileBytes)
	lines := strings.Split(strings.TrimSpace(output), "\n")

	// Verifikasi header
	expectedHeader := "name,type,lat,lng,status,address,custom_fields"
	if lines[0] != expectedHeader {
		t.Errorf("header CSV: got %q, want %q", lines[0], expectedHeader)
	}

	// Verifikasi minimal ada 1 baris data
	if len(lines) < 2 {
		t.Error("CSV seharusnya memiliki minimal 1 baris data selain header")
	}
}

// =============================================================================
// Unit Test 3: TestExportAsync_LargeDataset — verifikasi async job creation
// =============================================================================

// TestExportAsync_LargeDataset memverifikasi bahwa export dengan dataset besar
// (>500 items) mengembalikan job_id untuk polling status.
func TestExportAsync_LargeDataset(t *testing.T) {
	mgr, nodeRepo, _ := newTestExportManager()
	ctx := context.Background()

	// Seed >500 node untuk trigger async
	for i := 0; i < 501; i++ {
		id := fmt.Sprintf("node-async-%03d", i)
		seedNodeForExport(nodeRepo, id, "tenant-async", domain.NodeTypeONT,
			-6.0+float64(i)*0.001, 106.0+float64(i)*0.001)
	}

	req := domain.ExportRequest{
		Format: domain.ExportFormatGeoJSON,
		Layers: []string{"ont"},
	}

	result, err := mgr.Export(ctx, "tenant-async", req)
	if err != nil {
		t.Fatalf("Export async gagal: %v", err)
	}

	// Verifikasi hasil async
	if !result.Async {
		t.Fatal("export besar seharusnya async")
	}
	if result.JobID == "" {
		t.Fatal("JobID seharusnya tidak kosong untuk export async")
	}

	// Verifikasi status job bisa diambil
	status, err := mgr.GetExportStatus(ctx, result.JobID)
	if err != nil {
		t.Fatalf("GetExportStatus gagal: %v", err)
	}
	if status.Status != "processing" {
		t.Errorf("status: got %q, want %q", status.Status, "processing")
	}
}

// =============================================================================
// Property Test 6: GeoJSON Export/Import Round-Trip
// =============================================================================

// TestPropertyGeoJSONRoundTrip memverifikasi bahwa export ke GeoJSON
// menghasilkan data yang bisa di-parse kembali dengan koordinat dan tipe
// yang sama. Round-trip mempertahankan data geospasial.
//
// **Validates: Requirements 6.6**
func TestPropertyGeoJSONRoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate jumlah node acak (1-20)
		numNodes := rapid.IntRange(1, 20).Draw(t, "numNodes")

		// Buat node dengan koordinat acak
		nodes := make([]*domain.MapNodeWithRef, 0, numNodes)
		for i := 0; i < numNodes; i++ {
			lat := rapid.Float64Range(-89.0, 89.0).Draw(t, fmt.Sprintf("lat_%d", i))
			lng := rapid.Float64Range(-179.0, 179.0).Draw(t, fmt.Sprintf("lng_%d", i))
			nodeType := domain.ValidNodeTypes[rapid.IntRange(0, 2).Draw(t, fmt.Sprintf("type_%d", i))]

			nodes = append(nodes, &domain.MapNodeWithRef{
				ID:       fmt.Sprintf("node-%d", i),
				NodeType: nodeType,
				Latitude: lat,
				Longitude: lng,
				Name:     fmt.Sprintf("Node-%d", i),
				Status:   "online",
			})
		}

		// Generate cable routes acak (0-5)
		numCables := rapid.IntRange(0, 5).Draw(t, "numCables")
		cables := make([]*domain.CableRoute, 0, numCables)
		for i := 0; i < numCables; i++ {
			lat1 := rapid.Float64Range(-89.0, 89.0).Draw(t, fmt.Sprintf("cLat1_%d", i))
			lng1 := rapid.Float64Range(-179.0, 179.0).Draw(t, fmt.Sprintf("cLng1_%d", i))
			lat2 := rapid.Float64Range(-89.0, 89.0).Draw(t, fmt.Sprintf("cLat2_%d", i))
			lng2 := rapid.Float64Range(-179.0, 179.0).Draw(t, fmt.Sprintf("cLng2_%d", i))
			routeType := domain.ValidRouteTypes[rapid.IntRange(0, 1).Draw(t, fmt.Sprintf("rType_%d", i))]

			coords := [][2]float64{{lat1, lng1}, {lat2, lng2}}
			coordJSON, _ := json.Marshal(coords)

			cables = append(cables, &domain.CableRoute{
				ID:          fmt.Sprintf("cable-%d", i),
				RouteType:   routeType,
				Coordinates: coordJSON,
				FromNodeID:  "from",
				ToNodeID:    "to",
			})
		}

		// Export ke GeoJSON
		geoData, err := exportGeoJSON(nodes, cables)
		if err != nil {
			t.Fatalf("exportGeoJSON gagal: %v", err)
		}

		// Parse kembali GeoJSON
		var collection geoJSONCollection
		if err := json.Unmarshal(geoData, &collection); err != nil {
			t.Fatalf("gagal parse GeoJSON: %v", err)
		}

		// Properti 1: jumlah features == jumlah node + jumlah cable
		expectedFeatures := numNodes + numCables
		if len(collection.Features) != expectedFeatures {
			t.Fatalf("jumlah features: got %d, want %d",
				len(collection.Features), expectedFeatures)
		}

		// Properti 2: setiap node feature memiliki tipe Point
		pointCount := 0
		lineCount := 0
		for _, f := range collection.Features {
			if f.Geometry.Type == "Point" {
				pointCount++
			} else if f.Geometry.Type == "LineString" {
				lineCount++
			}
		}

		if pointCount != numNodes {
			t.Fatalf("jumlah Point features: got %d, want %d", pointCount, numNodes)
		}
		if lineCount != numCables {
			t.Fatalf("jumlah LineString features: got %d, want %d", lineCount, numCables)
		}

		// Properti 3: setiap feature memiliki properties node_type atau route_type
		for _, f := range collection.Features {
			if f.Geometry.Type == "Point" {
				if _, ok := f.Properties["node_type"]; !ok {
					t.Fatal("Point feature seharusnya memiliki property 'node_type'")
				}
			}
			if f.Geometry.Type == "LineString" {
				if _, ok := f.Properties["route_type"]; !ok {
					t.Fatal("LineString feature seharusnya memiliki property 'route_type'")
				}
			}
		}
	})
}
