-- Migrasi: membuat tabel webhook_logs untuk menyimpan log semua webhook request dari payment gateway.
-- Tabel ini bersifat append-only dan menyimpan seluruh request termasuk yang gagal verifikasi.
-- TIDAK menggunakan RLS karena webhook diterima sebelum identifikasi tenant.
-- tenant_id diisi setelah identifikasi payment link (bisa NULL saat penerimaan awal).

CREATE TABLE webhook_logs (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID REFERENCES tenants(id),
    gateway_provider    VARCHAR(20) NOT NULL,
    event_type          VARCHAR(100) NOT NULL,
    external_id         VARCHAR(255) NOT NULL,
    request_body        JSONB NOT NULL,
    source_ip           INET NOT NULL,
    signature_valid     BOOLEAN,
    processing_status   VARCHAR(20) NOT NULL DEFAULT 'received',
    error_message       TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- CHECK constraints
    CONSTRAINT chk_webhook_logs_provider CHECK (
        gateway_provider IN ('xendit', 'midtrans')
    ),
    CONSTRAINT chk_webhook_logs_processing_status CHECK (
        processing_status IN ('received', 'verified', 'processed', 'failed', 'duplicate')
    )
);

-- TIDAK mengaktifkan RLS: webhook logs tidak tenant-scoped saat penerimaan awal.
-- tenant_id diisi secara asinkron setelah identifikasi payment link.

-- Partial unique index: idempotency check untuk webhook yang sudah diproses
CREATE INDEX idx_webhook_logs_idempotency
    ON webhook_logs(external_id, event_type)
    WHERE processing_status = 'processed';

-- Index: lookup berdasarkan external_id dengan urutan terbaru
CREATE INDEX idx_webhook_logs_external_id
    ON webhook_logs(external_id, created_at DESC);

-- Partial index: untuk background job cleanup (hanya hapus log yang aman)
-- Tidak menghapus log dengan status 'failed' atau signature_valid = false
CREATE INDEX idx_webhook_logs_cleanup
    ON webhook_logs(created_at)
    WHERE processing_status NOT IN ('failed')
      AND (signature_valid IS NULL OR signature_valid = true);

-- Partial index: lookup berdasarkan tenant untuk query admin
CREATE INDEX idx_webhook_logs_tenant
    ON webhook_logs(tenant_id, created_at DESC)
    WHERE tenant_id IS NOT NULL;
