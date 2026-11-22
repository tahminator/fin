CREATE TABLE investment_securities (
    id             uuid        PRIMARY KEY DEFAULT uuidv7(),
    user_id        uuid        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    external_id    text        NOT NULL,
    name           text,
    ticker_symbol  text,
    type           text,
    subtype        text,
    currency       text,
    created_at     timestamptz NOT NULL DEFAULT now(),
    updated_at     timestamptz NOT NULL DEFAULT now(),
    UNIQUE (user_id, external_id)
);
