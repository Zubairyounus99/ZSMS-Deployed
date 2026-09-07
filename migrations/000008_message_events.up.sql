-- ==============================================================================
-- ZSMS Migration: 000008_message_events.up.sql
-- ==============================================================================

CREATE TABLE IF NOT EXISTS message_events (
    id BIGSERIAL PRIMARY KEY,
    message_id UUID NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    status VARCHAR(20) NOT NULL,
    source VARCHAR(20) NOT NULL,
    error_message TEXT NULL,
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_message_events_msg ON message_events(message_id, created_at ASC);
