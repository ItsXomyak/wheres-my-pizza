-- Create orders table
CREATE TABLE IF NOT EXISTS "orders" (
    "id"                SERIAL        PRIMARY KEY,
    "created_at"        TIMESTAMPTZ   NOT NULL    DEFAULT NOW(),
    "updated_at"        TIMESTAMPTZ   NOT NULL    DEFAULT NOW(),
    "number"            TEXT          UNIQUE NOT NULL,
    "customer_name"     TEXT          NOT NULL,
    "type"              TEXT          NOT NULL CHECK (type IN ('dine_in', 'takeout', 'delivery')),
    "table_number"      INTEGER,
    "delivery_address"  TEXT,
    "total_amount"      DECIMAL(10,2) NOT NULL,
    "priority"          INTEGER       DEFAULT 1,
    "status"            TEXT          DEFAULT 'received',
    "processed_by"      TEXT,
    "completed_at"      TIMESTAMPTZ
);
