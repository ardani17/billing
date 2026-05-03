-- Query SQL untuk operasi CRUD tabel customers (schema lengkap 24 kolom).
-- Digunakan oleh sqlc untuk menghasilkan kode Go yang type-safe.
-- Tabel customers dilindungi RLS, query hanya mengembalikan baris milik tenant aktif.
-- Semua query menyertakan WHERE deleted_at IS NULL untuk mengecualikan soft-deleted.

-- name: CreateCustomer :one
INSERT INTO customers (
    tenant_id, customer_id_seq, name, phone, email, address,
    area_id, latitude, longitude, package_id, activation_date,
    due_date, connection_method, pppoe_username, pppoe_password,
    mac_address, router_id, odp_port, credit_balance, notes, status
) VALUES (
    $1, $2, $3, $4, $5, $6,
    $7, $8, $9, $10, $11,
    $12, $13, $14, $15,
    $16, $17, $18, $19, $20, $21
)
RETURNING id, tenant_id, customer_id_seq, name, phone, email, address,
    area_id, latitude, longitude, package_id, activation_date,
    due_date, connection_method, pppoe_username, pppoe_password,
    mac_address, router_id, odp_port, credit_balance, notes, status,
    deleted_at, created_at, updated_at;

-- name: GetCustomerByID :one
SELECT id, tenant_id, customer_id_seq, name, phone, email, address,
    area_id, latitude, longitude, package_id, activation_date,
    due_date, connection_method, pppoe_username, pppoe_password,
    mac_address, router_id, odp_port, credit_balance, notes, status,
    deleted_at, created_at, updated_at
FROM customers
WHERE id = $1 AND deleted_at IS NULL;

-- name: UpdateCustomer :one
UPDATE customers SET
    name = $2,
    phone = $3,
    email = $4,
    address = $5,
    area_id = $6,
    latitude = $7,
    longitude = $8,
    package_id = $9,
    activation_date = $10,
    due_date = $11,
    connection_method = $12,
    pppoe_username = $13,
    pppoe_password = $14,
    mac_address = $15,
    router_id = $16,
    odp_port = $17,
    credit_balance = $18,
    notes = $19,
    updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL
RETURNING id, tenant_id, customer_id_seq, name, phone, email, address,
    area_id, latitude, longitude, package_id, activation_date,
    due_date, connection_method, pppoe_username, pppoe_password,
    mac_address, router_id, odp_port, credit_balance, notes, status,
    deleted_at, created_at, updated_at;

-- name: SoftDeleteCustomer :exec
UPDATE customers SET deleted_at = NOW(), updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL;

-- name: UpdateCustomerStatus :one
UPDATE customers SET status = $2, updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL
RETURNING id, tenant_id, customer_id_seq, name, phone, email, address,
    area_id, latitude, longitude, package_id, activation_date,
    due_date, connection_method, pppoe_username, pppoe_password,
    mac_address, router_id, odp_port, credit_balance, notes, status,
    deleted_at, created_at, updated_at;

-- name: UpdateCustomerPackage :one
UPDATE customers SET package_id = $2, updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL
RETURNING id, tenant_id, customer_id_seq, name, phone, email, address,
    area_id, latitude, longitude, package_id, activation_date,
    due_date, connection_method, pppoe_username, pppoe_password,
    mac_address, router_id, odp_port, credit_balance, notes, status,
    deleted_at, created_at, updated_at;

-- name: GetMaxCustomerSeq :one
SELECT COALESCE(MAX(
    CAST(SUBSTRING(customer_id_seq FROM 'PLG-(\d+)') AS INTEGER)
), 0)::integer AS max_seq
FROM customers
WHERE tenant_id = $1;

-- name: PhoneExists :one
SELECT EXISTS(
    SELECT 1 FROM customers
    WHERE tenant_id = $1 AND phone = $2 AND id != $3 AND deleted_at IS NULL
) AS exists;

-- name: CountCustomersByStatusAktif :one
SELECT COUNT(*) FROM customers WHERE status = 'aktif' AND deleted_at IS NULL;

-- name: CountCustomersByStatusPending :one
SELECT COUNT(*) FROM customers WHERE status = 'pending' AND deleted_at IS NULL;

-- name: CountCustomersByStatusIsolir :one
SELECT COUNT(*) FROM customers WHERE status = 'isolir' AND deleted_at IS NULL;

-- name: CountCustomersByStatusSuspend :one
SELECT COUNT(*) FROM customers WHERE status = 'suspend' AND deleted_at IS NULL;

-- name: CountCustomersByStatusBerhenti :one
SELECT COUNT(*) FROM customers WHERE status = 'berhenti' AND deleted_at IS NULL;

-- name: SearchCustomersForPayment :many
-- Mencari pelanggan berdasarkan nama, customer_id_seq, atau telepon untuk quick payment.
-- Mengembalikan maksimal 10 hasil, hanya status aktif atau isolir.
SELECT id, tenant_id, customer_id_seq, name, phone, email, address,
    area_id, latitude, longitude, package_id, activation_date,
    due_date, connection_method, pppoe_username, pppoe_password,
    mac_address, router_id, odp_port, credit_balance, notes, status,
    deleted_at, created_at, updated_at
FROM customers
WHERE tenant_id = $1
  AND (name ILIKE $2 OR customer_id_seq ILIKE $2 OR phone ILIKE $2)
  AND status IN ('aktif', 'isolir')
  AND deleted_at IS NULL
LIMIT 10;
