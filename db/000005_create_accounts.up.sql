CREATE TABLE accounts (
    id                 uuid           PRIMARY KEY DEFAULT uuidv7(),
    user_id            uuid           NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    connection_id      uuid           NOT NULL REFERENCES connections (id) ON DELETE CASCADE,
    external_id        text           NOT NULL,
    name               text           NOT NULL,
    official_name      text,
    mask               text,
    type               text           NOT NULL,
    subtype            text,
    current_balance    numeric(19, 4),
    available_balance  numeric(19, 4),
    currency           text,
    created_at         timestamptz    NOT NULL DEFAULT now(),
    updated_at         timestamptz    NOT NULL DEFAULT now(),
    UNIQUE (connection_id, external_id)
);

CREATE INDEX accounts_user_id_idx ON accounts (user_id);
