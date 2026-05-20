# UI-Bill Database Schema
## PostgreSQL 15+ — Partitioning, Audit, i18n, White-Label

---

## 1. Core Tables

### 1.1. `tenant`

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `id` | BIGSERIAL | PRIMARY KEY | Internal ID |
| `uuid` | UUID | UNIQUE, DEFAULT gen_random_uuid() | External ID |
| `name` | VARCHAR(100) | NOT NULL | Tenant name |
| `code` | VARCHAR(50) | UNIQUE, NOT NULL | URL-safe code |
| `is_active` | BOOLEAN | DEFAULT true | Status |
| `created_at` | TIMESTAMPTZ | DEFAULT NOW() | Creation time |
| `updated_at` | TIMESTAMPTZ | DEFAULT NOW() | Update time |

**Indexes:**
- `idx_tenant_uuid` (UUID) — for external lookups
- `idx_tenant_code` (code) — for subdomain resolution
- `idx_tenant_active` (is_active) — filtered queries

---

### 1.2. `users` (Staff users: Admin, Operator, Support, Viewer)

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `id` | BIGSERIAL | PRIMARY KEY | Internal ID |
| `uuid` | UUID | UNIQUE, DEFAULT gen_random_uuid() | External ID |
| `tenant_id` | BIGINT | NOT NULL, FK → tenant(id) ON DELETE CASCADE | Tenant |
| `email` | VARCHAR(255) | UNIQUE, NOT NULL | Login email |
| `password_hash` | VARCHAR(255) | NOT NULL | bcrypt hash |
| `role_id` | BIGINT | NOT NULL, FK → roles(id) ON DELETE RESTRICT | Role |
| `locale` | VARCHAR(5) | DEFAULT 'en' | Language code |
| `mfa_secret` | VARCHAR(255) | | TOTP secret (encrypted) |
| `mfa_enabled` | BOOLEAN | DEFAULT false | MFA status |
| `mfa_backup_codes` | JSONB | | Array of hashed backup codes |
| `is_active` | BOOLEAN | DEFAULT true | Status |
| `last_login_at` | TIMESTAMPTZ | | Last login |
| `failed_login_attempts` | INT | DEFAULT 0 | Counter |
| `locked_until` | TIMESTAMPTZ | | Lockout time |
| `created_at` | TIMESTAMPTZ | DEFAULT NOW() | Creation |
| `updated_at` | TIMESTAMPTZ | DEFAULT NOW() | Update |

**Constraints:**
- `CHECK (email ~* '^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$')`
- `CHECK (locale ~ '^[a-z]{2}(-[A-Z]{2})?$')`

**Indexes:**
- `idx_users_tenant_email` (tenant_id, email)
- `idx_users_uuid` (uuid)
- `idx_users_role` (role_id)

---

### 1.3. `roles`

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `id` | BIGSERIAL | PRIMARY KEY | Internal ID |
| `code` | VARCHAR(50) | UNIQUE, NOT NULL | Role code: root, reseller, admin, support, viewer |
| `name` | VARCHAR(100) | NOT NULL | Display name |
| `permissions` | JSONB | NOT NULL, DEFAULT '[]' | Array of permission strings |
| `is_system` | BOOLEAN | DEFAULT false | Protected from deletion |
| `created_at` | TIMESTAMPTZ | DEFAULT NOW() | Creation |

**Indexes:**
- `idx_roles_code` (code) — unique lookup

---

### 1.4. `permissions` (Permission registry)

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `id` | BIGSERIAL | PRIMARY KEY | Internal ID |
| `code` | VARCHAR(100) | UNIQUE, NOT NULL | Permission code: `subscriber:read`, `tariff:write` |
| `resource` | VARCHAR(50) | NOT NULL | Resource name |
| `action` | VARCHAR(20) | NOT NULL | Action: read, write, delete, execute |
| `description` | TEXT | | Human readable |
| `created_at` | TIMESTAMPTZ | DEFAULT NOW() | Creation |

---

### 1.5. `audit_log` (Partitioned by MONTH)

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `id` | BIGSERIAL | | Auto-increment |
| `tenant_id` | BIGINT | NOT NULL | Tenant |
| `user_id` | BIGINT | | Staff user (NULL for system) |
| `portal_type` | VARCHAR(20) | CHECK IN ('selfcare', 'operator', 'admin') | Portal |
| `action` | VARCHAR(50) | NOT NULL | Action code |
| `entity_type` | VARCHAR(50) | | Entity: subscriber, tariff, balance |
| `entity_id` | VARCHAR(255) | | Entity identifier |
| `old_data` | JSONB | | Previous state |
| `new_data` | JSONB | | New state |
| `ip_address` | INET | | Client IP |
| `user_agent` | TEXT | | Client UA |
| `request_id` | UUID | | Trace ID |
| `created_at` | TIMESTAMPTZ | NOT NULL, DEFAULT NOW() | Timestamp |

**Partitioning:**
```sql
CREATE TABLE audit_log (
    id BIGSERIAL,
    tenant_id BIGINT NOT NULL,
    user_id BIGINT,
    portal_type VARCHAR(20) NOT NULL CHECK (portal_type IN ('selfcare','operator','admin')),
    action VARCHAR(50) NOT NULL,
    entity_type VARCHAR(50),
    entity_id VARCHAR(255),
    old_data JSONB,
    new_data JSONB,
    ip_address INET,
    user_agent TEXT,
    request_id UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (id, created_at)
) PARTITION BY RANGE (created_at);
```

**Partition creation (automated via cron/pg_partman):**
```sql
CREATE TABLE audit_log_y2024m01 PARTITION OF audit_log
    FOR VALUES FROM ('2024-01-01') TO ('2024-02-01');
```

**Indexes (per partition):**
- `idx_audit_tenant_created` (tenant_id, created_at)
- `idx_audit_action` (action)
- `idx_audit_entity` (entity_type, entity_id)
- `idx_audit_request_id` (request_id)
- GIN index on `old_data` and `new_data`

---

## 2. Localization Tables (i18n)

### 2.1. `language`

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `id` | BIGSERIAL | PRIMARY KEY | Internal ID |
| `code` | VARCHAR(5) | UNIQUE, NOT NULL | Language code: en, ru, es, tr |
| `name` | VARCHAR(50) | NOT NULL | Display name |
| `is_active` | BOOLEAN | DEFAULT true | Available for selection |
| `default_rtl` | BOOLEAN | DEFAULT false | RTL direction |
| `created_at` | TIMESTAMPTZ | DEFAULT NOW() | Creation |

**Constraints:**
- `CHECK (code ~ '^[a-z]{2}(-[A-Z]{2})?$')`

---

### 2.2. `translation`

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `id` | BIGSERIAL | PRIMARY KEY | Internal ID |
| `language_id` | BIGINT | NOT NULL, FK → language(id) ON DELETE CASCADE | Language |
| `key` | VARCHAR(255) | NOT NULL | Translation key: `balance.monetary` |
| `value` | TEXT | NOT NULL | Translated text |
| `category` | VARCHAR(50) | DEFAULT 'common' | Grouping: common, errors, buttons |
| `updated_at` | TIMESTAMPTZ | DEFAULT NOW() | Update time |

**Constraints:**
- `UNIQUE(language_id, key, category)`

**Indexes:**
- `idx_translation_lookup` (language_id, key, category)
- `idx_translation_category` (category)

**Cache strategy:**
- All translations cached in Redis: `i18n:{locale}:{category}` (TTL 1 hour)
- Invalidation on UPDATE/DELETE via Pulsar `config.changes`

---

## 3. White-Label Tables

### 3.1. `branding_config`

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `id` | BIGSERIAL | PRIMARY KEY | Internal ID |
| `tenant_id` | BIGINT | NOT NULL, FK → tenant(id) ON DELETE CASCADE | Tenant |
| `primary_color` | VARCHAR(7) | DEFAULT '#007bff' | Primary brand color |
| `secondary_color` | VARCHAR(7) | DEFAULT '#6c757d' | Secondary color |
| `accent_color` | VARCHAR(7) | DEFAULT '#28a745' | Accent color |
| `danger_color` | VARCHAR(7) | DEFAULT '#dc3545' | Danger/error color |
| `logo_url` | TEXT | | Logo URL |
| `favicon_url` | TEXT | | Favicon URL |
| `login_background_url` | TEXT | | Login background |
| `product_name` | VARCHAR(100) | DEFAULT 'CGRateS Billing' | Product name |
| `support_email` | VARCHAR(255) | | Support email |
| `support_phone` | VARCHAR(50) | | Support phone |
| `support_telegram` | VARCHAR(100) | | Telegram support |
| `timezone` | VARCHAR(50) | DEFAULT 'UTC' | Default timezone |
| `date_format` | VARCHAR(20) | DEFAULT 'DD.MM.YYYY' | Date format |
| `first_day_of_week` | INT | DEFAULT 1 | 0=Sunday, 1=Monday |
| `currency_symbol` | VARCHAR(10) | DEFAULT '$' | Currency |
| `updated_at` | TIMESTAMPTZ | DEFAULT NOW() | Update time |

**Constraints:**
- `UNIQUE(tenant_id)`
- `CHECK (primary_color ~ '^#[0-9A-Fa-f]{6}$')` (same for other colors)

**Cache strategy:**
- Cached in Redis: `branding:{tenant_id}` (TTL 5 minutes)
- Invalidation on UPDATE via Pulsar `config.changes`

---

### 3.2. `email_template`

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `id` | BIGSERIAL | PRIMARY KEY | Internal ID |
| `tenant_id` | BIGINT | NOT NULL, FK → tenant(id) ON DELETE CASCADE | Tenant |
| `language_id` | BIGINT | FK → language(id) ON DELETE SET NULL | Language |
| `template_type` | VARCHAR(50) | NOT NULL | Type: welcome, reset_password, invoice |
| `subject_template` | VARCHAR(255) | NOT NULL | Subject with placeholders |
| `body_html_template` | TEXT | NOT NULL | HTML body with template syntax |
| `created_at` | TIMESTAMPTZ | DEFAULT NOW() | Creation |
| `updated_at` | TIMESTAMPTZ | DEFAULT NOW() | Update |

**Constraints:**
- `UNIQUE(tenant_id, language_id, template_type)`

---

### 3.3. `domain_tenant_mapping`

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `id` | BIGSERIAL | PRIMARY KEY | Internal ID |
| `domain` | VARCHAR(255) | UNIQUE, NOT NULL | Domain: tenant1.billing.com |
| `tenant_id` | BIGINT | NOT NULL, FK → tenant(id) ON DELETE CASCADE | Tenant |
| `is_active` | BOOLEAN | DEFAULT true | Status |
| `ssl_cert_id` | VARCHAR(100) | | cert-manager certificate name |
| `created_at` | TIMESTAMPTZ | DEFAULT NOW() | Creation |

**Indexes:**
- `idx_domain_mapping_lookup` (domain, is_active)

---

## 4. Multi-Portal Tables

### 4.1. `subscriber_credentials` (SelfCare only)

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `id` | BIGSERIAL | PRIMARY KEY | Internal ID |
| `tenant_id` | BIGINT | FK → tenant(id) ON DELETE CASCADE | Tenant |
| `msisdn` | VARCHAR(20) | UNIQUE, NOT NULL | Phone number |
| `imsi` | VARCHAR(15) | | SIM identifier |
| `pin_hash` | VARCHAR(255) | | PIN code hash |
| `password_hash` | VARCHAR(255) | | Password hash |
| `email` | VARCHAR(255) | | Email |
| `category` | VARCHAR(20) | DEFAULT 'prepaid' | prepaid, postpaid, enterprise |
| `is_active` | BOOLEAN | DEFAULT true | Status |
| `failed_login_attempts` | INT | DEFAULT 0 | Counter |
| `locked_until` | TIMESTAMPTZ | | Lockout |
| `last_login_at` | TIMESTAMPTZ | | Last login |
| `created_at` | TIMESTAMPTZ | DEFAULT NOW() | Creation |

**Constraints:**
- `CHECK (msisdn ~ '^[0-9]+$')`
- `CHECK (category IN ('prepaid', 'postpaid', 'enterprise'))`

**Indexes:**
- `idx_subscriber_msisdn` (msisdn)
- `idx_subscriber_tenant` (tenant_id)
- `idx_subscriber_imsi` (imsi)

---

### 4.2. `subscriber_sessions`

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `id` | UUID | PRIMARY KEY, DEFAULT gen_random_uuid() | Session ID |
| `subscriber_id` | BIGINT | FK → subscriber_credentials(id) ON DELETE CASCADE | Subscriber |
| `token_hash` | VARCHAR(255) | NOT NULL | Refresh token hash |
| `ip_address` | INET | | Client IP |
| `user_agent` | TEXT | | Client UA |
| `issued_at` | TIMESTAMPTZ | DEFAULT NOW() | Issue time |
| `expires_at` | TIMESTAMPTZ | NOT NULL | Expiration |
| `revoked_at` | TIMESTAMPTZ | | Revocation |

**Indexes:**
- `idx_sub_session_subscriber` (subscriber_id)
- `idx_sub_session_expires` (expires_at)

---

### 4.3. `portal_sessions` (Staff users)

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `id` | UUID | PRIMARY KEY, DEFAULT gen_random_uuid() | Session ID |
| `user_id` | BIGINT | FK → users(id) ON DELETE CASCADE | User |
| `portal_type` | VARCHAR(20) | NOT NULL CHECK IN ('admin','operator','support') | Portal |
| `token_hash` | VARCHAR(255) | NOT NULL | Refresh token hash |
| `ip_address` | INET | | Client IP |
| `mfa_verified` | BOOLEAN | DEFAULT false | MFA passed |
| `issued_at` | TIMESTAMPTZ | DEFAULT NOW() | Issue time |
| `expires_at` | TIMESTAMPTZ | NOT NULL | Expiration |
| `revoked_at` | TIMESTAMPTZ | | Revocation |

**Indexes:**
- `idx_portal_session_user` (user_id, portal_type)
- `idx_portal_session_expires` (expires_at)

---

### 4.4. `api_keys`

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `id` | BIGSERIAL | PRIMARY KEY | Internal ID |
| `tenant_id` | BIGINT | FK → tenant(id) ON DELETE CASCADE | Tenant |
| `name` | VARCHAR(100) | | Key name |
| `key_hash` | VARCHAR(255) | UNIQUE, NOT NULL | API key hash |
| `portal_type` | VARCHAR(20) | NOT NULL CHECK IN ('selfcare','operator','admin') | Portal |
| `scopes` | JSONB | NOT NULL | Allowed scopes |
| `rate_limit_rps` | INT | DEFAULT 10 | Rate limit |
| `expires_at` | TIMESTAMPTZ | | Expiration |
| `last_used_at` | TIMESTAMPTZ | | Last usage |
| `created_at` | TIMESTAMPTZ | DEFAULT NOW() | Creation |

**Indexes:**
- `idx_api_key_hash` (key_hash)
- `idx_api_key_tenant` (tenant_id)

---

## 5. Auxiliary Tables

### 5.1. `tariff_sandbox`

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `id` | BIGSERIAL | PRIMARY KEY | Internal ID |
| `tenant_id` | BIGINT | NOT NULL, FK → tenant(id) ON DELETE CASCADE | Tenant |
| `name` | VARCHAR(100) | NOT NULL | Sandbox name |
| `cgrates_tp_id` | VARCHAR(64) | | CGRateS TP identifier |
| `config_json` | JSONB | NOT NULL | Tariff configuration |
| `status` | VARCHAR(20) | DEFAULT 'draft' | draft, testing, active, archived |
| `created_by` | BIGINT | NOT NULL, FK → users(id) | Author |
| `created_at` | TIMESTAMPTZ | DEFAULT NOW() | Creation |
| `updated_at` | TIMESTAMPTZ | DEFAULT NOW() | Update |

---

### 5.2. `balance_history`

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `id` | BIGSERIAL | PRIMARY KEY | Internal ID |
| `tenant_id` | BIGINT | NOT NULL | Tenant |
| `subscriber_id` | BIGINT | NOT NULL | Subscriber |
| `balance_type` | VARCHAR(20) | NOT NULL | *monetary, *data, *sms, *voice |
| `amount_before` | DECIMAL(18,6) | | Previous amount |
| `amount_after` | DECIMAL(18,6) | | New amount |
| `operation` | VARCHAR(20) | NOT NULL | topup, charge, adjust, expire |
| `extra_data` | JSONB | | Additional context |
| `created_at` | TIMESTAMPTZ | DEFAULT NOW() | Creation |

**Constraints:**
- `CHECK (balance_type IN ('*monetary', '*data', '*sms', '*voice', '*bonus'))`

---

### 5.3. `action_trigger_presets`

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `id` | BIGSERIAL | PRIMARY KEY | Internal ID |
| `tenant_id` | BIGINT | NOT NULL | Tenant |
| `name` | VARCHAR(100) | NOT NULL | Preset name |
| `trigger_type` | VARCHAR(50) | NOT NULL | threshold, expiration, event |
| `conditions` | JSONB | NOT NULL | Trigger conditions |
| `actions` | JSONB | NOT NULL | Actions to execute |
| `is_active` | BOOLEAN | DEFAULT true | Status |
| `created_at` | TIMESTAMPTZ | DEFAULT NOW() | Creation |

---

### 5.4. `diameter_peer_monitoring`

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `id` | BIGSERIAL | PRIMARY KEY | Internal ID |
| `tenant_id` | BIGINT | NOT NULL | Tenant |
| `peer_address` | VARCHAR(255) | NOT NULL | Diameter peer IP:port |
| `peer_type` | VARCHAR(20) | NOT NULL | gy, gx, s6a |
| `status` | VARCHAR(20) | NOT NULL | online, offline, degraded |
| `last_seen_at` | TIMESTAMPTZ | | Last heartbeat |
| `metadata` | JSONB | | Peer metadata |
| `created_at` | TIMESTAMPTZ | DEFAULT NOW() | Creation |

---

### 5.5. `rpc_idempotency_keys`

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `id` | BIGSERIAL | PRIMARY KEY | Internal ID |
| `key` | VARCHAR(255) | UNIQUE, NOT NULL | Idempotency key |
| `portal_type` | VARCHAR(20) | NOT NULL | Portal |
| `request_hash` | VARCHAR(255) | | Request signature |
| `response_data` | JSONB | | Cached response |
| `expires_at` | TIMESTAMPTZ | NOT NULL | TTL |
| `created_at` | TIMESTAMPTZ | DEFAULT NOW() | Creation |

**Indexes:**
- `idx_idempotency_key` (key, portal_type)
- `idx_idempotency_expires` (expires_at)

---

## 6. Partitioning Strategy

| Table | Partition Type | Key | Retention |
|-------|---------------|-----|-----------|
| `audit_log` | RANGE (monthly) | `created_at` | 12 months (archive to S3) |
| `balance_history` | RANGE (monthly) | `created_at` | 24 months |
| `cdr_hourly_stats` | RANGE (daily) | `date` | 90 days |
| `cdr_daily_stats` | RANGE (monthly) | `date` | 36 months |

---

## 7. Row-Level Security (RLS)

### 7.1. SelfCare CDR Access

```sql
ALTER TABLE cdr_records ENABLE ROW LEVEL SECURITY;

CREATE POLICY subscriber_cdr_isolation ON cdr_records
    FOR SELECT
    USING (tenant_id = current_setting('app.current_tenant_id')::BIGINT
           AND subscriber_msisdn = current_setting('app.current_msisdn'));
```

### 7.2. Tenant Isolation

```sql
ALTER TABLE users ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_user_isolation ON users
    FOR ALL
    USING (tenant_id = current_setting('app.current_tenant_id')::BIGINT
           OR current_setting('app.current_role') = 'root');
```

---

## 8. Database Users & Grants

| User | Purpose | Grants |
|------|---------|--------|
| `selfcare_rw` | SelfCare Gateway | SELECT on subscriber_credentials, balance_history, cdr_records; INSERT on subscriber_sessions |
| `operator_rw` | Operator Gateway | Full CRUD on subscriber tables, tariff tables; INSERT on audit_log |
| `admin_rw` | Admin Gateway | Full CRUD all tables; CREATE on schema |
| `worker_rw` | Workers | INSERT on audit_log, balance_history; SELECT all; UPDATE on materialized views |
| `replicator` | Streaming replication | REPLICATION, LOGIN |

---

## 9. Extensions Required

```sql
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";      -- For text search
CREATE EXTENSION IF NOT EXISTS "btree_gin";    -- For composite GIN indexes
```
