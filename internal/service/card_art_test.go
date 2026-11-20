package service

import (
	"context"
	"errors"
	"slices"
	"testing"
	"uuid"

	"github.com/tahminator/fin/internal/domain"
	"github.com/tahminator/fin/internal/repository"
)

var testCatalogCards = []domain.CatalogCard{
	{ID: "us-chase-freedom-unlimited", Name: "Chase Freedom Unlimited® Credit Card", Issuer: new("Chase"), ImageURL: "https://cards.example/fu.png"},
	{ID: "us-chase-freedom-flex", Name: "Chase Freedom Flex®", Issuer: new("Chase"), ImageURL: "https://cards.example/ff.png"},
	{ID: "us-chase-sapphire-preferred", Name: "Chase Sapphire Preferred® Card", Issuer: new("Chase"), ImageURL: "https://cards.example/csp.png"},
	{ID: "us-amex-platinum", Name: "The Platinum Card®", Issuer: new("American Express"), ImageURL: "https://cards.example/amex-plat.png"},
	{ID: "us-other-platinum", Name: "The Platinum Card", Issuer: new("Other Bank"), ImageURL: "https://cards.example/other-plat.png"},
	{ID: "us-golden-card", Name: "Golden Card", Issuer: new("Other Bank"), ImageURL: "https://cards.example/golden.png"},
	{ID: "us-blank", Name: "®", Issuer: nil, ImageURL: "https://cards.example/blank.png"},
}

func TestCardMatcherBest(t *testing.T) {
	tests := []struct {
		name    string
		account domain.Account
		want    string
	}{
		{
			name:    "official name picks the exact product over siblings",
			account: domain.Account{Name: "CREDIT CARD", OfficialName: new("Chase Freedom Unlimited"), InstitutionName: new("Chase")},
			want:    "us-chase-freedom-unlimited",
		},
		{
			name:    "blank official name falls back to the account name",
			account: domain.Account{Name: "Sapphire Preferred", OfficialName: new("  ")},
			want:    "us-chase-sapphire-preferred",
		},
		{
			name:    "issuer breaks a tie between equally named cards",
			account: domain.Account{Name: "Platinum Card", InstitutionName: new("American Express")},
			want:    "us-amex-platinum",
		},
		{
			name:    "authorized user suffix still matches the base card",
			account: domain.Account{Name: "Platinum Card Authorized User", InstitutionName: new("American Express")},
			want:    "us-amex-platinum",
		},
		{
			name:    "generic name matches nothing",
			account: domain.Account{Name: "Credit Card", InstitutionName: new("Chase")},
		},
		{
			name:    "partial words do not match",
			account: domain.Account{Name: "Gold"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := newCardMatcher(tt.account).best(testCatalogCards)

			switch {
			case tt.want == "" && got != nil:
				t.Errorf("best = %s, want no match", got.ID)
			case tt.want != "" && (got == nil || got.ID != tt.want):
				t.Errorf("best = %v, want %s", got, tt.want)
			}
		})
	}
}

func TestCardSearchTerms(t *testing.T) {
	tests := []struct {
		name string
		want []string
	}{
		{name: "Chase Freedom Unlimited", want: []string{"Chase Freedom Unlimited"}},
		{name: "Platinum Card - Authorized User", want: []string{"Platinum Card - Authorized User", "platinum card"}},
		{name: "Authorized User", want: []string{"Authorized User"}},
	}

	for _, tt := range tests {
		if got := cardSearchTerms(tt.name); !slices.Equal(got, tt.want) {
			t.Errorf("cardSearchTerms(%q) = %q, want %q", tt.name, got, tt.want)
		}
	}
}

type cardArtFixture struct {
	userID       uuid.UUID
	connectionID uuid.UUID
	accounts     []domain.Account
	searches     []string
	matched      map[uuid.UUID]domain.CardArt
	searchErr    map[string]error
	service      *CardArtService
}

func newCardArtFixture(t *testing.T, accounts ...domain.Account) *cardArtFixture {
	t.Helper()

	f := &cardArtFixture{
		userID:       uuid.New(),
		connectionID: uuid.New(),
		accounts:     accounts,
		matched:      map[uuid.UUID]domain.CardArt{},
		searchErr:    map[string]error{},
	}

	repo := &fakeAccounts{
		listByUser: func(_ context.Context, userID uuid.UUID, filter repository.AccountFilter, _, _ int) ([]domain.Account, int, error) {
			if userID != f.userID || filter.ConnectionID == nil || *filter.ConnectionID != f.connectionID {
				t.Errorf("ListByUser(%v, %+v), want user %v and connection %v", userID, filter, f.userID, f.connectionID)
			}

			return f.accounts, len(f.accounts), nil
		},
		matchCardArt: func(_ context.Context, userID, id uuid.UUID, art domain.CardArt) error {
			if userID != f.userID {
				t.Errorf("MatchCardArt user = %v, want %v", userID, f.userID)
			}

			f.matched[id] = art

			return nil
		},
	}
	catalog := &fakeCatalog{
		search: func(_ context.Context, query string) ([]domain.CatalogCard, error) {
			f.searches = append(f.searches, query)

			if err := f.searchErr[query]; err != nil {
				return nil, err
			}

			return testCatalogCards, nil
		},
	}

	f.service = NewCardArtService(repo, catalog)

	return f
}

func (f *cardArtFixture) account(name string, accountType domain.AccountType, subtype string, source *domain.CardArtSource) domain.Account {
	return domain.Account{
		ID:            uuid.New(),
		UserID:        f.userID,
		ConnectionID:  f.connectionID,
		Name:          name,
		Type:          accountType,
		Subtype:       new(subtype),
		CardArtSource: source,
	}
}

func TestMatchConnectionOnlyMatchesPendingCreditAccounts(t *testing.T) {
	f := newCardArtFixture(t)

	credit := f.account("Freedom Unlimited", domain.AccountTypeCredit, "credit card", nil)
	checking := f.account("Sapphire Preferred", domain.AccountTypeDepository, "checking", nil)
	savings := f.account("Freedom Unlimited", domain.AccountTypeDepository, "savings", nil)
	selected := f.account("Freedom Flex", domain.AccountTypeCredit, "credit card", new(domain.CardArtSourceSelected))
	matched := f.account("Freedom Flex", domain.AccountTypeCredit, "credit card", new(domain.CardArtSourceMatched))
	f.accounts = []domain.Account{credit, checking, savings, selected, matched}

	f.service.MatchConnection(t.Context(), f.userID, f.connectionID)

	if !slices.Equal(f.searches, []string{"Freedom Unlimited"}) {
		t.Errorf("searches = %q, want only the pending credit account", f.searches)
	}

	want := map[uuid.UUID]string{
		credit.ID: "https://cards.example/fu.png",
	}

	if len(f.matched) != len(want) {
		t.Errorf("matched %d accounts, want %d: %+v", len(f.matched), len(want), f.matched)
	}

	for id, url := range want {
		art, ok := f.matched[id]
		if !ok || art.ImageURL == nil || *art.ImageURL != url || art.CardID == nil {
			t.Errorf("account %v art = %+v, want %s", id, art, url)
		}
	}
}

func TestMatchConnectionRecordsMissSoItIsNotSearchedAgain(t *testing.T) {
	f := newCardArtFixture(t)
	generic := f.account("Credit Card", domain.AccountTypeCredit, "credit card", nil)
	f.accounts = []domain.Account{generic}

	f.service.MatchConnection(t.Context(), f.userID, f.connectionID)

	art, ok := f.matched[generic.ID]
	if !ok {
		t.Fatal("no match result saved for an account without a matching card")
	}

	if art.CardID != nil || art.ImageURL != nil {
		t.Errorf("art = %+v, want an empty match", art)
	}
}

func TestMatchConnectionSearchFailureLeavesAccountPending(t *testing.T) {
	f := newCardArtFixture(t)
	failing := f.account("Platinum Card Authorized User", domain.AccountTypeCredit, "credit card", nil)
	healthy := f.account("Freedom Unlimited", domain.AccountTypeCredit, "credit card", nil)
	f.accounts = []domain.Account{failing, healthy}
	f.searchErr["platinum card"] = errors.Join(domain.ErrUnavailable, errors.New("429"))

	f.service.MatchConnection(t.Context(), f.userID, f.connectionID)

	if _, ok := f.matched[failing.ID]; ok {
		t.Error("saved card art although one of its searches failed")
	}

	if _, ok := f.matched[healthy.ID]; !ok {
		t.Error("a failure on one account stopped matching the next")
	}
}
