-- Migrasi: sesi login reseller terpisah dari sessions admin.
-- Tabel sessions memiliki FK ke users, sedangkan reseller memakai tabel resellers.

CREATE TABLE reseller_sessions (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    reseller_id UUID NOT NULL REFERENCES resellers(id) ON DELETE CASCADE,
    token_hash  VARCHAR(255) NOT NULL,
    device_info VARCHAR(500),
    ip_address  VARCHAR(45),
    expires_at  TIMESTAMPTZ NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_reseller_sessions_reseller_id ON reseller_sessions(reseller_id);
CREATE INDEX idx_reseller_sessions_token_hash ON reseller_sessions(token_hash);
CREATE INDEX idx_reseller_sessions_expires_at ON reseller_sessions(expires_at);
