CREATE TABLE IF NOT EXISTS outbox (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type TEXT        NOT NULL,
    payload    JSONB       NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    sent_at    TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_outbox_pending ON outbox (created_at) WHERE sent_at IS NULL;
