CREATE TABLE gauges (
                        metric_name TEXT PRIMARY KEY,
                        value DOUBLE PRECISION NOT NULL,
                        created_at_utc TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
                        updated_at_utc TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);