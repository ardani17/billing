// Package usecase berisi implementasi business logic untuk network-service.
// File ini berisi fungsi export KML dan KMZ untuk data peta FTTH.
package usecase

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// =============================================================================
// Struktur XML KML - representasi format Keyhole Markup Language
// =============================================================================

// kmlDocument adalah root element dokumen KML.
type kmlDocument struct {
	XMLName  xml.Name    `xml:"kml"`
	XMLNS    string      `xml:"xmlns,attr"`
	Document kmlDocInner `xml:"Document"`
}

// kmlDocInner berisi nama dokumen dan folder-folder di dalamnya.
type kmlDocInner struct {
	Name    string      `xml:"name"`
	Folders []kmlFolder `xml:"Folder"`
}

// kmlFolder merepresentasikan folder dalam dokumen KML (per tipe node).
type kmlFolder struct {
	Name       string         `xml:"name"`
	Placemarks []kmlPlacemark `xml:"Placemark"`
}

// kmlPlacemark merepresentasikan satu titik atau garis di peta KML.
type kmlPlacemark struct {
	Name        string    `xml:"name"`
	Description string    `xml:"description,omitempty"`
	Point       *kmlPoint `xml:"Point,omitempty"`
	LineString  *kmlLine  `xml:"LineString,omitempty"`
}

// kmlPoint berisi koordinat titik dalam format KML (lng,lat,altitude).
type kmlPoint struct {
	Coordinates string `xml:"coordinates"`
}

// kmlLine berisi koordinat garis dalam format KML.
type kmlLine struct {
	Coordinates string `xml:"coordinates"`
}

// =============================================================================
// Export KML - buat file KML dari data node dan cable route
// =============================================================================

// exportKML menghasilkan file KML dari data node dan cable route.
// Node diorganisir ke dalam folder berdasarkan tipe (OLT, ODP, ONT).
// Cable route dimasukkan ke folder terpisah (Backbone, Drop).
func exportKML(
	nodes []*domain.MapNodeWithRef,
	cables []*domain.CableRoute,
	opts domain.ExportOptions,
) ([]byte, error) {
	doc := buildKMLDocument(nodes, cables, opts)

	var buf bytes.Buffer
	buf.WriteString(xml.Header)
	enc := xml.NewEncoder(&buf)
	enc.Indent("", "  ")
	if err := enc.Encode(doc); err != nil {
		return nil, fmt.Errorf("gagal encode KML: %w", err)
	}

	return buf.Bytes(), nil
}

// buildKMLDocument membangun struktur dokumen KML dari data peta.
func buildKMLDocument(
	nodes []*domain.MapNodeWithRef,
	cables []*domain.CableRoute,
	opts domain.ExportOptions,
) kmlDocument {
	// Kelompokkan node berdasarkan tipe
	nodesByType := map[string][]*domain.MapNodeWithRef{}
	for _, n := range nodes {
		nodesByType[n.NodeType] = append(nodesByType[n.NodeType], n)
	}

	// Kelompokkan cable berdasarkan tipe route
	cablesByType := map[string][]*domain.CableRoute{}
	for _, c := range cables {
		cablesByType[c.RouteType] = append(cablesByType[c.RouteType], c)
	}

	var folders []kmlFolder

	// Buat folder per tipe node
	for _, nodeType := range domain.ValidNodeTypes {
		typeNodes := nodesByType[nodeType]
		if len(typeNodes) == 0 {
			continue
		}
		folder := kmlFolder{Name: fmt.Sprintf("Node %s", nodeType)}
		for _, n := range typeNodes {
			pm := kmlPlacemark{
				Name:  n.Name,
				Point: &kmlPoint{Coordinates: fmt.Sprintf("%.6f,%.6f,0", n.Longitude, n.Latitude)},
			}
			if opts.IncludeDescriptions {
				pm.Description = buildNodeDescription(n)
			}
			folder.Placemarks = append(folder.Placemarks, pm)
		}
		folders = append(folders, folder)
	}

	// Buat folder per tipe cable route
	for _, routeType := range domain.ValidRouteTypes {
		typeCables := cablesByType[routeType]
		if len(typeCables) == 0 {
			continue
		}
		folder := kmlFolder{Name: fmt.Sprintf("Cable %s", routeType)}
		for _, c := range typeCables {
			coords := formatCableCoordinatesKML(c.Coordinates)
			pm := kmlPlacemark{
				Name:       fmt.Sprintf("Route %s", c.ID[:8]),
				LineString: &kmlLine{Coordinates: coords},
			}
			folder.Placemarks = append(folder.Placemarks, pm)
		}
		folders = append(folders, folder)
	}

	return kmlDocument{
		XMLNS: "http://www.opengis.net/kml/2.2",
		Document: kmlDocInner{
			Name:    "FTTH Network Map Export",
			Folders: folders,
		},
	}
}

// buildNodeDescription membuat deskripsi HTML untuk node di KML.
func buildNodeDescription(n *domain.MapNodeWithRef) string {
	desc := fmt.Sprintf("Tipe: %s\nStatus: %s\nKoordinat: %.6f, %.6f",
		n.NodeType, n.Status, n.Latitude, n.Longitude)
	if n.Address != nil {
		desc += fmt.Sprintf("\nAlamat: %s", *n.Address)
	}
	return desc
}

// formatCableCoordinatesKML mengkonversi JSON coordinates ke format KML.
// Format KML: "lng,lat,0 lng,lat,0 ..."
func formatCableCoordinatesKML(raw json.RawMessage) string {
	var coords [][2]float64
	if err := json.Unmarshal(raw, &coords); err != nil {
		return ""
	}
	var buf bytes.Buffer
	for i, c := range coords {
		if i > 0 {
			buf.WriteByte(' ')
		}
		fmt.Fprintf(&buf, "%.6f,%.6f,0", c[1], c[0]) // KML: lng,lat,alt
	}
	return buf.String()
}

// =============================================================================
// Export KMZ - package KML + ikon ke dalam arsip ZIP
// =============================================================================

// exportKMZ menghasilkan file KMZ (KML terkompresi dalam ZIP).
// KMZ berisi file doc.kml dan opsional folder icons/.
func exportKMZ(
	nodes []*domain.MapNodeWithRef,
	cables []*domain.CableRoute,
	opts domain.ExportOptions,
) ([]byte, error) {
	// Buat KML terlebih dahulu
	kmlData, err := exportKML(nodes, cables, opts)
	if err != nil {
		return nil, fmt.Errorf("gagal generate KML untuk KMZ: %w", err)
	}

	// Buat arsip ZIP
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	// Tambahkan file KML ke arsip
	fw, err := zw.Create("doc.kml")
	if err != nil {
		return nil, fmt.Errorf("gagal membuat entry KML di KMZ: %w", err)
	}
	if _, err := fw.Write(kmlData); err != nil {
		return nil, fmt.Errorf("gagal menulis KML ke KMZ: %w", err)
	}

	if err := zw.Close(); err != nil {
		return nil, fmt.Errorf("gagal menutup arsip KMZ: %w", err)
	}

	return buf.Bytes(), nil
}
