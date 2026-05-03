-- Query SQL untuk operasi CRUD tabel customer_recurring_items.
-- Digunakan oleh sqlc untuk menghasilkan kode Go yang type-safe.
-- Tabel customer_recurring_items dilindungi RLS, query hanya mengembalikan baris milik tenant aktif.

-- name: CreateRecurringItem :one
-- Membuat recurring item baru dan mengembalikan semua kolom.
INSERT INTO customer_recurring_items (
    tenant_id, customer_id, description, amount,
    is_active, start_date, end_date
) VALUES (
    $1, $2, $3, $4,
    $5, $6, $7
)
RETURNING *;

-- name: GetRecurringItemByID :one
-- Mengambil recurring item berdasarkan ID.
SELECT *
FROM customer_recurring_items
WHERE id = $1;

-- name: UpdateRecurringItem :one
-- Memperbarui recurring item dan mengembalikan item yang diperbarui.
UPDATE customer_recurring_items SET
    description = $2,
    amount = $3,
    end_date = $4,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeactivateRecurringItem :exec
-- Menonaktifkan recurring item (set is_active = false).
UPDATE customer_recurring_items SET
    is_active = false,
    updated_at = NOW()
WHERE id = $1;

-- name: ListRecurringItemsByCustomer :many
-- Mengambil semua recurring item untuk customer tertentu.
SELECT *
FROM customer_recurring_items
WHERE customer_id = $1
ORDER BY created_at ASC;

-- name: ListActiveRecurringItemsByCustomer :many
-- Mengambil recurring item aktif untuk customer pada tanggal periode tertentu.
-- Item aktif: is_active = true, start_date <= period_date, dan (end_date IS NULL atau end_date > period_date).
SELECT *
FROM customer_recurring_items
WHERE customer_id = $1
  AND is_active = true
  AND start_date <= $2
  AND (end_date IS NULL OR end_date > $2)
ORDER BY created_at ASC;
