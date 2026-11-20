# fin

Personal finance tracking with a Go API, Postgres, Plaid and an installable React PWA.

## Local setup

Install Go, Node, pnpm, just, Docker, golangci-lint, golang-migrate and jq. Docker must be running.

1. Copy `.env.example` to `.env` and fill in the Google OAuth client ID and secret. Generate a persistent `SECRET_KEY` using the command in the example. Do not change or lose it: stored Plaid access tokens cannot be decrypted without it.
2. Register `http://localhost:5173/auth/google/callback` with the Google OAuth web client. Add each local user as a Google test user if the OAuth app is in testing mode.
3. Run `just install`, `just db` and `just migrate`.
4. Add each allowed user and their Plaid sandbox credentials with `just seed <email> <client-id> <secret>`. The command accepts an optional final environment argument, `sandbox` or `production`. Arguments can appear in shell history and process listings; avoid using production credentials on a shared machine.
5. Run `just dev` and open `http://localhost:5173`. Install the app using the on-screen guide before signing in. For development in a browser tab, set `VITE_SKIP_INSTALL_GATE=true` in `js/.env.local` and restart Vite.

The local Postgres container listens on port 5433. `just --list` shows all available commands. `just lint` runs Go and frontend checks; `just test` runs backend tests.

The frontend uses React Router with an explicit route table in `js/src/router.ts`. `js/src/app` mirrors the URL structure, with page entry points named `name.Page.tsx`; `_`-prefixed files and directories are implementation details rather than routes.

## Operations

`SYNC_CRON` controls scheduled account refreshes. The default is `0 */6 * * *`; use `off` to disable scheduled refreshes. Users can also refresh manually. Re-authentication requirements appear in the app after a sync detects them.

The API listens on `:8080`; `/openapi.json` and `/swagger` expose its schema. The separate operations listener defaults to `:9090`: `/healthz` is liveness, `/readyz` checks Postgres, and `/metrics` exposes Prometheus metrics. Keep the operations listener private because metrics include per-user data.

Run migrations explicitly with `just migrate`; the server never applies migrations on startup. `just docker-build` builds the runtime image without migrations. Before deploying, apply migrations separately with the golang-migrate CLI against the target database.

Production requires a public HTTPS URL, a cookie domain, secure cookies, Google OAuth credentials and each user's Plaid credentials. Register the production Google callback at `<PUBLIC_URL>/auth/google/callback` and the Plaid redirect URI at `<PUBLIC_URL>/connections/oauth`. Protect the database and backups: Plaid access tokens are encrypted, but manually entered Plaid client secrets are stored in plaintext.

## CI/CD

GitHub Actions (`.github/workflows/ci.yaml`) runs on pull requests and `main`: Go lint, format and tests; frontend codegen, check and build; and a migration up/down cycle against a disposable Postgres container. When all three pass, it builds one `linux/amd64` image. Pull requests push `tahminator/fin:staging-<short-sha>` and `sha-<sha>` and comment the tags on the PR; nothing is deployed. `main` pushes `<short-sha>` and `sha-<sha>`, then creates the next plain `X.Y.Z` Git tag (the first is `1.0.0`).

The tag push starts `.github/workflows/cd.yaml`. The tag points at a release commit whose parent is the `main` commit that was built, so CD promotes the parent SHA's image to `<version>` and `latest`, runs `just migrate` against `PRODUCTION_MIGRATION_DATABASE_URL`, then opens and merges the image-tag PR in `tahminator/k8s-personal`. Flux applies it; CI never touches the cluster, and a merged manifest update does not prove the app is healthy.

Secrets are repository-level: `DOCKER_HUB_PAT`, `_GITHUB_APP_APP_ID`, `_GITHUB_APP_INSTALLATION_ID` and `_GITHUB_APP_PEM_CONTENT` (shared), plus `PRODUCTION_MIGRATION_DATABASE_URL` for CD. Application runtime secrets belong in the SOPS-managed Kubernetes manifest, not CI.

`backendTest` installs golangci-lint through the pipeline setup action, and the migration test runs `LocalPostgresClient` with Postgres 19 because fin's migrations use `uuidv7()`. Both require `@tahminator/pipeline` 1.0.105 or later. See `PLAN.md` §12 for the full contract and prerequisites.
