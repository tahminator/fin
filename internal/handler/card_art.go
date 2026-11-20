package handler

import (
	"context"
	"net/http"
	"strings"

	"github.com/danielgtaylor/huma/v2"

	"github.com/tahminator/fin/internal/domain"
	"github.com/tahminator/fin/internal/service"
)

type CatalogCard struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Issuer   *string `json:"issuer"`
	ImageURL string  `json:"imageUrl"`
}

func newCatalogCard(card domain.CatalogCard) CatalogCard {
	return CatalogCard{
		ID:       card.ID,
		Name:     card.Name,
		Issuer:   card.Issuer,
		ImageURL: card.ImageURL,
	}
}

type CardArtSearchResults struct {
	Items []CatalogCard `json:"items" nullable:"false"`
}

type SearchCardArtInput struct {
	Query string `query:"query" required:"true" minLength:"1" maxLength:"100" doc:"Card name to search for"`
}

type SearchCardArtOutput struct {
	Body CardArtSearchResults
}

type SelectCardArtBody struct {
	CardID *string `json:"cardId" minLength:"1" maxLength:"200" doc:"Catalog card ID, or null to show the default icon"`
}

type SelectCardArtInput struct {
	AccountID string `path:"accountId" format:"uuid"`
	Body      SelectCardArtBody
}

type AccountOutput struct {
	Body Account
}

type CardArtHandler struct {
	service *service.CardArtService
}

func NewCardArtHandler(cardArtService *service.CardArtService) *CardArtHandler {
	return &CardArtHandler{service: cardArtService}
}

func (h *CardArtHandler) Register(api huma.API) {
	tags := []string{"card-art"}

	huma.Register(api, huma.Operation{
		OperationID: "search-card-art",
		Method:      http.MethodGet,
		Path:        "/card-art/search",
		Summary:     "Search the card catalog for card art",
		Tags:        tags,
		Security:    cookieAuth,
	}, h.search)

	huma.Register(api, huma.Operation{
		OperationID: "select-account-card-art",
		Method:      http.MethodPut,
		Path:        "/accounts/{accountId}/card-art",
		Summary:     "Choose the card art shown for an account",
		Tags:        tags,
		Security:    cookieAuth,
	}, h.selectCardArt)
}

func (h *CardArtHandler) search(ctx context.Context, input *SearchCardArtInput) (*SearchCardArtOutput, error) {
	query := strings.TrimSpace(input.Query)
	if query == "" {
		return nil, huma.Error422UnprocessableEntity("query must not be blank")
	}

	cards, err := h.service.Search(ctx, query)
	if err != nil {
		return nil, problem(ctx, err)
	}

	return &SearchCardArtOutput{Body: CardArtSearchResults{Items: mapItems(cards, newCatalogCard)}}, nil
}

func (h *CardArtHandler) selectCardArt(ctx context.Context, input *SelectCardArtInput) (*AccountOutput, error) {
	id, err := parseID(input.AccountID)
	if err != nil {
		return nil, err
	}

	account, err := h.service.Select(ctx, UserFrom(ctx).ID, id, input.Body.CardID)
	if err != nil {
		return nil, problem(ctx, err)
	}

	return &AccountOutput{Body: newAccount(account)}, nil
}
