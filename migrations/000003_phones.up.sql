-- ==============================================================================
-- ZSMS Migration: 000003_phones.up.sql
-- ==============================================================================

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
