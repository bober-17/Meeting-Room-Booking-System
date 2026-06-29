CREATE TABLE notifications (
    id         UUID PRIMARY KEY,
    user_id    UUID        NOT NULL,
    type       TEXT        NOT NULL,
    booking_id UUID        NOT NULL,
    room_name  TEXT        NOT NULL,
    slot_start TIMESTAMPTZ NOT NULL,
    slot_end   TIMESTAMPTZ NOT NULL,
    is_read    BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- идемпотентная вставка: один тип события на одну бронь
CREATE UNIQUE INDEX idx_notifications_booking_type ON notifications (booking_id, type);

-- быстрая выборка всех уведомлений пользователя по дате
CREATE INDEX idx_notifications_user_created ON notifications (user_id, created_at DESC);
