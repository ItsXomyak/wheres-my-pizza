CREATE TABLE IF NOT EXISTS workers (
    "id"                serial      primary key,
    "created_at"        timestamptz not null    default now(),
    "name"              text        unique not null,
    "type"              text        not null,
    "status"            text        default 'online',
    "last_seen"         timestamptz default current_timestamp,
    "orders_processed"  integer     default 0
);

-- For MarkOnline: fast filtering by status and last_seen
-- Helps with queries that check "status = 'offline'" OR "last_seen < now() - interval"
CREATE INDEX IF NOT EXISTS idx_workers_status_last_seen ON workers(status, last_seen);