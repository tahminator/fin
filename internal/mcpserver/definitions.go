package mcpserver

import (
	"sync"

	"github.com/tahminator/fin/internal/domain"
	"github.com/tahminator/fin/internal/metrics"
)

var Definitions = sync.OnceValue(func() map[metrics.MCPTool]ToolDefinition {
	read := domain.ScopeRead

	return map[metrics.MCPTool]ToolDefinition{
		metrics.ToolListConnections: define[ListConnectionsInput, ListConnectionsOutput](
			metrics.ToolListConnections,
			"List bank connections",
			"Lists the person's linked banks with their status, when each last synced, and how many accounts each has. Check this to judge how fresh the data is.",
			read, nil,
		),
		metrics.ToolListAccounts: define[ListAccountsInput, ListAccountsOutput](
			metrics.ToolListAccounts,
			"List accounts",
			"Lists accounts with their type, mask and current and available balances.",
			read, map[string][]any{"account_types": filterEnums["account_types"]},
		),
		metrics.ToolGetBalances: define[GetBalancesInput, GetBalancesOutput](
			metrics.ToolGetBalances,
			"Get balances and net worth",
			"Totals assets (depository, investment, other), liabilities (credit, loan) and net worth, per currency.",
			read, nil,
		),
		metrics.ToolSearchTransactions: define[SearchTransactionsInput, SearchTransactionsOutput](
			metrics.ToolSearchTransactions,
			"Search transactions",
			"Finds transactions matching filters, with paging and sorting. Use aggregate_transactions instead when you need totals.",
			read, withFilterEnums(map[string][]any{"sort": strings2any("date_desc", "date_asc", "amount_asc", "amount_desc")}),
		),
		metrics.ToolGetTransaction: define[GetTransactionInput, TransactionOutput](
			metrics.ToolGetTransaction,
			"Get a transaction",
			"Returns one transaction by ID.",
			read, nil,
		),
		metrics.ToolListFacets: define[ListFacetsInput, ListFacetsOutput](
			metrics.ToolListFacets,
			"List filter values",
			"Lists the categories, merchants, accounts or currencies present in matching transactions, most common first, with counts and net sums. Use it to find valid values for other tools' filters.",
			read, withFilterEnums(map[string][]any{"facet": strings2any("category", "merchant", "account", "currency")}),
		),
		metrics.ToolAggregateTransactions: define[AggregateTransactionsInput, AggregateTransactionsOutput](
			metrics.ToolAggregateTransactions,
			"Aggregate transactions",
			"Groups matching transactions by up to three dimensions (time buckets, category, merchant, account, bank, type, direction, pending) and computes sums, counts, averages, medians, minimums, maximums, money in and money out. Examples: spending by category per month, top merchants this year, average spend per weekday.",
			read, withFilterEnums(map[string][]any{
				"group_by": strings2any(dimensionNames...),
				"metrics":  strings2any(metricNames...),
				"order":    strings2any("asc", "desc"),
			}),
		),
		metrics.ToolCashFlow: define[CashFlowInput, CashFlowOutput](
			metrics.ToolCashFlow,
			"Cash flow",
			"Money in, money out, net and savings rate per period. Transfers between the person's own accounts and loan payments are excluded by default.",
			read, withFilterEnums(map[string][]any{"interval": strings2any("day", "week", "month", "quarter", "year")}),
		),
		metrics.ToolFindRecurring: define[FindRecurringInput, FindRecurringOutput](
			metrics.ToolFindRecurring,
			"Find recurring payments",
			"Detects subscriptions, bills and paychecks: merchants paid at a regular cadence with similar amounts, with the next expected date.",
			read, nil,
		),
		metrics.ToolRefreshData: define[RefreshDataInput, RefreshDataOutput](
			metrics.ToolRefreshData,
			"Refresh bank data",
			"Pulls the latest transactions and balances for banks that are already linked. Each bank can be refreshed once per cooldown window (shared with the fin app); refreshing early returns throttled. It cannot link, reconnect or remove banks.",
			domain.ScopeSync, nil,
		),
	}
})
