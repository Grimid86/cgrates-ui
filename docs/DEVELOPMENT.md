# UI-Bill Development Guide
## Getting Started for Parallel Development

---

## Prerequisites

- Docker 24+ & Docker Compose
- Go 1.22+
- Node.js 20+ & npm 10+
- PostgreSQL client (psql) — optional, for manual queries
- `migrate` CLI — for database migrations
  ```bash
  go install -tags postgres github.com/golang-migrate/migrate/v4/cmd/migrate@latest
  ```

---

## Quick Start (5 minutes)

```bash
# 1. Clone repository
git clone git@github.com:Grimid86/cgrates-ui.git
cd cgrates-ui

# 2. Setup environment
cp .env.example .env
# Edit .env if needed (defaults work for local Docker)

# 3. Start infrastructure
make up

# 4. Run migrations
make migrate-up

# 5. Seed initial data
make seed

# 6. Verify
curl http://localhost:8081/health
curl http://localhost:8082/health
curl http://localhost:8083/health
```

---

## Parallel Development Workflow

### Backend Developer (Gateway)

```bash
# Work in your gateway directory
cd backend/selfcare-gateway   # or operator-gateway / admin-gateway

# The shared packages are in backend/pkg/
# - config: environment configuration
# - db: PostgreSQL connection pool (pgx)
# - middleware: JWT, RBAC, rate limiting, CORS
# - rbac: permission checking
# - i18n: localization helpers
# - branding: white-label configuration
# - pulsar: Apache Pulsar producer/consumer
# - redis: Redis cache client
# - audit: audit logging to Pulsar
# - security: input sanitization, hash verification
# - cgrates: JSON-RPC client for CGRateS

# Run locally (requires PostgreSQL, Redis, Pulsar running)
make run-gateway-selfcare
```

**API Contract:** All endpoints are documented in `docs/API_CONTRACTS.md`. Do not change contracts without updating documentation first.

### Frontend Developer (UI)

```bash
# Work in your portal directory
cd frontend/selfcare-ui   # or operator-ui / admin-ui

# Each UI is an independent React + Vite project
npm install
npm run dev

# The UI expects the following environment:
# VITE_API_BASE_URL=http://localhost:8081  (or 8082/8083)
# VITE_DEFAULT_LOCALE=en
```

**Branding:** The UI calls `GET /api/v1/branding` on load to fetch CSS variables and logo URLs.

**i18n:** Translations are loaded from `GET /api/v1/translations/{locale}`. Fallback is English.

### DevOps / Infrastructure

```bash
# Work in infra/
cd infra/

# Docker images
docker-compose -f docker/docker-compose.yml up -d

# Kubernetes (requires kubectl + cluster)
make k8s-deploy
```

### Database Administrator

```bash
# Migrations live in database/migrations/
# Seeds live in database/seeds/

# Create new migration
make migrate-create name=add_subscriber_notes

# Edit database/migrations/00X_add_subscriber_notes.up.sql
# Edit database/migrations/00X_add_subscriber_notes.down.sql

# Apply
make migrate-up
```

**Rules:**
- Always use `TIMESTAMPTZ`, never `TIMESTAMP`
- Always use `gen_random_uuid()` for UUIDs
- Partition large tables by RANGE on `created_at`
- Add indexes in separate migration (002 pattern)
- Never drop columns with data without backup plan

---

## Project Structure

```
ui-bill/
├── docs/                          # Architecture, API, Security docs
│   ├── ARCHITECTURE.md            # High-level design
│   ├── DATABASE.md                # Schema documentation
│   ├── API_CONTRACTS.md           # OpenAPI contracts
│   ├── SECURITY.md                # Security policies
│   └── DEVELOPMENT.md             # This file
│
├── infra/                         # Infrastructure as Code
│   ├── docker/
│   │   ├── docker-compose.yml     # Local dev stack
│   │   ├── Dockerfile.gateway     # Multi-stage Go build
│   │   ├── Dockerfile.*-ui        # Frontend builds
│   │   ├── nginx-*.conf           # nginx configs
│   │   └── cgrates.json           # CGRateS dev config
│   └── k8s/                       # Kubernetes manifests
│       ├── namespaces/
│       ├── selfcare/
│       ├── operator/
│       ├── admin/
│       ├── database/
│       └── messaging/
│
├── database/                      # Database artifacts
│   ├── migrations/                # golang-migrate compatible
│   │   ├── 001_initial_schema.sql
│   │   └── 002_indexes_and_triggers.sql
│   ├── seeds/                     # Initial data
│   │   └── 001_initial_seed.sql
│   └── functions/                 # SQL functions
│
├── backend/                       # Go backend services
│   ├── pkg/                       # Shared libraries
│   │   ├── config/
│   │   ├── db/
│   │   ├── middleware/
│   │   │   ├── jwt.go
│   │   │   ├── rbac.go
│   │   │   ├── ratelimit.go
│   │   │   ├── cors.go
│   │   │   ├── csrf.go
│   │   │   ├── audit.go
│   │   │   └── sanitize.go
│   │   ├── rbac/
│   │   ├── i18n/
│   │   ├── branding/
│   │   ├── pulsar/
│   │   ├── redis/
│   │   ├── audit/
│   │   ├── security/
│   │   └── cgrates/
│   ├── selfcare-gateway/
│   │   └── cmd/main.go
│   ├── operator-gateway/
│   │   └── cmd/main.go
│   └── admin-gateway/
│       └── cmd/main.go
│
├── frontend/                      # React SPAs
│   ├── selfcare-ui/               # Subscriber portal
│   ├── operator-ui/               # BSS portal
│   └── admin-ui/                  # OSS portal
│
├── workers/                       # Background consumers
│   ├── audit-consumer/
│   ├── cdr-processor/
│   ├── email-consumer/
│   ├── sms-consumer/
│   ├── balance-monitor/
│   └── cache-invalidator/
│
├── Makefile                       # Automation
└── .env.example                   # Environment template
```

---

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `POSTGRES_HOST` | localhost | PostgreSQL host |
| `POSTGRES_PORT` | 5432 | PostgreSQL port |
| `POSTGRES_DB` | uibill | Database name |
| `POSTGRES_USER` | postgres | DB user |
| `POSTGRES_PASSWORD` | postgres | DB password |
| `REDIS_HOST` | localhost | Redis host |
| `REDIS_PORT` | 6379 | Redis port |
| `PULSAR_URL` | pulsar://localhost:6650 | Pulsar broker |
| `CGRATES_HOST` | localhost | CGRateS host |
| `CGRATES_PORT` | 2012 | CGRateS JSON-RPC port |
| `JWT_SECRET_SELFCARE` | — | SelfCare JWT secret |
| `JWT_SECRET_OPERATOR` | — | Operator JWT secret |
| `JWT_SECRET_ADMIN` | — | Admin JWT secret |
| `PORT_SELFCARE_GATEWAY` | 8081 | SelfCare API port |
| `PORT_OPERATOR_GATEWAY` | 8082 | Operator API port |
| `PORT_ADMIN_GATEWAY` | 8083 | Admin API port |

---

## Testing

```bash
# Backend tests
make test

# Integration tests (requires running services)
cd backend && go test ./... -tags=integration

# Frontend tests
cd frontend/selfcare-ui && npm test
```

---

## Git Workflow

1. Create feature branch from `main`
2. Make changes in your component directory
3. Update documentation if contracts change
4. Commit with conventional commits:
   - `feat(backend): add subscriber search endpoint`
   - `fix(frontend): resolve login redirect loop`
   - `docs(api): update branding endpoint spec`
   - `infra(k8s): add HPA for selfcare gateway`
5. Push to remote (passphrase: `5XzL01gh`)

---

## Troubleshooting

### PostgreSQL connection refused
```bash
docker-compose -f infra/docker/docker-compose.yml ps
docker-compose -f infra/docker/docker-compose.yml logs postgres
```

### Migrations fail
```bash
# Check version
migrate -path database/migrations -database "postgres://postgres:postgres@localhost:5432/uibill?sslmode=disable" version

# Force version (DANGEROUS)
migrate -path database/migrations -database "..." force 1
```

### Pulsar not starting
Pulsar requires significant memory. Ensure Docker has at least 4GB RAM allocated.

### SSH push fails
```bash
eval "$(ssh-agent -s)"
ssh-add ~/.ssh/id_ed25519
# Enter passphrase: 5XzL01gh
git push origin main
```

---

## Contacts & Resources

- **Repository:** https://github.com/Grimid86/cgrates-ui
- **CGRateS Docs:** https://cgrates.readthedocs.io/
- **Pulsar Docs:** https://pulsar.apache.org/docs/
