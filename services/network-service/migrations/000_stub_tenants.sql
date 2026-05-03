-- Stub: tabel tenants untuk referensi foreign key.
-- Tabel asli dikelola oleh billing-api. File ini hanya untuk sqlc schema parsing.

CREATE TABLE IF NOT EXISTS tenants (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name       VARCHAR(255) NOT NULL,
    domain     VARCHAR(255),
    plan       VARCHAR(50) NOT NULL DEFAULT 'starter',
    status     VARCHAR(50) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
