package plaid

import (
	"slices"
	"testing"

	plaidsdk "github.com/plaid/plaid-go/v50"
)

func TestInitialLinkTokenRequestsInvestmentsOptionally(t *testing.T) {
	p := &Provider{}

	request := p.initialLinkTokenRequest("user-id", "https://fin.example/connections/oauth")

	if want := []plaidsdk.Products{plaidsdk.ProductsTransactions}; !slices.Equal(request.Products, want) {
		t.Errorf("products = %v, want %v", request.Products, want)
	}

	if want := []plaidsdk.Products{plaidsdk.ProductsInvestments}; !slices.Equal(request.OptionalProducts, want) {
		t.Errorf("optional products = %v, want %v", request.OptionalProducts, want)
	}
}

func TestUpdateLinkTokenRequestCarriesAccessTokenWithoutExtraConsent(t *testing.T) {
	p := &Provider{}

	request := p.updateLinkTokenRequest("user-id", "access-token", "https://fin.example/connections/oauth")

	if request.AccessToken == nil || *request.AccessToken != "access-token" {
		t.Errorf("access token = %v, want access-token", request.AccessToken)
	}

	if len(request.AdditionalConsentedProducts) != 0 {
		t.Errorf("additional consented products = %v, want none", request.AdditionalConsentedProducts)
	}
}
