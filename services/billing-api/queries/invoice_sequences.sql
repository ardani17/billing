-- Kueri SQL untuk operasi pada tabel invoice_sequences.
-- Digunakan oleh sqlc untuk menghasilkan kode Go yang type-safe.
-- Tabel invoice_sequences dilindungi RLS, kueri hanya mengembalikan baris milik tenant aktif.
-- Sequence digunakan untuk auto-increment nomor invoice per tenant per bulan.

-- name: NextInvoiceSequence :one
-- Mengambil dan increment sequence secara atomik menggunakan INSERT ON CONFLICT.
-- Jika row belum ada untuk tenant/year/month, buat baru dengan last_seq = 1.
-- Jika sudah ada, increment last_seq dan kembalikan nilai baru.
INSERT INTO invoice_sequences (
    tenant_id, year, month, last_seq
) VALUES (
    $1, $2, $3, 1
)
ON CONFLICT (tenant_id, year, month) DO UPDATE SET
    last_seq = invoice_sequences.last_seq + 1,
    updated_at = NOW()
RETURNING last_seq;
