CREATE TABLE schedules (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    room_id      UUID NOT NULL UNIQUE REFERENCES rooms(id),
    days_of_week INTEGER[] NOT NULL,
    start_time   TIME NOT NULL,
    end_time     TIME NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
