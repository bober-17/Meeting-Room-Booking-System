CREATE TABLE bookings (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slot_id         UUID NOT NULL REFERENCES slots(id),
    user_id         UUID NOT NULL REFERENCES users(id),
    status          TEXT NOT NULL DEFAULT 'active',
    conference_link TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX idx_bookings_slot_active
    ON bookings (slot_id) WHERE status = 'active';

CREATE INDEX idx_bookings_user_id ON bookings (user_id);
