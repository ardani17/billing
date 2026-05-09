package domain

import (
	"context"
	"mime/multipart"
	"time"
)

// =============================================================================
// MapNodeRepository - operasi data untuk tabel map_nodes
// =============================================================================

// MapNodeRepository mendefinisikan operasi data untuk tabel map_nodes.
// Diimplementasikan oleh repositori.MapNodeRepo menggunakan sqlc.
type MapNodeRepository interface {
	// Buat membuat map node baru dan mengembalikan node yang dibuat.
	Create(ctx context.Context, node *MapNode) (*MapNode, error)

	// GetByID mengambil map node berdasarkan ID (tenant-scoped via RLS).
	GetByID(ctx context.Context, id string) (*MapNode, error)

	// Perbarui memperbarui data map node dan mengembalikan node yang diperbarui.
	Update(ctx context.Context, node *MapNode) (*MapNode, error)

	// SoftDelete melakukan hapus lunak map node (atur deleted_at).
	SoftDelete(ctx context.Context, id string) error

	// Restore mengembalikan map node yang sudah di-hapus lunak (clear deleted_at).
	Restore(ctx context.Context, id string) error

	// ListByBounds mengambil daftar map node dengan join data referensi (OLT/ODP/ONT)
	// berdasarkan bounding box dan filter opsional.
	ListByBounds(ctx context.Context, params MapNodeListParams) ([]*MapNodeWithRef, error)

	// GetByReference mengambil map node berdasarkan tenant_id, node_type, dan reference_id.
	// Digunakan untuk cek duplikasi sebelum buat.
	GetByReference(ctx context.Context, tenantID, nodeType, referenceID string) (*MapNode, error)

	// Pencarian melakukan pencarian full-text di map node dan entitas referensi.
	// Mengembalikan maksimal `limit` hasil pencarian.
	Search(ctx context.Context, tenantID, query string, limit int) ([]*MapSearchResult, error)

	// ListTrashed mengambil daftar map node yang sudah di-hapus lunak (tenant-scoped).
	ListTrashed(ctx context.Context, tenantID string) ([]*MapNode, error)

	// PermanentDeleteExpired menghapus permanen map node yang deleted_at lebih tua dari olderThan.
	// Mengembalikan jumlah baris yang dihapus.
	PermanentDeleteExpired(ctx context.Context, olderThan time.Time) (int64, error)

	// CountPhotosByNode menghitung jumlah foto aktif (non-deleted) untuk satu node.
	CountPhotosByNode(ctx context.Context, nodeID string) (int, error)
}

// =============================================================================
// CableRouteRepository - operasi data untuk tabel cable_routes
// =============================================================================

// CableRouteRepository mendefinisikan operasi data untuk tabel cable_routes.
// Diimplementasikan oleh repositori.CableRouteRepo menggunakan sqlc.
type CableRouteRepository interface {
	// Buat membuat cable route baru dan mengembalikan route yang dibuat.
	Create(ctx context.Context, route *CableRoute) (*CableRoute, error)

	// GetByID mengambil cable route berdasarkan ID (tenant-scoped via RLS).
	GetByID(ctx context.Context, id string) (*CableRoute, error)

	// Perbarui memperbarui data cable route dan mengembalikan route yang diperbarui.
	Update(ctx context.Context, route *CableRoute) (*CableRoute, error)

	// SoftDelete melakukan hapus lunak cable route (atur deleted_at).
	SoftDelete(ctx context.Context, id string) error

	// ListByBounds mengambil daftar cable route berdasarkan bounding box dan filter opsional.
	ListByBounds(ctx context.Context, params CableRouteListParams) ([]*CableRoute, error)

	// ListByNode mengambil daftar cable route yang terhubung ke node tertentu
	// (sebagai from_node_id atau to_node_id).
	ListByNode(ctx context.Context, nodeID string) ([]*CableRoute, error)
}

// =============================================================================
// NodePhotoRepository - operasi data untuk tabel node_photos
// =============================================================================

// NodePhotoRepository mendefinisikan operasi data untuk tabel node_photos.
// Diimplementasikan oleh repositori.NodePhotoRepo menggunakan sqlc.
type NodePhotoRepository interface {
	// Buat membuat record foto baru dan mengembalikan foto yang dibuat.
	Create(ctx context.Context, photo *NodePhoto) (*NodePhoto, error)

	// ListByNode mengambil daftar foto aktif (non-deleted) untuk satu node.
	ListByNode(ctx context.Context, nodeID string) ([]*NodePhoto, error)

	// SoftDelete melakukan hapus lunak foto (atur deleted_at).
	SoftDelete(ctx context.Context, id string) error

	// CountByNode menghitung jumlah foto aktif (non-deleted) untuk satu node.
	CountByNode(ctx context.Context, nodeID string) (int, error)
}

// =============================================================================
// ChangeHistoryRepository - operasi data untuk tabel map_change_history
// =============================================================================

// ChangeHistoryRepository mendefinisikan operasi data untuk tabel map_change_history.
// Tabel ini bersifat append-only: tidak ada operasi Perbarui atau Hapus.
// Diimplementasikan oleh repositori.ChangeHistoryRepo menggunakan sqlc.
type ChangeHistoryRepository interface {
	// Buat menyimpan entri riwayat perubahan baru.
	Create(ctx context.Context, entry *MapChangeHistory) (*MapChangeHistory, error)

	// ListByNode mengambil daftar riwayat perubahan untuk satu node
	// dengan paginasi (limit, offset), diurutkan berdasarkan created_at DESC.
	ListByNode(ctx context.Context, nodeID string, limit, offset int) ([]*MapChangeHistory, error)
}

// =============================================================================
// LabelSettingsRepository - operasi data untuk tabel map_label_settings
// =============================================================================

// LabelSettingsRepository mendefinisikan operasi data untuk tabel map_label_settings.
// Diimplementasikan oleh repositori.LabelSettingsRepo menggunakan sqlc.
type LabelSettingsRepository interface {
	// GetByTenantID mengambil konfigurasi label berdasarkan tenant_id.
	// Mengembalikan nil jika tenant belum memiliki konfigurasi.
	GetByTenantID(ctx context.Context, tenantID string) (*MapLabelSettings, error)

	// Upsert membuat atau memperbarui konfigurasi label untuk tenant.
	Upsert(ctx context.Context, settings *MapLabelSettings) (*MapLabelSettings, error)
}

// =============================================================================
// ShareLinkRepository - operasi data untuk tabel map_share_links
// =============================================================================

// ShareLinkRepository mendefinisikan operasi data untuk tabel map_share_links.
// Diimplementasikan oleh repositori.ShareLinkRepo menggunakan sqlc.
type ShareLinkRepository interface {
	// Buat membuat share link baru dan mengembalikan link yang dibuat.
	Create(ctx context.Context, link *MapShareLink) (*MapShareLink, error)

	// GetByToken mengambil share link berdasarkan token unik.
	GetByToken(ctx context.Context, token string) (*MapShareLink, error)

	// Hapus menghapus share link berdasarkan token.
	Delete(ctx context.Context, token string) error

	// ListByTenant mengambil daftar share link untuk satu tenant.
	ListByTenant(ctx context.Context, tenantID string) ([]*MapShareLink, error)

	// IncrementAccessCount menaikkan access_count share link saat diakses.
	IncrementAccessCount(ctx context.Context, token string) error
}

// =============================================================================
// GeocodingCacheRepository - operasi data untuk tabel geocoding_cache
// =============================================================================

// GeocodingCacheRepository mendefinisikan operasi data untuk tabel geocoding_cache.
// Diimplementasikan oleh repositori.GeocodingCacheRepo menggunakan sqlc.
type GeocodingCacheRepository interface {
	// Get mengambil cache geocoding berdasarkan koordinat yang sudah dibulatkan.
	// Mengembalikan nil jika cache tidak ditemukan atau sudah kedaluwarsa.
	Get(ctx context.Context, latRound, lngRound float64) (*GeocodingCache, error)

	// Set menyimpan atau memperbarui cache geocoding (upsert).
	Set(ctx context.Context, cache *GeocodingCache) error

	// DeleteExpired menghapus cache yang sudah kedaluwarsa (expires_at < now).
	// Mengembalikan jumlah baris yang dihapus.
	DeleteExpired(ctx context.Context) (int64, error)
}

// =============================================================================
// MapNodeManager - business logic untuk manajemen map node
// =============================================================================

// MapNodeManager mendefinisikan business logic untuk manajemen map node.
// Menangani CRUD node, foto, riwayat perubahan, trash, dan label settings.
type MapNodeManager interface {
	// CreateNode membuat map node baru dengan validasi input dan pencatatan riwayat.
	CreateNode(ctx context.Context, tenantID string, req CreateMapNodeRequest) (*MapNodeResponse, error)

	// GetNode mengambil detail lengkap map node termasuk foto, riwayat, dan data referensi.
	GetNode(ctx context.Context, id string) (*MapNodeDetailResponse, error)

	// UpdateNode memperbarui lokasi dan/atau kustom field node dengan pencatatan riwayat.
	UpdateNode(ctx context.Context, id string, req UpdateMapNodeRequest) (*MapNodeResponse, error)

	// DeleteNode melakukan hapus lunak node dengan pencatatan riwayat.
	DeleteNode(ctx context.Context, id string, performedBy string) error

	// ListNodes mengambil daftar node berdasarkan bounding box dan filter.
	ListNodes(ctx context.Context, params MapNodeListParams) ([]*MapNodeWithRefResponse, error)

	// Pencarian melakukan pencarian full-text di node dan entitas referensi.
	Search(ctx context.Context, tenantID, query string) ([]*MapSearchResult, error)

	// UploadPhoto meng-upload foto ke node dengan validasi tipe file dan batas jumlah.
	UploadPhoto(ctx context.Context, nodeID string, file multipart.File, header *multipart.FileHeader, caption, uploadedBy string) (*NodePhotoResponse, error)

	// ListPhotos mengambil daftar foto aktif untuk satu node.
	ListPhotos(ctx context.Context, nodeID string) ([]*NodePhotoResponse, error)

	// DeletePhoto melakukan hapus lunak foto dengan pencatatan riwayat.
	DeletePhoto(ctx context.Context, nodeID, photoID, performedBy string) error

	// GetHistory mengambil riwayat perubahan node dengan paginasi.
	GetHistory(ctx context.Context, nodeID string, limit, offset int) ([]*MapChangeHistoryResponse, error)

	// ListTrashed mengambil daftar node yang ada di trash (sudah di-hapus lunak).
	ListTrashed(ctx context.Context, tenantID string) ([]*MapNodeResponse, error)

	// RestoreNode mengembalikan node dari trash dengan pencatatan riwayat.
	RestoreNode(ctx context.Context, id, performedBy string) error

	// GetLabelSettings mengambil konfigurasi label untuk tenant (kembalikan bawaan jika belum ada).
	GetLabelSettings(ctx context.Context, tenantID string) (*MapLabelSettingsResponse, error)

	// UpdateLabelSettings memperbarui konfigurasi label untuk tenant.
	UpdateLabelSettings(ctx context.Context, tenantID string, req UpdateLabelSettingsRequest) (*MapLabelSettingsResponse, error)
}

// =============================================================================
// CableRouteManager - business logic untuk manajemen cable route
// =============================================================================

// CableRouteManager mendefinisikan business logic untuk manajemen cable route.
// Menangani CRUD cable route dengan auto-kalkulasi jarak via Haversine.
type CableRouteManager interface {
	// CreateRoute membuat cable route baru dengan validasi node dan kalkulasi jarak otomatis.
	CreateRoute(ctx context.Context, tenantID string, req CreateCableRouteRequest) (*CableRouteResponse, error)

	// GetRoute mengambil detail cable route berdasarkan ID.
	GetRoute(ctx context.Context, id string) (*CableRouteResponse, error)

	// UpdateRoute memperbarui cable route dengan kalkulasi ulang jarak jika koordinat berubah.
	UpdateRoute(ctx context.Context, id string, req UpdateCableRouteRequest) (*CableRouteResponse, error)

	// DeleteRoute melakukan hapus lunak cable route.
	DeleteRoute(ctx context.Context, id string) error

	// ListRoutes mengambil daftar cable route berdasarkan bounding box dan filter.
	ListRoutes(ctx context.Context, params CableRouteListParams) ([]*CableRouteResponse, error)
}

// =============================================================================
// MapExportManager - business logic untuk export peta
// =============================================================================

// MapExportManager mendefinisikan business logic untuk export peta.
// Mendukung format KML, KMZ, GeoJSON, dan CSV.
// Dataset besar (>500 items) diproses secara async via asynq job.
type MapExportManager interface {
	// Export mengekspor data peta ke format yang diminta.
	// Jika dataset ≤500 items, mengembalikan file langsung (sync).
	// Jika dataset >500 items, mengembalikan job_id untuk polling status (async).
	Export(ctx context.Context, tenantID string, req ExportRequest) (*ExportResult, error)

	// GetExportStatus mengecek status export async berdasarkan job_id.
	GetExportStatus(ctx context.Context, jobID string) (*ExportStatus, error)
}

// =============================================================================
// MapImportManager - business logic untuk import peta
// =============================================================================

// MapImportManager mendefinisikan business logic untuk import peta.
// Mendukung format KML, KMZ, dan GeoJSON.
// Dataset besar (>100 items) diproses secara async via asynq job.
type MapImportManager interface {
	// Preview mem-parsing file import dan mengembalikan preview item yang terdeteksi.
	Preview(ctx context.Context, tenantID string, file multipart.File, filename string) (*ImportPreview, error)

	// Execute mengeksekusi import berdasarkan mapping yang dipilih user.
	Execute(ctx context.Context, importID string, mapping ImportMapping) (*ImportSummary, error)

	// GetImportStatus mengecek status import async berdasarkan job_id.
	GetImportStatus(ctx context.Context, jobID string) (*ImportStatus, error)
}

// =============================================================================
// GeocodingManager - business logic untuk reverse geocoding
// =============================================================================

// GeocodingManager mendefinisikan business logic untuk reverse geocoding.
// Menggunakan cache PostgreSQL (TTL 30 hari) dan provider Nominatim/Google.
type GeocodingManager interface {
	// ReverseGeocode mengkonversi koordinat GPS menjadi alamat lengkap.
	// Mengecek cache terlebih dahulu, jika miss maka memanggil provider eksternal.
	ReverseGeocode(ctx context.Context, tenantID string, lat, lng float64) (*GeocodingResult, error)
}

// =============================================================================
// ShareManager - business logic untuk share link
// =============================================================================

// ShareManager mendefinisikan business logic untuk share link peta.
// Menangani pembuatan, validasi, dan penghapusan share link hanya baca.
type ShareManager interface {
	// CreateShareLink membuat share link baru dengan opsi expiry dan password.
	CreateShareLink(ctx context.Context, tenantID, createdBy string, req CreateShareLinkRequest) (*ShareLinkResponse, error)

	// GetSharedMap mengambil data peta berdasarkan token share link.
	// Memvalidasi expiry dan password, lalu mengembalikan data yang difilter.
	GetSharedMap(ctx context.Context, token, password string) (*SharedMapData, error)

	// DeleteShareLink menghapus share link berdasarkan token.
	DeleteShareLink(ctx context.Context, token string) error

	// ListShareLinks mengambil daftar share link untuk satu tenant.
	ListShareLinks(ctx context.Context, tenantID string) ([]*ShareLinkResponse, error)
}
