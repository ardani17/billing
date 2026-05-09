// map_import_test.go - unit test untuk MapImportManager.
// Semua komentar dalam Bahasa Indonesia.
package usecase

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"testing"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// =============================================================================
// Unit Tes 1: TestParseKML - verifikasi parsing file KML
// =============================================================================

// TestParseKML memverifikasi bahwa parseKMLImport mengekstrak placemark
// dari dokumen KML dengan benar.
func TestParseKML(t *testing.T) {
	kmlData := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<kml xmlns="http://www.opengis.net/kml/2.2">
  <Document>
    <name>Test</name>
    <Folder>
      <name>ODP</name>
      <Placemark>
        <name>ODP-001</name>
        <Point><coordinates>106.845600,-6.208800,0</coordinates></Point>
      </Placemark>
    </Folder>
    <Placemark>
      <name>Route-001</name>
      <LineString><coordinates>106.8,6.2,0 107.0,-6.3,0</coordinates></LineString>
    </Placemark>
  </Document>
</kml>`)

	items, err := parseKMLImport(kmlData)
	if err != nil {
		t.Fatalf("parseKMLImport gagal: %v", err)
	}

	// Verifikasi jumlah item
	if len(items) != 2 {
		t.Fatalf("jumlah items: got %d, want 2", len(items))
	}

	// Verifikasi item pertama (point)
	if items[0].Type != "point" {
		t.Errorf("item[0].Type: got %q, want %q", items[0].Type, "point")
	}
	if items[0].Name != "ODP-001" {
		t.Errorf("item[0].Name: got %q, want %q", items[0].Name, "ODP-001")
	}
	if items[0].Lat == nil || items[0].Lng == nil {
		t.Fatal("item[0] seharusnya memiliki koordinat")
	}

	// Verifikasi item kedua (line)
	if items[1].Type != "line" {
		t.Errorf("item[1].Type: got %q, want %q", items[1].Type, "line")
	}
}

// =============================================================================
// Unit Tes 2: TestParseKMZ - verifikasi ekstraksi KML dari KMZ
// =============================================================================

// TestParseKMZ memverifikasi bahwa parseKMZImport mengekstrak KML
// dari arsip ZIP (KMZ) dengan benar.
func TestParseKMZ(t *testing.T) {
	// Buat KMZ (ZIP berisi doc.kml)
	kmlContent := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<kml xmlns="http://www.opengis.net/kml/2.2">
  <Document>
    <Placemark>
      <name>Test-KMZ</name>
      <Point><coordinates>106.8,-6.2,0</coordinates></Point>
    </Placemark>
  </Document>
</kml>`)

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	fw, err := zw.Create("doc.kml")
	if err != nil {
		t.Fatalf("gagal membuat entry ZIP: %v", err)
	}
	if _, err := fw.Write(kmlContent); err != nil {
		t.Fatalf("gagal menulis ke ZIP: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("gagal menutup ZIP: %v", err)
	}

	items, err := parseKMZImport(buf.Bytes())
	if err != nil {
		t.Fatalf("parseKMZImport gagal: %v", err)
	}

	if len(items) != 1 {
		t.Fatalf("jumlah items: got %d, want 1", len(items))
	}
	if items[0].Name != "Test-KMZ" {
		t.Errorf("item[0].Name: got %q, want %q", items[0].Name, "Test-KMZ")
	}
}

// =============================================================================
// Unit Tes 3: TestParseGeoJSON - verifikasi parsing file GeoJSON
// =============================================================================

// TestParseGeoJSON memverifikasi bahwa parseGeoJSONImport mengekstrak
// features dari FeatureCollection GeoJSON dengan benar.
func TestParseGeoJSON(t *testing.T) {
	geojson := geoJSONCollection{
		Type: "FeatureCollection",
		Features: []geoJSONFeature{
			{
				Type: "Feature",
				Geometry: geoJSONGeometry{
					Type:        "Point",
					Coordinates: []interface{}{106.8456, -6.2088},
				},
				Properties: map[string]interface{}{
					"name":      "ODP-GeoJSON",
					"node_type": "odp",
				},
			},
			{
				Type: "Feature",
				Geometry: geoJSONGeometry{
					Type:        "LineString",
					Coordinates: []interface{}{[]interface{}{106.8, -6.2}, []interface{}{107.0, -6.3}},
				},
				Properties: map[string]interface{}{
					"name":       "Route-GeoJSON",
					"route_type": "backbone",
				},
			},
		},
	}

	data, err := json.Marshal(geojson)
	if err != nil {
		t.Fatalf("gagal marshal GeoJSON: %v", err)
	}

	items, err := parseGeoJSONImport(data)
	if err != nil {
		t.Fatalf("parseGeoJSONImport gagal: %v", err)
	}

	if len(items) != 2 {
		t.Fatalf("jumlah items: got %d, want 2", len(items))
	}

	// Verifikasi point
	if items[0].Type != "point" {
		t.Errorf("item[0].Type: got %q, want %q", items[0].Type, "point")
	}
	if items[0].Name != "ODP-GeoJSON" {
		t.Errorf("item[0].Name: got %q, want %q", items[0].Name, "ODP-GeoJSON")
	}

	// Verifikasi line
	if items[1].Type != "line" {
		t.Errorf("item[1].Type: got %q, want %q", items[1].Type, "line")
	}
}

// =============================================================================
// Unit Tes 4: TestImportCoordinateValidation - verifikasi validasi koordinat
// =============================================================================

// TestImportCoordinateValidation memverifikasi bahwa Execute menolak
func TestImportCoordinateValidation(t *testing.T) {
	nodeRepo := newMockMapNodeRepo()
	cableRepo := newMockCableRouteRepo()
	mgr := NewMapImportManager(nodeRepo, cableRepo)

	// Akses internal untuk inject preview data
	impl := mgr.(*mapImportManager)

	invalidLat := 91.0
	validLng := 106.0
	validLat := -6.2
	validLng2 := 106.8

	importID := "test-import-validation"
	impl.previews[importID] = &importPreviewData{
		tenantID: "tenant-import",
		items: []domain.ImportPreviewItem{
			{Name: "Invalid", Type: "point", Lat: &invalidLat, Lng: &validLng},
			{Name: "Valid", Type: "point", Lat: &validLat, Lng: &validLng2},
		},
	}

	mapping := domain.ImportMapping{
		ImportID:    importID,
		TypeMapping: map[string]string{"point": "odp"},
	}

	summary, err := impl.Execute(nil, importID, mapping)
	if err != nil {
		t.Fatalf("Execute gagal: %v", err)
	}

	if summary.Total != 2 {
		t.Errorf("Total: got %d, want 2", summary.Total)
	}
	if summary.Success != 1 {
		t.Errorf("Success: got %d, want 1", summary.Success)
	}
	if summary.Errors != 1 {
		t.Errorf("Errors: got %d, want 1", summary.Errors)
	}
	if len(summary.ErrorDetails) != 1 {
		t.Errorf("ErrorDetails: got %d, want 1", len(summary.ErrorDetails))
	}
}
