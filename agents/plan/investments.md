# fin — Investments Plan

Investment tracking adds brokerage and retirement data from Plaid's Investments product: current holdings, two years of investment transactions, and a locally recorded history of portfolio value. It also splits the Home page into bank and investment sections, adds an `/investments` page, and shows holdings and investment activity on the Transactions page.

## 0. Changes to earlier plans

- [init.md](init.md) §4 Database: adds migrations `000014`–`000019` (§2 below) and the `connections.investments_status` column.
- init.md §5.4 Provider abstraction: adds `CreateInvestmentsConsentLinkToken`, `InvestmentHoldings` and `InvestmentTransactions`. New Link tokens request `investments` as an optional product.
- init.md §5.5 Services: `ConnectionService.Sync` runs the investment sync (§3) after regular transactions and before the cursor is saved. Adds `CreateInvestmentsConsentLinkToken`, `CompleteInvestmentsConsent` and `InvestmentHistoryService`.
- init.md §5.6 HTTP surface: adds the endpoints in §5 and `investmentsStatus` on `Connection`. `/api/transactions` takes the shared `accountId`/`excludeAccountId` parameters through `AccountFilterParams`.
- init.md §5.8 Observability: adds the `investments_consent` sync trigger and the `investment_holdings` and `investment_transactions` provider operations.
- init.md §6.2 Routes: adds `/investments` (`src/app/investments/investments.Page.tsx`) and a fourth tab.
- init.md §8 Justfile: adds `test-db` (Postgres integration tests behind the `integration` build tag) and `seed-dev-investments`.
- init.md §12.6 `ci.yaml`: the migrations job installs Go and runs `just test-db` against the freshly migrated database.

## 1. Plaid product setup

- New Items: `/link/token/create` keeps `products: ["transactions"]` and adds `optional_products: ["investments"]`. Optional products do not limit the institution list. Plaid fetches them on a best-effort basis, and an institution without Investments still links.
- Existing Items created with Transactions only: the Connections page shows **Share investments** when `investmentsStatus` is `consent_required`. It opens update-mode Link with `additional_consented_products: ["investments"]` (`POST /api/connections/{id}/investments/link-token`). On success the app calls `POST /api/connections/{id}/investments/complete`, which runs a sync with trigger `investments_consent`. That sync's first call to an Investments endpoint adds the product to the Item.
- In Production each Plaid client needs the Investments product enabled in the Plaid Dashboard. Without it, calls fail with `INVALID_PRODUCT` and the connection shows `not_enabled`, with instructions.
- Investments is a subscription product: it is billed per Item while added, and it cannot be removed from an Item except by removing the Item.

## 2. Tables and retention

Every table carries `user_id`, and every query filters on it. Money and quantities are `numeric(19,4)`, selected as text, and summed only in SQL.

| Migration | Table | Purpose |
| --- | --- | --- |
| `000014` | `investment_securities` | Per-user catalog keyed by `(user_id, external_id)`. An upsert never overwrites name, ticker, type, subtype or currency with `NULL`, so a sold security keeps its metadata. |
| `000015` | `investment_holdings` | Latest snapshot per `(account_id, security_id)`. Each sync replaces the rows of the connection's investment accounts. Duplicate lots of one security in one account are summed. |
| `000016` | `investment_transactions` | Plaid investment transactions, unique per `(account_id, external_id)`. |
| `000017` | `connections.investments_status` | `ready`, `pending`, `consent_required`, `not_enabled`, `unsupported`, or `NULL` when the connection has no investment accounts. |
| `000018` | `investment_account_snapshots` | One row per investment account per capture: `captured_at` and `value`. The value is the account's current balance, or the sum of its holdings when the balance is missing. Unique per `(account_id, captured_at)`. |
| `000019` | `investment_holding_snapshots` | The holdings belonging to an account snapshot (`snapshot_id` cascades), with quantity, price, value and currency. |

Retention:

- **Transactions:** a rolling two-year window. Each sync asks Plaid for `[now − 2 years, now]`. Only after every page has been fetched does the repository upsert the result and delete the connection's investment transactions that are dated before the window or missing from the fetch. A failed or partial fetch deletes nothing.
- **Snapshots:** all rows from one sync share one `captured_at`: the sync time truncated to the second, in UTC. Captures are idempotent: repeating a capture at the same time updates it in place. A capture within 15 minutes of the previous one replaces it, so overlapping or rapid syncs keep only the latest capture. After 90 days, snapshots are thinned to the last capture of each UTC day. Nothing older is deleted, so history grows by about one row per account per day.
- Removing a connection cascades through accounts to holdings, transactions and snapshots.

## 3. Sync sequence and failure semantics

`ConnectionService.pull` runs:

1. Fetch accounts, upsert, match card art, then sync regular transactions (unchanged).
2. If no account of the Item is `investment`: set `investments_status` to `NULL` and skip to step 9.
3. `InvestmentHoldings`, then upsert the returned securities.
4. Replace holdings for the connection's investment accounts. Holdings of unknown accounts are dropped.
5. Capture a snapshot of those accounts at the sync's capture time.
6. `InvestmentTransactions` for the two-year window. Every page is fetched, and the securities returned with each page are kept.
7. Upsert those securities, then reconcile transactions as described in §2.
8. Set `investments_status = ready`.
9. Save the cursor and `last_synced_at` (always last).

Plaid errors are mapped at the provider boundary:

| Plaid `error_code` | Status | Effect |
| --- | --- | --- |
| `ADDITIONAL_CONSENT_REQUIRED` | `consent_required` | Regular sync still succeeds. Connections offers **Share investments**. |
| `PRODUCT_NOT_READY` | `pending` | Regular sync succeeds. Holdings and snapshots are kept if they were already saved, and transactions are retried on the next sync. |
| `INVALID_PRODUCT` | `not_enabled` | Regular sync succeeds. Connections explains how to enable the product in the Plaid Dashboard. |
| `PRODUCTS_NOT_SUPPORTED`, `NO_INVESTMENT_ACCOUNTS`, `NO_INVESTMENT_AUTH_ACCOUNTS` | `unsupported` | Regular sync succeeds. Connections notes that the institution does not share investment data. |
| `ITEM_LOGIN_REQUIRED` | — | `ErrReauthRequired`, as in init.md §5.12. |
| anything else | unchanged | The sync fails: `last_error` is set, the cursor is not saved, the cooldown is released and the error metric is recorded, so the failure is visible and retried. Regular transaction upserts are idempotent, so retrying is harmless. |

The status is written only when it changes. Of these codes, only `PRODUCT_NOT_READY`, `INVALID_PRODUCT` and `PRODUCTS_NOT_SUPPORTED` appear in the Plaid Go SDK v50 documentation. The others follow Plaid's error reference and have not yet been checked against a live Item. Confirm them in Plaid Sandbox, and extend the table if Plaid reports others. An unknown code is treated as a visible, retryable failure, so a missing entry never silently drops data.

Amounts are mapped at the provider boundary: `amount` is negated, because Plaid uses positive for cash debited, while fin uses negative for outflows. A purchase is stored as negative and a sale or dividend as positive. `quantity` keeps Plaid's sign (positive for buys, negative for sells). `fees` is the reported positive fee total, which is already included in `amount`. Plaid's legacy `cancel_transaction_id` is not stored.

## 4. Historical data limitations

- Plaid returns current holdings, not historical prices. fin's history consists of the snapshots it has captured itself. It never synthesizes or interpolates points, and it never backfills from investment transactions, because splits, transfers, corporate actions and missing history make that unreliable.
- A new connection starts with one point. A range whose start precedes the first snapshot, or that has fewer than two points, is marked `limited`, and the UI says "History will build as your accounts sync."
- How dense the points are depends on how often syncs run (`SYNC_CRON`, every 6 hours by default, plus manual refreshes). A `1D` range may have only a few points. The UI does not imply intraday market precision.
- The chart is labelled **Portfolio value**. Deposits and withdrawals change it, so it is not investment performance or return. Measuring return, or backfilling history, would require a separate market-data provider and is out of scope.
- An aggregate point sums each account's latest snapshot at or before that point. An account whose syncs stop contributes its last captured value until it syncs again or is removed.

## 5. API contracts

All endpoints require a session and accept `accountId` and repeated `excludeAccountId` (`AccountFilterParams`), applied in SQL. Arrays are never `null`.

| Method | Path | Response |
| --- | --- | --- |
| GET | `/api/investments/holdings` | `InvestmentHolding[]`, highest value first, with security name, ticker and type. |
| GET | `/api/investments/transactions` | `Page[InvestmentTransaction]` (`page`, `pageSize`), newest first. |
| GET | `/api/investments/securities` | `InvestmentSecurity[]`: securities with snapshot history in the filtered accounts, each flagged `held` if currently held. |
| GET | `/api/investments/history` | `InvestmentHistory` (below). `range` is required (`1d`,`3d`,`1w`,`1m`,`3m`,`1y`,`all`). Repeated `securityId` values chart only those securities; with none, the chart uses whole account values, cash included. |
| POST | `/api/connections/{id}/investments/link-token` | `{linkToken}`. 409 unless the connection is `active`. |
| POST | `/api/connections/{id}/investments/complete` | `Connection` after the sync. |

`InvestmentHistory`:

```json
{
  "range": "1m",
  "from": "2026-10-22T09:30:00Z",
  "earliestCapturedAt": "2026-07-25T09:39:00Z",
  "latestCapturedAt": "2026-11-22T09:39:00Z",
  "total": [
    {
      "currency": "USD",
      "points": [{ "at": "2026-10-22T15:39:00Z", "value": "38430.4200" }],
      "first": "38430.4200",
      "current": "38418.4200",
      "change": "-12.0000",
      "changePercent": "-0.03",
      "limited": false
    }
  ],
  "accounts": [{ "accountId": "…", "series": [{ "currency": "USD", "points": [] }] }]
}
```

- One series per currency. Currencies are never added together, and an empty `currency` means the institution reported none.
- Points are ordered by time. Each `at` is a real capture time: the latest capture in its bucket.
- Bucket sizes: `1d` 15 min, `3d` 1 h, `1w` 3 h, `1m` 12 h, `3m` 1 day, `1y` 7 days. For `all`, the span since the first snapshot is divided into at most 180 buckets of whole days, at least one day each. Bucketing uses `date_bin` in SQL.
- `change` is `current − first`. `changePercent` is `change / |first| × 100`, rounded to two decimals, or `null` when `first` is zero. Both are computed exactly with `math/big`.
- The per-account series in `accounts` come from the same query, so the page makes one history request rather than one per account.

## 6. Frontend

Query hooks live in `lib/queries/investments.ts`. Query keys include range, account filters and selected securities. History uses `keepPreviousData`, and every sync/link mutation invalidates the investment queries. Decimal sums for display use `lib/decimal.ts` (scaled `bigint`), never floating-point arithmetic. `Number()` is used only for chart coordinates and quantity formatting.

### Home

- The overall summary (net worth, assets, liabilities) still includes every visible account, investment accounts included.
- **Accounts** (shown when the user has non-investment accounts): the cash-flow chart excludes hidden accounts and all investment accounts, followed by the list of non-investment accounts with eye toggles.
- **Investments** (shown when the user has investment accounts): a one-month portfolio-value card per currency, a list of investment accounts with eye toggles, and a link to `/investments`. One snapshot shows the current value with the limited-history note, no snapshots shows a "starts with the next sync" message, and hiding every investment account shows "All investment accounts are hidden."
- One `useState` in `AccountsOverview` holds the hidden account IDs, which drive the summary, cash flow and portfolio queries.

### `/investments`

- URL state: `range` (default `1m`) and repeated `off` (hidden security IDs), validated with Zod. An invalid URL shows a reset action.
- A range selector (`1D`…`ALL`) appears above the charts.
- Portfolio card: per currency, the current value, change and percent for the range, and an area chart colored by direction (end versus start), with a tooltip showing time and value. The label is "Portfolio value" when every holding is selected. Otherwise it reads "Selected holdings value", with a note that cash and unselected holdings are excluded.
- Holdings in the chart: one checkbox row per security (stocks, ETFs, funds, cash and others alike), showing ticker and name or a stable fallback label, type, current quantity, current value per currency and owning accounts, plus **Show all** and **Hide all**. Toggling refetches only the history query.
- Accounts: per investment account, its identity, balance, a value chart from the shared history response, current holdings, and a link to `/transactions?accountId=…`.
- Each section handles its own loading, error and retry state, so one failed query does not blank the page.

### Transactions

The page has three sections, each with its own loading, error and retry state: holdings, investment activity (paginated independently), and bank and card transactions. All three honor `accountId` and `hide`. The page says "No transactions yet" only when all three are empty, or "Nothing to show" when accounts are hidden. Activity rows show ticker and security name, falling back to the transaction name, then date, type, account, fees when non-zero, and the signed amount.

### Connections

When `investmentsStatus` is not `ready`, a row explains the state, and **Share investments** starts the consent flow for `consent_required`. The Plaid launcher stores the flow as `investments-consent` so OAuth institutions can resume it.

## 7. Verification

- `just test`: provider mapping (amount signs, fees, error classification), sync orchestration (order, account scoping, statuses, retention window, failure semantics), handler filters, pagination and errors, and history and series calculations.
- `just migrate && just test-db`: real Postgres round trip of securities, holdings, transactions, snapshot idempotency and thinning, filters, history aggregation and the connection status.
- `just seed-dev-investments <email>` adds fictional brokerage data with 120 days of snapshots for manual UI checks.
- With Plaid Sandbox credentials, link `ins_109508` (First Platypus Bank) as `user_good` and check holdings, activity and a first snapshot. Then link an Item created without Investments consent and use **Share investments**.
