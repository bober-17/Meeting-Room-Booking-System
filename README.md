# Meeting Room Booking System

Корпоративная система бронирования переговорок с уведомлениями в реальном времени.  

Администраторы создают переговорки и задают расписание доступности. Система автоматически
генерирует 30-минутные слоты. Сотрудники бронируют слоты — при создании брони опционально
генерируется ссылка на видеоконференцию. После каждого события бронирования пользователь
получает мгновенное push-уведомление через SSE.

---

## Сервисы

| Сервис | Порт | README |
|--------|------|--------|
| **booking-service** | `8080` | [booking-service/README.md](booking-service/README.md) |
| **notification-service** | `8081` | [notification-service/README.md](notification-service/README.md) |

---

## Архитектура

```
┌──────────────────────────────────────────────────────────────┐
│                         Client                               │
│     REST (JWT)                              SSE stream       │
└─────────┬────────────────────────────────────────┬───────────┘
          │                                        │
          ▼                                        ▼
┌───────────────────┐                  ┌───────────────────────┐
│   booking-service │                  │ notification-service  │
│   :8080           │                  │ :8081                 │
│                   │                  │                       │
│  Handler          │                  │  Handler              │
│  Service          │                  │  Service              │
│  Repository       │                  │  Repository           │
│  Outbox Relay ────┼────► Kafka ──────┼► Kafka Consumer       │
│       │           │  booking.events  │       │               │
│       ▼           │                  │       ▼               │
│  PostgreSQL       │                  │  PostgreSQL           │
│  (booking-db)     │                  │  (notification-db)    │
└───────────────────┘                  └───────────────────────┘
```

### Поток события бронирования

```
POST /bookings/create
        │
        ▼
  Booking saved          ← транзакция #1
  Outbox record saved    ← та же транзакция (atomicity)
        │
        ▼
  Outbox Relay           ← фоновая горутина, FOR UPDATE SKIP LOCKED
  publishes to Kafka
        │
        ▼
  Kafka: booking.events
        │
        ▼
  Kafka Consumer         ← notification-service
  (at-least-once)
        │
        ▼
  INSERT notification    ← ON CONFLICT DO NOTHING (идемпотентность)
        │
        ▼
  SSE Hub.Broadcast()    ← мгновенная доставка подключённым клиентам 
```

---

## Стек

| Компонент | Выбор |
|-----------|-------|
| Язык | Go 1.25, Go Workspaces |
| HTTP-роутер | [chi v5](https://github.com/go-chi/chi) |
| База данных | PostgreSQL 15 |
| Драйвер БД | [pgx/v5](https://github.com/jackc/pgx) |
| Миграции | [golang-migrate](https://github.com/golang-migrate/migrate) |
| Брокер | Kafka (KRaft, без ZooKeeper) |
| Kafka-клиент | [segmentio/kafka-go](https://github.com/segmentio/kafka-go) |
| Push-уведомления | SSE (Server-Sent Events) |
| Аутентификация | JWT (HS256) |
| Конфигурация | `os.Getenv`, `mustEnv`/`optEnv` |
| Логирование | `slog` |
| Тесты | [testify](https://github.com/stretchr/testify) + [mockery](https://github.com/vektra/mockery) |
| Линтер | [golangci-lint](https://golangci-lint.run) |
| Нагрузочные тесты | [k6](https://k6.io) |

---

## Требования

- [Docker](https://docs.docker.com/get-docker/) + [Docker Compose](https://docs.docker.com/compose/) v2
- [Go 1.25+](https://go.dev/dl/) — только для локальной разработки и тестов
- [golangci-lint](https://golangci-lint.run/usage/install/) — для `make lint`

---

## Быстрый старт

```bash
# 1. Клонировать и перейти в директорию
git clone <repo-url>
cd meeting-room-booking-system

# 2. Создать .env из шаблона и заполнить обязательные переменные
cp .env.example .env

# 3. Запустить всю инфраструктуру
make up

# 4. Загрузить тестовые данные (переговорки + расписания + системные пользователи)
make seed
```

После старта:

| Сервис | URL |
|--------|-----|
| Booking API | http://localhost:8080 |
| Notification API | http://localhost:8081 |
| Kafka UI | http://localhost:8090 |
| Prometheus | http://localhost:9090 |
| Grafana | http://localhost:3000 (admin / admin) |

Остановить и удалить данные:

```bash
make down        # остановить, данные сохранить
make down-clean  # остановить и удалить все volumes
```

---

## Переменные окружения

Скопируйте `.env.example` в `.env`. Перед деплоем обязательно замените значения по умолчанию.

**Обязательные** (сервис не запустится без них):

| Переменная | Кто использует |
|------------|----------------|
| `BOOKING_DATABASE_HOST` / `_USER` / `_PASSWORD` / `_NAME` | booking-service |
| `NOTIFICATION_DATABASE_HOST` / `_USER` / `_PASSWORD` / `_NAME` | notification-service |
| `JWT_SECRET` | оба сервиса (общий секрет) |
| `KAFKA_BROKERS` | оба сервиса |

**Опциональные**:

| Переменная | Дефолт | Описание |
|------------|--------|----------|
| `BOOKING_SERVER_PORT` | `8080` | Порт booking-service |
| `NOTIFICATION_SERVER_PORT` | `8081` | Порт notification-service |
| `BOOKING_DATABASE_PORT` / `NOTIFICATION_DATABASE_PORT` | `5432` | Порт PostgreSQL |
| `KAFKA_TOPIC_BOOKING_EVENTS` | `booking.events` | Топик событий |
| `KAFKA_GROUP_ID` | `notification-service` | Consumer group |
| `KAFKA_EXTERNAL_PORT` | `9094` | Внешний порт Kafka |
| `KAFKA_UI_PORT` | `8090` | Порт Kafka UI |

---

## Сквозной сценарий

Пример полного цикла: получить токен → найти свободный слот → забронировать → получить уведомление.

**1. Получить JWT**

```bash
TOKEN=$(curl -s -X POST http://localhost:8080/dummyLogin \
  -H "Content-Type: application/json" \
  -d '{"role":"user"}' | jq -r .token)
```

**2. Получить свободные слоты** (Переговорная 2 работает каждый день)

```bash
curl -s "http://localhost:8080/rooms/10000000-0000-0000-0000-000000000002/slots/list?date=$(date +%Y-%m-%d --date='+1 day')" \
  -H "Authorization: Bearer $TOKEN" | jq .
```

**3. Создать бронь**

```bash
SLOT_ID="<id из шага 2>"

curl -s -X POST http://localhost:8080/bookings/create \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"slotId\":\"$SLOT_ID\",\"createConferenceLink\":true}" | jq .
```

**4. Подключиться к SSE-стриму уведомлений**

```bash
# Сначала получить одноразовый токен (EventSource не поддерживает кастомные заголовки)
SSE_TOKEN=$(curl -s -X POST http://localhost:8081/sse-token \
  -H "Authorization: Bearer $TOKEN" | jq -r .token)

# Подписаться на поток
curl -N "http://localhost:8081/notifications/stream?token=$SSE_TOKEN"
```

---

## Структура монорепо

```
.
├── booking-service/          — сервис бронирования
│   ├── cmd/                  — точка входа
│   ├── internal/             — вся бизнес-логика
│   ├── migrations/           — схема БД (golang-migrate)
│   ├── e2e/                  — E2E тесты
│   ├── loadtest/             — k6 скрипт и результаты
│   ├── scripts/              — seed.sql
│   ├── api.yaml              — OpenAPI-спецификация
│   ├── Dockerfile
│   └── docker-compose.yaml   — standalone-запуск сервиса
│
├── notification-service/     — сервис уведомлений
│   ├── cmd/                  — точка входа
│   ├── internal/             — Kafka consumer, REST API, SSE Hub
│   ├── migrations/           — схема БД
│   ├── Dockerfile
│   └── docker-compose.yaml   — standalone-запуск сервиса
│
├── shared/
│   └── events/               — общий Go-модуль: BookingEvent (схема Kafka-сообщения)
│
├── e2e/                      — системные E2E тесты (build tag: e2e_system)
│
├── docker-compose.yaml       — корневой compose: Kafka, Kafka UI + include сервисов
├── .env.example              — шаблон переменных окружения
├── Makefile                  — команды для всего монорепо
└── go.work                   — Go Workspace (booking-service, notification-service, shared/events, e2e)
```

Go Workspace позволяет работать с модулями локально без `replace`-директив: изменения в `shared/events` сразу видны в обоих сервисах.

---

## Команды

### Инфраструктура

| Команда | Описание |
|---------|----------|
| `make up` | Поднять всю инфраструктуру (все сервисы + Kafka) |
| `make down` | Остановить, данные сохранить |
| `make down-clean` | Остановить и удалить все volumes |
| `make seed` | Загрузить тестовые данные в booking-service |

### booking-service

| Команда | Описание |
|---------|----------|
| `make booking-test` | Юнит-тесты + итоговый процент покрытия |
| `make booking-test-integration` | Интеграционные тесты репозиториев (поднимает PostgreSQL на :5433) |
| `make booking-test-e2e` | E2E тесты (поднимает PostgreSQL на :5434) |
| `make booking-mock` | Перегенерировать моки через mockery |
| `make booking-load-test` | Нагрузочный тест k6 (требует `make up && make seed`) |

### notification-service

| Команда | Описание |
|---------|----------|
| `make notification-test-integration` | Интеграционные тесты репозиториев (поднимает PostgreSQL на :5435) |

### Системные тесты

| Команда | Описание |
|---------|----------|
| `make test-e2e-system` | E2E тесты всей системы (требует `make up`) |
| `make load-test-system` | Системный нагрузочный тест k6: 80 RPS reads + 20 RPS writes + 500 SSE (требует `make up && make seed`) |

### Качество кода

| Команда | Описание |
|---------|----------|
| `make lint` | golangci-lint по всем сервисам |

---

## Тестирование

Три уровня по каждому сервису:

| Уровень | Что проверяет | Build tag |
|---------|---------------|-----------|
| Юнит | Бизнес-логика, JWT, middleware | — |
| Интеграционный | SQL-запросы репозиториев на реальной БД | `integration` |
| E2E | Сквозные HTTP-сценарии | `e2e` |

Интеграционные и E2E тесты поднимают изолированный PostgreSQL-контейнер и сносят его после завершения. `go test ./...` без тегов запускает только юнит-тесты.

---

## Мониторинг

Стек: **Prometheus** (сбор метрик) + **Grafana** (визуализация). Поднимаются вместе с `make up`, дашборд доступен сразу без ручной настройки.

| Сервис | URL |
|--------|-----|
| Prometheus | http://localhost:9090 |
| Grafana | http://localhost:3000 (admin / admin) |

### Метрики

Каждая метрика отвечает на конкретный вопрос о системе.

| Метрика | Тип | Сервис | Вопрос |
|---------|-----|--------|--------|
| `http_request_duration_seconds{path, status}` | Histogram | оба | Выполняется ли SLA p95 < 200ms по каждому endpoint? |
| `booking_slot_conflicts_total` | Counter | booking | Сколько раз partial unique index отбил гонку? Доказательство корректности под нагрузкой |
| `booking_outbox_published_total` | Counter | booking | Relay работает? Должен расти вместе с бронями 1:1 |
| `booking_outbox_errors_total` | Counter | booking | Есть ли потери в Kafka pipeline? |
| `notification_delivery_latency_seconds` | Histogram | notification | E2E от события до SSE-broadcast; показывает вклад outbox poll interval |
| `notification_sse_connections_active` | Gauge | notification | Текущее число SSE-соединений на Hub |
| `notification_kafka_consumer_errors_total` | Counter | notification | Consumer падает? Любое ненулевое значение — инцидент |

### Дашборд

Три секции:
- **HTTP SLA** — p95 latency и error rate по endpoint обоих сервисов
- **Async pipeline** — outbox published rate, E2E delivery latency (p50 / p95), SSE connections
- **Correctness** — slot conflicts rate под нагрузкой

---

## Производительность и требования

Ключевые числа, которые определили архитектурные решения. Полное обоснование — в [REQUIREMENTS.md](REQUIREMENTS.md).

### Объём данных

| Сущность | Максимум |
|----------|----------|
| Переговорки | 50 |
| Слотов в день | 1 000 |
| Пользователей | 10 000 |
| Броней | 100 000 |

### Нагрузка

| Метрика | Значение |
|---------|----------|
| Суммарный RPS (booking-service) | 100 |
| Read / Write | 80 / 20 |
| Пиковых событий в Kafka | ~20/сек |
| Concurrent SSE-соединений | ~100 |

### SLA

| Метрика | Цель |
|---------|------|
| Availability booking-service | 99.9% |
| Latency `GET /slots` p95 | < 200ms |
| Delivery latency уведомления p95 | < 3s |

### Гарантии доставки

**At-least-once** через Outbox pattern:
- событие записывается в `outbox` в одной транзакции с бронью — потеря исключена
- Kafka consumer commit-ит offset после сохранения в БД — дубли возможны при рестарте
- `UNIQUE (booking_id, type)` + `ON CONFLICT DO NOTHING` — дубли идемпотентно игнорируются

---

## Разработка

### Добавить новый сервис

1. Создать директорию `my-service/` с `go.mod` (`module github.com/bober-17/meeting-room-booking-system/my-service`)
2. Добавить `./my-service` в `go.work`
3. Создать `my-service/docker-compose.yaml` и добавить `include:` в корневой `docker-compose.yaml`
4. Добавить команды в `Makefile` с префиксом `my-service-`

### Изменить контракт Kafka

Схема события живёт в `shared/events/events.go`. Оба сервиса используют этот модуль через Go Workspace — изменения применяются без пересборки зависимостей.

### Локальный запуск без Docker

Сервисные compose не пробрасывают порты БД на хост. Для запуска бинарника локально
нужна отдельная PostgreSQL (например, системная установка или `docker run` с `-p 5432:5432`).

```bash
# booking-service
cd booking-service
BOOKING_DATABASE_HOST=localhost \
BOOKING_DATABASE_USER=postgres \
BOOKING_DATABASE_PASSWORD=password \
BOOKING_DATABASE_NAME=booking \
JWT_SECRET=supersecretkey \
KAFKA_BROKERS=localhost:9094 \
  go run ./cmd/main.go

# notification-service (в отдельном терминале)
cd notification-service
NOTIFICATION_DATABASE_HOST=localhost \
NOTIFICATION_DATABASE_USER=postgres \
NOTIFICATION_DATABASE_PASSWORD=password \
NOTIFICATION_DATABASE_NAME=notifications \
JWT_SECRET=supersecretkey \
KAFKA_BROKERS=localhost:9094 \
  go run ./cmd/main.go
```
