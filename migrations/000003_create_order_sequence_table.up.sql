CREATE TABLE IF NOT EXISTS order_sequences (
    "date"        DATE         PRIMARY KEY,
    "last_value"  INTEGER      NOT NULL DEFAULT 0,
    "created_at"  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    "updated_at"  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);