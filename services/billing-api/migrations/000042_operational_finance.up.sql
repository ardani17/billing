-- Modul Keuangan Operasional: metadata expense, inventory, dan dasar cashflow.

ALTER TABLE expenses
    ADD COLUMN IF NOT EXISTS payment_method VARCHAR(50),
    ADD COLUMN IF NOT EXISTS vendor_name VARCHAR(255),
    ADD COLUMN IF NOT EXISTS reference_number VARCHAR(255),
    ADD COLUMN IF NOT EXISTS attachment_url TEXT,
    ADD COLUMN IF NOT EXISTS inventory_movement_id UUID;

CREATE TABLE IF NOT EXISTS inventory_items (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name         VARCHAR(255) NOT NULL,
    category     VARCHAR(100) NOT NULL,
    unit         VARCHAR(50) NOT NULL DEFAULT 'unit',
    track_serial BOOLEAN NOT NULL DEFAULT false,
    min_stock    INTEGER NOT NULL DEFAULT 0 CHECK (min_stock >= 0),
    default_cost BIGINT NOT NULL DEFAULT 0 CHECK (default_cost >= 0),
    is_active    BOOLEAN NOT NULL DEFAULT true,
    deleted_at   TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_inventory_items_tenant_name
    ON inventory_items(tenant_id, name)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_inventory_items_tenant_category
    ON inventory_items(tenant_id, category)
    WHERE deleted_at IS NULL;

ALTER TABLE inventory_items ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON inventory_items;
CREATE POLICY tenant_isolation ON inventory_items
    USING (tenant_id = current_setting('app.tenant_id')::uuid);
DROP POLICY IF EXISTS tenant_insert ON inventory_items;
CREATE POLICY tenant_insert ON inventory_items
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE TABLE IF NOT EXISTS inventory_assets (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id             UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    item_id               UUID NOT NULL REFERENCES inventory_items(id),
    serial_number         VARCHAR(255) NOT NULL,
    mac_address           VARCHAR(100),
    status                VARCHAR(50) NOT NULL DEFAULT 'in_stock',
    location_type         VARCHAR(50) NOT NULL DEFAULT 'warehouse',
    location_id           VARCHAR(255),
    assigned_customer_id  UUID REFERENCES customers(id),
    purchase_cost         BIGINT NOT NULL DEFAULT 0 CHECK (purchase_cost >= 0),
    purchase_date         DATE,
    warranty_until        DATE,
    deleted_at            TIMESTAMPTZ,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_inventory_asset_status CHECK (status IN ('in_stock','assigned','damaged','lost','rma','retired')),
    CONSTRAINT chk_inventory_asset_location CHECK (location_type IN ('warehouse','technician','customer','pop','odp','odc','damaged','rma','lost'))
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_inventory_assets_tenant_serial
    ON inventory_assets(tenant_id, serial_number)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_inventory_assets_tenant_item
    ON inventory_assets(tenant_id, item_id)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_inventory_assets_tenant_status
    ON inventory_assets(tenant_id, status)
    WHERE deleted_at IS NULL;

ALTER TABLE inventory_assets ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON inventory_assets;
CREATE POLICY tenant_isolation ON inventory_assets
    USING (tenant_id = current_setting('app.tenant_id')::uuid);
DROP POLICY IF EXISTS tenant_insert ON inventory_assets;
CREATE POLICY tenant_insert ON inventory_assets
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE TABLE IF NOT EXISTS inventory_movements (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id          UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    item_id            UUID NOT NULL REFERENCES inventory_items(id),
    asset_id           UUID REFERENCES inventory_assets(id),
    movement_type      VARCHAR(50) NOT NULL,
    quantity           INTEGER NOT NULL CHECK (quantity <> 0),
    from_location_type VARCHAR(50),
    from_location_id   VARCHAR(255),
    to_location_type   VARCHAR(50),
    to_location_id     VARCHAR(255),
    customer_id        UUID REFERENCES customers(id),
    expense_id         UUID REFERENCES expenses(id),
    unit_cost          BIGINT NOT NULL DEFAULT 0 CHECK (unit_cost >= 0),
    notes              TEXT,
    created_by_id      UUID NOT NULL REFERENCES users(id),
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_inventory_movement_type CHECK (movement_type IN ('purchase','install','return','transfer','adjustment','damaged','lost'))
);

CREATE INDEX IF NOT EXISTS idx_inventory_movements_tenant_item
    ON inventory_movements(tenant_id, item_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_inventory_movements_tenant_type
    ON inventory_movements(tenant_id, movement_type, created_at DESC);

ALTER TABLE inventory_movements ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON inventory_movements;
CREATE POLICY tenant_isolation ON inventory_movements
    USING (tenant_id = current_setting('app.tenant_id')::uuid);
DROP POLICY IF EXISTS tenant_insert ON inventory_movements;
CREATE POLICY tenant_insert ON inventory_movements
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);
