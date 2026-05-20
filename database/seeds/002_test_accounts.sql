-- UI-Bill Test Accounts Seed
-- Password for all staff accounts: TestPass123!
-- PIN for subscriber: 1234

BEGIN;

-- ============================================================
-- 1. TEST TENANT
-- ============================================================

INSERT INTO tenant (name, code, is_active, created_at)
VALUES ('Demo Telecom', 'demo', true, NOW())
ON CONFLICT (code) DO NOTHING;

-- ============================================================
-- 2. TEST STAFF USERS (Operator / Admin portals)
-- ============================================================

INSERT INTO users (
    tenant_id, email, password_hash, role_id, locale, is_active, created_at
) VALUES
    -- Admin portal users
    (
        (SELECT id FROM tenant WHERE code = 'system'),
        'admin@test.com',
        '$2b$12$Gh0.unEpktse8cAeX3jvBODgAx.0Wu.n1/JvQbugmGApOAt2ZOGbm',
        (SELECT id FROM roles WHERE code = 'admin'),
        'en',
        true,
        NOW()
    ),
    (
        (SELECT id FROM tenant WHERE code = 'system'),
        'reseller@test.com',
        '$2b$12$Gh0.unEpktse8cAeX3jvBODgAx.0Wu.n1/JvQbugmGApOAt2ZOGbm',
        (SELECT id FROM roles WHERE code = 'reseller'),
        'en',
        true,
        NOW()
    ),
    (
        (SELECT id FROM tenant WHERE code = 'system'),
        'support@test.com',
        '$2b$12$Gh0.unEpktse8cAeX3jvBODgAx.0Wu.n1/JvQbugmGApOAt2ZOGbm',
        (SELECT id FROM roles WHERE code = 'support'),
        'en',
        true,
        NOW()
    ),
    (
        (SELECT id FROM tenant WHERE code = 'system'),
        'viewer@test.com',
        '$2b$12$Gh0.unEpktse8cAeX3jvBODgAx.0Wu.n1/JvQbugmGApOAt2ZOGbm',
        (SELECT id FROM roles WHERE code = 'viewer'),
        'en',
        true,
        NOW()
    )
ON CONFLICT (tenant_id, email) DO NOTHING;

-- ============================================================
-- 3. TEST SUBSCRIBER (SelfCare portal)
-- ============================================================

INSERT INTO subscriber_credentials (
    tenant_id, msisdn, imsi, pin_hash, email, category, is_active, created_at
) VALUES
    (
        (SELECT id FROM tenant WHERE code = 'demo'),
        '79161234567',
        '250011234567890',
        '$2b$12$7thMLbg.cZO8v5SV7rqonOmXhSSdSw31/jZSyqSMxV7KDAiLGb9.6',
        'subscriber@demo.com',
        'prepaid',
        true,
        NOW()
    ),
    (
        (SELECT id FROM tenant WHERE code = 'demo'),
        '79169876543',
        '250019876543210',
        '$2b$12$7thMLbg.cZO8v5SV7rqonOmXhSSdSw31/jZSyqSMxV7KDAiLGb9.6',
        'subscriber2@demo.com',
        'postpaid',
        true,
        NOW()
    )
ON CONFLICT (msisdn) DO NOTHING;

COMMIT;
