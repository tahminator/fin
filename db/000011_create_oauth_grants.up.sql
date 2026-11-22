CREATE TABLE oauth_grants (
    id            uuid        PRIMARY KEY DEFAULT uuidv7(),
    user_id       uuid        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    client_id     text        NOT NULL REFERENCES oauth_clients (id) ON DELETE CASCADE,
    scope         text        NOT NULL,
    resource      text        NOT NULL,
    created_at    timestamptz NOT NULL DEFAULT now(),
    expires_at    timestamptz NOT NULL,
    last_used_at  timestamptz,
    UNIQUE (user_id, client_id)
);

CREATE INDEX oauth_grants_client_id_idx ON oauth_grants (client_id);
