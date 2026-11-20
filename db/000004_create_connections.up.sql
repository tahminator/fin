CREATE TABLE connections (
    id                       uuid        PRIMARY KEY DEFAULT uuidv7(),
    user_id                  uuid        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    provider                 text        NOT NULL,
    external_id              text        NOT NULL,
    access_token_ciphertext  text        NOT NULL,
    institution_id           text,
    institution_name         text,
    status                   text        NOT NULL,
    status_changed_at        timestamptz NOT NULL DEFAULT now(),
    sync_cursor              text        NOT NULL DEFAULT '',
    last_synced_at           timestamptz,
    last_error               text,
    created_at               timestamptz NOT NULL DEFAULT now(),
    updated_at               timestamptz NOT NULL DEFAULT now(),
    UNIQUE (provider, external_id)
);

CREATE INDEX connections_user_id_idx ON connections (user_id);
