-- ==============================================================================
-- ZSMS Migration: 000006_message_threads.up.sql
-- ==============================================================================

CREATE TABLE IF NOT EXISTS message_threads (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    phone_id UUID NOT NULL REFERENCES phones(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    recipient_number VARCHAR(30) NOT NULL,
    last_message_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    snippet TEXT NULL,
    unread_count INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (phone_id, recipient_number)
);

CREATE INDEX IF NOT EXISTS idx_threads_user_phone ON message_threads(user_id, phone_id, last_message_at DESC);
