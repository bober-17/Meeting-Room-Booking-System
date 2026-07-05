# Расширенное техническое задание

Оригинальное задание — в [TASK.md](TASK.md).

Этот документ описывает функциональность, реализованную сверх оригинального ТЗ:
уведомления в реальном времени и асинхронное взаимодействие между сервисами.

---

## Мотивация

Оригинальное ТЗ описывает сервис бронирования как монолитный REST API. В реальной корпоративной
среде пользователям важно получать мгновенную обратную связь: уведомление о том, что бронь создана
или отменена (в том числе администратором). Прямая доставка уведомлений внутри booking-service
нарушила бы принцип единственной ответственности и создала бы жёсткую связанность.

Решение: выделить notification-service как отдельный микросервис, связанный с booking-service
через Kafka.

---

## Новые бизнес-требования

### Уведомления

Пользователь должен получать уведомление в следующих случаях:

| Событие | Тип уведомления |
|---------|----------------|
| Создание брони | `booking.created` |
| Отмена брони | `booking.cancelled` |

**Требования к доставке:**
- Уведомление доставляется пользователю в реальном времени, если он подключён к системе
- Если пользователь не подключён — уведомление сохраняется и доступно при следующем входе
- Пользователь может отметить уведомление как прочитанное
- Дубликат уведомления по одному событию не создаётся (идемпотентность)
- Retention уведомлений — 90 дней

### Нагрузка notification-service

| Метрика | Значение |
|---------|----------|
| Событий из Kafka (пик) | ~20/сек |
| Concurrent SSE-соединений | ~500 |
| Допустимый lag consumer-а | < 30 сек |
| Delivery latency p95 | < 1 сек |

---

## Архитектура расширения

### Схема взаимодействия

```
booking-service                          notification-service
      │                                          │
      │  BEGIN TRANSACTION                       │
      │  INSERT bookings                         │
      │  INSERT outbox          Kafka            │
      │  COMMIT          ──────────────────►     │  FetchMessage()
      │                    booking.events        │  INSERT notifications
      │                                          │    ON CONFLICT DO NOTHING
      │                                          │  CommitOffset()
      │                                          │  Hub.Broadcast()
      │                                          │       │
      │                                          │       ▼
      │                                    SSE stream → Client
```

### Outbox Relay (booking-service)

Публикация событий в Kafka не происходит напрямую из транзакции — вместо этого используется
**Outbox pattern**:

1. В той же транзакции что и бронь создаётся запись в таблице `outbox`
2. Фоновая горутина (`Relay`) каждые 2 секунды читает необработанные записи через
   `SELECT ... FOR UPDATE SKIP LOCKED` и публикует их в Kafka
3. После успешной публикации записи помечаются как обработанные

**Почему не прямая публикация в Kafka:**
- Прямой `producer.Publish()` внутри транзакции нарушает атомарность: бронь может сохраниться,
  а Kafka-сообщение потеряться при сбое
- Outbox гарантирует: либо оба артефакта (бронь + событие) существуют, либо ни одного

### Kafka Consumer (notification-service)

- Стратегия доставки: **at-least-once**
- Offset коммитится после записи уведомления в БД — если сервис упадёт до коммита, Kafka
  повторит сообщение
- Дублирование при повторной доставке безопасно: `UNIQUE (booking_id, type)` +
  `ON CONFLICT DO NOTHING` в таблице `notifications`

### SSE Hub (notification-service)

Уведомления доставляются подключённым клиентам через **Server-Sent Events**:

- Клиент подключается к `GET /notifications/stream?token=<one-time-token>`
- Hub хранит map `user_id → send chan` в памяти
- `Broadcast()` не блокируется на медленных клиентах: каждый клиент читает из своего канала
  в отдельной горутине; при переполнении канала клиент отключается

**Почему SSE, а не WebSocket:**
- Уведомления односторонние (сервер → клиент) — WebSocket избыточен
- SSE поддерживает автоматическое переподключение браузером
- Проще в реализации и отладке

**Одноразовый токен для SSE:**
- `EventSource` (Browser API) не поддерживает кастомные заголовки — JWT нельзя передать
  через `Authorization`
- Клиент вызывает `POST /sse-token` с JWT → получает одноразовый токен с TTL 30 сек
- TTL 30 сек — это время на установку соединения, не длительность самого стрима:
  после того как `GET /notifications/stream` принял токен, SSE-соединение живёт
  сколько угодно, токен сгорает при первом использовании
- Токены хранятся in-memory (`sync.Map`), не персистируются — при рестарте сервиса
  клиент запрашивает новый токен и переподключается

---

## API notification-service

| Метод | Путь | Auth | Описание |
|-------|------|------|----------|
| `POST` | `/sse-token` | JWT | Получить одноразовый токен для подключения к SSE (действителен 30 сек, сгорает при использовании) |
| `GET` | `/notifications/stream?token=<tok>` | one-time token | SSE поток уведомлений |
| `GET` | `/notifications` | JWT | Список уведомлений (`limit`, `offset`, `unread_only`) |
| `POST` | `/notifications/{id}/read` | JWT | Отметить уведомление прочитанным |
| `POST` | `/notifications/read-all` | JWT | Отметить все уведомления прочитанными |

### Модель уведомления

```json
{
  "id": "uuid",
  "type": "booking.created",
  "booking_id": "uuid",
  "room_name": "Переговорная 1",
  "slot_start": "2026-07-01T09:00:00Z",
  "slot_end": "2026-07-01T09:30:00Z",
  "is_read": false,
  "created_at": "2026-06-30T14:00:00Z"
}
```

### SSE формат события

```
data: {"id":"uuid","type":"booking.created","room_name":"Переговорная 1","slot_start":"...","is_read":false}

```

---

## Схема данных notification-service

```sql
CREATE TABLE notifications (
    id         UUID PRIMARY KEY,
    user_id    UUID NOT NULL,
    type       TEXT NOT NULL CHECK (type IN ('booking.created', 'booking.cancelled')),
    booking_id UUID NOT NULL,
    room_name  TEXT NOT NULL,
    slot_start TIMESTAMPTZ NOT NULL,
    slot_end   TIMESTAMPTZ NOT NULL,
    is_read    BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- идемпотентность consumer-а: один тип события на одну бронь
CREATE UNIQUE INDEX idx_notifications_booking_type ON notifications (booking_id, type);

-- покрывающий индекс для GET /notifications (выборка по пользователю, сортировка по дате)
CREATE INDEX idx_notifications_user_created ON notifications (user_id, created_at DESC);
```

---

## Структура монорепо

Переход от single-service к монорепо обусловлен появлением второго сервиса.
Подробнее — в корневом [README.md](README.md).

---

## Гарантии и fault tolerance

| Сценарий | Поведение |
|---------|-----------|
| Kafka недоступна при создании брони | бронь создаётся; outbox накапливает; relay опубликует после восстановления |
| notification-service недоступен | сообщения хранятся в Kafka (retention 7 дней); обрабатываются после рестарта |
| SSE-соединение разорвалось | клиент переподключается; пропущенные уведомления читает через `GET /notifications?unread_only=true` |
| БД booking-service недоступна | транзакция (бронь + outbox) откатывается целиком; частичных записей не возникает |
| БД notification-service недоступна | consumer получает ошибку при INSERT; offset не коммитится; Kafka повторит сообщение после восстановления |
