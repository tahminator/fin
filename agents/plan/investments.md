# fin — Investments plan

Investment tracking for fin: Plaid investment holdings and trades, live and historical prices from Alpaca, and an Investments tab with a Robinhood-style portfolio chart. The base application is planned in [init.md](init.md); "init.md §N" points there. Conventions from [STYLEGUIDE.md](../../STYLEGUIDE.md) apply unchanged.

This plan changes these sections of earlier plans:

- init.md §4 (Database): migrations `000014`–`000016` add investment tables (summarised there, specified here).
- init.md §5.4 (Provider abstraction): `provider.Provider` gains `InvestmentHoldings` and `InvestmentTransactions`; `CreateUpdateLinkToken` gains `consentInvestments`.
- init.md §5.5 (Services): connection sync pulls investments after the regular-transaction cursor is saved.
- init.md §5.6 (HTTP surface): `excludeType` / `excludeAccountType` filters and the `/api/investments/*` endpoints below.
- init.md §5.8 (Observability): provider operations `investment_holdings` and `investment_transactions`.
- init.md §6.2 (Routes): a fourth tab, `/investments`.
- init.md §8 (Justfile): `seed-alpaca`.

## 1. Decisions

| Topic            | Decision                                                                                                                                                                                                                                                                 | Why                                                                                                                                       |
| ---------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ----------------------------------------------------------------------------------------------------------------------------------------- |
| What you own     | Plaid Investments: holdings snapshot (quantity, institution price, cost basis) and up to 24 months of trades                                                                                                                                                             | Already linked; positions only change when you trade                                                                                     |
| What it is worth | Alpaca Market Data, free Basic plan: historical bars with `feed=sip` ending 16 minutes ago, latest trade with `feed=iex`                                                                                                                                                | Plaid has no price history and refreshes holdings about daily. Alpaca's free plan allows ~200 calls/min and history back to 2016          |
| Alpaca key       | Per user, stored in `user_provider_credentials` with `provider = 'alpaca'` (`client_id` = key ID, `secret` = secret key), added with `just seed-alpaca email key_id secret_key`                                                                                         | Same storage as Plaid credentials; each user brings their own rate limit                                                                 |
| No snapshots     | History is rebuilt on request from current holdings, trades and bars; nothing is stored per sync                                                                                                                                                                       | Accurate at any resolution and available from day one, instead of filling in at sync frequency                                          |
| Gain             | A position's change over a range is `end value − start value + Σ trade amounts in range` (amounts use fin's sign, so buys add back the cash spent and sells subtract proceeds). Percent is gain ÷ (start value + purchases in range). Cash positions never gain or lose | Buying or selling is not performance; dividends recorded against a security count as gain                                                |
| Cash             | A holding with security type `cash` is rebuilt backward from the account's trade amounts, so moving cash into a stock does not move the total                                                                                                                         | Keeps the chart flat across purchases funded from cash in the account                                                                     |
| Currency         | USD only (Alpaca); holdings and trades in other currencies are left out                                                                                                                                                                                                 | Never add amounts in different currencies                                                                                                |
| Arithmetic       | `math/big.Rat` from decimal strings; values rounded to cents only when formatted                                                                                                                                                                                       | No floating point for money (STYLEGUIDE)                                                                                                 |
| Cash views       | Home and the Transactions page exclude accounts of type `investment`; the Connections page and MCP tools still see every account                                                                                                                                       | Investments are not liquid and should not count towards cash, net worth on Home, or cash flow                                            |

## 2. Data

Migrations `000014_create_investment_securities`, `000015_create_investment_holdings`, `000016_create_investment_transactions`:

- `investment_securities`: one row per Plaid security ID (name, ticker, type, currency). Upserts keep existing values when Plaid sends nulls.
- `investment_holdings`: per-account snapshot, replaced on every sync (delete then insert; no database transaction, per STYLEGUIDE). User-tenanted.
- `investment_transactions`: Plaid's investment trades, upserted on `(account_id, external_id)`. Quantity is positive for buys and negative for sells; `amount` is negated at the provider boundary so money leaving the account is negative.

## 3. Sync

- New Links request `transactions` (required) and `investments` (optional), so banks without investments still appear in Link.
- After a connection's regular transactions and cursor are saved, Items with an investment account fetch holdings and two years of investment transactions (securities from both responses are kept, deduplicated).
- `ADDITIONAL_CONSENT_REQUIRED` maps to `ErrReauthRequired`, so the connection shows as needing reconnection; its update-mode Link token adds `investments` to `additional_consented_products` only when the connection has investment accounts.
- `NO_INVESTMENT_ACCOUNTS` and `PRODUCTS_NOT_SUPPORTED` map to `ErrUnsupported` and skip investments without failing the sync.

## 4. Performance

`PortfolioService.Performance(userID, range, filter{accountId, securityId})`:

| Range | Window             | Bars   |
| ----- | ------------------ | ------ |
| `1h`  | last hour          | 1Min   |
| `1d`  | last 24 hours      | 5Min   |
| `3d`  | last 72 hours      | 15Min  |
| `7d`  | last 7 days        | 1Hour  |
| `1mo` | last month         | 1Hour  |
| `1y`  | last year          | 1Day   |
| `all` | since first trade  | 1Day   |

1. Load holdings and the last two years of investment trades. Build one position per (account, security), including securities sold since the start of the range.
2. Quantity at time `t` = current quantity − quantities of trades dated after `t`'s day (clamped at zero). Trades carry a date only, so a trade counts from the start of its day.
3. Fetch bars for every equity or ETF ticker from the range start minus a lookback (4 days intraday, 10 days daily) so each range opens on a real price, plus the latest trade for the final point.
4. For `1h` and `1d`, when the market is closed and the window has no bars, the window shifts to end at the last bar (the most recent session), as Robinhood does.
5. Series times are the range start, every bar time in the window and an end point: the latest trade when it is newer than the last bar (normally a moment ago while the market is open), otherwise the last bar, so a closed market ends on its last session rather than on today. Without market data the end point is now. A position's price at `t` is the last close at or before `t`; holdings without market data (cash, mutual funds, bonds, crypto) use the brokerage's price throughout.
6. Totals are summed per point, per security and per account, with gain as in §1.

Without an Alpaca key the response is `marketData: not_configured` and every value uses the brokerage's price, so only quantity changes move the chart. If Alpaca fails, `marketData: unavailable` with the same fallback; the failure is logged, not returned.

Known limits: brokerage accounts that keep cash outside a `cash` holding still show a step when shares are bought (the gain figure is unaffected); intraday charts trail the market by about 15 minutes except for the final point; IEX latest trades can differ slightly from the consolidated price for thinly traded tickers.

## 5. HTTP surface

| Endpoint                             | Notes                                                                                                         |
| ------------------------------------ | ------------------------------------------------------------------------------------------------------------- |
| `GET /api/investments/performance`   | `range` (enum, default `1d`), `accountId`, `securityId`. Returns points, start/end values, change, positions (with cost basis, average cost, total return), accounts, `marketData` |
| `GET /api/investments/transactions`  | Paged; `accountId`, `securityId`, `excludeAccountId`                                                          |
| `GET /api/investments/transactions/{transactionId}` | Full investment activity detail; user-tenanted |
| `GET /api/investments/holdings`      | Raw holdings; `accountId`, `excludeAccountId`                                                                  |
| `GET /api/accounts`                  | `excludeType` (account types)                                                                                  |
| `GET /api/summary`, `/summary/cash-flow`, `/transactions` | `excludeAccountType` (account types)                                                       |

## 6. Frontend

- Tab bar: Home, Transactions, Investments, Connections.
- `/investments?range=`: summary card (value, change for the range coloured by direction with an arrow icon, area chart with a dashed baseline at the start value and a crosshair tooltip, range picker 1H 1D 3D 1W 1M 1Y ALL), holdings, accounts and activity. A notice explains when market prices are off or unavailable. The range lives in the URL.
- Detail pages are nested routes that stack like an iOS navigation controller: `/investments/accounts/:accountId`, `/investments/holdings/:securityId` and `/investments/accounts/:accountId/holdings/:securityId`. Each renders a fixed panel over its parent (which stays mounted underneath) that slides in from the right, has a back button in its top bar, and follows a swipe that starts within 32px of the left edge; releasing past a third of the width goes back, otherwise it springs back. Back uses history when there is any, else the parent link (deep links). Reduced motion skips the animations. Drill-down links pass the title in history state and use `preventScrollReset` so the page underneath keeps its scroll position.
  - Account: value and chart for the account, its holdings, its activity.
  - Holding: chart, shares, price, market value, average cost and total return, then that holding's activity.
- The tab bar sits above detail panels.
- Home and Transactions request `excludeType=investment` / `excludeAccountType=investment`.

## 7. Operations

- Add each user's key in production with `just seed-alpaca email key_id secret_key` against the production `DATABASE_URL`.
- The production network must reach `data.alpaca.markets`.
- Moving to a paid market data plan only changes the `feed` choices in `internal/marketdata/alpaca.go`; another provider implements `marketdata.Provider`.
