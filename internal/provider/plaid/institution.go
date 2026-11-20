package plaid

import (
	"context"
	"time"

	plaidsdk "github.com/plaid/plaid-go/v50"

	"github.com/tahminator/fin/internal/domain"
	"github.com/tahminator/fin/internal/metrics"
	"github.com/tahminator/fin/internal/provider"
)

func (p *Provider) Institution(ctx context.Context, creds domain.Credential, institutionID string) (provider.InstitutionInfo, error) {
	started := time.Now()

	info, err := p.institution(ctx, creds, institutionID)

	p.observe(metrics.OperationInstitution, started, err)

	return info, err
}

func (p *Provider) institution(ctx context.Context, creds domain.Credential, institutionID string) (provider.InstitutionInfo, error) {
	c, err := p.client(creds)
	if err != nil {
		return provider.InstitutionInfo{}, err
	}

	response, err := c.Institutions.GetByID(ctx, &plaidsdk.InstitutionsGetByIDRequest{
		InstitutionID: institutionID,
		CountryCodes:  p.countryCodes,
		Options: &plaidsdk.InstitutionsGetByIDRequestOptions{
			IncludeOptionalMetadata: new(true),
		},
	})
	if err != nil {
		return provider.InstitutionInfo{}, mapError(err)
	}

	info := provider.InstitutionInfo{}

	if response.Institution != nil {
		info.URL = response.Institution.URL
	}

	return info, nil
}
