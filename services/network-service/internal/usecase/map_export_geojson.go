// Package usecase berisi implementasi business logic untuk network-service.
// File ini berisi fungsi export GeoJSON dan CSV untuk data peta FTTH.
package usecase

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// =============================================================================
// Struktur GeoJSON — representasi format GeoJSON (RFC 7946)
// =============================================================================

// geoJSONCollection merepresentasikan FeatureCollection dalam format GeoJSON.
type geoJSONCollection struct {
	Type     string           `json:"type"`
	Features []geoJSONFeature `json:"features"`
}

// geoJSONFeature merepresentasikan satu Feature dalam GeoJSON.
type geoJSONFeature struct {
	Type       string                 `json:"type"`
	Geometry   geoJSONGeometry        `json:"geometry"`
	Properties map[string]interface{} `json:"properties"`
}

// geoJSONGeometry merepresentasikan geometri (Point atau LineString).
type geoJSONGeometry struct {
	Type        string      `json:"type"`
	Coordinates interface{} `json:"coordinates"`
}

// =============================================================================
// Export GeoJSON — generate FeatureCollection dari data peta
// =============================================================================

// exportGeoJSON menghasilkan file GeoJSON dari data node dan cable route.
// Node direpresentasikan sebagai Point, cable route sebagai LineString.
// Semua data dikemas dalam satu FeatureCollection.
func exportGeoJSON(
	nodes []*domain.MapNodeWithRef,
	cables []*domain.CableRoute,
) ([]byte, error) {
	features := make([]geoJSONFeature, 0, len(nodes)+len(cables))

	// Tambahkan node sebagai Point features
	for _, n := range nodes {
		props := map[string]interface{}{
			"id":        n.ID,
			"name":      n.Name,
			"node_type": n.NodeType,
			"status":    n.Status,
		}
		if n.Address != nil {
			props["address"] = *n.Address
		}

		feature := geoJSONFeature{
			Type: "Feature",
			Geometry: geoJSONGeometry{
				Type:        "Point",
				Coordinates: [2]float64{n.Longitude, n.Latitude}, // GeoJSON: [lng, lat]
			},
			Properties: props,
		}
		features = append(features, feature)
	}

	// Tambahkan cable route sebagai LineString features
	for _, c := range cables {
		coords := parseGeoJSONLineCoords(c.Coordinates)
		props := map[string]interface{}{
			"id":              c.ID,
			"route_type":      c.RouteType,
			"distance_meters": c.DistanceMeters,
			"from_node_id":    c.FromNodeID,
			"to_node_id":      c.ToNodeID,
		}

		feature := geoJSONFeature{
			Type: "Feature",
			Geometry: geoJSONGeometry{
				Type:        "LineString",
				Coordinates: coords,
			},
			Properties: props,
		}
		features = append(features, feature)
	}

	collection := geoJSONCollection{
		Type:     "FeatureCollection",
		Features: features,
	}

	data, err := json.MarshalIndent(collection, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("gagal encode GeoJSON: %w", err)
	}

	return data, nil
}

// parseGeoJSONLineCoords mengkonversi JSON coordinates ke format GeoJSON LineString.
// Input: [[lat,lng], ...] → Output: [[lng,lat], ...] (GeoJSON menggunakan lng,lat).
func parseGeoJSONLineCoords(raw json.RawMessage) [][2]float64 {
	var coords [][2]float64
	if err := json.Unmarshal(raw, &coords); err != nil {
		return nil
	}
	// Konversi dari [lat,lng] ke [lng,lat] untuk GeoJSON
	result := make([][2]float64, len(coords))
	for i, c := range coords {
		result[i] = [2]float64{c[1], c[0]} // swap: lng, lat
	}
	return result
}

// =============================================================================
// Export CSV — generate file CSV dari data peta
// =============================================================================

// csvHeader adalah header kolom untuk file CSV export.
const csvHeader = "name,type,lat,lng,status,address,custom_fields\n"

// exportCSV menghasilkan file CSV dari data node dan cable route.
// Kolom: name, type, lat, lng, status, address, custom_fields.
func exportCSV(
	nodes []*domain.MapNodeWithRef,
	cables []*domain.CableRoute,
) ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteString(csvHeader)

	// Tulis data node
	for _, n := range nodes {
		address := ""
		if n.Address != nil {
			address = escapeCSV(*n.Address)
		}
		customFields := ""
		if n.CustomFields != nil {
			customFields = escapeCSV(string(n.CustomFields))
		}

		fmt.Fprintf(&buf, "%s,%s,%.6f,%.6f,%s,%s,%s\n",
			escapeCSV(n.Name),
			n.NodeType,
			n.Latitude,
			n.Longitude,
			n.Status,
			address,
			customFields,
		)
	}

	// Tulis data cable route sebagai baris tambahan
	for _, c := range cables {
		desc := ""
		if c.Description != nil {
			desc = escapeCSV(*c.Description)
		}
		// Ambil koordinat pertama sebagai representasi lokasi
		lat, lng := firstCoordinate(c.Coordinates)

		fmt.Fprintf(&buf, "%s,%s,%.6f,%.6f,%s,%s,%s\n",
			escapeCSV(fmt.Sprintf("Route %s", c.ID[:8])),
			c.RouteType,
			lat,
			lng,
			"active",
			desc,
			"",
		)
	}

	return buf.Bytes(), nil
}

// escapeCSV meng-escape string untuk format CSV.
// Jika string mengandung koma, newline, atau kutip ganda, dibungkus dengan kutip ganda.
func escapeCSV(s string) string {
	needsQuote := false
	for _, c := range s {
		if c == ',' || c == '\n' || c == '"' {
			needsQuote = true
			break
		}
	}
	if !needsQuote {
		return s
	}
	var buf bytes.Buffer
	buf.WriteByte('"')
	for _, c := range s {
		if c == '"' {
			buf.WriteString(`""`)
		} else {
			buf.WriteRune(c)
		}
	}
	buf.WriteByte('"')
	return buf.String()
}

// firstCoordinate mengambil koordinat pertama dari JSON array coordinates.
// Mengembalikan (0,0) jika parsing gagal atau array kosong.
func firstCoordinate(raw json.RawMessage) (float64, float64) {
	var coords [][2]float64
	if err := json.Unmarshal(raw, &coords); err != nil || len(coords) == 0 {
		return 0, 0
	}
	return coords[0][0], coords[0][1]
}
