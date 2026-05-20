# UI-Bill API Contracts
## OpenAPI 3.0 Specifications for All Gateways

---

## 1. Common Standards

### 1.1. Base URLs

| Portal | Base URL | CORS Origin |
|--------|----------|-------------|
| SelfCare | `https://my.billing.company.com/api/v1` | `my.billing.company.com`, `*.tenant.com` |
| Operator | `https://bss.billing.company.com/api/v1` | `bss.billing.company.com` |
| Admin | `https://oss.billing.company.com/api/v1` | `oss.billing.company.com` |

### 1.2. Authentication

All endpoints (except explicitly marked) require `Authorization: Bearer {jwt}` header.

Admin endpoints additionally require `X-MFA-Code: {totp}` for critical operations.

### 1.3. Rate Limit Headers

```http
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1716192900
```

### 1.4. Common Error Response

```json
{
  "error": {
    "code": "INSUFFICIENT_FUNDS",
    "message": "Balance is insufficient for this operation",
    "details": {
      "required": 100.00,
      "available": 45.50,
      "currency": "USD"
    },
    "request_id": "550e8400-e29b-41d4-a716-446655440000",
    "timestamp": "2026-05-20T09:20:27Z"
  }
}
```

### 1.5. Idempotency

Write endpoints accept `Idempotency-Key: {uuid}` header. Response is cached for 24 hours.

---

## 2. SelfCare Gateway API

### 2.1. Authentication

#### POST `/auth/login`
**Description:** Subscriber login by MSISDN + PIN/Password

**Request:**
```json
{
  "msisdn": "79001234567",
  "pin": "1234",
  "captcha_token": "hcaptcha_token_here"
}
```

**Response 200:**
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIs...",
  "refresh_token": "eyJhbGciOiJIUzI1NiIs...",
  "expires_in": 900,
  "token_type": "Bearer",
  "subscriber": {
    "id": "uuid",
    "msisdn": "79001234567",
    "locale": "ru",
    "category": "prepaid"
  }
}
```

**Response 429:**
```json
{
  "error": {
    "code": "TOO_MANY_ATTEMPTS",
    "message": "Account locked for 15 minutes",
    "retry_after": 900
  }
}
```

#### POST `/auth/refresh`
**Description:** Refresh access token

**Request:**
```json
{
  "refresh_token": "eyJhbGciOiJIUzI1NiIs..."
}
```

#### POST `/auth/logout`
**Description:** Revoke current session

---

### 2.2. Balance

#### GET `/balance`
**Description:** Get current balances (monetary, data, voice, SMS)

**Response 200:**
```json
{
  "subscriber_id": "uuid",
  "balances": [
    {
      "type": "*monetary",
      "value": 150.50,
      "currency": "RUB",
      "expiry_date": null
    },
    {
      "type": "*data",
      "value": 5120.00,
      "unit": "MB",
      "expiry_date": "2026-06-20T00:00:00Z"
    },
    {
      "type": "*voice",
      "value": 300.00,
      "unit": "minutes",
      "expiry_date": "2026-06-20T00:00:00Z"
    }
  ],
  "last_updated": "2026-05-20T09:15:00Z"
}
```

---

### 2.3. CDR History

#### GET `/cdr`
**Description:** Call detail records with pagination

**Query Parameters:**
- `from` (date): Start date
- `to` (date): End date
- `type` (string): Filter by type — voice, data, sms
- `page` (int): Page number (default 1)
- `per_page` (int): Items per page (default 20, max 100)

**Response 200:**
```json
{
  "data": [
    {
      "id": "uuid",
      "type": "voice",
      "destination": "79009876543",
      "duration_seconds": 125,
      "cost": 12.50,
      "currency": "RUB",
      "started_at": "2026-05-19T14:30:00Z",
      "status": "completed"
    }
  ],
  "pagination": {
    "page": 1,
    "per_page": 20,
    "total": 145,
    "total_pages": 8
  }
}
```

---

### 2.4. Top-Up

#### POST `/topup`
**Description:** Account top-up (initiates payment)

**Headers:**
- `Idempotency-Key: uuid`

**Request:**
```json
{
  "amount": 500.00,
  "currency": "RUB",
  "payment_method": "card",
  "return_url": "https://my.billing.company.com/topup/success"
}
```

**Response 202:**
```json
{
  "status": "pending",
  "payment_url": "https://payment.provider.com/session/abc123",
  "transaction_id": "uuid",
  "expires_at": "2026-05-20T09:30:00Z"
}
```

---

### 2.5. Profile

#### GET `/profile`
**Response 200:**
```json
{
  "msisdn": "79001234567",
  "email": "user@example.com",
  "locale": "ru",
  "category": "prepaid",
  "tariff_plan": {
    "id": "uuid",
    "name": "Unlimited Plus",
    "description": "Unlimited calls and 20GB internet"
  },
  "notifications": {
    "email_enabled": true,
    "sms_enabled": false,
    "push_enabled": true
  }
}
```

#### PUT `/profile`
#### PUT `/profile/change-pin`
#### GET `/profile/sessions`
#### DELETE `/profile/sessions/{id}`

---

## 3. Operator Gateway API

### 3.1. Authentication

#### POST `/auth/login`
```json
{
  "email": "operator@company.com",
  "password": "secure_password",
  "mfa_code": "123456"
}
```

**Response 200:**
```json
{
  "access_token": "eyJ...",
  "refresh_token": "eyJ...",
  "expires_in": 900,
  "user": {
    "id": 100,
    "email": "operator@company.com",
    "role": "admin",
    "tenant_id": 1,
    "permissions": ["subscriber:read", "subscriber:write", "tariff:read"],
    "mfa_required": true
  }
}
```

---

### 3.2. Subscribers

#### GET `/subscribers`
**Query Parameters:**
- `search` (string): MSISDN, IMSI, or email
- `status` (string): active, blocked, suspended
- `tariff_id` (string): Filter by tariff
- `page`, `per_page`

**Response 200:**
```json
{
  "data": [
    {
      "id": "uuid",
      "msisdn": "79001234567",
      "imsi": "250011234567890",
      "status": "active",
      "tariff_plan": "Unlimited Plus",
      "balance_monetary": 150.50,
      "created_at": "2026-01-15T10:00:00Z",
      "last_activity": "2026-05-19T18:45:00Z"
    }
  ],
  "pagination": { ... }
}
```

#### GET `/subscribers/{id}`
#### POST `/subscribers`
#### PUT `/subscribers/{id}`
#### POST `/subscribers/{id}/block`
#### POST `/subscribers/{id}/unblock`
#### POST `/subscribers/{id}/migrate-tariff`

---

### 3.3. Balance Management

#### POST `/subscribers/{id}/balance/adjust`
**Headers:** `Idempotency-Key`, `X-MFA-Code` (for > $100)

**Request:**
```json
{
  "balance_type": "*monetary",
  "amount": 100.00,
  "operation": "credit",
  "reason": "Compensation for service outage #12345",
  "notify_subscriber": true
}
```

**Response 202:**
```json
{
  "transaction_id": "uuid",
  "status": "processing",
  "new_balance": 250.50
}
```

#### POST `/subscribers/{id}/balance/freeze`
#### POST `/subscribers/{id}/balance/unfreeze`

---

### 3.4. Tariffs

#### GET `/tariffs`
#### GET `/tariffs/{id}`
#### POST `/tariffs`
#### PUT `/tariffs/{id}`
#### DELETE `/tariffs/{id}`
#### POST `/tariffs/{id}/activate`

---

### 3.5. Bulk Operations

#### POST `/bulk/tariff-change`
**Request:**
```json
{
  "subscriber_ids": ["uuid1", "uuid2", "uuid3"],
  "new_tariff_id": "uuid",
  "effective_date": "2026-06-01T00:00:00Z",
  "notify_subscribers": true
}
```

**Response 202:**
```json
{
  "batch_id": "uuid",
  "status": "queued",
  "total_count": 3,
  "estimated_completion": "2026-05-20T09:25:00Z"
}
```

---

### 3.6. CDR & Reports

#### GET `/cdr`
#### GET `/cdr/export`
**Query Parameters:** `format` (csv, xlsx), `from`, `to`

**Response 202:**
```json
{
  "export_id": "uuid",
  "status": "processing",
  "download_url": null
}
```

#### GET `/reports/usage`
#### GET `/reports/revenue`

---

### 3.7. Sessions

#### GET `/sessions/active`
**Response 200:**
```json
{
  "data": [
    {
      "subscriber_msisdn": "79001234567",
      "session_id": "uuid",
      "type": "data",
      "started_at": "2026-05-20T09:15:00Z",
      "usage_mb": 45.2,
      "current_tariff": "Unlimited Plus",
      "diameter_peer": "10.0.1.15:3868"
    }
  ],
  "total_active": 1247
}
```

---

## 4. Admin Gateway API

### 4.1. Authentication

#### POST `/auth/login`
```json
{
  "email": "admin@system.local",
  "password": "very_secure_password",
  "mfa_code": "123456"
}
```

**Response 200:**
```json
{
  "access_token": "eyJ...",
  "refresh_token": "eyJ...",
  "expires_in": 900,
  "mfa_required": true,
  "user": {
    "id": 1,
    "email": "admin@system.local",
    "role": "root",
    "permissions": ["system:config", "tenant:create", "user:create"]
  }
}
```

#### POST `/auth/mfa/setup`
#### POST `/auth/mfa/verify`
#### POST `/auth/mfa/disable`

---

### 4.2. Tenants

#### GET `/tenants`
#### GET `/tenants/{id}`
#### POST `/tenants`
**Request:**
```json
{
  "name": "Orange Telecom",
  "code": "orange_telecom",
  "limits": {
    "max_subscribers": 100000,
    "max_staff_users": 50,
    "api_rate_limit_rps": 100
  }
}
```

#### PUT `/tenants/{id}`
#### POST `/tenants/{id}/suspend`
#### DELETE `/tenants/{id}`

---

### 4.3. Staff Users

#### GET `/users`
#### GET `/users/{id}`
#### POST `/users`
#### PUT `/users/{id}`
#### POST `/users/{id}/reset-mfa`
#### POST `/users/{id}/reset-password`
#### DELETE `/users/{id}`

---

### 4.4. RBAC

#### GET `/rbac/roles`
#### GET `/rbac/roles/{id}`
#### POST `/rbac/roles`
#### PUT `/rbac/roles/{id}/permissions`

#### GET `/rbac/permissions`
**Response 200:**
```json
{
  "permissions": [
    { "code": "subscriber:read", "resource": "subscriber", "action": "read" },
    { "code": "subscriber:write", "resource": "subscriber", "action": "write" },
    { "code": "tariff:delete", "resource": "tariff", "action": "delete" },
    { "code": "system:config", "resource": "system", "action": "execute" }
  ]
}
```

#### GET `/rbac/matrix`
**Response:** Full permission matrix (roles × permissions)

---

### 4.5. White Labeling

#### GET `/branding`
**Description:** Public endpoint (no auth required). Returns branding by domain or tenant_id.

**Query Parameters:**
- `domain` (string): e.g., `orange.billing.com`
- `tenant_id` (int): Direct tenant ID

**Response 200:**
```json
{
  "tenant_id": 2,
  "product_name": "Orange Control Panel",
  "colors": {
    "primary": "#FF6600",
    "secondary": "#333333",
    "accent": "#00CC00",
    "danger": "#FF0000"
  },
  "logo_url": "https://cdn.billing.com/branding/2/logo.png?v=12345",
  "favicon_url": "https://cdn.billing.com/branding/2/favicon.ico",
  "login_background_url": "https://cdn.billing.com/branding/2/bg.jpg",
  "contacts": {
    "email": "support@orange.com",
    "phone": "+1-800-ORANGE",
    "telegram": "@orange_support"
  },
  "regional": {
    "timezone": "America/New_York",
    "date_format": "MM/DD/YYYY",
    "first_day_of_week": 0,
    "currency_symbol": "$"
  }
}
```

#### PUT `/branding`
#### POST `/branding/logo`
**Content-Type:** `multipart/form-data`
**Max file size:** 2MB
**Allowed types:** image/png, image/jpeg

#### GET `/branding/email-templates`
#### PUT `/branding/email-templates/{id}`
#### POST `/branding/email-templates/{id}/preview`

---

### 4.6. Localization

#### GET `/locales`
**Response 200:**
```json
{
  "locales": [
    { "code": "en", "name": "English", "active": true, "rtl": false },
    { "code": "ru", "name": "Русский", "active": true, "rtl": false },
    { "code": "es", "name": "Español", "active": true, "rtl": false }
  ]
}
```

#### GET `/translations/{locale}`
**Response 200:**
```json
{
  "locale": "ru",
  "categories": {
    "common": {
      "app.title": "Orange Billing",
      "buttons.save": "Сохранить",
      "buttons.cancel": "Отмена"
    },
    "errors": {
      "insufficient_funds": "Недостаточно средств",
      "invalid_msisdn": "Неверный номер телефона"
    }
  }
}
```

#### POST `/translations`
**Description:** Bulk import translations (Admin only)
**Content-Type:** `multipart/form-data` (CSV or JSON file)

---

### 4.7. System Monitoring

#### GET `/health`
**Response 200:**
```json
{
  "status": "healthy",
  "version": "1.0.0",
  "components": {
    "postgresql": "healthy",
    "redis": "healthy",
    "pulsar": "healthy",
    "cgrates": "healthy"
  },
  "timestamp": "2026-05-20T09:20:27Z"
}
```

#### GET `/metrics`
**Description:** Prometheus metrics endpoint

#### GET `/audit-log`
**Query Parameters:** `tenant_id`, `user_id`, `portal_type`, `action`, `from`, `to`, `page`

#### GET `/pulsar/topics`
#### GET `/pulsar/lag`

#### GET `/database/status`
**Response 200:**
```json
{
  "primary": { "status": "up", "lag_seconds": 0 },
  "replicas": [
    { "host": "postgres-replica-1", "status": "up", "lag_seconds": 0.5 },
    { "host": "postgres-replica-2", "status": "up", "lag_seconds": 0.3 }
  ],
  "partitions": {
    "audit_log": ["audit_log_y2024m01", "audit_log_y2024m02", "audit_log_y2024m03"]
  },
  "vacuum_status": "healthy"
}
```

---

### 4.8. API Keys

#### GET `/api-keys`
#### POST `/api-keys`
#### DELETE `/api-keys/{id}`

---

## 5. WebSocket / Real-Time (Future)

### 5.1. SelfCare
- `wss://my.billing.company.com/ws/balance` — Real-time balance updates
- `wss://my.billing.company.com/ws/notifications` — Push notifications

### 5.2. Operator
- `wss://bss.billing.company.com/ws/sessions` — Active session monitor
- `wss://bss.billing.company.com/ws/alerts` — System alerts

---

## 6. CGRateS RPC Mapping

| UI-Bill Endpoint | CGRateS Method | Portal |
|------------------|----------------|--------|
| GET /balance | `ApierV1.GetAccount` | SelfCare, Operator |
| POST /topup | `ApierV1.AddBalance` | SelfCare |
| POST /balance/adjust | `ApierV1.AddBalance` / `ApierV1.DebitBalance` | Operator |
| GET /cdr | `ApierV1.GetCDRs` | SelfCare, Operator |
| POST /subscribers | `ApierV1.SetAccount` | Operator |
| GET /sessions/active | `SMGv1.GetActiveSessions` | Operator |
| GET /tariffs | `ApierV1.GetTPIds` / `ApierV1.GetTPRatingPlan` | Operator, Admin |
| POST /tariffs | `ApierV1.SetTPRatingPlan` | Operator |
| POST /bulk/tariff-change | `ApierV1.SetAccount` (batch) | Operator |
