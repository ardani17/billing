# Tasks — FTTH Visual Mapping

## Overview

Implementasi FTTH Visual Mapping di `services/network-service/` (backend) dan `apps/web/` (frontend). Fitur ini menambahkan layer geospasial di atas modul yang sudah ada (OLT Management, ODP, ONT Provisioning, Pelanggan) untuk memvisualisasikan seluruh jaringan fiber optik ISP pada peta interaktif.

Backend: REST API baru di bawah prefix `/api/v1/network-map/*` untuk manajemen map node, cable route, foto, export/import, share link, reverse geocoding, loss calculator, dan label settings.

Frontend: Halaman peta interaktif di route `/network-map` menggunakan Leaflet.js + OpenStreetMap + react-leaflet, dengan detail panel, drawing tools, layer control, topology view, heatmap, offline mode, dan export PNG/PDF.

## Tasks

- [x] 1. Domain Entities, Constants, Errors, dan Pure Functions
  - [x] 1.1 Buat file `internal/domain/map_node.go` — MapNode entity struct, NodeType constants ('olt', 'odp', 'ont'), MapNodeWithRef struct (join data dari OLT/ODP/ONT), MapNodeListParams struct (bounds, node_type, status, billing_status, package_id, area_id, odp_id), MapSearchResult struct
    - Maksimal 200 baris per file
    - _Requirements: 1.1, 2.1, 2.5, 5.2_

  - [x] 1.2 Buat file `internal/domain/cable_route.go` — CableRoute entity struct, RouteType constants ('backbone', 'drop'), CableRouteListParams struct (bounds, route_type, from_node_id, to_node_id)
    - Maksimal 200 baris per file
    - _Requirements: 1.2, 3.1_

  - [x] 1.3 Buat file `internal/domain/node_photo.go` — NodePhoto entity struct, AllowedPhotoTypes constant (JPEG, PNG, WebP), MaxPhotosPerNode constant (5), MaxPhotoSizeBytes constant (1MB)
    - Maksimal 200 baris per file
    - _Requirements: 1.3, 4.4, 4.6_

  - [x] 1.4 Buat file `internal/domain/map_change_history.go` — MapChangeHistory entity struct, ChangeAction constants ('created', 'location_moved', 'custom_fields_updated', 'photo_added', 'photo_removed', 'deleted', 'restored')
    - Maksimal 200 baris per file
    - _Requirements: 10.1, 10.3_

  - [x] 1.5 Buat file `internal/domain/map_label_settings.go` — MapLabelSettings entity struct, default label values per node type (OLT: name+brand_model+ont_count, ODP: name+splitter_type+capacity, ONT: customer_name+package), DefaultMinZoomLevel constant (15)
    - Maksimal 200 baris per file
    - _Requirements: 12.1, 12.3, 12.4_

  - [x] 1.6 Buat file `internal/domain/map_share_link.go` — MapShareLink entity struct, token generation helper (crypto/rand, 32 bytes hex)
    - Maksimal 200 baris per file
    - _Requirements: 9.1, 9.2_

  - [x] 1.7 Buat file `internal/domain/geocoding_cache.go` — GeocodingCache entity struct, RoundCoordinate helper (round to 5 decimal places), CacheTTLDays constant (30)
    - Maksimal 200 baris per file
    - _Requirements: 8.3_

  - [x] 1.8 Buat file `internal/domain/haversine.go` — Haversine(lat1, lng1, lat2, lng2) float64 pure function (radius bumi 6371000m), CalculateRouteDistance(coordinates [][2]float64) float64 pure function, ValidateCoordinate(lat, lng float64) error helper
    - Maksimal 200 baris per file
    - _Requirements: 3.3, 7.7, 25.1, 25.2_

  - [x] 1.9 Buat file `internal/domain/loss_calculator.go` — LossCalculatorInput struct, LossCalculatorResult struct, konstanta (FiberLossPerKm=0.35, ConnectorLossEach=0.5, SpliceLossEach=0.1, SafetyMargin=3.0), SplitterLoss map, CalculateLoss(input) LossCalculatorResult pure function
    - Maksimal 200 baris per file
    - _Requirements: 11.1, 11.2, 11.3, 11.4, 11.5_

  - [x] 1.10 Update file `internal/domain/errors.go` — Tambahkan map-specific domain errors: ErrMapNodeNotFound, ErrMapNodeDuplicate, ErrMapNodeDeleted, ErrInvalidNodeType, ErrInvalidCoordinates, ErrReferenceNotFound, ErrCableRouteNotFound, ErrInvalidRouteType, ErrInvalidCoordArray, ErrNodeNotFound, ErrPhotoLimitReached, ErrInvalidFileType, ErrFileTooLarge, ErrPhotoNotFound, ErrUnsupportedFormat, ErrInvalidImportFile, ErrImportNotFound, ErrExportNotFound, ErrShareLinkNotFound, ErrShareLinkExpired, ErrShareLinkPassword, ErrGeocodingFailed, ErrGeocodingRateLimit, ErrInvalidSplitterType, ErrInvalidLossInput
    - _Requirements: 1.6, 3.6, 4.4, 7.7, 9.3, 9.4, 11.2_

- [x] 2. Domain DTOs (Request/Response Types)
  - [x] 2.1 Buat file `internal/domain/map_node_dto.go` — Request DTOs (CreateMapNodeRequest, UpdateMapNodeRequest), Response DTOs (MapNodeResponse, MapNodeDetailResponse, MapNodeWithRefResponse), UpdateLabelSettingsRequest, MapLabelSettingsResponse
    - Maksimal 200 baris per file
    - _Requirements: 2.2, 2.3, 2.6, 12.2_

  - [x] 2.2 Buat file `internal/domain/cable_route_dto.go` — Request DTOs (CreateCableRouteRequest, UpdateCableRouteRequest), Response DTOs (CableRouteResponse)
    - Maksimal 200 baris per file
    - _Requirements: 3.2, 3.4_

  - [x] 2.3 Buat file `internal/domain/map_export_dto.go` — ExportRequest struct (format, layers, options), ExportResult struct, ExportStatus struct, ImportPreview struct, ImportMapping struct, ImportSummary struct, ImportStatus struct
    - Maksimal 200 baris per file
    - _Requirements: 6.1, 6.7, 7.1, 7.3, 7.5_

  - [x] 2.4 Buat file `internal/domain/map_share_dto.go` — CreateShareLinkRequest struct, ShareLinkResponse struct, SharedMapData struct, NodePhotoResponse struct, MapChangeHistoryResponse struct, GeocodingResult struct
    - Maksimal 200 baris per file
    - _Requirements: 4.2, 9.1, 9.6, 10.4_

- [x] 3. Repository Interfaces (Extend Existing)
  - [x] 3.1 Buat file `internal/domain/repository_mapping.go` — Tambahkan MapNodeRepository, CableRouteRepository, NodePhotoRepository, ChangeHistoryRepository, LabelSettingsRepository, ShareLinkRepository, GeocodingCacheRepository interfaces. Tambahkan MapNodeManager, CableRouteManager, MapExportManager, MapImportManager, GeocodingManager, ShareManager usecase interfaces.
    - _Requirements: 1.1, 1.2, 1.3, 2.1, 3.1, 6.1, 7.1, 8.1, 9.1, 10.1, 12.1_

- [x] 4. Property Tests untuk Domain Logic (Haversine, LossCalculator, Coordinate Validation)
  - [x] 4.1 Buat file `internal/domain/haversine_test.go` — Property test: Haversine Segment Additivity — untuk array koordinat [A,B,C,...,N], CalculateRouteDistance == sum Haversine per segment. Unit test: jarak Jakarta-Bandung ≈ 120km.
    - **Property 1: Haversine Segment Additivity**
    - **Validates: Requirements 25.1, 25.4**

  - [x] 4.2 Buat file `internal/domain/haversine_positive_test.go` — Property test: Haversine Positive Distance — array dengan ≥2 titik distinct menghasilkan distance > 0. Property test: Distance Calculation Determinism — dua kali kalkulasi input sama menghasilkan output identik.
    - **Property 2: Haversine Positive Distance**
    - **Property 3: Distance Calculation Determinism**
    - **Validates: Requirements 25.3, 3.6**

  - [x] 4.3 Buat file `internal/domain/loss_calculator_test.go` — Property test: Loss Calculator Decomposition — total_loss == fiber + splitter + connector + splice + safety_margin, dan total_loss >= 3.0 dB. Property test: Loss Calculator Signal Formula — estimated_signal == sfp_tx_power - (total_loss - safety_margin). Unit test: contoh kalkulasi spesifik.
    - **Property 4: Loss Calculator Decomposition**
    - **Property 5: Loss Calculator Signal Formula**
    - **Validates: Requirements 11.2, 11.3, 11.4, 11.5**

  - [x] 4.4 Buat file `internal/domain/coordinate_validation_test.go` — Property test: Coordinate Validation — accept jika -90≤lat≤90 dan -180≤lng≤180, reject jika di luar range.
    - **Property 7: Coordinate Validation**
    - **Validates: Requirements 7.7**

- [x] 5. Checkpoint — Domain Layer
  - Ensure all tests pass, ask the user if questions arise.

- [x] 6. Database Migrations (7 New Tables)
  - [x] 6.1 Buat SQL migration file `migrations/000016_create_map_nodes.up.sql` — CREATE TABLE map_nodes dengan semua kolom, UNIQUE constraint (tenant_id, node_type, reference_id WHERE deleted_at IS NULL), index location (tenant_id, latitude, longitude), index type (tenant_id, node_type), RLS policy
    - _Requirements: 1.1, 1.4, 1.5_

  - [x] 6.2 Buat SQL migration file `migrations/000017_create_cable_routes.up.sql` — CREATE TABLE cable_routes dengan semua kolom, indexes (tenant_id, from_node_id, to_node_id WHERE deleted_at IS NULL), RLS policy
    - _Requirements: 1.2, 1.4_

  - [x] 6.3 Buat SQL migration file `migrations/000018_create_node_photos.up.sql` — CREATE TABLE node_photos dengan semua kolom, index (map_node_id WHERE deleted_at IS NULL), RLS policy
    - _Requirements: 1.3, 1.4_

  - [x] 6.4 Buat SQL migration file `migrations/000019_create_map_change_history.up.sql` — CREATE TABLE map_change_history dengan semua kolom, index (map_node_id, created_at DESC), RLS policy. Append-only: no update/delete.
    - _Requirements: 10.1, 10.2_

  - [x] 6.5 Buat SQL migration file `migrations/000020_create_map_label_settings.up.sql` — CREATE TABLE map_label_settings dengan semua kolom, UNIQUE constraint (tenant_id), default JSONB values
    - _Requirements: 12.4_

  - [x] 6.6 Buat SQL migration file `migrations/000021_create_map_share_links.up.sql` — CREATE TABLE map_share_links dengan semua kolom, UNIQUE index (token), index (tenant_id)
    - _Requirements: 9.1_

  - [x] 6.7 Buat SQL migration file `migrations/000022_create_geocoding_cache.up.sql` — CREATE TABLE geocoding_cache dengan semua kolom, UNIQUE index (tenant_id, lat_round, lng_round), index (expires_at) untuk cleanup
    - _Requirements: 8.3_

- [x] 7. sqlc Queries dan Repository Wrappers
  - [x] 7.1 Buat sqlc queries file `queries/map_nodes.sql` — CRUD queries (Create, GetByID, Update, SoftDelete, Restore, ListByBounds, GetByReference, Search, ListTrashed, PermanentDeleteExpired, CountPhotosByNode)
    - _Requirements: 1.1, 2.1, 2.4, 2.5, 2.6, 5.1, 10.5, 10.6_

  - [x] 7.2 Buat sqlc queries file `queries/cable_routes.sql` — CRUD queries (Create, GetByID, Update, SoftDelete, ListByBounds, ListByNode)
    - _Requirements: 1.2, 3.1, 3.4, 3.5_

  - [x] 7.3 Buat sqlc queries file `queries/node_photos.sql` — CRUD queries (Create, ListByNode, SoftDelete, CountByNode)
    - _Requirements: 1.3, 4.1, 4.2, 4.3_

  - [x] 7.4 Buat sqlc queries file `queries/map_change_history.sql` — Create, ListByNode (ordered by created_at DESC, with pagination)
    - _Requirements: 10.1, 10.4_

  - [x] 7.5 Buat sqlc queries file `queries/map_label_settings.sql` — GetByTenantID, Upsert
    - _Requirements: 12.1, 12.2_

  - [x] 7.6 Buat sqlc queries file `queries/map_share_links.sql` — Create, GetByToken, Delete, ListByTenant, IncrementAccessCount
    - _Requirements: 9.1, 9.2, 9.5, 9.6_

  - [x] 7.7 Buat sqlc queries file `queries/geocoding_cache.sql` — Get (by lat_round+lng_round), Set (upsert), DeleteExpired
    - _Requirements: 8.3_

  - [x] 7.8 Jalankan `sqlc generate` dan buat repository wrapper `internal/repository/map_node_repo.go`
    - _Requirements: 1.1, 2.1_

  - [x] 7.9 Buat repository wrapper `internal/repository/cable_route_repo.go`
    - _Requirements: 1.2, 3.1_

  - [x] 7.10 Buat repository wrapper `internal/repository/node_photo_repo.go`
    - _Requirements: 1.3_

  - [x] 7.11 Buat repository wrapper `internal/repository/change_history_repo.go`
    - _Requirements: 10.1_

  - [x] 7.12 Buat repository wrapper `internal/repository/label_settings_repo.go`
    - _Requirements: 12.1_

  - [x] 7.13 Buat repository wrapper `internal/repository/share_link_repo.go`
    - _Requirements: 9.1_

  - [x] 7.14 Buat repository wrapper `internal/repository/geocoding_cache_repo.go`
    - _Requirements: 8.3_

- [x] 8. Checkpoint — Infrastructure Layer
  - Ensure all tests pass, ask the user if questions arise.

- [x] 9. MapNodeManager Usecase
  - [x] 9.1 Buat file `internal/usecase/map_node_manager.go` — Struct mapNodeManager dengan dependencies (MapNodeRepo, NodePhotoRepo, ChangeHistoryRepo, LabelSettingsRepo, OLTRepo, ODPRepo, ONTRepo), constructor NewMapNodeManager
    - Maksimal 200 baris per file
    - _Requirements: 2.1_

  - [x] 9.2 Implementasi `CreateNode` — Validate input (node_type, coordinates, reference_id exists), check unique constraint (tenant_id, node_type, reference_id), insert map_nodes, create change_history entry ('created'), return MapNodeResponse
    - _Requirements: 2.2, 1.5, 10.3_

  - [x] 9.3 Implementasi `GetNode` — Get map_node by ID, join data dari OLT/ODP/ONT repo berdasarkan node_type dan reference_id, include custom_fields, photos, change history
    - _Requirements: 2.6_

  - [x] 9.4 Implementasi `UpdateNode` — Validate coordinates, update lat/lng/custom_fields, create change_history entry ('location_moved' atau 'custom_fields_updated' sesuai field yang berubah)
    - _Requirements: 2.3, 10.3_

  - [x] 9.5 Implementasi `DeleteNode` dan `RestoreNode` — Soft delete (set deleted_at), create change_history ('deleted'). Restore: clear deleted_at, create change_history ('restored')
    - _Requirements: 2.4, 10.5, 10.6_

  - [x] 9.6 Implementasi `ListNodes` — Query by bounding box, apply filters (node_type, status, billing_status, package_id, area_id, odp_id), join data dari OLT/ODP/ONT
    - _Requirements: 2.1, 2.5_

  - [x] 9.7 Implementasi `Search` — Full-text search across customer name, customer ID, ODP name, OLT name, ONT serial number, address. Max 20 results, ordered by relevance.
    - _Requirements: 5.1, 5.2, 5.3, 5.4_

  - [x] 9.8 Implementasi `UploadPhoto`, `ListPhotos`, `DeletePhoto` — Validate file type (JPEG/PNG/WebP), check photo count limit (max 5), compress image ke max 1MB, store di `uploads/{tenant_id}/map-photos/{node_id}/{photo_id}.{ext}`, create change_history entries
    - Split ke file `internal/usecase/map_node_photo.go` jika melebihi 200 baris
    - _Requirements: 4.1, 4.2, 4.3, 4.4, 4.5, 4.6, 1.6_

  - [x] 9.9 Implementasi `GetHistory`, `ListTrashed` — Query change_history per node (paginated), query trashed nodes (deleted_at within 30 days)
    - _Requirements: 10.4, 10.5_

  - [x] 9.10 Implementasi `GetLabelSettings`, `UpdateLabelSettings` — Get settings by tenant_id (return defaults jika tidak ada), upsert settings
    - _Requirements: 12.1, 12.2, 12.3_

  - [x] 9.11 Buat file `internal/usecase/map_node_manager_test.go` — Unit tests: CreateNode happy path, duplicate error, invalid coordinates, UpdateNode, DeleteNode+RestoreNode, ListNodes with bounds, Search. Property test: Bounding Box Filtering, Photo Limit Enforcement, Search Result Limit.
    - **Property 8: Bounding Box Filtering**
    - **Property 9: Photo Limit Enforcement**
    - **Property 11: Search Result Limit and Completeness**
    - **Validates: Requirements 2.5, 1.6, 5.2, 5.3**

- [x] 10. CableRouteManager Usecase
  - [x] 10.1 Buat file `internal/usecase/cable_route_manager.go` — Struct cableRouteManager dengan dependencies (CableRouteRepo, MapNodeRepo), constructor NewCableRouteManager. Implementasi CreateRoute (validate from/to node exists, validate coordinates ≥2 points, calculate distance via CalculateRouteDistance, insert), GetRoute, UpdateRoute (recalculate distance), DeleteRoute, ListRoutes (by bounds, filters)
    - Maksimal 200 baris per file
    - _Requirements: 3.1, 3.2, 3.3, 3.4, 3.5, 25.1, 25.2_

  - [x] 10.2 Buat file `internal/usecase/cable_route_manager_test.go` — Unit tests: CreateRoute happy path, invalid coordinates, auto-distance calculation, UpdateRoute recalculation
    - _Requirements: 3.2, 3.3, 3.6_

- [x] 11. MapExportManager Usecase
  - [x] 11.1 Buat file `internal/usecase/map_export_manager.go` — Struct mapExportManager dengan dependencies (MapNodeRepo, CableRouteRepo, asynq.Client), constructor NewMapExportManager. Implementasi Export: jika dataset ≤500 items → sync generate file, jika >500 → enqueue asynq job. GetExportStatus: check job status.
    - Maksimal 200 baris per file
    - _Requirements: 6.1, 6.7_

  - [x] 11.2 Buat file `internal/usecase/map_export_kml.go` — Implementasi export KML: organize folders per node type, include description per node. Export KMZ: package KML + icons ke ZIP archive.
    - Maksimal 200 baris per file
    - _Requirements: 6.2, 6.3_

  - [x] 11.3 Buat file `internal/usecase/map_export_geojson.go` — Implementasi export GeoJSON: FeatureCollection dengan Point (nodes) dan LineString (cables). Export CSV: columns name, type, lat, lng, status, address, custom fields.
    - Maksimal 200 baris per file
    - _Requirements: 6.4, 6.5_

  - [x] 11.4 Buat file `internal/usecase/map_export_test.go` — Property test: GeoJSON Export/Import Round-Trip. Unit tests: KML output structure, CSV columns, async job creation for large datasets.
    - **Property 6: GeoJSON Export/Import Round-Trip**
    - **Validates: Requirements 6.6**

- [x] 12. MapImportManager Usecase
  - [x] 12.1 Buat file `internal/usecase/map_import_manager.go` — Struct mapImportManager dengan dependencies (MapNodeRepo, CableRouteRepo, asynq.Client), constructor NewMapImportManager. Implementasi Preview: parse KML/KMZ/GeoJSON, detect items (points, lines, polygons), return ImportPreview. Execute: apply type mapping, validate coordinates, auto-match names, insert nodes/cables. Async jika >100 items.
    - Maksimal 200 baris per file, split ke `map_import_parser.go` jika perlu
    - _Requirements: 7.1, 7.2, 7.3, 7.4, 7.5, 7.6, 7.7_

  - [x] 12.2 Buat file `internal/usecase/map_import_test.go` — Unit tests: KML parsing, KMZ extraction, GeoJSON parsing, coordinate validation, auto-match, import summary
    - _Requirements: 7.1, 7.5, 7.7_

- [x] 13. GeocodingManager Usecase
  - [x] 13.1 Buat file `internal/usecase/geocoding_manager.go` — Struct geocodingManager dengan dependencies (GeocodingCacheRepo, HTTP client), constructor NewGeocodingManager. Implementasi ReverseGeocode: round coordinates to 5 decimal places, check cache, jika miss → call Nominatim (rate limit 1 req/sec), store cache (TTL 30 hari), return GeocodingResult. Handle provider error gracefully (return coordinates tanpa address).
    - Maksimal 200 baris per file
    - _Requirements: 8.1, 8.2, 8.3, 8.4, 8.5_

  - [x] 13.2 Buat file `internal/usecase/geocoding_manager_test.go` — Property test: Geocoding Cache Key Consistency. Unit tests: cache hit, cache miss, provider error, rate limiting.
    - **Property 10: Geocoding Cache Key Consistency**
    - **Validates: Requirements 8.3**

- [x] 14. ShareManager Usecase
  - [x] 14.1 Buat file `internal/usecase/share_manager.go` — Struct shareManager dengan dependencies (ShareLinkRepo, MapNodeRepo, CableRouteRepo), constructor NewShareManager. Implementasi CreateShareLink (generate token, hash password jika ada, set expiry), GetSharedMap (validate token, check expiry, check password, filter by visible_layers, increment access_count), DeleteShareLink, ListShareLinks.
    - Maksimal 200 baris per file
    - _Requirements: 9.1, 9.2, 9.3, 9.4, 9.5, 9.6_

  - [x] 14.2 Buat file `internal/usecase/share_manager_test.go` — Property test: Share Link Expiry Enforcement. Unit tests: create, access, expired link, wrong password, delete, list.
    - **Property 12: Share Link Expiry Enforcement**
    - **Validates: Requirements 9.3**

- [x] 15. Checkpoint — Business Logic Layer
  - Ensure all tests pass, ask the user if questions arise.

- [x] 16. HTTP Handlers
  - [x] 16.1 Buat file `internal/handler/map_node_handler.go` — MapNodeHandler struct dengan methods: ListNodes (GET /nodes), CreateNode (POST /nodes), GetNode (GET /nodes/:id), UpdateNode (PUT /nodes/:id), DeleteNode (DELETE /nodes/:id), ListPhotos (GET /nodes/:id/photos), UploadPhoto (POST /nodes/:id/photos), DeletePhoto (DELETE /nodes/:id/photos/:photo_id), GetHistory (GET /nodes/:id/history)
    - Maksimal 200 baris per file, split ke `map_node_handler_photo.go` jika perlu
    - _Requirements: 2.1, 2.2, 2.3, 2.4, 2.6, 4.1, 4.2, 4.3, 10.4_

  - [x] 16.2 Buat file `internal/handler/cable_route_handler.go` — CableRouteHandler struct dengan methods: ListRoutes (GET /cables), CreateRoute (POST /cables), GetRoute (GET /cables/:id), UpdateRoute (PUT /cables/:id), DeleteRoute (DELETE /cables/:id)
    - Maksimal 200 baris per file
    - _Requirements: 3.1, 3.2, 3.4, 3.5_

  - [x] 16.3 Buat file `internal/handler/map_search_handler.go` — SearchHandler struct dengan method: Search (GET /search) — validate query min 2 chars, return max 20 results
    - Maksimal 200 baris per file
    - _Requirements: 5.1, 5.2, 5.3_

  - [x] 16.4 Buat file `internal/handler/map_export_handler.go` — ExportHandler struct dengan methods: Export (POST /export), GetExportStatus (GET /export/status/:job_id)
    - Maksimal 200 baris per file
    - _Requirements: 6.1, 6.7_

  - [x] 16.5 Buat file `internal/handler/map_import_handler.go` — ImportHandler struct dengan methods: Preview (POST /import), Execute (POST /import/execute), GetImportStatus (GET /import/status/:job_id)
    - Maksimal 200 baris per file
    - _Requirements: 7.1, 7.3, 7.6_

  - [x] 16.6 Buat file `internal/handler/geocoding_handler.go` — GeocodingHandler struct dengan method: ReverseGeocode (GET /geocode/reverse) — accept lat, lng params
    - Maksimal 200 baris per file
    - _Requirements: 8.1_

  - [x] 16.7 Buat file `internal/handler/share_handler.go` — ShareHandler struct dengan methods: CreateShareLink (POST /share), ListShareLinks (GET /share), GetSharedMap (GET /share/:token — public, no auth), DeleteShareLink (DELETE /share/:token)
    - Maksimal 200 baris per file
    - _Requirements: 9.1, 9.2, 9.5, 9.6_

  - [x] 16.8 Buat file `internal/handler/loss_calc_handler.go` — LossCalcHandler struct dengan method: CalculateLoss (POST /loss-calculator) — parse input, call domain CalculateLoss, return result
    - Maksimal 200 baris per file
    - _Requirements: 11.1, 11.2, 11.3_

  - [x] 16.9 Buat file `internal/handler/label_settings_handler.go` — LabelSettingsHandler struct dengan methods: GetLabelSettings (GET /settings/labels), UpdateLabelSettings (PUT /settings/labels)
    - Maksimal 200 baris per file
    - _Requirements: 12.1, 12.2_

  - [x] 16.10 Buat file `internal/handler/trash_handler.go` — TrashHandler struct dengan methods: ListTrashed (GET /trash), RestoreNode (POST /trash/:id/restore)
    - Maksimal 200 baris per file
    - _Requirements: 10.5, 10.6_

  - [x] 16.11 Buat file `internal/handler/map_node_handler_test.go` — Unit tests: request validation, response format, error mapping (400/404/409/410)
    - _Requirements: 2.1, 4.4_

  - [x] 16.12 Buat file `internal/handler/share_handler_test.go` — Unit tests: public access (no auth), expired link (410), wrong password (401)
    - _Requirements: 9.3, 9.4_

- [x] 17. Route Registration + Wiring
  - [x] 17.1 Update `internal/handler/router.go` — Tambahkan MapNodeHandler, CableRouteHandler, SearchHandler, ExportHandler, ImportHandler, GeocodingHandler, ShareHandler, LossCalcHandler, LabelSettingsHandler, TrashHandler ke RouterConfig struct. Register route group /api/v1/network-map/* dengan semua endpoint. Route GET /share/:token bersifat publik (tanpa auth middleware).
    - _Requirements: 2.1, 3.1, 5.1, 6.1, 7.1, 8.1, 9.1, 9.2, 11.1, 12.1, 10.5_

  - [x] 17.2 Update `cmd/main.go` — Wire mapping dependencies: MapNodeRepo, CableRouteRepo, NodePhotoRepo, ChangeHistoryRepo, LabelSettingsRepo, ShareLinkRepo, GeocodingCacheRepo, MapNodeManager, CableRouteManager, MapExportManager, MapImportManager, GeocodingManager, ShareManager. Buat semua handler instances. Register ke RouterConfig. Tambahkan export/import worker ke asynq ServeMux jika menggunakan async jobs.
    - _Requirements: 2.1, 3.1, 6.1, 7.1, 8.1, 9.1_

- [x] 18. Integration Tests
  - [x] 18.1 Buat file `internal/usecase/map_integration_test.go` — Integration test end-to-end: create map node → upload photo → create cable route → verify distance calculation → update node location → verify change history → soft delete → verify trash → restore. Test cross-tenant isolation (RLS).
    - _Requirements: 1.4, 2.1, 3.3, 4.1, 10.3, 10.5, 10.6_

  - [x] 18.2 Buat file `internal/usecase/map_export_import_integration_test.go` — Integration test: create nodes + cables → export GeoJSON → import GeoJSON → verify round-trip. Test async export for large dataset.
    - _Requirements: 6.4, 6.6, 7.1_

- [x] 19. Final Checkpoint — Backend
  - Ensure all tests pass, ask the user if questions arise.

- [x] 20. Frontend — MapPage + MapCanvas (Leaflet.js Setup)
  - [x] 20.1 Install dependencies: `react-leaflet`, `leaflet`, `leaflet.markercluster`, `@types/leaflet` di `apps/web/`
    - _Requirements: 13.1_

  - [x] 20.2 Buat file `apps/web/app/network-map/page.tsx` — MapPage component, route `/network-map`, responsive layout (desktop: 70% map + 30% panel, mobile: full screen map + floating toolbar)
    - _Requirements: 13.1, 13.6_

  - [x] 20.3 Buat file `apps/web/app/network-map/components/MapCanvas.tsx` — Leaflet map component via react-leaflet, OpenStreetMap tiles, bounding box data fetching, marker rendering per node type (OLT: tower/dark blue/large, ODP: square/blue/medium, ONT: circle/green|yellow|red|gray/small), polyline rendering (backbone: solid dark blue 4px, drop: solid green 2px online / dashed red 2px offline)
    - _Requirements: 13.1, 13.2, 13.3, 13.5_

  - [x] 20.4 Buat file `apps/web/app/network-map/components/MarkerCluster.tsx` — Leaflet.markercluster integration untuk ONT markers, cluster coloring (green: all online, yellow: some weak, red: some offline)
    - _Requirements: 13.4_

  - [x] 20.5 Buat file `apps/web/app/network-map/hooks/useMapNodes.ts` — Custom hook untuk fetch nodes dari API berdasarkan bounding box, debounce map move events, manage node state
    - _Requirements: 13.5, 2.1_

  - [x] 20.6 Buat file `apps/web/app/network-map/hooks/useCableRoutes.ts` — Custom hook untuk fetch cable routes dari API berdasarkan bounding box
    - _Requirements: 3.1_

  - [x] 20.7 Buat file `apps/web/app/network-map/lib/api.ts` — API client functions untuk semua network-map endpoints (nodes, cables, photos, search, export, import, geocode, share, loss-calculator, labels, trash)
    - _Requirements: 2.1, 3.1, 5.1, 6.1, 7.1, 8.1, 9.1, 11.1, 12.1, 10.5_

- [x] 21. Frontend — DetailPanel
  - [x] 21.1 Buat file `apps/web/app/network-map/components/DetailPanel.tsx` — Panel samping (desktop) / bottom sheet (mobile). Tabs: Info, Keterangan, Foto, Riwayat. OLT view: name, brand, status, PON ports, ONT count, alarms, action buttons. ODP view: name, splitter, port usage, address, ONT summary, custom fields, photos, actions. ONT view: customer name/ID, package, signal dBm, serial number, ODP, custom fields, photos, actions.
    - Split ke `DetailPanelOLT.tsx`, `DetailPanelODP.tsx`, `DetailPanelONT.tsx` jika perlu
    - _Requirements: 14.1, 14.2, 14.3, 14.4, 14.5_

  - [x] 21.2 Buat file `apps/web/app/network-map/components/CustomFieldsEditor.tsx` — Form editor untuk custom_fields (IP pool, VLAN, gateway, tipe kabel, lokasi detail, catatan bebas) di tab Keterangan
    - _Requirements: 14.4_

  - [x] 21.3 Buat file `apps/web/app/network-map/components/PhotoGallery.tsx` — Gallery foto per node di tab Foto, upload button, caption, delete. Max 5 foto indicator.
    - _Requirements: 14.4, 4.1, 4.2, 4.3_

  - [x] 21.4 Buat file `apps/web/app/network-map/components/ChangeHistory.tsx` — Timeline riwayat perubahan per node di tab Riwayat
    - _Requirements: 14.4, 10.4_

- [x] 22. Frontend — DrawingToolbar
  - [x] 22.1 Buat file `apps/web/app/network-map/components/DrawingToolbar.tsx` — Toolbar dengan buttons: Add Marker (📍), Draw Line (✏️), Measure Distance (📐), Delete (🗑️). Active mode state management.
    - _Requirements: 15.1_

  - [x] 22.2 Implementasi Add Marker mode — Klik peta → open form create ODP baru, pre-fill lat/lng, trigger reverse geocoding untuk address
    - _Requirements: 15.2_

  - [x] 22.3 Implementasi Draw Line mode — Klik multiple points → draw polyline, on complete → open form save cable route (from_node, to_node, route_type, core_count, description, auto-calculated distance)
    - _Requirements: 15.3_

  - [x] 22.4 Implementasi Measure Distance mode — Klik 2 points → display straight-line distance in km
    - _Requirements: 15.4_

  - [x] 22.5 Implementasi Delete mode — Klik node/cable → confirmation dialog → soft delete via API
    - _Requirements: 15.5_

- [x] 23. Frontend — LayerControl + Filter
  - [x] 23.1 Buat file `apps/web/app/network-map/components/LayerControl.tsx` — Panel toggle per layer: OLT, ODP, ONT Online, ONT Offline, Kabel Backbone, Kabel Drop, Area, Satellite, Heatmap. Default visibility sesuai spec. Filter controls: ONT status, billing status, package, area, ODP. Visible count indicator ("Menampilkan: X dari Y pelanggan"). Reset Filter button.
    - _Requirements: 16.1, 16.2, 16.3, 16.4, 16.5_

- [x] 24. Frontend — SearchBar
  - [x] 24.1 Buat file `apps/web/app/network-map/components/SearchBar.tsx` — Search input (🔍) dengan autocomplete dari API search endpoint. Debounce 300ms. Min 2 chars. Results: type icon, name, identifier, address. On select: center map + zoom + highlight node.
    - _Requirements: 17.1, 17.2, 17.3, 17.4_

- [x] 25. Frontend — TopologyView
  - [x] 25.1 Buat file `apps/web/app/network-map/components/TopologyView.tsx` — Toggle antara map view dan topology view. Collapsible tree: OLT → PON Port → ODP → ONT. Status colors matching map markers. Click node → navigate to detail page. Summary counts per level (total ONT per PON port, online/offline per ODP).
    - _Requirements: 18.1, 18.2, 18.3, 18.4, 18.5_

- [x] 26. Frontend — HeatmapOverlay
  - [x] 26.1 Buat file `apps/web/app/network-map/components/HeatmapOverlay.tsx` — Heatmap overlay menggunakan leaflet-heat plugin. Colors: green (-8 to -20 dBm), yellow (-20 to -25 dBm), orange (-25 to -27 dBm), red (below -27 dBm). Intensity based on average signal per cluster. Legend component.
    - _Requirements: 19.1, 19.2, 19.3_

- [x] 27. Frontend — Edit Location (Drag & Drop)
  - [x] 27.1 Buat file `apps/web/app/network-map/components/DraggableMarker.tsx` — Draggable marker mode saat "Edit Lokasi" diklik. On drag end → trigger reverse geocoding → show updated address. Confirm/Cancel buttons. PUT request ke API on confirm.
    - _Requirements: 20.1, 20.2, 20.3, 20.4_

- [x] 28. Frontend — Navigation + My Location
  - [x] 28.1 Buat file `apps/web/app/network-map/components/MyLocation.tsx` — "Lokasi Saya" button (📍), request browser GPS, display pulsing blue marker. Distance from user to node di DetailPanel. "Navigasi" button → deep link ke Google Maps atau Waze. Handle GPS permission denied gracefully.
    - _Requirements: 21.1, 21.2, 21.3, 21.4_

- [x] 29. Frontend — Offline Mode
  - [x] 29.1 Buat file `apps/web/app/network-map/lib/offline-manager.ts` — Service Worker registration, IndexedDB schema untuk cached tiles + node data + cable routes + photo thumbnails. Download area selection UI. Max 100MB per area, expire after 7 days.
    - _Requirements: 22.1, 22.2, 22.7_

  - [x] 29.2 Buat file `apps/web/app/network-map/hooks/useOfflineMode.ts` — Custom hook untuk offline detection, display "⚡ Mode Offline" indicator, allow local add/edit/move nodes (store di IndexedDB), auto-sync on reconnect ("✅ Tersinkronisasi"), last-write-wins conflict resolution.
    - _Requirements: 22.3, 22.4, 22.5, 22.6_

- [x] 30. Frontend — Export PNG/PDF
  - [x] 30.1 Buat file `apps/web/app/network-map/components/ExportMapDialog.tsx` — Export dialog dengan opsi: PNG (screenshot viewport), PDF (A3/A4 dengan legend, title, node list). PNG: capture via html2canvas atau leaflet-image. PDF: generate via jsPDF atau react-pdf.
    - _Requirements: 23.1, 23.2, 23.3_

- [x] 31. Final Checkpoint — Frontend
  - Ensure all tests pass, ask the user if questions arise.

## Notes

- Tasks marked with `*` are optional and can be skipped for faster MVP
- Each task references specific requirements for traceability
- Checkpoints ensure incremental validation
- Property tests validate universal correctness properties from the design document (12 properties)
- Unit tests validate specific examples and edge cases
- Maksimal 200 baris per file Go — split ke file terpisah jika melebihi
- Gunakan `pgregory.net/rapid` untuk property-based testing (sudah ada di go.mod)
- Gunakan sqlc untuk query generation (sudah ada di codebase)
- Fiber v2 untuk HTTP handlers (sudah ada di codebase)
- asynq untuk async export/import jobs (sudah ada di codebase)
- Semua komentar dalam bahasa Indonesia
- Migration numbering mulai dari 000016 (setelah 000015_create_provisioning_settings dari olt-provisioning)
- Layer ini extend interface dan file yang sudah ada, bukan membuat ulang
- Frontend menggunakan react-leaflet + Leaflet.js + OpenStreetMap (gratis, tanpa API key)
- Frontend mengikuti pattern Next.js App Router yang sudah ada di `apps/web`
- Graceful degradation: peta tetap berfungsi meskipun modul OLT/MikroTik belum aktif (Requirements 24.1-24.4)
- Route GET /api/v1/network-map/share/:token bersifat publik (tanpa auth middleware)
- Photo storage: local filesystem `uploads/{tenant_id}/map-photos/{node_id}/{photo_id}.{ext}`
