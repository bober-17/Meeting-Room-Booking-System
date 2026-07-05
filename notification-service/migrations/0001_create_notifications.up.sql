CREATE TABLE IF NOT EXISTS notifications (
    id         UUID        PRIMARY KEY,
    user_id    UUID        NOT NULL,
    type       TEXT        NOT NULL CHECK (type IN ('booking.created', 'booking.cancelled')),
    booking_id UUID        NOT NULL,
    room_name  TEXT        NOT NULL,
    slot_start TIMESTAMPTZ NOT NULL,
    slot_end   TIMESTAMPTZ NOT NULL CHECK (slot_end > slot_start),
    is_read    BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_notifications_booking_type ON notifications (booking_id, type);
CREATE INDEX IF NOT EXISTS idx_notifications_user_created ON notifications (user_id, created_at DESC);
