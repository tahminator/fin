CREATE TABLE rate_limits (
    scope              text        NOT NULL,
    key                text        NOT NULL,
    window_started_at  timestamptz NOT NULL,
    expires_at         timestamptz NOT NULL,
    hits               integer     NOT NULL,
    PRIMARY KEY (scope, key)
);

CREATE INDEX rate_limits_expires_at_idx ON rate_limits (expires_at);
