package plaid

import (
	"context"
	"time"

	plaidsdk "github.com/plaid/plaid-go/v50"

	"github.com/tahminator/fin/internal/domain"
	"github.com/tahminator/fin/internal/metrics"
	"github.com/tahminator/fin/internal/provider"
)

const transactionDaysRequested = 730

func (p *Provider) CreateLinkToken(ctx context.Context, creds domain.Credential, userID, redirectURI string) (string, error) {
	started := time.Now()

	request := p.linkTokenRequest(userID, redirectURI)
	request.Products = []plaidsdk.Products{plaidsdk.ProductsTransactions}
	request.Transactions = &plaidsdk.LinkTokenTransactions{DaysRequested: new(transactionDaysRequested)}

	token, err := p.createLinkToken(ctx, creds, request)

	p.observe(metrics.OperationCreateLinkToken, started, err)

	return token, err
}

func (p *Provider) CreateUpdateLinkToken(ctx context.Context, creds domain.Credential, userID, accessToken, redirectURI string) (string, error) {
	return p.createUpdateLinkToken(ctx, creds, p.updateLinkTokenRequest(userID, accessToken, redirectURI))
}

func (p *Provider) CreateAccountSelectionLinkToken(ctx context.Context, creds domain.Credential, userID, accessToken, redirectURI string) (string, error) {
	request := p.updateLinkTokenRequest(userID, accessToken, redirectURI)
	request.Update = &plaidsdk.LinkTokenCreateRequestUpdate{AccountSelectionEnabled: new(true)}

	return p.createUpdateLinkToken(ctx, creds, request)
}

func (p *Provider) Exchange(ctx context.Context, creds domain.Credential, publicToken string) (provider.LinkedItem, error) {
	started := time.Now()

	item, err := p.exchange(ctx, creds, publicToken)

	p.observe(metrics.OperationExchange, started, err)

	return item, err
}

func (p *Provider) Remove(ctx context.Context, creds domain.Credential, accessToken string) error {
	started := time.Now()

	err := p.remove(ctx, creds, accessToken)

	p.observe(metrics.OperationRemove, started, err)

	return err
}

func (p *Provider) ResetLogin(ctx context.Context, creds domain.Credential, accessToken string) error {
	c, err := p.client(creds)
	if err != nil {
		return err
	}

	_, err = c.Sandbox.ResetItemLogin(ctx, &plaidsdk.SandboxItemResetLoginRequest{AccessToken: accessToken})

	return mapError(err)
}

func (p *Provider) linkTokenRequest(userID, redirectURI string) *plaidsdk.LinkTokenCreateRequest {
	return &plaidsdk.LinkTokenCreateRequest{
		ClientName:   clientName,
		Language:     "en",
		CountryCodes: p.countryCodes,
		User:         &plaidsdk.LinkTokenCreateRequestUser{ClientUserID: userID},
		RedirectURI:  &redirectURI,
	}
}

func (p *Provider) updateLinkTokenRequest(userID, accessToken, redirectURI string) *plaidsdk.LinkTokenCreateRequest {
	request := p.linkTokenRequest(userID, redirectURI)
	request.AccessToken = &accessToken

	return request
}

func (p *Provider) createUpdateLinkToken(ctx context.Context, creds domain.Credential, request *plaidsdk.LinkTokenCreateRequest) (string, error) {
	started := time.Now()

	token, err := p.createLinkToken(ctx, creds, request)

	p.observe(metrics.OperationCreateUpdateLinkToken, started, err)

	return token, err
}

func (p *Provider) createLinkToken(ctx context.Context, creds domain.Credential, request *plaidsdk.LinkTokenCreateRequest) (string, error) {
	c, err := p.client(creds)
	if err != nil {
		return "", err
	}

	response, err := c.Link.CreateToken(ctx, request)
	if err != nil {
		return "", mapError(err)
	}

	return response.LinkToken, nil
}

func (p *Provider) exchange(ctx context.Context, creds domain.Credential, publicToken string) (provider.LinkedItem, error) {
	c, err := p.client(creds)
	if err != nil {
		return provider.LinkedItem{}, err
	}

	exchanged, err := c.Item.ExchangePublicToken(ctx, &plaidsdk.ItemPublicTokenExchangeRequest{PublicToken: publicToken})
	if err != nil {
		return provider.LinkedItem{}, mapError(err)
	}

	details, err := c.Item.Get(ctx, &plaidsdk.ItemGetRequest{AccessToken: exchanged.AccessToken})
	if err != nil {
		return provider.LinkedItem{}, mapError(err)
	}

	item := provider.LinkedItem{
		ExternalID:  exchanged.ItemID,
		AccessToken: exchanged.AccessToken,
	}

	if details.Item != nil {
		item.InstitutionID = details.Item.InstitutionID
		item.InstitutionName = details.Item.InstitutionName
	}

	return item, nil
}

func (p *Provider) remove(ctx context.Context, creds domain.Credential, accessToken string) error {
	c, err := p.client(creds)
	if err != nil {
		return err
	}

	_, err = c.Item.Remove(ctx, &plaidsdk.ItemRemoveRequest{AccessToken: accessToken})

	return mapError(err)
}
