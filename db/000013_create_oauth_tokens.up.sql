CREATE TABLE oauth_tokens (
    id          uuid        PRIMARY KEY DEFAULT uuidv7(),
    grant_id    uuid        NOT NULL REFERENCES oauth_grants (id) ON DELETE CASCADE,
    user_id     uuid        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    kind        text        NOT NULL,
    token_hash  bytea       NOT NULL UNIQUE,
    scope       text        NOT NULL,
    expires_at  timestamptz NOT NULL,
    used_at     timestamptz,
    created_at  timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX oauth_tokens_grant_id_idx ON oauth_tokens (grant_id);
CREATE INDEX oauth_tokens_expires_at_idx ON oauth_tokens (expires_at);
