# fin Style Guide

This guide collects the engineering conventions and technical constraints in [PLAN.md](PLAN.md). The plan remains the source for feature behavior, endpoint details, milestones, and deployment configuration. These conventions describe the intended design; they do not imply every part is already implemented.

## General

- Keep implementations simple and group related behavior by resource or responsibility.
- Use natural blank lines between logical blocks in every written file, including Go, TypeScript, SQL, Justfiles, Dockerfiles, YAML, and env files. Separate declaration blocks, independent groups of statements, functions and types; put a blank line before a return following other logic. In Go, keep `value, err := f()` immediately adjacent to its error check.
- Do not write comments in code or configuration. Go permits `//go:` directives; the explicitly requested `SECRET_KEY` generation hint in `.env.example` is the one other exception. Use Justfile `[doc()]` attributes instead of comments. Generated frontend files are excluded from the comment rule.
- Split types with more than a couple of small methods into sibling files. Move DTOs into a sibling such as `connection_types.go` when they make a handler file longer than a screen.
- Add dependencies at their latest version when scaffolding; use the repository's pinned versions and lockfiles thereafter. Use `just outdated` to inspect drift.

## Go backend

### Structure and dependencies

- Use Go, `net/http` ServeMux patterns, `log/slog`, and huma with the `humago` adapter. Wire dependencies explicitly with constructors in `serve.go`; do not introduce a dependency-injection framework.
- Keep domain types, string enums, and sentinel errors in `internal/domain`. Domain types have no JSON tags. Define repository interfaces in `internal/repository` and pgx implementations with raw SQL in `internal/repository/postgres`.
- Handlers may depend on repository interfaces and concrete services. Services depend on interfaces for repositories, providers, and identity. Read-only endpoints may use repositories directly rather than adding pass-through services.
- Keep provider-specific details behind `provider.Provider` and identity details behind `IdentityProvider`. The provider accepts plaintext access tokens; encryption belongs to the service layer.
- Group each resource's HTTP operations in a handler with `Register(api huma.API)`. List all routes centrally in `internal/server/routes.go`.
- Keep API DTOs in `internal/handler`, separate from domain types, with JSON and huma documentation and enum tags. Map explicitly so internal fields such as tokens and cursors cannot leak.
- Represent domain enums as named string types with constants and `Valid()`; expose their values with huma `enum` tags. Use sentinel errors such as `ErrNotFound`, `ErrNotAllowed`, and `ErrConflict`; map them to appropriate HTTP errors at the handler boundary.
- Configuration is handwritten in `config.NewFromEnv()` using the standard library, with defaults, aggregated errors, and cross-field validation. Do not add an env library or config package tests.

### API and persistence

- Use offset pagination: `page` is 1-based, `pageSize` defaults to 25 and is capped at 100. Return `Page[T]` with `items`, `page`, `pageSize`, `total`, and `hasNext`; `items` must be an empty array rather than null.
- Generate the OpenAPI document from huma without requiring a database or environment. Generate frontend types from that document; do not commit generated `openapi.json` or `schema.gen.ts`.
- Enforce user-data tenancy in SQL: repository methods that access user-owned data take a user ID and filter on it. Translate `pgx.ErrNoRows` to `domain.ErrNotFound`.
- Use Postgres `uuid DEFAULT uuidv7()` for IDs. Store enums in `text` columns without `CHECK` constraints.
- Store money as `numeric(19,4)` in Postgres and decimal strings in Go and JSON; do not use floating-point arithmetic for stored amounts. Outflows are negative. Select numeric values as text, cast input to numeric, and perform sums in SQL. Map Plaid's positive outflows to negative values at the provider boundary.
- Use raw SQL constants near the methods that execute them. Use `pgx.CollectRows` and `pgx.RowToStructByName` with `db` tags where appropriate. Use separate readable page and count queries for lists.
- Format SQL with uppercase keywords, one column per line, aligned types in table definitions, explicit `AS` aliases, and trailing semicolons. Apply the same style in migrations and Go SQL strings.
- Use sequential golang-migrate migrations, one table per migration, with matching up and down files. Run migrations explicitly through the CLI or Justfile, never from the application at startup. Do not use database transactions; sync writes are idempotent and the cursor is saved last.

### Security and operations

- Authenticate with Google OIDC and PKCE. Only allow users already in the manually managed `users` allow-list. Use random session tokens, store only their SHA-256 hashes, and keep session cookies `HttpOnly`, `Secure`, and `SameSite=Strict`; the short-lived OAuth callback cookie uses `SameSite=Lax`.
- Encrypt Plaid access tokens at rest with AES-256-GCM using `cipher.NewGCMWithRandomNonce` and the 32-byte `SECRET_KEY`. Repositories store opaque ciphertext; API DTOs never include it. There is no key versioning: changing the key requires re-linking connections. Per-user Plaid secrets are plaintext in the database, so protect database access and backups.
- Keep `/healthz`, `/readyz`, and `/metrics` on the separate `OPS_ADDR` listener. Serve static assets from `STATIC_DIR`, not `go:embed`; do not let unknown `/api/` routes fall through to the SPA.
- Only `internal/metrics` manipulates Prometheus instruments. Other packages record through `metrics.Recorder`; use zero-sized `metrics.Prometheus{}` in production and `metrics.Noop{}` or embedding fakes in tests.
- Metric labels use typed `iota` enums with exhaustive `String()` switches, and known series are pre-initialized. The sole data-derived label is `user` on database snapshot gauges, sourced from allow-listed emails. Do not add arbitrary high-cardinality labels.
- Log requests as structured JSON with `slog`. Record sync failures in metrics. Scheduled sync uses a cron parser and an application-owned goroutine; `SYNC_CRON=off` disables it. No webhooks are planned.

### Go quality and tests

- Use golangci-lint v2 with `gofumpt` and `goimports` (local prefix `github.com/tahminator/fin`). Enable the plan's security, error, context, SQL, exhaustiveness, and resource-closing linters. Disable checks that require package or exported-symbol comments.
- Run the Go AST no-comments checker through `just lint-go`; allow only Go directives.
- Unit-test backend business logic in adjacent `*_test.go` files using standard `testing`, table-driven `t.Run`, `humatest`, and handwritten interface fakes. Embed interfaces or `metrics.Noop` when a fake only needs to override selected methods. Do not add mocking or assertion libraries.
- Keep unit tests deterministic and isolated: no network, database, or sleeps. Inject time where needed and test scheduler work through `RunOnce`. Run `just test` (`go test -race ./...`).
- The plan excludes config, raw Postgres repositories, the thin Google identity wrapper, dependency wiring, and seed flag parsing from unit tests. Verify complex cross-system flows manually as described in PLAN.md §5.13. There are no frontend tests yet.

## Frontend

### Stack and organization

- Use Vite, React, TypeScript, React Router, Tailwind CSS v4, shadcn, TanStack Query, `openapi-fetch`, `openapi-react-query`, and generated `openapi-typescript` types. Use pnpm only for the `js/` application; pin the package manager and enforce it with `devEngines`.
- Use an explicit route table in `src/router.ts`. Mirror URL paths under `src/app/`; route entry points use `name.Page.tsx`. Prefix private route directories with `_` and restrict `_components/` and `_hooks/` imports to their owning route subtree. Other frontend filenames use kebab-case.
- Use the `@/` alias for internal imports, exports, and dynamic imports; do not use relative imports. Prefer named exports. Keep shared UI in `src/components`, shared logic in `src/lib`, and route-private code under its route.
- Enable strict TypeScript, including `noUncheckedIndexedAccess`, `exactOptionalPropertyTypes`, `noImplicitOverride`, `noPropertyAccessFromIndexSignature`, unused checks, `verbatimModuleSyntax`, and `erasableSyntaxOnly`. Do not use TypeScript enums or explicit `any`; use consistent type imports and handle promises.
- Expose aliases for generated API schemas from `lib/types/api.ts`; components do not index generated schema types directly.

### Data and components

- Create the typed API client in `lib/api/client.ts`. Access it only inside `src/lib`; components consume resource-specific composite hooks from `lib/queries`. Keep React Query as the source of truth for server data and let mutations own their query invalidations, including notifications after sync, link, or re-auth.
- Infinite queries explicitly start at page 1, use `hasNext` to determine the next page, and flatten pages with `select`. Use the plan's 30-second query staleness, no retries on 4xx, and clear the query client and navigate to login on 401.
- Route components handle loading, error, and empty states before rendering view components with guaranteed data. Use shared loading, error, and empty states.
- Do not call `useEffect`, `useLayoutEffect`, or `useInsertionEffect` outside `src/lib/hooks/`. Put browser subscriptions in shared hooks using `useSyncExternalStore`.
- Use at most one `useState` per component; move additional state into a composite hook, page-local under `_hooks/` or shared under `lib/hooks/`.
- Use shadcn components for UI; custom UI is reserved for the install guide and list rows. Keep the implementation simple.
- The PWA install gate precedes login and authenticated routes, with a development-only escape hatch. Do not cache API responses; exclude API, auth, and documentation routes from the service worker's navigation fallback.

### Frontend quality

- Use oxlint and oxfmt only, not ESLint or Prettier. Oxlint may load `eslint-plugin-react-hooks` as a JS plugin; ESLint itself is not the lint runner. Enable type-aware linting and deny warnings.
- Enforce alias imports, no comments, route-private imports, effect placement, and the component state limit with the custom `fin` oxlint plugin. Restrict direct API client and React Query hook imports outside `src/lib`. Exclude generated API types from lint and formatting.
- Enable the plan's correctness, suspicious, pedantic, performance, and style categories, accessibility and React rules, exhaustive switches, strict booleans, promise checks, import-cycle checks, and no-console rule. Turn off noisy rules explicitly in config, not with inline disables.
- Use oxfmt with a 100-column print width, sorted imports and Tailwind classes, preserving natural blank lines. Run `pnpm check` for typecheck, lint, and format checks; run `just lint-js` for the repository entry point.

## CI/CD scripts and infrastructure

- CI/CD follows the reference repositories identified in PLAN.md §12; do not invent new mechanisms when the references specify one. Use the `@tahminator/pipeline` APIs for Docker, GitHub, versioning, and local Postgres.
- The frontend app uses pnpm; `.github/scripts` is a separate Bun-based tooling package. Callable scripts live in grouped directories with an `index.ts`, and reusable modules live in `src/lib/`.
- Parse script arguments at module scope with strict yargs when arguments exist; omit yargs otherwise. Put work in `async function main()`, parse required environment variables with `parseCiEnv`, and exit explicitly on success or failure. Keep shared constants in `src/lib/consts.ts`.
- Apply the no-comments and natural-blank-lines rules to `.github` too. Composite actions use named inputs and a named bash step; pin external actions as specified by the plan.
- Keep migrations an explicit deployment step before rollout. The runtime image contains neither migration files nor migration code. Use a multi-stage build and a nonroot distroless runtime; derive build tool versions from the repository's Go and frontend configuration.
- Keep local secrets in gitignored env files; do not commit credentials, generated build output, or generated API schemas. Use Justfile recipes as the common interface for local development, checks, migrations, and builds.
