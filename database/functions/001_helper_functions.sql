-- Helper SQL functions for application logic

-- Function to get subscriber by MSISDN with tenant context
CREATE OR REPLACE FUNCTION get_subscriber_by_msisdn(p_msisdn VARCHAR)
RETURNS TABLE (
    id BIGINT,
    tenant_id BIGINT,
    msisdn VARCHAR,
    imsi VARCHAR,
    email VARCHAR,
    category VARCHAR,
    is_active BOOLEAN,
    pin_hash VARCHAR,
    failed_login_attempts INT,
    locked_until TIMESTAMPTZ,
    last_login_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ
) AS $$
BEGIN
    RETURN QUERY
    SELECT s.id, s.tenant_id, s.msisdn, s.imsi, s.email, s.category,
           s.is_active, s.pin_hash, s.failed_login_attempts,
           s.locked_until, s.last_login_at, s.created_at
    FROM subscriber_credentials s
    WHERE s.msisdn = p_msisdn;
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

-- Function to paginate any query result
CREATE OR REPLACE FUNCTION paginate(
    p_page INT DEFAULT 1,
    p_per_page INT DEFAULT 20
)
RETURNS TABLE (
    page INT,
    per_page INT,
    offset_val INT
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        GREATEST(p_page, 1),
        LEAST(GREATEST(p_per_page, 1), 100),
        (GREATEST(p_page, 1) - 1) * LEAST(GREATEST(p_per_page, 1), 100);
END;
$$ LANGUAGE plpgsql;

-- Function to get active subscriber sessions
CREATE OR REPLACE FUNCTION get_active_subscriber_sessions(p_subscriber_id BIGINT)
RETURNS TABLE (
    id UUID,
    subscriber_id BIGINT,
    ip_address INET,
    user_agent TEXT,
    issued_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ
) AS $$
BEGIN
    RETURN QUERY
    SELECT s.id, s.subscriber_id, s.ip_address, s.user_agent, s.issued_at, s.expires_at
    FROM subscriber_sessions s
    WHERE s.subscriber_id = p_subscriber_id
      AND s.revoked_at IS NULL
      AND s.expires_at > NOW()
    ORDER BY s.issued_at DESC;
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

-- Function to create audit log entry (can be called from application)
CREATE OR REPLACE FUNCTION create_audit_log(
    p_tenant_id BIGINT,
    p_user_id BIGINT,
    p_portal_type VARCHAR,
    p_action VARCHAR,
    p_entity_type VARCHAR,
    p_entity_id VARCHAR,
    p_old_data JSONB,
    p_new_data JSONB,
    p_ip_address INET,
    p_user_agent TEXT
)
RETURNS BIGINT AS $$
DECLARE
    v_id BIGINT;
BEGIN
    INSERT INTO audit_log (
        tenant_id, user_id, portal_type, action, entity_type,
        entity_id, old_data, new_data, ip_address, user_agent
    ) VALUES (
        p_tenant_id, p_user_id, p_portal_type, p_action, p_entity_type,
        p_entity_id, p_old_data, p_new_data, p_ip_address, p_user_agent
    )
    RETURNING audit_log.id INTO v_id;
    
    RETURN v_id;
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

-- Function to check if idempotency key exists
CREATE OR REPLACE FUNCTION check_idempotency_key(
    p_key VARCHAR,
    p_portal_type VARCHAR
)
RETURNS JSONB AS $$
DECLARE
    v_result JSONB;
BEGIN
    SELECT response_data INTO v_result
    FROM rpc_idempotency_keys
    WHERE key = p_key AND portal_type = p_portal_type AND expires_at > NOW();
    
    RETURN v_result;
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

-- Function to store idempotency key
CREATE OR REPLACE FUNCTION store_idempotency_key(
    p_key VARCHAR,
    p_portal_type VARCHAR,
    p_request_hash VARCHAR,
    p_response_data JSONB,
    p_expires_at TIMESTAMPTZ
)
RETURNS VOID AS $$
BEGIN
    INSERT INTO rpc_idempotency_keys (key, portal_type, request_hash, response_data, expires_at)
    VALUES (p_key, p_portal_type, p_request_hash, p_response_data, p_expires_at)
    ON CONFLICT (key, portal_type) DO UPDATE SET
        request_hash = EXCLUDED.request_hash,
        response_data = EXCLUDED.response_data,
        expires_at = EXCLUDED.expires_at;
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

-- Function to clean expired idempotency keys (run via cron)
CREATE OR REPLACE FUNCTION cleanup_expired_idempotency_keys()
RETURNS INT AS $$
DECLARE
    v_count INT;
BEGIN
    DELETE FROM rpc_idempotency_keys WHERE expires_at < NOW();
    GET DIAGNOSTICS v_count = ROW_COUNT;
    RETURN v_count;
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;
