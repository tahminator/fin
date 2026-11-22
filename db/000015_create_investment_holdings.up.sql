CREATE TABLE investment_holdings (
    id                      uuid        PRIMARY KEY DEFAULT uuidv7(),
    user_id                 uuid        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    account_id              uuid        NOT NULL REFERENCES accounts (id) ON DELETE CASCADE,
    security_id             uuid        NOT NULL REFERENCES investment_securities (id) ON DELETE CASCADE,
    institution_price       numeric(19, 4) NOT NULL,
    institution_price_as_of date,
    institution_value       numeric(19, 4) NOT NULL,
    cost_basis              numeric(19, 4),
    quantity                numeric(19, 4) NOT NULL,
    currency                text,
    vested_quantity         numeric(19, 4),
    vested_value            numeric(19, 4),
    created_at              timestamptz NOT NULL DEFAULT now(),
    updated_at              timestamptz NOT NULL DEFAULT now(),
    UNIQUE (account_id, security_id)
);

CREATE INDEX investment_holdings_user_idx ON investment_holdings (user_id);
CREATE INDEX investment_holdings_account_idx ON investment_holdings (account_id);
