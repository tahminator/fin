CREATE TABLE investment_securities (
    id             uuid PRIMARY KEY DEFAULT uuidv7(),
    external_id    text NOT NULL UNIQUE,
    name           text,
    ticker_symbol  text,
    type           text,
    subtype        text,
    currency       text,
    created_at     timestamptz NOT NULL DEFAULT now(),
    updated_at     timestamptz NOT NULL DEFAULT now()
);
