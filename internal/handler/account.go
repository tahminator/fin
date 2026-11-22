package handler

import (
	"context"
	"net/http"
	"uuid"

	"github.com/danielgtaylor/huma/v2"

	"github.com/tahminator/fin/internal/domain"
	"github.com/tahminator/fin/internal/repository"
)

type Account struct {
	ID                 uuid.UUID             `json:"id"`
	ConnectionID       uuid.UUID             `json:"connectionId"`
	InstitutionName    *string               `json:"institutionName"`
	InstitutionLogoURL *string               `json:"institutionLogoUrl"`
	Name               string                `json:"name"`
	OfficialName       *string               `json:"officialName"`
	Mask               *string               `json:"mask"`
	Type               domain.AccountType    `json:"type" enum:"depository,credit,loan,investment,other"`
	Subtype            *string               `json:"subtype"`
	CurrentBalance     *string               `json:"currentBalance" doc:"Decimal string"`
	AvailableBalance   *string               `json:"availableBalance" doc:"Decimal string"`
	Currency           *string               `json:"currency"`
	CardArtID          *string               `json:"cardArtId" doc:"Catalog ID of the card shown for this account"`
	CardArtURL         *string               `json:"cardArtUrl" doc:"Card image URL; null shows the default icon"`
	CardArtSource      *domain.CardArtSource `json:"cardArtSource" enum:"matched,selected" doc:"null until the card has been matched"`
}

func newAccount(account domain.Account) Account {
	return Account{
		ID:                 account.ID,
		ConnectionID:       account.ConnectionID,
		InstitutionName:    account.InstitutionName,
		InstitutionLogoURL: institutionLogoURL(account.InstitutionURL),
		Name:               account.Name,
		OfficialName:       account.OfficialName,
		Mask:               account.Mask,
		Type:               account.Type,
		Subtype:            account.Subtype,
		CurrentBalance:     account.CurrentBalance,
		AvailableBalance:   account.AvailableBalance,
		Currency:           account.Currency,
		CardArtID:          account.CardArtID,
		CardArtURL:         account.CardArtURL,
		CardArtSource:      account.CardArtSource,
	}
}

type ListAccountsInput struct {
	PageParams
	ConnectionID string               `query:"connectionId" format:"uuid" doc:"Only accounts of this connection"`
	ExcludeType  []domain.AccountType `query:"excludeType,explode" enum:"depository,credit,loan,investment,other" doc:"Leave out accounts of these types"`
}

type AccountHandler struct {
	accounts repository.AccountRepository
}

func NewAccountHandler(accounts repository.AccountRepository) *AccountHandler {
	return &AccountHandler{accounts: accounts}
}

func (h *AccountHandler) Register(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "list-accounts",
		Method:      http.MethodGet,
		Path:        "/accounts",
		Summary:     "List accounts",
		Tags:        []string{"accounts"},
		Security:    cookieAuth,
	}, h.list)
}

func (h *AccountHandler) list(ctx context.Context, input *ListAccountsInput) (*PageOutput[Account], error) {
	filter := repository.AccountFilter{ExcludeTypes: input.ExcludeType}

	if input.ConnectionID != "" {
		connectionID, err := uuid.Parse(input.ConnectionID)
		if err != nil {
			return nil, huma.Error422UnprocessableEntity("connectionId must be a UUID")
		}

		filter.ConnectionID = &connectionID
	}

	accounts, total, err := h.accounts.ListByUser(ctx, UserFrom(ctx).ID, filter, input.Limit(), input.Offset())
	if err != nil {
		return nil, problem(ctx, err)
	}

	return &PageOutput[Account]{Body: NewPage(mapItems(accounts, newAccount), input.PageParams, total)}, nil
}
