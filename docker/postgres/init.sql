-- ==============================================================================
-- ZSMS PostgreSQL Initialization Script (Executed on container first boot)
-- ==============================================================================

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- 1. System Settings Table
CREATE TABLE IF NOT EXISTS system_settings (
    key VARCHAR(100) PRIMARY KEY,
    value JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO system_settings (key, value)
VALUES (
    'zsms_instance_info',
    '{"app_name": "ZSMS", "version": "1.0.0-foundation", "vendor": "ZTechAI", "initialized_at": "2026-09-07T00:00:00Z"}'::jsonb
)
ON CONFLICT (key) DO NOTHING;

-- 2. Users Table
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    full_name VARCHAR(150) NOT NULL,
    status VARCHAR(30) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ NULL
);
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email) WHERE deleted_at IS NULL;

-- 3. Phones Table
CREATE TABLE IF NOT EXISTS phones (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    device_identifier VARCHAR(100) NOT NULL,
    phone_number VARCHAR(30) NOT NULL DEFAULT '',
    sim_carrier VARCHAR(100) NOT NULL DEFAULT '',
    sim_slot_count INT NOT NULL DEFAULT 1,
    default_sim_slot INT NOT NULL DEFAULT 1,
    battery_level INT NOT NULL DEFAULT 100,
    battery_charging BOOLEAN NOT NULL DEFAULT FALSE,
    network_type VARCHAR(30) NOT NULL DEFAULT 'wifi',
    signal_strength INT NULL,
    app_version VARCHAR(30) NOT NULL DEFAULT '1.0.0',
    os_version VARCHAR(50) NOT NULL DEFAULT '',
    status VARCHAR(30) NOT NULL DEFAULT 'pending',
    last_seen_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ NULL,
    UNIQUE (user_id, device_identifier)
);
CREATE INDEX IF NOT EXISTS idx_phones_user_id ON phones(user_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_phones_status ON phones(status) WHERE deleted_at IS NULL;

-- 4. Pairing Sessions Table
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

-- 5. Phone Credentials Table
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

-- 6. Message Threads Table
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

-- 7. Messages Table
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

-- 8. Message Events Table
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
