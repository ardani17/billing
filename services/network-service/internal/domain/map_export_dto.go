package domain

// =============================================================================
// Export Format Constants — format yang didukung untuk export peta
// =============================================================================

const (
	// ExportFormatKML adalah format Keyhole Markup Language untuk Google Earth.
	ExportFormatKML = "kml"
	// ExportFormatKMZ adalah format KML terkompresi (ZIP) dengan ikon.
	ExportFormatKMZ = "kmz"
	// ExportFormatGeoJSON adalah format GeoJSON untuk GIS tools.
	ExportFormatGeoJSON = "geojson"
	// ExportFormatCSV adalah format CSV untuk spreadsheet.
	ExportFormatCSV = "csv"
)

// ValidExportFormats berisi daftar format export yang valid.
var ValidExportFormats = []string{
	ExportFormatKML,
	ExportFormatKMZ,
	ExportFormatGeoJSON,
	ExportFormatCSV,
}

// IsValidExportFormat mengecek apakah format yang diberikan valid untuk export.
func IsValidExportFormat(format string) bool {
	for _, f := range ValidExportFormats {
		if f == format {
			return true
		}
	}
	return false
}

// =============================================================================
// Export DTOs — request/response untuk export peta
// =============================================================================

// ExportOptions berisi opsi tambahan untuk export peta.
// Digunakan terutama untuk format KMZ yang mendukung ikon dan foto.
type ExportOptions struct {
	IncludeIcons        bool `json:"include_icons"`
	IncludeDescriptions bool `json:"include_descriptions"`
	IncludePhotos       bool `json:"include_photos"`
}

// ExportRequest adalah payload untuk POST /api/v1/network-map/export.
// Format menentukan output (kml/kmz/geojson/csv).
// Layers menentukan tipe node dan cable route yang akan di-export.
// Options berisi opsi tambahan seperti ikon dan deskripsi.
type ExportRequest struct {
	Format  string       `json:"format" validate:"required,oneof=kml kmz geojson csv"`
	Layers  []string     `json:"layers" validate:"required,min=1"`
	Options ExportOptions `json:"options"`
}

// ExportResult adalah respons dari operasi export.
// Jika dataset kecil (≤500 items), FileBytes berisi data file langsung (sync).
// Jika dataset besar (>500 items), Async=true dan JobID berisi ID job async.
type ExportResult struct {
	JobID       string `json:"job_id,omitempty"`
	FileBytes   []byte `json:"-"`
	FileName    string `json:"file_name,omitempty"`
	ContentType string `json:"content_type,omitempty"`
	Async       bool   `json:"async"`
}

// ExportStatus adalah respons untuk GET /api/v1/network-map/export/status/:job_id.
// Digunakan untuk mengecek status export async.
type ExportStatus struct {
	JobID       string `json:"job_id"`
	Status      string `json:"status"`
	DownloadURL string `json:"download_url,omitempty"`
	Error       string `json:"error,omitempty"`
}

// =============================================================================
// Import DTOs — request/response untuk import peta
// =============================================================================

// ImportPreviewItem adalah satu item yang terdeteksi dari file import.
// Type bisa berupa "point", "line", atau "polygon".
// Lat dan Lng hanya diisi untuk item bertipe "point".
type ImportPreviewItem struct {
	Name string   `json:"name"`
	Type string   `json:"type"`
	Lat  *float64 `json:"lat,omitempty"`
	Lng  *float64 `json:"lng,omitempty"`
}

// ImportPreview adalah respons dari POST /api/v1/network-map/import.
// Berisi ringkasan item yang terdeteksi dari file import sebelum dieksekusi.
type ImportPreview struct {
	ImportID string              `json:"import_id"`
	FileName string              `json:"file_name"`
	Points   int                 `json:"points"`
	Lines    int                 `json:"lines"`
	Polygons int                 `json:"polygons"`
	Items    []ImportPreviewItem `json:"items"`
}

// ImportMapping adalah payload untuk POST /api/v1/network-map/import/execute.
// TypeMapping memetakan tipe item import ke tipe node (misal "point" -> "odp").
// AutoMatch mengaktifkan pencocokan otomatis nama item dengan data existing.
type ImportMapping struct {
	ImportID    string            `json:"import_id" validate:"required"`
	TypeMapping map[string]string `json:"type_mapping" validate:"required"`
	AutoMatch   bool              `json:"auto_match"`
}

// ImportError berisi detail error untuk satu item yang gagal di-import.
type ImportError struct {
	ItemName string `json:"item_name"`
	Reason   string `json:"reason"`
}

// ImportSummary adalah ringkasan hasil eksekusi import.
// Berisi jumlah total, sukses, dilewati, dan error beserta detailnya.
type ImportSummary struct {
	Total        int           `json:"total"`
	Success      int           `json:"success"`
	Skipped      int           `json:"skipped"`
	Errors       int           `json:"errors"`
	ErrorDetails []ImportError `json:"error_details,omitempty"`
}

// ImportStatus adalah respons untuk GET /api/v1/network-map/import/status/:job_id.
// Digunakan untuk mengecek status import async.
// Progress berisi persentase progres (0-100).
// Summary diisi setelah import selesai.
type ImportStatus struct {
	JobID    string         `json:"job_id"`
	Status   string         `json:"status"`
	Progress int            `json:"progress"`
	Summary  *ImportSummary `json:"summary,omitempty"`
}
