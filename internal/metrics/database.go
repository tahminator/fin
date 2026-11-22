package metrics

import (
	"context"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/tahminator/fin/internal/domain"
	"github.com/tahminator/fin/internal/repository"
)

const snapshotTimeout = 3 * time.Second

var (
	usersDesc                 = newDesc("fin_users", "Allow-listed users.")
	usersWithCredentialsDesc  = newDesc("fin_users_with_credentials", "Users with stored provider credentials.", "provider")
	activeSessionsDesc        = newDesc("fin_active_sessions", "Unexpired sessions per user.", "user")
	connectionsDesc           = newDesc("fin_connections", "Connections per user and status.", "user", "status")
	connectionsWithErrorsDesc = newDesc("fin_connections_with_errors", "Connections whose last sync failed, per user.", "user")
	accountsDesc              = newDesc("fin_accounts", "Accounts per user.", "user")
	transactionsDesc          = newDesc("fin_transactions", "Transactions per user.", "user")
	netWorthDesc              = newDesc("fin_net_worth", "Net worth per user and currency.", "user", "currency")
	oldestSyncDesc            = newDesc("fin_connection_oldest_sync_timestamp_seconds", "Unix time of the least recently synced connection per user.", "user")
	poolAcquiredDesc          = newDesc("fin_db_pool_acquired_conns", "Database connections currently in use.")
	poolIdleDesc              = newDesc("fin_db_pool_idle_conns", "Idle database connections.")
	poolTotalDesc             = newDesc("fin_db_pool_total_conns", "Total database connections.")
	snapshotUpDesc            = newDesc("fin_metrics_snapshot_up", "Whether the last database snapshot succeeded.")
)

type poolStater interface {
	Stat() *pgxpool.Stat
}

type databaseCollector struct {
	repo repository.MetricsRepository
	pool poolStater
}

func WatchDatabase(repo repository.MetricsRepository, pool poolStater) {
	registry.MustRegister(databaseCollector{repo: repo, pool: pool})
}

func (c databaseCollector) Describe(descs chan<- *prometheus.Desc) {
	for _, desc := range []*prometheus.Desc{
		usersDesc,
		usersWithCredentialsDesc,
		activeSessionsDesc,
		connectionsDesc,
		connectionsWithErrorsDesc,
		accountsDesc,
		transactionsDesc,
		netWorthDesc,
		oldestSyncDesc,
		poolAcquiredDesc,
		poolIdleDesc,
		poolTotalDesc,
		snapshotUpDesc,
	} {
		descs <- desc
	}
}

func (c databaseCollector) Collect(metrics chan<- prometheus.Metric) {
	stat := c.pool.Stat()

	metrics <- gauge(poolAcquiredDesc, float64(stat.AcquiredConns()))
	metrics <- gauge(poolIdleDesc, float64(stat.IdleConns()))
	metrics <- gauge(poolTotalDesc, float64(stat.TotalConns()))

	ctx, cancel := context.WithTimeout(context.Background(), snapshotTimeout)
	defer cancel()

	snapshot, err := c.repo.Snapshot(ctx)
	if err != nil {
		metrics <- gauge(snapshotUpDesc, 0)

		return
	}

	metrics <- gauge(snapshotUpDesc, 1)
	metrics <- gauge(usersDesc, float64(snapshot.Users))

	for _, provider := range allProviders() {
		count := snapshot.UsersWithCredentials[domainProviderOf(provider)]

		metrics <- gauge(usersWithCredentialsDesc, float64(count), provider.String())
	}

	for _, user := range snapshot.PerUser {
		collectUser(metrics, user)
	}
}

func collectUser(metrics chan<- prometheus.Metric, user domain.UserSnapshot) {
	metrics <- gauge(activeSessionsDesc, float64(user.ActiveSessions), user.Email)
	metrics <- gauge(connectionsWithErrorsDesc, float64(user.ConnectionsWithErrors), user.Email)
	metrics <- gauge(accountsDesc, float64(user.Accounts), user.Email)
	metrics <- gauge(transactionsDesc, float64(user.Transactions), user.Email)

	for currency, net := range user.NetWorthByCurrency {
		value, err := strconv.ParseFloat(net, 64)
		if err == nil {
			metrics <- gauge(netWorthDesc, value, user.Email, currency)
		}
	}

	byStatus := make(map[ConnectionStatus]int)

	for status, count := range user.ConnectionsByStatus {
		byStatus[connectionStatusOf(status)] += count
	}

	for _, status := range allConnectionStatuses() {
		metrics <- gauge(connectionsDesc, float64(byStatus[status]), user.Email, status.String())
	}

	if user.OldestSyncAt != nil {
		metrics <- gauge(oldestSyncDesc, unixSeconds(*user.OldestSyncAt), user.Email)
	}
}

func domainProviderOf(provider Provider) domain.Provider {
	if provider == ProviderPlaid {
		return domain.ProviderPlaid
	}

	return ""
}

func newDesc(name, help string, labels ...string) *prometheus.Desc {
	return prometheus.NewDesc(name, help, labels, nil)
}

func gauge(desc *prometheus.Desc, value float64, labels ...string) prometheus.Metric {
	return prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, value, labels...)
}
