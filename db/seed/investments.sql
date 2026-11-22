\set ON_ERROR_STOP on

BEGIN;

INSERT INTO users (email, name)
VALUES (:'email', 'Local User')
ON CONFLICT (lower(email)) DO NOTHING;

INSERT INTO connections (
    user_id, provider, external_id, access_token_ciphertext,
    institution_name, status, last_synced_at, investments_status
)
SELECT id, 'plaid', 'local-brokerage-' || id::text, 'local-seed-not-a-token',
       'Example Brokerage', 'active', now(), 'ready'
FROM users
WHERE lower(email) = lower(:'email')
ON CONFLICT (provider, external_id) DO NOTHING;

INSERT INTO accounts (
    user_id, connection_id, external_id, name, official_name, mask,
    type, subtype, current_balance, available_balance, currency
)
SELECT u.id, c.id, sample.external_id, sample.name, sample.name, sample.mask,
       'investment', sample.subtype, sample.balance, NULL, 'USD'
FROM users AS u
JOIN connections AS c ON c.user_id = u.id AND c.external_id = 'local-brokerage-' || u.id::text
CROSS JOIN (VALUES
    ('local-brokerage', 'Individual Brokerage', '4321', 'brokerage', 25840.00::numeric),
    ('local-roth', 'Roth IRA', '8765', 'roth', 14210.00::numeric)
) AS sample(external_id, name, mask, subtype, balance)
WHERE lower(u.email) = lower(:'email')
ON CONFLICT (connection_id, external_id) DO NOTHING;

INSERT INTO investment_securities (user_id, external_id, name, ticker_symbol, type, currency)
SELECT u.id, sample.external_id, sample.name, sample.ticker, sample.type, 'USD'
FROM users AS u
CROSS JOIN (VALUES
    ('local-vti', 'Vanguard Total Stock Market ETF', 'VTI', 'etf'),
    ('local-aapl', 'Apple Inc.', 'AAPL', 'equity'),
    ('local-fxaix', 'Fidelity 500 Index Fund', 'FXAIX', 'mutual fund'),
    ('local-cash', 'U S Dollar', NULL, 'cash'),
    ('local-sold', 'Example Growth Fund', 'EXGF', 'mutual fund')
) AS sample(external_id, name, ticker, type)
WHERE lower(u.email) = lower(:'email')
ON CONFLICT (user_id, external_id) DO NOTHING;

INSERT INTO investment_holdings (
    user_id, account_id, security_id, institution_price, institution_value, quantity, cost_basis, currency
)
SELECT a.user_id, a.id, s.id, sample.price, sample.price * sample.quantity, sample.quantity, sample.cost, 'USD'
FROM accounts AS a
JOIN (VALUES
    ('local-brokerage', 'local-vti', 280.00::numeric, 50::numeric, 11000.00::numeric),
    ('local-brokerage', 'local-aapl', 230.00::numeric, 40::numeric, 6400.00::numeric),
    ('local-brokerage', 'local-cash', 1.00::numeric, 1640::numeric, 1640.00::numeric),
    ('local-roth', 'local-fxaix', 205.00::numeric, 60::numeric, 9800.00::numeric),
    ('local-roth', 'local-vti', 280.00::numeric, 6::numeric, 1500.00::numeric)
) AS sample(account_external_id, security_external_id, price, quantity, cost)
    ON sample.account_external_id = a.external_id
JOIN investment_securities AS s ON s.user_id = a.user_id AND s.external_id = sample.security_external_id
JOIN users AS u ON u.id = a.user_id
WHERE lower(u.email) = lower(:'email')
ON CONFLICT (account_id, security_id) DO NOTHING;

UPDATE accounts AS a
SET current_balance = held.value
FROM (
    SELECT h.account_id, sum(h.institution_value) AS value
    FROM investment_holdings AS h
    GROUP BY h.account_id
) AS held
JOIN users AS u ON true
WHERE held.account_id = a.id
  AND a.user_id = u.id
  AND lower(u.email) = lower(:'email')
  AND a.external_id IN ('local-brokerage', 'local-roth');

INSERT INTO investment_transactions (
    user_id, account_id, security_id, external_id, posted_on, name,
    quantity, amount, price, fees, type, subtype, currency
)
SELECT a.user_id, a.id, s.id, sample.external_id, CURRENT_DATE - sample.days_ago, sample.name,
       sample.quantity, sample.amount, sample.price, sample.fees, sample.type, sample.subtype, 'USD'
FROM accounts AS a
JOIN (VALUES
    ('local-brokerage', 'local-sold', 'local-itx-sell', 20, 'SELL Example Growth Fund', -30::numeric, 3150.00::numeric, 105.00::numeric, 4.95::numeric, 'sell', 'sell'),
    ('local-brokerage', 'local-aapl', 'local-itx-buy', 45, 'BUY Apple Inc.', 10::numeric, -2250.00::numeric, 225.00::numeric, 0::numeric, 'buy', 'buy'),
    ('local-brokerage', NULL, 'local-itx-deposit', 60, 'ACH deposit', 0::numeric, 2000.00::numeric, 0::numeric, NULL, 'cash', 'deposit'),
    ('local-roth', 'local-fxaix', 'local-itx-dividend', 10, 'Dividend Fidelity 500 Index Fund', 0::numeric, 42.18::numeric, 0::numeric, NULL, 'cash', 'dividend'),
    ('local-roth', NULL, 'local-itx-fee', 30, 'Account maintenance fee', 0::numeric, -25.00::numeric, 0::numeric, 25.00::numeric, 'fee', 'account fee')
) AS sample(account_external_id, security_external_id, external_id, days_ago, name, quantity, amount, price, fees, type, subtype)
    ON sample.account_external_id = a.external_id
LEFT JOIN investment_securities AS s ON s.user_id = a.user_id AND s.external_id = sample.security_external_id
JOIN users AS u ON u.id = a.user_id
WHERE lower(u.email) = lower(:'email')
ON CONFLICT (account_id, external_id) DO NOTHING;

INSERT INTO investment_account_snapshots (user_id, account_id, captured_at, value, currency)
SELECT a.user_id, a.id, captured.at,
       round(a.current_balance * (0.9 + 0.1 * (captured.step / 480.0) + 0.02 * sin(captured.step / 7.0)::numeric), 4),
       'USD'
FROM accounts AS a
JOIN users AS u ON u.id = a.user_id
CROSS JOIN LATERAL (
    SELECT date_trunc('minute', now()) - make_interval(hours => 6 * (480 - step)) AS at, step
    FROM generate_series(0, 480) AS step
) AS captured
WHERE lower(u.email) = lower(:'email')
  AND a.external_id IN ('local-brokerage', 'local-roth')
ON CONFLICT (account_id, captured_at) DO NOTHING;

INSERT INTO investment_holding_snapshots (
    snapshot_id, user_id, account_id, security_id, captured_at,
    quantity, institution_price, institution_value, currency
)
SELECT snapshot.id, h.user_id, h.account_id, h.security_id, snapshot.captured_at,
       h.quantity,
       round(h.institution_price * snapshot.value / a.current_balance, 4),
       round(h.institution_value * snapshot.value / a.current_balance, 4),
       h.currency
FROM investment_account_snapshots AS snapshot
JOIN accounts AS a ON a.id = snapshot.account_id
JOIN investment_holdings AS h ON h.account_id = snapshot.account_id
JOIN users AS u ON u.id = a.user_id
WHERE lower(u.email) = lower(:'email')
ON CONFLICT (snapshot_id, security_id) DO NOTHING;

COMMIT;
