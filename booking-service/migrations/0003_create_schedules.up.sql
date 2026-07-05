CREATE TABLE IF NOT EXISTS schedules (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    room_id      UUID NOT NULL UNIQUE REFERENCES rooms(id) ON DELETE RESTRICT,
    days_of_week INTEGER[] NOT NULL CHECK (cardinality(days_of_week) > 0),
    start_time   TIME NOT NULL,
    end_time     TIME NOT NULL CHECK (end_time > start_time),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
