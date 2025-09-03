-- Create order_status_log table
CREATE TABLE IF NOT EXISTS order_status_log (
    "id"          SERIAL        PRIMARY KEY,
    "created_at"  TIMESTAMPTZ   NOT NULL    DEFAULT NOW(),
    "order_id"    INTEGER       REFERENCES orders(id),
    "status"      TEXT,
    "changed_by"  TEXT,
    "changed_at"  TIMESTAMPTZ   DEFAULT CURRENT_TIMESTAMP,
    "notes"       TEXT
);
