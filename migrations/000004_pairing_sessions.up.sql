-- ==============================================================================
-- ZSMS Migration: 000004_pairing_sessions.up.sql
-- ==============================================================================

CREATE TABLE IF NOT EXISTS pairing_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    pairing_code VARCHAR(6) NOT NULL,
    session_token VARCHAR(64) UNIQUE NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    claimed_at TIMESTAMPTZ NULL,
    claimed_by_phone_id UUID REFERENCES phones(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_pairing_code_active ON pairing_sessions(pairing_code, expires_at) WHERE claimed_at IS NULL;
