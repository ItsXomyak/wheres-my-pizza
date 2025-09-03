-- Create order_items table
CREATE TABLE IF NOT EXISTS order_items (
    "id"          SERIAL        PRIMARY KEY,
    "created_at"  TIMESTAMPTZ   NOT NULL    DEFAULT NOW(),
    "order_id"    INTEGER       REFERENCES orders(id),
    "name"        TEXT          NOT NULL,
    "quantity"    INTEGER       NOT NULL,
    "price"       DECIMAL(8,2)  NOT NULL
);
