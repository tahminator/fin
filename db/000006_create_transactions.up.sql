CREATE TABLE transactions (
    id             uuid           PRIMARY KEY DEFAULT uuidv7(),
    user_id        uuid           NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    account_id     uuid           NOT NULL REFERENCES accounts (id) ON DELETE CASCADE,
    external_id    text           NOT NULL,
    amount         numeric(19, 4) NOT NULL,
    currency       text,
    posted_on      date           NOT NULL,
    authorized_on  date,
    name           text           NOT NULL,
    merchant_name  text,
    category       text,
    pending        boolean        NOT NULL DEFAULT false,
    created_at     timestamptz    NOT NULL DEFAULT now(),
    updated_at     timestamptz    NOT NULL DEFAULT now(),
    UNIQUE (account_id, external_id)
);

CREATE INDEX transactions_user_posted_idx ON transactions (user_id, posted_on DESC, id DESC);
CREATE INDEX transactions_account_posted_idx ON transactions (account_id, posted_on DESC, id DESC);
