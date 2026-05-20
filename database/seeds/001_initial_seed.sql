-- UI-Bill Initial Seed Data
-- Creates: default languages, root role, root user, default tenant, default branding

BEGIN;

-- ============================================================
-- 1. LANGUAGES
-- ============================================================

INSERT INTO language (code, name, is_active, default_rtl) VALUES
    ('en', 'English', true, false),
    ('ru', 'Русский', true, false),
    ('es', 'Español', true, false)
ON CONFLICT (code) DO NOTHING;

-- ============================================================
-- 2. PERMISSIONS
-- ============================================================

INSERT INTO permissions (code, resource, action, description) VALUES
    -- Subscriber management
    ('subscriber:read', 'subscriber', 'read', 'View subscriber data'),
    ('subscriber:write', 'subscriber', 'write', 'Create and update subscribers'),
    ('subscriber:delete', 'subscriber', 'delete', 'Delete subscribers'),
    
    -- Balance management
    ('balance:read', 'balance', 'read', 'View balances'),
    ('balance:write', 'balance', 'write', 'Adjust balances'),
    ('balance:delete', 'balance', 'delete', 'Delete balance records'),
    
    -- Tariff management
    ('tariff:read', 'tariff', 'read', 'View tariff plans'),
    ('tariff:write', 'tariff', 'write', 'Create and update tariffs'),
    ('tariff:delete', 'tariff', 'delete', 'Delete tariff plans'),
    
    -- CDR and reports
    ('cdr:read', 'cdr', 'read', 'View CDR records'),
    ('cdr:export', 'cdr', 'execute', 'Export CDR data'),
    ('report:read', 'report', 'read', 'View reports'),
    
    -- Sessions
    ('session:read', 'session', 'read', 'View active sessions'),
    ('session:terminate', 'session', 'execute', 'Terminate sessions'),
    
    -- System management (Admin only)
    ('system:config', 'system', 'execute', 'Change system configuration'),
    ('system:monitor', 'system', 'read', 'View system metrics'),
    
    -- Tenant management
    ('tenant:read', 'tenant', 'read', 'View tenants'),
    ('tenant:create', 'tenant', 'write', 'Create tenants'),
    ('tenant:update', 'tenant', 'write', 'Update tenants'),
    ('tenant:delete', 'tenant', 'delete', 'Delete tenants'),
    
    -- User management
    ('user:read', 'user', 'read', 'View users'),
    ('user:create', 'user', 'write', 'Create users'),
    ('user:update', 'user', 'write', 'Update users'),
    ('user:delete', 'user', 'delete', 'Delete users'),
    ('user:reset_mfa', 'user', 'execute', 'Reset MFA for users'),
    
    -- Branding / White-label
    ('branding:read', 'branding', 'read', 'View branding settings'),
    ('branding:manage', 'branding', 'write', 'Manage branding'),
    
    -- Localization
    ('translation:read', 'translation', 'read', 'View translations'),
    ('translation:manage', 'translation', 'write', 'Manage translations'),
    
    -- API Keys
    ('api_key:read', 'api_key', 'read', 'View API keys'),
    ('api_key:manage', 'api_key', 'write', 'Manage API keys')
ON CONFLICT (code) DO NOTHING;

-- ============================================================
-- 3. ROLES
-- ============================================================

INSERT INTO roles (code, name, permissions, is_system) VALUES
    ('root', 'System Root', '["subscriber:read","subscriber:write","subscriber:delete","balance:read","balance:write","balance:delete","tariff:read","tariff:write","tariff:delete","cdr:read","cdr:export","report:read","session:read","session:terminate","system:config","system:monitor","tenant:read","tenant:create","tenant:update","tenant:delete","user:read","user:create","user:update","user:delete","user:reset_mfa","branding:read","branding:manage","translation:read","translation:manage","api_key:read","api_key:manage"]', true),
    
    ('reseller', 'Reseller', '["subscriber:read","subscriber:write","balance:read","balance:write","tariff:read","tariff:write","cdr:read","cdr:export","report:read","session:read","user:read","user:create","user:update","branding:read","branding:manage","translation:read","translation:manage"]', true),
    
    ('admin', 'Tenant Admin', '["subscriber:read","subscriber:write","subscriber:delete","balance:read","balance:write","tariff:read","tariff:write","tariff:delete","cdr:read","cdr:export","report:read","session:read","session:terminate","user:read","user:create","user:update","user:delete","branding:read","translation:read"]', true),
    
    ('support', 'Support Agent', '["subscriber:read","balance:read","cdr:read","report:read","session:read"]', true),
    
    ('viewer', 'Read-Only Viewer', '["subscriber:read","balance:read","cdr:read","report:read"]', true)
ON CONFLICT (code) DO NOTHING;

-- ============================================================
-- 4. DEFAULT TENANT
-- ============================================================

INSERT INTO tenant (name, code, is_active, created_at)
VALUES ('System', 'system', true, NOW())
ON CONFLICT (code) DO NOTHING;

-- ============================================================
-- 5. ROOT USER
-- ============================================================
-- NOTE: Password must be changed after first login
-- Default password hash below is for 'ChangeMe123!'
-- Generated with bcrypt cost 12

INSERT INTO users (
    tenant_id,
    email,
    password_hash,
    role_id,
    locale,
    is_active,
    created_at
) VALUES (
    (SELECT id FROM tenant WHERE code = 'system'),
    'root@cgrates.local',
    '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewKyNiAYMyzJ/I1m',
    (SELECT id FROM roles WHERE code = 'root'),
    'en',
    true,
    NOW()
)
ON CONFLICT (tenant_id, email) DO NOTHING;

-- ============================================================
-- 6. DEFAULT BRANDING
-- ============================================================

INSERT INTO branding_config (
    tenant_id,
    product_name,
    support_email,
    timezone,
    date_format,
    first_day_of_week,
    currency_symbol
) VALUES (
    (SELECT id FROM tenant WHERE code = 'system'),
    'CGRateS Default',
    'support@cgrates.local',
    'UTC',
    'DD.MM.YYYY',
    1,
    '$'
)
ON CONFLICT (tenant_id) DO NOTHING;

-- ============================================================
-- 7. DEFAULT EMAIL TEMPLATES (English)
-- ============================================================

INSERT INTO email_template (
    tenant_id,
    language_id,
    template_type,
    subject_template,
    body_html_template
) VALUES
(
    (SELECT id FROM tenant WHERE code = 'system'),
    (SELECT id FROM language WHERE code = 'en'),
    'welcome',
    'Welcome to {{product_name}}!',
    '<html><body><h1>Welcome!</h1><p>Your account has been created.</p><p>Product: {{product_name}}</p></body></html>'
),
(
    (SELECT id FROM tenant WHERE code = 'system'),
    (SELECT id FROM language WHERE code = 'en'),
    'password_reset',
    'Password Reset Request',
    '<html><body><h1>Password Reset</h1><p>Click <a href="{{reset_link}}">here</a> to reset your password.</p></body></html>'
),
(
    (SELECT id FROM tenant WHERE code = 'system'),
    (SELECT id FROM language WHERE code = 'en'),
    'mfa_backup_codes',
    'Your MFA Backup Codes',
    '<html><body><h1>MFA Backup Codes</h1><p>Save these codes securely: {{backup_codes}}</p></body></html>'
)
ON CONFLICT (tenant_id, language_id, template_type) DO NOTHING;

-- ============================================================
-- 8. DEFAULT TRANSLATIONS (English)
-- ============================================================

INSERT INTO translation (language_id, key, value, category) VALUES
    ((SELECT id FROM language WHERE code = 'en'), 'app.title', 'CGRateS Billing', 'common'),
    ((SELECT id FROM language WHERE code = 'en'), 'buttons.save', 'Save', 'buttons'),
    ((SELECT id FROM language WHERE code = 'en'), 'buttons.cancel', 'Cancel', 'buttons'),
    ((SELECT id FROM language WHERE code = 'en'), 'buttons.login', 'Login', 'buttons'),
    ((SELECT id FROM language WHERE code = 'en'), 'errors.invalid_credentials', 'Invalid credentials', 'errors'),
    ((SELECT id FROM language WHERE code = 'en'), 'errors.insufficient_funds', 'Insufficient funds', 'errors'),
    ((SELECT id FROM language WHERE code = 'en'), 'errors.invalid_msisdn', 'Invalid phone number', 'errors'),
    ((SELECT id FROM language WHERE code = 'en'), 'errors.rate_limit_exceeded', 'Too many requests. Please try again later.', 'errors'),
    ((SELECT id FROM language WHERE code = 'en'), 'balance.monetary', 'Balance', 'balance'),
    ((SELECT id FROM language WHERE code = 'en'), 'balance.data', 'Internet', 'balance'),
    ((SELECT id FROM language WHERE code = 'en'), 'balance.voice', 'Voice', 'balance'),
    ((SELECT id FROM language WHERE code = 'en'), 'balance.sms', 'SMS', 'balance')
ON CONFLICT (language_id, key, category) DO NOTHING;

-- ============================================================
-- 9. DEFAULT TRANSLATIONS (Russian)
-- ============================================================

INSERT INTO translation (language_id, key, value, category) VALUES
    ((SELECT id FROM language WHERE code = 'ru'), 'app.title', 'CGRateS Биллинг', 'common'),
    ((SELECT id FROM language WHERE code = 'ru'), 'buttons.save', 'Сохранить', 'buttons'),
    ((SELECT id FROM language WHERE code = 'ru'), 'buttons.cancel', 'Отмена', 'buttons'),
    ((SELECT id FROM language WHERE code = 'ru'), 'buttons.login', 'Войти', 'buttons'),
    ((SELECT id FROM language WHERE code = 'ru'), 'errors.invalid_credentials', 'Неверные учетные данные', 'errors'),
    ((SELECT id FROM language WHERE code = 'ru'), 'errors.insufficient_funds', 'Недостаточно средств', 'errors'),
    ((SELECT id FROM language WHERE code = 'ru'), 'errors.invalid_msisdn', 'Неверный номер телефона', 'errors'),
    ((SELECT id FROM language WHERE code = 'ru'), 'errors.rate_limit_exceeded', 'Слишком много запросов. Попробуйте позже.', 'errors'),
    ((SELECT id FROM language WHERE code = 'ru'), 'balance.monetary', 'Баланс', 'balance'),
    ((SELECT id FROM language WHERE code = 'ru'), 'balance.data', 'Интернет', 'balance'),
    ((SELECT id FROM language WHERE code = 'ru'), 'balance.voice', 'Звонки', 'balance'),
    ((SELECT id FROM language WHERE code = 'ru'), 'balance.sms', 'SMS', 'balance')
ON CONFLICT (language_id, key, category) DO NOTHING;

-- ============================================================
-- 10. DEFAULT TRANSLATIONS (Spanish)
-- ============================================================

INSERT INTO translation (language_id, key, value, category) VALUES
    ((SELECT id FROM language WHERE code = 'es'), 'app.title', 'CGRateS Facturación', 'common'),
    ((SELECT id FROM language WHERE code = 'es'), 'buttons.save', 'Guardar', 'buttons'),
    ((SELECT id FROM language WHERE code = 'es'), 'buttons.cancel', 'Cancelar', 'buttons'),
    ((SELECT id FROM language WHERE code = 'es'), 'buttons.login', 'Iniciar sesión', 'buttons'),
    ((SELECT id FROM language WHERE code = 'es'), 'errors.invalid_credentials', 'Credenciales inválidas', 'errors'),
    ((SELECT id FROM language WHERE code = 'es'), 'errors.insufficient_funds', 'Fondos insuficientes', 'errors'),
    ((SELECT id FROM language WHERE code = 'es'), 'errors.invalid_msisdn', 'Número de teléfono inválido', 'errors'),
    ((SELECT id FROM language WHERE code = 'es'), 'errors.rate_limit_exceeded', 'Demasiadas solicitudes. Intente más tarde.', 'errors'),
    ((SELECT id FROM language WHERE code = 'es'), 'balance.monetary', 'Saldo', 'balance'),
    ((SELECT id FROM language WHERE code = 'es'), 'balance.data', 'Internet', 'balance'),
    ((SELECT id FROM language WHERE code = 'es'), 'balance.voice', 'Voz', 'balance'),
    ((SELECT id FROM language WHERE code = 'es'), 'balance.sms', 'SMS', 'balance')
ON CONFLICT (language_id, key, category) DO NOTHING;

COMMIT;
