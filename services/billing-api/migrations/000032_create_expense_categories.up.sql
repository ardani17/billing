-- Migrasi: membuat tabel expense_categories untuk menyimpan kategori pengeluaran per tenant.
-- Setiap kategori dimiliki oleh satu tenant dan dilindungi oleh RLS.
-- Mendukung hapus lunak dan kategori bawaan untuk tenant baru.

CREATE TABLE expense_categories (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES tenants(id),
    name        VARCHAR(255) NOT NULL,
    is_default  BOOLEAN NOT NULL DEFAULT false,
    deleted_at  TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Aktifkan RLS pada tabel expense_categories
ALTER TABLE expense_categories ENABLE ROW LEVEL SECURITY;

-- Policy: isolasi data per tenant (SELECT, UPDATE, DELETE)
CREATE POLICY expense_categories_tenant_isolation ON expense_categories
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

-- Policy: INSERT harus sesuai tenant session
CREATE POLICY expense_categories_tenant_insert ON expense_categories
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);

-- Unique constraint: nama kategori unik per tenant (hanya untuk yang belum dihapus)
CREATE UNIQUE INDEX uq_expense_categories_tenant_name
    ON expense_categories (tenant_id, name)
    WHERE deleted_at IS NULL;
