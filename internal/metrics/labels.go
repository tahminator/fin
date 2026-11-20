package metrics

type SyncTrigger int

const (
	SyncTriggerLink SyncTrigger = iota
	SyncTriggerManual
	SyncTriggerScheduled
	SyncTriggerReauth
)

func (t SyncTrigger) String() string {
	switch t {
	case SyncTriggerLink:
		return "link"
	case SyncTriggerManual:
		return "manual"
	case SyncTriggerScheduled:
		return "scheduled"
	case SyncTriggerReauth:
		return "reauth"
	}

	return "unknown"
}

func allSyncTriggers() []SyncTrigger {
	return []SyncTrigger{SyncTriggerLink, SyncTriggerManual, SyncTriggerScheduled, SyncTriggerReauth}
}

type SyncOutcome int

const (
	SyncOutcomeSuccess SyncOutcome = iota
	SyncOutcomeReauthRequired
	SyncOutcomeError
)

func (o SyncOutcome) String() string {
	switch o {
	case SyncOutcomeSuccess:
		return "success"
	case SyncOutcomeReauthRequired:
		return "reauth_required"
	case SyncOutcomeError:
		return "error"
	}

	return "unknown"
}

func allSyncOutcomes() []SyncOutcome {
	return []SyncOutcome{SyncOutcomeSuccess, SyncOutcomeReauthRequired, SyncOutcomeError}
}

type ScheduledRunOutcome int

const (
	ScheduledRunSuccess ScheduledRunOutcome = iota
	ScheduledRunPartialFailure
	ScheduledRunFailure
)

func (o ScheduledRunOutcome) String() string {
	switch o {
	case ScheduledRunSuccess:
		return "success"
	case ScheduledRunPartialFailure:
		return "partial_failure"
	case ScheduledRunFailure:
		return "failure"
	}

	return "unknown"
}

func allScheduledRunOutcomes() []ScheduledRunOutcome {
	return []ScheduledRunOutcome{ScheduledRunSuccess, ScheduledRunPartialFailure, ScheduledRunFailure}
}

type LoginOutcome int

const (
	LoginSuccess LoginOutcome = iota
	LoginNotAllowed
	LoginInvalidState
	LoginError
)

func (o LoginOutcome) String() string {
	switch o {
	case LoginSuccess:
		return "success"
	case LoginNotAllowed:
		return "not_allowed"
	case LoginInvalidState:
		return "invalid_state"
	case LoginError:
		return "error"
	}

	return "unknown"
}

func allLoginOutcomes() []LoginOutcome {
	return []LoginOutcome{LoginSuccess, LoginNotAllowed, LoginInvalidState, LoginError}
}

type Provider int

const (
	ProviderPlaid Provider = iota
)

func (p Provider) String() string {
	if p == ProviderPlaid {
		return "plaid"
	}

	return "unknown"
}

func allProviders() []Provider {
	return []Provider{ProviderPlaid}
}

type ProviderOperation int

const (
	OperationCreateLinkToken ProviderOperation = iota
	OperationCreateUpdateLinkToken
	OperationExchange
	OperationAccounts
	OperationSyncTransactions
	OperationRemove
)

func (o ProviderOperation) String() string {
	switch o {
	case OperationCreateLinkToken:
		return "create_link_token"
	case OperationCreateUpdateLinkToken:
		return "create_update_link_token"
	case OperationExchange:
		return "exchange"
	case OperationAccounts:
		return "accounts"
	case OperationSyncTransactions:
		return "sync_transactions"
	case OperationRemove:
		return "remove"
	}

	return "unknown"
}

func allProviderOperations() []ProviderOperation {
	return []ProviderOperation{OperationCreateLinkToken, OperationCreateUpdateLinkToken, OperationExchange, OperationAccounts, OperationSyncTransactions, OperationRemove}
}

type CallOutcome int

const (
	CallSuccess CallOutcome = iota
	CallError
)

func (o CallOutcome) String() string {
	switch o {
	case CallSuccess:
		return "success"
	case CallError:
		return "error"
	}

	return "unknown"
}

func allCallOutcomes() []CallOutcome {
	return []CallOutcome{CallSuccess, CallError}
}

type RouteGroup int

const (
	RouteGroupAPI RouteGroup = iota
	RouteGroupAuth
	RouteGroupDocs
	RouteGroupStatic
)

func (g RouteGroup) String() string {
	switch g {
	case RouteGroupAPI:
		return "api"
	case RouteGroupAuth:
		return "auth"
	case RouteGroupDocs:
		return "docs"
	case RouteGroupStatic:
		return "static"
	}

	return "unknown"
}

func allRouteGroups() []RouteGroup {
	return []RouteGroup{RouteGroupAPI, RouteGroupAuth, RouteGroupDocs, RouteGroupStatic}
}

type Method int

const (
	MethodGet Method = iota
	MethodPost
	MethodPut
	MethodPatch
	MethodDelete
	MethodOther
)

func (m Method) String() string {
	switch m {
	case MethodGet:
		return "get"
	case MethodPost:
		return "post"
	case MethodPut:
		return "put"
	case MethodPatch:
		return "patch"
	case MethodDelete:
		return "delete"
	case MethodOther:
		return "other"
	}

	return "unknown"
}

func allMethods() []Method {
	return []Method{MethodGet, MethodPost, MethodPut, MethodPatch, MethodDelete, MethodOther}
}

type StatusClass int

const (
	StatusClass1xx StatusClass = iota
	StatusClass2xx
	StatusClass3xx
	StatusClass4xx
	StatusClass5xx
)

func (c StatusClass) String() string {
	switch c {
	case StatusClass1xx:
		return "1xx"
	case StatusClass2xx:
		return "2xx"
	case StatusClass3xx:
		return "3xx"
	case StatusClass4xx:
		return "4xx"
	case StatusClass5xx:
		return "5xx"
	}

	return "unknown"
}

func allStatusClasses() []StatusClass {
	return []StatusClass{StatusClass1xx, StatusClass2xx, StatusClass3xx, StatusClass4xx, StatusClass5xx}
}

type ConnectionStatus int

const (
	ConnectionStatusActive ConnectionStatus = iota
	ConnectionStatusNeedsReauth
	ConnectionStatusOther
)

func (s ConnectionStatus) String() string {
	switch s {
	case ConnectionStatusActive:
		return "active"
	case ConnectionStatusNeedsReauth:
		return "needs_reauth"
	case ConnectionStatusOther:
		return "other"
	}

	return "unknown"
}

func allConnectionStatuses() []ConnectionStatus {
	return []ConnectionStatus{ConnectionStatusActive, ConnectionStatusNeedsReauth, ConnectionStatusOther}
}
