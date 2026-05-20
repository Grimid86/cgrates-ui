-- UI-Bill Indexes, Triggers, and Functions
-- Migration 002

-- ============================================================
-- 1. CORE TABLE INDEXES
-- ============================================================

-- Tenant indexes
CREATE INDEX idx_tenant_uuid ON tenant(uuid);
CREATE INDEX idx_tenant_code ON tenant(code);
CREATE INDEX idx_tenant_active ON tenant(is_active) WHERE is_active = true;

-- Users indexes
CREATE INDEX idx_users_tenant_email ON users(tenant_id, email);
CREATE INDEX idx_users_uuid ON users(uuid);
CREATE INDEX idx_users_role ON users(role_id);
CREATE INDEX idx_users_active ON users(is_active) WHERE is_active = true;
CREATE INDEX idx_users_locked ON users(locked_until) WHERE locked_until IS NOT NULL;

-- Roles indexes
CREATE INDEX idx_roles_code ON roles(code);

-- Permissions indexes
CREATE INDEX idx_permissions_resource ON permissions(resource);

-- ============================================================
-- 2. LOCALIZATION INDEXES
-- ============================================================

CREATE INDEX idx_translation_lookup ON translation(language_id, key, category);
CREATE INDEX idx_translation_category ON translation(category);

-- ============================================================
-- 3. WHITE-LABEL INDEXES
-- ============================================================

CREATE INDEX idx_branding_tenant ON branding_config(tenant_id);
CREATE INDEX idx_email_template_lookup ON email_template(tenant_id, template_type, language_id);
CREATE INDEX idx_domain_mapping_lookup ON domain_tenant_mapping(domain, is_active);
CREATE INDEX idx_domain_mapping_tenant ON domain_tenant_mapping(tenant_id);

-- ============================================================
-- 4. MULTI-PORTAL INDEXES
-- ============================================================

-- Subscriber credentials
CREATE INDEX idx_subscriber_msisdn ON subscriber_credentials(msisdn);
CREATE INDEX idx_subscriber_tenant ON subscriber_credentials(tenant_id);
CREATE INDEX idx_subscriber_imsi ON subscriber_credentials(imsi);
CREATE INDEX idx_subscriber_active ON subscriber_credentials(is_active) WHERE is_active = true;
CREATE INDEX idx_subscriber_locked ON subscriber_credentials(locked_until) WHERE locked_until IS NOT NULL;

-- Subscriber sessions
CREATE INDEX idx_sub_session_subscriber ON subscriber_sessions(subscriber_id);
CREATE INDEX idx_sub_session_expires ON subscriber_sessions(expires_at);
CREATE INDEX idx_sub_session_revoked ON subscriber_sessions(revoked_at) WHERE revoked_at IS NULL;

-- Portal sessions
CREATE INDEX idx_portal_session_user ON portal_sessions(user_id, portal_type);
CREATE INDEX idx_portal_session_expires ON portal_sessions(expires_at);
CREATE INDEX idx_portal_session_revoked ON portal_sessions(revoked_at) WHERE revoked_at IS NULL;

-- API keys
CREATE INDEX idx_api_key_hash ON api_keys(key_hash);
CREATE INDEX idx_api_key_tenant ON api_keys(tenant_id);
CREATE INDEX idx_api_key_expires ON api_keys(expires_at);

-- ============================================================
-- 5. AUDIT LOG INDEXES (on partitioned table — inherited)
-- ============================================================

CREATE INDEX idx_audit_tenant_created ON audit_log(tenant_id, created_at);
CREATE INDEX idx_audit_action ON audit_log(action);
CREATE INDEX idx_audit_entity ON audit_log(entity_type, entity_id);
CREATE INDEX idx_audit_request_id ON audit_log(request_id);
CREATE INDEX idx_audit_user ON audit_log(user_id) WHERE user_id IS NOT NULL;
CREATE INDEX idx_audit_portal ON audit_log(portal_type);

-- GIN indexes for JSONB search
CREATE INDEX idx_audit_old_data_gin ON audit_log USING GIN (old_data);
CREATE INDEX idx_audit_new_data_gin ON audit_log USING GIN (new_data);

-- ============================================================
-- 6. AUXILIARY TABLE INDEXES
-- ============================================================

CREATE INDEX idx_tariff_sandbox_tenant ON tariff_sandbox(tenant_id);
CREATE INDEX idx_tariff_sandbox_status ON tariff_sandbox(status);

CREATE INDEX idx_balance_history_subscriber ON balance_history(subscriber_id);
CREATE INDEX idx_balance_history_created ON balance_history(created_at);
CREATE INDEX idx_balance_history_type ON balance_history(balance_type);

CREATE INDEX idx_trigger_presets_tenant ON action_trigger_presets(tenant_id);
CREATE INDEX idx_trigger_presets_active ON action_trigger_presets(is_active) WHERE is_active = true;

CREATE INDEX idx_diameter_peer_tenant ON diameter_peer_monitoring(tenant_id);
CREATE INDEX idx_diameter_peer_status ON diameter_peer_monitoring(status);
CREATE INDEX idx_diameter_peer_address ON diameter_peer_monitoring(peer_address);

CREATE INDEX idx_idempotency_key_lookup ON rpc_idempotency_keys(key, portal_type);
CREATE INDEX idx_idempotency_expires ON rpc_idempotency_keys(expires_at);

-- ============================================================
-- 7. AUTO-UPDATE updated_at TRIGGER FUNCTION
-- ============================================================

CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Apply to tables with updated_at
CREATE TRIGGER trg_tenant_updated_at
    BEFORE UPDATE ON tenant
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trg_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trg_translation_updated_at
    BEFORE UPDATE ON translation
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trg_branding_config_updated_at
    BEFORE UPDATE ON branding_config
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trg_email_template_updated_at
    BEFORE UPDATE ON email_template
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trg_tariff_sandbox_updated_at
    BEFORE UPDATE ON tariff_sandbox
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ============================================================
-- 8. AUDIT LOG INSERT TRIGGER (for computed fields)
-- ============================================================

CREATE OR REPLACE FUNCTION audit_log_insert_hook()
RETURNS TRIGGER AS $$
BEGIN
    -- Ensure request_id is set if not provided
    IF NEW.request_id IS NULL THEN
        NEW.request_id = gen_random_uuid();
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_audit_log_insert
    BEFORE INSERT ON audit_log
    FOR EACH ROW EXECUTE FUNCTION audit_log_insert_hook();

-- ============================================================
-- 9. PARTITION AUTO-CREATION FUNCTION
-- ============================================================

CREATE OR REPLACE FUNCTION create_audit_log_partition()
RETURNS void AS $$
DECLARE
    partition_date DATE;
    partition_name TEXT;
    start_date TEXT;
    end_date TEXT;
BEGIN
    partition_date := DATE_TRUNC('month', NOW() + INTERVAL '1 month');
    partition_name := 'audit_log_y' || TO_CHAR(partition_date, 'YYYY') || 'm' || TO_CHAR(partition_date, 'MM');
    start_date := TO_CHAR(partition_date, 'YYYY-MM-DD');
    end_date := TO_CHAR(partition_date + INTERVAL '1 month', 'YYYY-MM-DD');
    
    EXECUTE format(
        'CREATE TABLE IF NOT EXISTS %I PARTITION OF audit_log FOR VALUES FROM (%L) TO (%L)',
        partition_name, start_date, end_date
    );
END;
$$ LANGUAGE plpgsql;

-- ============================================================
-- 10. ROW-LEVEL SECURITY POLICIES
-- ============================================================

-- Enable RLS on tenant-isolated tables
ALTER TABLE users ENABLE ROW LEVEL SECURITY;
ALTER TABLE subscriber_credentials ENABLE ROW LEVEL SECURITY;
ALTER TABLE branding_config ENABLE ROW LEVEL SECURITY;

-- Tenant isolation policy for users (root can see all)
CREATE POLICY tenant_users_isolation ON users
    FOR ALL
    USING (
        tenant_id = COALESCE(NULLIF(current_setting('app.current_tenant_id', true), ''), '0')::BIGINT
        OR current_setting('app.current_role', true) = 'root'
    );

-- Tenant isolation for subscribers
CREATE POLICY tenant_subscriber_isolation ON subscriber_credentials
    FOR ALL
    USING (
        tenant_id = COALESCE(NULLIF(current_setting('app.current_tenant_id', true), ''), '0')::BIGINT
        OR current_setting('app.current_role', true) = 'root'
    );

-- Branding isolation (users can only see their tenant branding)
CREATE POLICY tenant_branding_isolation ON branding_config
    FOR ALL
    USING (
        tenant_id = COALESCE(NULLIF(current_setting('app.current_tenant_id', true), ''), '0')::BIGINT
        OR current_setting('app.current_role', true) = 'root'
    );

-- ============================================================
-- 11. AUDIT TRIGGER FOR SENSITIVE TABLES
-- ============================================================

CREATE OR REPLACE FUNCTION audit_trigger_func()
RETURNS TRIGGER AS $$
DECLARE
    audit_row JSONB;
    old_data JSONB := NULL;
    new_data JSONB := NULL;
    action_type VARCHAR(50);
    entity_id_val VARCHAR(255);
BEGIN
    IF TG_OP = 'INSERT' THEN
        action_type := 'CREATE';
        new_data := to_jsonb(NEW);
        entity_id_val := NEW.id::TEXT;
    ELSIF TG_OP = 'UPDATE' THEN
        action_type := 'UPDATE';
        old_data := to_jsonb(OLD);
        new_data := to_jsonb(NEW);
        entity_id_val := NEW.id::TEXT;
    ELSIF TG_OP = 'DELETE' THEN
        action_type := 'DELETE';
        old_data := to_jsonb(OLD);
        entity_id_val := OLD.id::TEXT;
    END IF;

    INSERT INTO audit_log (
        tenant_id,
        user_id,
        portal_type,
        action,
        entity_type,
        entity_id,
        old_data,
        new_data,
        ip_address,
        created_at
    ) VALUES (
        COALESCE(NULLIF(current_setting('app.current_tenant_id', true), ''), '0')::BIGINT,
        COALESCE(NULLIF(current_setting('app.current_user_id', true), ''), '0')::BIGINT,
        COALESCE(current_setting('app.current_portal', true), 'system'),
        action_type,
        TG_TABLE_NAME,
        entity_id_val,
        old_data,
        new_data,
        NULLIF(current_setting('app.current_ip', true), '')::INET,
        NOW()
    );

    IF TG_OP = 'DELETE' THEN
        RETURN OLD;
    ELSE
        RETURN NEW;
    END IF;
END;
$$ LANGUAGE plpgsql;

-- Apply audit trigger to sensitive tables
CREATE TRIGGER trg_audit_users
    AFTER INSERT OR UPDATE OR DELETE ON users
    FOR EACH ROW EXECUTE FUNCTION audit_trigger_func();

CREATE TRIGGER trg_audit_subscriber_credentials
    AFTER INSERT OR UPDATE OR DELETE ON subscriber_credentials
    FOR EACH ROW EXECUTE FUNCTION audit_trigger_func();

CREATE TRIGGER trg_audit_branding_config
    AFTER INSERT OR UPDATE OR DELETE ON branding_config
    FOR EACH ROW EXECUTE FUNCTION audit_trigger_func();

CREATE TRIGGER trg_audit_tariff_sandbox
    AFTER INSERT OR UPDATE OR DELETE ON tariff_sandbox
    FOR EACH ROW EXECUTE FUNCTION audit_trigger_func();
