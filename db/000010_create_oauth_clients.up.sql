CREATE TABLE oauth_clients (
    id                          text        PRIMARY KEY,
    source                      text        NOT NULL,
    client_name                 text        NOT NULL,
    client_uri                  text,
    logo_uri                    text,
    redirect_uris               text[]      NOT NULL,
    token_endpoint_auth_method  text        NOT NULL,
    client_secret_hash          bytea,
    metadata_expires_at         timestamptz,
    created_at                  timestamptz NOT NULL DEFAULT now(),
    updated_at                  timestamptz NOT NULL DEFAULT now()
);
