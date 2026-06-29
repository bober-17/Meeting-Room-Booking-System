CREATE TABLE IF NOT EXISTS slots (
    id       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    room_id  UUID NOT NULL REFERENCES rooms(id) ON DELETE RESTRICT,
    start_at TIMESTAMPTZ NOT NULL,
    end_at   TIMESTAMPTZ NOT NULL CHECK (end_at > start_at),
    UNIQUE (room_id, start_at)
);
