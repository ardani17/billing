-- Query SQL untuk operasi CRUD tabel expenses.
-- Digunakan oleh sqlc untuk menghasilkan kode Go yang type-safe.
-- Tabel expenses dilindungi RLS, query hanya mengembalikan baris milik tenant aktif.
-- Semua query menyertakan WHERE deleted_at IS NULL untuk mengecualikan soft-deleted.

-- name: CreateExpense :one
-- Membuat pengeluaran baru dan mengembalikan semua kolom.
INSERT INTO expenses (
    tenant_id, category_id, amount, description,
    expense_date, is_recurring, recurring_day, created_by_id
) VALUES (
    $1, $2, $3, $4,
    $5, $6, $7, $8
)
RETURNING *;

-- name: GetExpenseByID :one
-- Mengambil pengeluaran berdasarkan ID beserta nama kategori melalui JOIN.
SELECT e.*,
    ec.name AS category_name
FROM expenses e
JOIN expense_categories ec ON ec.id = e.category_id
WHERE e.id = $1 AND e.deleted_at IS NULL;

-- name: UpdateExpense :one
-- Memperbarui data pengeluaran (kategori, jumlah, deskripsi, tanggal, recurring).
UPDATE expenses SET
    category_id = $2,
    amount = $3,
    description = $4,
    expense_date = $5,
    is_recurring = $6,
    recurring_day = $7,
    updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: SoftDeleteExpense :exec
-- Menghapus pengeluaran secara soft delete (set deleted_at).
UPDATE expenses SET
    deleted_at = NOW(),
    updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL;

-- name: ListExpenses :many
-- Mengambil daftar pengeluaran untuk tenant dengan filter periode dan kategori opsional.
-- Menggunakan sqlc.narg untuk category_id opsional.
-- Diurutkan berdasarkan expense_date descending (terbaru di atas).
SELECT e.*,
    ec.name AS category_name
FROM expenses e
JOIN expense_categories ec ON ec.id = e.category_id
WHERE e.tenant_id = $1
  AND e.expense_date >= $2
  AND e.expense_date <= $3
  AND (sqlc.narg('category_id')::uuid IS NULL OR e.category_id = sqlc.narg('category_id')::uuid)
  AND e.deleted_at IS NULL
ORDER BY e.expense_date DESC;

-- name: ListRecurringExpenses :many
-- Mengambil semua pengeluaran berulang yang aktif (untuk auto-create bulanan oleh worker).
SELECT * FROM expenses
WHERE is_recurring = true AND deleted_at IS NULL;

-- name: SumExpensesByCategory :many
-- Menghitung total pengeluaran per kategori untuk laporan laba rugi.
-- Mengembalikan nama kategori dan total amount, diurutkan berdasarkan total terbesar.
SELECT ec.name AS label,
    COALESCE(SUM(e.amount), 0)::bigint AS amount
FROM expenses e
JOIN expense_categories ec ON ec.id = e.category_id
WHERE e.tenant_id = $1
  AND e.expense_date >= $2
  AND e.expense_date <= $3
  AND e.deleted_at IS NULL
GROUP BY ec.name
ORDER BY amount DESC;
