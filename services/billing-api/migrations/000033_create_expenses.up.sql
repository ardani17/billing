-- Migrasi: membuat tabel expenses untuk menyimpan pengeluaran bisnis per tenant.
-- Setiap pengeluaran terkait dengan satu kategori dan satu user yang membuat.
-- Mendukung soft delete, recurring expenses, dan dilindungi oleh RLS.

CREATE TABLE expenses (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES tenants(id),
    category_id     UUID NOT NULL REFERENCES expense_categories(id),
    amount          BIGINT NOT NULL CHECK (amount > 0),
    description     TEXT NOT NULL DEFAULT '',
    expense_date    DATE NOT NULL,
    is_recurring    BOOLEAN NOT NULL DEFAULT false,
    recurring_day   INTEGER CHECK (recurring_day >= 1 AND recurring_day <= 28),
    created_by_id   UUID NOT NULL REFERENCES users(id),
    deleted_at      TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Aktifkan RLS pada tabel expenses
ALTER TABLE expenses ENABLE ROW LEVEL SECURITY;

-- Policy: isolasi data per tenant (SELECT, UPDATE, DELETE)
CREATE POLICY expenses_tenant_isolation ON expenses
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

-- Policy: INSERT harus sesuai tenant session
CREATE POLICY expenses_tenant_insert ON expenses
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);

-- Index: pencarian pengeluaran per tenant dan periode (hanya yang belum dihapus)
CREATE INDEX idx_expenses_tenant_period
    ON expenses (tenant_id, expense_date)
    WHERE deleted_at IS NULL;
