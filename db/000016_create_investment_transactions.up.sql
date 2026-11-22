CREATE TABLE investment_transactions (
    id                    uuid        PRIMARY KEY DEFAULT uuidv7(),
    user_id               uuid        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    account_id            uuid        NOT NULL REFERENCES accounts (id) ON DELETE CASCADE,
    security_id           uuid        REFERENCES investment_securities (id) ON DELETE SET NULL,
    external_id           text        NOT NULL,
    cancel_external_id    text,
    posted_on             date        NOT NULL,
    transaction_datetime  timestamptz,
    name                  text        NOT NULL,
    quantity              numeric(19, 4),
    amount                numeric(19, 4) NOT NULL,
    price                 numeric(19, 4),
    fees                  numeric(19, 4),
    type                  text        NOT NULL,
    subtype               text        NOT NULL,
    currency              text,
    created_at            timestamptz NOT NULL DEFAULT now(),
    updated_at            timestamptz NOT NULL DEFAULT now(),
    UNIQUE (account_id, external_id)
);

CREATE INDEX investment_transactions_user_posted_idx ON investment_transactions (user_id, posted_on DESC, id DESC);
CREATE INDEX investment_transactions_account_posted_idx ON investment_transactions (account_id, posted_on DESC, id DESC);
