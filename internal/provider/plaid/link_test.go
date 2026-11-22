package plaid

import (
	"slices"
	"testing"

	plaidsdk "github.com/plaid/plaid-go/v50"
)

func TestInitialLinkTokenRequestsInvestments(t *testing.T) {
	p := &Provider{}

	request := p.initialLinkTokenRequest("user-id", "https://fin.example/connections/oauth")

	want := []plaidsdk.Products{plaidsdk.ProductsTransactions, plaidsdk.ProductsInvestments}
	if !slices.Equal(request.Products, want) {
		t.Errorf("products = %v, want %v", request.Products, want)
	}
}

func TestUpdateLinkTokenRequestsAdditionalInvestmentsConsent(t *testing.T) {
	p := &Provider{}

	request := p.updateLinkTokenRequest("user-id", "access-token", "https://fin.example/connections/oauth")

	want := []plaidsdk.Products{plaidsdk.ProductsInvestments}
	if !slices.Equal(request.AdditionalConsentedProducts, want) {
		t.Errorf(
			"additional consented products = %v, want %v",
			request.AdditionalConsentedProducts,
			want,
		)
	}

	if request.AccessToken == nil || *request.AccessToken != "access-token" {
		t.Errorf("access token = %v, want access-token", request.AccessToken)
	}
}
