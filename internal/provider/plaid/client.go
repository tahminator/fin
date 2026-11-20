package plaid

import (
	"fmt"
	"net/http"
	"time"

	plaidsdk "github.com/plaid/plaid-go/v50"
	"github.com/plaid/plaid-go/v50/client"
	"github.com/plaid/plaid-go/v50/option"

	"github.com/tahminator/fin/internal/domain"
	"github.com/tahminator/fin/internal/metrics"
)

const clientName = "fin"

type Provider struct {
	httpClient   *http.Client
	recorder     metrics.Recorder
	countryCodes []plaidsdk.CountryCode
}

func New(httpClient *http.Client, recorder metrics.Recorder, countryCodes []string) *Provider {
	codes := make([]plaidsdk.CountryCode, len(countryCodes))

	for i, code := range countryCodes {
		codes[i] = plaidsdk.CountryCode(code)
	}

	return &Provider{
		httpClient:   httpClient,
		recorder:     recorder,
		countryCodes: codes,
	}
}

func (p *Provider) client(creds domain.Credential) (*client.Client, error) {
	baseURL, err := baseURLFor(creds.Environment)
	if err != nil {
		return nil, err
	}

	return client.NewClient(
		option.WithClientID(&creds.ClientID),
		option.WithSecret(creds.Secret),
		option.WithBaseURL(baseURL),
		option.WithHTTPClient(p.httpClient),
	), nil
}

func (p *Provider) observe(operation metrics.ProviderOperation, started time.Time, err error) {
	outcome := metrics.CallSuccess

	if err != nil {
		outcome = metrics.CallError
	}

	p.recorder.ProviderCalled(metrics.ProviderPlaid, operation, outcome, time.Since(started))
}

func baseURLFor(environment domain.ProviderEnvironment) (string, error) {
	switch environment {
	case domain.ProviderEnvironmentSandbox:
		return plaidsdk.Environments.Sandbox, nil
	case domain.ProviderEnvironmentProduction:
		return plaidsdk.Environments.Production, nil
	}

	return "", fmt.Errorf("unknown plaid environment %q", environment)
}
