package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"uuid"

	"github.com/tahminator/fin/internal/domain"
	"github.com/tahminator/fin/internal/handler"
	"github.com/tahminator/fin/internal/service"
)

func newCardArtHandler(accounts *fakeAccounts, catalog *fakeCatalog) *handler.CardArtHandler {
	return handler.NewCardArtHandler(service.NewCardArtService(accounts, catalog))
}

func TestSearchCardArtRejectsBlankQuery(t *testing.T) {
	catalog := &fakeCatalog{
		search: func(context.Context, string) ([]domain.CatalogCard, error) {
			t.Error("catalog searched for a blank query")

			return nil, nil
		},
	}
	api := newAPI(t, newUser(), newCardArtHandler(&fakeAccounts{}, catalog))

	for _, path := range []string{"/api/card-art/search", "/api/card-art/search?query=", "/api/card-art/search?query=%20%20"} {
		if rec := api.Get(path); rec.Code != http.StatusUnprocessableEntity {
			t.Errorf("GET %s status = %d, want %d", path, rec.Code, http.StatusUnprocessableEntity)
		}
	}
}

func TestSearchCardArtReturnsCatalogCards(t *testing.T) {
	var gotQuery string

	catalog := &fakeCatalog{
		search: func(_ context.Context, query string) ([]domain.CatalogCard, error) {
			gotQuery = query

			return []domain.CatalogCard{
				{ID: "us-chase-freedom-unlimited", Name: "Chase Freedom Unlimited", Issuer: new("Chase"), ImageURL: "https://cards.example/fu.png"},
			}, nil
		},
	}
	api := newAPI(t, newUser(), newCardArtHandler(&fakeAccounts{}, catalog))

	rec := api.Get("/api/card-art/search?query=%20freedom%20unlimited%20")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	if gotQuery != "freedom unlimited" {
		t.Errorf("catalog query = %q, want trimmed query", gotQuery)
	}

	items := decode[handler.CardArtSearchResults](t, rec).Items
	want := handler.CatalogCard{ID: "us-chase-freedom-unlimited", Name: "Chase Freedom Unlimited", Issuer: new("Chase"), ImageURL: "https://cards.example/fu.png"}

	if len(items) != 1 || items[0].ID != want.ID || items[0].ImageURL != want.ImageURL || *items[0].Issuer != *want.Issuer {
		t.Errorf("items = %+v, want [%+v]", items, want)
	}
}

func TestSearchCardArtCatalogUnavailable(t *testing.T) {
	catalog := &fakeCatalog{
		search: func(context.Context, string) ([]domain.CatalogCard, error) {
			return nil, fmt.Errorf("%w: card catalog returned status 429", domain.ErrUnavailable)
		},
	}
	api := newAPI(t, newUser(), newCardArtHandler(&fakeAccounts{}, catalog))

	rec := api.Get("/api/card-art/search?query=sapphire")

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d; body %s", rec.Code, http.StatusServiceUnavailable, rec.Body.String())
	}
}

func TestSelectCardArt(t *testing.T) {
	user := newUser()
	accountID := uuid.New()

	tests := []struct {
		name       string
		body       string
		card       func(ctx context.Context, id string) (domain.CatalogCard, error)
		wantStatus int
		wantArt    *domain.CardArt
	}{
		{
			name: "catalog card",
			body: `{"cardId":"us-chase-sapphire-preferred"}`,
			card: func(_ context.Context, id string) (domain.CatalogCard, error) {
				return domain.CatalogCard{ID: id, Name: "Sapphire Preferred", ImageURL: "https://cards.example/csp.png"}, nil
			},
			wantStatus: http.StatusOK,
			wantArt:    &domain.CardArt{CardID: new("us-chase-sapphire-preferred"), ImageURL: new("https://cards.example/csp.png")},
		},
		{
			name:       "default icon",
			body:       `{"cardId":null}`,
			wantStatus: http.StatusOK,
			wantArt:    &domain.CardArt{},
		},
		{
			name: "unknown card",
			body: `{"cardId":"us-made-up"}`,
			card: func(context.Context, string) (domain.CatalogCard, error) {
				return domain.CatalogCard{}, domain.ErrNotFound
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name: "catalog unavailable",
			body: `{"cardId":"us-chase-sapphire-preferred"}`,
			card: func(context.Context, string) (domain.CatalogCard, error) {
				return domain.CatalogCard{}, domain.ErrUnavailable
			},
			wantStatus: http.StatusServiceUnavailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var saved *domain.CardArt

			accounts := &fakeAccounts{
				selectCardArt: func(_ context.Context, userID, id uuid.UUID, art domain.CardArt) (domain.Account, error) {
					if userID != user.ID || id != accountID {
						return domain.Account{}, domain.ErrNotFound
					}

					saved = &art

					return domain.Account{ID: id, CardArtID: art.CardID, CardArtURL: art.ImageURL, CardArtSource: new(domain.CardArtSourceSelected)}, nil
				},
			}
			catalog := &fakeCatalog{
				card: func(ctx context.Context, id string) (domain.CatalogCard, error) {
					if tt.card == nil {
						t.Error("catalog looked up a card for a null cardId")

						return domain.CatalogCard{}, domain.ErrNotFound
					}

					return tt.card(ctx, id)
				},
			}
			api := newAPI(t, user, newCardArtHandler(accounts, catalog))

			rec := api.Put("/api/accounts/"+accountID.String()+"/card-art", strings.NewReader(tt.body))

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body %s", rec.Code, tt.wantStatus, rec.Body.String())
			}

			if tt.wantArt == nil {
				if saved != nil {
					t.Errorf("saved card art %+v after a failed lookup", *saved)
				}

				return
			}

			if saved == nil || !equalStrings(saved.CardID, tt.wantArt.CardID) || !equalStrings(saved.ImageURL, tt.wantArt.ImageURL) {
				t.Errorf("saved card art = %+v, want %+v", saved, tt.wantArt)
			}

			account := decode[handler.Account](t, rec)
			if account.CardArtSource == nil || *account.CardArtSource != domain.CardArtSourceSelected {
				t.Errorf("cardArtSource = %v, want selected", account.CardArtSource)
			}
		})
	}
}

func TestSelectCardArtUnknownAccount(t *testing.T) {
	accounts := &fakeAccounts{
		selectCardArt: func(context.Context, uuid.UUID, uuid.UUID, domain.CardArt) (domain.Account, error) {
			return domain.Account{}, domain.ErrNotFound
		},
	}
	api := newAPI(t, newUser(), newCardArtHandler(accounts, &fakeCatalog{}))

	rec := api.Put("/api/accounts/"+uuid.New().String()+"/card-art", strings.NewReader(`{"cardId":null}`))

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d; body %s", rec.Code, http.StatusNotFound, rec.Body.String())
	}
}

func equalStrings(got, want *string) bool {
	if got == nil || want == nil {
		return got == want
	}

	return *got == *want
}
