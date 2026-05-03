# Requirements Document

## Introduction

Dokumen ini mendefinisikan requirements untuk **FTTH Visual Mapping** di platform ISPBoss. Fitur ini menyediakan peta interaktif untuk memvisualisasikan seluruh jaringan fiber optik ISP: dari OLT di NOC sampai ONT di rumah pelanggan. Peta membantu teknisi merencanakan pemasangan, troubleshoot gangguan, dan memantau kapasitas jaringan secara visual.

FTTH Visual Mapping terdiri dari dua komponen utama:
- **Frontend** (`apps/web`): Halaman peta interaktif menggunakan Leaflet.js + OpenStreetMap + react-leaflet di Next.js
- **Backend** (`services/network-service`): REST API untuk manajemen node peta, jalur kabel, export/import, share link, dan data geospasial

Fitur ini mengambil data dari modul yang sudah ada (OLT Management, ODP, ONT, Pelanggan) dan menambahkan layer geospasial: jalur kabel, custom fields per node, foto, anotasi, dan fitur peta lanjutan. Peta dirancang untuk **graceful degradation** — tetap berfungsi meskipun modul OLT atau MikroTik belum aktif.

## Glossary

- **Map_Page**: Halaman peta interaktif di frontend (`/network-map`) yang menampilkan node jaringan, jalur kabel, dan layer overlay
- **Map_API**: REST API di Network Service (`/api/v1/network-map/*`) yang menyediakan data geospasial untuk peta
- **Map_Node**: Entitas titik di peta yang merepresentasikan OLT, ODP, atau ONT beserta koordinat dan metadata
- **Cable_Route**: Entitas polyline di peta yang merepresentasikan jalur kabel fiber antara dua node, dengan koordinat titik-titik dan metadata
- **Node_Custom_Field**: Field keterangan tambahan per node (IP pool, VLAN, gateway, tipe kabel, lokasi detail, catatan) yang diisi manual oleh admin/teknisi
- **Node_Photo**: Foto yang di-upload per node untuk dokumentasi instalasi (max 5 foto, auto-compress ke 1 MB)
- **Label_Setting**: Konfigurasi per tenant yang menentukan informasi apa yang tampil di label node di peta (nama, IP pool, kapasitas, signal)
- **Layer_Control**: Kontrol toggle visibility per layer di peta: OLT, ODP, ONT, kabel backbone, kabel drop, area, satellite, heatmap
- **Detail_Panel**: Panel samping (desktop) atau bottom sheet (mobile) yang menampilkan informasi detail saat node diklik
- **Drawing_Tool**: Toolbar di peta untuk menambah marker, menggambar polyline jalur kabel, dan mengukur jarak
- **Topology_View**: Tampilan hierarki tree non-peta: OLT → PON Port → ODP → ONT
- **Heatmap_Overlay**: Layer overlay di peta yang menampilkan kualitas signal per area berdasarkan rata-rata signal ONT
- **Loss_Calculator**: Kalkulator optical loss budget yang menghitung total loss fiber, splitter, konektor, dan splice untuk perencanaan jalur
- **Share_Link**: Link read-only ke peta dengan opsi expiry dan password untuk berbagi dengan pihak eksternal
- **Embed_Code**: Kode iframe untuk menyematkan peta read-only di website eksternal
- **Offline_Cache**: Data peta dan node yang di-download ke browser (Service Worker + IndexedDB) untuk akses offline
- **Reverse_Geocoder**: Komponen yang mengkonversi koordinat GPS menjadi alamat lengkap menggunakan Nominatim (default) atau provider lain
- **Geocoding_Cache**: Cache hasil reverse geocoding selama 30 hari untuk mengurangi request ke provider
- **Change_History**: Riwayat perubahan per node yang mencatat setiap modifikasi beserta waktu dan pelaku
- **Soft_Delete**: Mekanisme penghapusan node yang menyimpan data di "Trash" selama 30 hari sebelum permanent delete
- **Marker_Cluster**: Pengelompokan marker ONT yang berdekatan menjadi satu marker dengan angka saat zoom out
- **Network_Service**: Go microservice (`services/network-service/`) yang menangani semua integrasi perangkat jaringan
- **Tenant**: Organisasi ISP yang menggunakan platform ISPBoss (multi-tenant SaaS)

## Requirements

### Requirement 1: Backend — Database Schema untuk Map Node dan Cable Route

**User Story:** As a platform engineer, I want database tables for storing map nodes (with custom fields, photos) and cable routes, so that the system can persist all geospatial data for the FTTH visual mapping feature.

#### Acceptance Criteria

1. THE Map_API SHALL store map node metadata in a `map_nodes` table with columns: id (UUID), tenant_id (UUID), node_type (VARCHAR 20: 'olt', 'odp', 'ont'), reference_id (UUID — FK ke tabel olts/odps/onts sesuai node_type), latitude (DOUBLE PRECISION), longitude (DOUBLE PRECISION), custom_fields (JSONB nullable — menyimpan IP pool, VLAN, gateway, tipe kabel, lokasi detail, catatan bebas), deleted_at (TIMESTAMPTZ nullable), created_at (TIMESTAMPTZ), updated_at (TIMESTAMPTZ)
2. THE Map_API SHALL store cable route entities in a `cable_routes` table with columns: id (UUID), tenant_id (UUID), from_node_id (UUID FK to map_nodes), to_node_id (UUID FK to map_nodes), route_type (VARCHAR 20: 'backbone', 'drop'), coordinates (JSONB — array of [lat, lng] waypoints), distance_meters (DOUBLE PRECISION — auto-calculated from coordinates), core_count (INTEGER nullable), description (TEXT nullable), deleted_at (TIMESTAMPTZ nullable), created_at (TIMESTAMPTZ), updated_at (TIMESTAMPTZ)
3. THE Map_API SHALL store node photos in a `node_photos` table with columns: id (UUID), tenant_id (UUID), map_node_id (UUID FK to map_nodes), file_path (VARCHAR 500), file_size_bytes (INTEGER), caption (VARCHAR 200 nullable), uploaded_by (VARCHAR 100), deleted_at (TIMESTAMPTZ nullable), created_at (TIMESTAMPTZ)
4. THE Map_API SHALL enforce Row-Level Security on the `map_nodes`, `cable_routes`, and `node_photos` tables so that queries only return rows matching the current tenant context
5. THE Map_API SHALL enforce a unique constraint on (tenant_id, node_type, reference_id) WHERE deleted_at IS NULL on the `map_nodes` table to prevent duplicate map entries for the same network entity
6. THE Map_API SHALL enforce a maximum of 5 photos per map node — IF a photo upload would exceed 5 photos, THEN THE Map_API SHALL return an error indicating the photo limit is reached

### Requirement 2: Backend — REST API untuk Map Node CRUD

**User Story:** As a frontend developer, I want REST API endpoints for creating, reading, updating, and deleting map nodes, so that the map page can manage node positions and custom fields.

#### Acceptance Criteria

1. THE Map_API SHALL expose GET /api/v1/network-map/nodes accepting query parameters: bounds (bounding box lat/lng), node_type (filter by olt/odp/ont), status (filter by ONT status: online/offline/weak/pending), billing_status (filter by aktif/isolir/pending), package_id, area_id, odp_id, and returning a list of map nodes with joined data (nama, status, signal) from the referenced entity
2. THE Map_API SHALL expose POST /api/v1/network-map/nodes accepting node_type, reference_id, latitude, longitude, custom_fields and returning the created map node with HTTP 201
3. THE Map_API SHALL expose PUT /api/v1/network-map/nodes/:id accepting latitude, longitude, custom_fields and returning the updated map node
4. THE Map_API SHALL expose DELETE /api/v1/network-map/nodes/:id performing a soft delete (set deleted_at) and returning HTTP 204
5. WHEN a GET request includes bounds parameter, THE Map_API SHALL return only nodes within the specified geographic bounding box to optimize data transfer for the visible map area
6. THE Map_API SHALL expose GET /api/v1/network-map/nodes/:id returning full node detail including custom_fields, photos, linked entity data (OLT/ODP/ONT info), and change history

### Requirement 3: Backend — REST API untuk Cable Route CRUD

**User Story:** As a frontend developer, I want REST API endpoints for creating, reading, updating, and deleting cable routes, so that the map page can manage fiber cable paths between nodes.

#### Acceptance Criteria

1. THE Map_API SHALL expose GET /api/v1/network-map/cables accepting query parameters: bounds (bounding box), route_type (backbone/drop), from_node_id, to_node_id, and returning a list of cable routes with coordinates
2. THE Map_API SHALL expose POST /api/v1/network-map/cables accepting from_node_id, to_node_id, route_type, coordinates (array of [lat, lng] waypoints), core_count, description and returning the created cable route with HTTP 201
3. WHEN a cable route is created or updated, THE Map_API SHALL auto-calculate distance_meters from the coordinates array using the Haversine formula
4. THE Map_API SHALL expose PUT /api/v1/network-map/cables/:id accepting coordinates, core_count, description and returning the updated cable route
5. THE Map_API SHALL expose DELETE /api/v1/network-map/cables/:id performing a soft delete and returning HTTP 204
6. FOR ALL valid cable route coordinate arrays, calculating distance then re-calculating from the same coordinates SHALL produce the same distance value (idempotent distance calculation)

### Requirement 4: Backend — Photo Upload per Node

**User Story:** As a field technician, I want to upload photos for each network node from my phone, so that installation documentation is linked directly to the node on the map.

#### Acceptance Criteria

1. THE Map_API SHALL expose POST /api/v1/network-map/nodes/:id/photos accepting a multipart file upload with optional caption, auto-compressing the image to a maximum of 1 MB, and returning the created photo record with HTTP 201
2. THE Map_API SHALL expose GET /api/v1/network-map/nodes/:id/photos returning a list of photos for the specified node including file_path, caption, uploaded_by, and created_at
3. THE Map_API SHALL expose DELETE /api/v1/network-map/nodes/:id/photos/:photo_id performing a soft delete on the photo record and returning HTTP 204
4. WHEN a photo is uploaded, THE Map_API SHALL validate the file type (JPEG, PNG, WebP only) and reject other formats with a descriptive error
5. WHEN a photo is uploaded, THE Map_API SHALL store the file in a tenant-isolated path: `uploads/{tenant_id}/map-photos/{node_id}/{photo_id}.{ext}`
6. IF the uploaded image exceeds 1 MB, THEN THE Map_API SHALL compress the image to fit within 1 MB while preserving acceptable visual quality

### Requirement 5: Backend — Search Endpoint

**User Story:** As an ISP admin, I want to search for customers, ODP, OLT by name, ID, address, or serial number on the map, so that I can quickly locate any network entity.

#### Acceptance Criteria

1. THE Map_API SHALL expose GET /api/v1/network-map/search accepting a query string parameter (min 2 characters) and returning matching results across: customer name, customer ID, ODP name, OLT name, ONT serial number, and address
2. WHEN search results are returned, THE Map_API SHALL include for each result: type (olt/odp/ont), name, identifier, latitude, longitude, and a brief description
3. THE Map_API SHALL return a maximum of 20 search results, ordered by relevance (exact match first, then partial match)
4. THE Map_API SHALL respond to search requests within 200ms for datasets up to 10,000 nodes

### Requirement 6: Backend — Export Peta

**User Story:** As an ISP admin, I want to export map data in multiple formats (KML, KMZ, GeoJSON, CSV, PNG, PDF), so that I can use the data in Google Earth, GIS tools, or printed documentation.

#### Acceptance Criteria

1. THE Map_API SHALL expose POST /api/v1/network-map/export accepting format (kml, kmz, geojson, csv), layers (array of node types and cable routes to include), and options (include_icons, include_descriptions, include_photos for KMZ)
2. WHEN exporting to KML format, THE Map_API SHALL organize data in folders per node type (OLT, ODP, ONT, Kabel) and include description per node with: nama, tipe, status, signal, paket, alamat
3. WHEN exporting to KMZ format, THE Map_API SHALL package the KML file with custom icons per node type into a single ZIP archive
4. WHEN exporting to GeoJSON format, THE Map_API SHALL produce a valid GeoJSON FeatureCollection with Point features for nodes and LineString features for cable routes
5. WHEN exporting to CSV format, THE Map_API SHALL produce a CSV file with columns: name, type, latitude, longitude, status, address, and custom fields
6. FOR ALL valid map node datasets, exporting to GeoJSON then importing the same GeoJSON SHALL produce an equivalent set of nodes and cable routes (round-trip property)
7. WHEN the export dataset exceeds 500 items, THE Map_API SHALL process the export asynchronously and return a job ID that the frontend can poll for completion

### Requirement 7: Backend — Import Peta

**User Story:** As an ISP admin, I want to import map data from KML, KMZ, or GeoJSON files, so that I can migrate existing mapping data from Google Earth or other GIS tools into ISPBoss.

#### Acceptance Criteria

1. THE Map_API SHALL expose POST /api/v1/network-map/import accepting a file upload (KML, KMZ, or GeoJSON) and returning a preview of detected items: count of placemarks (points), linestrings (lines), and polygons (areas)
2. WHEN importing a KMZ file, THE Map_API SHALL extract the ZIP archive and parse the contained KML file
3. THE Map_API SHALL expose POST /api/v1/network-map/import/execute accepting the import_id and a type mapping configuration (which KML placemarks map to ODP, ONT, etc.) and executing the import
4. WHERE auto-match is enabled, THE Map_API SHALL attempt to match imported placemark names with existing customer names or ODP names in the database and link them automatically
5. WHEN import is complete, THE Map_API SHALL return a summary: total items, successfully imported count, skipped count, and error details per failed item
6. IF the import file contains more than 100 items, THEN THE Map_API SHALL process the import asynchronously and return a job ID for progress tracking
7. THE Map_API SHALL validate imported coordinates are within valid geographic ranges (latitude -90 to 90, longitude -180 to 180) and reject items with invalid coordinates

### Requirement 8: Backend — Reverse Geocoding

**User Story:** As a field technician, I want to click on the map and get the full address automatically, so that I can quickly fill in location details when adding or editing nodes.

#### Acceptance Criteria

1. THE Map_API SHALL expose GET /api/v1/network-map/geocode/reverse accepting latitude and longitude parameters and returning the full address (street, kelurahan, kecamatan, city, province, postal code)
2. THE Map_API SHALL use Nominatim (OpenStreetMap) as the default reverse geocoding provider with a rate limit of 1 request per second
3. THE Map_API SHALL cache reverse geocoding results in the Geocoding_Cache for 30 days — WHEN the same coordinates (rounded to 5 decimal places) are requested again within 30 days, THE Map_API SHALL return the cached result without calling the external provider
4. IF the reverse geocoding provider returns an error or times out, THEN THE Map_API SHALL return the coordinates without an address and include an error indicator in the response
5. WHERE a tenant has configured Google Geocoding API in settings, THE Map_API SHALL use Google Geocoding instead of Nominatim for higher accuracy

### Requirement 9: Backend — Share Link dan Embed

**User Story:** As an ISP admin, I want to create read-only share links and embed codes for the map, so that I can share network visualization with investors, partners, or embed it on the ISP website.

#### Acceptance Criteria

1. THE Map_API SHALL expose POST /api/v1/network-map/share accepting: visible_layers (array of layer types to show), expiry_days (nullable — null means no expiry), password (nullable), and returning a share token and URL
2. THE Map_API SHALL expose GET /api/v1/network-map/share/:token returning the map data for the shared view, filtered to only include the layers specified at creation time
3. WHEN a share link has an expiry set, THE Map_API SHALL reject access to expired share links with HTTP 410 Gone
4. WHEN a share link has a password set, THE Map_API SHALL require the password as a query parameter or header and reject access with HTTP 401 if the password is incorrect
5. THE Map_API SHALL expose DELETE /api/v1/network-map/share/:token allowing the admin to revoke a share link before its expiry
6. THE Map_API SHALL expose GET /api/v1/network-map/share listing all active share links for the tenant with their creation date, expiry, and access count

### Requirement 10: Backend — Change History dan Soft Delete

**User Story:** As an ISP admin, I want to see the history of changes for each map node and restore accidentally deleted nodes, so that I can audit modifications and recover from mistakes.

#### Acceptance Criteria

1. THE Map_API SHALL store change history in a `map_change_history` table with columns: id (UUID), tenant_id (UUID), map_node_id (UUID), action (VARCHAR 50: 'created', 'location_moved', 'custom_fields_updated', 'photo_added', 'photo_removed', 'deleted', 'restored'), old_value (JSONB nullable), new_value (JSONB nullable), performed_by (VARCHAR 100), created_at (TIMESTAMPTZ)
2. THE Map_API SHALL enforce Row-Level Security on the `map_change_history` table so that queries only return rows matching the current tenant context
3. WHEN a map node's location, custom fields, or photos are modified, THE Map_API SHALL create a change history entry recording the old and new values
4. THE Map_API SHALL expose GET /api/v1/network-map/nodes/:id/history returning the change history for a specific node, ordered by most recent first
5. THE Map_API SHALL expose GET /api/v1/network-map/trash returning a list of soft-deleted nodes within the last 30 days
6. THE Map_API SHALL expose POST /api/v1/network-map/trash/:id/restore restoring a soft-deleted node by clearing its deleted_at field and creating a 'restored' change history entry
7. WHEN a soft-deleted node has been in trash for more than 30 days, THE Map_API SHALL permanently delete the node and its associated photos and change history via a scheduled cleanup job

### Requirement 11: Backend — Optical Loss Budget Calculator

**User Story:** As a field technician, I want to calculate the optical loss budget for a planned fiber route, so that I can verify whether the route is feasible before installing cable.

#### Acceptance Criteria

1. THE Map_API SHALL expose POST /api/v1/network-map/loss-calculator accepting: distance_olt_to_odp_km, distance_odp_to_ont_km, splitter_count, splitter_type (1:4, 1:8, 1:16, 1:32), connector_count, splice_count, sfp_tx_power_dbm, ont_sensitivity_dbm
2. THE Map_API SHALL calculate total loss using standard parameters: fiber loss = total_distance × 0.35 dB/km, splitter loss per type (1:4 = 7.0 dB, 1:8 = 10.5 dB, 1:16 = 13.5 dB, 1:32 = 17.0 dB), connector loss = connector_count × 0.5 dB, splice loss = splice_count × 0.1 dB, safety margin = 3.0 dB
3. THE Map_API SHALL return: total_loss_db, budget_available_db (sfp_tx_power - ont_sensitivity), remaining_margin_db (budget_available - total_loss), estimated_signal_at_ont_dbm (sfp_tx_power - total_loss + safety_margin), and feasibility status (feasible if remaining_margin > 0)
4. FOR ALL valid loss calculator inputs, THE Loss_Calculator SHALL produce total_loss_db that is greater than or equal to the safety margin (3.0 dB) since at minimum the safety margin is always included
5. FOR ALL valid loss calculator inputs, THE Loss_Calculator SHALL produce estimated_signal_at_ont_dbm that equals sfp_tx_power_dbm minus (total_loss_db minus safety_margin)

### Requirement 12: Backend — Label Settings per Tenant

**User Story:** As an ISP admin, I want to configure which information appears on node labels on the map, so that I can customize the map display to show the most relevant data for my team.

#### Acceptance Criteria

1. THE Map_API SHALL expose GET /api/v1/network-map/settings/labels returning the current label configuration per node type (OLT, ODP, ONT) for the authenticated tenant
2. THE Map_API SHALL expose PUT /api/v1/network-map/settings/labels accepting label configuration per node type: for OLT (name, brand_model, ont_count, ip_address, uptime), for ODP (name, splitter_type, capacity, ip_pool, pon_port, vlan, notes), for ONT (customer_name, package, signal_dbm, customer_id, ip_address, serial_number), and a minimum zoom level for label display
3. WHEN a tenant has no label settings, THE Map_API SHALL use default values: OLT shows name + brand_model + ont_count, ODP shows name + splitter_type + capacity, ONT shows customer_name + package, minimum zoom level 15
4. THE Map_API SHALL store label settings in a `map_label_settings` table with columns: id (UUID), tenant_id (UUID UNIQUE), olt_labels (JSONB), odp_labels (JSONB), ont_labels (JSONB), min_zoom_level (INTEGER DEFAULT 15), created_at (TIMESTAMPTZ), updated_at (TIMESTAMPTZ)

### Requirement 13: Frontend — Halaman Peta Interaktif dengan Leaflet.js

**User Story:** As an ISP admin, I want an interactive map page showing all network nodes (OLT, ODP, ONT) with status colors and connection lines, so that I can visualize the entire FTTH network geographically.

#### Acceptance Criteria

1. THE Map_Page SHALL render an interactive map using Leaflet.js with OpenStreetMap tiles via react-leaflet in the Next.js frontend at route `/network-map`
2. THE Map_Page SHALL display node markers with distinct icons and colors per type: OLT (tower icon, dark blue, large), ODP (square icon, blue, medium), ONT Online (circle, green, small), ONT Weak Signal (circle, yellow, small), ONT Offline/LOS (circle, red, small), ONT Pending (circle, gray, small)
3. THE Map_Page SHALL display connection lines between nodes: backbone lines (OLT→ODP) as solid dark blue 4px polylines, drop lines (ODP→ONT) as solid green 2px polylines for online ONT and dashed red 2px polylines for offline ONT
4. THE Map_Page SHALL use Leaflet.markercluster to group nearby ONT markers when zoomed out, displaying the count and coloring the cluster: green (all online), yellow (some weak), red (some offline)
5. THE Map_Page SHALL load node data from the Map_API using bounding box queries, fetching only nodes visible in the current map viewport
6. THE Map_Page SHALL use a responsive layout: split view on desktop (70% map, 30% detail panel) and full-screen map with floating toolbar and bottom sheet on mobile

### Requirement 14: Frontend — Detail Panel

**User Story:** As an ISP admin, I want to click on any node on the map and see its detailed information in a side panel, so that I can quickly inspect node status, custom fields, photos, and linked entities.

#### Acceptance Criteria

1. WHEN a user clicks an OLT marker, THE Map_Page SHALL display the Detail_Panel showing: OLT name, brand and model, status, PON port count, total ONT count, alarm count, and action buttons (Lihat Detail OLT, Lihat ONT di OLT ini)
2. WHEN a user clicks an ODP marker, THE Map_Page SHALL display the Detail_Panel showing: ODP name, splitter type, port usage (used/total), address, ONT status summary (online/weak/offline counts), custom fields (IP pool, VLAN, gateway), photos, and action buttons (Lihat Detail ODP, Tambah ONT, Edit Lokasi)
3. WHEN a user clicks an ONT marker, THE Map_Page SHALL display the Detail_Panel showing: customer name and ID, package name, signal strength (dBm), ONT serial number, ODP name and port, custom fields, photos, and action buttons (Lihat Pelanggan, Lihat Detail ONT, Edit Lokasi)
4. THE Detail_Panel SHALL include tabs for: Info (default), Keterangan (custom fields editor), Foto (photo gallery and upload), Riwayat (change history)
5. WHEN displayed on mobile, THE Detail_Panel SHALL render as a bottom sheet that can be swiped up for full detail

### Requirement 15: Frontend — Drawing Tools

**User Story:** As an ISP admin, I want drawing tools on the map to add new ODP markers, draw cable routes as polylines, and measure distances, so that I can plan and document the physical network layout.

#### Acceptance Criteria

1. THE Map_Page SHALL provide a toolbar with drawing tools: Add Marker (📍), Draw Line (✏️), Measure Distance (📐), and Delete (🗑️)
2. WHEN the Add Marker tool is active and the user clicks on the map, THE Map_Page SHALL open a form to create a new ODP at the clicked coordinates, pre-filling the latitude and longitude and triggering reverse geocoding for the address
3. WHEN the Draw Line tool is active and the user clicks multiple points on the map, THE Map_Page SHALL draw a polyline representing a cable route, and upon completion open a form to save the cable route with: from_node, to_node, route_type (backbone/drop), core_count, description, and auto-calculated distance
4. WHEN the Measure Distance tool is active and the user clicks two points on the map, THE Map_Page SHALL display the straight-line distance between the two points in kilometers
5. WHEN the Delete tool is active and the user clicks a node or cable route, THE Map_Page SHALL prompt for confirmation before performing a soft delete via the Map_API

### Requirement 16: Frontend — Layer Control dan Filter

**User Story:** As an ISP admin, I want to toggle visibility of different map layers and filter nodes by status, billing, package, area, and ODP, so that I can focus on specific aspects of the network.

#### Acceptance Criteria

1. THE Map_Page SHALL provide a Layer_Control panel with toggles for: OLT (default visible), ODP (default visible), ONT Online (default visible), ONT Offline (default visible), Kabel Backbone (default visible), Kabel Drop (default hidden), Area/Wilayah (default hidden), Satellite tiles (default hidden), Heatmap Signal (default hidden)
2. WHEN a layer toggle is changed, THE Map_Page SHALL immediately show or hide the corresponding markers and polylines on the map without reloading the page
3. THE Map_Page SHALL provide filter controls for: ONT status (online/offline/weak signal), billing status (aktif/isolir/pending), package, area, and ODP
4. WHEN filters are applied, THE Map_Page SHALL display a count of visible nodes out of total nodes (e.g., "Menampilkan: 87 dari 847 pelanggan")
5. THE Map_Page SHALL provide a Reset Filter button that clears all active filters and restores default layer visibility

### Requirement 17: Frontend — Search di Peta

**User Story:** As an ISP admin, I want to search for customers, ODP, OLT by name, ID, address, or serial number directly on the map, so that I can quickly navigate to any network entity.

#### Acceptance Criteria

1. THE Map_Page SHALL provide a search input (🔍) in the toolbar that accepts text queries with autocomplete suggestions from the Map_API search endpoint
2. WHEN the user types at least 2 characters, THE Map_Page SHALL display autocomplete results showing: type icon, name, identifier, and brief address
3. WHEN the user selects a search result, THE Map_Page SHALL center the map on the selected node's coordinates, zoom to an appropriate level, and highlight the node marker
4. THE Map_Page SHALL debounce search input by 300ms to avoid excessive API calls

### Requirement 18: Frontend — Topology View

**User Story:** As an ISP admin, I want a tree hierarchy view showing OLT → PON Port → ODP → ONT, so that I can understand the logical network structure without needing the geographic map.

#### Acceptance Criteria

1. THE Map_Page SHALL provide a toggle between geographic map view and Topology_View
2. THE Topology_View SHALL display a collapsible tree hierarchy: OLT → PON Port → ODP → ONT, with status colors matching the map markers
3. WHEN a user clicks a node in the Topology_View, THE Map_Page SHALL navigate to the detail page of that entity (OLT detail, ODP detail, or customer detail)
4. THE Topology_View SHALL support expand/collapse per level, allowing the user to drill down from OLT to individual ONT
5. THE Topology_View SHALL display summary counts at each level: total ONT per PON port, online/offline counts per ODP

### Requirement 19: Frontend — Heatmap Signal Quality

**User Story:** As an ISP admin, I want a heatmap overlay on the map showing signal quality per area, so that I can identify areas with poor network quality for proactive maintenance.

#### Acceptance Criteria

1. WHEN the Heatmap Signal layer is enabled, THE Map_Page SHALL render a heatmap overlay using ONT signal strength data, with colors ranging from green (good signal, -8 to -20 dBm) through yellow (-20 to -25 dBm) and orange (-25 to -27 dBm) to red (critical, below -27 dBm)
2. THE Map_Page SHALL calculate heatmap intensity based on the average signal strength of ONT nodes per geographic cluster
3. THE Map_Page SHALL display a legend explaining the heatmap color scale and dBm ranges

### Requirement 20: Frontend — Edit Lokasi Node (Drag & Drop)

**User Story:** As an ISP admin, I want to drag and drop node markers on the map to update their GPS coordinates, so that I can correct inaccurate positions without editing forms.

#### Acceptance Criteria

1. WHEN the user clicks "Edit Lokasi" on a node's Detail_Panel, THE Map_Page SHALL make the node marker draggable
2. WHEN the user drags a marker to a new position and confirms, THE Map_Page SHALL send a PUT request to the Map_API to update the node's latitude and longitude
3. WHEN a marker is dragged to a new position, THE Map_Page SHALL trigger reverse geocoding for the new coordinates and display the updated address
4. THE Map_Page SHALL provide a Cancel button to revert the marker to its original position without saving

### Requirement 21: Frontend — Navigasi dan Lokasi Saya

**User Story:** As a field technician, I want to see my current GPS location on the map and navigate to any node using Google Maps or Waze, so that I can efficiently find network equipment in the field.

#### Acceptance Criteria

1. THE Map_Page SHALL provide a "Lokasi Saya" button (📍) that requests browser GPS permission and displays the technician's current position as a pulsing blue marker on the map
2. WHEN the user's location is available, THE Map_Page SHALL display the distance from the user's position to each node in the Detail_Panel
3. WHEN the user clicks "Navigasi" on a node's Detail_Panel, THE Map_Page SHALL offer options to open Google Maps or Waze with the node's coordinates as the destination (via deep link)
4. IF the browser denies GPS permission, THEN THE Map_Page SHALL display a message explaining that location access is needed for navigation features and continue functioning without location-based features

### Requirement 22: Frontend — Offline Mode

**User Story:** As a field technician, I want to download a map area with node data for offline use, so that I can view and edit the map in areas without internet connectivity.

#### Acceptance Criteria

1. THE Map_Page SHALL provide a "Download Peta Offline" feature that allows the user to select a rectangular area on the map and download: map tiles (OpenStreetMap), node data (OLT, ODP, ONT), cable routes, and photo thumbnails
2. THE Map_Page SHALL use Service Worker and IndexedDB to cache downloaded map data in the browser
3. WHILE the browser is offline, THE Map_Page SHALL display cached map tiles and node data with an "⚡ Mode Offline" indicator
4. WHILE the browser is offline, THE Map_Page SHALL allow the user to add, edit, and move nodes, storing changes locally in IndexedDB
5. WHEN the browser returns online, THE Map_Page SHALL automatically sync locally stored changes to the Map_API and display a "✅ Tersinkronisasi" indicator upon completion
6. IF a sync conflict occurs (same node modified both offline and online), THEN THE Map_Page SHALL use last-write-wins strategy by default and log the conflict for admin review
7. THE Map_Page SHALL limit offline cache to a maximum of 100 MB per area and expire cached data after 7 days

### Requirement 23: Frontend — Export PNG dan PDF dari Peta

**User Story:** As an ISP admin, I want to export the current map view as a PNG screenshot or a printable PDF with legend, so that I can include the map in physical documentation or presentations.

#### Acceptance Criteria

1. THE Map_Page SHALL provide export options for PNG (screenshot of current map view) and PDF (printable map with legend, in A3 or A4 format)
2. WHEN exporting to PNG, THE Map_Page SHALL capture the current map viewport including all visible markers, lines, and labels as a raster image
3. WHEN exporting to PDF, THE Map_Page SHALL generate a document containing: the map image, a legend explaining node icons and colors, a title with tenant name and export date, and optionally a list of visible nodes

### Requirement 24: Backend — Graceful Degradation

**User Story:** As an ISP admin whose OLT or MikroTik module is not yet active, I want the map to still work for plotting customer locations and planning cable routes, so that I can start using the mapping feature before fully integrating network equipment.

#### Acceptance Criteria

1. WHILE the OLT module is not active for a tenant, THE Map_Page SHALL display ONT markers based on customer GPS coordinates without signal data, and hide OLT and ODP markers
2. WHILE the MikroTik module is not active for a tenant, THE Map_Page SHALL hide MikroTik router markers and omit PPPoE status from node details
3. THE Map_Page SHALL always make Drawing_Tool, cable route management, search, export, import, and reverse geocoding features available regardless of which modules are active
4. WHEN the OLT module becomes active for a tenant, THE Map_Page SHALL automatically start displaying OLT markers, ODP markers, and ONT signal data without requiring manual configuration

### Requirement 25: Backend — Cable Route Distance Calculation

**User Story:** As a platform engineer, I want cable route distances to be automatically calculated from polyline coordinates using the Haversine formula, so that distance data is accurate and consistent.

#### Acceptance Criteria

1. WHEN a cable route is created or updated with a coordinates array, THE Map_API SHALL calculate the total distance by summing the Haversine distance between each consecutive pair of coordinates
2. THE Map_API SHALL store the calculated distance in the distance_meters column of the cable_routes table
3. FOR ALL valid coordinate arrays with 2 or more points, THE Map_API SHALL produce a distance_meters value greater than 0
4. FOR ALL valid coordinate arrays, calculating distance from coordinates [A, B, C] SHALL equal the sum of distance(A,B) + distance(B,C) (segment additivity property)
