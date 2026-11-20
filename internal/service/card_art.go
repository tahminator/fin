package service

import (
	"context"
	"fmt"
	"log/slog"
	"uuid"

	"github.com/tahminator/fin/internal/cardart"
	"github.com/tahminator/fin/internal/domain"
	"github.com/tahminator/fin/internal/repository"
)

const cardArtAccountsLimit = 1000

type CardArtMatcher interface {
	MatchConnection(ctx context.Context, userID, connectionID uuid.UUID)
}

type CardArtService struct {
	accounts repository.AccountRepository
	catalog  cardart.Catalog
}

func NewCardArtService(accounts repository.AccountRepository, catalog cardart.Catalog) *CardArtService {
	return &CardArtService{accounts: accounts, catalog: catalog}
}

func (s *CardArtService) Search(ctx context.Context, query string) ([]domain.CatalogCard, error) {
	return s.catalog.Search(ctx, query)
}

func (s *CardArtService) Select(ctx context.Context, userID, accountID uuid.UUID, cardID *string) (domain.Account, error) {
	var art domain.CardArt

	if cardID != nil {
		card, err := s.catalog.Card(ctx, *cardID)
		if err != nil {
			return domain.Account{}, err
		}

		art = domain.CardArt{CardID: &card.ID, ImageURL: &card.ImageURL}
	}

	return s.accounts.SelectCardArt(ctx, userID, accountID, art)
}

func (s *CardArtService) MatchConnection(ctx context.Context, userID, connectionID uuid.UUID) {
	filter := repository.AccountFilter{ConnectionID: &connectionID}

	accounts, _, err := s.accounts.ListByUser(ctx, userID, filter, cardArtAccountsLimit, 0)
	if err != nil {
		slog.WarnContext(ctx, "list accounts for card art failed", "connection_id", connectionID, "err", err)

		return
	}

	for _, account := range accounts {
		if account.CardArtSource != nil || !account.SupportsCardArt() {
			continue
		}

		if err := s.matchAccount(ctx, account); err != nil {
			slog.WarnContext(ctx, "match card art failed", "account_id", account.ID, "err", err)
		}
	}
}

func (s *CardArtService) matchAccount(ctx context.Context, account domain.Account) error {
	name := cardSearchName(account)
	if name == "" {
		return nil
	}

	var cards []domain.CatalogCard

	for _, term := range cardSearchTerms(name) {
		found, err := s.catalog.Search(ctx, term)
		if err != nil {
			return fmt.Errorf("search card catalog: %w", err)
		}

		cards = append(cards, found...)
	}

	var art domain.CardArt

	if card := newCardMatcher(account).best(cards); card != nil {
		art = domain.CardArt{CardID: &card.ID, ImageURL: &card.ImageURL}
	}

	if err := s.accounts.MatchCardArt(ctx, account.UserID, account.ID, art); err != nil {
		return fmt.Errorf("save matched card art: %w", err)
	}

	return nil
}
