CREATE TABLE investment_holding_snapshots (
    snapshot_id        uuid           NOT NULL REFERENCES investment_account_snapshots (id) ON DELETE CASCADE,
    user_id            uuid           NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    account_id         uuid           NOT NULL REFERENCES accounts (id) ON DELETE CASCADE,
    security_id        uuid           NOT NULL REFERENCES investment_securities (id) ON DELETE CASCADE,
    captured_at        timestamptz    NOT NULL,
    quantity           numeric(19, 4) NOT NULL,
    institution_price  numeric(19, 4) NOT NULL,
    institution_value  numeric(19, 4) NOT NULL,
    currency           text,
    PRIMARY KEY (snapshot_id, security_id)
);

CREATE INDEX investment_holding_snapshots_user_security_captured_idx ON investment_holding_snapshots (user_id, security_id, captured_at);
CREATE INDEX investment_holding_snapshots_user_account_captured_idx ON investment_holding_snapshots (user_id, account_id, captured_at);
