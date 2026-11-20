package handler_test

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"uuid"

	"github.com/tahminator/fin/internal/domain"
	"github.com/tahminator/fin/internal/handler"
	"github.com/tahminator/fin/internal/repository"
)

func TestListAccountsRejectsInvalidPaging(t *testing.T) {
	tests := []struct {
		name  string
		query string
	}{
		{name: "page size above maximum", query: "pageSize=101"},
		{name: "page size zero", query: "pageSize=0"},
		{name: "page zero", query: "page=0"},
		{name: "negative page", query: "page=-1"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			accounts := &fakeAccounts{
				listByUser: func(context.Context, uuid.UUID, repository.AccountFilter, int, int) ([]domain.Account, int, error) {
					t.Error("repository called for invalid paging")

					return nil, 0, nil
				},
			}
			api := newAPI(t, newUser(), handler.NewAccountHandler(accounts))

			rec := api.Get("/api/accounts?" + tc.query)

			if rec.Code != http.StatusUnprocessableEntity {
				t.Errorf("status = %d, want %d; body %s", rec.Code, http.StatusUnprocessableEntity, rec.Body.String())
			}
		})
	}
}

func TestListAccountsPaging(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		returned int
		total    int
		limit    int
		offset   int
		hasNext  bool
	}{
		{name: "defaults to first page of 25", query: "", returned: 25, total: 26, limit: 25, offset: 0, hasNext: true},
		{name: "maximum page size is accepted", query: "?pageSize=100", returned: 3, total: 3, limit: 100, offset: 0, hasNext: false},
		{name: "third page of ten with more remaining", query: "?page=3&pageSize=10", returned: 10, total: 35, limit: 10, offset: 20, hasNext: true},
		{name: "third page of ten reaching total", query: "?page=3&pageSize=10", returned: 10, total: 30, limit: 10, offset: 20, hasNext: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			user := newUser()

			var gotUser uuid.UUID
			var gotLimit, gotOffset int

			accounts := &fakeAccounts{
				listByUser: func(_ context.Context, userID uuid.UUID, _ repository.AccountFilter, limit, offset int) ([]domain.Account, int, error) {
					gotUser, gotLimit, gotOffset = userID, limit, offset

					return make([]domain.Account, tc.returned), tc.total, nil
				},
			}
			api := newAPI(t, user, handler.NewAccountHandler(accounts))

			rec := api.Get("/api/accounts" + tc.query)

			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d; body %s", rec.Code, http.StatusOK, rec.Body.String())
			}

			if gotUser != user.ID || gotLimit != tc.limit || gotOffset != tc.offset {
				t.Errorf("repository got (%s, %d, %d), want (%s, %d, %d)", gotUser, gotLimit, gotOffset, user.ID, tc.limit, tc.offset)
			}

			page := decode[handler.Page[handler.Account]](t, rec)

			if page.HasNext != tc.hasNext || page.Total != tc.total || page.PageSize != tc.limit || len(page.Items) != tc.returned {
				t.Errorf("page = {hasNext %v total %d pageSize %d items %d}, want {hasNext %v total %d pageSize %d items %d}",
					page.HasNext, page.Total, page.PageSize, len(page.Items), tc.hasNext, tc.total, tc.limit, tc.returned)
			}
		})
	}
}

func TestListAccountsConnectionFilter(t *testing.T) {
	connectionID := uuid.New()

	tests := []struct {
		name       string
		query      string
		connection *uuid.UUID
	}{
		{name: "no filter lists every connection", query: "", connection: nil},
		{name: "connection id narrows to one connection", query: "?connectionId=" + connectionID.String(), connection: &connectionID},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var gotFilter repository.AccountFilter

			accounts := &fakeAccounts{
				listByUser: func(_ context.Context, _ uuid.UUID, filter repository.AccountFilter, _, _ int) ([]domain.Account, int, error) {
					gotFilter = filter

					return nil, 0, nil
				},
			}
			api := newAPI(t, newUser(), handler.NewAccountHandler(accounts))

			rec := api.Get("/api/accounts" + tc.query)

			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d; body %s", rec.Code, http.StatusOK, rec.Body.String())
			}

			switch {
			case tc.connection == nil && gotFilter.ConnectionID != nil:
				t.Errorf("filter connection = %s, want none", gotFilter.ConnectionID)
			case tc.connection != nil && (gotFilter.ConnectionID == nil || *gotFilter.ConnectionID != *tc.connection):
				t.Errorf("filter connection = %v, want %s", gotFilter.ConnectionID, tc.connection)
			}
		})
	}
}

func TestListAccountsRejectsInvalidConnectionID(t *testing.T) {
	accounts := &fakeAccounts{
		listByUser: func(context.Context, uuid.UUID, repository.AccountFilter, int, int) ([]domain.Account, int, error) {
			t.Error("repository called for invalid connection id")

			return nil, 0, nil
		},
	}
	api := newAPI(t, newUser(), handler.NewAccountHandler(accounts))

	rec := api.Get("/api/accounts?connectionId=not-a-uuid")

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want %d; body %s", rec.Code, http.StatusUnprocessableEntity, rec.Body.String())
	}
}

func TestListAccountsEmptyEncodesItemsAsArray(t *testing.T) {
	accounts := &fakeAccounts{
		listByUser: func(context.Context, uuid.UUID, repository.AccountFilter, int, int) ([]domain.Account, int, error) {
			return nil, 0, nil
		},
	}
	api := newAPI(t, newUser(), handler.NewAccountHandler(accounts))

	rec := api.Get("/api/accounts")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	if !strings.Contains(rec.Body.String(), `"items":[]`) {
		t.Errorf("body = %s, want items encoded as []", rec.Body.String())
	}
}

func TestListAccountsMapsFields(t *testing.T) {
	account := domain.Account{
		ID:               uuid.New(),
		UserID:           uuid.New(),
		ConnectionID:     uuid.New(),
		InstitutionName:  new("First Platypus Bank"),
		InstitutionURL:   new("https://www.platypusbank.example"),
		ExternalID:       "plaid-account-1",
		Name:             "Checking",
		OfficialName:     new("Platypus Gold Checking"),
		Mask:             new("0000"),
		Type:             domain.AccountTypeDepository,
		Subtype:          new("checking"),
		CurrentBalance:   new("110.25"),
		AvailableBalance: nil,
		Currency:         new("USD"),
	}
	accounts := &fakeAccounts{
		listByUser: func(context.Context, uuid.UUID, repository.AccountFilter, int, int) ([]domain.Account, int, error) {
			return []domain.Account{account}, 1, nil
		},
	}
	api := newAPI(t, newUser(), handler.NewAccountHandler(accounts))

	rec := api.Get("/api/accounts")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	items := decode[struct {
		Items []map[string]any `json:"items"`
	}](t, rec).Items

	if len(items) != 1 {
		t.Fatalf("items = %v, want one account", items)
	}

	want := map[string]any{
		"id":                 account.ID.String(),
		"connectionId":       account.ConnectionID.String(),
		"institutionName":    "First Platypus Bank",
		"institutionLogoUrl": "https://www.google.com/s2/favicons?domain=www.platypusbank.example&sz=64",
		"name":               "Checking",
		"officialName":       "Platypus Gold Checking",
		"mask":               "0000",
		"type":               "depository",
		"subtype":            "checking",
		"currentBalance":     "110.25",
		"availableBalance":   nil,
		"currency":           "USD",
	}

	for key, value := range want {
		got, ok := items[0][key]
		if !ok {
			t.Errorf("missing %q in %v", key, items[0])

			continue
		}

		if got != value {
			t.Errorf("%s = %#v, want %#v", key, got, value)
		}
	}

	for _, leaked := range []string{"userId", "externalId", "UserID", "ExternalID"} {
		if _, ok := items[0][leaked]; ok {
			t.Errorf("response exposes %q", leaked)
		}
	}
}
