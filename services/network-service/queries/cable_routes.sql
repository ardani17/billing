-- Kueri SQL untuk operasi CRUD tabel cable_routes.
-- Digunakan oleh sqlc untuk menghasilkan kode Go yang type-safe.
-- Tabel cable_routes dilindungi RLS, kueri hanya mengembalikan baris milik tenant aktif.

-- name: CreateCableRoute :one
INSERT INTO cable_routes (
    tenant_id, from_node_id, to_node_id, route_type,
    coordinates, distance_meters, core_count, description
) VALUES (
    $1, $2, $3, $4,
    $5, $6, $7, $8
)
RETURNING id, tenant_id, from_node_id, to_node_id, route_type,
    coordinates, distance_meters, core_count, description,
    deleted_at, created_at, updated_at;

-- name: GetCableRouteByID :one
SELECT id, tenant_id, from_node_id, to_node_id, route_type,
    coordinates, distance_meters, core_count, description,
    deleted_at, created_at, updated_at
FROM cable_routes
WHERE id = $1 AND deleted_at IS NULL;

-- name: UpdateCableRoute :one
UPDATE cable_routes SET
    route_type = $2,
    coordinates = $3,
    distance_meters = $4,
    core_count = $5,
    description = $6,
    updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL
RETURNING id, tenant_id, from_node_id, to_node_id, route_type,
    coordinates, distance_meters, core_count, description,
    deleted_at, created_at, updated_at;

-- name: SoftDeleteCableRoute :exec
UPDATE cable_routes SET deleted_at = NOW(), updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL;

-- name: ListCableRoutesByBounds :many
SELECT cr.id, cr.tenant_id, cr.from_node_id, cr.to_node_id, cr.route_type,
    cr.coordinates, cr.distance_meters, cr.core_count, cr.description,
    cr.deleted_at, cr.created_at, cr.updated_at
FROM cable_routes cr
JOIN map_nodes fn ON cr.from_node_id = fn.id
JOIN map_nodes tn ON cr.to_node_id = tn.id
WHERE cr.deleted_at IS NULL
  AND (
    (fn.latitude BETWEEN $1 AND $2 AND fn.longitude BETWEEN $3 AND $4)
    OR
    (tn.latitude BETWEEN $1 AND $2 AND tn.longitude BETWEEN $3 AND $4)
  )
  AND (sqlc.narg('route_type')::varchar IS NULL OR cr.route_type = sqlc.narg('route_type')::varchar)
ORDER BY cr.created_at DESC;

-- name: ListCableRoutesByNode :many
SELECT id, tenant_id, from_node_id, to_node_id, route_type,
    coordinates, distance_meters, core_count, description,
    deleted_at, created_at, updated_at
FROM cable_routes
WHERE (from_node_id = $1 OR to_node_id = $1) AND deleted_at IS NULL
ORDER BY created_at DESC;
