-- Migrasi: membuat tabel password_resets dan email_verifications.
-- Token disimpan sebagai SHA-256 hash, plaintext hanya dikirim ke user.

CREATE TABLE password_resets (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash  VARCHAR(255) NOT NULL UNIQUE,
    expires_at  TIMESTAMPTZ NOT NULL,
    used        BOOLEAN NOT NULL DEFAULT false,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index untuk lookup berdasarkan token hash
CREATE INDEX idx_password_resets_token_hash ON password_resets(token_hash);

-- Index untuk cleanup dan invalidasi token per user
CREATE INDEX idx_password_resets_user_id ON password_resets(user_id);

CREATE TABLE email_verifications (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash  VARCHAR(255) NOT NULL UNIQUE,
    expires_at  TIMESTAMPTZ NOT NULL,
    used        BOOLEAN NOT NULL DEFAULT false,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index untuk lookup berdasarkan token hash
CREATE INDEX idx_email_verifications_token_hash ON email_verifications(token_hash);

-- Index untuk cleanup dan invalidasi token per user
CREATE INDEX idx_email_verifications_user_id ON email_verifications(user_id);
