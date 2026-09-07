CREATE TABLE counters (
                          metric_name TEXT PRIMARY KEY,
                          value BIGINT NOT NULL,
                          created_at_utc TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
                          updated_at_utc TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);