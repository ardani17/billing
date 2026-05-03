-- Migrasi: membuat tabel customers dengan schema lengkap.
-- Menggantikan tabel sample dari migrasi 000002.

CREATE TABLE customers (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         UUID NOT NULL REFERENCES tenants(id),
    customer_id_seq   VARCHAR(20),
    name              VARCHAR(255) NOT NULL,
    phone             VARCHAR(20) NOT NULL,
    email             VARCHAR(255),
    address           TEXT NOT NULL,
    area_id           UUID REFERENCES areas(id) ON DELETE SET NULL,
    latitude          DECIMAL(10, 7) NOT NULL,
    longitude         DECIMAL(10, 7) NOT NULL,
    package_id        UUID NOT NULL,
    activation_date   DATE NOT NULL,
    due_date          INTEGER NOT NULL,
    connection_method VARCHAR(20) NOT NULL,
    pppoe_username    VARCHAR(100),
    pppoe_password    VARCHAR(100),
    mac_address       VARCHAR(17),
    router_id         UUID,
    odp_port          VARCHAR(100),
    credit_balance    BIGINT NOT NULL DEFAULT 0,
    notes             TEXT,
    status            VARCHAR(20) NOT NULL DEFAULT 'pending',
    deleted_at        TIMESTAMPTZ,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- CHECK constraints
    CONSTRAINT chk_customers_due_date CHECK (due_date >= 1 AND due_date <= 28),
    CONSTRAINT chk_customers_connection_method CHECK (
        connection_method IN ('pppoe', 'hotspot', 'dhcp_binding', 'static')
    ),
    CONSTRAINT chk_customers_status CHECK (
        status IN ('pending', 'aktif', 'isolir', 'suspend', 'berhenti')
    )
);

-- Aktifkan RLS pada tabel customers
ALTER TABLE customers ENABLE ROW LEVEL SECURITY;

-- Policy: isolasi data per tenant (SELECT, UPDATE, DELETE)
CREATE POLICY tenant_isolation ON customers
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

-- Policy: INSERT harus sesuai tenant session
CREATE POLICY tenant_insert ON customers
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);

-- Unique constraints
ALTER TABLE customers ADD CONSTRAINT uq_customers_tenant_phone
    UNIQUE (tenant_id, phone);
ALTER TABLE customers ADD CONSTRAINT uq_customers_tenant_id_seq
    UNIQUE (tenant_id, customer_id_seq);

-- Composite indexes untuk performa query
CREATE INDEX idx_customers_tenant_status ON customers(tenant_id, status);
CREATE INDEX idx_customers_tenant_id_seq ON customers(tenant_id, customer_id_seq);
CREATE INDEX idx_customers_tenant_phone ON customers(tenant_id, phone);
CREATE INDEX idx_customers_tenant_area ON customers(tenant_id, area_id);
CREATE INDEX idx_customers_tenant_package ON customers(tenant_id, package_id);
CREATE INDEX idx_customers_tenant_due_date ON customers(tenant_id, due_date);

-- Partial index: exclude soft-deleted dari query umum
CREATE INDEX idx_customers_active ON customers(tenant_id)
    WHERE deleted_at IS NULL;
