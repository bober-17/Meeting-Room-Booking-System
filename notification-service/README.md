# Notification Service

Микросервис реального времени для доставки уведомлений о бронированиях переговорок.

## Что делает

Подписывается на события из Kafka (booking.created, booking.cancelled), сохраняет уведомления в PostgreSQL и мгновенно доставляет их пользователям через SSE (Server-Sent Events). Предоставляет REST API для управления уведомлениями.

## Архитектура

```
Kafka (booking.events)
       ↓
  Consumer          читает события, вызывает service
       ↓
  NotificationService  маппинг события → уведомление, сохранение в БД
       ↓               после сохранения — broadcast в SSE Hub
  Repository        INSERT с ON CONFLICT DO NOTHING (идемпотентность)
       ↓
  PostgreSQL (notification-db)

SSE Hub (in-memory)
  ← Register/Unregister (при подключении/отключении клиента)
  ← Broadcast (из NotificationService после сохранения)
  → send chan (goroutine per client, не блокирует Broadcast)
```

Три слоя, интерфейс объявляется на стороне потребителя:

```
HTTP Handler / Kafka Consumer
    ↓  (интерфейс сервиса объявлен в handler-файле / consumer-файле)
NotificationService
    ↓  (интерфейс репо и хаба объявлен в service/notification/contract.go)
Repository + SSE Hub
    ↓
PostgreSQL
```

## API

| Метод | Путь | Auth | Описание |
|-------|------|------|----------|
| `POST` | `/sse-token` | JWT | Получить одноразовый токен для SSE (TTL 30s) |
| `GET` | `/notifications/stream?token=<tok>` | one-time token | SSE поток уведомлений |
| `GET` | `/notifications` | JWT | Список уведомлений с пагинацией |
| `POST` | `/notifications/{id}/read` | JWT | Отметить уведомление прочитанным |
| `POST` | `/notifications/read-all` | JWT | Отметить все уведомления прочитанными |

### Почему одноразовый токен для SSE

Browser EventSource API не позволяет устанавливать кастомные заголовки — передать JWT через `Authorization` невозможно. Решение: клиент сначала делает `POST /sse-token` (с JWT), получает короткоживущий токен и передаёт его в query-параметре при подключении к стриму.

### GET /notifications

Query params: `limit` (default 20, max 100), `offset` (default 0), `unread_only` (bool).

```json
{
  "notifications": [
    {
      "id": "uuid",
      "type": "booking.created",
      "booking_id": "uuid",
      "room_name": "Переговорная 1",
      "slot_start": "2026-06-30T09:00:00Z",
      "slot_end": "2026-06-30T09:30:00Z",
      "is_read": false,
      "created_at": "2026-06-29T14:11:43Z"
    }
  ],
  "total": 42
}
```

### SSE формат события

```
data: {"id":"uuid","type":"booking.created","room_name":"Переговорная 1","slot_start":"...","is_read":false}

```

## Модель данных

```sql
CREATE TABLE IF NOT EXISTS notifications (
    id         UUID PRIMARY KEY,
    user_id    UUID NOT NULL,
    type       TEXT NOT NULL CHECK (type IN ('booking.created', 'booking.cancelled')),
    booking_id UUID NOT NULL,
    room_name  TEXT NOT NULL,
    slot_start TIMESTAMPTZ NOT NULL,
    slot_end   TIMESTAMPTZ NOT NULL,
    is_read    BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (booking_id, type)     -- идемпотентность consumer-а
);

CREATE INDEX IF NOT EXISTS idx_notifications_user_created
    ON notifications (user_id, created_at DESC);
```

Индекс `(user_id, created_at DESC)` покрывает `GET /notifications` — выборка по пользователю, сортировка по убыванию даты.

`UNIQUE (booking_id, type)` — один тип события на одну бронь. При повторной доставке из Kafka `ON CONFLICT DO NOTHING` делает операцию идемпотентной.

## Карта файлов

```
notification-service/
  cmd/main.go                     — DI, Kafka consumer, HTTP server, graceful shutdown
  internal/
    config/config.go              — конфиг из env (NOTIFICATION_*)
    model/
      notification.go             — доменная модель (Notification, NotificationType)
      errors.go                   — ErrNotificationNotFound, ErrForbidden
    service/notification/
      contract.go                 — интерфейсы Repository, Hub, NotificationService
      service.go                  — CreateFromEvent, ListByUserID, MarkAsRead, MarkAllAsRead
    repo/notification/
      repo.go                     — SQL: Create, ListByUserID, MarkAsRead, MarkAllAsRead
    kafka/
      consumer.go                 — Run(ctx): читает сообщения, вызывает service.CreateFromEvent
    sse/
      hub.go                      — Hub: Register/Unregister/Broadcast, goroutine per client
    token/
      store.go                    — InMemoryTokenStore: генерация UUID-токена, валидация, TTL 30s
    http/
      middleware/
        auth.go                   — JWT → user_id в context (переиспользует shared JWT_SECRET)
      handlers/
        router.go                 — chi-роутер, маршруты
        response.go               — respondJSON, respondError
        notification.go           — GET /notifications, POST /read, POST /read-all
        sse.go                    — POST /sse-token, GET /stream
  migrations/
    0001_create_notifications.up.sql
    0001_create_notifications.down.sql
```

## Ключевые решения

| Решение | Почему |
|---------|--------|
| SSE вместо WebSocket | Уведомления односторонние (сервер→клиент); SSE проще, встроенный автореконнект |
| Одноразовый токен для SSE | EventSource не поддерживает кастомные заголовки — JWT нельзя передать напрямую |
| Токены in-memory (sync.Map + TTL 30s) | Короткоживущие, нет смысла персистировать; Redis добавил бы overhead |
| `ON CONFLICT DO NOTHING` | Гарантирует идемпотентность при at-least-once доставке из Kafka |
| Hub с `chan` per client | Broadcast не блокируется медленными клиентами; клиент читает из канала в своей горутине |
| Offset коммитится после DB write | Если сервис упадёт до коммита — Kafka повторит доставку; дубль безопасен |
| NotificationType ≠ events.EventType | Доменная модель независима от транспорта; маппинг в consumer (Anti-Corruption Layer) |

## Конфигурация

| Переменная | Описание | Пример |
|------------|----------|--------|
| `NOTIFICATION_SERVER_PORT` | HTTP порт | `8081` |
| `NOTIFICATION_DATABASE_HOST` | PostgreSQL хост | `notification-db` |
| `NOTIFICATION_DATABASE_PORT` | PostgreSQL порт | `5432` |
| `NOTIFICATION_DATABASE_USER` | Пользователь | `postgres` |
| `NOTIFICATION_DATABASE_PASSWORD` | Пароль | `password` |
| `NOTIFICATION_DATABASE_NAME` | База данных | `notifications` |
| `JWT_SECRET` | Общий секрет с booking-service | `supersecretkey` |
| `KAFKA_BROKERS` | Адреса брокеров | `kafka:9092` |
| `KAFKA_TOPIC_BOOKING_EVENTS` | Топик | `booking.events` |
| `KAFKA_GROUP_ID` | Consumer group | `notification-service` |

## Запуск

```bash
# Из корня монорепо
make up       # поднимает все сервисы включая notification-service
make seed     # тестовые данные для booking-service
```

Сервис доступен на `http://localhost:8081`.

## Graceful Shutdown

1. SIGTERM → ctx отменяется
2. Kafka consumer завершает текущее сообщение, коммитит offset, закрывает reader
3. HTTP сервер перестаёт принимать новые соединения, дожидается in-flight запросов
4. SSE Hub закрывает все активные подключения (клиенты получат EOF и переподключатся)
5. Пул БД закрывается
