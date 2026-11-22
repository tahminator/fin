package metrics

import "testing"

func TestEnumLabels(t *testing.T) {
	tests := []struct {
		name       string
		labels     []string
		declared   int
		outOfRange []string
	}{
		{
			name:       "SyncTrigger",
			labels:     labelValues(allSyncTriggers()),
			declared:   int(SyncTriggerMCP) + 1,
			outOfRange: []string{SyncTrigger(-1).String(), (SyncTriggerMCP + 1).String()},
		},
		{
			name:       "SyncOutcome",
			labels:     labelValues(allSyncOutcomes()),
			declared:   int(SyncOutcomeThrottled) + 1,
			outOfRange: []string{SyncOutcome(-1).String(), (SyncOutcomeThrottled + 1).String()},
		},
		{
			name:       "ScheduledRunOutcome",
			labels:     labelValues(allScheduledRunOutcomes()),
			declared:   int(ScheduledRunFailure) + 1,
			outOfRange: []string{ScheduledRunOutcome(-1).String(), (ScheduledRunFailure + 1).String()},
		},
		{
			name:       "LoginOutcome",
			labels:     labelValues(allLoginOutcomes()),
			declared:   int(LoginError) + 1,
			outOfRange: []string{LoginOutcome(-1).String(), (LoginError + 1).String()},
		},
		{
			name:       "Provider",
			labels:     labelValues(allProviders()),
			declared:   int(ProviderPlaid) + 1,
			outOfRange: []string{Provider(-1).String(), (ProviderPlaid + 1).String()},
		},
		{
			name:       "ProviderOperation",
			labels:     labelValues(allProviderOperations()),
			declared:   int(OperationInstitution) + 1,
			outOfRange: []string{ProviderOperation(-1).String(), (OperationInstitution + 1).String()},
		},
		{
			name:       "CallOutcome",
			labels:     labelValues(allCallOutcomes()),
			declared:   int(CallError) + 1,
			outOfRange: []string{CallOutcome(-1).String(), (CallError + 1).String()},
		},
		{
			name:       "RouteGroup",
			labels:     labelValues(allRouteGroups()),
			declared:   int(RouteGroupOAuth) + 1,
			outOfRange: []string{RouteGroup(-1).String(), (RouteGroupOAuth + 1).String()},
		},
		{
			name:       "OAuthGrantType",
			labels:     labelValues(allOAuthGrantTypes()),
			declared:   int(OAuthGrantRefreshToken) + 1,
			outOfRange: []string{OAuthGrantType(-1).String(), (OAuthGrantRefreshToken + 1).String()},
		},
		{
			name:       "OAuthOutcome",
			labels:     labelValues(allOAuthOutcomes()),
			declared:   int(OAuthOutcomeError) + 1,
			outOfRange: []string{OAuthOutcome(-1).String(), (OAuthOutcomeError + 1).String()},
		},
		{
			name:       "ClientSource",
			labels:     labelValues(allClientSources()),
			declared:   int(ClientSourceRegistered) + 1,
			outOfRange: []string{ClientSource(-1).String(), (ClientSourceRegistered + 1).String()},
		},
		{
			name:       "ConsentDecision",
			labels:     labelValues(allConsentDecisions()),
			declared:   int(ConsentDenied) + 1,
			outOfRange: []string{ConsentDecision(-1).String(), (ConsentDenied + 1).String()},
		},
		{
			name:       "MCPTool",
			labels:     labelValues(AllMCPTools()),
			declared:   int(ToolRefreshData) + 1,
			outOfRange: []string{MCPTool(-1).String(), (ToolRefreshData + 1).String()},
		},
		{
			name:       "ToolOutcome",
			labels:     labelValues(allToolOutcomes()),
			declared:   int(ToolOutcomeError) + 1,
			outOfRange: []string{ToolOutcome(-1).String(), (ToolOutcomeError + 1).String()},
		},
		{
			name:       "Method",
			labels:     labelValues(allMethods()),
			declared:   int(MethodOther) + 1,
			outOfRange: []string{Method(-1).String(), (MethodOther + 1).String()},
		},
		{
			name:       "StatusClass",
			labels:     labelValues(allStatusClasses()),
			declared:   int(StatusClass5xx) + 1,
			outOfRange: []string{StatusClass(-1).String(), (StatusClass5xx + 1).String()},
		},
		{
			name:       "ConnectionStatus",
			labels:     labelValues(allConnectionStatuses()),
			declared:   int(ConnectionStatusOther) + 1,
			outOfRange: []string{ConnectionStatus(-1).String(), (ConnectionStatusOther + 1).String()},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name+" lists every declared constant", func(t *testing.T) {
			if len(tt.labels) != tt.declared {
				t.Fatalf("got %d values, want %d", len(tt.labels), tt.declared)
			}
		})

		t.Run(tt.name+" labels are unique and known", func(t *testing.T) {
			seen := make(map[string]bool, len(tt.labels))

			for _, label := range tt.labels {
				if label == "" || label == "unknown" {
					t.Errorf("declared constant has label %q", label)
				}

				if seen[label] {
					t.Errorf("duplicate label %q", label)
				}

				seen[label] = true
			}
		})

		t.Run(tt.name+" out of range is unknown", func(t *testing.T) {
			for _, label := range tt.outOfRange {
				if label != "unknown" {
					t.Errorf("got %q, want %q", label, "unknown")
				}
			}
		})
	}
}
