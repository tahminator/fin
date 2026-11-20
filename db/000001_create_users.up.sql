CREATE TABLE users (
    id          uuid        PRIMARY KEY DEFAULT uuidv7(),
    email       text        NOT NULL,
    google_sub  text        UNIQUE,
    name        text,
    created_at  timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX users_email_lower_idx ON users (lower(email));
