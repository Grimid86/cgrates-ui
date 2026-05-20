-- UI-Bill Initial Schema Migration
-- PostgreSQL 15+ with partitioning, JSONB, UUID
-- Created: 2026-05-20

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";
CREATE EXTENSION IF NOT EXISTS "btree_gin";

-- ============================================================
-- 1. CORE TABLES
-- ============================================================

CREATE TABLE tenant (
    id BIGSERIAL PRIMARY KEY,
    uuid UUID DEFAULT gen_random_uuid() NOT NULL,
    name VARCHAR(100) NOT NULL,
    code VARCHAR(50) UNIQUE NOT NULL,
    is_active BOOLEAN DEFAULT true NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW() NOT NULL,
    updated_at TIMESTAMPTZ DEFAULT NOW() NOT NULL
);

CREATE TABLE language (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(5) UNIQUE NOT NULL,
    name VARCHAR(50) NOT NULL,
    is_active BOOLEAN DEFAULT true NOT NULL,
    default_rtl BOOLEAN DEFAULT false NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW() NOT NULL,
    CONSTRAINT chk_language_code CHECK (code ~ '^[a-z]{2}(-[A-Z]{2})?$')
);

CREATE TABLE roles (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(50) UNIQUE NOT NULL,
    name VARCHAR(100) NOT NULL,
    permissions JSONB NOT NULL DEFAULT '[]',
    is_system BOOLEAN DEFAULT false NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW() NOT NULL
);

CREATE TABLE permissions (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(100) UNIQUE NOT NULL,
    resource VARCHAR(50) NOT NULL,
    action VARCHAR(20) NOT NULL,
    description TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW() NOT NULL,
    CONSTRAINT chk_permission_action CHECK (action IN ('read', 'write', 'delete', 'execute'))
);

CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,
    uuid UUID DEFAULT gen_random_uuid() NOT NULL,
    tenant_id BIGINT NOT NULL REFERENCES tenant(id) ON DELETE CASCADE,
    email VARCHAR(255) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    role_id BIGINT NOT NULL REFERENCES roles(id) ON DELETE RESTRICT,
    locale VARCHAR(5) DEFAULT 'en' NOT NULL,
    mfa_secret VARCHAR(255),
    mfa_enabled BOOLEAN DEFAULT false NOT NULL,
    mfa_backup_codes JSONB DEFAULT '[]',
    is_active BOOLEAN DEFAULT true NOT NULL,
    last_login_at TIMESTAMPTZ,
    failed_login_attempts INT DEFAULT 0 NOT NULL,
    locked_until TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW() NOT NULL,
    updated_at TIMESTAMPTZ DEFAULT NOW() NOT NULL,
    CONSTRAINT chk_user_email CHECK (email ~* '^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$'),
    CONSTRAINT chk_user_locale CHECK (locale ~ '^[a-z]{2}(-[A-Z]{2})?$'),
    UNIQUE(tenant_id, email)
);

-- ============================================================
-- 2. LOCALIZATION TABLES
-- ============================================================

CREATE TABLE translation (
    id BIGSERIAL PRIMARY KEY,
    language_id BIGINT NOT NULL REFERENCES language(id) ON DELETE CASCADE,
    key VARCHAR(255) NOT NULL,
    value TEXT NOT NULL,
    category VARCHAR(50) DEFAULT 'common' NOT NULL,
    updated_at TIMESTAMPTZ DEFAULT NOW() NOT NULL,
    UNIQUE(language_id, key, category)
);

-- ============================================================
-- 3. WHITE-LABEL TABLES
-- ============================================================

CREATE TABLE branding_config (
    id BIGSERIAL PRIMARY KEY,
    tenant_id BIGINT NOT NULL REFERENCES tenant(id) ON DELETE CASCADE,
    primary_color VARCHAR(7) DEFAULT '#007bff' NOT NULL,
    secondary_color VARCHAR(7) DEFAULT '#6c757d' NOT NULL,
    accent_color VARCHAR(7) DEFAULT '#28a745' NOT NULL,
    danger_color VARCHAR(7) DEFAULT '#dc3545' NOT NULL,
    logo_url TEXT,
    favicon_url TEXT,
    login_background_url TEXT,
    product_name VARCHAR(100) DEFAULT 'CGRateS Billing' NOT NULL,
    support_email VARCHAR(255),
    support_phone VARCHAR(50),
    support_telegram VARCHAR(100),
    timezone VARCHAR(50) DEFAULT 'UTC' NOT NULL,
    date_format VARCHAR(20) DEFAULT 'DD.MM.YYYY' NOT NULL,
    first_day_of_week INT DEFAULT 1 NOT NULL,
    currency_symbol VARCHAR(10) DEFAULT '$' NOT NULL,
    updated_at TIMESTAMPTZ DEFAULT NOW() NOT NULL,
    UNIQUE(tenant_id),
    CONSTRAINT chk_branding_primary_color CHECK (primary_color ~ '^#[0-9A-Fa-f]{6}$'),
    CONSTRAINT chk_branding_secondary_color CHECK (secondary_color ~ '^#[0-9A-Fa-f]{6}$'),
    CONSTRAINT chk_branding_accent_color CHECK (accent_color ~ '^#[0-9A-Fa-f]{6}$'),
    CONSTRAINT chk_branding_danger_color CHECK (danger_color ~ '^#[0-9A-Fa-f]{6}$')
);

CREATE TABLE email_template (
    id BIGSERIAL PRIMARY KEY,
    tenant_id BIGINT NOT NULL REFERENCES tenant(id) ON DELETE CASCADE,
    language_id BIGINT REFERENCES language(id) ON DELETE SET NULL,
    template_type VARCHAR(50) NOT NULL,
    subject_template VARCHAR(255) NOT NULL,
    body_html_template TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW() NOT NULL,
    updated_at TIMESTAMPTZ DEFAULT NOW() NOT NULL,
    UNIQUE(tenant_id, language_id, template_type)
);

CREATE TABLE domain_tenant_mapping (
    id BIGSERIAL PRIMARY KEY,
    domain VARCHAR(255) UNIQUE NOT NULL,
    tenant_id BIGINT NOT NULL REFERENCES tenant(id) ON DELETE CASCADE,
    is_active BOOLEAN DEFAULT true NOT NULL,
    ssl_cert_id VARCHAR(100),
    created_at TIMESTAMPTZ DEFAULT NOW() NOT NULL
);

-- ============================================================
-- 4. MULTI-PORTAL TABLES
-- ============================================================

CREATE TABLE subscriber_credentials (
    id BIGSERIAL PRIMARY KEY,
    tenant_id BIGINT REFERENCES tenant(id) ON DELETE CASCADE,
    msisdn VARCHAR(20) UNIQUE NOT NULL,
    imsi VARCHAR(15),
    pin_hash VARCHAR(255),
    password_hash VARCHAR(255),
    email VARCHAR(255),
    category VARCHAR(20) DEFAULT 'prepaid' NOT NULL,
    is_active BOOLEAN DEFAULT true NOT NULL,
    failed_login_attempts INT DEFAULT 0 NOT NULL,
    locked_until TIMESTAMPTZ,
    last_login_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW() NOT NULL,
    CONSTRAINT chk_subscriber_msisdn CHECK (msisdn ~ '^[0-9]+$'),
    CONSTRAINT chk_subscriber_category CHECK (category IN ('prepaid', 'postpaid', 'enterprise'))
);

CREATE TABLE subscriber_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    subscriber_id BIGINT REFERENCES subscriber_credentials(id) ON DELETE CASCADE,
    token_hash VARCHAR(255) NOT NULL,
    ip_address INET,
    user_agent TEXT,
    issued_at TIMESTAMPTZ DEFAULT NOW() NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ
);

CREATE TABLE portal_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id BIGINT REFERENCES users(id) ON DELETE CASCADE,
    portal_type VARCHAR(20) NOT NULL,
    token_hash VARCHAR(255) NOT NULL,
    ip_address INET,
    mfa_verified BOOLEAN DEFAULT false NOT NULL,
    issued_at TIMESTAMPTZ DEFAULT NOW() NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    CONSTRAINT chk_portal_type CHECK (portal_type IN ('admin', 'operator', 'support'))
);

CREATE TABLE api_keys (
    id BIGSERIAL PRIMARY KEY,
    tenant_id BIGINT REFERENCES tenant(id) ON DELETE CASCADE,
    name VARCHAR(100),
    key_hash VARCHAR(255) UNIQUE NOT NULL,
    portal_type VARCHAR(20) NOT NULL,
    scopes JSONB NOT NULL DEFAULT '[]',
    rate_limit_rps INT DEFAULT 10 NOT NULL,
    expires_at TIMESTAMPTZ,
    last_used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW() NOT NULL,
    CONSTRAINT chk_api_key_portal_type CHECK (portal_type IN ('selfcare', 'operator', 'admin'))
);

-- ============================================================
-- 5. AUDIT LOG (Partitioned)
-- ============================================================

CREATE TABLE audit_log (
    id BIGSERIAL,
    tenant_id BIGINT NOT NULL,
    user_id BIGINT,
    portal_type VARCHAR(20) NOT NULL,
    action VARCHAR(50) NOT NULL,
    entity_type VARCHAR(50),
    entity_id VARCHAR(255),
    old_data JSONB,
    new_data JSONB,
    ip_address INET,
    user_agent TEXT,
    request_id UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (id, created_at),
    CONSTRAINT chk_audit_portal_type CHECK (portal_type IN ('selfcare', 'operator', 'admin'))
) PARTITION BY RANGE (created_at);

-- Create initial partitions for 2026
CREATE TABLE audit_log_y2026m01 PARTITION OF audit_log
    FOR VALUES FROM ('2026-01-01') TO ('2026-02-01');
CREATE TABLE audit_log_y2026m02 PARTITION OF audit_log
    FOR VALUES FROM ('2026-02-01') TO ('2026-03-01');
CREATE TABLE audit_log_y2026m03 PARTITION OF audit_log
    FOR VALUES FROM ('2026-03-01') TO ('2026-04-01');
CREATE TABLE audit_log_y2026m04 PARTITION OF audit_log
    FOR VALUES FROM ('2026-04-01') TO ('2026-05-01');
CREATE TABLE audit_log_y2026m05 PARTITION OF audit_log
    FOR VALUES FROM ('2026-05-01') TO ('2026-06-01');
CREATE TABLE audit_log_y2026m06 PARTITION OF audit_log
    FOR VALUES FROM ('2026-06-01') TO ('2026-07-01');

-- ============================================================
-- 6. AUXILIARY TABLES
-- ============================================================

CREATE TABLE tariff_sandbox (
    id BIGSERIAL PRIMARY KEY,
    tenant_id BIGINT NOT NULL REFERENCES tenant(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    cgrates_tp_id VARCHAR(64),
    config_json JSONB NOT NULL DEFAULT '{}',
    status VARCHAR(20) DEFAULT 'draft' NOT NULL,
    created_by BIGINT NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ DEFAULT NOW() NOT NULL,
    updated_at TIMESTAMPTZ DEFAULT NOW() NOT NULL,
    CONSTRAINT chk_tariff_status CHECK (status IN ('draft', 'testing', 'active', 'archived'))
);

CREATE TABLE balance_history (
    id BIGSERIAL PRIMARY KEY,
    tenant_id BIGINT NOT NULL,
    subscriber_id BIGINT NOT NULL,
    balance_type VARCHAR(20) NOT NULL,
    amount_before DECIMAL(18,6),
    amount_after DECIMAL(18,6),
    operation VARCHAR(20) NOT NULL,
    extra_data JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW() NOT NULL,
    CONSTRAINT chk_balance_type CHECK (balance_type IN ('*monetary', '*data', '*sms', '*voice', '*bonus')),
    CONSTRAINT chk_balance_operation CHECK (operation IN ('topup', 'charge', 'adjust', 'expire', 'transfer'))
);

CREATE TABLE action_trigger_presets (
    id BIGSERIAL PRIMARY KEY,
    tenant_id BIGINT NOT NULL,
    name VARCHAR(100) NOT NULL,
    trigger_type VARCHAR(50) NOT NULL,
    conditions JSONB NOT NULL DEFAULT '{}',
    actions JSONB NOT NULL DEFAULT '{}',
    is_active BOOLEAN DEFAULT true NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW() NOT NULL,
    CONSTRAINT chk_trigger_type CHECK (trigger_type IN ('threshold', 'expiration', 'event', 'schedule'))
);

CREATE TABLE diameter_peer_monitoring (
    id BIGSERIAL PRIMARY KEY,
    tenant_id BIGINT NOT NULL,
    peer_address VARCHAR(255) NOT NULL,
    peer_type VARCHAR(20) NOT NULL,
    status VARCHAR(20) NOT NULL,
    last_seen_at TIMESTAMPTZ,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW() NOT NULL,
    CONSTRAINT chk_diameter_peer_type CHECK (peer_type IN ('gy', 'gx', 's6a', 's9')),
    CONSTRAINT chk_diameter_status CHECK (status IN ('online', 'offline', 'degraded', 'unknown'))
);

CREATE TABLE rpc_idempotency_keys (
    id BIGSERIAL PRIMARY KEY,
    key VARCHAR(255) NOT NULL,
    portal_type VARCHAR(20) NOT NULL,
    request_hash VARCHAR(255),
    response_data JSONB,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW() NOT NULL,
    UNIQUE(key, portal_type),
    CONSTRAINT chk_idempotency_portal_type CHECK (portal_type IN ('selfcare', 'operator', 'admin'))
);
