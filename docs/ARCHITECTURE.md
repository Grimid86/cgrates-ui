# UI-Bill Architecture Document
## CGRateS Web GUI Platform — Multi-Portal Billing System

**Version:** 1.0.0  
**Date:** 2026-05-20  
**Status:** In Development

---

## 1. Executive Summary

UI-Bill is a white-label, multi-tenant, multi-portal Web GUI platform for CGRateS + Open5GS integration. It provides three physically and logically isolated web portals:

1. **Self-Care Portal** — Subscriber-facing personal account
2. **Operator BSS Portal** — Business support system for operator staff
3. **Admin OSS Portal** — System administration and infrastructure management

---

## 2. High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              EXTERNAL USERS                                  │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐                       │
│  │  Subscribers │  │   Operators  │  │    Admins    │                       │
│  │  (Internet)  │  │  (VPN/Corp)  │  │  (DMZ/NOC)   │                       │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘                       │
└─────────┼────────────────┼────────────────┼─────────────────────────────────┘
          │                │                │
┌─────────▼────────────────▼────────────────▼─────────────────────────────────┐
│                           INGRESS LAYER                                      │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐                       │
│  │  selfcare-ui │  │ operator-ui  │  │   admin-ui   │   (React SPA + nginx) │
│  │   :80/:443   │  │   :80/:443   │  │   :80/:443   │                       │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘                       │
└─────────┼────────────────┼────────────────┼─────────────────────────────────┘
          │                │                │
┌─────────▼────────────────▼────────────────▼─────────────────────────────────┐
│                         GATEWAY LAYER (Go)                                   │
│  ┌──────────────────┐  ┌──────────────────┐  ┌──────────────────┐           │
│  │ selfcare-gateway │  │ operator-gateway │  │  admin-gateway   │           │
│  │    :8081/TCP     │  │    :8082/TCP     │  │    :8083/TCP     │           │
│  │  30 req/min/user │  │  100 req/sec/user│  │  50 req/sec/user │           │
│  │  JWT_SECRET_SC   │  │  JWT_SECRET_OP   │  │  JWT_SECRET_AD   │           │
│  └──────┬───────────┘  └──────┬───────────┘  └──────┬───────────┘           │
└─────────┼────────────────────┼────────────────────┼─────────────────────────┘
          │                    │                    │
          │         ┌──────────┴────────────────────┘
          │         │
┌─────────▼─────────▼─────────────────────────────────────────────────────────┐
│                      EVENT BUS (Apache Pulsar)                               │
│  ┌─────────────────────────────────────────────────────────────────────┐    │
│  │  Tenant: billing                                                    │    │
│  │  ├── Namespace: events      (charges, topups, sessions, cdr.raw)   │    │
│  │  ├── Namespace: audit       (events, security)                     │    │
│  │  ├── Namespace: notifications (email, sms, push)                   │    │
│  │  ├── Namespace: commands    (balance.adjust, tariff.update)        │    │
│  │  └── Namespace: config      (cache invalidation)                   │    │
│  └─────────────────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────────────────┘
          │                    │                    │
┌─────────▼────────────────────▼────────────────────▼─────────────────────────┐
│                      WORKER LAYER (Go microservices)                         │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐ ┌─────────────────────┐   │
│  │audit-consumer│ │cdr-processor│ │email-consumer│ │balance-monitor     │   │
│  └─────────────┘ └─────────────┘ └─────────────┘ └─────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────────┘
          │
┌─────────▼─────────────────────────────────────────────────────────────────────┐
│                      DATA LAYER                                              │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────────────────┐  │
│  │   PostgreSQL    │  │  Redis Cluster  │  │         CGRateS             │  │
│  │  (Primary + 2   │  │  (Cache +       │  │   JSON-RPC :2012            │  │
│  │   Hot Standby)  │  │   Sessions +    │  │   GOB-RPC :2013             │  │
│  │  Partitions:    │  │   Rate Limit)   │  │   Internal network only     │  │
│  │  audit_log,     │  │                 │  │                             │  │
│  │  cdr_hourly     │  │                 │  │                             │  │
│  └─────────────────┘  └─────────────────┘  └─────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 3. Network Isolation

### 3.1. Kubernetes Network Policies

```yaml
# Default deny all cross-portal traffic
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: default-deny-cross-portal
  namespace: billing
spec:
  podSelector: {}
  policyTypes:
  - Ingress
```

### 3.2. Allowed Traffic Matrix

| Source → Destination | SelfCare UI | Operator UI | Admin UI | SelfCare GW | Operator GW | Admin GW | CGRateS | PostgreSQL | Pulsar |
|---------------------|:-----------:|:-----------:|:--------:|:-----------:|:-----------:|:--------:|:-------:|:----------:|:------:|
| **Internet**        | ✅ 443      | ❌          | ❌       | ❌          | ❌          | ❌       | ❌      | ❌         | ❌     |
| **VPN/Corp**        | ❌          | ✅ 443      | ❌       | ❌          | ❌          | ❌       | ❌      | ❌         | ❌     |
| **DMZ/NOC**         | ❌          | ❌          | ✅ 443   | ❌          | ❌          | ❌       | ❌      | ❌         | ❌     |
| **SelfCare UI**     | —           | ❌          | ❌       | ✅ 8080     | ❌          | ❌       | ❌      | ❌         | ❌     |
| **Operator UI**     | ❌          | —           | ❌       | ❌          | ✅ 8080     | ❌       | ❌      | ❌         | ❌     |
| **Admin UI**        | ❌          | ❌          | —        | ❌          | ❌          | ✅ 8080  | ❌      | ❌         | ❌     |
| **SelfCare GW**     | ❌          | ❌          | ❌       | —           | ❌          | ❌       | ✅ 2012 | ✅ 5432    | ✅ 6650|
| **Operator GW**     | ❌          | ❌          | ❌       | ❌          | —           | ❌       | ✅ 2012 | ✅ 5432    | ✅ 6650|
| **Admin GW**        | ❌          | ❌          | ❌       | ❌          | ❌          | —        | ✅ 2012 | ✅ 5432    | ✅ 6650|
| **Workers**         | ❌          | ❌          | ❌       | ❌          | ❌          | ❌       | ✅ 2012 | ✅ 5432    | ✅ 6650|

### 3.3. Zero Trust mTLS

All inter-service communication uses mTLS (Istio/Linkerd or cert-manager):
- Gateway → Worker: signed JWT + mTLS
- Worker → CGRateS: API key + mTLS
- Worker → PostgreSQL: TLS 1.3 + separate DB users

---

## 4. Data Flow

### 4.1. Self-Care: Balance Check

```
1. Subscriber → SelfCare UI → GET /api/v1/balance
2. SelfCare Gateway:
   a. Rate limit check (Redis sliding window)
   b. JWT validation (JWT_SECRET_SELFCARE)
   c. Tenant enforcement (subscriber belongs to tenant)
   d. Scope check (balance in whitelist)
   e. Cache check (Redis: balance:sub:{id}, TTL 30s)
   f. Cache miss → CGRateS RPC (ApierV1.GetAccount)
   g. Audit log → Pulsar (async)
3. Response → Subscriber
```

### 4.2. Operator: Bulk Tariff Update

```
1. Operator → Operator UI → POST /api/v1/tariffs/bulk-update
2. Operator Gateway:
   a. IP whitelist check
   b. JWT validation + MFA check (for > $100 operations)
   c. RBAC check (tariff:write permission)
   d. Idempotency check (Redis: idempotency:{key}, TTL 24h)
   e. Validation (JSON Schema whitelist)
   f. Publish to Pulsar: commands.tariff.update
   g. Return 202 Accepted (async processing)
3. Worker:
   a. Consume from Pulsar
   b. Process each subscriber
   c. CGRateS RPC (ApierV1.SetTPActions)
   d. Audit log → Pulsar
   e. Notify completion via email/push
```

### 4.3. Admin: Tenant Creation

```
1. Admin → Admin UI → POST /api/v1/tenants
2. Admin Gateway:
   a. IP whitelist (DMZ only)
   b. JWT validation + mandatory MFA
   c. RBAC check (tenant:create)
   d. Four-eyes principle (second admin approval)
   e. Validation
   f. INSERT INTO tenant (PostgreSQL primary)
   g. Publish config.changes → Pulsar
   h. Sync audit log (critical operation)
3. Cache Invalidator Worker:
   a. Consume config.changes
   b. Invalidate Redis cache
4. Response → Admin
```

---

## 5. Security Architecture

### 5.1. JWT Token Structure

**SelfCare Token Claims:**
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

**Operator Token Claims:**
```json
{
  "user_id": 100,
  "tenant_id": 1,
  "role": "admin",
  "permissions": ["subscriber:read", "subscriber:write", "tariff:read"],
  "locale": "en",
  "branding_id": 1,
  "mfa_verified": true,
  "iat": 1716192000,
  "exp": 1716192900
}
```

**Admin Token Claims:**
```json
{
  "user_id": 1,
  "role": "root",
  "locale": "en",
  "mfa_verified": true,
  "session_id": "uuid",
  "iat": 1716192000,
  "exp": 1716193800,
  "type": "access"
}
```

### 5.2. Middleware Chain (per portal)

See `docs/SECURITY.md` for full middleware implementation details.

### 5.3. RPC Injection Protection

- Balance type whitelist: `["*monetary", "*data", "*sms", "*voice", "*bonus"]`
- IMSI/MSISDN regex: `^[0-9]+$`
- Tariff name regex: `^[A-Za-z0-9_\-]+$`
- JSON Schema validation before any CGRateS RPC call
- Prepared statements for all DB queries

---

## 6. Scalability & Performance

### 6.1. Horizontal Scaling

| Component | Min Replicas | Max Replicas | HPA Trigger |
|-----------|:------------:|:------------:|:-----------:|
| SelfCare UI | 3 | 50 | CPU 70% |
| SelfCare Gateway | 3 | 100 | CPU 70% + p99 latency < 200ms |
| Operator UI | 2 | 10 | CPU 70% |
| Operator Gateway | 3 | 20 | CPU 70% |
| Admin UI | 1 | 3 | CPU 70% |
| Admin Gateway | 2 | 5 | CPU 70% |
| CDR Processor | 5 | 50 | Pulsar consumer lag > 1000 |
| Audit Consumer | 3 | 10 | Pulsar consumer lag > 500 |

### 6.2. CQRS Pattern

- **Commands (Writes):** Gateway → PostgreSQL Primary + Pulsar events
- **Queries (Reads):** Gateway → PostgreSQL Hot Standby + Redis Cache
- **Read Models:**
  - `mv_subscriber_balance` — materialized view
  - `cdr_hourly_stats`, `cdr_daily_stats` — pre-aggregated
  - Redis cache with TTL 30s for balance queries

### 6.3. Backpressure & Circuit Breaker

- Pulsar accumulates messages during peak load
- Consumers scale based on lag metrics
- Circuit breaker between Gateway and CGRateS:
  - If CGRateS unavailable → fallback to async Pulsar processing
  - Response to user: "202 Accepted — processing in background"

---

## 7. Technology Stack

| Layer | Technology |
|-------|-----------|
| Frontend | React 18 + Vite + TailwindCSS |
| Backend Gateway | Go 1.22 + Echo/Gin + pgx |
| Database | PostgreSQL 15 (partitioning, JSONB) |
| Cache / Sessions | Redis 7 Cluster |
| Message Bus | Apache Pulsar 3.2 |
| Object Storage | MinIO / S3 |
| Container | Docker + distroless |
| Orchestration | Kubernetes |
| Service Mesh | Istio (mTLS) |
| Monitoring | Prometheus + Grafana + Jaeger |
| CI/CD | GitHub Actions |

---

## 8. Repository Structure

```
ui-bill/
├── docs/                  # Architecture, API contracts, security
├── infra/                 # Docker, K8s, Terraform
├── database/              # Migrations, seeds, functions
├── backend/
│   ├── pkg/               # Shared libraries
│   ├── selfcare-gateway/  # SelfCare API server
│   ├── operator-gateway/  # Operator BSS API server
│   └── admin-gateway/     # Admin OSS API server
├── frontend/
│   ├── selfcare-ui/       # React SPA
│   ├── operator-ui/       # React SPA
│   └── admin-ui/          # React SPA
└── workers/               # Pulsar consumers
```

---

## 9. Development Workflow

1. **Clone repo:** `git clone git@github.com:Grimid86/cgrates-ui.git`
2. **Copy env:** `cp .env.example .env`
3. **Start infra:** `make up`
4. **Run migrations:** `make migrate-up`
5. **Seed data:** `make seed`
6. **Start gateway:** `make run-gateway-selfcare`
7. **Start UI:** `make dev-ui-selfcare`

For parallel development:
- Backend developer works in `backend/{gateway}/`
- Frontend developer works in `frontend/{ui}/`
- DevOps works in `infra/`
- DBA works in `database/`

No reverse engineering required — all contracts are documented in `docs/API_CONTRACTS.md`.
