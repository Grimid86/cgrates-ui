# UI-Bill Security Policy

## Overview

This document defines the security architecture, policies, and implementation details for the UI-Bill platform. It is designed to enable parallel development without requiring security reverse engineering.

---

## 1. Threat Model

### 1.1. Assets

| Asset | Sensitivity | Owner |
|-------|-------------|-------|
| Subscriber MSISDN/PIN | Critical | SelfCare |
| Staff credentials (passwords, MFA) | Critical | All portals |
| JWT secrets | Critical | Infrastructure |
| CGRateS RPC access | Critical | Backend |
| CDR records | High | Operator |
| Balance data | High | SelfCare, Operator |
| Tariff configurations | Medium | Operator |
| Branding configs | Low | Admin |

### 1.2. Threat Actors

- **External attacker** — targets SelfCare login, attempts credential stuffing
- **Malicious insider (operator)** — attempts unauthorized balance adjustments
- **Compromised admin account** — attempts to modify tenant isolation
- **Network attacker** — attempts to intercept inter-service traffic

### 1.3. Attack Vectors

- RPC injection via crafted API requests
- JWT token theft / replay
- CSRF on state-changing operations
- SQL injection (mitigated by pgx prepared statements)
- XSS via unescaped user input
- Privilege escalation via role manipulation

---

## 2. Authentication

### 2.1. SelfCare (Subscribers)

**Method:** MSISDN + PIN or MSISDN + OTP

**Flow:**
```
1. POST /auth/login { msisdn, pin }
2. Server validates against subscriber_credentials (bcrypt pin_hash)
3. If failed_login_attempts >= 5 → account locked for 15 minutes
4. On success: generate access_token (15 min) + refresh_token (7 days, rotating)
5. Store refresh token hash in subscriber_sessions table
```

**Token Claims:**
```json
{
  "sub_id": 12345,
  "tenant_id": 1,
  "msisdn": "79001234567",
  "locale": "ru",
  "iat": 1716192000,
  "exp": 1716192900,
  "type": "access"
}
```

### 2.2. Operator (Staff)

**Method:** Email + Password + MFA (for Admin/Reseller roles)

**Flow:**
```
1. POST /auth/login { email, password, mfa_code }
2. Validate password against users.password_hash (bcrypt)
3. If role in [admin, reseller] → verify TOTP code
4. On success: issue JWT with mfa_verified claim
```

**Token Claims:**
```json
{
  "user_id": 100,
  "tenant_id": 1,
  "role": "admin",
  "permissions": ["subscriber:read", "subscriber:write"],
  "locale": "en",
  "branding_id": 1,
  "mfa_verified": true,
  "type": "access"
}
```

### 2.3. Admin (System)

**Method:** Email + Password + Mandatory MFA

**Additional Requirements:**
- Short access token TTL: 15 minutes (mandatory refresh)
- Backup codes generated on MFA setup (8 codes, single use)
- Alert on every admin login (Pulsar → email/Slack)

---

## 3. Authorization (RBAC)

### 3.1. Role Hierarchy

```
root
├── reseller
│   └── admin
│       ├── support
│       └── viewer
```

### 3.2. Permission Matrix

| Permission | Root | Reseller | Admin | Support | Viewer |
|-----------|:----:|:--------:|:-----:|:-------:|:------:|
| subscriber:read | ✅ | ✅ | ✅ | ✅ | ✅ |
| subscriber:write | ✅ | ✅ | ✅ | ❌ | ❌ |
| subscriber:delete | ✅ | ❌ | ✅ | ❌ | ❌ |
| balance:read | ✅ | ✅ | ✅ | ✅ | ✅ |
| balance:write | ✅ | ✅ | ✅ | ❌ | ❌ |
| tariff:read | ✅ | ✅ | ✅ | ✅ | ❌ |
| tariff:write | ✅ | ✅ | ✅ | ❌ | ❌ |
| tariff:delete | ✅ | ❌ | ✅ | ❌ | ❌ |
| cdr:read | ✅ | ✅ | ✅ | ✅ | ❌ |
| cdr:export | ✅ | ✅ | ✅ | ❌ | ❌ |
| session:read | ✅ | ✅ | ✅ | ✅ | ❌ |
| system:config | ✅ | ❌ | ❌ | ❌ | ❌ |
| tenant:create | ✅ | ❌ | ❌ | ❌ | ❌ |
| user:create | ✅ | ✅ | ✅ | ❌ | ❌ |
| branding:manage | ✅ | ✅ | ❌ | ❌ | ❌ |
| translation:manage | ✅ | ✅ | ❌ | ❌ | ❌ |

### 3.3. Tenant Isolation Rules

1. **Root** can access all tenants
2. **Reseller** can access own tenant and child tenants (if hierarchy enabled)
3. **Admin/Support/Viewer** can access only own tenant
4. **SelfCare subscriber** can access only own MSISDN data
5. **Branding** is isolated per tenant — user from tenant A cannot see tenant B branding

---

## 4. Rate Limiting

### 4.1. Per-Portal Limits

| Endpoint | SelfCare | Operator | Admin |
|----------|----------|----------|-------|
| /login | 5/min/IP | 10/min/IP | 5/min/IP |
| /api/* (authenticated) | 30/min/user | 100/sec/user | 50/sec/user |
| /api/* (global per IP) | 300/min/IP | 2000/sec/IP | 500/sec/IP |

### 4.2. Implementation

Redis sliding window counter:
```go
func (c *Client) SlidingWindowRateLimit(ctx, key string, limit int, window time.Duration) (bool, error) {
    pipe := c.rdb.Pipeline()
    now := time.Now().Unix()
    windowStart := now - int64(window.Seconds())
    pipe.ZRemRangeByScore(ctx, key, "0", fmt.Sprintf("%d", windowStart))
    pipe.ZCard(ctx, key)
    pipe.ZAdd(ctx, key, redis.Z{Score: float64(now), Member: now})
    pipe.Expire(ctx, key, window)
    cmders, _ := pipe.Exec(ctx)
    count := cmders[1].(*redis.IntCmd).Val()
    return count < int64(limit), nil
}
```

---

## 5. CSRF Protection

**Method:** Double-submit cookie pattern

**Flow:**
```
1. On login success: server sets HttpOnly cookie "csrf_token={random}"
2. Frontend reads cookie and sends same value in header "X-CSRF-Token"
3. Server compares cookie value with header value
4. Mismatch → 403 Forbidden
```

**Cookie Attributes:**
```
Set-Cookie: csrf_token=xxx; Path=/; HttpOnly; Secure; SameSite=Strict
```

---

## 6. Input Sanitization

### 6.1. Allowed Patterns (Regex Whitelist)

| Field | Pattern | Example |
|-------|---------|---------|
| MSISDN | `^[0-9]+$` | 79001234567 |
| IMSI | `^[0-9]{15}$` | 250011234567890 |
| Tariff name | `^[A-Za-z0-9_\-]+$` | Unlimited_Plus_v2 |
| Email | RFC 5322 simplified | user@example.com |
| Balance type | Enum whitelist | *monetary, *data, *sms, *voice, *bonus |
| Hex color | `^#[0-9A-Fa-f]{6}$` | #FF6600 |

### 6.2. Body Size Limits

- SelfCare: 1MB
- Operator: 5MB
- Admin: 2MB

### 6.3. JSON Schema Validation

All CGRateS RPC requests validated against JSON Schema before forwarding:
```json
{
  "type": "object",
  "required": ["Tenant", "Account", "BalanceType", "Value"],
  "properties": {
    "BalanceType": { "enum": ["*monetary", "*data", "*sms", "*voice", "*bonus"] },
    "Value": { "type": "number", "minimum": 0 }
  }
}
```

---

## 7. Audit Logging

### 7.1. Events Logged

- All authentication attempts (success and failure)
- All balance adjustments
- All tariff changes
- All tenant/user modifications
- All MFA enable/disable events
- All API key rotations

### 7.2. Audit Entry Format

```json
{
  "tenant_id": 1,
  "user_id": 100,
  "portal_type": "operator",
  "action": "UPDATE",
  "entity_type": "subscriber_credentials",
  "entity_id": "12345",
  "old_data": { "status": "active" },
  "new_data": { "status": "blocked" },
  "ip_address": "10.0.1.15",
  "user_agent": "Mozilla/5.0 ...",
  "request_id": "550e8400-e29b-41d4-a716-446655440000",
  "timestamp": "2026-05-20T09:20:27Z"
}
```

### 7.3. Delivery

- Auth events: synchronous (block until written)
- Other events: asynchronous via Pulsar → Audit Consumer → PostgreSQL

---

## 8. Network Security

### 8.1. mTLS

All inter-service communication uses mutual TLS:
- Gateway → CGRateS: TLS 1.3 + API key
- Gateway → PostgreSQL: TLS 1.3 + separate DB users
- Gateway → Pulsar: TLS 1.3 + auth token
- Worker → CGRateS: TLS 1.3 + API key

### 8.2. Network Policies (Kubernetes)

```yaml
# SelfCare can only receive from SelfCare UI
ingress:
  - from:
    - podSelector:
        matchLabels:
          app: selfcare-ui

# Admin cannot receive from SelfCare or Operator
ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          name: billing-admin
```

### 8.3. IP Whitelisting

| Portal | Allowed Sources |
|--------|-----------------|
| SelfCare | Internet (0.0.0.0/0) |
| Operator | Corporate VPN (10.0.0.0/8, 172.16.0.0/12) |
| Admin | DMZ only (10.255.0.0/24, bastion host) |

---

## 9. Secrets Management

### 9.1. Rotation Policy

| Secret | Rotation Period | Method |
|--------|-----------------|--------|
| JWT secrets | 90 days | Kubernetes rolling update |
| DB passwords | 90 days | PostgreSQL ALTER USER + rolling restart |
| Pulsar tokens | 90 days | Pulsar admin API + rolling restart |
| MFA secrets | On demand | Admin resets per user |
| API keys | 180 days | Auto-expiry + notification |

### 9.2. Storage

- Kubernetes Secrets (encrypted at rest with etcd encryption)
- No secrets in Git (use SealedSecrets or Vault)
- No secrets in Docker images (multi-stage builds with distroless)

---

## 10. Vulnerability Response

### 10.1. Severity Classification

| Severity | CVSS | Response Time |
|----------|------|---------------|
| Critical | 9.0-10.0 | 4 hours |
| High | 7.0-8.9 | 24 hours |
| Medium | 4.0-6.9 | 72 hours |
| Low | 0.1-3.9 | Next release |

### 10.2. CI/CD Security Gates

- Trivy scan on every build (blocks on CRITICAL/HIGH)
- `go vet` and `golangci-lint` on every PR
- Dependency vulnerability scan (Snyk or Dependabot)
- Container image signed with Cosign
