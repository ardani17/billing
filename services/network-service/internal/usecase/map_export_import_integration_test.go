//go:build integration

// map_export_import_integration_test.go — integration test untuk export/import GeoJSON.
// Menguji alur: buat node + cable → export GeoJSON → parse GeoJSON → verifikasi round-trip.
// Menggunakan mock in-memory repository (sama seperti map_node_manager_test.go).
// Semua komentar dalam Bahasa Indonesia.
package usecase

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// =============================================================================
// Integration Test: Export GeoJSON → Parse → Verifikasi Round-Trip
// =============================================================================

// TestIntegration_ExportImportGeoJSON menguji alur end-to-end:
// 1. Buat beberapa node (OLT, ODP, ONT)
// 2. Buat cable route antar node
// 3. Export ke format GeoJSON
// 4. Parse GeoJSON yang dihasilkan
// 5. Verifikasi jumlah feature sesuai (node=Point, cable=LineString)
// 6. Verifikasi properti node (node_type, name) terjaga
// 7. Verifikasi properti cable (route_type, distance_meters) terjaga
func TestIntegration_ExportImportGeoJSON(t *testing.T) {
	// --- Inisialisasi mock repository ---
	nodeRepo := newMockMapNodeRepo()
	photoRepo := newMockNodePhotoRepo()
	historyRepo := newMockChangeHistoryRepo()
	labelRepo := newMockLabelSettingsRepo()
	cableRepo := newMockCableRouteRepo()

	nodeMgr := NewMapNodeManager(nodeRepo, photoRepo, historyRepo, labelRepo)
	cableMgr := NewCableRouteManager(cableRepo, nodeRepo)
	exportMgr := NewMapExportManager(nodeRepo, cableRepo)

	ctx := context.Background()
	tenantID := "tenant-export-integ"

	// =========================================================================
	// Langkah 1: Buat node OLT
	// =========================================================================
	t.Log("Langkah 1: Membuat node OLT")
	oltNode, err := nodeMgr.CreateNode(ctx, tenantID, domain.CreateMapNodeRequest{
		NodeType:    domain.NodeTypeOLT,
		ReferenceID: "olt-export-001",
		Latitude:    -6.2088,
		Longitude:   106.8456,
	})
	if err != nil {
		t.Fatalf("gagal membuat node OLT: %v", err)
	}

	// =========================================================================
	// Langkah 2: Buat node ODP
	// =========================================================================
	t.Log("Langkah 2: Membuat node ODP")
	odpNode, err := nodeMgr.CreateNode(ctx, tenantID, domain.CreateMapNodeRequest{
		NodeType:    domain.NodeTypeODP,
		ReferenceID: "odp-export-001",
		Latitude:    -6.3000,
		Longitude:   106.9000,
	})
	if err != nil {
		t.Fatalf("gagal membuat node ODP: %v", err)
	}

	// =========================================================================
	// Langkah 3: Buat node ONT
	// =========================================================================
	t.Log("Langkah 3: Membuat node ONT")
	ontNode, err := nodeMgr.CreateNode(ctx, tenantID, domain.CreateMapNodeRequest{
		NodeType:    domain.NodeTypeONT,
		ReferenceID: "ont-export-001",
		Latitude:    -6.3500,
		Longitude:   106.9500,
	})
	if err != nil {
		t.Fatalf("gagal membuat node ONT: %v", err)
	}

	// =========================================================================
	// Langkah 4: Buat cable route backbone (OLT → ODP)
	// =========================================================================
	t.Log("Langkah 4: Membuat cable route backbone OLT → ODP")
	backboneCoords := [][2]float64{
		{-6.2088, 106.8456},
		{-6.3000, 106.9000},
	}
	backboneCoordsJSON, _ := json.Marshal(backboneCoords)

	backboneCable, err := cableMgr.CreateRoute(ctx, tenantID, domain.CreateCableRouteRequest{
		FromNodeID:  oltNode.ID,
		ToNodeID:    odpNode.ID,
		RouteType:   domain.RouteTypeBackbone,
		Coordinates: backboneCoordsJSON,
	})
	if err != nil {
		t.Fatalf("gagal membuat cable route backbone: %v", err)
	}

	// =========================================================================
	// Langkah 5: Buat cable route drop (ODP → ONT)
	// =========================================================================
	t.Log("Langkah 5: Membuat cable route drop ODP → ONT")
	dropCoords := [][2]float64{
		{-6.3000, 106.9000},
		{-6.3500, 106.9500},
	}
	dropCoordsJSON, _ := json.Marshal(dropCoords)

	dropCable, err := cableMgr.CreateRoute(ctx, tenantID, domain.CreateCableRouteRequest{
		FromNodeID:  odpNode.ID,
		ToNodeID:    ontNode.ID,
		RouteType:   domain.RouteTypeDrop,
		Coordinates: dropCoordsJSON,
	})
	if err != nil {
		t.Fatalf("gagal membuat cable route drop: %v", err)
	}

	// =========================================================================
	// Langkah 6: Export ke GeoJSON
	// =========================================================================
	t.Log("Langkah 6: Export ke GeoJSON")
	exportReq := domain.ExportRequest{
		Format: domain.ExportFormatGeoJSON,
		Layers: []string{domain.NodeTypeOLT, domain.NodeTypeODP, domain.NodeTypeONT},
	}
	result, err := exportMgr.Export(ctx, tenantID, exportReq)
	if err != nil {
		t.Fatalf("gagal export GeoJSON: %v", err)
	}

	// Verifikasi export sync (bukan async, karena dataset kecil)
	if result.Async {
		t.Fatal("export kecil seharusnya sync, bukan async")
	}
	if result.FileBytes == nil {
		t.Fatal("FileBytes seharusnya tidak nil untuk export sync")
	}

	// =========================================================================
	// Langkah 7: Parse GeoJSON yang dihasilkan
	// =========================================================================
	t.Log("Langkah 7: Parse GeoJSON yang dihasilkan")
	var collection geoJSONCollection
	if err := json.Unmarshal(result.FileBytes, &collection); err != nil {
		t.Fatalf("gagal parse GeoJSON: %v", err)
	}

	// Verifikasi tipe FeatureCollection
	if collection.Type != "FeatureCollection" {
		t.Errorf("tipe collection: got %q, want %q", collection.Type, "FeatureCollection")
	}

	// =========================================================================
	// Langkah 8: Verifikasi jumlah feature (3 node + 2 cable = 5)
	// =========================================================================
	t.Log("Langkah 8: Verifikasi jumlah feature")
	expectedFeatures := 5 // 3 node (Point) + 2 cable (LineString)
	if len(collection.Features) != expectedFeatures {
		t.Errorf("jumlah features: got %d, want %d", len(collection.Features), expectedFeatures)
	}

	// Hitung jumlah Point dan LineString
	pointCount := 0
	lineCount := 0
	for _, f := range collection.Features {
		switch f.Geometry.Type {
		case "Point":
			pointCount++
		case "LineString":
			lineCount++
		}
	}
	if pointCount != 3 {
		t.Errorf("jumlah Point features: got %d, want 3", pointCount)
	}
	if lineCount != 2 {
		t.Errorf("jumlah LineString features: got %d, want 2", lineCount)
	}

	// =========================================================================
	// Langkah 9: Verifikasi properti node terjaga di GeoJSON
	// =========================================================================
	t.Log("Langkah 9: Verifikasi properti node di GeoJSON")
	nodeTypeSet := make(map[string]bool)
	for _, f := range collection.Features {
		if f.Geometry.Type == "Point" {
			nt, ok := f.Properties["node_type"]
			if !ok {
				t.Error("Point feature seharusnya memiliki property 'node_type'")
				continue
			}
			nodeTypeSet[nt.(string)] = true

			// Verifikasi property 'name' ada
			if _, ok := f.Properties["name"]; !ok {
				t.Error("Point feature seharusnya memiliki property 'name'")
			}

			// Verifikasi property 'id' ada
			if _, ok := f.Properties["id"]; !ok {
				t.Error("Point feature seharusnya memiliki property 'id'")
			}
		}
	}
	// Verifikasi semua tipe node ada di GeoJSON
	if !nodeTypeSet[domain.NodeTypeOLT] {
		t.Error("GeoJSON seharusnya mengandung node tipe OLT")
	}
	if !nodeTypeSet[domain.NodeTypeODP] {
		t.Error("GeoJSON seharusnya mengandung node tipe ODP")
	}
	if !nodeTypeSet[domain.NodeTypeONT] {
		t.Error("GeoJSON seharusnya mengandung node tipe ONT")
	}

	// =========================================================================
	// Langkah 10: Verifikasi properti cable terjaga di GeoJSON
	// =========================================================================
	t.Log("Langkah 10: Verifikasi properti cable di GeoJSON")
	routeTypeSet := make(map[string]bool)
	for _, f := range collection.Features {
		if f.Geometry.Type == "LineString" {
			rt, ok := f.Properties["route_type"]
			if !ok {
				t.Error("LineString feature seharusnya memiliki property 'route_type'")
				continue
			}
			routeTypeSet[rt.(string)] = true

			// Verifikasi property 'distance_meters' ada
			if _, ok := f.Properties["distance_meters"]; !ok {
				t.Error("LineString feature seharusnya memiliki property 'distance_meters'")
			}

			// Verifikasi property 'from_node_id' dan 'to_node_id' ada
			if _, ok := f.Properties["from_node_id"]; !ok {
				t.Error("LineString feature seharusnya memiliki property 'from_node_id'")
			}
			if _, ok := f.Properties["to_node_id"]; !ok {
				t.Error("LineString feature seharusnya memiliki property 'to_node_id'")
			}
		}
	}
	// Verifikasi kedua tipe route ada di GeoJSON
	if !routeTypeSet[domain.RouteTypeBackbone] {
		t.Error("GeoJSON seharusnya mengandung route tipe backbone")
	}
	if !routeTypeSet[domain.RouteTypeDrop] {
		t.Error("GeoJSON seharusnya mengandung route tipe drop")
	}

	// =========================================================================
	// Langkah 11: Verifikasi round-trip — parse GeoJSON kembali sebagai import
	// =========================================================================
	t.Log("Langkah 11: Verifikasi round-trip — parse GeoJSON sebagai import")
	importItems, err := parseGeoJSONImport(result.FileBytes)
	if err != nil {
		t.Fatalf("gagal parse GeoJSON sebagai import: %v", err)
	}

	// Verifikasi jumlah item yang terdeteksi
	if len(importItems) != expectedFeatures {
		t.Errorf("jumlah import items: got %d, want %d", len(importItems), expectedFeatures)
	}

	// Hitung tipe item import
	importPoints := 0
	importLines := 0
	for _, item := range importItems {
		switch item.Type {
		case "point":
			importPoints++
		case "line":
			importLines++
		}
	}
	if importPoints != 3 {
		t.Errorf("jumlah import points: got %d, want 3", importPoints)
	}
	if importLines != 2 {
		t.Errorf("jumlah import lines: got %d, want 2", importLines)
	}

	// Verifikasi koordinat point items memiliki lat/lng yang valid
	for _, item := range importItems {
		if item.Type == "point" {
			if item.Lat == nil || item.Lng == nil {
				t.Error("import point item seharusnya memiliki lat dan lng")
				continue
			}
			// Koordinat harus dalam range valid
			if *item.Lat < -90 || *item.Lat > 90 {
				t.Errorf("import point lat di luar range: %f", *item.Lat)
			}
			if *item.Lng < -180 || *item.Lng > 180 {
				t.Errorf("import point lng di luar range: %f", *item.Lng)
			}
		}
	}

	// =========================================================================
	// Langkah 12: Verifikasi jarak cable route konsisten setelah round-trip
	// =========================================================================
	t.Log("Langkah 12: Verifikasi jarak cable route konsisten")
	// Ambil cable route yang sudah dibuat dan verifikasi jarak > 0
	backboneRoute, err := cableMgr.GetRoute(ctx, backboneCable.ID)
	if err != nil {
		t.Fatalf("gagal mengambil cable route backbone: %v", err)
	}
	if backboneRoute.DistanceMeters <= 0 {
		t.Error("jarak backbone route seharusnya > 0")
	}

	dropRoute, err := cableMgr.GetRoute(ctx, dropCable.ID)
	if err != nil {
		t.Fatalf("gagal mengambil cable route drop: %v", err)
	}
	if dropRoute.DistanceMeters <= 0 {
		t.Error("jarak drop route seharusnya > 0")
	}

	t.Log("Integration test export/import GeoJSON round-trip berhasil!")
}

// =============================================================================
// Integration Test: Export Async untuk Dataset Besar
// =============================================================================

// TestIntegration_ExportAsyncLargeDataset memverifikasi bahwa export dengan
// dataset besar (>500 items) menghasilkan job async, bukan file langsung.
func TestIntegration_ExportAsyncLargeDataset(t *testing.T) {
	// --- Inisialisasi mock repository ---
	nodeRepo := newMockMapNodeRepo()
	cableRepo := newMockCableRouteRepo()
	photoRepo := newMockNodePhotoRepo()
	historyRepo := newMockChangeHistoryRepo()
	labelRepo := newMockLabelSettingsRepo()

	nodeMgr := NewMapNodeManager(nodeRepo, photoRepo, historyRepo, labelRepo)
	exportMgr := NewMapExportManager(nodeRepo, cableRepo)

	ctx := context.Background()
	tenantID := "tenant-async-integ"

	// =========================================================================
	// Buat >500 node untuk trigger export async
	// =========================================================================
	t.Log("Membuat 510 node untuk trigger export async")
	for i := 0; i < 510; i++ {
		_, err := nodeMgr.CreateNode(ctx, tenantID, domain.CreateMapNodeRequest{
			NodeType:    domain.NodeTypeONT,
			ReferenceID: generateRefID("ont-async", i),
			Latitude:    -6.0 + float64(i)*0.001,
			Longitude:   106.0 + float64(i)*0.001,
		})
		if err != nil {
			t.Fatalf("gagal membuat node ke-%d: %v", i, err)
		}
	}

	// =========================================================================
	// Export — harus menghasilkan async job
	// =========================================================================
	t.Log("Export GeoJSON — seharusnya async")
	result, err := exportMgr.Export(ctx, tenantID, domain.ExportRequest{
		Format: domain.ExportFormatGeoJSON,
		Layers: []string{domain.NodeTypeONT},
	})
	if err != nil {
		t.Fatalf("gagal export: %v", err)
	}

	if !result.Async {
		t.Fatal("export >500 items seharusnya async")
	}
	if result.JobID == "" {
		t.Fatal("JobID seharusnya tidak kosong untuk export async")
	}

	// =========================================================================
	// Verifikasi status job bisa diambil
	// =========================================================================
	t.Log("Verifikasi status export async")
	status, err := exportMgr.GetExportStatus(ctx, result.JobID)
	if err != nil {
		t.Fatalf("gagal mengambil status export: %v", err)
	}
	if status.Status != "processing" {
		t.Errorf("status export: got %q, want %q", status.Status, "processing")
	}

	t.Log("Integration test export async dataset besar berhasil!")
}

// generateRefID membuat reference ID unik untuk testing.
func generateRefID(prefix string, index int) string {
	// Format: prefix-XXXXXXXX (8 digit hex-like)
	return prefix + "-" + padInt(index, 8)
}

// padInt mengkonversi integer ke string dengan padding nol di depan.
func padInt(n, width int) string {
	s := ""
	for n > 0 || len(s) < width {
		s = string(rune('0'+n%10)) + s
		n /= 10
		if len(s) >= width {
			break
		}
	}
	for len(s) < width {
		s = "0" + s
	}
	return s
}
