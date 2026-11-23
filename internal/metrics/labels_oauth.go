package metrics

type OAuthGrantType int

const (
	OAuthGrantAuthorizationCode OAuthGrantType = iota
	OAuthGrantRefreshToken
)

func (g OAuthGrantType) String() string {
	switch g {
	case OAuthGrantAuthorizationCode:
		return "authorization_code"
	case OAuthGrantRefreshToken:
		return "refresh_token"
	}

	return "unknown"
}

func allOAuthGrantTypes() []OAuthGrantType {
	return []OAuthGrantType{OAuthGrantAuthorizationCode, OAuthGrantRefreshToken}
}

type OAuthOutcome int

const (
	OAuthOutcomeSuccess OAuthOutcome = iota
	OAuthOutcomeInvalidRequest
	OAuthOutcomeInvalidClient
	OAuthOutcomeInvalidGrant
	OAuthOutcomeReuseDetected
	OAuthOutcomeError
)

func (o OAuthOutcome) String() string {
	switch o {
	case OAuthOutcomeSuccess:
		return "success"
	case OAuthOutcomeInvalidRequest:
		return "invalid_request"
	case OAuthOutcomeInvalidClient:
		return "invalid_client"
	case OAuthOutcomeInvalidGrant:
		return "invalid_grant"
	case OAuthOutcomeReuseDetected:
		return "reuse_detected"
	case OAuthOutcomeError:
		return "error"
	}

	return "unknown"
}

func allOAuthOutcomes() []OAuthOutcome {
	return []OAuthOutcome{
		OAuthOutcomeSuccess,
		OAuthOutcomeInvalidRequest,
		OAuthOutcomeInvalidClient,
		OAuthOutcomeInvalidGrant,
		OAuthOutcomeReuseDetected,
		OAuthOutcomeError,
	}
}

type ClientSource int

const (
	ClientSourceMetadataDocument ClientSource = iota
	ClientSourceRegistered
)

func (c ClientSource) String() string {
	switch c {
	case ClientSourceMetadataDocument:
		return "metadata_document"
	case ClientSourceRegistered:
		return "registered"
	}

	return "unknown"
}

func allClientSources() []ClientSource {
	return []ClientSource{ClientSourceMetadataDocument, ClientSourceRegistered}
}

type ConsentDecision int

const (
	ConsentApproved ConsentDecision = iota
	ConsentDenied
)

func (d ConsentDecision) String() string {
	switch d {
	case ConsentApproved:
		return "approved"
	case ConsentDenied:
		return "denied"
	}

	return "unknown"
}

func allConsentDecisions() []ConsentDecision {
	return []ConsentDecision{ConsentApproved, ConsentDenied}
}

type MCPTool int

const (
	ToolListConnections MCPTool = iota
	ToolListAccounts
	ToolGetBalances
	ToolSearchTransactions
	ToolGetTransaction
	ToolListFacets
	ToolAggregateTransactions
	ToolCashFlow
	ToolFindRecurring
	ToolListInvestmentHoldings
	ToolSearchInvestmentTransactions
	ToolGetInvestmentPerformance
	ToolRefreshData
)

func (t MCPTool) String() string {
	switch t {
	case ToolListConnections:
		return "list_connections"
	case ToolListAccounts:
		return "list_accounts"
	case ToolGetBalances:
		return "get_balances"
	case ToolSearchTransactions:
		return "search_transactions"
	case ToolGetTransaction:
		return "get_transaction"
	case ToolListFacets:
		return "list_facets"
	case ToolAggregateTransactions:
		return "aggregate_transactions"
	case ToolCashFlow:
		return "cash_flow"
	case ToolFindRecurring:
		return "find_recurring"
	case ToolListInvestmentHoldings:
		return "list_investment_holdings"
	case ToolSearchInvestmentTransactions:
		return "search_investment_transactions"
	case ToolGetInvestmentPerformance:
		return "get_investment_performance"
	case ToolRefreshData:
		return "refresh_data"
	}

	return "unknown"
}

func AllMCPTools() []MCPTool {
	return []MCPTool{
		ToolListConnections,
		ToolListAccounts,
		ToolGetBalances,
		ToolSearchTransactions,
		ToolGetTransaction,
		ToolListFacets,
		ToolAggregateTransactions,
		ToolCashFlow,
		ToolFindRecurring,
		ToolListInvestmentHoldings,
		ToolSearchInvestmentTransactions,
		ToolGetInvestmentPerformance,
		ToolRefreshData,
	}
}

type ToolOutcome int

const (
	ToolOutcomeSuccess ToolOutcome = iota
	ToolOutcomeInvalidParams
	ToolOutcomeNotFound
	ToolOutcomeError
)

func (o ToolOutcome) String() string {
	switch o {
	case ToolOutcomeSuccess:
		return "success"
	case ToolOutcomeInvalidParams:
		return "invalid_params"
	case ToolOutcomeNotFound:
		return "not_found"
	case ToolOutcomeError:
		return "error"
	}

	return "unknown"
}

func allToolOutcomes() []ToolOutcome {
	return []ToolOutcome{ToolOutcomeSuccess, ToolOutcomeInvalidParams, ToolOutcomeNotFound, ToolOutcomeError}
}
