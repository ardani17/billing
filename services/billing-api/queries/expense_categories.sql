-- Query SQL untuk operasi CRUD tabel expense_categories.
-- Digunakan oleh sqlc untuk menghasilkan kode Go yang type-safe.
-- Tabel expense_categories dilindungi RLS, query hanya mengembalikan baris milik tenant aktif.

-- name: CreateExpenseCategory :one
-- Membuat kategori pengeluaran baru dan mengembalikan semua kolom.
INSERT INTO expense_categories (tenant_id, name, is_default)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetExpenseCategoryByID :one
-- Mengambil kategori pengeluaran berdasarkan ID.
SELECT * FROM expense_categories
WHERE id = $1 AND deleted_at IS NULL;

-- name: UpdateExpenseCategory :one
-- Memperbarui nama kategori pengeluaran.
UPDATE expense_categories SET
    name = $2,
    updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: SoftDeleteExpenseCategory :exec
-- Menghapus kategori secara soft delete (set deleted_at).
UPDATE expense_categories SET
    deleted_at = NOW(),
    updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL;

-- name: ListExpenseCategories :many
-- Mengambil semua kategori pengeluaran aktif untuk tenant, diurutkan berdasarkan nama.
SELECT * FROM expense_categories
WHERE tenant_id = $1 AND deleted_at IS NULL
ORDER BY name ASC;

-- name: ExpenseCategoryNameExists :one
-- Mengecek apakah nama kategori sudah ada di tenant (exclude ID tertentu, hanya yang aktif).
SELECT EXISTS(
    SELECT 1 FROM expense_categories
    WHERE tenant_id = $1 AND name = $2 AND id != $3 AND deleted_at IS NULL
) AS exists;

-- name: ExpenseCategoryExpenseCount :one
-- Menghitung jumlah pengeluaran aktif dalam kategori.
SELECT COUNT(*)::integer AS count
FROM expenses
WHERE category_id = $1 AND deleted_at IS NULL;

-- name: CreateDefaultExpenseCategories :exec
-- Membuat kategori default untuk tenant baru (7 kategori).
INSERT INTO expense_categories (tenant_id, name, is_default) VALUES
    ($1, 'Bandwidth/Upstream', true),
    ($1, 'Gaji Karyawan', true),
    ($1, 'Sewa Tiang/Infrastruktur', true),
    ($1, 'Listrik & Operasional', true),
    ($1, 'Perangkat', true),
    ($1, 'Notifikasi', true),
    ($1, 'Lainnya', true);
