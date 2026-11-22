CREATE TABLE investment_account_snapshots (
    id           uuid           PRIMARY KEY DEFAULT uuidv7(),
    user_id      uuid           NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    account_id   uuid           NOT NULL REFERENCES accounts (id) ON DELETE CASCADE,
    captured_at  timestamptz    NOT NULL,
    value        numeric(19, 4) NOT NULL,
    currency     text,
    created_at   timestamptz    NOT NULL DEFAULT now(),
    UNIQUE (account_id, captured_at)
);

CREATE INDEX investment_account_snapshots_user_captured_idx ON investment_account_snapshots (user_id, captured_at);
CREATE INDEX investment_account_snapshots_user_account_captured_idx ON investment_account_snapshots (user_id, account_id, captured_at);
