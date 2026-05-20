# UI-Bill: План функционального наполнения

## Сводка текущего состояния (что уже работает)

### Инфраструктура ✅
- Docker Compose со всем стеком (PostgreSQL, Redis, Pulsar, MinIO, CGRateS mock)
- Go 1.23 + Echo — 3 gateway (admin, operator, selfcare) компилируются и запущены
- React 18 + Vite + TailwindCSS — 3 UI-сборки собираются без ошибок
- PostgreSQL миграции + seed-данные (тестовые аккаунты)
- JWT-аутентификация + RBAC + rate limiting + CSRF + audit middleware
- White-label branding + i18n backend (ru/en/es)

### Phase 1 — Foundation Hardening ✅ (Выполнено)
- **MFA/TOTP** — реальная генерация секрета, верификация, backup codes, AES-шифрование
- **Idempotency middleware** — реальная проверка `rpc_idempotency_keys`
- **Refresh token rotation + revocation** — Redis blacklist, rotation on refresh
- **Account lockout** — N неудачных попыток → блокировка, frontend auto-refresh

### Phase 2A — Backend Stubs → Real Implementation ✅ (Выполнено)
- **MinIO Logo Upload** — `POST /branding/logo`, валидация, upload, DB update, Pulsar event
- **Tariff CRUD** — GetTariff, UpdateTariff, DeleteTariff (soft delete)
- **Balance Freeze/Unfreeze** — `balance_frozen_at`, freeze/unfreeze operations + CGRateS
- **CDR Export** — CSV/JSON генерация, upload в MinIO, `cdr_exports` таблица
- **Revenue Report** — SQL агрегация `balance_history` по дням
- **RBAC Role Detail** — `GET /rbac/roles/:id` с permissions JSONB

### Phase 2B — Frontend Gaps ✅ (Выполнено)
- **Sidebar Navigation** — collapsible, responsive, группировка по ролям, language switcher
- **Subscriber New** — форма создания абонента
- **TopUp Success** — страница подтверждения пополнения
- **Edit/Delete Flows** — inline edit/delete для Tenants, Users, Tariffs
- **Subscriber Actions** — Block/Unblock/Freeze/Unfreeze кнопки
- **I18nContext** — Admin/Operator UI, загрузка переводов с backend

### Частично работает / есть заглушки ⚠️
- **CGRateS интеграция** — mock-сервис отдаёт статику; реальные RPC-вызовы не подключены
- **Workers** — 6 микросервисов написаны, но не собраны в Docker Compose и не запущены
- **Email/SMS** — шаблоны в БД есть, отправка не реализована
- **Monitoring** — `/metrics`, `/pulsar/lag`, `/database/status` — заглушки
- **Локализация UI** — механизм работает, но много hardcoded строк (см. Phase 2.5)

---

## Фаза 2.5 — Полная локализация UI (0.5–1 неделя) ⏳ Следующая итерация

### 2.5.1 Сборка ключей
| Задача | Где | Объём |
|--------|-----|-------|
| Просканировать все JSX на hardcoded строки | `frontend/*-ui/src/**/*.jsx` | ~40–60 компонентов |
| Выделить общие категории: `common`, `nav`, `forms`, `tables`, `buttons`, `errors`, `balance`, `cdr`, `tariffs`, `subscribers`, `reports` | — | ~200–300 ключей |

### 2.5.2 Backend: управление переводами
| Задача | Где |
|--------|-----|
| `POST /translations` — bulk upsert переводов | `admin-gateway` |
| `DELETE /translations/:key` — удаление ключа | `admin-gateway` |
| `GET /translations/export?format=csv|json` — экспорт | `admin-gateway` |
| `POST /translations/import` — импорт CSV/JSON | `admin-gateway` |

### 2.5.3 Frontend: полное покрытие t()
| Задача | Где |
|--------|-----|
| Admin UI — все страницы (Tenants, Users, Roles, Branding, Audit) | `frontend/admin-ui/src/pages/*.jsx` |
| Operator UI — все страницы (Subscribers, Tariffs, CDR, Reports, Sessions) | `frontend/operator-ui/src/pages/*.jsx` |
| SelfCare UI — все страницы (Profile, TopUp, History, CDR) | `frontend/selfcare-ui/src/pages/*.jsx` |
| Shared components — Table, Modal, Pagination, Form inputs | `frontend/shared-components/*.jsx` |
| Добавить fallback: если ключ не найден → `[key]` (для дебага) | `I18nContext` |

### 2.5.4 Переводы
| Язык | Ключи | Статус |
|------|-------|--------|
| EN (base) | ~250 | ⏳ Сгенерировать из кода |
| RU | ~250 | ⏳ Перевести |
| ES | ~250 | ⏳ Перевести |

---

## Фаза 3 — Event Pipeline & Workers (1.5–2 недели)

### 3.1 Workers запуск
| Задача | Где | Приоритет |
|--------|-----|-----------|
| Dockerfile для каждого worker + docker-compose | `workers/*/Dockerfile` | 🔴 Высокий |
| Audit Consumer — пишет audit events из Pulsar в PostgreSQL | `workers/audit-consumer/` | 🔴 Высокий |
| Email Consumer — отправка email через SMTP/SES | `workers/email-consumer/` | 🟡 Средний |
| SMS Consumer — SMS-уведомления | `workers/sms-consumer/` | 🟡 Средний |
| Cache Invalidator — слушает `config.changes`, чистит Redis | `workers/cache-invalidator/` | 🟢 Низкий |
| CDR Processor — агрегация hourly/daily stats | `workers/cdr-processor/` | 🟡 Средний |
| Balance Monitor — алерты при низком балансе | `workers/balance-monitor/` | 🟢 Низкий |

### 3.2 Pulsar publishing во всех mutating handlers
| Задача | Где |
|--------|-----|
| Tenant CRUD → `tenant.created/updated/deleted` | `admin-gateway` |
| User CRUD/Reset → `user.updated/reset_password` | `admin-gateway` |
| Subscriber CRUD/Block/Freeze → `subscriber.*` | `operator-gateway` |
| Balance operations → `balance.topup/charge/freeze` | `operator-gateway` / `selfcare-gateway` |
| Tariff publish → `tariff.published` | `operator-gateway` |
| DLQ (Dead Letter Queue) — retry + poison messages | `pkg/pulsar/` |

### 3.3 Real Email / SMS
| Задача | Где |
|--------|-----|
| SMTP provider integration (SendGrid/SES/localhost) | `backend/pkg/email/` |
| SMS provider integration (Twilio/SMSC) | `backend/pkg/sms/` |
| HTML email templates (welcome, reset, invoice, alert) | `workers/email-consumer/templates/` |
| Email preview endpoint (`POST /admin/email/preview`) | `admin-gateway` |

---

## Фаза 4. Operator BSS — оставшийся функционал (1.5–2 недели)

### 4.1 Абоненты
| Задача | Статус | Что нужно |
|--------|--------|-----------|
| Список / поиск / фильтрация | ✅ Есть | Дополнить фильтрами (тариф, статус, дата) |
| Карточка абонента (детали) | ✅ Есть | Добавить историю баланса, CDR, сессии |
| Создание / редактирование абонента | ✅ Есть | Подключить реальный CGRateS `SetAccount` |
| Блокировка / разблокировка | ✅ Есть | Подключить CGRateS |
| Миграция тарифа | ⚠️ TODO | Реальный вызов `SetTPRatingPlan` + нотификация |
| Balance history страница | ❌ Нет | Таблица + фильтры + pagination |

### 4.2 Управление балансом
| Задача | Статус | Что нужно |
|--------|--------|-----------|
| Корректировка баланса (credit/debit) | ✅ Есть | Подключить `ApierV1.AddBalance` / `DebitBalance` |
| Freeze / Unfreeze баланса | ✅ Есть | Реализовано через БД + CGRateS mock |
| Bulk bonus (массовое начисление) | ✅ Есть | Очередь через Pulsar + worker |
| История операций по балансу | ⚠️ TODO | Frontend страница (API готово) |

### 4.3 CDR & Отчёты
| Задача | Статус | Что нужно |
|--------|--------|-----------|
| CDR список с фильтрами | ✅ Есть | Добавить агрегацию по типу |
| Экспорт CDR (CSV/XLSX) | ✅ Есть | API готово, можно расширить async worker |
| Отчёт по usage | ⚠️ Stub | Агрегация из CDR + графики |
| Отчёт по revenue | ✅ Есть | Работает, можно добавить графики |

### 4.4 Сессии
| Задача | Статус | Что нужно |
|--------|--------|-----------|
| Active sessions (real-time) | ✅ Есть | Подключить `SMGv1.GetActiveSessions` |
| Детализация сессии | ❌ Нет | Модальное окно с трафиком |
| Принудительное отключение (kick) | ❌ Нет | CGRateS `SMGv1.DisconnectSession` |

---

## Фаза 5. SelfCare Portal — полноценный личный кабинет (1.5–2 недели)

### 5.1 Авторизация
| Задача | Статус | Что нужно |
|--------|--------|-----------|
| Логин по MSISDN + PIN | ✅ Есть | Captcha (hcaptcha/google) |
| Логин по email + password | ❌ Нет | Альтернативный метод входа |
| Восстановление PIN | ❌ Нет | SMS/Email OTP |
| Регистрация нового абонента | ❌ Нет | Форма + OTP на номер |

### 5.2 Баланс
| Задача | Статус | Что нужно |
|--------|--------|-----------|
| Просмотр балансов | ✅ Есть | Добавить прогресс-бары для пакетов |
| История пополнений / списаний | ⚠️ TODO | Страница + API (реализовано в Phase 2A) |
| Автопополнение (подписка) | ❌ Нет | Сохранение карты + триггеры |

### 5.3 Пополнение (Top-Up)
| Задача | Статус | Что нужно |
|--------|--------|-----------|
| Форма пополнения | ✅ Есть | Интеграция платёжного провайдера (Stripe/YooKassa) |
| История платежей | ❌ Нет | Таблица `payment_transactions` |
| Чек / квитанция | ❌ Нет | PDF-генерация |

### 5.4 Профиль
| Задача | Статус | Что нужно |
|--------|--------|-----------|
| Просмотр профиля | ✅ Есть | Добавить tariff_plan details |
| Редактирование email, уведомлений | ⚠️ TODO | Сохранение в БД + CGRateS sync |
| Смена PIN | ✅ Есть | Добавить валидацию старого PIN |
| Управление сессиями / устройствами | ✅ Есть | Улучшить UI (иконки устройств, география) |

### 5.5 CDR
| Задача | Статус | Что нужно |
|--------|--------|-----------|
| Детализация звонков | ✅ Есть | Фильтры по типу, дате, направлению |
| Детализация интернета | ❌ Нет | Сессии с объёмом трафика |
| Экспорт детализации | ❌ Нет | PDF/CSV |

---

## Фаза 6. Admin OSS — управление платформой (1–1.5 недели)

### 6.1 Тенанты
| Задача | Статус | Что нужно |
|--------|--------|-----------|
| CRUD тенантов | ✅ Есть | Лимиты (max_subscribers, max_staff, rate_limit) |
| Suspend / Activate | ✅ Есть | Каскадная блокировка |
| Domain management (white-label domains) | ❌ Нет | CRUD `domain_tenant_mapping` + SSL |
| Tenant onboarding wizard | ❌ Нет | Пошаговая настройка |

### 6.2 Пользователи и RBAC
| Задача | Статус | Что нужно |
|--------|--------|-----------|
| CRUD staff users | ✅ Есть | Фильтры, массовые операции |
| Roles & Permissions | ✅ Есть | Drag-and-drop матрица прав |
| Reset password / Reset MFA | ✅ Есть | Email-нотификация |
| Four-eyes principle | ❌ Нет | Одобрение критических операций вторым админом |

### 6.3 White Labeling
| Задача | Статус | Что нужно |
|--------|--------|-----------|
| Редактор брендинга (цвета, тексты) | ✅ Есть | Live preview |
| Загрузка логотипа / favicon | ✅ Есть | MinIO upload + CDN URL |
| Email-шаблоны | ⚠️ Stub | Редактор + preview с переменными |
| Custom CSS | ❌ Нет | Дополнительные CSS-переменные |

### 6.4 Локализация
| Задача | Статус | Что нужно |
|--------|--------|-----------|
| Управление языками | ✅ Есть | Активация/деактивация |
| Редактор переводов | ✅ Есть | Inline editing + поиск |
| Импорт / экспорт CSV/JSON | ⚠️ Stub | Bulk upload translations (см. Phase 2.5) |

### 6.5 Мониторинг & API Keys
| Задача | Статус | Что нужно |
|--------|--------|-----------|
| Health dashboard | ✅ Есть | Добавить статусы компонентов |
| Audit Log | ✅ Есть | Фильтры, экспорт |
| Prometheus metrics | ⚠️ Stub | Реальные метрики (Grafana-ready) |
| Pulsar topics / lag | ⚠️ Stub | Pulsar Admin API |
| API Keys management | ✅ Backend | Frontend страница |

---

## Фаза 7. Advanced Features (2–3 недели)

### 7.1 WebSocket / Real-Time
| Задача | Где |
|--------|-----|
| WebSocket сервер (Echo) для balance updates | `selfcare-gateway` |
| WebSocket для active sessions monitor | `operator-gateway` |
| WebSocket для system alerts | `admin-gateway` |
| Frontend: reconnect + heartbeat | Все 3 UI |

### 7.2 CGRateS Deep Integration
| Задача | Где |
|--------|-----|
| Полноценный RPC-клиент с circuit breaker | `pkg/cgrates/` |
| Account creation → CGRateS `SetAccount` | `operator-gateway` |
| Balance query → `ApierV1.GetAccount` | `selfcare-gateway` / `operator-gateway` |
| CDR query → `ApierV1.GetCDRs` | Все gateway |
| Tariff publishing → `ApierV1.SetTPRatingPlan` | `operator-gateway` |
| Active sessions → `SMGv1.GetActiveSessions` | `operator-gateway` |

### 7.3 Analytics & Dashboards
| Задача | Где |
|--------|-----|
| Revenue dashboard (operator) | `operator-ui` |
| Churn prediction (базовый) | `workers/` + PostgreSQL |
| Network topology map (diameter peers) | `admin-ui` |
| Real-time traffic graphs | `operator-ui` + WebSocket |

### 7.4 Security Hardening
| Задача | Где |
|--------|-----|
| IP whitelist для admin/operator | Middleware |
| Geo-blocking | Middleware + GeoIP |
| WAF rules (SQLi, XSS) | `SanitizeMiddleware` + расширение |
| Security audit log (failed logins, suspicious activity) | `audit_log` + alerts |

---

## Приоритеты (рекомендация)

| Приоритет | Что делать | Бизнес-ценность |
|-----------|-----------|-----------------|
| **P0** | CGRateS balance/CDR integration, Workers (audit + CDR), Real email | Без этого система не production-ready |
| **P1** | Полная локализация, Top-Up платежи, Balance history UI, WebSocket | Критично для операторов и абонентов |
| **P2** | Analytics dashboard, API Keys UI, Custom CSS, Churn prediction | Конкурентное преимущество |
| **P3** | Auto-topup, Network topology, Geo-blocking | Advanced features |

---

## Текущий прогресс

| Фаза | Задача | Статус |
|------|--------|--------|
| Phase 1 | MFA/TOTP Backend | ✅ Готово |
| Phase 1 | MFA/TOTP Frontend | ✅ Готово |
| Phase 1 | Idempotency middleware | ✅ Готово |
| Phase 1 | Refresh token rotation | ✅ Готово |
| Phase 1 | Account lockout | ✅ Готово |
| Phase 2A | MinIO Logo Upload | ✅ Готово |
| Phase 2A | Tariff CRUD | ✅ Готово |
| Phase 2A | Balance Freeze/Unfreeze | ✅ Готово |
| Phase 2A | CDR Export | ✅ Готово |
| Phase 2A | Revenue Report | ✅ Готово |
| Phase 2A | RBAC Role Detail | ✅ Готово |
| Phase 2B | Sidebar Navigation | ✅ Готово |
| Phase 2B | Subscriber New Page | ✅ Готово |
| Phase 2B | TopUp Success Page | ✅ Готово |
| Phase 2B | Edit/Delete Flows | ✅ Готово |
| Phase 2B | Localization (I18nContext + seeds) | ✅ Готово |
| **Phase 2.5** | **Полная локализация UI** | ⏳ **Следующая итерация** |
| Phase 3 | Workers Docker build | ⏳ Ожидает |
| Phase 3 | Audit Consumer | ⏳ Ожидает |
| Phase 3 | Email Consumer | ⏳ Ожидает |
| Phase 3 | Pulsar DLQ | ⏳ Ожидает |
| Phase 4 | Balance history UI | ⏳ Ожидает |
| Phase 4 | Usage Report | ⏳ Ожидает |
| Phase 5 | Payment integration | ⏳ Ожидает |
| Phase 5 | PDF receipts | ⏳ Ожидает |
| Phase 7 | CGRateS Deep Integration | ⏳ Ожидает |

---

## Оценка трудозатрат (rough estimate)

| Фаза | Backend | Frontend | DevOps/QA | Итого |
|------|---------|----------|-----------|-------|
| Phase 2.5: Full i18n | 1 день | 3 дня | — | ~0.5 недели |
| Phase 3: Workers & Events | 4 дня | — | 2 дня | ~1.5 недели |
| Phase 4: Operator BSS remaining | 2 дня | 3 дня | 1 день | ~1.5 недели |
| Phase 5: SelfCare | 3 дня | 3 дня | 1 день | ~1.5 недели |
| Phase 6: Admin OSS | 2 дня | 2 дня | 1 день | ~1 неделя |
| Phase 7: Advanced | 5 дней | 4 дня | 2 дня | ~2.5 недели |
| **Итого (осталось)** | | | | **~8–9 недель (2 человека)** |
| **Всего проект** | | | | **~11 недель** |
