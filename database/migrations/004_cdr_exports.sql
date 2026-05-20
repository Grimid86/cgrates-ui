-- CDR export tracking table
CREATE TABLE IF NOT EXISTS cdr_exports (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id BIGINT NOT NULL REFERENCES tenant(id) ON DELETE CASCADE,
    created_by BIGINT NOT NULL REFERENCES users(id),
    filter JSONB DEFAULT '{}',
    format VARCHAR(10) NOT NULL DEFAULT 'csv',
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    object_key VARCHAR(255),
    download_url VARCHAR(512),
    record_count INT DEFAULT 0,
    error_message TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW() NOT NULL,
    completed_at TIMESTAMPTZ,
    CONSTRAINT chk_cdr_export_format CHECK (format IN ('csv', 'json')),
    CONSTRAINT chk_cdr_export_status CHECK (status IN ('pending', 'processing', 'completed', 'failed'))
);

CREATE INDEX IF NOT EXISTS idx_cdr_exports_tenant ON cdr_exports(tenant_id);
CREATE INDEX IF NOT EXISTS idx_cdr_exports_status ON cdr_exports(status);
