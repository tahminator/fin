package mcpserver_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"slices"
	"sort"
	"testing"
	"time"
	"uuid"

	"github.com/tahminator/fin/internal/domain"
	"github.com/tahminator/fin/internal/mcpserver"
	"github.com/tahminator/fin/internal/metrics"
	"github.com/tahminator/fin/internal/repository"
)

const validToken = "fin_at_valid"

type fakeVerifier struct {
	grant domain.AccessGrant
}

func (f fakeVerifier) VerifyAccessToken(_ context.Context, token string) (domain.AccessGrant, error) {
	if token != validToken {
		return domain.AccessGrant{}, domain.ErrInvalidToken
	}

	return f.grant, nil
}

type fakeData struct {
	repository.TransactionQueryRepository
	userIDs     []uuid.UUID
	connections []domain.Connection
	syncErrs    map[uuid.UUID]error
	synced      []metrics.SyncTrigger
}

func (f *fakeData) ListByUser(_ context.Context, userID uuid.UUID, _, _ int) ([]domain.Connection, int, error) {
	f.userIDs = append(f.userIDs, userID)

	return f.connections, len(f.connections), nil
}

func (f *fakeData) Sync(_ context.Context, userID, connectionID uuid.UUID, trigger metrics.SyncTrigger) error {
	f.userIDs = append(f.userIDs, userID)
	f.synced = append(f.synced, trigger)

	return f.syncErrs[connectionID]
}

func (f *fakeData) Get(_ context.Context, userID, _ uuid.UUID) (repository.TransactionDetail, error) {
	f.userIDs = append(f.userIDs, userID)

	return repository.TransactionDetail{}, domain.ErrNotFound
}

type fakeAccounts struct{}

func (fakeAccounts) ListByUser(context.Context, uuid.UUID, repository.AccountFilter, int, int) ([]domain.Account, int, error) {
	return nil, 0, nil
}

type fakeBalances struct{}

func (fakeBalances) Balances(context.Context, uuid.UUID, []uuid.UUID, []domain.AccountType) ([]domain.CurrencyBalance, error) {
	return nil, nil
}

type refusingLimiter struct{ refuse bool }

func (l refusingLimiter) Allow(context.Context, domain.RateLimit, string) (domain.RateLimitDecision, error) {
	return domain.RateLimitDecision{Allowed: !l.refuse, RetryAfter: 30 * time.Second}, nil
}

func newServer(t *testing.T, scopes []domain.Scope, data *fakeData, limiter refusingLimiter) (*httptest.Server, domain.AccessGrant) {
	t.Helper()

	grant := domain.AccessGrant{User: domain.User{ID: uuid.New()}, GrantID: uuid.New(), Scopes: scopes, ExpiresAt: time.Now().Add(time.Hour)}

	handler := mcpserver.NewHandler(mcpserver.Deps{
		Verifier:     fakeVerifier{grant: grant},
		Connections:  data,
		Accounts:     fakeAccounts{},
		Balances:     fakeBalances{},
		Transactions: data,
		Syncer:       data,
		Limiter:      limiter,
		Recorder:     metrics.Noop{},
		Config: mcpserver.Config{
			ResourceMetadataURL: "https://fin.example.test/.well-known/oauth-protected-resource/mcp",
			AllowedOrigins:      []string{"https://allowed.example.test"},
			CallLimit:           mcpserver.DefaultCallLimit(),
		},
	})

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	return server, grant
}

type reply struct {
	status int
	header http.Header
}

func post(t *testing.T, server *httptest.Server, headers map[string]string, method string, params any) (reply, map[string]any) {
	t.Helper()

	body, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": method, "params": params})

	req, _ := http.NewRequestWithContext(t.Context(), http.MethodPost, server.URL, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST: %v", err)
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	var decoded map[string]any

	_ = json.NewDecoder(resp.Body).Decode(&decoded)

	return reply{status: resp.StatusCode, header: resp.Header}, decoded
}

var bearer = map[string]string{"Authorization": "Bearer " + validToken}

func toolNames(t *testing.T, server *httptest.Server) []string {
	t.Helper()

	_, body := post(t, server, bearer, "tools/list", map[string]any{})

	result, ok := body["result"].(map[string]any)
	if !ok {
		t.Fatalf("tools/list = %v", body)
	}

	var names []string

	for _, tool := range result["tools"].([]any) {
		names = append(names, tool.(map[string]any)["name"].(string))
	}

	sort.Strings(names)

	return names
}

func callTool(t *testing.T, server *httptest.Server, name string, args map[string]any) map[string]any {
	t.Helper()

	_, body := post(t, server, bearer, "tools/call", map[string]any{"name": name, "arguments": args})

	result, ok := body["result"].(map[string]any)
	if !ok {
		t.Fatalf("tools/call %s = %v", name, body)
	}

	return result
}

func TestBearerChecks(t *testing.T) {
	server, _ := newServer(t, []domain.Scope{domain.ScopeRead}, &fakeData{}, refusingLimiter{})

	tests := []struct {
		name    string
		headers map[string]string
		status  int
		error   string
	}{
		{name: "no token", headers: nil, status: http.StatusUnauthorized},
		{name: "bad token", headers: map[string]string{"Authorization": "Bearer nope"}, status: http.StatusUnauthorized, error: `error="invalid_token"`},
		{name: "session cookie only", headers: map[string]string{"Cookie": "fin_session=abc"}, status: http.StatusUnauthorized},
		{name: "foreign origin", headers: map[string]string{"Authorization": "Bearer " + validToken, "Origin": "https://evil.example.test"}, status: http.StatusForbidden},
		{name: "allowed origin", headers: map[string]string{"Authorization": "Bearer " + validToken, "Origin": "https://allowed.example.test"}, status: http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, _ := post(t, server, tt.headers, "tools/list", map[string]any{})

			if resp.status != tt.status {
				t.Fatalf("status = %d, want %d", resp.status, tt.status)
			}

			challenge := resp.header.Get("WWW-Authenticate")
			if tt.status == http.StatusUnauthorized && !bytes.Contains([]byte(challenge), []byte(`resource_metadata="https://fin.example.test/.well-known/oauth-protected-resource/mcp", scope="fin:read fin:sync"`)) {
				t.Errorf("WWW-Authenticate = %q", challenge)
			}

			if tt.error != "" && !bytes.Contains([]byte(challenge), []byte(tt.error)) {
				t.Errorf("WWW-Authenticate = %q, want %s", challenge, tt.error)
			}
		})
	}
}

func TestQueryStringTokensAreRejected(t *testing.T) {
	server, _ := newServer(t, []domain.Scope{domain.ScopeRead}, &fakeData{}, refusingLimiter{})

	req, _ := http.NewRequestWithContext(t.Context(), http.MethodPost, server.URL+"?access_token="+validToken, bytes.NewReader([]byte(`{}`)))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}

	_ = resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}
}

func TestRateLimitedGrantGets429(t *testing.T) {
	server, _ := newServer(t, []domain.Scope{domain.ScopeRead}, &fakeData{}, refusingLimiter{refuse: true})

	resp, _ := post(t, server, bearer, "tools/list", map[string]any{})

	if resp.status != http.StatusTooManyRequests || resp.header.Get("Retry-After") != "30" {
		t.Errorf("status = %d Retry-After = %q", resp.status, resp.header.Get("Retry-After"))
	}
}

func TestRefreshDataNeedsTheSyncScope(t *testing.T) {
	readOnly, _ := newServer(t, []domain.Scope{domain.ScopeRead}, &fakeData{}, refusingLimiter{})
	withSync, _ := newServer(t, []domain.Scope{domain.ScopeRead, domain.ScopeSync}, &fakeData{}, refusingLimiter{})

	if slices.Contains(toolNames(t, readOnly), "refresh_data") {
		t.Error("refresh_data listed without fin:sync")
	}

	if !slices.Contains(toolNames(t, withSync), "refresh_data") {
		t.Error("refresh_data missing with fin:sync")
	}

	_, body := post(t, readOnly, bearer, "tools/call", map[string]any{"name": "refresh_data", "arguments": map[string]any{}})
	if _, ok := body["error"]; !ok {
		t.Errorf("refresh_data without fin:sync = %v, want a JSON-RPC error", body)
	}
}

func TestNoToolCanManageBankConnections(t *testing.T) {
	server, _ := newServer(t, []domain.Scope{domain.ScopeRead, domain.ScopeSync}, &fakeData{}, refusingLimiter{})

	for _, name := range toolNames(t, server) {
		for _, forbidden := range []string{"link", "reauth", "reconnect", "remove", "delete", "exchange"} {
			if bytes.Contains([]byte(name), []byte(forbidden)) {
				t.Errorf("tool %s looks like it manages connections", name)
			}
		}
	}
}

func TestRefreshDataMapsSyncResults(t *testing.T) {
	synced, throttled, failing, broken := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	data := &fakeData{
		connections: []domain.Connection{
			{ID: synced, Status: domain.ConnectionStatusActive},
			{ID: throttled, Status: domain.ConnectionStatusActive},
			{ID: failing, Status: domain.ConnectionStatusActive},
			{ID: broken, Status: domain.ConnectionStatusNeedsReauth},
		},
		syncErrs: map[uuid.UUID]error{
			throttled: &domain.ThrottledError{RetryAfter: 90 * time.Second},
			failing:   errors.New("plaid down"),
		},
	}

	server, grant := newServer(t, []domain.Scope{domain.ScopeRead, domain.ScopeSync}, data, refusingLimiter{})

	output := callTool(t, server, "refresh_data", map[string]any{})["structuredContent"].(map[string]any)

	results := map[string]map[string]any{}
	for _, raw := range output["results"].([]any) {
		result := raw.(map[string]any)
		results[result["connection_id"].(string)] = result
	}

	want := map[uuid.UUID]string{synced: "synced", throttled: "throttled", failing: "failed", broken: "needs_reauth"}
	for id, expected := range want {
		if got := results[id.String()]["result"]; got != expected {
			t.Errorf("%s result = %v, want %s", id, got, expected)
		}
	}

	if results[throttled.String()]["retry_after_seconds"] != float64(90) {
		t.Errorf("retry_after_seconds = %v", results[throttled.String()]["retry_after_seconds"])
	}

	if len(data.synced) != 3 || slices.ContainsFunc(data.synced, func(trigger metrics.SyncTrigger) bool { return trigger != metrics.SyncTriggerMCP }) {
		t.Errorf("sync triggers = %v, want three mcp syncs and none for needs_reauth", data.synced)
	}

	for _, id := range data.userIDs {
		if id != grant.User.ID {
			t.Errorf("repository called for user %s, want %s", id, grant.User.ID)
		}
	}
}

func TestToolsPassTheTokenUserAndHideInternalErrors(t *testing.T) {
	data := &fakeData{}
	server, grant := newServer(t, []domain.Scope{domain.ScopeRead}, data, refusingLimiter{})

	result := callTool(t, server, "get_transaction", map[string]any{"transaction_id": uuid.New().String()})

	if result["isError"] != true {
		t.Fatalf("result = %v, want error", result)
	}

	if !slices.Equal(data.userIDs, []uuid.UUID{grant.User.ID}) {
		t.Errorf("user ids = %v", data.userIDs)
	}

	invalid := callTool(t, server, "search_transactions", map[string]any{"from": "2020-01-01", "to": "2026-01-01"})
	if invalid["isError"] != true {
		t.Errorf("a 6-year range was accepted: %v", invalid)
	}
}

func TestOpenAPIToolsMatchToolsList(t *testing.T) {
	server, _ := newServer(t, []domain.Scope{domain.ScopeRead, domain.ScopeSync}, &fakeData{}, refusingLimiter{})

	var documented []string

	for _, document := range mcpserver.ToolDocuments() {
		documented = append(documented, document.Name)

		if document.InputSchema == nil || document.OutputSchema == nil {
			t.Errorf("%s has no schema", document.Name)
		}

		if (document.Name == "refresh_data") != (document.Scope == string(domain.ScopeSync)) || (document.Scope == string(domain.ScopeRead)) != document.ReadOnly {
			t.Errorf("%s scope %s readOnly %v", document.Name, document.Scope, document.ReadOnly)
		}
	}

	sort.Strings(documented)

	if listed := toolNames(t, server); !slices.Equal(documented, listed) {
		t.Errorf("documented %v, listed %v", documented, listed)
	}
}
