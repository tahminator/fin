CREATE TABLE oauth_authorizations (
    id                  uuid        PRIMARY KEY DEFAULT uuidv7(),
    request_token_hash  bytea       NOT NULL UNIQUE,
    client_id           text        NOT NULL REFERENCES oauth_clients (id) ON DELETE CASCADE,
    user_id             uuid        REFERENCES users (id) ON DELETE CASCADE,
    redirect_uri        text        NOT NULL,
    scope               text        NOT NULL,
    resource            text        NOT NULL,
    state               text,
    code_challenge      text        NOT NULL,
    code_hash           bytea       UNIQUE,
    code_expires_at     timestamptz,
    consumed_at         timestamptz,
    grant_id            uuid        REFERENCES oauth_grants (id) ON DELETE CASCADE,
    created_at          timestamptz NOT NULL DEFAULT now(),
    expires_at          timestamptz NOT NULL
);

CREATE INDEX oauth_authorizations_expires_at_idx ON oauth_authorizations (expires_at);
