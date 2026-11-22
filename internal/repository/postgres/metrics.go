package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/tahminator/fin/internal/domain"
)

type MetricsRepository struct {
	pool *pgxpool.Pool
}

func NewMetricsRepository(pool *pgxpool.Pool) *MetricsRepository {
	return &MetricsRepository{pool: pool}
}

const countAllUsers = `
SELECT COUNT(*) AS count
FROM users;
`

const usersWithCredentialsByProvider = `
SELECT
    c.provider,
    COUNT(DISTINCT c.user_id) AS users
FROM user_provider_credentials AS c
GROUP BY c.provider;
`

const perUserSnapshot = `
SELECT
    u.email,
    (
        SELECT COUNT(*)
        FROM sessions AS s
        WHERE s.user_id = u.id
          AND s.expires_at > now()
    ) AS active_sessions,
    (
        SELECT COUNT(*)
        FROM connections AS c
        WHERE c.user_id = u.id
          AND c.last_error IS NOT NULL
    ) AS connections_with_errors,
    (
        SELECT COUNT(*)
        FROM accounts AS a
        WHERE a.user_id = u.id
    ) AS accounts,
    (
        SELECT COUNT(*)
        FROM transactions AS t
        WHERE t.user_id = u.id
    ) AS transactions,
    (
        SELECT MIN(c.last_synced_at)
        FROM connections AS c
        WHERE c.user_id = u.id
    ) AS oldest_sync_at
FROM users AS u
ORDER BY u.email;
`

const connectionsByUserAndStatus = `
SELECT
    u.email,
    c.status,
    COUNT(*) AS connections
FROM connections AS c
JOIN users AS u ON u.id = c.user_id
GROUP BY u.email, c.status;
`

const netWorthByUserAndCurrency = `
WITH totals AS (
    SELECT
        u.email,
        COALESCE(a.currency, 'XXX') AS currency,
        COALESCE(SUM(a.current_balance) FILTER (WHERE a.type NOT IN ('credit', 'loan')), 0) AS assets,
        COALESCE(SUM(a.current_balance) FILTER (WHERE a.type IN ('credit', 'loan')), 0)     AS liabilities
    FROM users AS u
    LEFT JOIN accounts AS a ON a.user_id = u.id
    WHERE a.current_balance IS NOT NULL
    GROUP BY u.email, COALESCE(a.currency, 'XXX')
)
SELECT
    t.email,
    t.currency,
    (t.assets - t.liabilities)::text AS net
FROM totals AS t
ORDER BY t.email, t.currency;
`

type providerCount struct {
	Provider domain.Provider `db:"provider"`
	Users    int             `db:"users"`
}

type statusCount struct {
	Email       string                  `db:"email"`
	Status      domain.ConnectionStatus `db:"status"`
	Connections int                     `db:"connections"`
}

type netWorthCount struct {
	Email    string `db:"email"`
	Currency string `db:"currency"`
	Net      string `db:"net"`
}

func (r *MetricsRepository) Snapshot(ctx context.Context) (domain.MetricsSnapshot, error) {
	users, err := count(ctx, r.pool, countAllUsers)
	if err != nil {
		return domain.MetricsSnapshot{}, err
	}

	providers, err := collect[providerCount](ctx, r.pool, usersWithCredentialsByProvider)
	if err != nil {
		return domain.MetricsSnapshot{}, err
	}

	perUser, err := collect[domain.UserSnapshot](ctx, r.pool, perUserSnapshot)
	if err != nil {
		return domain.MetricsSnapshot{}, err
	}

	statuses, err := collect[statusCount](ctx, r.pool, connectionsByUserAndStatus)
	if err != nil {
		return domain.MetricsSnapshot{}, err
	}

	netWorth, err := collect[netWorthCount](ctx, r.pool, netWorthByUserAndCurrency)
	if err != nil {
		return domain.MetricsSnapshot{}, err
	}

	snapshot := domain.MetricsSnapshot{
		Users:                users,
		UsersWithCredentials: make(map[domain.Provider]int, len(providers)),
		PerUser:              perUser,
	}

	for _, provider := range providers {
		snapshot.UsersWithCredentials[provider.Provider] = provider.Users
	}

	attachStatuses(snapshot.PerUser, statuses)
	attachNetWorth(snapshot.PerUser, netWorth)

	return snapshot, nil
}

func attachStatuses(perUser []domain.UserSnapshot, statuses []statusCount) {
	byEmail := make(map[string]map[domain.ConnectionStatus]int)

	for _, status := range statuses {
		if byEmail[status.Email] == nil {
			byEmail[status.Email] = make(map[domain.ConnectionStatus]int)
		}

		byEmail[status.Email][status.Status] = status.Connections
	}

	for i := range perUser {
		perUser[i].ConnectionsByStatus = byEmail[perUser[i].Email]
	}
}

func attachNetWorth(perUser []domain.UserSnapshot, netWorth []netWorthCount) {
	byEmail := make(map[string]map[string]string)

	for _, value := range netWorth {
		if byEmail[value.Email] == nil {
			byEmail[value.Email] = make(map[string]string)
		}

		byEmail[value.Email][value.Currency] = value.Net
	}

	for i := range perUser {
		perUser[i].NetWorthByCurrency = byEmail[perUser[i].Email]
	}
}

func collect[T any](ctx context.Context, pool *pgxpool.Pool, query string) ([]T, error) {
	rows, err := pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query snapshot: %w", err)
	}

	values, err := pgx.CollectRows(rows, pgx.RowToStructByName[T])
	if err != nil {
		return nil, fmt.Errorf("collect snapshot: %w", err)
	}

	return values, nil
}
