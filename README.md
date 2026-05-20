# UI-Bill — Billing & Subscriber Management Platform

> **Telecom-grade BSS/OSS platform** для управления абонентами, тарифами, балансом и CDR с white-label поддержкой и мультиязычным интерфейсом.

---

## Содержание

- [Обзор](#обзор)
- [Архитектура](#архитектура)
- [Технологический стек](#технологический-стек)
- [Выполненная работа](#выполненная-работа)
  - [Phase 1 — Foundation Hardening](#phase-1--foundation-hardening)
  - [Phase 2A — Backend: Stubs → Real](#phase-2a--backend-stubs--real)
  - [Phase 2B — Frontend Gaps](#phase-2b--frontend-gaps)
  - [Phase 2B+ — Localization](#phase-2b--localization)
- [Быстрый старт](#быстрый-старт)
- [Структура проекта](#структура-проекта)
- [API Endpoints](#api-endpoints)
- [База данных](#база-данных)
- [Аутентификация и тестовые данные](#аутентификация-и-тестовые-данные)
- [Workers и Event Pipeline](#workers-и-event-pipeline)
- [Переменные окружения](#переменные-окружения)
- [Roadmap](#roadmap)
- [Лицензия](#лицензия)

---

## Обзор

UI-Bill — это полноценная платформа биллинга для телеком-операторов, построенная на микросервисной архитектуре. Платформа предоставляет три портала:

| Портал | Назначение | URL (dev) |
|--------|-----------|-----------|
| **SelfCare** | Личный кабинет абонента | http://localhost:13001 |
| **Operator** | Панель оператора (BSS) | http://localhost:13002 |
| **Admin** | Управление платформой (OSS) | http://localhost:13003 |

**Ключевые возможности:**
- JWT-аутентификация с MFA/TOTP и backup codes
- RBAC с granular permissions
- White-label branding (логотипы, цвета, CSS)
- Мультиязычность (EN/RU/ES) с inline-редактором
- Управление абонентами, тарифами, балансом
- CDR экспорт (CSV/JSON) в MinIO
- Revenue-отчёты
- Event-driven архитектура на Apache Pulsar
- Audit log с полным трассированием

---

## Архитектура

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              CLIENT LAYER                                    │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐                      │
│  │  SelfCare UI │  │ Operator UI  │  │   Admin UI   │   React 18 + Vite   │
│  │   :13001     │  │   :13002     │  │   :13003     │   TailwindCSS       │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘                      │
└─────────┼─────────────────┼─────────────────┼──────────────────────────────┘
          │                 │                 │
┌─────────┼─────────────────┼─────────────────┼──────────────────────────────┐
│         ▼                 ▼                 ▼                              │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐                      │
│  │SelfCare GW   │  │ Operator GW  │  │  Admin GW    │   Go 1.23 + Echo v4 │
│  │   :18081     │  │   :18082     │  │   :18083     │                      │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘                      │
└─────────┼─────────────────┼─────────────────┼──────────────────────────────┘
          │                 │                 │
          └─────────────────┼─────────────────┘
                            ▼
          ┌─────────────────────────────────────┐
          │         SHARED SERVICES              │
          │  PostgreSQL 15  │  Redis 7           │
          │  Apache Pulsar  │  MinIO (S3)        │
          │  CGRateS (mock) │                    │
          └─────────────────────────────────────┘
                            │
          ┌─────────────────┼─────────────────┐
          ▼                 ▼                 ▼
   ┌─────────────┐  ┌─────────────┐  ┌─────────────┐
   │Audit Consumer│  │Email Consumer│  │CDR Processor│
   │Cache Invalid │  │SMS Consumer  │  │Balance Mon  │
   └─────────────┘  └─────────────┘  └─────────────┘
```

---

## Технологический стек

### Backend
| Компонент | Технология | Версия |
|-----------|-----------|--------|
| Language | Go | 1.23 |
| Web Framework | Echo | v4.12.0 |
| Database Driver | pgx | v5.6.0 |
| Cache | go-redis | v9.5.3 |
| Messaging | pulsar-client-go | v0.12.1 |
| JWT | golang-jwt/jwt | v5.2.1 |
| Crypto | golang.org/x/crypto | v0.24.0 |
| TOTP | pquerna/otp | v1.5.0 |
| Object Storage | minio-go | v7.0.74 |
| UUID | google/uuid | v1.6.0 |

### Frontend
| Компонент | Технология | Версия |
|-----------|-----------|--------|
| Framework | React | 18 |
| Build Tool | Vite | latest |
| Styling | TailwindCSS | v3 |
| HTTP Client | Axios | latest |
| Icons | Lucide React | latest |
| State | React Query (TanStack) | latest |

### Infrastructure
| Компонент | Технология |
|-----------|-----------|
| Database | PostgreSQL 15 |
| Cache | Redis 7 |
| Message Bus | Apache Pulsar 3.2 |
| Object Storage | MinIO |
| Rating Engine | CGRateS (mock для dev) |
| Containerization | Docker + Docker Compose |
| Orchestration | Kubernetes manifests |

---

## Выполненная работа

### Phase 1 — Foundation Hardening ✅

| Функционал | Статус | Описание |
|-----------|--------|----------|
| **MFA/TOTP** | ✅ Реализовано | Генерация секрета через `crypto/rand` + base32, верификация через `pquerna/otp`, QR-код, backup codes (10 шт., хешированные), AES-шифрование секрета |
| **Idempotency Middleware** | ✅ Реализовано | Проверка `rpc_idempotency_keys` в PostgreSQL, TTL 24ч, защита от дублирования mutating запросов |
| **Refresh Token Rotation** | ✅ Реализовано | Rotation при каждом refresh, blacklist в Redis (TTL = срок refresh token), family tracking |
| **Account Lockout** | ✅ Реализовано | 5 неудачных попыток → блокировка на 15 мин, frontend auto-refresh после разблокировки |
| **Rate Limiting** | ✅ Реализовано | SelfCare: 30 RPM, Operator: 100 RPS, Admin: 50 RPS |
| **CSRF Protection** | ✅ Реализовано | Double-submit cookie pattern |
| **Audit Middleware** | ✅ Реализовано | Логирование всех запросов: user, IP, endpoint, payload hash, timestamp |

**Файлы:**
- `backend/pkg/middleware/idempotency.go`
- `backend/pkg/auth/staff.go`
- `backend/pkg/auth/selfcare.go`
- `backend/pkg/security/security.go`
- `backend/pkg/middleware/blacklist.go`

### Phase 2A — Backend: Stubs → Real ✅

| Функционал | Статус | Описание |
|-----------|--------|----------|
| **MinIO Logo Upload** | ✅ Реализовано | `POST /branding/logo`, multipart form, валидация (png/jpg/svg, max 2MB), upload в MinIO, update DB, publish `branding_updated` event |
| **Tariff CRUD** | ✅ Реализовано | `GET/PUT/DELETE /tariffs/:id`, soft delete (status='archived'), защита поля status при update |
| **Balance Freeze/Unfreeze** | ✅ Реализовано | `PUT /subscribers/:id/freeze|unfreeze`, колонка `balance_frozen_at`, операции `freeze`/`unfreeze` в `balance_history`, CGRateS sync |
| **CDR Export** | ✅ Реализовано | `GET /cdr/export`, форматы CSV/JSON, fetch из CGRateS, генерация файла, upload в MinIO, `cdr_exports` таблица |
| **Revenue Report** | ✅ Реализовано | `GET /reports/revenue`, SQL агрегация `balance_history` по дням: `total_charges`, `total_topups`, `net_revenue`, `subscriber_count` |
| **RBAC Role Detail** | ✅ Реализовано | `GET /rbac/roles/:id`, возвращает полную роль с permissions JSONB |

**Новые таблицы:**
- `cdr_exports` — трекинг экспортов
- `balance_history` — расширен операциями `freeze`/`unfreeze`
- `subscriber_credentials` — добавлена `balance_frozen_at`

**Файлы:**
- `backend/admin-gateway/handlers/handlers.go`
- `backend/operator-gateway/handlers/handlers.go`
- `backend/pkg/storage/minio.go`
- `database/migrations/003_balance_freeze.sql`
- `database/migrations/004_cdr_exports.sql`

### Phase 2B — Frontend Gaps ✅

| Функционал | Статус | Описание |
|-----------|--------|----------|
| **Sidebar Navigation** | ✅ Реализовано | Collapsible, responsive, группировка пунктов по ролям, active state, language switcher, tenant branding |
| **Subscriber New Page** | ✅ Реализовано | Форма (MSISDN, IMSI, Email, Category, PIN), POST `/subscribers`, redirect на карточку |
| **TopUp Success Page** | ✅ Реализовано | Отображает сумму пополнения, запрашивает текущий баланс |
| **Edit/Delete Flows** | ✅ Реализовано | Inline edit/delete для Tenants, Users (с Reset MFA/Password), Tariffs (с Activate) |
| **Subscriber Actions** | ✅ Реализовано | Block/Unblock/Freeze/Unfreeze кнопки на карточке абонента |
| **Layout Integration** | ✅ Реализовано | Sidebar интегрирован в Admin и Operator UI |

**Файлы:**
- `frontend/admin-ui/src/components/layout/Layout.jsx`
- `frontend/operator-ui/src/components/layout/Layout.jsx`
- `frontend/operator-ui/src/pages/SubscriberNewPage.jsx`
- `frontend/selfcare-ui/src/pages/TopUpSuccessPage.jsx`

### Phase 2B+ — Localization ✅

| Функционал | Статус | Описание |
|-----------|--------|----------|
| **I18nContext** | ✅ Реализовано | React context для Admin/Operator UI, загрузка переводов с backend `/translations/:locale`, persist в localStorage |
| **Language Switcher** | ✅ Реализовано | В Sidebar (EN/RU/ES), переключение без перезагрузки |
| **Translation Seeds** | ✅ Реализовано | ~75 ключей в БД для категорий: `common`, `buttons`, `errors`, `balance`, `nav` |
| **UI Coverage** | ⚠️ Частично | Sidebar и ключевые страницы (Tenants, Subscribers) используют `t()`, остальные страницы — в процессе |

**Файлы:**
- `frontend/admin-ui/src/contexts/I18nContext.jsx`
- `frontend/operator-ui/src/contexts/I18nContext.jsx`
- `frontend/admin-ui/src/main.jsx` (I18nProvider обёртка)
- `frontend/operator-ui/src/main.jsx` (I18nProvider обёртка)
- `database/migrations/005_translations_ui.sql`

---

## Быстрый старт

### Требования
- Docker 24.x + Docker Compose
- Go 1.23 (для локальной разработки backend)
- Node.js 20+ (для локальной разработки frontend)
- Make (опционально)

### 1. Клонирование и конфигурация

```bash
git clone git@github.com:Grimid86/cgrates-ui.git
cd cgrates-ui
cp .env.example .env
# Отредактируйте .env при необходимости
```

### 2. Запуск инфраструктуры

```bash
cd infra/docker
docker-compose up -d
```

### 3. Проверка сервисов

```bash
# PostgreSQL
docker-compose exec postgres pg_isready -U postgres -d uibill

# Redis
docker-compose exec redis redis-cli ping

# Pulsar
docker-compose exec pulsar bin/pulsar-admin brokers healthcheck

# MinIO
curl -f http://localhost:19000/minio/health/live
```

### 4. Запуск gateway (локальная разработка)

```bash
cd backend/admin-gateway/cmd
go run main.go

cd backend/operator-gateway/cmd
go run main.go

cd backend/selfcare-gateway/cmd
go run main.go
```

### 5. Запуск frontend (локальная разработка)

```bash
cd frontend/admin-ui
npm install
npm run dev

cd frontend/operator-ui
npm install
npm run dev

cd frontend/selfcare-ui
npm install
npm run dev
```

### 6. Полный запуск через Docker Compose

```bash
cd infra/docker
docker-compose up -d --build
```

Все 17 контейнеров будут собраны и запущены. Проверка:
```bash
docker-compose ps
```

---

## Структура проекта

```
cgrates-ui/
├── backend/
│   ├── admin-gateway/          # Admin OSS API (:8080 → :18083)
│   │   ├── cmd/main.go
│   │   └── handlers/handlers.go
│   ├── operator-gateway/       # Operator BSS API (:8080 → :18082)
│   │   ├── cmd/main.go
│   │   └── handlers/handlers.go
│   ├── selfcare-gateway/       # SelfCare API (:8080 → :18081)
│   │   ├── cmd/main.go
│   │   └── handlers/handlers.go
│   └── pkg/
│       ├── audit/              # Audit log middleware
│       ├── auth/               # JWT, TOTP, staff/selfcare auth
│       ├── branding/           # White-label config
│       ├── cgrates/            # CGRateS RPC client (mock)
│       ├── config/             # Environment config
│       ├── db/                 # PostgreSQL connection pool
│       ├── i18n/               # Translation service
│       ├── middleware/         # Idempotency, CORS, CSRF, rate limit
│       ├── models/             # Shared data models
│       ├── pulsar/             # Pulsar producer/consumer
│       ├── rbac/               # Role-based access control
│       ├── redis/              # Redis client wrapper
│       ├── security/           # Password hashing, encryption
│       └── storage/            # MinIO S3 client
├── database/
│   ├── migrations/             # SQL миграции (numbered)
│   ├── seeds/                  # Тестовые данные
│   └── functions/              # PostgreSQL functions
├── docs/
│   ├── DEVELOPMENT_PLAN.md     # План разработки
│   ├── API_CONTRACTS.md        # API контракты
│   ├── SECURITY.md             # Security guide
│   ├── DATABASE.md             # ER diagram, schema docs
│   └── ARCHITECTURE.md         # Архитектурные решения
├── frontend/
│   ├── admin-ui/               # Admin portal (React + Vite)
│   ├── operator-ui/            # Operator portal (React + Vite)
│   ├── selfcare-ui/            # SelfCare portal (React + Vite)
│   └── shared-components/      # Shared UI components
├── infra/
│   ├── docker/                 # Dockerfiles + docker-compose.yml
│   └── k8s/                    # Kubernetes manifests
├── workers/
│   ├── audit-consumer/         # Audit log consumer
│   ├── cdr-processor/          # CDR aggregation
│   ├── email-consumer/         # Email notifications
│   ├── sms-consumer/           # SMS notifications
│   ├── balance-monitor/        # Low balance alerts
│   └── cache-invalidator/      # Redis cache invalidation
└── scripts/                    # Utility scripts
```

---

## API Endpoints

### Admin Gateway (`:18083`)

| Method | Endpoint | Описание | Auth |
|--------|----------|----------|------|
| POST | `/auth/login` | Логин админа | Публичный |
| POST | `/auth/mfa/verify` | MFA верификация | JWT (partial) |
| POST | `/auth/refresh` | Refresh token | Refresh token |
| GET | `/tenants` | Список тенантов | Admin |
| POST | `/tenants` | Создать тенант | Admin |
| PUT | `/tenants/:id` | Обновить тенант | Admin |
| DELETE | `/tenants/:id` | Удалить тенант | Admin |
| GET | `/users` | Список пользователей | Admin |
| POST | `/users` | Создать пользователя | Admin |
| PUT | `/users/:id` | Обновить пользователя | Admin |
| DELETE | `/users/:id` | Удалить пользователя | Admin |
| POST | `/users/:id/reset-password` | Сброс пароля | Admin |
| POST | `/users/:id/reset-mfa` | Сброс MFA | Admin |
| GET | `/rbac/roles` | Список ролей | Admin |
| GET | `/rbac/roles/:id` | Детали роли | Admin |
| POST | `/rbac/roles` | Создать роль | Admin |
| PUT | `/rbac/roles/:id` | Обновить роль | Admin |
| DELETE | `/rbac/roles/:id` | Удалить роль | Admin |
| POST | `/branding/logo` | Загрузка логотипа | Admin |
| GET | `/branding` | Конфиг брендинга | Admin |
| GET | `/audit-log` | Audit log | Admin |
| GET | `/health` | Health check | Публичный |
| GET | `/translations/:locale` | Переводы | Публичный |

### Operator Gateway (`:18082`)

| Method | Endpoint | Описание | Auth |
|--------|----------|----------|------|
| POST | `/auth/login` | Логин оператора | Публичный |
| POST | `/auth/mfa/verify` | MFA верификация | JWT (partial) |
| POST | `/auth/refresh` | Refresh token | Refresh token |
| GET | `/subscribers` | Список абонентов | Operator |
| POST | `/subscribers` | Создать абонента | Operator |
| GET | `/subscribers/:id` | Карточка абонента | Operator |
| PUT | `/subscribers/:id` | Обновить абонента | Operator |
| PUT | `/subscribers/:id/block` | Блокировать | Operator |
| PUT | `/subscribers/:id/unblock` | Разблокировать | Operator |
| PUT | `/subscribers/:id/freeze` | Заморозить баланс | Operator |
| PUT | `/subscribers/:id/unfreeze` | Разморозить баланс | Operator |
| GET | `/tariffs` | Список тарифов | Operator |
| GET | `/tariffs/:id` | Детали тарифа | Operator |
| POST | `/tariffs` | Создать тариф | Operator |
| PUT | `/tariffs/:id` | Обновить тариф | Operator |
| DELETE | `/tariffs/:id` | Архивировать тариф | Operator |
| POST | `/tariffs/:id/activate` | Активировать тариф | Operator |
| GET | `/cdr` | Список CDR | Operator |
| GET | `/cdr/export` | Экспорт CDR | Operator |
| GET | `/reports/revenue` | Revenue отчёт | Operator |
| GET | `/reports/usage` | Usage отчёт (stub) | Operator |
| GET | `/sessions` | Active sessions | Operator |
| GET | `/balance-history` | История баланса | Operator |
| GET | `/translations/:locale` | Переводы | Публичный |

### SelfCare Gateway (`:18081`)

| Method | Endpoint | Описание | Auth |
|--------|----------|----------|------|
| POST | `/auth/login` | Логин по MSISDN+PIN | Публичный |
| POST | `/auth/refresh` | Refresh token | Refresh token |
| GET | `/profile` | Профиль абонента | SelfCare |
| PUT | `/profile` | Обновить профиль | SelfCare |
| GET | `/balance` | Баланс | SelfCare |
| POST | `/topup` | Пополнение (stub) | SelfCare |
| GET | `/cdr` | CDR абонента | SelfCare |
| POST | `/pin/change` | Смена PIN | SelfCare |
| GET | `/translations/:locale` | Переводы | Публичный |

---

## База данных

### ER Diagram (основные сущности)

```
┌─────────────────────┐     ┌─────────────────────┐     ┌─────────────────────┐
│      tenants        │     │       users         │     │       roles         │
├─────────────────────┤     ├─────────────────────┤     ├─────────────────────┤
│ id (PK)             │◄────┤ tenant_id (FK)      │     │ id (PK)             │
│ name                │     │ id (PK)             │     │ tenant_id (FK)      │
│ domain              │     │ email               │     │ name                │
│ config_json         │     │ password_hash       │     │ permissions (JSONB) │
│ branding_config(FK) │     │ role_id (FK)        │────►│ is_system           │
│ status              │     │ mfa_secret          │     │ status              │
│ created_at          │     │ mfa_enabled         │     └─────────────────────┘
└─────────────────────┘     │ mfa_backup_codes    │
                            │ last_login_at       │
                            │ failed_login_count  │
                            │ locked_until        │
                            │ status              │
                            └─────────────────────┘
                                      │
                                      │
┌─────────────────────┐     ┌────────┴────────────┐     ┌─────────────────────┐
│subscriber_credentials│     │    subscribers      │     │   tariff_sandbox    │
├─────────────────────┤     ├─────────────────────┤     ├─────────────────────┤
│ id (PK)             │◄────┤ id (PK)             │     │ id (PK)             │
│ subscriber_id (FK)  │     │ tenant_id (FK)      │◄────┤ tenant_id (FK)      │
│ msisdn              │     │ msisdn              │     │ name                │
│ pin_hash            │     │ imsi                │     │ config_json         │
│ email               │     │ email               │     │ cgrates_tp_id       │
│ category            │     │ category            │     │ status              │
│ balance_frozen_at   │     │ tariff_plan_id (FK) │────►│ created_at          │
│ status              │     │ status              │     │ updated_at          │
│ created_at          │     │ created_at          │     └─────────────────────┘
│ updated_at          │     │ updated_at          │
└─────────────────────┘     └─────────────────────┘
          │
          │
┌─────────┴─────────────┐
│    balance_history    │
├───────────────────────┤
│ id (PK)               │
│ tenant_id (FK)        │
│ subscriber_id (FK)    │
│ operation             │  -- topup, charge, adjust, expire, transfer, freeze, unfreeze
│ amount_before         │
│ amount_after          │
│ currency              │
│ reference             │
│ metadata (JSONB)      │
│ created_at            │
└───────────────────────┘
```

### Миграции

| Файл | Описание |
|------|----------|
| `001_initial_schema.sql` | Базовая схема: tenants, users, roles, subscribers, balance_history |
| `002_test_accounts.sql` | Seed данные: тестовые аккаунты, переводы |
| `003_balance_freeze.sql` | `balance_frozen_at`, расширение `balance_history.operation` |
| `004_cdr_exports.sql` | Таблица `cdr_exports` |
| `005_translations_ui.sql` | UI переводы (nav, buttons, forms) |

---

## Аутентификация и тестовые данные

### Admin / Operator

| Поле | Значение |
|------|----------|
| Email | `admin@test.com` |
| Password | `TestPass123!` |
| MFA Code | `123456` (dev mode) |

### SelfCare

| Поле | Значение |
|------|----------|
| MSISDN | `79161234567` |
| PIN | `1234` |

### Auth Flow

```
1. POST /auth/login
   → { email, password }
   ← { mfa_required: true, temp_token }

2. POST /auth/mfa/verify
   → { temp_token, code }
   ← { access_token, refresh_token, expires_in }

3. Используем access_token в Authorization: Bearer header

4. POST /auth/refresh
   → { refresh_token }
   ← { access_token, refresh_token } (rotation)
```

---

## Workers и Event Pipeline

### Pulsar Topics

| Namespace | Topic | Partitions | Назначение |
|-----------|-------|------------|------------|
| `billing/events` | `charges` | 6 | События списаний |
| `billing/events` | `topups` | 6 | События пополнений |
| `billing/events` | `sessions.started` | 6 | Начало сессии |
| `billing/events` | `sessions.ended` | 6 | Окончание сессии |
| `billing/events` | `cdr.raw` | 12 | Сырые CDR |
| `billing/audit` | `audit.events` | 3 | Audit события |
| `billing/audit` | `audit.security` | 3 | Security события |
| `billing/notifications` | `email` | 3 | Email уведомления |
| `billing/notifications` | `sms` | 3 | SMS уведомления |
| `billing/notifications` | `push` | 3 | Push уведомления |
| `billing/commands` | `balance.adjust` | 3 | Команды корректировки |
| `billing/commands` | `tariff.update` | 3 | Команды обновления тарифа |
| `billing/config` | `config.changes` | 1 | Изменения конфига |

### Workers

| Worker | Статус | Назначение |
|--------|--------|------------|
| `audit-consumer` | ⏳ Собирается | Постоянная запись audit events из Pulsar в `audit_log` |
| `cdr-processor` | ⏳ Собирается | Агрегация CDR, обновление materialized views |
| `email-consumer` | ⏳ Собирается | Отправка transactional emails через SMTP |
| `sms-consumer` | ⏳ Собирается | SMS-уведомления |
| `balance-monitor` | ⏳ Собирается | Алерты при низком балансе |
| `cache-invalidator` | ⏳ Собирается | Инвалидация Redis по событиям |

---

## Переменные окружения

### Обязательные

| Переменная | Описание | Default |
|-----------|----------|---------|
| `POSTGRES_HOST` | PostgreSQL host | `localhost` |
| `POSTGRES_PORT` | PostgreSQL port | `5432` |
| `POSTGRES_DB` | Database name | `uibill` |
| `POSTGRES_USER` | Database user | `postgres` |
| `POSTGRES_PASSWORD` | Database password | `postgres` |
| `REDIS_HOST` | Redis host | `localhost` |
| `REDIS_PORT` | Redis port | `6379` |
| `PULSAR_URL` | Pulsar broker URL | `pulsar://localhost:6650` |
| `JWT_SECRET_*` | JWT secrets (3 шт.) | — |

### MinIO / S3

| Переменная | Описание | Default |
|-----------|----------|---------|
| `MINIO_ENDPOINT` | MinIO endpoint | `http://localhost:9000` |
| `MINIO_ACCESS_KEY` | Access key | `minioadmin` |
| `MINIO_SECRET_KEY` | Secret key | `minioadmin` |
| `MINIO_BUCKET` | Bucket name | `uibill` |
| `MINIO_USE_SSL` | Use SSL | `false` |

### SMTP (Email Worker)

| Переменная | Описание |
|-----------|----------|
| `SMTP_HOST` | SMTP server host |
| `SMTP_PORT` | SMTP server port |
| `SMTP_USER` | SMTP username |
| `SMTP_PASSWORD` | SMTP password |
| `SMTP_FROM` | From address |

См. полный список в `.env.example`.

---

## Roadmap

| Phase | Название | Статус | Оценка |
|-------|----------|--------|--------|
| 1 | Foundation Hardening | ✅ Готово | ~1 неделя |
| 2A | Backend: Stubs → Real | ✅ Готово | ~1.5 недели |
| 2B | Frontend Gaps | ✅ Готово | ~1 неделя |
| 2B+ | Localization (механизм + seeds) | ✅ Готово | ~0.5 недели |
| **2.5** | **Полная локализация UI** | ⏳ **Следующая** | **~0.5 недели** |
| 3 | Workers & Event Pipeline | ⏳ Ожидает | ~1.5 недели |
| 4 | Operator BSS (остатки) | ⏳ Ожидает | ~1.5 недели |
| 5 | SelfCare Portal | ⏳ Ожидает | ~1.5 недели |
| 6 | Admin OSS | ⏳ Ожидает | ~1 неделя |
| 7 | Advanced Features | ⏳ Ожидает | ~2.5 недели |

Подробный план: [`docs/DEVELOPMENT_PLAN.md`](docs/DEVELOPMENT_PLAN.md)

---

## Разработка

### Backend

```bash
cd backend
go mod download
go test ./...
```

### Frontend

```bash
cd frontend/admin-ui   # или operator-ui / selfcare-ui
npm install
npm run dev
npm run build
```

### База данных

```bash
# Применить миграции (внутри контейнера)
docker-compose exec postgres psql -U postgres -d uibill -f /docker-entrypoint-initdb.d/migrations/001_initial_schema.sql
```

---

## Лицензия

MIT License — см. [LICENSE](LICENSE) файл.

---

## Контакты

- **Repository:** https://github.com/Grimid86/cgrates-ui
- **Issues:** https://github.com/Grimid86/cgrates-ui/issues
