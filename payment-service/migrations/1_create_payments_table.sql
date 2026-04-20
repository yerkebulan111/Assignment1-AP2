CREATE TABLE IF NOT EXISTS payments (
    id             TEXT        PRIMARY KEY,
    order_id       TEXT        NOT NULL UNIQUE,
    transaction_id TEXT,
    amount         BIGINT      NOT NULL,
    status         TEXT        NOT NULL     CHECK (status IN ('Authorized', 'Declined')),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_payments_order_id ON payments (order_id);