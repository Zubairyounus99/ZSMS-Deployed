-- ==============================================================================
-- ZSMS Migration: 000005_phone_credentials.up.sql
-- ==============================================================================

CREATE TABLE IF NOT EXISTS phone_credentials (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    phone_id UUID NOT NULL REFERENCES phones(id) ON DELETE CASCADE,
    token_hash VARCHAR(64) UNIQUE NOT NULL,
    token_prefix VARCHAR(16) NOT NULL,
    last_used_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    revoked_at TIMESTAMPTZ NULL
);

CREATE INDEX IF NOT EXISTS idx_phone_credentials_hash ON phone_credentials(token_hash) WHERE revoked_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_phone_credentials_phone_id ON phone_credentials(phone_id);
