-- Migrasi: membuat tabel users dengan Row Level Security (RLS).
-- Tabel users menyimpan data pengguna per tenant, termasuk kredensial,
-- role, dan status verifikasi email. Mendukung login via email/password
-- maupun Google OAuth.

CREATE TABLE users (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name            VARCHAR(255) NOT NULL,
    email           VARCHAR(255) NOT NULL,
    phone           VARCHAR(50),
    password_hash   VARCHAR(255),
    role            VARCHAR(50) NOT NULL DEFAULT 'operator',
    email_verified  BOOLEAN NOT NULL DEFAULT false,
    google_id       VARCHAR(255),
    status          VARCHAR(50) NOT NULL DEFAULT 'active',
    last_login      TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Constraint unik: satu email hanya boleh dipakai sekali per tenant
ALTER TABLE users ADD CONSTRAINT uq_users_tenant_email UNIQUE (tenant_id, email);

-- Aktifkan RLS pada tabel users untuk isolasi data antar tenant
ALTER TABLE users ENABLE ROW LEVEL SECURITY;

-- Policy: hanya baris dengan tenant_id yang cocok dengan session variable
-- app.tenant_id yang bisa diakses (SELECT, UPDATE, DELETE)
CREATE POLICY tenant_isolation ON users
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

-- Policy untuk INSERT: memastikan tenant_id yang di-insert sesuai dengan session variable
CREATE POLICY tenant_insert ON users
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);

-- Index pada tenant_id untuk performa query dan RLS filtering
CREATE INDEX idx_users_tenant_id ON users(tenant_id);

-- Index komposit untuk lookup user berdasarkan email per tenant
CREATE INDEX idx_users_tenant_email ON users(tenant_id, email);

-- Index parsial untuk lookup user berdasarkan Google ID (hanya yang terisi)
CREATE INDEX idx_users_google_id ON users(google_id) WHERE google_id IS NOT NULL;

-- Index komposit untuk query user berdasarkan status per tenant
CREATE INDEX idx_users_status ON users(tenant_id, status);
