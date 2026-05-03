-- Tabel notification_configs: menyimpan konfigurasi provider notifikasi per tenant per channel.
-- Setiap tenant bisa punya satu konfigurasi per channel (WA, SMS, Email).
CREATE TABLE notification_configs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    channel VARCHAR(20) NOT NULL,
    provider VARCHAR(50) NOT NULL,
    credentials JSONB NOT NULL,
    is_enabled BOOLEAN NOT NULL DEFAULT false,
    priority INTEGER NOT NULL DEFAULT 1,
    settings JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_notif_config_channel CHECK (channel IN ('whatsapp', 'sms', 'email')),
    CONSTRAINT uq_notif_config_tenant_channel UNIQUE (tenant_id, channel)
);

-- RLS: isolasi data per tenant
ALTER TABLE notification_configs ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_select_notif_config ON notification_configs
    FOR SELECT USING (tenant_id = current_setting('app.tenant_id')::uuid);
CREATE POLICY tenant_insert_notif_config ON notification_configs
    FOR INSERT WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);
CREATE POLICY tenant_update_notif_config ON notification_configs
    FOR UPDATE USING (tenant_id = current_setting('app.tenant_id')::uuid);
CREATE POLICY tenant_delete_notif_config ON notification_configs
    FOR DELETE USING (tenant_id = current_setting('app.tenant_id')::uuid);

-- Index untuk query performa
CREATE INDEX idx_notif_config_tenant_enabled ON notification_configs (tenant_id, is_enabled);

-- Tabel notification_templates: menyimpan template notifikasi per tenant.
-- Setiap template punya slug unik per tenant dan bisa di-link ke event_type.
CREATE TABLE notification_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    slug VARCHAR(100) NOT NULL,
    name VARCHAR(255) NOT NULL,
    category VARCHAR(20) NOT NULL,
    event_type VARCHAR(100),
    channels JSONB NOT NULL DEFAULT '[]',
    body_whatsapp TEXT,
    body_sms TEXT,
    body_email_subject TEXT,
    body_email_html TEXT,
    variables JSONB NOT NULL DEFAULT '[]',
    is_active BOOLEAN NOT NULL DEFAULT true,
    is_default BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_notif_template_category CHECK (
        category IN ('transactional', 'reminder', 'promotion', 'information')
    ),
    CONSTRAINT uq_notif_template_tenant_slug UNIQUE (tenant_id, slug)
);

-- RLS: isolasi data per tenant
ALTER TABLE notification_templates ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_select_notif_template ON notification_templates
    FOR SELECT USING (tenant_id = current_setting('app.tenant_id')::uuid);
CREATE POLICY tenant_insert_notif_template ON notification_templates
    FOR INSERT WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);
CREATE POLICY tenant_update_notif_template ON notification_templates
    FOR UPDATE USING (tenant_id = current_setting('app.tenant_id')::uuid);
CREATE POLICY tenant_delete_notif_template ON notification_templates
    FOR DELETE USING (tenant_id = current_setting('app.tenant_id')::uuid);

-- Index untuk lookup berdasarkan event_type dan status aktif
CREATE INDEX idx_notif_template_tenant_event ON notification_templates (tenant_id, event_type);
CREATE INDEX idx_notif_template_tenant_active ON notification_templates (tenant_id, is_active);

-- Tabel notification_logs: mencatat setiap pengiriman notifikasi beserta status dan detail.
-- Digunakan untuk audit trail, retry tracking, dan deduplication.
CREATE TABLE notification_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    customer_id UUID NOT NULL,
    template_id UUID REFERENCES notification_templates(id),
    channel VARCHAR(20) NOT NULL,
    provider VARCHAR(50) NOT NULL,
    recipient VARCHAR(255) NOT NULL,
    subject TEXT,
    body TEXT NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    retry_count INTEGER NOT NULL DEFAULT 0,
    max_retries INTEGER NOT NULL DEFAULT 3,
    error_message TEXT,
    dedup_key VARCHAR(500),
    metadata JSONB DEFAULT '{}',
    sent_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_notif_log_status CHECK (
        status IN ('pending', 'sending', 'sent', 'delivered', 'read', 'failed', 'retrying', 'skipped')
    ),
    CONSTRAINT chk_notif_log_channel CHECK (channel IN ('whatsapp', 'sms', 'email'))
);

-- RLS: isolasi data per tenant
ALTER TABLE notification_logs ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_select_notif_log ON notification_logs
    FOR SELECT USING (tenant_id = current_setting('app.tenant_id')::uuid);
CREATE POLICY tenant_insert_notif_log ON notification_logs
    FOR INSERT WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);
CREATE POLICY tenant_update_notif_log ON notification_logs
    FOR UPDATE USING (tenant_id = current_setting('app.tenant_id')::uuid);
CREATE POLICY tenant_delete_notif_log ON notification_logs
    FOR DELETE USING (tenant_id = current_setting('app.tenant_id')::uuid);

-- Index untuk query performa
CREATE INDEX idx_notif_log_tenant_customer ON notification_logs (tenant_id, customer_id);
CREATE INDEX idx_notif_log_tenant_status ON notification_logs (tenant_id, status);
CREATE INDEX idx_notif_log_tenant_created ON notification_logs (tenant_id, created_at DESC);
CREATE INDEX idx_notif_log_dedup ON notification_logs (dedup_key);

-- Partial unique index untuk deduplication: hanya satu notifikasi aktif per dedup_key dalam 1 jam.
-- Status 'skipped' dikecualikan agar tidak memblokir pengiriman ulang.
CREATE UNIQUE INDEX uq_notif_log_dedup_active
    ON notification_logs (dedup_key)
    WHERE dedup_key IS NOT NULL
      AND status NOT IN ('skipped', 'failed');
