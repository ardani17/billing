-- Migrasi: membuat tabel sessions untuk tracking sesi login aktif.
-- Setiap login dari device berbeda membuat record session baru.

CREATE TABLE sessions (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash  VARCHAR(255) NOT NULL,
    device_info VARCHAR(500),
    ip_address  VARCHAR(45),
    expires_at  TIMESTAMPTZ NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index untuk lookup session berdasarkan user (list sessions)
CREATE INDEX idx_sessions_user_id ON sessions(user_id);

-- Index untuk lookup session berdasarkan token hash (refresh token validation)
CREATE INDEX idx_sessions_token_hash ON sessions(token_hash);

-- Index untuk cleanup session expired
CREATE INDEX idx_sessions_expires_at ON sessions(expires_at);
