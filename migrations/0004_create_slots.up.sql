CREATE TABLE slots (
    id       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    room_id  UUID NOT NULL REFERENCES rooms(id),
    start_at TIMESTAMPTZ NOT NULL,
    end_at   TIMESTAMPTZ NOT NULL,
    UNIQUE (room_id, start_at)
);
