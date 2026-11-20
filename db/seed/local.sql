\set ON_ERROR_STOP on

BEGIN;

INSERT INTO users (email, name)
VALUES (:'email', 'Local User')
ON CONFLICT (lower(email)) DO NOTHING;

INSERT INTO connections (
    user_id, provider, external_id, access_token_ciphertext,
    institution_name, status, last_synced_at
)
SELECT id, 'plaid', 'local-seed-' || id::text, 'local-seed-not-a-token',
       'Example Bank', 'needs_reauth', now()
FROM users
WHERE lower(email) = lower(:'email')
ON CONFLICT (provider, external_id) DO NOTHING;

INSERT INTO accounts (
    user_id, connection_id, external_id, name, official_name, mask,
    type, subtype, current_balance, available_balance, currency
)
SELECT u.id, c.id, sample.external_id, sample.name, sample.official_name, sample.mask,
       sample.type, sample.subtype, sample.current_balance, sample.available_balance, 'USD'
FROM users AS u
JOIN connections AS c ON c.user_id = u.id AND c.provider = 'plaid'
    AND c.external_id = 'local-seed-' || u.id::text
CROSS JOIN (VALUES
    ('local-checking', 'Checking', 'Example Bank Checking', '1234', 'depository', 'checking', 4825.35::numeric, 4720.35::numeric),
    ('local-savings', 'Savings', 'Example Bank Savings', '5678', 'depository', 'savings', 12500.00::numeric, 12500.00::numeric),
    ('local-credit', 'Credit Card', 'Example Bank Credit Card', '9012', 'credit', 'credit card', 764.82::numeric, NULL::numeric)
) AS sample(external_id, name, official_name, mask, type, subtype, current_balance, available_balance)
WHERE lower(u.email) = lower(:'email')
ON CONFLICT (connection_id, external_id) DO NOTHING;

INSERT INTO transactions (
    user_id, account_id, external_id, amount, currency, posted_on,
    authorized_on, name, merchant_name, category, pending
)
SELECT u.id, a.id, sample.external_id, sample.amount, 'USD',
       CURRENT_DATE - sample.days_ago, CURRENT_DATE - sample.days_ago,
       sample.name, sample.merchant_name, sample.category, sample.pending
FROM users AS u
JOIN connections AS c ON c.user_id = u.id AND c.provider = 'plaid'
    AND c.external_id = 'local-seed-' || u.id::text
JOIN accounts AS a ON a.connection_id = c.id
JOIN (VALUES
    ('local-checking', 'local-payroll', -3200.00::numeric, 3, 'Payroll Deposit', 'Example Employer', 'INCOME', false),
    ('local-checking', 'local-rent', 1500.00::numeric, 8, 'Monthly Rent', 'Example Property', 'RENT_AND_UTILITIES', false),
    ('local-checking', 'local-groceries', 87.42::numeric, 2, 'Grocery Store', 'Market', 'FOOD_AND_DRINK', false),
    ('local-checking', 'local-coffee', 5.75::numeric, 0, 'Coffee Shop', 'Coffee Shop', 'FOOD_AND_DRINK', true),
    ('local-credit', 'local-dinner', 64.30::numeric, 1, 'Restaurant', 'Restaurant', 'FOOD_AND_DRINK', false),
    ('local-credit', 'local-transit', 25.00::numeric, 5, 'Transit', 'Transit', 'TRANSPORTATION', false),
    ('local-credit', 'local-shopping', 120.19::numeric, 12, 'Online Purchase', 'Online Store', 'GENERAL_MERCHANDISE', false)
) AS sample(account_external_id, external_id, amount, days_ago, name, merchant_name, category, pending)
    ON sample.account_external_id = a.external_id
WHERE lower(u.email) = lower(:'email')
ON CONFLICT (account_id, external_id) DO NOTHING;

COMMIT;
