CREATE TABLE IF NOT EXISTS rooms (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT NOT NULL,
    description TEXT,
    capacity    INTEGER,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
