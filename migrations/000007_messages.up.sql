-- ==============================================================================
-- ZSMS Migration: 000007_messages.up.sql
-- ==============================================================================

CREATE TABLE IF NOT EXISTS messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    phone_id UUID NOT NULL REFERENCES phones(id) ON DELETE CASCADE,
    thread_id UUID REFERENCES message_threads(id) ON DELETE SET NULL,
    request_id VARCHAR(100) UNIQUE NULL,
    direction VARCHAR(10) NOT NULL,
    sender VARCHAR(30) NOT NULL,
    recipient VARCHAR(30) NOT NULL,
    content TEXT NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'queued',
    error_code VARCHAR(50) NULL,
    error_message TEXT NULL,
    sim_slot INT NOT NULL DEFAULT 1,
    retry_count INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    sent_at TIMESTAMPTZ NULL,
    delivered_at TIMESTAMPTZ NULL,
    failed_at TIMESTAMPTZ NULL
);

CREATE INDEX IF NOT EXISTS idx_messages_user_phone ON messages(user_id, phone_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_messages_thread ON messages(thread_id, created_at ASC);
CREATE INDEX IF NOT EXISTS idx_messages_status ON messages(status);
CREATE INDEX IF NOT EXISTS idx_messages_request_id ON messages(request_id) WHERE request_id IS NOT NULL;
