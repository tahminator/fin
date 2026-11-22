# fin — Build Plan

A personal finance tracker for a small allow-list of users. Go backend (`./`), Vite + React PWA frontend (`./js/`), Postgres migrations (`./db/`), container build (`./infra/`).

## 0. Decisions & assumptions

| Topic                 | Decision                                                                                                                                                                                                                                                                                          | Why                                                                                                                                    |
| --------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------- |
| "Stripe key" per user | Read as **per-user aggregation-provider credentials** (Plaid `client_id` + `secret` + environment), stored in `user_provider_credentials` keyed by `(user_id, provider)`                                                                                                                          | Plaid is the account provider; the table is provider-agnostic, so a future Stripe Financial Connections provider adds rows, not schema |
| Allow-list            | A row in `users` = allowed. Rows inserted manually (no admin UI)                                                                                                                                                                                                                                  | Requirement (o)                                                                                                                        |
| Money                 | `numeric(19,4)` in Postgres, decimal **string** in Go and JSON. Outflows negative                                                                                                                                                                                                                 | No float math, currency-exponent agnostic, sums happen in SQL                                                                          |
| IDs                   | `uuid DEFAULT uuidv7()` (Postgres 18+)                                                                                                                                                                                                                                                            | Time-ordered, non-enumerable                                                                                                           |
| Enums                 | `text` columns, no `CHECK`; Go `type X string` + consts + `Valid()`; exposed to OpenAPI via huma `enum:"..."` tags                                                                                                                                                                                | Requirement (ad); openapi-typescript emits string unions                                                                               |
| Pagination            | Offset: `page` (1-based) + `pageSize` (max 100); response `Page[T]{items,page,pageSize,total,hasNext}`                                                                                                                                                                                            | Readable SQL; huma names generic schemas `PageTransaction`, etc.                                                                       |
| Migrations            | Explicit step only, via the golang-migrate **CLI** (`migrate -path db -database "$DATABASE_URL" up`, wrapped by `just migrate`). The app binary has no migrate code and the runtime image carries no migration files; production uses the `migrate/migrate` image against `./db` before deploying | Schema changes are deliberate; nothing migrates at runtime                                                                             |
| Static assets         | Served from disk at `STATIC_DIR` (not `go:embed`)                                                                                                                                                                                                                                                 | No directive comments, dev build works without `js/dist`                                                                               |
| Ops endpoints         | `/healthz`, `/readyz`, `/metrics` on the main `HTTP_ADDR` listener, outside request logging/metrics; the ingress blocks them publicly                                                                                                                                                             | Metrics carry per-user labels; one listener is simpler than a second port                                                              |
| Metric labels         | Every label value is a typed Go enum (`iota` + `String()`), so the set of series is fixed at compile time. Sole exception: `user` (email) on the DB-snapshot gauges, whose values come only from the `users` allow-list                                                                           | Easy refactors, locked-down cardinality; per-user counts were requested                                                                |
| Metrics access        | Instruments are unexported in `internal/metrics`; code records through the `metrics.Recorder` interface (`metrics.Prometheus{}` in prod, zero-sized `metrics.Noop{}` / embedding fakes in tests)                                                                                                  | No direct manipulation, trivially testable                                                                                             |
| Sync trigger          | Manual refresh (all of a user's connections, or one), initial sync after linking, and a backend goroutine on a cron schedule from `SYNC_CRON` (default `0 */6 * * *`). Every failure is counted in Prometheus. No webhooks                                                                        | Keeps data fresh and detects re-auth without exposing a public webhook endpoint                                                        |
| DB transactions       | None. Sync is idempotent (upserts) and saves the cursor last                                                                                                                                                                                                                                      | Requirement (i); a failed sync simply re-runs                                                                                          |
| Comments              | Zero, enforced by lint on both codebases (Go AST tool + custom oxlint rule). Justfile uses `[doc()]` attributes instead of `#`. Only sanctioned comment: the `SECRET_KEY` hint in `.env.example` (explicitly requested)                                                                           | Requirement (d)                                                                                                                        |
| Code layout           | Natural blank lines between logical blocks in **every** file we write: Go, TS/TSX, SQL, Justfile, Dockerfile, YAML, env files                                                                                                                                                                     | Non-negotiable readability rule                                                                                                        |
| Lint/format (JS)      | **oxlint** + **oxfmt** only (no ESLint, no Prettier)                                                                                                                                                                                                                                              | One fast toolchain                                                                                                                     |
| Config                | Handwritten `config.NewFromEnv()` over `os.LookupEnv`; no env library; no tests for the config package                                                                                                                                                                                            | Explicit requirement                                                                                                                   |
| Token encryption      | Plaid access tokens encrypted at rest with AES-256-GCM keyed by `SECRET_KEY` (32 chars). No key versioning                                                                                                                                                                                        | Explicit requirement                                                                                                                   |
| Re-auth               | Detected on every sync, stored as `connections.status = needs_reauth`, exposed via `GET /api/notifications`, shown as a banner + nav badge + per-connection **Reconnect** (Plaid update mode)                                                                                                     | You want to know when a bank needs sign-in again                                                                                       |
| Tests                 | Backend business logic unit-tested (stdlib `testing`, `humatest`, interface fakes). No frontend tests yet. Complex flows verified manually (§5.13)                                                                                                                                                | Explicit requirement                                                                                                                   |
| MCP server            | MCP server at `/mcp` on the same deployment and in the OpenAPI spec; fin is its own OAuth 2.1 authorization server on top of Google login; opaque hashed tokens; query tools plus a scoped refresh tool; linking, re-auth and removal are unreachable (§13)                                       | Agents can analyze finances; linking banks stays a human, in-app action                                                                |

Out of scope: admin UI, Plaid webhooks, push notifications, `SECRET_KEY` rotation/versioning, and budgets/categorization rules. CI/CD is specified in phase 2 (§12). An MCP server for agents, with its own OAuth authorization server, is specified in phase 3 (§13); it can query data and refresh already-linked connections, but can never create, re-authenticate or remove bank connections.

## 1. Repository layout

```
.
├── PLAN.md
├── README.md
├── Justfile
├── .env.example
├── .env                          gitignored; local credentials (Google, SECRET_KEY)
├── .gitignore
├── .dockerignore
├── .golangci.yml
├── go.mod
├── go.sum
├── main.go                       command dispatch: serve | openapi
├── serve.go                      config, cipher, pool, repos, services, handlers, scheduler, servers (DI)
├── openapi.go                    prints spec with zero-value handlers, no DB/env needed
├── cmd/
│   ├── seed/main.go              tiny CLI: allow an email + store its Plaid credentials
│   └── sandbox-reset-login/main.go  dev-only: force a Plaid sandbox item into re-auth
├── db/
│   ├── 000001_create_users.up.sql / .down.sql
│   ├── 000002_create_sessions.up.sql / .down.sql
│   ├── 000003_create_user_provider_credentials.up.sql / .down.sql
│   ├── 000004_create_connections.up.sql / .down.sql
│   ├── 000005_create_accounts.up.sql / .down.sql
│   └── 000006_create_transactions.up.sql / .down.sql
├── internal/
│   ├── config/
│   │   ├── config.go             Config, Environment enum, NewFromEnv, validation (no tests)
│   │   └── env.go                envReader helpers
│   ├── encryption/cipher.go      AES-256-GCM keyed by SECRET_KEY
│   ├── domain/                   internal types + Go enums + sentinel errors (no json tags)
│   │   ├── user.go
│   │   ├── session.go
│   │   ├── credential.go         Provider, ProviderEnvironment enums
│   │   ├── connection.go         ConnectionStatus enum
│   │   ├── account.go            AccountType enum
│   │   ├── transaction.go
│   │   ├── summary.go
│   │   ├── notification.go       Notification, NotificationKind enum
│   │   └── errors.go             ErrNotFound, ErrNotAllowed, ErrConflict, ErrReauthRequired
│   ├── repository/               interfaces only
│   │   ├── user.go
│   │   ├── session.go
│   │   ├── credential.go
│   │   ├── connection.go
│   │   ├── account.go
│   │   ├── transaction.go
│   │   ├── summary.go
│   │   ├── metrics.go
│   │   ├── pinger.go
│   │   └── postgres/             pgx implementations, raw SQL
│   │       ├── user.go
│   │       ├── session.go
│   │       ├── ...one file per interface
│   │       └── pinger.go         SELECT 1
│   ├── provider/
│   │   ├── provider.go           Provider interface
│   │   ├── types.go              LinkedItem, Account, Transaction, SyncResult
│   │   └── plaid/
│   │       ├── client.go         per-credential *plaid.APIClient
│   │       ├── link.go           CreateLinkToken, Exchange, Remove
│   │       ├── sync.go           Accounts, SyncTransactions
│   │       └── mapping.go        plaid → provider types, sign flip, error mapping
│   ├── identity/google.go        OIDC + PKCE wrapper behind IdentityProvider interface
│   ├── service/
│   │   ├── auth.go               login/callback/logout/session resolution
│   │   ├── connection.go         link, sync, re-auth, remove; encrypts/decrypts access tokens
│   │   ├── notification.go       re-auth notifications
│   │   └── scheduler.go          periodic sync of active connections
│   ├── handler/                  one file per resource; each type has Register()
│   │   ├── page.go               Page[T], PageParams
│   │   ├── auth.go               Login, Callback (net/http) + Me, Logout (huma)
│   │   ├── connection.go
│   │   ├── connection_types.go   DTOs once they outgrow the handler file
│   │   ├── account.go
│   │   ├── transaction.go
│   │   ├── summary.go
│   │   ├── notification.go
│   │   └── health.go             liveness/readiness
│   ├── server/
│   │   ├── api.go                huma config (/openapi.json, /swagger)
│   │   ├── routes.go             the ONE place every route is listed
│   │   ├── middleware.go         session auth (huma), request logging + HTTP metrics (net/http)
│   │   ├── spa.go                static files + index.html fallback
│   │   └── ops.go                ops mux
│   └── metrics/
│       ├── labels.go             iota enums + String() for every label
│       ├── instruments.go        unexported registry + collectors, series pre-initialized from the enums
│       ├── recorder.go           Recorder interface, Prometheus{} and Noop{} (both zero-sized)
│       ├── database.go           WatchDatabase: DB-snapshot collector evaluated at scrape time
│       └── handler.go            Handler() for /metrics
├── tools/nocomments/main.go      fails on any non-directive Go comment
├── infra/
│   ├── Dockerfile
│   └── compose.yaml              local Postgres only
└── js/
    ├── package.json
    ├── pnpm-lock.yaml
    ├── pnpm-workspace.yaml       pnpm settings (onlyBuiltDependencies) if needed
    ├── index.html
    ├── vite.config.ts
    ├── tsconfig.json / tsconfig.app.json / tsconfig.node.json
    ├── .oxlintrc.json
    ├── .oxfmtrc.json
    ├── components.json           shadcn
    ├── lint/fin-plugin.ts        custom oxlint rules
    ├── public/                   PWA icons, apple-touch-icon
    └── src/
        ├── main.tsx
        ├── index.css
        ├── app/                  route-shaped pages and private components (see §6.2)
        ├── components/
        │   ├── ui/               shadcn
        │   ├── loading-state.tsx
        │   ├── error-state.tsx
        │   └── empty-state.tsx
        └── lib/
            ├── api/
            │   ├── schema.gen.ts generated, gitignored
            │   └── client.ts     openapi-fetch + openapi-react-query ($api)
            ├── queries/          composite hooks wrapping $api (only consumers of $api)
            ├── hooks/            shared hooks; the ONLY place useEffect may appear
            ├── types/api.ts      aliases over components["schemas"]
            ├── pwa/platform.ts   platform detection for install guide
            ├── plaid/link-token-store.ts
            ├── format.ts         money/date formatting
            ├── query-client.ts
            └── utils.ts          shadcn cn()
```

`*_test.go` files sit next to the code they test (§5.13).

## 2. Tooling — latest versions

Every dependency is added with `@latest` / `go get <mod>@latest` at scaffold time; no versions are copied from this document. Verified current at writing: Postgres **19** (GA 2026-11-19), oxlint 1.9x (JS plugins + type-aware), oxfmt 0.7x.

| Area             | Tool                                                                                                                              |
| ---------------- | --------------------------------------------------------------------------------------------------------------------------------- |
| Go               | latest stable toolchain (`go.mod` `go` + `toolchain` lines), `log/slog`, `net/http` ServeMux patterns                             |
| HTTP/OpenAPI     | `github.com/danielgtaylor/huma/v2` + `adapters/humago`                                                                            |
| DB               | `github.com/jackc/pgx/v5` (`pgxpool`); migrations via the golang-migrate CLI (`brew install golang-migrate`), not a Go dependency |
| Config           | stdlib only: handwritten `config.NewFromEnv()` over `os.LookupEnv`                                                                |
| Auth             | `golang.org/x/oauth2`, `github.com/coreos/go-oidc/v3`                                                                             |
| Provider         | `github.com/plaid/plaid-go/v<latest major>/plaid`                                                                                 |
| Metrics          | `github.com/prometheus/client_golang`                                                                                             |
| Scheduling       | `github.com/robfig/cron/v3` — **parser only** (`cron.ParseStandard` → `Schedule.Next`); our own goroutine does the waiting        |
| Crypto           | stdlib `crypto/aes`, `crypto/cipher` (`NewGCMWithRandomNonce`)                                                                    |
| Tests (Go)       | stdlib `testing`, `humatest`, hand-written interface fakes (no mocking/assert library)                                            |
| Go lint          | `golangci-lint` v2 (`gofumpt`, `goimports` formatters) + `tools/nocomments`                                                       |
| JS runtime       | Node LTS, **pnpm only** (`packageManager` + `devEngines.packageManager.onFail: "error"`)                                          |
| Build            | `vite`, `@vitejs/plugin-react`, `@tailwindcss/vite`, `tailwindcss`, `vite-plugin-pwa`, `typescript`                               |
| Routing          | `react-router` with an explicit route table; `zod` for search parameters                                                          |
| Data             | `@tanstack/react-query`, `openapi-fetch`, `openapi-react-query`, `openapi-typescript`                                             |
| UI               | `shadcn` CLI (`pnpm dlx shadcn@latest`), `react-plaid-link`, `lucide-react`, `sonner`                                             |
| Lint/format (JS) | **oxlint** (+ `oxlint-tsgolint` for type-aware rules, `eslint-plugin-react-hooks` loaded as an oxlint JS plugin) and **oxfmt**    |

`just outdated` (§8) runs `go list -u -m all` and `pnpm outdated` so drift is visible.

## 3. Configuration

Handwritten, stdlib only. `config.NewFromEnv()` reads each variable with `os.LookupEnv`, applies defaults, collects **every** problem (so one run reports all missing/invalid vars), then runs cross-field validation. The Justfile has `set dotenv-load`, so `.env` feeds local runs; production gets real env. Per your instruction, the config package has no tests.

`internal/config/config.go`:

```go
package config

import (
	"errors"
	"fmt"
	"time"

	"github.com/robfig/cron/v3"
)

type Environment string

const (
	Development Environment = "development"
	Production  Environment = "production"
)

type Config struct {
	AppEnv            Environment
	HTTPAddr          string
	DatabaseURL       string
	PublicURL         string
	StaticDir         string
	CookieDomain      string
	CookieSecure      bool
	SessionTTL        time.Duration
	SyncSchedule      cron.Schedule
	GoogleClientID    string
	GoogleSecret      string
	SecretKey         []byte
	PlaidCountryCodes []string
	Version           string
}

func NewFromEnv() (Config, error) {
	env := &envReader{}

	cfg := Config{
		AppEnv:            Environment(env.optional("APP_ENV", string(Production))),
		HTTPAddr:          env.optional("HTTP_ADDR", ":8080"),
		DatabaseURL:       env.required("DATABASE_URL"),
		PublicURL:         env.required("PUBLIC_URL"),
		StaticDir:         env.optional("STATIC_DIR", ""),
		CookieDomain:      env.optional("COOKIE_DOMAIN", ""),
		CookieSecure:      env.boolean("COOKIE_SECURE", true),
		SessionTTL:        env.duration("SESSION_TTL", 30*24*time.Hour),
		SyncSchedule:      env.schedule("SYNC_CRON", "0 */6 * * *"),
		GoogleClientID:    env.required("GOOGLE_CLIENT_ID"),
		GoogleSecret:      env.required("GOOGLE_CLIENT_SECRET"),
		SecretKey:         []byte(env.required("SECRET_KEY")),
		PlaidCountryCodes: env.list("PLAID_COUNTRY_CODES", []string{"US"}),
		Version:           env.optional("VERSION", ""),
	}

	env.errs = append(env.errs, cfg.validate()...)

	if err := errors.Join(env.errs...); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func (c Config) validate() []error {
	var errs []error

	if c.AppEnv != Development && c.AppEnv != Production {
		errs = append(errs, fmt.Errorf("APP_ENV must be %q or %q", Development, Production))
	}

	if len(c.SecretKey) != 32 {
		errs = append(errs, errors.New("SECRET_KEY must be exactly 32 characters"))
	}

	if c.AppEnv == Production && c.CookieDomain == "" {
		errs = append(errs, errors.New("COOKIE_DOMAIN is required in production"))
	}

	if c.AppEnv == Production && !c.CookieSecure {
		errs = append(errs, errors.New("COOKIE_SECURE must be true in production"))
	}

	return errs
}

func (c Config) GoogleRedirectURL() string {
	return c.PublicURL + "/auth/google/callback"
}

func (c Config) PlaidRedirectURI() string {
	return c.PublicURL + "/connections/oauth"
}
```

`internal/config/env.go` (moved out of `config.go` per rule c):

```go
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type envReader struct {
	errs []error
}

func (e *envReader) required(key string) string {
	value := strings.TrimSpace(os.Getenv(key))

	if value == "" {
		e.errs = append(e.errs, fmt.Errorf("%s is required", key))
	}

	return value
}

func (e *envReader) optional(key, fallback string) string {
	value, ok := os.LookupEnv(key)

	if !ok || strings.TrimSpace(value) == "" {
		return fallback
	}

	return strings.TrimSpace(value)
}

func (e *envReader) boolean(key string, fallback bool) bool {
	raw := e.optional(key, strconv.FormatBool(fallback))

	value, err := strconv.ParseBool(raw)
	if err != nil {
		e.errs = append(e.errs, fmt.Errorf("%s must be true or false: %w", key, err))
	}

	return value
}

func (e *envReader) duration(key string, fallback time.Duration) time.Duration {
	raw := e.optional(key, fallback.String())

	value, err := time.ParseDuration(raw)
	if err != nil {
		e.errs = append(e.errs, fmt.Errorf("%s must be a duration such as 6h: %w", key, err))
	}

	return value
}

func (e *envReader) schedule(key, fallback string) cron.Schedule {
	raw := e.optional(key, fallback)

	if raw == "off" {
		return nil
	}

	schedule, err := cron.ParseStandard(raw)
	if err != nil {
		e.errs = append(e.errs, fmt.Errorf("%s must be a 5-field cron expression or off: %w", key, err))
	}

	return schedule
}

func (e *envReader) list(key string, fallback []string) []string {
	raw := e.optional(key, "")

	if raw == "" {
		return fallback
	}

	var values []string

	for item := range strings.SplitSeq(raw, ",") {
		if trimmed := strings.TrimSpace(item); trimmed != "" {
			values = append(values, trimmed)
		}
	}

	return values
}
```

`.env.example` (the comment is the one you asked for):

```
APP_ENV=development
HTTP_ADDR=:8080

DATABASE_URL=postgres://fin:fin@localhost:5433/fin?sslmode=disable

PUBLIC_URL=http://localhost:5173
STATIC_DIR=

COOKIE_DOMAIN=
COOKIE_SECURE=true
SESSION_TTL=720h

GOOGLE_CLIENT_ID=
GOOGLE_CLIENT_SECRET=

# Generate with: LC_ALL=C tr -dc 'A-Za-z0-9' < /dev/urandom | head -c 32 | pbcopy
SECRET_KEY=

SYNC_CRON="0 */6 * * *"
PLAID_COUNTRY_CODES=US

VERSION=
```

Dev `PUBLIC_URL` points at the Vite dev server so OAuth redirects and cookies land on the same origin the SPA uses (Vite proxies `/api`, `/auth`, `/openapi.json`, `/swagger` to `:8080`). Empty `COOKIE_DOMAIN` → host-only cookie (browsers mishandle `Domain=localhost`). `SYNC_CRON=off` disables the scheduler; `CRON_TZ=America/New_York 0 6 * * *` pins a timezone (`main.go` blank-imports `time/tzdata` because the distroless image has no zoneinfo). The `env.go` import block gains `github.com/robfig/cron/v3`.

## 4. Database

Postgres 19. Migrations are sequential (`000001_...`), one table per migration, each with a `.down.sql` that drops it. SQL style used everywhere (migrations and Go): uppercase keywords, one column per line, aligned types, explicit `AS` aliases, trailing semicolon.

```sql
CREATE TABLE users (
    id          uuid        PRIMARY KEY DEFAULT uuidv7(),
    email       text        NOT NULL,
    google_sub  text        UNIQUE,
    name        text,
    created_at  timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX users_email_lower_idx ON users (lower(email));
```

```sql
CREATE TABLE sessions (
    id          uuid        PRIMARY KEY DEFAULT uuidv7(),
    user_id     uuid        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    token_hash  bytea       NOT NULL UNIQUE,
    created_at  timestamptz NOT NULL DEFAULT now(),
    expires_at  timestamptz NOT NULL
);

CREATE INDEX sessions_user_id_idx ON sessions (user_id);
```

```sql
CREATE TABLE user_provider_credentials (
    user_id      uuid        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    provider     text        NOT NULL,
    environment  text        NOT NULL,
    client_id    text        NOT NULL,
    secret       text        NOT NULL,
    created_at   timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, provider)
);
```

```sql
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
```

```sql
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
```

Migration `000008_add_account_card_art` adds `card_art_id`, `card_art_url` and `card_art_source` (`text`, all nullable) to `accounts`. `card_art_source` is `matched` (set by sync) or `selected` (set by the user); `NULL` means the account has not been matched yet. Account upserts never touch these columns.

```sql
CREATE TABLE transactions (
    id             uuid           PRIMARY KEY DEFAULT uuidv7(),
    user_id        uuid           NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    account_id     uuid           NOT NULL REFERENCES accounts (id) ON DELETE CASCADE,
    external_id    text           NOT NULL,
    amount         numeric(19, 4) NOT NULL,
    currency       text,
    posted_on      date           NOT NULL,
    authorized_on  date,
    name           text           NOT NULL,
    merchant_name  text,
    category       text,
    pending        boolean        NOT NULL DEFAULT false,
    created_at     timestamptz    NOT NULL DEFAULT now(),
    updated_at     timestamptz    NOT NULL DEFAULT now(),
    UNIQUE (account_id, external_id)
);

CREATE INDEX transactions_user_posted_idx ON transactions (user_id, posted_on DESC, id DESC);
CREATE INDEX transactions_account_posted_idx ON transactions (account_id, posted_on DESC, id DESC);
```

Onboarding a user = `just seed <email> <plaid-client-id> <plaid-secret> [sandbox|production]` (§5.1). Point `DATABASE_URL` at production to onboard the real two users the same way.

Plaid access tokens are encrypted at rest (§5.11). Per-user Plaid `secret`s stay plaintext because they are inserted by hand; protect the DB (network + backups).

## 5. Backend

### 5.1 Commands (`main.go`)

`main` → `run(os.Args[1:])` → switch on first arg (default `serve`):

- `serve` (`serve.go`): `config.NewFromEnv` → `encryption.New(cfg.SecretKey)` → `pgxpool.New` → construct every postgres repo → `recorder := metrics.Prometheus{}` → `metrics.WatchDatabase(metricsRepo, pool)` → Plaid provider (shared `*http.Client` with timeout, recorder) → Google identity provider (`oidc.NewProvider` for `https://accounts.google.com`) → services → handlers → `go scheduler.Run(ctx)` → `server.WithOps(server.Observe(mux, …), health)` → one `http.Server` with `ReadHeaderTimeout`, run until `signal.NotifyContext(SIGINT, SIGTERM)`, then `Shutdown` with timeout. All DI is explicit constructor calls in this one function (requirement m).
- `openapi` (`openapi.go`): builds the huma API on a throwaway mux with `server.Handlers{}` (nil handler pointers are fine: registration reads types only), writes `api.OpenAPI()` as indented JSON to stdout. No env, no DB. `just gen` redirects it to `./openapi.json` (gitignored).

Migrations are not a command of this binary: `just migrate`, `just migrate-down N`, `just migrate-version` and `just migration <name>` call the golang-migrate CLI directly.

`cmd/sandbox-reset-login/main.go` (dev-only, `just sandbox-reset-login <connection-id>`): loads the connection via `ConnectionRepository.FindByID`, refuses unless the owner's credential environment is `sandbox`, decrypts the token, calls the Plaid provider's `ResetLogin` (`/sandbox/item/reset_login`). Not part of `provider.Provider` and not in the production binary.

`cmd/seed/main.go` — a separate dev/onboarding binary. It reads `DATABASE_URL` from the environment (`just` loads `.env`), takes everything else as flags, and reuses the postgres repositories instead of carrying its own SQL. Both writes are upserts, so re-running with the same email updates the credential instead of creating duplicate rows:

```go
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/tahminator/fin/internal/domain"
	"github.com/tahminator/fin/internal/repository/postgres"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	email := flag.String("email", "", "Google account email to allow")
	clientID := flag.String("client-id", "", "Plaid client ID")
	secret := flag.String("secret", "", "Plaid secret")
	environment := flag.String("env", "sandbox", "Plaid environment: sandbox or production")

	flag.Parse()

	plaidEnv := domain.ProviderEnvironment(*environment)

	if *email == "" || *clientID == "" || *secret == "" || !plaidEnv.Valid() {
		flag.Usage()

		return errors.New("email, client-id, secret and a valid env are required")
	}

	ctx := context.Background()

	pool, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}

	defer pool.Close()

	user, err := postgres.NewUserRepository(pool).Upsert(ctx, *email)
	if err != nil {
		return fmt.Errorf("upsert user: %w", err)
	}

	err = postgres.NewCredentialRepository(pool).Upsert(ctx, domain.Credential{
		UserID:      user.ID,
		Provider:    domain.ProviderPlaid,
		Environment: plaidEnv,
		ClientID:    *clientID,
		Secret:      *secret,
	})
	if err != nil {
		return fmt.Errorf("upsert credential: %w", err)
	}

	fmt.Printf("allowed %s with %s %s credentials\n", user.Email, domain.ProviderPlaid, plaidEnv)

	return nil
}
```

### 5.2 Layers and dependency direction

```mermaid
flowchart LR
  routes[server/routes.go] --> handlers[handler/*]
  handlers --> services[service/*]
  handlers --> repoif[repository interfaces]
  services --> repoif
  services --> prov[provider.Provider]
  services --> ident[identity.IdentityProvider]
  services --> enc[encryption.Cipher]
  pg[repository/postgres] -. implements .-> repoif
  plaid[provider/plaid] -. implements .-> prov
  google[identity/google.go] -. implements .-> ident
```

- Handlers depend on **interfaces** (repositories) and **concrete services**; services depend only on interfaces. Read-only endpoints (accounts, transactions, summary) use repositories directly — no pass-through services.
- Every repository method that reads or mutates user data takes `userID` and filters `WHERE user_id = $n` — tenancy is enforced in SQL, not in handlers.
- Repositories return `domain.ErrNotFound` for `pgx.ErrNoRows`; handlers map sentinel errors to `huma.Error404NotFound` / `Error403Forbidden` / `Error409Conflict`, and `domain.ErrUnavailable` (an upstream such as the card catalog failed) to `Error503ServiceUnavailable`.

### 5.3 Repository interfaces

Example shape (one file per interface in `internal/repository/`):

```go
type TransactionRepository interface {
	ListByUser(ctx context.Context, userID uuid.UUID, filter TransactionFilter, limit, offset int) ([]domain.Transaction, int, error)
	Upsert(ctx context.Context, userID uuid.UUID, transactions []domain.Transaction) error
	DeleteByExternalIDs(ctx context.Context, userID uuid.UUID, externalIDs []string) error
}
```

Full set: `UserRepository` (`FindByEmail`, `FindByID`, `Upsert(email)` (`ON CONFLICT ((lower(email)))`), `SetGoogleSub`, `Count`), `SessionRepository` (`Create`, `FindValidByTokenHash`, `Delete`, `DeleteExpired`), `CredentialRepository` (`Find(userID, provider)`, `Upsert(credential)` (`ON CONFLICT (user_id, provider) DO UPDATE`)), `ConnectionRepository` (`Create`, `Get(userID, id)`, `FindByID(id)`, `ListByUser`, `ListByStatus(status)`, `ListByUserAndStatus(userID, status, limit, offset)`, `UpdateSyncState`, `UpdateStatus` (moves `status_changed_at` only when the status actually changes), `Delete`), `AccountRepository` (`Upsert` returning `map[externalID]accountID`, `ListByUser`), `TransactionRepository`, `SummaryRepository` (`Balances(userID)`), `MetricsRepository` (`Snapshot`), `Pinger` (`Ping`).

Postgres impls: SQL as a `const` next to the method that uses it, scanned with `pgx.CollectRows(rows, pgx.RowToStructByName[...])` into domain structs (`db` tags). Numeric columns are selected as `amount::text` and written as `$n::numeric`, so Go only handles strings. List methods run two readable queries (page + `COUNT(*)`).

```go
const listTransactionsByUser = `
SELECT
    t.id,
    t.account_id,
    t.amount::text AS amount,
    t.currency,
    t.posted_on,
    t.name,
    t.merchant_name,
    t.category,
    t.pending
FROM transactions AS t
WHERE t.user_id = $1
  AND ($2::uuid IS NULL OR t.account_id = $2)
ORDER BY t.posted_on DESC, t.id DESC
LIMIT $3
OFFSET $4;
`
```

Unit tests use thin fakes that embed the interface and override only what the test needs:

```go
type fakeConnections struct {
	repository.ConnectionRepository
	get func(ctx context.Context, userID, id uuid.UUID) (domain.Connection, error)
}

func (f fakeConnections) Get(ctx context.Context, userID, id uuid.UUID) (domain.Connection, error) {
	return f.get(ctx, userID, id)
}
```

### 5.4 Provider abstraction

```go
type Provider interface {
	CreateLinkToken(ctx context.Context, creds domain.Credential, userID, redirectURI string) (string, error)
	CreateUpdateLinkToken(ctx context.Context, creds domain.Credential, userID, accessToken, redirectURI string) (string, error)
	Exchange(ctx context.Context, creds domain.Credential, publicToken string) (LinkedItem, error)
	Accounts(ctx context.Context, creds domain.Credential, accessToken string) ([]Account, error)
	SyncTransactions(ctx context.Context, creds domain.Credential, accessToken, cursor string) (SyncResult, error)
	Remove(ctx context.Context, creds domain.Credential, accessToken string) error
}
```

Plaid implementation notes:

- A Plaid client is built per call from `domain.Credential` (`PLAID-CLIENT-ID`/`PLAID-SECRET` headers, `Sandbox`/`Production` from `environment`). Client construction is cheap; the `*http.Client` is shared.
- Link token: products `transactions`, `transactions.days_requested = 730`, `client_user_id = user UUID`, `redirect_uri = cfg.PlaidRedirectURI()` (must be registered in **each user's** Plaid dashboard — OAuth banks like Chase require it on mobile web).
- `SyncTransactions` loops `/transactions/sync` until `has_more=false`; on `TRANSACTIONS_SYNC_MUTATION_DURING_PAGINATION` it restarts from the original cursor.
- Mapping: amounts formatted with `strconv.FormatFloat(-amount, 'f', -1, 64)` (Plaid positive = outflow; ours negative = outflow). Plaid `ITEM_LOGIN_REQUIRED` → `domain.ErrReauthRequired`; extend that mapping if other update-mode errors show up in practice.
- Update mode: `/link/token/create` with the item's `access_token` and no `products`. Success needs no token exchange; the item keeps its access token.
- Accepts **plaintext** access tokens only. Encryption is the service's job, so the provider stays swappable and ignorant of storage.

### 5.5 Services

`AuthService` (`service/auth.go`)

- `BeginLogin()` → `state`, `nonce`, PKCE verifier (`oauth2.GenerateVerifier`) and the Google auth URL (`oauth2.S256ChallengeOption`, scopes `openid email profile`).
- `CompleteLogin(ctx, code, verifier, nonce)` → identity provider exchanges code and verifies the ID token (issuer, audience, nonce) → requires `email_verified` → `UserRepository.FindByEmail` (case-insensitive) → missing = `domain.ErrNotAllowed` → if `google_sub` is null set it, else it must match → `SessionRepository.DeleteExpired` → create session.
- Session token: 32 bytes from `crypto/rand`, base64url in the cookie, SHA-256 stored in `token_hash`.
- `ResolveSession(ctx, token)` → `domain.User`; `Logout(ctx, token)`.

`ConnectionService` (`service/connection.go`; constructed with repositories, `provider.Provider`, `*encryption.Cipher`, `metrics.Recorder`)

- `CreateLinkToken(userID)` → loads credential (missing = `ErrNotAllowed` with a clear message).
- `Link(userID, publicToken)` → exchange → `cipher.Encrypt(accessToken)` → insert connection (`status=active`) → `Sync(trigger=SyncTriggerLink)`.
- `Sync(userID, connectionID, trigger metrics.SyncTrigger)` → load connection + credential → `cipher.Decrypt` → `Accounts` → `AccountRepository.Upsert` → `CardArtMatcher.MatchConnection` → `SyncTransactions(cursor)` → map account external IDs → upsert added+modified → delete removed → `UpdateSyncState(cursor, last_synced_at=now, last_error=null)`. On `ErrReauthRequired`: `UpdateStatus(needs_reauth)`, cursor untouched. Other errors: `last_error` set, status unchanged. Always ends with `recorder.ConnectionSynced(trigger, outcome, elapsed)`.
- `SyncAll(userID, trigger)` → every `active` connection of the user, sequentially → `SyncSummary{Synced, ReauthRequired, Failed}`. Backs the manual **Refresh** button.
- `CreateReauthLinkToken(userID, connectionID)` → `ErrConflict` unless `needs_reauth` → decrypt → `CreateUpdateLinkToken`.
- `CompleteReauth(userID, connectionID)` → `UpdateStatus(active)`, clear `last_error` → `Sync(trigger=SyncTriggerReauth)`.
- `Remove(userID, connectionID)` → decrypt → provider `Remove` → delete row (cascades accounts/transactions).

Manual and scheduled syncs of the same connection may overlap; that is harmless because every write is an idempotent upsert and the cursor is saved last.

`CardArtService` (`service/card_art.go`; constructed with `AccountRepository` and `cardart.Catalog`, implemented by `cardart.OpenCard` against `https://opencard-db.davidweng.workers.dev/v1` with a 10s timeout). Plaid exposes no card product identifier, so art is matched by name once and stored per account:

- `MatchConnection(userID, connectionID)` runs inside sync for credit accounts whose `card_art_source` is `NULL` (the catalog has no debit card art, so checking accounts keep the generic card icon). It searches the catalog for the official name (falling back to `name`) plus the name without "authorized user"/"additional"/"supplementary", scores candidates by shared distinctive words (generic words such as card, credit, visa are ignored) with an issuer bonus, and saves the best card as `matched`, or an empty `matched` result when nothing scores. Catalog failures leave the account `NULL` so the next sync retries; they never fail the sync. The save only applies while the source is still `NULL`, so it cannot overwrite a user's choice.
- `Search(query)` proxies catalog search; cards without artwork (or with the catalog placeholder) are dropped.
- `Select(userID, accountID, cardID)` resolves the card from the catalog and stores it as `selected`; a `nil` card ID selects the default icon. Selected art is never re-matched.

`NotificationService` (`service/notification.go`)

- `List(userID, limit, offset)` → `ListByUserAndStatus(needs_reauth)` → `domain.Notification{Kind: ReauthRequired, ConnectionID, InstitutionName, Since: status_changed_at}`, newest first.

`SyncScheduler` (`service/scheduler.go`) — one goroutine started from `serve.go`, waits for each cron tick, never overlaps itself:

```go
type connectionSyncer interface {
	Sync(ctx context.Context, userID, connectionID uuid.UUID, trigger metrics.SyncTrigger) error
}

type SyncScheduler struct {
	schedule    cron.Schedule
	connections repository.ConnectionRepository
	syncer      connectionSyncer
	recorder    metrics.Recorder
	logger      *slog.Logger
	now         func() time.Time
}

func (s *SyncScheduler) Run(ctx context.Context) {
	if s.schedule == nil {
		return
	}

	for {
		wait := s.schedule.Next(s.now()).Sub(s.now())
		timer := time.NewTimer(wait)

		select {
		case <-ctx.Done():
			timer.Stop()

			return
		case <-timer.C:
			s.RunOnce(ctx)
		}
	}
}

func (s *SyncScheduler) RunOnce(ctx context.Context) {
	started := s.now()

	connections, err := s.connections.ListByStatus(ctx, domain.ConnectionStatusActive)
	if err != nil {
		s.logger.ErrorContext(ctx, "scheduled sync: list connections", "err", err)
		s.finish(metrics.ScheduledRunFailure, 0, started)

		return
	}

	failed := 0

	for _, connection := range connections {
		err := s.syncer.Sync(ctx, connection.UserID, connection.ID, metrics.SyncTriggerScheduled)
		if err != nil {
			failed++
			s.logger.WarnContext(ctx, "scheduled sync: connection failed", "connection_id", connection.ID, "err", err)
		}
	}

	outcome := metrics.ScheduledRunSuccess

	if failed > 0 {
		outcome = metrics.ScheduledRunPartialFailure
	}

	s.finish(outcome, failed, started)
}

func (s *SyncScheduler) finish(outcome metrics.ScheduledRunOutcome, failed int, started time.Time) {
	finished := s.now()

	s.recorder.ScheduledRunFinished(metrics.ScheduledRun{
		Outcome:           outcome,
		FailedConnections: failed,
		Elapsed:           finished.Sub(started),
		FinishedAt:        finished,
	})
}
```

A connection that hits re-auth counts as a failed connection for the run (it was not synced) and is skipped by later runs until reconnected, because only `active` connections are listed.

### 5.6 HTTP surface

`server/routes.go` is the single routing table:

```go
func RegisterRoutes(mux *http.ServeMux, api huma.API, h Handlers) {
	mux.HandleFunc("GET /auth/google/login", h.Auth.Login)
	mux.HandleFunc("GET /auth/google/callback", h.Auth.Callback)

	authed := huma.NewGroup(api, "/api")
	authed.UseMiddleware(h.RequireSession)
	h.Auth.Register(authed)
	h.Summary.Register(authed)
	h.Connections.Register(authed)
	h.Accounts.Register(authed)
	h.CardArt.Register(authed)
	h.Transactions.Register(authed)
	h.Notifications.Register(authed)

	mux.Handle("/api/", http.NotFoundHandler())

	if h.SPA != nil {
		mux.Handle("GET /", h.SPA)
	}
}
```

Each handler file groups all operations for one resource behind `Register(api huma.API)`:

```go
func (h *TransactionHandler) Register(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "list-transactions",
		Method:      http.MethodGet,
		Path:        "/transactions",
		Tags:        []string{"transactions"},
		Security:    cookieAuth,
	}, h.list)
}
```

| Method | Path                                                | Handler                             | Notes                                                                                                                                                                       |
| ------ | --------------------------------------------------- | ----------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| GET    | `/auth/google/login`                                | `AuthHandler.Login`                 | sets `fin_oauth` cookie, 302 to Google                                                                                                                                      |
| GET    | `/auth/google/callback`                             | `AuthHandler.Callback`              | sets `fin_session`, 303 to `/` (or `/login?error=not_allowed`)                                                                                                              |
| GET    | `/api/me`                                           | `AuthHandler.me`                    | 401 when no session                                                                                                                                                         |
| POST   | `/api/auth/logout`                                  | `AuthHandler.logout`                | deletes session, expires cookie                                                                                                                                             |
| GET    | `/api/summary`                                      | `SummaryHandler`                    | per-currency assets / liabilities / net; connection + account counts                                                                                                        |
| GET    | `/api/summary/cash-flow`                            | `SummaryHandler.cashFlow`           | `?days=` 1–365, default 30; per-currency daily money in / out (out positive), zero-filled through today; excludes `TRANSFER_IN`, `TRANSFER_OUT`, `LOAN_PAYMENTS` categories |
| GET    | `/api/connections`                                  | `ConnectionHandler.list`            | `Page[Connection]`                                                                                                                                                          |
| POST   | `/api/connections/link-token`                       | `ConnectionHandler.linkToken`       | `{linkToken}`                                                                                                                                                               |
| POST   | `/api/connections`                                  | `ConnectionHandler.create`          | body `{publicToken}`; links + initial sync                                                                                                                                  |
| POST   | `/api/connections/{connectionId}/sync`              | `ConnectionHandler.sync`            |                                                                                                                                                                             |
| POST   | `/api/connections/sync`                             | `ConnectionHandler.syncAll`         | manual refresh of all the user's active connections → `SyncSummary`                                                                                                         |
| DELETE | `/api/connections/{connectionId}`                   | `ConnectionHandler.remove`          | 204                                                                                                                                                                         |
| POST   | `/api/connections/{connectionId}/reauth/link-token` | `ConnectionHandler.reauthLinkToken` | update-mode `{linkToken}`; 409 unless `needs_reauth`                                                                                                                        |
| POST   | `/api/connections/{connectionId}/reauth/complete`   | `ConnectionHandler.completeReauth`  | marks `active`, syncs                                                                                                                                                       |
| GET    | `/api/accounts`                                     | `AccountHandler.list`               | `Page[Account]`, optional `connectionId`; includes `cardArtId`, `cardArtUrl`, `cardArtSource`                                                                               |
| GET    | `/api/card-art/search`                              | `CardArtHandler.search`             | `?query=` (1–100 chars, not blank) → `{items: CatalogCard[]}`; 503 when the catalog is unavailable                                                                          |
| PUT    | `/api/accounts/{accountId}/card-art`                | `CardArtHandler.selectCardArt`      | body `{cardId: string \| null}` → `Account`; 404 for unknown account or card                                                                                                |
| GET    | `/api/transactions`                                 | `TransactionHandler.list`           | `Page[Transaction]`, optional `accountId`                                                                                                                                   |
| GET    | `/api/notifications`                                | `NotificationHandler.list`          | `Page[Notification]`; currently only `reauth_required`                                                                                                                      |
| GET    | `/version`                                          | `VersionHandler.get`                | public; `{version}` from runtime `VERSION` (k8s sets it to the image tag), `null` when unset                                                                                |
| GET    | `/openapi.json`, `/openapi.yaml`                    | huma                                |                                                                                                                                                                             |
| GET    | `/swagger`                                          | huma SwaggerUI                      |                                                                                                                                                                             |
| GET    | `/*`                                                | `server/spa.go`                     | only when `STATIC_DIR` set                                                                                                                                                  |
| GET    | `/healthz`                                          | `HealthHandler.live`                | always 200; `server/ops.go`, not observed                                                                                                                                   |
| GET    | `/readyz`                                           | `HealthHandler.ready`               | `Pinger.Ping` (`SELECT 1`, 2s timeout) → 200 / 503                                                                                                                          |
| GET    | `/metrics`                                          | promhttp on custom registry         | blocked at the ingress                                                                                                                                                      |

Huma config (`server/api.go`): `huma.DefaultConfig("fin", version)`, `OpenAPIPath = "/openapi"` (serves `.json` and `.yaml`), `DocsPath = "/swagger"`, `DocsRenderer = huma.DocsRendererSwaggerUI`, `Components.SecuritySchemes["cookieAuth"] = {Type: "apiKey", In: "cookie", Name: "fin_session"}`, `CreateHooks = nil` (drops the `$schema` response field so generated TS types stay clean).

Pagination types (`handler/page.go`):

```go
type PageParams struct {
	Page     int `query:"page" minimum:"1" default:"1"`
	PageSize int `query:"pageSize" minimum:"1" maximum:"100" default:"25"`
}

type Page[T any] struct {
	Items    []T  `json:"items"`
	Page     int  `json:"page"`
	PageSize int  `json:"pageSize"`
	Total    int  `json:"total"`
	HasNext  bool `json:"hasNext"`
}
```

`PageParams` is embedded in list inputs; `NewPage` guarantees `Items` is never `nil`. API DTOs live in `handler` (json + huma `doc`/`enum` tags) and are mapped from domain types, so internal fields (access tokens, cursors) can never leak by tag mistake. Schema names come out as `Connection`, `PageConnection`, `Transaction`, `PageTransaction`, …

### 5.7 Cookies and auth details

| Cookie        | Attributes                                                                             | Purpose                       |
| ------------- | -------------------------------------------------------------------------------------- | ----------------------------- |
| `fin_session` | `HttpOnly; Secure; SameSite=Strict; Path=/; Domain=COOKIE_DOMAIN; Max-Age=SESSION_TTL` | session                       |
| `fin_oauth`   | `HttpOnly; Secure; SameSite=Lax; Path=/auth/google/callback; Max-Age=600`              | state + nonce + PKCE verifier |

- `fin_oauth` must be `Lax`: the callback is a cross-site top-level navigation from Google, and `Strict` cookies are not sent on it.
- With `Strict`, the HTML load right after the callback redirect may not carry `fin_session`. That is fine by design: the SPA shell is public and auth is decided by the same-origin `fetch('/api/me')`, which does carry it.
- `RequireSession` (huma middleware on the `/api` group) reads the cookie via `huma.ReadCookie`, resolves the user, stores it with `huma.WithValue`; handlers read it via `auth.UserFrom(ctx)`. Missing/invalid → `huma.WriteErr(api, ctx, 401, ...)`.
- CSRF: covered by `SameSite=Strict` + JSON-only request bodies.

### 5.8 Observability

`internal/metrics` is the only place that touches Prometheus. Everything else depends on the `metrics.Recorder` interface and passes enum values; there is no exported counter, gauge, label string or registry.

**Labels — `labels.go`.** Each label is an `iota` enum with an exhaustive `String()` switch (the `exhaustive` linter fails the build when a new constant lacks a case) and an unexported `all…()` list used to pre-initialize every series to zero:

```go
type SyncTrigger int

const (
	SyncTriggerLink SyncTrigger = iota
	SyncTriggerManual
	SyncTriggerScheduled
	SyncTriggerReauth
)

func (t SyncTrigger) String() string {
	switch t {
	case SyncTriggerLink:
		return "link"
	case SyncTriggerManual:
		return "manual"
	case SyncTriggerScheduled:
		return "scheduled"
	case SyncTriggerReauth:
		return "reauth"
	}

	return "unknown"
}

func allSyncTriggers() []SyncTrigger {
	return []SyncTrigger{SyncTriggerLink, SyncTriggerManual, SyncTriggerScheduled, SyncTriggerReauth}
}
```

| Enum                  | Values                                                                                                 |
| --------------------- | ------------------------------------------------------------------------------------------------------ |
| `SyncTrigger`         | `link`, `manual`, `scheduled`, `reauth`                                                                |
| `SyncOutcome`         | `success`, `reauth_required`, `error`                                                                  |
| `ScheduledRunOutcome` | `success`, `partial_failure`, `failure`                                                                |
| `LoginOutcome`        | `success`, `not_allowed`, `invalid_state`, `error`                                                     |
| `Provider`            | `plaid`                                                                                                |
| `ProviderOperation`   | `create_link_token`, `create_update_link_token`, `exchange`, `accounts`, `sync_transactions`, `remove` |
| `CallOutcome`         | `success`, `error`                                                                                     |
| `RouteGroup`          | `api`, `auth`, `docs`, `static`                                                                        |
| `Method`              | `get`, `post`, `put`, `patch`, `delete`, `other`                                                       |
| `StatusClass`         | `1xx`, `2xx`, `3xx`, `4xx`, `5xx` (derived inside the package from the status code)                    |
| `ConnectionStatus`    | `active`, `needs_reauth`, `other` (mapped from `domain.ConnectionStatus` inside the package)           |

Domain enums stay `string`-typed because they round-trip through Postgres and JSON; metric enums are `iota` because they only ever become label values.

**Recording — `recorder.go`.**

```go
type Recorder interface {
	HTTPRequestServed(group RouteGroup, method Method, status int, elapsed time.Duration)
	LoginAttempted(outcome LoginOutcome)
	ProviderCalled(provider Provider, operation ProviderOperation, outcome CallOutcome, elapsed time.Duration)
	ConnectionSynced(trigger SyncTrigger, outcome SyncOutcome, elapsed time.Duration)
	ScheduledRunFinished(run ScheduledRun)
}

type Prometheus struct{}

func (Prometheus) ConnectionSynced(trigger SyncTrigger, outcome SyncOutcome, elapsed time.Duration) {
	syncsTotal.WithLabelValues(trigger.String(), outcome.String()).Inc()
	syncDuration.WithLabelValues(trigger.String()).Observe(elapsed.Seconds())
}

type Noop struct{}

func (Noop) ConnectionSynced(SyncTrigger, SyncOutcome, time.Duration) {}
```

Both implementations are zero-sized. `Prometheus{}` writes to the package-level unexported instruments in `instruments.go`; `Noop{}` is for tests that don't care. Tests that assert recording embed `metrics.Noop` and override one method — the same pattern as the repository fakes in §5.3.

**Recorded metrics** (callers in parentheses):

| Metric                                                               | Type                | Labels                             | Max series   |
| -------------------------------------------------------------------- | ------------------- | ---------------------------------- | ------------ |
| `fin_http_requests_total` (server middleware)                        | counter             | `group`, `method`, `status_class`  | 120          |
| `fin_http_request_duration_seconds`                                  | histogram           | `group`, `method`                  | 24 × buckets |
| `fin_logins_total` (`AuthService`)                                   | counter             | `outcome`                          | 4            |
| `fin_provider_requests_total` / `_duration_seconds` (Plaid provider) | counter / histogram | `provider`, `operation`, `outcome` | 12           |
| `fin_syncs_total` (`ConnectionService.Sync`)                         | counter             | `trigger`, `outcome`               | 12           |
| `fin_sync_duration_seconds`                                          | histogram           | `trigger`                          | 4 × buckets  |
| `fin_scheduled_sync_runs_total` (`SyncScheduler`)                    | counter             | `outcome`                          | 3            |
| `fin_scheduled_sync_duration_seconds`                                | histogram           | —                                  | buckets      |
| `fin_scheduled_sync_last_run_failed_connections`                     | gauge               | —                                  | 1            |
| `fin_scheduled_sync_last_run_timestamp_seconds`                      | gauge               | —                                  | 1            |
| `fin_scheduled_sync_last_success_timestamp_seconds`                  | gauge               | —                                  | 1            |

**DB snapshot — `database.go`.** `metrics.WatchDatabase(repo, pool)` registers a collector that calls `MetricsRepository.Snapshot` (3s timeout) on each scrape, so values are always current without another goroutine. A failed snapshot sets `fin_metrics_snapshot_up` to 0.

| Metric                                                        | Labels                 |
| ------------------------------------------------------------- | ---------------------- |
| `fin_users`                                                   | —                      |
| `fin_users_with_credentials`                                  | `provider`             |
| `fin_active_sessions`                                         | `user`                 |
| `fin_connections`                                             | `user`, `status`       |
| `fin_connections_with_errors`                                 | `user`                 |
| `fin_accounts`                                                | `user`                 |
| `fin_transactions`                                            | `user`                 |
| `fin_connection_oldest_sync_timestamp_seconds`                | `user`                 |
| `fin_db_pool_acquired_conns` / `_idle_conns` / `_total_conns` | — (from `pool.Stat()`) |
| `fin_metrics_snapshot_up`                                     | —                      |

`user` is the one data-derived label: its values are emails from the `users` table only, so cardinality is the size of the allow-list (2) times the enum it is paired with.

**Alerts this enables:**

```
increase(fin_syncs_total{trigger="scheduled", outcome!="success"}[1d]) > 0
increase(fin_scheduled_sync_runs_total{outcome="failure"}[1d]) > 0
time() - fin_scheduled_sync_last_success_timestamp_seconds > 2 * 6 * 3600
fin_connections{status="needs_reauth"} > 0
```

Request logging: `slog` JSON handler, one line per request from the same middleware that calls `HTTPRequestServed`. The middleware maps the path prefix to a `RouteGroup` (`/api` → `api`, `/auth` → `auth`, `/openapi*` and `/swagger` → `docs`, everything else → `static`).

### 5.9 SPA serving (`server/spa.go`)

- If the requested path exists under `STATIC_DIR` → serve it. `/assets/*` gets `Cache-Control: public, max-age=31536000, immutable`; `index.html`, `sw.js`, `manifest.webmanifest` get `no-cache`.
- Otherwise → `index.html` (client-side routes). `/api/*` never falls through (explicit 404 in `routes.go`).

### 5.10 Go lint and style enforcement

`.golangci.yml` (v2 format): `linters.default: standard` plus `bodyclose`, `contextcheck`, `errorlint`, `exhaustive`, `gocritic`, `gosec`, `noctx`, `rowserrcheck`, `sqlclosecheck`, `unparam`, `misspell`. Formatters: `gofumpt`, `goimports` (local prefix `github.com/tahminator/fin`). Comment-demanding checks are disabled: `staticcheck.checks: ["all", "-ST1000", "-ST1020", "-ST1021", "-ST1022"]`, and `revive`'s `exported`/`package-comments` rules are not enabled.

`tools/nocomments/main.go`: walks `*.go` (skipping `js/`, `node_modules/`), parses with `parser.ParseComments`, and fails on any comment that is not a `//go:` directive. Run by `just lint-go`.

File-size rule (c): a type with more than a couple of small methods, or DTOs that push a handler file past one screen, move to a sibling file in the same package (`connection_types.go` pattern).

Natural newlines (non-negotiable, every language): blank line after a block of declarations, between independent statements groups, before `return` that follows other logic, and between functions/types. The idiomatic exception is Go's `value, err := f()` immediately followed by `if err != nil`. gofumpt and oxfmt preserve single blank lines, so the formatters never fight this rule; reviewers enforce it.

### 5.11 Token encryption

- `internal/encryption/cipher.go`: AES-256-GCM via `cipher.NewGCMWithRandomNonce`, so `Seal` generates and prepends the nonce. Output is base64 (std) in `connections.access_token_ciphertext`.
- Key = the 32 raw bytes of `SECRET_KEY` (32 chars of `[A-Za-z0-9]` ≈ 190 bits). `config.NewFromEnv` rejects any other length; `encryption.New` rejects it again.
- Only `ConnectionService` (and the dev-only sandbox command) encrypt/decrypt. Repositories store opaque strings; DTOs never carry the field.
- No versioning: a changed `SECRET_KEY` makes stored tokens undecryptable. That surfaces as a sync error; fix is remove + re-link.

```go
package encryption

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"fmt"
)

type Cipher struct {
	aead cipher.AEAD
}

func New(key []byte) (*Cipher, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create block cipher: %w", err)
	}

	aead, err := cipher.NewGCMWithRandomNonce(block)
	if err != nil {
		return nil, fmt.Errorf("create gcm: %w", err)
	}

	return &Cipher{aead: aead}, nil
}

func (c *Cipher) Encrypt(plaintext string) string {
	sealed := c.aead.Seal(nil, nil, []byte(plaintext), nil)

	return base64.StdEncoding.EncodeToString(sealed)
}

func (c *Cipher) Decrypt(encoded string) (string, error) {
	sealed, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("decode ciphertext: %w", err)
	}

	plaintext, err := c.aead.Open(nil, nil, sealed, nil)
	if err != nil {
		return "", fmt.Errorf("open ciphertext: %w", err)
	}

	return string(plaintext), nil
}
```

`aes.NewCipher` only accepts 16/24/32-byte keys; `New` additionally requires exactly 32 so AES-256 is guaranteed.

### 5.12 Re-auth detection and notification

```mermaid
flowchart LR
  sync["Sync (link / manual / scheduled)"] -->|ITEM_LOGIN_REQUIRED| flag["status = needs_reauth<br/>status_changed_at = now"]
  flag --> api["GET /api/notifications"]
  api --> ui["Banner + nav badge + Reconnect"]
  ui --> token["POST …/reauth/link-token"]
  token --> link["Plaid Link (update mode)"]
  link --> done["POST …/reauth/complete → active + sync"]
```

- Detection latency is at most one `SYNC_CRON` period even if nobody opens the app, because the scheduler syncs every active connection.
- `Notification` DTO (`handler/notification.go`):

```go
type Notification struct {
	Kind            string    `json:"kind" enum:"reauth_required"`
	ConnectionID    uuid.UUID `json:"connectionId"`
	InstitutionName string    `json:"institutionName"`
	Since           time.Time `json:"since"`
}
```

- `kind` is a Go enum exposed as a string union, so new notification kinds (e.g. repeated sync failures) can be added without breaking the client.
- Ops visibility: `fin_connections{status="needs_reauth"} > 0` is the alert expression.
- Frontend presentation: §6.7.

### 5.13 Backend tests

Every piece of business logic ships with unit tests in the same milestone. Tests use stdlib `testing`, table-driven `t.Run` cases, the embedded-interface fakes from §5.3, recorder fakes that embed `metrics.Noop`, and `humatest` for handlers. No network, no database, no sleeps (`RunOnce` instead of waiting for a tick; `now func() time.Time` injected where time matters). `just test` runs `go test -race ./...`.

| Package                                  | Covered behavior                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                      |
| ---------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `internal/encryption`                    | round-trip; same plaintext encrypts to different ciphertexts; flipped byte fails; wrong key fails; invalid base64 fails; `New` rejects keys that are not 32 bytes                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                     |
| `internal/service` — auth                | unverified email rejected; unknown email → `ErrNotAllowed`; email match is case-insensitive; first login stores `google_sub`; mismatched `google_sub` rejected; stored `token_hash` is SHA-256 of the cookie token, never the token; expired session → unauthorized; logout deletes the session                                                                                                                                                                                                                                                                                                                                                                                                                                       |
| `internal/service` — connection          | missing credentials → `ErrNotAllowed`; `Link` stores ciphertext that decrypts to the provider token; provider always receives the decrypted token; accounts upserted before transactions; removed transactions deleted; cursor saved only after all writes (fake records call order); `ErrReauthRequired` → `needs_reauth` and cursor unchanged; other errors → `last_error`, status unchanged; every outcome calls `ConnectionSynced` with the right trigger/outcome; `SyncAll` covers only the user's active connections and its summary counts match; another user's connection → `ErrNotFound`; reauth link token → `ErrConflict` unless `needs_reauth`; `CompleteReauth` → `active` + sync; `Remove` calls provider then deletes |
| `internal/service` — notification        | only `needs_reauth` connections produce notifications; `since` = `status_changed_at`; ordering and paging                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                             |
| `internal/service` — scheduler           | `RunOnce` syncs every active connection with `SyncTriggerScheduled`; one failure does not stop the rest; failures → `partial_failure` with the right `FailedConnections`; list error → `failure`; all good → `success`; nil schedule → `Run` returns immediately                                                                                                                                                                                                                                                                                                                                                                                                                                                                      |
| `internal/provider/plaid` (mapping only) | sign flip and decimal formatting without exponents; `ITEM_LOGIN_REQUIRED` → `ErrReauthRequired`; other Plaid errors wrapped with their code; unknown account type → `other`                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                           |
| `internal/handler`                       | 401 without session; sentinel error → 404/403/409; `pageSize=101` → 422; tenancy (user B gets 404 for user A's IDs); `NewPage` (`hasNext`, offset, non-nil `items`); readiness 200/503 with a fake `Pinger`                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                           |
| `internal/server`                        | SPA: existing asset served with immutable cache header; unknown path → `index.html`; `/api/unknown` → 404                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                             |
| `internal/metrics`                       | every enum's `String()` is unique and never `unknown` for declared constants; all label combinations pre-initialized to 0; `Prometheus{}.ConnectionSynced(scheduled, error)` increments exactly `fin_syncs_total{trigger="scheduled",outcome="error"}` (`testutil.ToFloat64` delta); `ScheduledRunFinished` sets the timestamp gauges, and the success timestamp only on `success`; status code → `StatusClass` boundaries; unknown `domain.ConnectionStatus` → `other`                                                                                                                                                                                                                                                               |

Not unit-tested: `internal/config` (by request), `repository/postgres` (raw SQL, exercised by the manual checks below), `identity/google.go` (thin go-oidc wrapper), wiring in `serve.go`, `cmd/seed` (flag parsing over two repository calls). No frontend tests at this time.

Example of the expected shape:

```go
func TestCipherRoundTrip(t *testing.T) {
	c, err := encryption.New([]byte("0123456789abcdefghijklmnopqrstuv"))
	if err != nil {
		t.Fatal(err)
	}

	sealed := c.Encrypt("access-sandbox-123")

	opened, err := c.Decrypt(sealed)
	if err != nil {
		t.Fatal(err)
	}

	if opened != "access-sandbox-123" {
		t.Fatalf("got %q", opened)
	}
}
```

Manual verification (complex, cross-system flows):

1. Google login inside the installed PWA on iOS and Android; a non-allow-listed email lands on `/login?error=not_allowed`.
2. Plaid sandbox link (`user_good` / `pass_good`) → accounts and transactions appear; `access_token_ciphertext` in the DB does not start with `access-sandbox-`.
3. Sandbox OAuth institution ("Platypus OAuth Bank") through `/connections/oauth` on a phone.
4. Re-auth: `just sandbox-reset-login <id>` → tap Refresh (or set `SYNC_CRON="* * * * *"` and wait a minute) → `GET /api/notifications` lists it, banner + nav badge appear, `fin_syncs_total{outcome="reauth_required"}` increments → Reconnect → banner clears and sync succeeds.
5. Install gate on every platform in §6.5.
6. `/readyz` flips to 503 with Postgres stopped and back to 200 after restart.
7. Docker image serves the PWA, API, `/swagger`.
8. Scheduler failure: with `SYNC_CRON="* * * * *"`, break the seeded credential (`UPDATE user_provider_credentials SET secret = 'invalid'`) → within a minute `fin_syncs_total{trigger="scheduled",outcome="error"}`, `fin_scheduled_sync_runs_total{outcome="partial_failure"}` and `fin_scheduled_sync_last_run_failed_connections` rise on `:8080/metrics`; re-running `just seed` with the real values restores the credential.

## 6. Frontend

### 6.1 Build setup

- `pnpm create vite@latest js --template react-ts`, then shadcn init (Tailwind v4, `@/` alias, `src/components/ui`).
- `vite.config.ts` uses `react()`, `tailwindcss()` and `VitePWA(...)`. `resolve.alias["@"] = src`. `server.proxy` for `/api`, `/version`, `/healthz`, `/readyz`, `/metrics`, `/auth`, `/openapi.json`, `/swagger`, `/mcp`, `/oauth`, `/.well-known` → `http://localhost:8080`.
- `tsconfig.app.json`: `strict`, `noUncheckedIndexedAccess`, `exactOptionalPropertyTypes`, `noImplicitOverride`, `noPropertyAccessFromIndexSignature`, `noUnusedLocals`, `noUnusedParameters`, `verbatimModuleSyntax`, `erasableSyntaxOnly` (no TS `enum`s), `paths: { "@/*": ["./src/*"] }`, `types: ["vite/client", "vite-plugin-pwa/client"]` (no triple-slash directive file needed).
- `package.json` scripts:

```json
{
  "dev": "vite",
  "gen": "openapi-typescript ../openapi.json -o src/lib/api/schema.gen.ts",
  "typecheck": "tsc -b",
  "lint": "oxlint --type-aware --deny-warnings src",
  "lint:fix": "oxlint --type-aware --fix src",
  "fmt": "oxfmt",
  "fmt:check": "oxfmt --check",
  "check": "pnpm typecheck && pnpm lint && pnpm fmt:check",
  "build": "pnpm gen && pnpm typecheck && vite build"
}
```

### 6.2 Routes

React Router uses an explicit route table in `src/router.ts`. The `src/app` directory mirrors URL paths; page entry points are named `name.Page.tsx`. Directories and files prefixed with `_` are private implementation details, not routes.

```
src/app/
├── home.Page.tsx
├── login.Page.tsx
├── _components/
├── connections/
│   ├── connections.Page.tsx
│   ├── oauth.Page.tsx
│   ├── _components/
│   └── _hooks/
└── transactions/
    ├── transactions.Page.tsx
    └── _components/
```

The root layout gates access on PWA installation (except `/consent`, which MCP clients open in a normal browser tab; §13.11). An authenticated layout uses React Query to resolve the session and redirects unauthenticated visitors to `/login`.

### 6.3 Data layer (`src/lib`)

- `lib/api/client.ts`: `createFetchClient<paths>({ credentials: "same-origin" })` + `export const $api = createClient(fetchClient)`. Imported **only** inside `src/lib` (lint-enforced).
- `lib/queries/*.ts`: the composite-hook layer components use. One file per resource: `me.ts` (`meQueryOptions`, `useMe`, `useLogout`), `summary.ts`, `connections.ts` (`useConnections`, `useLinkToken`, `useLinkConnection`, `useSyncConnection`, `useSyncAll`, `useRemoveConnection`), `notifications.ts` (`useNotifications`, `useReauthLinkToken`, `useCompleteReauth`), `accounts.ts`, `transactions.ts`, `card-art.ts` (`useCardArtSearch`, cached for 7 days because the catalog rate-limits; `useSelectCardArt`). Mutations own their invalidations (`queryClient.invalidateQueries({ queryKey: ["get", "/api/accounts"] })`, etc.), so React Query stays the single source of truth; components never hold server data in local state. Every sync/link/reauth mutation also invalidates `/api/notifications`.
- Infinite lists use `$api.useInfiniteQuery` with `pageParamName: "page"`, **explicit** `initialPageParam: 1` (openapi-react-query sends `0` otherwise), `getNextPageParam: (last) => (last.hasNext ? last.page + 1 : undefined)`, and `select` to flatten pages.
- `lib/query-client.ts`: `staleTime: 30_000`; no retries on 4xx; `QueryCache.onError` on 401 → `queryClient.clear()` + `router.navigate({ to: "/login" })`.
- `lib/types/api.ts`: `export type Transaction = components["schemas"]["Transaction"];` etc. — components never index the generated schema directly.
- Link token: `useLinkToken` is a query over `POST /api/connections/link-token` (`staleTime` 30 min) so the Connect button opens Plaid in one tap. The token is mirrored to `localStorage` by `lib/plaid/link-token-store.ts` purely so `connections/oauth.tsx` can resume Link after a bank OAuth redirect.
- Card art: credit account rows on the Connections page show **Wrong card?** (or **Pick card** when nothing matched), opening a dialog with the current art, a catalog search prefilled with the account's official name, and a **Use the default icon instead** action.

### 6.4 Component contract (requirement w)

The route component fetches and handles every non-success state; the view component receives guaranteed data.

```tsx
function TransactionsPage() {
  const { accountId } = Route.useSearch();
  const transactions = useTransactions(accountId);

  if (transactions.isPending) {
    return <LoadingState />;
  }

  if (transactions.isError) {
    return <ErrorState error={transactions.error} onRetry={transactions.refetch} />;
  }

  if (transactions.data.length === 0) {
    return <EmptyState title="No transactions yet" />;
  }

  return (
    <TransactionList
      transactions={transactions.data}
      hasMore={transactions.hasNextPage}
      loadingMore={transactions.isFetchingNextPage}
      onLoadMore={transactions.fetchNextPage}
    />
  );
}
```

Rules:

- No `useEffect`/`useLayoutEffect`/`useInsertionEffect` outside `src/lib/hooks/`. Browser subscriptions use `useSyncExternalStore` in `lib/hooks`.
- At most one `useState` per component; anything more becomes a composite hook (`_hooks/` if page-local, `lib/hooks/` if shared).
- shadcn components for everything; custom UI only for the install guide and list rows.

### 6.5 PWA and install gate

- `vite-plugin-pwa`: `registerType: "autoUpdate"`, manifest (`name`/`short_name` "fin", `display: "standalone"`, `start_url: "/"`, `scope: "/"`, theme/background colors, 192/512 + maskable icons generated once with `@vite-pwa/assets-generator` and committed under `public/`). Workbox `navigateFallback: "/index.html"` with `navigateFallbackDenylist: [/^\/api\//, /^\/auth\//, /^\/swagger/, /^\/openapi/, /^\/version$/, /^\/healthz$/, /^\/readyz$/, /^\/metrics$/]` — without the denylist the service worker would swallow the Google login navigation and serve the SPA for server-owned paths. API responses are never cached.
- Cross-origin images (card art, institution favicons) use a Workbox `CacheFirst` route named `remote-images` with a 7-day expiry and 200 entries, accepting opaque responses, so issuer image hosts are hit at most once a week per image.
- `lib/hooks/use-is-standalone.ts`: `useSyncExternalStore` over `matchMedia("(display-mode: standalone)")` plus iOS `navigator.standalone`.
- The root layout renders `<InstallGuide/>` instead of `<Outlet/>` when not standalone. Nothing else (including login) is reachable in a browser tab. Dev escape hatch: `VITE_SKIP_INSTALL_GATE=true` in `js/.env.local`.
- The gate also prevents a real bug: iOS home-screen apps have their own cookie jar, so logging in from Safari would not carry over.
- Install guide: horizontal CSS scroll-snap track (native swipe, no carousel lib, no effects), dots + Back/Next buttons driven by `use-step-scroller.ts` (`useRef` + `onScroll` → step index; buttons call `scrollTo`). Steps from `lib/pwa/platform.ts`. iOS Safari steps show light/dark screenshots from `src/assets/install/`, captured in the iOS 27 simulator with the tap target outlined; the step text also covers iOS 26.

| Platform            | Steps                                                                                                                                                                                                            |
| ------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| iOS Safari          | 1. Tap the **page menu** button next to the address bar · 2. Tap **Share** · 3. Tap **View More** · 4. Tap **Add to Home Screen** · 5. Leave **Open as Web App** on, tap **Add**, open fin from your Home Screen |
| iOS other browser   | 1. Open this page in **Safari** · then iOS Safari steps                                                                                                                                                          |
| Android Chrome      | 1. Tap **⋮** · 2. Tap **Install app** · 3. Open fin from your home screen                                                                                                                                        |
| Desktop Chrome/Edge | 1. Click the **install icon** in the address bar · 2. Open fin from your apps                                                                                                                                    |
| Desktop Safari      | 1. **File → Add to Dock** · 2. Open fin from the Dock                                                                                                                                                            |

### 6.6 oxlint + oxfmt (aggressive)

oxlint and oxfmt are the only JS lint/format tools (no ESLint, no Prettier). Install in M0:

```sh
pnpm add -D oxlint oxlint-tsgolint oxfmt eslint-plugin-react-hooks
```

| Tool   | Config                                        | Scope                             | Commands                                                 |
| ------ | --------------------------------------------- | --------------------------------- | -------------------------------------------------------- |
| oxlint | `js/.oxlintrc.json` + `js/lint/fin-plugin.ts` | `src/`                            | `pnpm lint`, `pnpm lint:fix`, `just lint-js`, `just fix` |
| oxfmt  | `js/.oxfmtrc.json`                            | all of `js/` except ignored paths | `pnpm fmt`, `pnpm fmt:check`, `just fmt`, `just lint-js` |

`.oxlintrc.json`:

```json
{
  "$schema": "./node_modules/oxlint/configuration_schema.json",
  "plugins": [
    "typescript",
    "react",
    "react-perf",
    "jsx-a11y",
    "import",
    "unicorn",
    "promise",
    "oxc"
  ],
  "jsPlugins": [
    "./lint/fin-plugin.ts",
    { "name": "react-hooks-js", "specifier": "eslint-plugin-react-hooks" }
  ],
  "categories": {
    "correctness": "error",
    "suspicious": "error",
    "pedantic": "error",
    "perf": "error",
    "style": "error"
  },
  "ignorePatterns": ["src/lib/api/schema.gen.ts"],
  "rules": {
    "fin/alias-imports": "error",
    "fin/no-comments": "error",
    "fin/route-private-imports": "error",
    "fin/no-effects-outside-hooks": "error",
    "fin/max-one-use-state": "error",
    "no-restricted-imports": [
      "error",
      {
        "paths": [
          { "name": "@/lib/api/client", "message": "Use a hook from @/lib/queries." },
          { "name": "openapi-react-query", "message": "Use a hook from @/lib/queries." },
          { "name": "openapi-fetch", "message": "Use a hook from @/lib/queries." },
          {
            "name": "@tanstack/react-query",
            "importNames": [
              "useQuery",
              "useMutation",
              "useInfiniteQuery",
              "useSuspenseQuery",
              "useQueryClient"
            ],
            "message": "Use a hook from @/lib/queries."
          }
        ]
      }
    ],
    "typescript/no-explicit-any": "error",
    "typescript/consistent-type-imports": "error",
    "typescript/no-floating-promises": "error",
    "typescript/no-misused-promises": "error",
    "typescript/strict-boolean-expressions": "error",
    "typescript/switch-exhaustiveness-check": "error",
    "typescript/no-unnecessary-condition": "error",
    "react/rules-of-hooks": "error",
    "react/exhaustive-deps": "error",
    "react-hooks-js/purity": "error",
    "react-hooks-js/refs": "error",
    "react-hooks-js/set-state-in-render": "error",
    "react-hooks-js/immutability": "error",
    "react-hooks-js/static-components": "error",
    "import/no-cycle": "error",
    "import/no-default-export": "error",
    "unicorn/filename-case": ["error", { "case": "kebabCase" }],
    "eqeqeq": "error",
    "no-console": "error"
  },
  "overrides": [{ "files": ["src/lib/**"], "rules": { "no-restricted-imports": "off" } }]
}
```

Tune noisy style/pedantic rules (e.g. `sort-keys`, `no-magic-numbers`, `id-length`, `func-style`, `react/jsx-props-no-spreading`, `no-ternary`) to `off` during M0 with the explicit list kept in the config — never via inline disables. Rule names for `eslint-plugin-react-hooks` are checked against the installed version.

`lint/fin-plugin.ts` custom rules:

| Rule                       | Behavior                                                                                                                                                |
| -------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `alias-imports`            | Report any `import`/`export … from`/dynamic `import()` whose source starts with `.` (covers `./` and `../`).                                            |
| `no-comments`              | Report every comment from `sourceCode.getAllComments()`. Generated files are in `ignorePatterns`.                                                       |
| `route-private-imports`    | Imports from a route's `_components/` or `_hooks/` are restricted to that route subtree; those directories belong under `src/app/`.                     |
| `no-effects-outside-hooks` | Report calls to `useEffect`/`useLayoutEffect`/`useInsertionEffect` (bare or `React.` member) in files not under `src/lib/hooks/`.                       |
| `max-one-use-state`        | Count `useState` calls per enclosing function; report the second one unless the file is under `src/lib/hooks/`, `src/lib/queries/`, or a `_hooks/` dir. |

Each rule gets a smoke check during M0: add a violating line, run `pnpm lint`, confirm failure, remove it.

`.oxfmtrc.json`:

```json
{
  "$schema": "./node_modules/oxfmt/configuration_schema.json",
  "printWidth": 100,
  "ignorePatterns": ["dist", "src/lib/api/schema.gen.ts"],
  "sortImports": {
    "internalPattern": ["@/"],
    "newlinesBetween": true
  },
  "sortTailwindcss": {
    "stylesheet": "./src/index.css",
    "functions": ["cn", "cva"]
  }
}
```

oxfmt preserves the single blank lines we write, so it never collapses the natural-newline layout. `fmt:check` is part of `pnpm check`, which `just lint-js` runs.

### 6.7 Re-auth indicator

- `useNotifications()` (`lib/queries/notifications.ts`): `$api.useQuery("get", "/api/notifications", …)` with `refetchOnWindowFocus: true` and `refetchInterval: 5 * 60_000`, so reopening the PWA re-checks immediately.
- `reauth-banner.tsx` (rendered by AppShell above every authed page): pending or error → renders nothing (non-blocking); data with items → shadcn destructive `Alert`: "**Chase** needs you to sign in again" + **Fix** link to `/connections`.
- Bottom-nav Connections tab shows a shadcn `Badge` with `total` when > 0.
- Connections page: a `needs_reauth` row shows a "Sign-in needed" badge, the date since, and a **Reconnect** button. `use-plaid-launcher.ts` opens Link in update mode with the token from `useReauthLinkToken(connectionId)`; `onSuccess` → `useCompleteReauth` → invalidations clear the banner.
- Push notifications are out of scope; the indicator appears whenever the app is opened or refocused.

## 7. Infrastructure

`infra/compose.yaml` — local Postgres only:

```yaml
services:
  postgres:
    image: postgres:19
    environment:
      POSTGRES_USER: fin
      POSTGRES_PASSWORD: fin
      POSTGRES_DB: fin
    ports:
      - "5433:5432"
    volumes:
      - pgdata:/var/lib/postgresql

volumes:
  pgdata:
```

`infra/Dockerfile` (build context = repo root; `just docker-build`): stages `go-base` → `spec` (`go run . openapi`) → `go-build` (`CGO_ENABLED=0`) → `web` (`node:${NODE_VERSION}`, standalone pnpm `${PNPM_VERSION}`, `pnpm install --frozen-lockfile`, `pnpm build` with the generated spec) → `gcr.io/distroless/static-debian13:nonroot` with `/app/fin` and `/app/static`, `STATIC_DIR=/app/static`, port 8080, `CMD ["serve"]`. The file in the repo is the source of truth.

`GO_VERSION`, `NODE_VERSION`, `PNPM_VERSION` come from `go.mod`, `js/package.json#devEngines.runtime.version` and `js/package.json#packageManager` via `just docker-build`. No migration files are in the image. Production migrations, run before deploying:

```
docker run --rm -v "$PWD/db:/migrations" migrate/migrate -path=/migrations -database "$DATABASE_URL" up
```

`.gitignore` additions: `/openapi.json`, `/bin/`, `/coverage.out`, `.env`, `js/node_modules/`, `js/dist/`, `js/src/lib/api/schema.gen.ts`, `js/.env.local`.
`.dockerignore`: `.git`, `js/node_modules`, `js/dist`, `bin`, `.env`, `openapi.json`.

## 8. Justfile

The `Justfile` in the repo root is the source of truth; `just --list` shows every recipe with its `[doc()]` description. Groups:

| Group    | Recipes                                                                                                                    |
| -------- | -------------------------------------------------------------------------------------------------------------------------- |
| Setup    | `install`, `db`, `db-stop`                                                                                                 |
| Database | `migrate`, `migrate-down n`, `migrate-version`, `migration name` (golang-migrate CLI), `seed email client_id secret [env]` |
| Dev      | `gen`, `dev-api`, `dev-web`, `dev` (db + migrate + gen + both servers)                                                     |
| Quality  | `lint`, `lint-go`, `lint-js`, `fix`, `fmt`, `test`, `cover`                                                                |
| Dev aids | `sandbox-reset-login connection_id`                                                                                        |
| Ship     | `build`, `docker-build`, `outdated`                                                                                        |

Prerequisites (README): Go, Node LTS, pnpm (standalone, not corepack), just, Docker, golangci-lint, golang-migrate, jq.

## 9. Milestones

Each milestone ends with `just lint && just test` green, and any business logic it introduces lands with its unit tests (§5.13) in the same milestone.

**M0 — Scaffold & tooling**

- `go mod init github.com/tahminator/fin`, `main.go` dispatch, `.golangci.yml`, `tools/nocomments`.
- `js/` via Vite react-ts, pnpm pinning, shadcn init, React Router, **oxlint + oxfmt** configs, `lint/fin-plugin.ts`.
- Justfile, `.env.example` (with the `SECRET_KEY` hint), `.gitignore`, `.dockerignore`, `infra/compose.yaml`.
- Accept: `just install`, `just lint` pass on the empty skeleton; each custom lint rule proven to fire once; `pnpm fmt:check` fails on a deliberately misformatted file.

**M1 — Config, encryption, DB, ops**

- `config.NewFromEnv` (no tests), `internal/encryption` + tests, `internal/metrics` (enums, recorder, instruments, DB collector) + tests, pool, migrations 000001–000006 applied with the migrate CLI, `cmd/seed` + `just seed`.
- Ops routes: `/healthz`, `/readyz` (Pinger) + tests, `/metrics` serving the pre-initialized series and DB snapshot.
- Huma API with `/openapi.json` + `/swagger`; `fin openapi`; `just gen` produces `schema.gen.ts`.
- Accept: empty env reports every missing variable in one error; short `SECRET_KEY` rejected; `just db && just migrate`; `/readyz` 200 → 503 with Postgres stopped, `/healthz` stays 200; `/swagger` renders.

**M2 — Auth + PWA shell**

- Google OIDC login/callback, sessions, `RequireSession`, `/api/me`, logout; `AuthService` tests.
- SPA: query client, `_authed` guard, login page, install gate + swipeable guide, AppShell.
- Accept: manual checks 1 and 5; cookies show `HttpOnly; Secure; SameSite=Strict`.

**M3 — Connections**

- Provider interface + Plaid impl (mapping tests, provider-call metrics), `ConnectionService` with token encryption and `SyncAll` (tests), connection endpoints (handler tests).
- UI: connections page (each row lists the connection's accounts via `useConnectionAccounts(connectionId)`), Plaid Link launcher, OAuth return page, per-connection Sync/Remove, header Refresh.
- Accept: manual checks 2 and 3 using the seeded sandbox user; Refresh syncs every connection and is idempotent (row counts unchanged on re-run); Remove deletes everything; tenancy covered by handler tests.

**M4 — Re-auth**

- Update-mode link token + complete endpoints, `NotificationService`, `GET /api/notifications`, cron `SyncScheduler` goroutine, `cmd/sandbox-reset-login`; tests for all three services.
- UI: banner, nav badge, Reconnect.
- Accept: manual checks 4 and 8.

**M5 — Accounts, transactions, summary**

- Endpoints + dashboard, transactions list with load-more and `accountId` filter; handler tests.
- Accept: pages advance until `hasNext=false`; `pageSize=101` → 422; amounts render with correct sign and currency.

**M6 — Production packaging**

- SPA serving from `STATIC_DIR` (tests), cache headers, Dockerfile.
- Accept: manual checks 6 and 7; `just docker-build`; migrate with the `migrate/migrate` image, then run the container; Lighthouse PWA installability passes; metrics show per-user connection/account/transaction counts and `needs_reauth`.

**M7 — OAuth authorization server** and **M8 — MCP tools**: §13.13.

## 10. Risks

- **iOS standalone + OAuth (Google and bank OAuth via Plaid)**: out-of-scope navigations open in an in-app browser sheet; confirm on a device that the callback returns into the installed app and that `localStorage` survives the Plaid OAuth round-trip.
- **Plaid production access**: each user needs their own approved Plaid app and must register `PUBLIC_URL/connections/oauth` as an allowed redirect URI.
- **oxlint JS plugins are alpha**: custom rules use only the ESLint-compatible AST/`context` API; if a rule misbehaves, it is a 20-line file to adjust.
- **`SECRET_KEY` loss or change**: no versioning, so every connection must be re-linked. Keep the key in a password manager.
- **Re-auth latency**: without webhooks, detection waits for the next sync (≤ one `SYNC_CRON` period); Plaid's webhook-only `PENDING_EXPIRATION` warnings are not surfaced.
- **`robfig/cron/v3` is feature-frozen**: only its parser is used, behind `config`'s `schedule` helper and `cron.Schedule`; swapping parsers touches two files.
- **Plaid secrets plaintext in DB**: accepted for two hand-inserted users; access tokens are encrypted.
- **MCP / public OAuth server**: see §13.14.

## 11. Requirement traceability

| Req                                                                  | Where                                                                                                     |
| -------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------- |
| oxlint + oxfmt                                                       | §6.1 scripts, §6.6, §8 `lint-js` / `fix` / `fmt`                                                          |
| `config.NewFromEnv()` handwritten, untested                          | §3, §5.13 exclusions                                                                                      |
| Backend business logic unit-tested, no frontend tests                | §5.13, §9                                                                                                 |
| Re-auth endpoint + indicator                                         | §5.5, §5.6 routes, §5.12, §6.7                                                                            |
| `SECRET_KEY` token encryption + `.env.example` hint                  | §3, §5.11                                                                                                 |
| Natural newlines everywhere                                          | §0, §5.10, every snippet                                                                                  |
| Manual refresh + cron goroutine (env-configurable)                   | §3 `SYNC_CRON`, §5.5 `SyncAll` + `SyncScheduler`, §5.6 `POST /api/connections/sync`, §6.2 Refresh button  |
| Sync failures visible in Prometheus                                  | §5.8 `fin_syncs_total`, `fin_scheduled_sync_*`, alerts                                                    |
| `iota` enum labels, cardinality locked                               | §5.8 labels table + max-series column                                                                     |
| Metrics only via high-level functions; zero-sized recorder for tests | §5.8 `Recorder`, `Prometheus{}`, `Noop{}`                                                                 |
| Gitignored `.env` for local creds; seed CLI for email + Plaid creds  | §1, §4 onboarding, §5.1 `cmd/seed`, §8 `seed`                                                             |
| a, e, m                                                              | §5.1 (Go, `net/http`, wiring in `serve.go`)                                                               |
| b, c, h (grouping)                                                   | §1 layout, §5.6 handler-per-resource `Register`, §5.10 file-size rule                                     |
| d                                                                    | §5.10 `tools/nocomments`, §6.6 `fin/no-comments`, Justfile `[doc()]`                                      |
| f                                                                    | §5.6 `server/routes.go`                                                                                   |
| g (huma spec), ah                                                    | §5.1 `fin openapi`, §5.6 huma config, gitignored `openapi.json`                                           |
| h (config)                                                           | §3                                                                                                        |
| g (DI)                                                               | §5.1, §5.2                                                                                                |
| i                                                                    | §4, §5.3 (pgx, raw formatted SQL, no transactions)                                                        |
| j                                                                    | §5.3 interfaces + embedded-interface fakes                                                                |
| k                                                                    | §5.4                                                                                                      |
| l                                                                    | §5.5, §5.7                                                                                                |
| n                                                                    | §5.8, ops routes in §5.6                                                                                  |
| o                                                                    | §4 manual onboarding                                                                                      |
| p                                                                    | §1, §4, §5.1 (migrate CLI)                                                                                |
| q, r                                                                 | §6.1, §6.2                                                                                                |
| s, aa5, t                                                            | §6.4 rules + §6.6 `no-effects-outside-hooks`, `max-one-use-state`                                         |
| u, v, z                                                              | §6.3 (`$api` only in `src/lib`, `lib/queries` hooks, lint-enforced)                                       |
| w                                                                    | §6.4                                                                                                      |
| x                                                                    | §6.1 `gen`, gitignored `schema.gen.ts`                                                                    |
| y                                                                    | §2, §7 (pnpm only, `devEngines`)                                                                          |
| aa                                                                   | §6.2 `_components/`/`_hooks/`, §6.6 `route-private-imports`                                               |
| aa2                                                                  | §2                                                                                                        |
| aa3                                                                  | §6.6                                                                                                      |
| aa4                                                                  | §6.6 `alias-imports`                                                                                      |
| ab                                                                   | §1, §6.3                                                                                                  |
| ab2, ab3                                                             | §6.4 (shadcn, KISS)                                                                                       |
| ab4                                                                  | §6.5                                                                                                      |
| ac                                                                   | §5.6 `Page[T]`, §6.3 infinite queries                                                                     |
| ad                                                                   | §0, §4 (`text` + Go enums)                                                                                |
| ae                                                                   | §8                                                                                                        |
| af                                                                   | §5.9                                                                                                      |
| ag                                                                   | §7                                                                                                        |
| Public MCP server at `/mcp`, same deployment, in OpenAPI             | §13.1, §13.3 (OpenAPI), §13.7                                                                             |
| Flexible querying and aggregation for agents                         | §13.6 (`search_transactions`, `aggregate_transactions`, `cash_flow`, `list_facets`, `find_recurring`)     |
| MCP cannot add bank connections; may refresh existing ones           | §13.1 Plaid isolation + scopes, §13.6 `refresh_data` and not-exposed list, §13.7 `depguard`, §13.12 tests |
| MCP OAuth                                                            | §13.4, §13.5, §13.9                                                                                       |

## 12. Phase 2 — CI/CD

### 12.1 Approach

Copy the reference repositories; do not invent mechanisms they do not use. When this section is silent, do what the reference does. Deliberate deviations are listed in §12.9 and nowhere else.

| Reference (local checkout)                                | What to take from it                                                                                                                                                                                             |
| --------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `~/github/Patina-Network/members-admin-dashboard/.github` | **Primary template.** Backend + frontend app with one image: `ci.yml`/`cd.yml` job graph, composite actions, `src/consts.ts`, `build-image`, `create-tag`, `promote-image`, `deploy`, `pretest`, `test` scripts. |
| `~/github/Patina-Network/members-grpc-service/.github`    | Migration testing (`src/test/run-migration-tests.ts` with `LocalPostgresClient`), production migration step inside the deploy job (`composite/migrate`).                                                         |
| `~/github/Patina-Network/codebloom/.github`               | Separate backend pre-test vs test, frontend pre-test (lint, typecheck, build; no tests).                                                                                                                         |
| `~/github/tahminator/abclang/.github`                     | Personal-account variant: hardcoded Docker Hub username, `k8s-personal` deploy target.                                                                                                                           |
| `~/github/tahminator/pipeline`                            | `DockerClient`, `GitHubClient`, `VersioningClient`, `LocalPostgresClient`, and the setup action. Use these APIs only: no Octokit, no `gh api`, no hand-rolled HTTP.                                              |

```mermaid
flowchart LR
  subgraph CI[ci.yaml]
    BT[backendTest] --> BI[buildImage]
    FT[frontendTest] --> BI
    TM[testMigrations] --> BI
    BI -->|main only| CT[createTag]
  end
  BI -->|PR| PRC[PR comment with staging tags]
  CT -->|tag push X.Y.Z| PI
  subgraph CD[cd.yaml]
    PI[promoteImage] --> DP[deployProduction: migrate, then GitOps PR]
  end
```

- **PR:** checks → build image tagged `staging-<short-sha>` and `sha-<full-sha>` → PR comment listing the tags. Nothing is deployed; there is no staging infrastructure.
- **main:** checks → build image tagged `<short-sha>` and `sha-<full-sha>` → `createTag` creates the next semver Git tag via `GitHubClient.createTag`, which pushes a tag event that starts CD.
- **CD:** promote the image of the tag's parent commit to `<version>` and `latest` → apply production migrations → open and merge the image-tag PR in `tahminator/k8s-personal`. Flux reconciles; CI never touches the cluster.

### 12.2 Layout

```text
.github/
├── workflows/
│   ├── ci.yaml
│   └── cd.yaml
├── composite/
│   ├── pretest/action.yaml
│   ├── frontend-pretest/action.yaml
│   ├── test/action.yaml
│   ├── build-image/action.yaml
│   ├── create-tag/action.yaml
│   ├── promote-image/action.yaml
│   ├── migrate/action.yaml
│   └── deploy/action.yaml
└── scripts/
    ├── .gitignore
    ├── package.json
    ├── bun.lock
    ├── tsconfig.json
    ├── oxlint.config.ts
    ├── oxfmt.config.ts
    └── src/
        ├── lib/
        │   ├── consts.ts
        │   └── env.ts
        ├── pretest/
        │   ├── backend/index.ts
        │   └── frontend/index.ts
        ├── test/
        │   ├── backend/index.ts
        │   └── migrations/index.ts
        ├── build-image/index.ts
        ├── create-tag/index.ts
        ├── promote-image/index.ts
        └── deploy/index.ts
```

Every callable script is an `index.ts` under at least one grouping directory and is invoked by directory, as the references do: `bun run .github/scripts/src/<dir>`. Non-callable modules live only in `src/lib/`.

### 12.3 Script conventions

Copied from the references:

- Scripts with arguments parse them at module level, before `main`, with static imports:
  ```ts
  const { releaseTag } = await yargs(hideBin(process.argv))
    .option("releaseTag", { type: "string", describe: "…", demandOption: true })
    .strict()
    .parse();
  ```
  Scripts without arguments do not import yargs.
- All work happens in `async function main()`. Each file ends with:
  ```ts
  main()
    .then(() => {
      process.exit(0);
    })
    .catch((e) => {
      console.error(e);
      process.exit(1);
    });
  ```
- `main` starts with `parseCiEnv([...])` from `src/lib/env.ts` for every environment variable it reads (the helper returns a typed record and throws listing all missing names). It replaces the references' inline `requiredEnv` calls.
- `src/lib/consts.ts`: `GITHUB_OWNER = "tahminator"`, `GITHUB_REPOSITORY = "fin"`, `K8S_MANIFESTS_REPOSITORY = "k8s-personal"`, `DOCKER_HUB_USERNAME = "tahminator"`, `DOCKER_REPOSITORY = "fin"`, `KUSTOMIZATION_FILE_PATH = "base/production/fin/kustomization.yaml"`, and `shortSha(sha)` exactly as in members-admin-dashboard (throws unless 40 lowercase hex characters; returns the first 7).
- `package.json`: scripts `typecheck`, `test`, `fix`, `lint`, `lint:fix`, `fmt`, `fmt:fix` as in the references. Dependencies are `@tahminator/pipeline` (pinned `1.0.105`) and `yargs` only. Dev dependencies are `@types/bun`, `@types/yargs`, `oxfmt`, `oxlint` and `typescript`. Keep fin's existing `tsconfig.json`, `oxlint.config.ts` and `oxfmt.config.ts`, rather than the Patina-only `@Patina-Network/cicd-config`.
- Repository style applies: no comments anywhere in `.github` (this also drops the references' `# vX` pin comments and `yaml-language-server` line), natural blank lines between logical blocks, and no extra validation beyond what the references do.

### 12.4 Scripts

| Script             | Args                                                                                                                  | Env                                                                                              | Behavior                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                            |
| ------------------ | --------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------ | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `pretest/backend`  | —                                                                                                                     | —                                                                                                | `just lint-go` (golangci-lint, no-comments check), then `golangci-lint fmt --diff ./...`.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                           |
| `pretest/frontend` | —                                                                                                                     | —                                                                                                | `pnpm --dir js install --frozen-lockfile`, `just gen`, `pnpm --dir js check`, `pnpm --dir js build`. No frontend tests.                                                                                                                                                                                                                                                                                                                                                                                                                                                             |
| `test/backend`     | —                                                                                                                     | —                                                                                                | `just test`.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                        |
| `test/migrations`  | —                                                                                                                     | —                                                                                                | Copy grpc `run-migration-tests.ts`: `await using pgClient = await LocalPostgresClient.create({ database: "fin" })`, build `DATABASE_URL` from `pgClient.state` (`sslmode=disable`), then via `$.env({...process.env, DATABASE_URL})` run `just migrate && just migrate && just migrate-down 1 && just migrate && just migrate-down -all && just migrate`.                                                                                                                                                                                                                           |
| `build-image`      | `--environment staging\|production` (choices, `satisfies Environment[]`), `--sha` (required), `--prId` (default `""`) | `DOCKER_HUB_PAT`, `_GITHUB_APP_APP_ID`, `_GITHUB_APP_INSTALLATION_ID`, `_GITHUB_APP_PEM_CONTENT` | Copy members-admin-dashboard `build-image` with a single `linux/amd64` platform. Tags: staging `[staging-<short>, sha-<sha>]`, production `[<short>, sha-<sha>]`. `dockerFileLocation: "infra/Dockerfile"`, `dockerRepository: DOCKER_REPOSITORY`. Pass `buildArgs` `GO_VERSION` (from the `go` directive in `go.mod`), `NODE_VERSION` (`js/package.json` `devEngines.runtime.version`) and `PNPM_VERSION` (`packageManager` without the `pnpm@` prefix and `+hash` suffix), matching `just docker-build`. If `prId` is set, send the reference PR comment listing the pushed tags. |
| `create-tag`       | —                                                                                                                     | GitHub App vars                                                                                  | Verbatim members-admin-dashboard `create-tag`: `VersioningClient(ghClient, VersionUpdatingStrategy.NONE)`, `ghClient.createTag({ nextTag: await versioningClient.next(), onPreTagCreate })`. Versions are plain `X.Y.Z`; the first is `1.0.0`.                                                                                                                                                                                                                                                                                                                                      |
| `promote-image`    | `--releaseTag`, `--sha` (both required)                                                                               | `DOCKER_HUB_PAT`                                                                                 | `promoteDockerImage({ originalTag: shortSha(sha), newGithubTags: [releaseTag, "latest"], repository: DOCKER_REPOSITORY })`.                                                                                                                                                                                                                                                                                                                                                                                                                                                         |
| `deploy`           | `--releaseTag` (required)                                                                                             | GitHub App vars                                                                                  | `updateK8sTagWithPR({ manifestRepo: [GITHUB_OWNER, K8S_MANIFESTS_REPOSITORY], originRepo: [GITHUB_OWNER, GITHUB_REPOSITORY], kustomizationFilePath: KUSTOMIZATION_FILE_PATH, imageName: \`${DOCKER_HUB_USERNAME}/${DOCKER_REPOSITORY}\`, newTag: releaseTag, environment: "production" })`. Production only, so there is no `--sha`or`--environment`.                                                                                                                                                                                                                               |

Production migrations in CD have no script. As in grpc, the `migrate` composite runs the Justfile recipe: `just migrate` (which already reads `DATABASE_URL`). The `/try` slash command (§12.11) runs the same recipe from its script.

### 12.5 Composite actions

Same shape as members-admin-dashboard: `name`, `description`, UPPER_SNAKE `inputs` with descriptions, `runs.using: composite`, and one named step with `shell: bash` and `run: bun run .github/scripts/src/<dir> --flag "${{ inputs.X }}"`.

| Composite          | Inputs                                       | Runs                                                              |
| ------------------ | -------------------------------------------- | ----------------------------------------------------------------- |
| `pretest`          | —                                            | `src/pretest/backend`                                             |
| `frontend-pretest` | —                                            | `src/pretest/frontend`                                            |
| `test`             | —                                            | `src/test/backend`                                                |
| `build-image`      | `ENVIRONMENT`, `SHA`, `PR_ID` (default `""`) | `src/build-image`                                                 |
| `create-tag`       | —                                            | `src/create-tag`                                                  |
| `promote-image`    | `RELEASE_TAG`, `SHA`                         | `src/promote-image`                                               |
| `migrate`          | —                                            | `just migrate`; caller supplies `DATABASE_URL` through step `env` |
| `deploy`           | `RELEASE_TAG`                                | `src/deploy`                                                      |

As in grpc, the workflow step runs `test/migrations` directly; it has no composite.

### 12.6 `ci.yaml`

Header as in members-admin-dashboard: `name: CI`, `run-name: Running continuous integration on ${{ github.actor }}'s commits`, top-level `permissions: contents: read`, `on: push: branches: [main]` and `pull_request`, `concurrency: group: ci-${{ github.ref }}`, `cancel-in-progress: ${{ github.event_name == 'pull_request' }}`.

Every job: camelCase id with a `name`, `runs-on: ubuntu-26.04`, step `Checkout Repository` using `actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1`, then step `Setup Pipeline` calling `tahminator/pipeline/actions/setup@3fe33a6496cc37947213e96e6f03445afa2e2a9c` (1.0.105) directly with `SECRETS: ${{ toJSON(secrets) }}` and `LINT_CI: true`, plus the job's install inputs. There is no local setup wrapper.

| Job                                                                                                  | Needs / condition                                                                   | Extra setup inputs                                                                                                                                                   | Steps                                                                                                                                                                                                                                                                           |
| ---------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `backendTest` "Backend tests"                                                                        | —                                                                                   | `INSTALL_GO: true`, `GO_VERSION_FILE: go.mod`, `INSTALL_JUST: true`, `INSTALL_GOLANGCI_LINT: true`                                                                   | `pretest`, then `test` composites                                                                                                                                                                                                                                               |
| `frontendTest` "Frontend tests"                                                                      | —                                                                                   | `INSTALL_GO: true`, `GO_VERSION_FILE: go.mod`, `INSTALL_JUST: true`, `INSTALL_NPM: true`, `NODE_VERSION: "26.13.1"`, `INSTALL_PNPM: true`, `PNPM_VERSION: "12.21.0"` | `frontend-pretest` composite                                                                                                                                                                                                                                                    |
| `testMigrations` "Test database migrations"                                                          | `timeout-minutes: 10`                                                               | `INSTALL_JUST: true`, `INSTALL_MIGRATE: "true"`                                                                                                                      | `run: bun run .github/scripts/src/test/migrations`                                                                                                                                                                                                                              |
| `buildImage` "Build ${{ github.event_name == 'pull_request' && 'staging' \|\| 'production' }} image" | needs all three checks; no `if` (PR and main)                                       | —                                                                                                                                                                    | Checkout with `ref: ${{ github.event.pull_request.head.sha \|\| github.sha }}`; `build-image` composite with `ENVIRONMENT` (staging on PR, else production), `SHA: ${{ github.event.pull_request.head.sha \|\| github.sha }}`, `PR_ID: ${{ github.event.pull_request.number }}` |
| `createTag` "Create new tag"                                                                         | needs `buildImage`; `if: github.ref_name == 'main'`; `permissions: contents: write` | —                                                                                                                                                                    | `create-tag` composite                                                                                                                                                                                                                                                          |

### 12.7 `cd.yaml`

Header as in members-admin-dashboard: `name: CD`, `run-name: Running continuous deployment on ${{ github.actor }}'s commits`, `permissions: contents: read`, `on: push: tags: ["*"]`, `concurrency: group: cd`, `cancel-in-progress: false`.

| Job                                                      | Details                                                                                                                                                                                                                                                                                                                                                                                                       |
| -------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `promoteImage` "Promote image to ${{ github.ref_name }}" | Checkout with `fetch-depth: 2`; setup; step `Get parent SHA` (`id: parent`) runs `echo "sha=$(git rev-parse ${{ github.sha }}^1)" >> "$GITHUB_OUTPUT"`; `promote-image` composite with `RELEASE_TAG: ${{ github.ref_name }}`, `SHA: ${{ steps.parent.outputs.sha }}`.                                                                                                                                         |
| `deployProduction` "Deploy to production"                | `needs: promoteImage`; job concurrency `deploy-production`, `cancel-in-progress: false`; `timeout-minutes: 20`; setup adds `INSTALL_JUST: true`, `INSTALL_MIGRATE: "true"`; step `Migrate production database` uses the `migrate` composite with `env: DATABASE_URL: ${{ secrets.PRODUCTION_MIGRATION_DATABASE_URL }}`; step `Deploy` uses the `deploy` composite with `RELEASE_TAG: ${{ github.ref_name }}`. |

`GitHubClient.createTag` creates a release commit whose parent is the `main` head at tag time. With `VersionUpdatingStrategy.NONE`, its tree equals the parent's tree. That is why CD promotes the parent SHA's image, and why migrations from the tag checkout match that image. The first deploy replaces the `newTag: changeme` placeholder in `k8s-personal`.

### 12.8 Secrets and prerequisites

Repository-level Actions secrets only; no GitHub environments. `tahminator/omnis` fans out its shared secrets to every personal repository. `gh secret list --repo tahminator/fin` returned `[]` during planning, so confirm the names once IaC has been applied.

| Secret                                                                                  | Use                                                                          |
| --------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------- |
| `DOCKER_HUB_PAT` (shared)                                                               | `DockerClient` as `tahminator`: build, push, promote.                        |
| `_GITHUB_APP_APP_ID`, `_GITHUB_APP_INSTALLATION_ID`, `_GITHUB_APP_PEM_CONTENT` (shared) | `GitHubClient.createWithGithubAppToken`: PR comment, release tag, GitOps PR. |
| `PRODUCTION_MIGRATION_DATABASE_URL` (fin-specific)                                      | `DATABASE_URL` for `just migrate` in CD and `/try`; TLS, migration role.     |

`SONAR_TOKEN` is also shared, but unused because fin has no Sonar project.

Pipeline 1.0.105 provides the two capabilities fin needed beyond the references: the setup input `INSTALL_GOLANGCI_LINT` (used by `backendTest`), and `LocalPostgresClient.create({ database, version })`. `test/migrations` passes `version: 19` because fin's migrations use `uuidv7()`, which needs Postgres 18 or newer.

Other prerequisites:

- Install the GitHub App on `tahminator/fin` (Contents: write, Pull requests: write) and `tahminator/k8s-personal` (Contents: write, Pull requests: write).
- Make GitHub-hosted runners able to reach the production database.
- Keep the Docker Hub image publicly readable, or provision a pull secret.
- Production nodes must support linux/amd64.
- In `k8s-personal/base/production/fin/secrets.yaml`, fill `DATABASE_URL`, `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET` and `SECRET_KEY`, then SOPS-encrypt the file. Keep the 32-byte `SECRET_KEY` stable across deployments, or stored Plaid access tokens become unreadable.
- Register `https://fin.tahmid.io/auth/google/callback` with Google and `https://fin.tahmid.io/connections/oauth` with Plaid; configure DNS. The server never migrates on startup.

### 12.9 Deviations from the references

- One `linux/amd64` image and Docker repository: no arch matrix and no `-arm` repository.
- No Sonar or coverage upload and no staging deploy jobs. The `/try` and `/revert` slash commands (§12.11) are fin-specific.
- No same-repository `if` guard on `buildImage`; it runs on every PR.
- Docker Hub username is a constant (`abclang` style); there is no `DOCKER_HUB_USERNAME` secret.
- `build-image` passes `GO_VERSION`, `NODE_VERSION` and `PNPM_VERSION` build args, which `infra/Dockerfile` requires.
- Production migrations use one `DATABASE_URL` secret and fin's existing `just migrate` recipe, not grpc's `DATABASE_*` parts and `migrate-prod` recipe.
- `parseCiEnv` in `src/lib/env.ts` replaces inline `requiredEnv`; constants live in `src/lib/consts.ts` instead of `src/consts.ts`; scripts have a second grouping level (`pretest/backend`, `test/migrations`, …).
- Repository no-comments rule: no pin version comments.

### 12.10 Acceptance

1. `bun run --cwd .github/scripts test` passes (typecheck, oxfmt, oxlint), which is what `LINT_CI` runs.
2. `actionlint .github/workflows/ci.yaml .github/workflows/cd.yaml` passes.
3. Locally, `pretest/backend`, `pretest/frontend` and `test/backend` succeed.
4. `test/migrations` passes against `LocalPostgresClient` Postgres 19.
5. Scripts reject unknown arguments (strict yargs) and report missing environment variables.
6. `just docker-build` still builds the image with the same build arguments `build-image` passes.
7. Remote delivery is proven only by real runs once secrets exist:
   - a PR run pushes `staging-<short>` and `sha-<sha>` and comments on the PR;
   - a main run pushes `<short>` and `sha-<sha>` and creates the next tag;
   - CD promotes that image, migrates and merges the `k8s-personal` PR.

   Flux reconciliation and app health are checked separately.

### 12.11 PR slash commands

`slash.yaml` uses `tahminator/pipeline/actions/commands`, which checks that the commenter has write access and dispatches `try-command.yml` or `revert-command.yml` on the PR head ref. Both share the `deploy-production` concurrency group with CD and report through a PR comment and a status check on the PR head.

- `/try` (`src/try`): waits for the PR's `staging-<short>` image, reads the production tag from `k8s-personal`, then applies pending migrations from the PR branch to production with `just migrate` (up only, never down), then deploys the staging image with the `deploy` mechanism. The workflow installs just and golang-migrate in setup (`INSTALL_JUST: true`, `INSTALL_MIGRATE: "true"`) and passes `DATABASE_URL: ${{ secrets.PRODUCTION_MIGRATION_DATABASE_URL }}` to the script step. Migrations run only once the image exists, so a missing image never migrates production. The script reads `migrate version` before and after and reports, for example, `Migrated production from version 8 to version 13` or `No pending migrations (version 8)`. A dirty database or a failed migration fails `/try` without deploying; the PR comment and status check point to the run logs and never include raw migrate output.
- `/revert` (`src/revert`): restores the image the PR's `/try` replaced, as long as production still runs the tried image. It does not roll back migrations, so the restored image runs against the migrated schema.

## 13. Phase 3 — MCP server

A public [Model Context Protocol](https://modelcontextprotocol.io) server at `PUBLIC_URL/mcp`, served by the same `fin serve` process and deployment. Any MCP client (Claude, Claude Code, Cursor, the MCP Inspector, custom agents) connects with OAuth on behalf of an allow-listed user and can query, filter and aggregate that user's connections, accounts, balances and transactions. With the user's permission it can also **refresh** the user's already-linked connections when the data is stale (§13.6 `refresh_data`). Nothing reachable through MCP can create, re-authenticate or remove a bank connection: linking and fixing banks through Plaid Link stays a human-only, in-app flow.

### 13.1 Decisions

| Topic                | Decision                                                                                                                                                                                                                                                                                                                                                                                              | Why                                                                                                                                                                     |
| -------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Spec                 | Target the newest MCP revision the official Go SDK supports at scaffold time. Written against the **2025-11-25** authorization rules: protected resource metadata (RFC 9728), resource indicators (RFC 8707), PKCE S256, Client ID Metadata Documents preferred, DCR as fallback                                                                                                                      | These are the rules every current client implements                                                                                                                     |
| SDK                  | `github.com/modelcontextprotocol/go-sdk` (`mcp`, `auth`, `oauthex`), `@latest`                                                                                                                                                                                                                                                                                                                        | Official SDK; typed tool handlers infer JSON Schema from Go structs                                                                                                     |
| Transport            | Streamable HTTP, **stateless** (`StreamableHTTPOptions{Stateless: true, JSONResponse: true}`), one endpoint `/mcp`; `GET` and `DELETE` → 405                                                                                                                                                                                                                                                          | Tools need no server push or session; any replica can answer any request                                                                                                |
| Deployment           | Same binary, same listener, same image. `/mcp`, `/oauth/*` and `/.well-known/oauth-*` are plain `net/http` routes in `server/routes.go`, **documented in the OpenAPI spec** like every other route (§13.3)                                                                                                                                                                                            | Requested; no second service to deploy, configure or monitor, and `/swagger` stays the complete picture of the HTTP surface                                             |
| Authorization server | fin **is** the authorization server, with Google login as the upstream identity (the existing `AuthService`). Hand-written, no OAuth framework                                                                                                                                                                                                                                                        | Google tokens cannot carry an audience of `PUBLIC_URL/mcp`, and the MCP spec forbids passing upstream tokens through. One client type (public, PKCE) keeps the AS small |
| Tokens               | **Opaque** random tokens (`fin_at_…`, `fin_rt_…`, 32 bytes `crypto/rand`, base64url), SHA-256 stored, same pattern as `sessions.token_hash`. No JWTs                                                                                                                                                                                                                                                  | AS and RS share a database, so lookups are free, revocation is instant and there is no signing key to manage or rotate                                                  |
| Scopes               | Two scopes: `fin:read` (every query tool) and `fin:sync` (only `refresh_data`). Clients request both by default; the consent page lets the user untick refreshing. No other write scopes exist                                                                                                                                                                                                        | Refreshing is the only action agents need; a separate scope lets a user grant pure read access                                                                          |
| Plaid isolation      | `internal/mcpserver` gets read-only repository interfaces plus one narrow `connectionSyncer` interface (`Sync(ctx, userID, connectionID, trigger) error`, the same shape `SyncScheduler` already uses), declared inside `mcpserver` and satisfied by `ConnectionService`. `depguard` forbids `mcpserver` from importing `internal/provider`, `internal/encryption`, `internal/service` and `plaid-go` | Link, exchange, re-auth and remove are not in any interface `mcpserver` holds, so "cannot add a bank" is enforced by the compiler and lint, not by a missing `if`       |
| No raw SQL tool      | Agents get a typed aggregation tool instead (§13.6)                                                                                                                                                                                                                                                                                                                                                   | A SQL tool would bypass `WHERE user_id = $n` tenancy and expose `access_token_ciphertext`, cursors and Plaid secrets                                                    |
| Consent UI           | A SPA page (`/consent`) behind the existing session, not a server-rendered page                                                                                                                                                                                                                                                                                                                       | `fin_session` is `SameSite=Strict`, so it is not sent on the cross-site navigation into `/oauth/authorize`; a same-origin `fetch` from the SPA does carry it            |

### 13.2 Layout additions

```
internal/
├── domain/oauth.go               OAuthClient, Authorization, Grant, Token; ClientSource, TokenKind, Scope enums
├── oauth/                        pure protocol helpers, no I/O except the CIMD fetcher
│   ├── pkce.go                   S256 verification (constant-time compare)
│   ├── redirect.go               redirect URI validation + exact matching
│   ├── scope.go                  parse/format, subset checks
│   ├── metadata_document.go      Client ID Metadata Document fetcher (SSRF-guarded)
│   └── errors.go                 RFC 6749 error codes as a Go enum + JSON writer
├── repository/
│   ├── oauth_client.go
│   ├── oauth_authorization.go
│   ├── oauth_grant.go
│   ├── oauth_token.go
│   └── transaction_query.go      Search + Aggregate + Facets for MCP (typed specs, no SQL in callers)
├── service/oauth.go              OAuthService: authorize, consent, exchange, refresh, revoke, verify
├── handler/
│   ├── oauth.go                  net/http: metadata docs, authorize, token, register, revoke
│   ├── oauth_openapi.go          Document(api): documentation-only OpenAPI operations for the routes above
│   └── oauth_consent.go          huma (/api): pending request, approve, deny, grants list/revoke
└── mcpserver/
    ├── server.go                 New(deps) → *mcp.Server; registers every tool
    ├── auth.go                   bearer middleware (auth.RequireBearerToken + our TokenVerifier), Origin check
    ├── types.go                  tool input/output DTOs (json + jsonschema tags)
    ├── tools_accounts.go         list_connections, list_accounts, get_balances
    ├── tools_transactions.go     search_transactions, get_transaction, list_facets
    ├── tools_aggregate.go        aggregate_transactions, cash_flow
    ├── tools_recurring.go        find_recurring
    ├── tools_refresh.go          refresh_data
    └── openapi.go                Document(api): documentation-only OpenAPI operation for /mcp + x-mcp-tools
js/src/app/
├── consent.Page.tsx              approve/deny an MCP client
└── apps/apps.Page.tsx            connected apps: list + revoke grants
```

### 13.3 Endpoints

| Method | Path                                        | Handler                          | Auth            | Notes                                                                                               |
| ------ | ------------------------------------------- | -------------------------------- | --------------- | --------------------------------------------------------------------------------------------------- |
| POST   | `/mcp`                                      | `mcpserver` streamable handler   | Bearer          | JSON-RPC; `GET`/`DELETE` → 405                                                                      |
| GET    | `/.well-known/oauth-protected-resource/mcp` | `OAuthHandler.resourceMetadata`  | public, CORS \* | RFC 9728; also served at `/.well-known/oauth-protected-resource`                                    |
| GET    | `/.well-known/oauth-authorization-server`   | `OAuthHandler.serverMetadata`    | public, CORS \* | RFC 8414; also at `/.well-known/openid-configuration` for clients that only try OIDC discovery      |
| GET    | `/oauth/authorize`                          | `OAuthHandler.authorize`         | public          | validates, stores a pending authorization, 303 → `/consent?request=<id>`                            |
| POST   | `/oauth/token`                              | `OAuthHandler.token`             | client, CORS \* | `application/x-www-form-urlencoded`; `authorization_code`, `refresh_token`                          |
| POST   | `/oauth/register`                           | `OAuthHandler.register`          | public, CORS \* | RFC 7591 dynamic registration (fallback for clients without CIMD), rate-limited                     |
| POST   | `/oauth/revoke`                             | `OAuthHandler.revoke`            | client, CORS \* | RFC 7009; always 200                                                                                |
| GET    | `/api/oauth/requests/{requestId}`           | `OAuthConsentHandler.get`        | session         | client name, client id host, verified flag, redirect host, scopes, expiry; 404 when expired or used |
| POST   | `/api/oauth/requests/{requestId}/approve`   | `OAuthConsentHandler.approve`    | session         | → `{redirectTo}` with `code`, `state`, `iss`                                                        |
| POST   | `/api/oauth/requests/{requestId}/deny`      | `OAuthConsentHandler.deny`       | session         | → `{redirectTo}` with `error=access_denied`, `state`, `iss`                                         |
| GET    | `/api/oauth/grants`                         | `OAuthConsentHandler.listGrants` | session         | `Page[OAuthGrant]`: client name, scopes, created, last used                                         |
| DELETE | `/api/oauth/grants/{grantId}`               | `OAuthConsentHandler.revoke`     | session         | 204; revokes the grant and every token issued under it                                              |

`routes.go` additions (the one routing table stays the one routing table):

```go
mux.Handle("POST /mcp", h.MCP)
mux.Handle("/mcp", http.HandlerFunc(methodNotAllowed))

mux.HandleFunc("GET /.well-known/oauth-protected-resource/mcp", h.OAuth.ResourceMetadata)
mux.HandleFunc("GET /.well-known/oauth-protected-resource", h.OAuth.ResourceMetadata)
mux.HandleFunc("GET /.well-known/oauth-authorization-server", h.OAuth.ServerMetadata)
mux.HandleFunc("GET /.well-known/openid-configuration", h.OAuth.ServerMetadata)

mux.HandleFunc("GET /oauth/authorize", h.OAuth.Authorize)
mux.HandleFunc("POST /oauth/token", h.OAuth.Token)
mux.HandleFunc("POST /oauth/register", h.OAuth.Register)
mux.HandleFunc("POST /oauth/revoke", h.OAuth.Revoke)
mux.Handle("/oauth/", http.NotFoundHandler())
mux.Handle("/.well-known/", http.NotFoundHandler())

h.OAuthConsent.Register(authed)

h.OAuth.Document(api)
mcpserver.Document(api)
```

`/mcp`, `/oauth/` and `/.well-known/` never fall through to the SPA (`spa_test.go` gains those cases). The OAuth endpoints are served by `net/http`, not huma: their bodies are form-encoded and their errors must be RFC 6749 JSON (`{"error":"invalid_grant","error_description":"…"}`), which huma's problem+json does not produce. `/mcp` is served by the SDK's handler for the same reason. CORS `*` without credentials is set only on the metadata, token, register and revoke endpoints so browser-based MCP clients can complete discovery; it is safe because none of them read cookies.

**OpenAPI.** Serving these routes with `net/http` does not hide them from the spec. `RegisterRoutes` calls `h.OAuth.Document(api)` and `mcpserver.Document(api)`, which add **documentation-only** operations with `api.OpenAPI().AddOperation(…)` (huma records them in the spec but does not route them, so the mux above stays in charge). Both functions read types only, so `fin openapi` documents them with zero-value dependencies, the same as every huma handler (§5.1).

- Security scheme `mcpOAuth`: `type: oauth2`, `authorizationCode` flow with `authorizationUrl` `/oauth/authorize`, `tokenUrl` and `refreshUrl` `/oauth/token`, scopes `fin:read` and `fin:sync`. `cookieAuth` stays the scheme for `/api`.
- `POST /mcp` (tag `mcp`, security `mcpOAuth: [fin:read]`): JSON-RPC 2.0 request and response schemas, `401`/`403` responses documenting the `WWW-Authenticate` header, `429`. `GET /mcp` and `DELETE /mcp` are listed as `405`. The operation carries an `x-mcp-tools` extension: for each tool its name, description, required scope, annotations and input/output JSON Schema, generated from the **same** Go types the SDK registers, so the spec cannot drift from `tools/list` (a test compares them).
- `/oauth/*` (tag `oauth`): form-encoded request bodies (`application/x-www-form-urlencoded`) for token and revoke, JSON for register, the RFC 6749 error schema, `302`/`303` responses for authorize. `/.well-known/*` (tag `oauth`): the two metadata document schemas.
- The SPA's generated `schema.gen.ts` gains these paths; nothing in `src/lib/queries` calls them.

### 13.4 OAuth flow

```mermaid
sequenceDiagram
  participant C as MCP client
  participant F as fin (/mcp + AS)
  participant B as User's browser
  participant G as Google

  C->>F: POST /mcp (no token)
  F-->>C: 401 WWW-Authenticate: Bearer resource_metadata=".../.well-known/oauth-protected-resource/mcp", scope="fin:read fin:sync"
  C->>F: GET protected resource metadata → authorization_servers=[PUBLIC_URL]
  C->>F: GET /.well-known/oauth-authorization-server
  Note over C,F: client_id = HTTPS URL of its metadata document (CIMD),<br/>or POST /oauth/register if it has none
  C->>B: open /oauth/authorize?client_id&redirect_uri&code_challenge(S256)&state&scope&resource=PUBLIC_URL/mcp
  B->>F: GET /oauth/authorize
  F-->>B: 303 /consent?request=<id>
  B->>F: SPA: GET /api/oauth/requests/<id> (no session → /login?returnTo=/consent?request=<id>)
  B->>G: Google login (existing flow), callback 303 → returnTo
  B->>F: POST /api/oauth/requests/<id>/approve
  F-->>B: {redirectTo: redirect_uri?code&state&iss}
  B->>C: redirect with code
  C->>F: POST /oauth/token (code, code_verifier, redirect_uri, client_id, resource)
  F-->>C: access_token (1h) + refresh_token (rotating)
  C->>F: POST /mcp Authorization: Bearer fin_at_…
```

**Discovery.** Unauthenticated `/mcp` requests get `401` with `WWW-Authenticate: Bearer resource_metadata="<PUBLIC_URL>/.well-known/oauth-protected-resource/mcp", scope="fin:read fin:sync"`. Protected resource metadata: `{resource: "<PUBLIC_URL>/mcp", authorization_servers: ["<PUBLIC_URL>"], scopes_supported: ["fin:read", "fin:sync"], bearer_methods_supported: ["header"], resource_name: "fin"}`. Authorization server metadata: `issuer = PUBLIC_URL`, the four endpoints, `response_types_supported: ["code"]`, `grant_types_supported: ["authorization_code", "refresh_token"]`, `code_challenge_methods_supported: ["S256"]`, `token_endpoint_auth_methods_supported: ["none", "client_secret_basic", "client_secret_post"]`, `scopes_supported: ["fin:read", "fin:sync"]`, `client_id_metadata_document_supported: true`, `authorization_response_iss_parameter_supported: true`.

**Client registration — how fin learns which app is asking.** Before an app like Claude or Cursor can ask you for access, fin needs two facts about it: a name to show on the consent page, and the exact URL(s) it is allowed to send you back to after you approve (so a stolen authorization code can't be delivered to someone else). Plaid-style manual setup per app is not needed; MCP defines two self-service ways, and fin supports both:

1. **The app publishes a small JSON file about itself (preferred; "Client ID Metadata Document").** The app's ID _is_ an HTTPS URL on its own domain, e.g. `client_id=https://claude.ai/oauth/mcp-client.json`. When fin sees an ID like that, it downloads that file, which lists the app's name and redirect URLs. Because only the owner of `claude.ai` can put a file there, fin knows the request really comes from `claude.ai`, and the consent page shows that domain with a **verified** badge.
   - Fetch rules: HTTPS only, no redirects, 5s timeout, 64 KiB cap, and the connection refuses loopback, private, link-local, CGNAT and multicast addresses after DNS resolution, so the download cannot be aimed at fin's own network (SSRF guard; the resolver is injected for tests).
   - The file must be JSON whose `client_id` equals its own URL, with a non-empty `client_name` and `redirect_uris`, and `token_endpoint_auth_method: none`.
   - Cached in `oauth_clients` (`source = metadata_document`) for its `Cache-Control` max-age, capped at 24h.
2. **The app registers itself by API (fallback; "Dynamic Client Registration").** Older apps instead `POST /oauth/register` with their name and redirect URLs, and fin replies with a freshly generated ID (`fin_client_` + 16 random bytes). Anyone on the internet can do this and claim any name, so the consent page marks these apps **unverified** and shows the redirect host prominently.
   - Accepts RFC 7591 JSON with `authorization_code` + `refresh_token` grants and `code` responses only.
   - `token_endpoint_auth_method` `none` (default) or `client_secret_basic`/`client_secret_post`; a secret is returned once and only its SHA-256 is stored.
   - Rate-limited per IP (§13.9); registrations that never led to an approval are deleted after 7 days.

Neither path grants anything. Registering only makes the app _able to ask_; access exists only after an allow-listed user signs in and clicks **Approve**.

**Redirect URIs.** Exact string match against the registered list, except that loopback `http://127.0.0.1`, `http://[::1]` and `http://localhost` match on any port (RFC 8252 §7.3). Allowed: `https://` URIs, loopback `http://`, and private-use schemes (`cursor://…`, `vscode://…`) for native apps. Refused: any other `http://`, `javascript:`, `data:`, `file:`, fragments, and userinfo. Invalid `client_id` or `redirect_uri` renders an error at fin and **never** redirects (open-redirect guard); every other error redirects back with `error`, `state` and `iss`.

**Authorize** (`GET /oauth/authorize`). Requires `response_type=code`, `client_id`, `redirect_uri`, `code_challenge` with `code_challenge_method=S256` (plain is refused), `state` (passed through untouched), `resource` equal to `PUBLIC_URL/mcp` after normalization (RFC 8707: tokens for any other audience are never minted; a missing `resource` defaults to it for older clients), and `scope` ⊆ `{fin:read, fin:sync}` (empty → both). Stores an `oauth_authorizations` row (10-minute TTL, `user_id` null) and 303s to `/consent?request=<id>`. The `request` value is a random 32-byte base64url token (43 characters) whose SHA-256 is stored, not the row's time-ordered UUIDv7, so it cannot be guessed.

**Consent** (`/consent`, SPA). Rendered inside the authenticated layout but **exempt from the PWA install gate** (§6.5), because MCP clients open it in a normal browser tab. An unauthenticated visitor is sent to `/login?returnTo=/consent?request=<id>`; `GET /auth/google/login?returnTo=…` stores `returnTo` in the `fin_oauth` cookie and the callback 303s to it. `returnTo` must match `^/consent\?request=[A-Za-z0-9_-]{43}$`, so the login flow gains no open redirect. A non-allow-listed Google account still lands on `/login?error=not_allowed`. The page shows: client name and logo (CIMD only; DCR logos are not shown), verified/unverified badge, the client id host, where it redirects, the requested access in plain words ("Read your accounts, balances and transactions." and, when `fin:sync` was requested, a checkbox ticked by default: "Refresh data from your already-connected banks." followed by "It cannot connect, reconnect or remove banks."), and **Approve** / **Deny**. Approve sends `{scopes}` (the requested scopes minus anything unticked; `fin:read` cannot be unticked), binds `user_id`, mints an authorization code (32 bytes, SHA-256 stored, 60-second TTL) and returns `{redirectTo}`; the SPA calls `window.location.assign(redirectTo)`. CSRF on approve/deny is covered the same way as every other `/api` mutation: `SameSite=Strict` session + JSON body. Approving a client the user already granted reuses the existing `oauth_grants` row (scopes unioned).

**Token** (`POST /oauth/token`).

- Client authentication: public clients send `client_id` only; confidential DCR clients authenticate with Basic or post (constant-time hash compare). `invalid_client` → 401.
- `authorization_code`: atomically consume the code with one statement (`UPDATE oauth_authorizations SET consumed_at = now() WHERE code_hash = $1 AND consumed_at IS NULL AND code_expires_at > now() RETURNING …`), then require same `client_id`, same `redirect_uri`, same `resource`, and `S256(code_verifier) == code_challenge`. A code presented twice revokes the grant it produced (RFC 6749 §4.1.2). Issues an access token (`MCP_ACCESS_TOKEN_TTL`, default 1h) and a refresh token (`MCP_REFRESH_TOKEN_TTL`, default 30d).
- `refresh_token`: refresh tokens **rotate**. Consume with `UPDATE oauth_tokens SET used_at = now() WHERE token_hash = $1 AND kind = 'refresh' AND used_at IS NULL AND expires_at > now() RETURNING …`; if that matches nothing but the token exists with `used_at` set, it is a replay → revoke the whole grant and count `reuse_detected`. A refresh may narrow but never widen scope, and `resource` must still match. A grant also has an absolute lifetime (`MCP_GRANT_MAX_AGE`, default 90d) after which the user must approve again.
- Every response: `Cache-Control: no-store`, `token_type: Bearer`, `expires_in`, `scope`.
- No DB transactions (§0): each step is a single atomic statement, and a crash between consuming a code/refresh token and inserting new tokens only forces the client to authorize again.

**Revoke.** `POST /oauth/revoke` (RFC 7009) revokes the presented token; revoking a refresh token revokes its grant. Users revoke from **Settings → Connected apps** (`/apps`), which deletes the grant (cascading its tokens). Deleting a user row cascades everything, so removing someone from the allow-list cuts every agent off immediately.

**Resource server** (`mcpserver/auth.go`). `auth.RequireBearerToken` from the SDK with our `TokenVerifier`:

- Only the `Authorization: Bearer` header is accepted; tokens in query strings are rejected.
- One indexed query joins `oauth_tokens` → `oauth_grants` → `users`: token kind `access`, not expired, grant not revoked and within max age, `resource = PUBLIC_URL/mcp`, scope contains `fin:read`. `refresh_data` additionally requires `fin:sync`. The user is stored in the request context the same way `RequireSession` does, and every tool passes that `user.ID` to every repository call.
- `last_used_at` on the grant is updated at most every 5 minutes (conditional `UPDATE … WHERE last_used_at < now() - interval '5 minutes'`), so reads don't write on every call.
- Failures: missing/invalid/expired → 401 with the `WWW-Authenticate` header above plus `error="invalid_token"`; missing scope → 403 `error="insufficient_scope", scope="fin:read fin:sync"`, which tells the client to send the user back through consent to add the scope (step-up authorization). Clients granted only `fin:read` do not see `refresh_data` in `tools/list` at all.
- The `fin_session` cookie is **never** accepted on `/mcp`, so a logged-in browser cannot be driven into it cross-site.
- DNS-rebinding guard required by the transport spec: a request carrying an `Origin` header is rejected with 403 unless the origin is in `MCP_ALLOWED_ORIGINS` (default empty; hosted agents call server-to-server and send none).

### 13.5 Database

Four migrations, one table each, following §4's conventions (`text` enums without `CHECK`, `uuidv7()` ids, every user-data table carries `user_id` with `ON DELETE CASCADE`).

```sql
CREATE TABLE oauth_clients (
    id                          text        PRIMARY KEY,
    source                      text        NOT NULL,
    client_name                 text        NOT NULL,
    client_uri                  text,
    logo_uri                    text,
    redirect_uris               text[]      NOT NULL,
    token_endpoint_auth_method  text        NOT NULL,
    client_secret_hash          bytea,
    metadata_fetched_at         timestamptz,
    metadata_expires_at         timestamptz,
    created_at                  timestamptz NOT NULL DEFAULT now(),
    last_used_at                timestamptz
);
```

```sql
CREATE TABLE oauth_authorizations (
    id                     uuid        PRIMARY KEY DEFAULT uuidv7(),
    request_token_hash     bytea       NOT NULL UNIQUE,
    client_id              text        NOT NULL REFERENCES oauth_clients (id) ON DELETE CASCADE,
    user_id                uuid        REFERENCES users (id) ON DELETE CASCADE,
    redirect_uri           text        NOT NULL,
    scope                  text        NOT NULL,
    resource               text        NOT NULL,
    state                  text,
    code_challenge         text        NOT NULL,
    code_hash              bytea       UNIQUE,
    code_expires_at        timestamptz,
    consumed_at            timestamptz,
    grant_id               uuid,
    created_at             timestamptz NOT NULL DEFAULT now(),
    expires_at             timestamptz NOT NULL
);
```

```sql
CREATE TABLE oauth_grants (
    id            uuid        PRIMARY KEY DEFAULT uuidv7(),
    user_id       uuid        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    client_id     text        NOT NULL REFERENCES oauth_clients (id) ON DELETE CASCADE,
    scope         text        NOT NULL,
    resource      text        NOT NULL,
    created_at    timestamptz NOT NULL DEFAULT now(),
    expires_at    timestamptz NOT NULL,
    last_used_at  timestamptz,
    revoked_at    timestamptz,
    UNIQUE (user_id, client_id)
);

CREATE INDEX oauth_grants_user_id_idx ON oauth_grants (user_id);
```

```sql
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
```

Expired authorizations and tokens are deleted opportunistically by `OAuthService` on each token issuance (`DeleteExpired`, the same pattern as sessions). Nothing secret is stored in plaintext: codes, request tokens, access/refresh tokens and client secrets are SHA-256 hashes (high-entropy random values, so a plain hash is sufficient; no `SECRET_KEY` involvement).

### 13.6 Tools

Every tool except `refresh_data` is annotated `readOnlyHint: true`, `destructiveHint: false`, `idempotentHint: true`, `openWorldHint: false`. All tools declare an `outputSchema`, and return `structuredContent` plus the same JSON as a text block for clients that ignore structured output. Conventions shared by every tool:

- Money is a decimal **string**, outflows negative (§0). Every amount travels with its `currency`; nothing ever sums across currencies (aggregations always group by currency implicitly).
- Dates are `YYYY-MM-DD` (`posted_on`), timestamps RFC 3339.
- IDs are fin UUIDs. Plaid `external_id`s, access tokens, cursors, `last_error` text and credentials never appear in any DTO; DTOs live in `mcpserver/types.go` and are mapped from domain types, like the API DTOs (§5.6).
- Shared transaction filter (`TransactionQuery`), accepted by every transaction tool:

| Field                                | Meaning                                                                                     |
| ------------------------------------ | ------------------------------------------------------------------------------------------- |
| `from`, `to`                         | inclusive `posted_on` range; default last 90 days; max span 5 years                         |
| `account_ids`, `exclude_account_ids` | include / exclude lists                                                                     |
| `connection_ids`                     | restrict to institutions                                                                    |
| `account_types`                      | `depository`, `credit`, `loan`, `investment`, `other`                                       |
| `categories`, `exclude_categories`   | exact Plaid category values (discover with `list_facets`)                                   |
| `exclude_transfers`                  | drops `TRANSFER_IN`, `TRANSFER_OUT`, `LOAN_PAYMENTS` (same set as `/api/summary/cash-flow`) |
| `merchant`                           | case-insensitive substring over `merchant_name` and `name` (`ILIKE`, wildcards escaped)     |
| `direction`                          | `inflow` (amount > 0), `outflow` (amount < 0), `any`                                        |
| `min_amount`, `max_amount`           | bounds on the **absolute** amount, decimal strings                                          |
| `currency`                           | ISO code                                                                                    |
| `pending`                            | `true`, `false`, or omitted for both                                                        |

| Tool                     | Input                                                                                                                                                                                                                                                                                                                                           | Output                                                                                                                                                                                                                 |
| ------------------------ | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `list_connections`       | —                                                                                                                                                                                                                                                                                                                                               | institution, status (`active`/`needs_reauth`), `last_synced_at`, `has_error`, account count. Lets the agent judge freshness, decide whether to call `refresh_data`, and caveat data ("Chase needs sign-in in the app") |
| `list_accounts`          | `connection_ids`, `account_types`                                                                                                                                                                                                                                                                                                               | id, institution, name, official name, mask, type, subtype, current/available balance, currency                                                                                                                         |
| `get_balances`           | `exclude_account_ids`                                                                                                                                                                                                                                                                                                                           | per currency: assets, liabilities, net worth; plus per-account-type subtotals                                                                                                                                          |
| `search_transactions`    | `TransactionQuery` + `sort` (`date_desc`, `date_asc`, `amount_asc`, `amount_desc`), `page`, `page_size` (≤ 500)                                                                                                                                                                                                                                 | transactions with account/institution names inlined, `total`, `has_next`                                                                                                                                               |
| `get_transaction`        | `transaction_id`                                                                                                                                                                                                                                                                                                                                | one transaction incl. `authorized_on`                                                                                                                                                                                  |
| `list_facets`            | `TransactionQuery`, `facet` (`category`, `merchant`, `account`, `currency`), `limit` (≤ 500)                                                                                                                                                                                                                                                    | distinct values with count and sum, so agents learn valid filter values before filtering                                                                                                                               |
| `aggregate_transactions` | `TransactionQuery` + `group_by` (0–3 of `day`, `week`, `month`, `quarter`, `year`, `weekday`, `category`, `merchant`, `account`, `connection`, `account_type`, `direction`, `pending`) + `metrics` (1+ of `sum`, `inflow`, `outflow`, `count`, `avg`, `min`, `max`, `median`) + `order_by` (a metric or dimension, asc/desc) + `limit` (≤ 1000) | rows of `{dimensions…, currency, metrics…}`, `truncated` flag. Covers "spend by category per month", "top 10 merchants this year", "average weekday spend", "income by account"                                        |
| `cash_flow`              | `TransactionQuery` + `interval` (`day`, `week`, `month`, `quarter`, `year`); `exclude_transfers` defaults to `true` here                                                                                                                                                                                                                        | per period and currency: inflow, outflow (positive), net, savings rate                                                                                                                                                 |
| `find_recurring`         | `from`/`to` (default 13 months), `min_occurrences` (default 3)                                                                                                                                                                                                                                                                                  | candidate subscriptions/bills: merchant, typical amount, cadence (`weekly`, `biweekly`, `monthly`, `quarterly`, `yearly`), last and next expected date, occurrences                                                    |
| `refresh_data`           | `connection_ids` (default: all of the user's connections), `stale_after_minutes` (default 60, minimum 15)                                                                                                                                                                                                                                       | per connection: `synced`, `skipped_fresh`, `skipped_needs_reauth` ("sign in again in the fin app"), or `failed`, with the new `last_synced_at`; plus counts. Requires `fin:sync`                                       |

`aggregate_transactions` is the flexible-analysis core. It never sees SQL from the caller: `group_by`, `metrics` and `order_by` are Go enums mapped to fixed SQL expressions in `repository/postgres/transaction_query.go` (e.g. `month` → `date_trunc('month', t.posted_on)::date`, `merchant` → `coalesce(t.merchant_name, t.name)`, `median` → `percentile_cont(0.5) WITHIN GROUP (ORDER BY t.amount)`, `outflow` → `-sum(t.amount) FILTER (WHERE t.amount < 0)`); every value is a bind parameter, `t.user_id = $1` is always the first predicate, and `t.currency` is always a grouping column. The query builder is a pure function (`buildAggregateQuery(spec) (string, []any)`) and is the one part of `repository/postgres` that gets unit tests: golden SQL per dimension/metric, every identifier comes from the whitelist, user id is always `$1`.

`find_recurring` runs in Go over `aggregate`-style rows (merchant, amount, dates) fetched with one query: group by normalized merchant, compute median gap between occurrences, accept when ≥ `min_occurrences` and gaps cluster within ±20% of a known cadence and amounts within ±15% of the median. Pure function, unit-tested.

`refresh_data` is the only tool with a side effect. It runs the same `ConnectionService.Sync` the Refresh button and the scheduler use, through the narrow `connectionSyncer` interface (§13.1), with a new trigger `metrics.SyncTriggerMCP`:

- Only connections that are `active` and whose `last_synced_at` is older than `stale_after_minutes` are synced; everything else is skipped and reported. A hard **15-minute floor** applies whatever the input says, so an agent in a loop cannot hammer Plaid; the `/mcp` rate limit (§13.9) applies on top.
- `needs_reauth` connections are never touched: fixing them needs Plaid Link in the app, which is exactly what MCP must not do. The result tells the agent to ask the user to reconnect in fin.
- Connections are synced sequentially under a 60-second deadline (separate from the 5-second query deadline); whatever did not finish is reported as `failed` and the next sync picks it up, because syncs are idempotent (§5.5).
- It pulls what Plaid already has (`/transactions/sync`); it does not call Plaid's billed `/transactions/refresh`, so freshness is bounded by Plaid's own update cadence. The server `instructions` say this, and tell agents to check `list_connections` first and refresh only when the data is older than the question needs.
- Annotations: `readOnlyHint: false`, `destructiveHint: false`, `idempotentHint: true`, `openWorldHint: true`.

Response size: every tool caps rows (limits above) and sets `truncated`/`has_next`, so a single call cannot dump the database into an agent's context. Every query runs under a 5-second context deadline.

Not exposed, ever: link tokens, public-token exchange, re-auth (update-mode link tokens and completion), connection removal, card-art selection, Plaid credentials, sessions, other users. Write tools beyond `refresh_data` (e.g. notes or categories) would need a new scope and a new consent line; out of scope for this phase.

Resources and prompts are out of scope for v1; everything is a tool, which every client supports. An `instructions` string on the server explains the money sign convention, currencies, that data can be up to one `SYNC_CRON` period old unless refreshed, when to use `refresh_data`, and that transaction text is untrusted data.

### 13.7 Wiring

`serve.go` adds, in the same explicit-constructor style:

```go
oauthClients := postgres.NewOAuthClientRepository(pool)
oauthAuthorizations := postgres.NewOAuthAuthorizationRepository(pool)
oauthGrants := postgres.NewOAuthGrantRepository(pool)
oauthTokens := postgres.NewOAuthTokenRepository(pool)
transactionQueries := postgres.NewTransactionQueryRepository(pool)

metadataFetcher := oauth.NewMetadataDocumentFetcher(httpClient, oauth.PublicAddressesOnly)
oauthService := service.NewOAuthService(cfg.OAuth(), oauthClients, oauthAuthorizations, oauthGrants, oauthTokens, metadataFetcher, recorder)

mcpHandler := mcpserver.NewHandler(mcpserver.Deps{
	Verifier:     oauthService,
	Connections:  connections,
	Accounts:     accounts,
	Summary:      summary,
	Transactions: transactionQueries,
	Syncer:       connectionService,
	Recorder:     recorder,
	Config:       cfg.MCP(),
})
```

`mcpserver.Deps` holds **repository interfaces**, one `TokenVerifier` interface (`VerifyAccessToken(ctx, token) (domain.User, []domain.Scope, error)`) and one `connectionSyncer` interface (`Sync` only), all declared inside `mcpserver`. None of them exposes link, exchange, re-auth or remove. `.golangci.yml` enables `depguard` with a rule scoped to `internal/mcpserver/**` denying `github.com/tahminator/fin/internal/provider`, `.../encryption`, `.../service` and `github.com/plaid/plaid-go`; `serve.go` is where `ConnectionService` is handed over as the interface. `ConnectionRepository` is passed as a read-only interface (`ListByUser` only), so the repository methods that mutate connections are out of reach too.

### 13.8 Configuration

| Variable                | Default | Notes                                                                                   |
| ----------------------- | ------- | --------------------------------------------------------------------------------------- |
| `MCP_ENABLED`           | `true`  | `false` unregisters `/mcp`, `/oauth/*` and the metadata endpoints (all 404)             |
| `MCP_ACCESS_TOKEN_TTL`  | `1h`    |                                                                                         |
| `MCP_REFRESH_TOKEN_TTL` | `720h`  | rotating; each refresh issues a new one                                                 |
| `MCP_GRANT_MAX_AGE`     | `2160h` | re-consent after 90 days regardless of use                                              |
| `MCP_ALLOWED_ORIGINS`   | empty   | comma list for browser-based clients; empty rejects any request with an `Origin` header |

Derived, not configured: `cfg.OAuthIssuer() = PUBLIC_URL`, `cfg.MCPResource() = PUBLIC_URL + "/mcp"`. Validation additions: in production `PUBLIC_URL` must be `https://` with no path, query or trailing slash (the issuer and resource identifiers are compared as exact strings); `MCP_ACCESS_TOKEN_TTL` ≤ 24h. Dev: Vite's `server.proxy` gains `/mcp`, `/oauth`, `/.well-known`, and `PUBLIC_URL=http://localhost:5173` keeps issuer, resource and consent page on one origin.

### 13.9 Abuse protection and privacy

- Rate limits (in-process `golang.org/x/time/rate`, keyed per client IP from the ingress `X-Forwarded-For` hop, per replica): `/oauth/register` 10/hour, `/oauth/token` 60/minute, `/oauth/authorize` 60/minute, `/mcp` 120/minute per **grant**. Excess → 429 with `Retry-After`. Request bodies capped at 64 KiB (`http.MaxBytesReader`).
- Logging: the request logger never logs query strings for `/oauth/*` (codes, state) or any `Authorization` header. MCP calls log tool name, user id, grant id, outcome, rows returned and duration; tool arguments are not logged (merchant searches are personal data).
- Prompt injection: merchant and transaction names are third-party text and reach the agent verbatim. The server is scoped to one user and its only side effect is `refresh_data`, which is idempotent, rate-floored and touches nothing but already-linked connections, so an injected instruction can at most make the agent read more or refresh early; it cannot move money, link or remove banks, or touch another user. The server `instructions` remind clients that transaction text is untrusted data.
- Ingress: `/mcp`, `/oauth/*` and `/.well-known/oauth-*` are public (TLS terminated at the ingress); `/metrics`, `/healthz`, `/readyz` stay blocked as today.

### 13.10 Observability

New `iota` label enums in `metrics/labels.go`: `RouteGroup` gains `mcp` and `oauth` (`routeGroupOf` maps `/mcp`, `/oauth/`, `/.well-known/`); `MCPTool` (one value per tool in §13.6); `SyncTrigger` gains `mcp` (so `fin_syncs_total` and `fin_sync_duration_seconds` split MCP refreshes from manual and scheduled ones; 15 and 5 × buckets series); `ToolOutcome` (`success`, `invalid_params`, `not_found`, `error`); `OAuthGrantType` (`authorization_code`, `refresh_token`); `OAuthOutcome` (`success`, `invalid_request`, `invalid_client`, `invalid_grant`, `reuse_detected`, `error`); `ClientSource` (`metadata_document`, `registered`); `ConsentDecision` (`approved`, `denied`).

| Metric                                            | Type      | Labels                  | Max series      |
| ------------------------------------------------- | --------- | ----------------------- | --------------- |
| `fin_mcp_tool_calls_total` (`mcpserver`)          | counter   | `tool`, `outcome`       | 10 × 4          |
| `fin_mcp_tool_duration_seconds`                   | histogram | `tool`                  | 10 × buckets    |
| `fin_oauth_token_requests_total` (`OAuthService`) | counter   | `grant_type`, `outcome` | 12              |
| `fin_oauth_client_registrations_total`            | counter   | `source`, `outcome`     | 4               |
| `fin_oauth_consents_total`                        | counter   | `decision`              | 2               |
| `fin_oauth_active_grants` (DB snapshot)           | gauge     | `user`                  | allow-list size |
| `fin_oauth_clients` (DB snapshot)                 | gauge     | `source`                | 2               |

`Recorder` gains `MCPToolCalled(tool, outcome, elapsed)`, `OAuthTokenRequested(grantType, outcome)`, `OAuthClientRegistered(source, outcome)` and `OAuthConsentDecided(decision)`. Client names and ids are **not** labels (unbounded). Alerts:

```
increase(fin_oauth_token_requests_total{outcome="reuse_detected"}[1d]) > 0
increase(fin_oauth_client_registrations_total[1h]) > 20
increase(fin_mcp_tool_calls_total{outcome="error"}[1h]) > 0
increase(fin_syncs_total{trigger="mcp", outcome!="success"}[1d]) > 0
```

### 13.11 Frontend

- Routes: `/consent` (authenticated layout, install gate skipped, no AppShell nav) and `/apps` (authenticated, linked from the account menu as **Connected apps**).
- `lib/queries/oauth.ts`: `useAuthorizationRequest(requestId)`, `useApproveAuthorization`, `useDenyAuthorization`, `useGrants` (infinite), `useRevokeGrant` (invalidates `/api/oauth/grants`).
- Login: the authed layout's redirect and `login.Page.tsx` carry a zod-validated `returnTo` search param through to `/auth/google/login?returnTo=…` (only `/consent?request=…` values pass, mirroring the server check).
- `/consent` follows the component contract (§6.4): loading, 404 ("This request expired, start again from your app"), and the view with Approve/Deny. `/apps` lists grants with client name, verified badge, granted scopes ("Read" / "Read + refresh"), connected date, last used, and a **Revoke** confirm dialog.
- The MCP endpoint URL (`PUBLIC_URL/mcp`) is shown on `/apps` with a copy button, plus one-line setup hints (Claude: add a custom connector with that URL; Claude Code: `claude mcp add --transport http fin <url>`).

### 13.12 Tests

Same rules as §5.13 (stdlib, fakes, no network/DB/sleeps; `now` injected).

| Package                                     | Covered behavior                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                               |
| ------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `internal/oauth`                            | S256 known-answer vectors from RFC 7636; plain method rejected; redirect URI rules (https ok, http non-loopback rejected, loopback any port matches, custom scheme ok, `javascript:`/fragment/userinfo rejected, exact match otherwise); scope parse/subset; CIMD fetcher refuses private/loopback/link-local resolutions, redirects, oversize bodies, mismatched `client_id`, missing `redirect_uris`                                                                                                                                                                                                                                                                                                                                                                                         |
| `internal/service` — oauth                  | unknown client / bad redirect → error without redirect; missing PKCE → `invalid_request`; wrong `resource` → `invalid_target`; code exchanges once, second use → `invalid_grant` + grant revoked; expired code; verifier mismatch; redirect_uri mismatch; client mismatch; refresh rotates and the old token then triggers `reuse_detected` + grant revoked; refresh cannot widen scope; grant past max age refused; revoked grant → verify fails; stored hashes are SHA-256 of the issued tokens, never the tokens; confidential DCR client wrong secret → `invalid_client`; recorder called with the right enums                                                                                                                                                                             |
| `internal/handler` — oauth                  | metadata documents (exact fields, issuer/resource strings, CORS header); the OpenAPI spec contains every `/oauth/*`, `/.well-known/*` and `/mcp` operation with the `mcpOAuth` scheme; consent approve honours unticked `fin:sync` and refuses dropping `fin:read`; 401 `WWW-Authenticate` format; token endpoint returns RFC 6749 JSON errors with `no-store`; register validates and never echoes a stored hash; consent endpoints: 401 without session, 404 for expired/consumed/other user's request, approve returns `code`+`state`+`iss`, deny returns `access_denied`; grants tenancy (user B cannot list or revoke A's grants)                                                                                                                                                         |
| `internal/handler` — auth                   | `returnTo` accepted only for `/consent?request=<43 chars>`; anything else (absolute URL, `//evil`, other paths) falls back to `/`                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                              |
| `internal/mcpserver`                        | via the SDK's in-memory transport: every tool lists with an output schema and every tool but `refresh_data` with `readOnlyHint`; `refresh_data` hidden and 403 `insufficient_scope` without `fin:sync`; `refresh_data` syncs only stale `active` connections with `SyncTriggerMCP`, never `needs_reauth` ones, honours the 15-minute floor, reports per-connection results, stops at the deadline; `x-mcp-tools` in the generated spec equals `tools/list` (names, schemas, scopes); bearer middleware (no header, query-string token, expired, wrong scope, `Origin` not allowed, session cookie ignored); every repository call receives the token's user id (tenancy); filter mapping (defaults, 5-year cap, page size caps, invalid enum → `invalid_params`); DTOs contain no external ids |
| `internal/repository/postgres` (query only) | `buildAggregateQuery` golden SQL for each dimension and metric; `$1` is always the user id; identifiers only from the whitelist; currency always grouped; `merchant` wildcards escaped                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                         |
| `internal/mcpserver` — recurring            | monthly/biweekly/yearly cadences detected; jittered dates within tolerance accepted; amount drift beyond 15% rejected; fewer than `min_occurrences` ignored                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                    |
| `internal/metrics`                          | new enums unique and complete; series pre-initialized; `routeGroupOf` maps `/mcp`, `/oauth/…`, `/.well-known/…`                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                |

Manual verification:

1. `npx @modelcontextprotocol/inspector` against `http://localhost:5173/mcp`: discovery, DCR, consent, token, `tools/list`, every tool returns structured content.
2. Claude (custom connector) and Claude Code (`claude mcp add --transport http fin https://<host>/mcp`) against staging: CIMD client shows **verified**, consent → tools usable; ask "what did I spend on dining per month this year?" and compare with the transactions page.
3. A non-allow-listed Google account at consent → `/login?error=not_allowed`, no code issued.
4. Revoke from `/apps` → the next tool call returns 401 and the client restarts authorization.
5. Replay a used refresh token with `curl` → `invalid_grant`, grant revoked, `fin_oauth_token_requests_total{outcome="reuse_detected"}` increments.
6. Consent page opens in Safari on iOS (not the installed PWA) without the install gate.
7. `curl -X POST /mcp -H 'Origin: https://evil.example'` with a valid token → 403.
8. Ask the agent to "refresh my data, then show today's transactions": `refresh_data` syncs stale connections, a second call within 15 minutes reports `skipped_fresh`, `fin_syncs_total{trigger="mcp"}` increments; with a sandbox connection in `needs_reauth` it is reported, not synced. Re-approve with refreshing unticked → `refresh_data` disappears from the tool list.
9. `/swagger` shows the `mcp` and `oauth` tags and the `mcpOAuth` scheme.

### 13.13 Milestones

**M7 — OAuth authorization server**

- Migrations 000009–000012, `internal/oauth`, `OAuthService`, OAuth + consent handlers (with their OpenAPI documentation), `returnTo` on login, metadata endpoints, `/consent` and `/apps` pages, rate limits, metrics; tests per §13.12.
- Accept: manual checks 3–6 using the Inspector's auth debugger; `just lint && just test` green.

**M8 — MCP tools**

- `internal/mcpserver` with the bearer middleware, every tool in §13.6 including `refresh_data` and `SyncTriggerMCP`, the `/mcp` OpenAPI operation with `x-mcp-tools`, `transaction_query` repository + builder tests, `depguard` rule, Vite proxy entries, ingress paths.
- Accept: manual checks 1, 2, 7, 8 and 9; `depguard` fails a deliberate `internal/provider` import from `mcpserver`.

### 13.14 Risks

- **Public authorization server**: DCR lets anyone create clients. Mitigated by rate limits, unverified labelling, exact redirect matching, unused-client cleanup, and the allow-list — a client is useless until an allow-listed user approves it.
- **Spec churn**: MCP auth has changed across revisions (DCR-first → CIMD-first). Discovery is served at both RFC 8414 and OIDC paths and both client-identification methods are supported; recheck against the newest revision and the SDK at scaffold time.
- **Phishing via consent**: a malicious client can mimic a known name. CIMD clients show their verified domain; DCR clients show **unverified** and the redirect host; the consent copy states exactly what is allowed and that banks cannot be connected, reconnected or removed.
- **Sensitive data leaves fin**: approved agents send transactions to their model provider. This is the point of the feature, but the consent screen says so in plain words, and grants expire after `MCP_GRANT_MAX_AGE`.
- **Agent-triggered refreshes**: more Plaid calls than the cron alone. Bounded by the 15-minute floor per connection, the `/mcp` rate limit and the `fin:sync` opt-out; `/transactions/sync` is not billed per call.
- **In-memory rate limits are per replica**: acceptable at one or two replicas; move to Postgres-backed counters if the deployment scales out.
