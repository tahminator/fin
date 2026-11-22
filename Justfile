set dotenv-load
set shell := ["bash", "-euo", "pipefail", "-c"]

[doc("List recipes")]
default:
    @just --list

[doc("Install Go modules and JS dependencies")]
install:
    go mod download
    pnpm --dir js install

[doc("Start local Postgres")]
db:
    docker compose -f infra/compose.yaml up -d postgres

[doc("Stop local Postgres")]
db-stop:
    docker compose -f infra/compose.yaml down

[doc("Apply all migrations with the golang-migrate CLI")]
migrate:
    migrate -path db -database "$DATABASE_URL" up

[doc("Roll back N migrations")]
migrate-down n="1":
    migrate -path db -database "$DATABASE_URL" down {{n}}

[doc("Show the current migration version")]
migrate-version:
    migrate -path db -database "$DATABASE_URL" version

[doc("Create a new sequential migration pair")]
migration name:
    migrate create -ext sql -dir db -seq -digits 6 {{name}}

[doc("Allow an email and store its Plaid credentials (DATABASE_URL from .env)")]
seed email client_id secret env="sandbox":
    @go run ./cmd/seed -email {{quote(email)}} -client-id {{quote(client_id)}} -secret {{quote(secret)}} -env {{quote(env)}}

[doc("Add fictional sample data to the local database")]
seed-dev email:
    psql "$DATABASE_URL" -v email={{quote(email)}} -f db/seed/local.sql

[doc("Generate openapi.json and TypeScript API types")]
gen:
    go run . openapi > openapi.json
    pnpm --dir js gen

[doc("Run the Go API")]
dev-api:
    go run . serve

[doc("Run the Vite dev server")]
dev-web:
    pnpm --dir js dev

[doc("Run DB (skip with --no-db), migrations, codegen, API and web together")]
dev *flags:
    #!/usr/bin/env bash
    set -euo pipefail
    skip_db=false
    for f in {{flags}}; do
        case "$f" in
            --no-db) skip_db=true ;;
            *) echo "dev: unknown flag '$f' (expected --no-db)" >&2; exit 1 ;;
        esac
    done
    if [ "$skip_db" = false ]; then just db; fi
    just migrate
    just gen
    trap 'kill 0' EXIT; npx mprocs "just dev-api" "just dev-web"

[doc("Lint everything")]
lint: lint-go lint-js

[doc("Lint Go with golangci-lint and the no-comments check")]
lint-go:
    golangci-lint run ./...
    go run ./tools/nocomments .

[doc("Typecheck, oxlint and oxfmt --check the frontend")]
lint-js: gen
    pnpm --dir js check

[doc("Apply lint autofixes")]
fix:
    golangci-lint run --fix ./...
    pnpm --dir js lint:fix

[doc("Format Go (gofumpt, goimports) and JS (oxfmt)")]
fmt:
    golangci-lint fmt ./...
    pnpm --dir js fmt

[doc("Run Go unit tests")]
test:
    go test -race ./...

[doc("Run Postgres integration tests against DATABASE_URL after just migrate")]
test-db:
    go test -race -count=1 -tags integration ./internal/repository/postgres/...

[doc("Run Go unit tests with a coverage report")]
cover:
    go test -race -coverprofile=coverage.out ./...
    go tool cover -func=coverage.out

[doc("Force a sandbox connection into re-auth for manual verification")]
sandbox-reset-login connection_id:
    go run ./cmd/sandbox-reset-login {{connection_id}}

[doc("Build frontend and backend binary")]
build: gen
    pnpm --dir js build
    go build -o bin/fin .

[doc("Build the production image")]
docker-build:
    docker build -f infra/Dockerfile \
        --build-arg GO_VERSION="$(go mod edit -json | jq -r .Go)" \
        --build-arg NODE_VERSION="$(jq -r .devEngines.runtime.version js/package.json)" \
        --build-arg PNPM_VERSION="$(jq -r .packageManager js/package.json | cut -d@ -f2 | cut -d+ -f1)" \
        -t fin .

[doc("Show outdated Go and JS dependencies")]
outdated:
    go list -u -m all
    pnpm --dir js outdated
