// Fungsi client API untuk semua network-map endpoints.
// Permintaan browser melewati proxy Next.js agar backend menerima
// header JWT/tenant dev dari sisi server.

const API_BASE = process.env.NEXT_PUBLIC_API_URL ?? '';
const MAP_API = `${API_BASE}/api/network-map`;

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export interface MapNodeWithRef {
  id: string;
  tenant_id: string;
  node_type: 'olt' | 'odp' | 'ont';
  reference_id: string;
  latitude: number;
  longitude: number;
  custom_fields: Record<string, unknown> | null;
  deleted_at: string | null;
  created_at: string;
  updated_at: string;
  // Joined reference data
  name?: string;
  status?: string;
  signal_dbm?: number;
  billing_status?: string;
  package_name?: string;
  customer_name?: string;
  customer_id?: string;
  serial_number?: string;
  odp_name?: string;
  brand_model?: string;
  ont_count?: number;
  splitter_type?: string;
  capacity?: string;
  port_usage?: string;
}

export interface CableRoute {
  id: string;
  tenant_id: string;
  from_node_id: string;
  to_node_id: string;
  route_type: 'backbone' | 'drop';
  coordinates: [number, number][];
  distance_meters: number;
  core_count: number | null;
  description: string | null;
  deleted_at: string | null;
  created_at: string;
  updated_at: string;
  // Joined node data untuk rendering
  from_node_status?: string;
  to_node_status?: string;
}

export interface MapNodeDetail extends MapNodeWithRef {
  photos: NodePhoto[];
  history: MapChangeHistory[];
}

export interface NodePhoto {
  id: string;
  map_node_id: string;
  file_path: string;
  file_size_bytes: number;
  caption: string | null;
  uploaded_by: string;
  created_at: string;
}

export interface MapChangeHistory {
  id: string;
  map_node_id: string;
  action: string;
  old_value: Record<string, unknown> | null;
  new_value: Record<string, unknown> | null;
  performed_by: string;
  created_at: string;
}

export interface SearchResult {
  type: 'olt' | 'odp' | 'ont';
  name: string;
  identifier: string;
  latitude: number;
  longitude: number;
  description: string;
}

export interface BoundingBox {
  minLat: number;
  minLng: number;
  maxLat: number;
  maxLng: number;
}

export interface NodeFilters {
  node_type?: string;
  status?: string;
  billing_status?: string;
  package_id?: string;
  area_id?: string;
  odp_id?: string;
}

export interface LabelSettings {
  id: string;
  tenant_id: string;
  olt_labels: string[];
  odp_labels: string[];
  ont_labels: string[];
  min_zoom_level: number;
}

export interface LossCalculatorInput {
  distance_olt_to_odp_km: number;
  distance_odp_to_ont_km: number;
  splitter_count: number;
  splitter_type: string;
  connector_count: number;
  splice_count: number;
  sfp_tx_power_dbm: number;
  ont_sensitivity_dbm: number;
}

export interface LossCalculatorResult {
  total_loss_db: number;
  budget_available_db: number;
  remaining_margin_db: number;
  estimated_signal_at_ont_dbm: number;
  feasible: boolean;
  fiber_loss_db: number;
  splitter_loss_db: number;
  connector_loss_db: number;
  splice_loss_db: number;
  safety_margin_db: number;
}

export interface ShareLink {
  id: string;
  token: string;
  visible_layers: string[];
  expires_at: string | null;
  access_count: number;
  created_by: string;
  created_at: string;
}

export interface GeocodingResult {
  address: string;
  raw: Record<string, unknown> | null;
  error?: string;
}

export interface ExportRequest {
  format: 'kml' | 'kmz' | 'geojson' | 'csv';
  layers: string[];
  options?: {
    include_icons?: boolean;
    include_descriptions?: boolean;
    include_photos?: boolean;
  };
}

export interface ExportStatus {
  job_id: string;
  status: 'processing' | 'completed' | 'failed';
  download_url?: string;
  error?: string;
}

export interface ImportPreview {
  import_id: string;
  points: number;
  lines: number;
  polygons: number;
  items: Array<{ name: string; type: string; coordinates: [number, number] }>;
}

export interface ImportMapping {
  type_mapping: Record<string, string>;
  auto_match: boolean;
}

export interface ImportSummary {
  total: number;
  success: number;
  skipped: number;
  errors: Array<{ item: string; error: string }>;
}

// ---------------------------------------------------------------------------
// Fungsi bantu
// ---------------------------------------------------------------------------

async function apiFetch<T>(url: string, init?: RequestInit): Promise<T> {
  const res = await fetch(url, {
    ...init,
    headers: {
      'Content-Type': 'application/json',
      ...init?.headers,
    },
  });
  if (!res.ok) {
    const body = await res.json().catch(() => ({}));
    throw new Error(
      apiErrorMessage(body) ?? `API error ${res.status}`,
    );
  }
  if (res.status === 204) return undefined as T;

  const body = await res.json();
  return unwrapApiResponse<T>(body);
}

function unwrapApiResponse<T>(body: unknown): T {
  if (
    body &&
    typeof body === 'object' &&
    'success' in body &&
    'data' in body
  ) {
    return (body as { data: T }).data;
  }

  return body as T;
}

function apiErrorMessage(body: unknown): string | null {
  if (!body || typeof body !== 'object' || !('error' in body)) return null;
  const error = (body as { error?: unknown }).error;
  if (typeof error === 'string') return error;
  if (error && typeof error === 'object' && 'message' in error) {
    const message = (error as { message?: unknown }).message;
    return typeof message === 'string' ? message : null;
  }
  return null;
}

// ---------------------------------------------------------------------------
// Nodes
// ---------------------------------------------------------------------------

export async function fetchNodes(
  bounds: BoundingBox,
  filters?: NodeFilters,
): Promise<MapNodeWithRef[]> {
  const params = new URLSearchParams({
    min_lat: String(bounds.minLat),
    min_lng: String(bounds.minLng),
    max_lat: String(bounds.maxLat),
    max_lng: String(bounds.maxLng),
  });
  if (filters?.node_type) params.set('node_type', filters.node_type);
  if (filters?.status) params.set('status', filters.status);
  if (filters?.billing_status)
    params.set('billing_status', filters.billing_status);
  if (filters?.package_id) params.set('package_id', filters.package_id);
  if (filters?.area_id) params.set('area_id', filters.area_id);
  if (filters?.odp_id) params.set('odp_id', filters.odp_id);

  return apiFetch<MapNodeWithRef[]>(`${MAP_API}/nodes?${params.toString()}`);
}

export async function fetchNodeDetail(id: string): Promise<MapNodeDetail> {
  return apiFetch<MapNodeDetail>(`${MAP_API}/nodes/${id}`);
}

export async function createNode(
  data: {
    node_type: string;
    reference_id: string;
    latitude: number;
    longitude: number;
    custom_fields?: Record<string, unknown>;
  },
): Promise<MapNodeWithRef> {
  return apiFetch<MapNodeWithRef>(`${MAP_API}/nodes`, {
    method: 'POST',
    body: JSON.stringify(data),
  });
}

export async function updateNode(
  id: string,
  data: {
    latitude?: number;
    longitude?: number;
    custom_fields?: Record<string, unknown>;
  },
): Promise<MapNodeWithRef> {
  return apiFetch<MapNodeWithRef>(`${MAP_API}/nodes/${id}`, {
    method: 'PUT',
    body: JSON.stringify(data),
  });
}

export async function deleteNode(id: string): Promise<void> {
  await apiFetch<void>(`${MAP_API}/nodes/${id}`, { method: 'DELETE' });
}

// ---------------------------------------------------------------------------
// Cables
// ---------------------------------------------------------------------------

export async function fetchCables(
  bounds: BoundingBox,
  filters?: { route_type?: string; from_node_id?: string; to_node_id?: string },
): Promise<CableRoute[]> {
  const params = new URLSearchParams({
    min_lat: String(bounds.minLat),
    min_lng: String(bounds.minLng),
    max_lat: String(bounds.maxLat),
    max_lng: String(bounds.maxLng),
  });
  if (filters?.route_type) params.set('route_type', filters.route_type);
  if (filters?.from_node_id) params.set('from_node_id', filters.from_node_id);
  if (filters?.to_node_id) params.set('to_node_id', filters.to_node_id);

  return apiFetch<CableRoute[]>(`${MAP_API}/cables?${params.toString()}`);
}

export async function createCable(
  data: {
    from_node_id: string;
    to_node_id: string;
    route_type: string;
    coordinates: [number, number][];
    core_count?: number;
    description?: string;
  },
): Promise<CableRoute> {
  return apiFetch<CableRoute>(`${MAP_API}/cables`, {
    method: 'POST',
    body: JSON.stringify(data),
  });
}

export async function updateCable(
  id: string,
  data: {
    coordinates?: [number, number][];
    core_count?: number;
    description?: string;
  },
): Promise<CableRoute> {
  return apiFetch<CableRoute>(`${MAP_API}/cables/${id}`, {
    method: 'PUT',
    body: JSON.stringify(data),
  });
}

export async function deleteCable(id: string): Promise<void> {
  await apiFetch<void>(`${MAP_API}/cables/${id}`, { method: 'DELETE' });
}

// ---------------------------------------------------------------------------
// Photos
// ---------------------------------------------------------------------------

export async function fetchPhotos(nodeId: string): Promise<NodePhoto[]> {
  return apiFetch<NodePhoto[]>(`${MAP_API}/nodes/${nodeId}/photos`);
}

export async function uploadPhoto(
  nodeId: string,
  file: File,
  caption?: string,
): Promise<NodePhoto> {
  const formData = new FormData();
  formData.append('file', file);
  if (caption) formData.append('caption', caption);

  const res = await fetch(`${MAP_API}/nodes/${nodeId}/photos`, {
    method: 'POST',
    body: formData,
  });
  if (!res.ok) {
    const body = await res.json().catch(() => ({}));
    throw new Error(
      apiErrorMessage(body) ?? `Upload error ${res.status}`,
    );
  }
  const body = await res.json();
  return unwrapApiResponse<NodePhoto>(body);
}

export async function deletePhoto(
  nodeId: string,
  photoId: string,
): Promise<void> {
  await apiFetch<void>(`${MAP_API}/nodes/${nodeId}/photos/${photoId}`, {
    method: 'DELETE',
  });
}

// ---------------------------------------------------------------------------
// Pencarian
// ---------------------------------------------------------------------------

export async function searchNodes(query: string): Promise<SearchResult[]> {
  const params = new URLSearchParams({ q: query });
  return apiFetch<SearchResult[]>(`${MAP_API}/search?${params.toString()}`);
}

// ---------------------------------------------------------------------------
// Export
// ---------------------------------------------------------------------------

export async function exportMap(
  data: ExportRequest,
): Promise<Blob | ExportStatus> {
  const res = await fetch(`${MAP_API}/export`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  });

  if (res.status === 202) {
    const body = await res.json();
    return unwrapApiResponse<ExportStatus>(body);
  }
  if (!res.ok) {
    const body = await res.json().catch(() => ({}));
    throw new Error(
      apiErrorMessage(body) ?? `Export error ${res.status}`,
    );
  }
  return res.blob();
}

export async function getExportStatus(jobId: string): Promise<ExportStatus> {
  return apiFetch<ExportStatus>(`${MAP_API}/export/status/${jobId}`);
}

// ---------------------------------------------------------------------------
// Import
// ---------------------------------------------------------------------------

export async function previewImport(file: File): Promise<ImportPreview> {
  const formData = new FormData();
  formData.append('file', file);

  const res = await fetch(`${MAP_API}/import`, {
    method: 'POST',
    body: formData,
  });
  if (!res.ok) {
    const body = await res.json().catch(() => ({}));
    throw new Error(
      apiErrorMessage(body) ?? `Import error ${res.status}`,
    );
  }
  const body = await res.json();
  return unwrapApiResponse<ImportPreview>(body);
}

export async function executeImport(
  importId: string,
  mapping: ImportMapping,
): Promise<ImportSummary> {
  return apiFetch<ImportSummary>(`${MAP_API}/import/execute`, {
    method: 'POST',
    body: JSON.stringify({ import_id: importId, ...mapping }),
  });
}

export async function getImportStatus(
  jobId: string,
): Promise<{ status: string; progress?: number }> {
  return apiFetch<{ status: string; progress?: number }>(
    `${MAP_API}/import/status/${jobId}`,
  );
}

// ---------------------------------------------------------------------------
// Geocoding
// ---------------------------------------------------------------------------

export async function reverseGeocode(
  lat: number,
  lng: number,
): Promise<GeocodingResult> {
  const params = new URLSearchParams({
    lat: String(lat),
    lng: String(lng),
  });
  return apiFetch<GeocodingResult>(
    `${MAP_API}/geocode/reverse?${params.toString()}`,
  );
}

// ---------------------------------------------------------------------------
// Share
// ---------------------------------------------------------------------------

export async function createShareLink(data: {
  visible_layers: string[];
  expiry_days?: number | null;
  password?: string | null;
}): Promise<ShareLink & { url: string; embed_code: string }> {
  return apiFetch<ShareLink & { url: string; embed_code: string }>(
    `${MAP_API}/share`,
    { method: 'POST', body: JSON.stringify(data) },
  );
}

export async function listShareLinks(): Promise<ShareLink[]> {
  return apiFetch<ShareLink[]>(`${MAP_API}/share`);
}

export async function getSharedMap(
  token: string,
  password?: string,
): Promise<{
  nodes: MapNodeWithRef[];
  cables: CableRoute[];
  visible_layers: string[];
}> {
  const params = new URLSearchParams();
  if (password) params.set('password', password);
  const qs = params.toString();
  return apiFetch<{
    nodes: MapNodeWithRef[];
    cables: CableRoute[];
    visible_layers: string[];
  }>(`${MAP_API}/share/${token}${qs ? `?${qs}` : ''}`);
}

export async function deleteShareLink(token: string): Promise<void> {
  await apiFetch<void>(`${MAP_API}/share/${token}`, { method: 'DELETE' });
}

// ---------------------------------------------------------------------------
// Loss Calculator
// ---------------------------------------------------------------------------

export async function calculateLoss(
  input: LossCalculatorInput,
): Promise<LossCalculatorResult> {
  return apiFetch<LossCalculatorResult>(`${MAP_API}/loss-calculator`, {
    method: 'POST',
    body: JSON.stringify(input),
  });
}

// ---------------------------------------------------------------------------
// Label Settings
// ---------------------------------------------------------------------------

export async function fetchLabelSettings(): Promise<LabelSettings> {
  return apiFetch<LabelSettings>(`${MAP_API}/settings/labels`);
}

export async function updateLabelSettings(
  data: Partial<Omit<LabelSettings, 'id' | 'tenant_id'>>,
): Promise<LabelSettings> {
  return apiFetch<LabelSettings>(`${MAP_API}/settings/labels`, {
    method: 'PUT',
    body: JSON.stringify(data),
  });
}

// ---------------------------------------------------------------------------
// Trash
// ---------------------------------------------------------------------------

export async function fetchTrashed(): Promise<MapNodeWithRef[]> {
  return apiFetch<MapNodeWithRef[]>(`${MAP_API}/trash`);
}

export async function restoreNode(id: string): Promise<void> {
  await apiFetch<void>(`${MAP_API}/trash/${id}/restore`, { method: 'POST' });
}
