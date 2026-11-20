package handler_test

import (
	"context"
	"net/http"
	"testing"
	"time"
	"uuid"

	"github.com/tahminator/fin/internal/domain"
	"github.com/tahminator/fin/internal/handler"
	"github.com/tahminator/fin/internal/repository"
)

func TestListTransactionsMapsFields(t *testing.T) {
	posted := domain.Transaction{
		ID:          uuid.New(),
		AccountID:   uuid.New(),
		AccountName: "Checking",
		ExternalID:  "plaid-tx-1",
		Amount:      "-12.50",
		Currency:    new("USD"),
		PostedOn:    time.Date(2026, time.March, 5, 0, 0, 0, 0, time.UTC),
		Name:        "Coffee",
		Category:    new("FOOD_AND_DRINK"),
	}
	authorized := domain.Transaction{
		ID:           uuid.New(),
		AccountID:    posted.AccountID,
		AccountName:  "Checking",
		Amount:       "1500.00",
		PostedOn:     time.Date(2026, time.December, 31, 0, 0, 0, 0, time.UTC),
		AuthorizedOn: new(time.Date(2026, time.December, 30, 0, 0, 0, 0, time.UTC)),
		Name:         "Payroll",
		MerchantName: new("Acme"),
		Pending:      true,
	}
	transactions := &fakeTransactions{
		listByUser: func(context.Context, uuid.UUID, repository.TransactionFilter, int, int) ([]domain.Transaction, int, error) {
			return []domain.Transaction{posted, authorized}, 2, nil
		},
	}
	api := newAPI(t, newUser(), handler.NewTransactionHandler(transactions))

	rec := api.Get("/api/transactions")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	items := decode[struct {
		Items []map[string]any `json:"items"`
	}](t, rec).Items

	if len(items) != 2 {
		t.Fatalf("items = %v, want two transactions", items)
	}

	tests := []struct {
		name string
		item map[string]any
		want map[string]any
	}{
		{
			name: "posted without authorization date",
			item: items[0],
			want: map[string]any{
				"id":           posted.ID.String(),
				"accountId":    posted.AccountID.String(),
				"accountName":  "Checking",
				"amount":       "-12.50",
				"currency":     "USD",
				"postedOn":     "2026-03-05",
				"authorizedOn": nil,
				"name":         "Coffee",
				"merchantName": nil,
				"category":     "FOOD_AND_DRINK",
				"pending":      false,
			},
		},
		{
			name: "pending with authorization date",
			item: items[1],
			want: map[string]any{
				"id":           authorized.ID.String(),
				"amount":       "1500.00",
				"currency":     nil,
				"postedOn":     "2026-12-31",
				"authorizedOn": "2026-12-30",
				"merchantName": "Acme",
				"category":     nil,
				"pending":      true,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			for key, value := range tc.want {
				got, ok := tc.item[key]
				if !ok {
					t.Errorf("missing %q in %v", key, tc.item)

					continue
				}

				if got != value {
					t.Errorf("%s = %#v, want %#v", key, got, value)
				}
			}

			if _, ok := tc.item["externalId"]; ok {
				t.Error("response exposes externalId")
			}
		})
	}
}

func TestListTransactionsAccountFilter(t *testing.T) {
	accountID := uuid.New()

	tests := []struct {
		name    string
		query   string
		account *uuid.UUID
	}{
		{name: "no filter lists every account", query: "", account: nil},
		{name: "account id narrows to one account", query: "?accountId=" + accountID.String(), account: &accountID},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var gotFilter repository.TransactionFilter

			transactions := &fakeTransactions{
				listByUser: func(_ context.Context, _ uuid.UUID, filter repository.TransactionFilter, _, _ int) ([]domain.Transaction, int, error) {
					gotFilter = filter

					return nil, 0, nil
				},
			}
			api := newAPI(t, newUser(), handler.NewTransactionHandler(transactions))

			rec := api.Get("/api/transactions" + tc.query)

			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d; body %s", rec.Code, http.StatusOK, rec.Body.String())
			}

			switch {
			case tc.account == nil && gotFilter.AccountID != nil:
				t.Errorf("filter account = %s, want none", gotFilter.AccountID)
			case tc.account != nil && (gotFilter.AccountID == nil || *gotFilter.AccountID != *tc.account):
				t.Errorf("filter account = %v, want %s", gotFilter.AccountID, tc.account)
			}
		})
	}
}

func TestListTransactionsRejectsInvalidAccountID(t *testing.T) {
	transactions := &fakeTransactions{
		listByUser: func(context.Context, uuid.UUID, repository.TransactionFilter, int, int) ([]domain.Transaction, int, error) {
			t.Error("repository called for invalid account id")

			return nil, 0, nil
		},
	}
	api := newAPI(t, newUser(), handler.NewTransactionHandler(transactions))

	rec := api.Get("/api/transactions?accountId=not-a-uuid")

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want %d; body %s", rec.Code, http.StatusUnprocessableEntity, rec.Body.String())
	}
}
