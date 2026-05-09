-- File helper schema: mendefinisikan tabel shared dari billing-api agar sqlc bisa melakukan inferensi tipe.
-- File ini TIDAK dijalankan sebagai migrasi - hanya digunakan oleh sqlc untuk mengetahui struktur tabel.
-- Tabel customers dan tenants sudah dibuat oleh migrasi billing-api di database yang sama.

CREATE TABLE IF NOT EXISTS tenants (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name       VARCHAR(255) NOT NULL,
    domain     VARCHAR(255),
    plan       VARCHAR(50) NOT NULL DEFAULT 'starter',
    status     VARCHAR(50) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS customers (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL,
    customer_id_seq VARCHAR(20),
    name            VARCHAR(255) NOT NULL,
    phone           VARCHAR(20) NOT NULL,
    email           VARCHAR(255),
    package_id      UUID NOT NULL,
    status          VARCHAR(20) NOT NULL DEFAULT 'pending',
    deleted_at      TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
