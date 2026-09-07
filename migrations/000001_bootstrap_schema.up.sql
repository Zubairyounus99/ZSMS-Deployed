-- ==============================================================================
-- ZSMS Migration: 000001_bootstrap_schema.up.sql
-- Brand: ZTechAI
-- Purpose: Initialize PostgreSQL extensions and baseline bootstrap metadata
-- ==============================================================================

-- 1. Enable cryptographic UUID generation
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- 2. System Settings Table (Bootstrap store for instance parameters)
CREATE TABLE IF NOT EXISTS system_settings (
    key VARCHAR(100) PRIMARY KEY,
    value JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Insert baseline bootstrap version record
INSERT INTO system_settings (key, value)
VALUES (
    'zsms_instance_info',
    '{"app_name": "ZSMS", "version": "1.0.0-foundation", "vendor": "ZTechAI", "initialized_at": "2026-09-07T00:00:00Z"}'::jsonb
)
ON CONFLICT (key) DO NOTHING;
