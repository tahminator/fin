package handler

import (
	"context"
	"net/http"
	"net/url"
	"time"
	"uuid"

	"github.com/danielgtaylor/huma/v2"

	"github.com/tahminator/fin/internal/domain"
	"github.com/tahminator/fin/internal/oauth"
	"github.com/tahminator/fin/internal/service"
)

type OAuthConsentHandler struct {
	service *service.OAuthService
}

func NewOAuthConsentHandler(oauthService *service.OAuthService) *OAuthConsentHandler {
	return &OAuthConsentHandler{service: oauthService}
}

type ConsentRequest struct {
	ClientName   string         `json:"clientName"`
	ClientHost   string         `json:"clientHost" doc:"Host that identifies the app: its client ID URL's host when verified"`
	Verified     bool           `json:"verified" doc:"True when the app proved its identity with a client metadata document on its own domain"`
	LogoURL      *string        `json:"logoUrl"`
	RedirectHost string         `json:"redirectHost" doc:"Where the user returns after deciding"`
	Scopes       []domain.Scope `json:"scopes" enum:"fin:read,fin:sync" nullable:"false"`
	ExpiresAt    time.Time      `json:"expiresAt"`
}

type ConsentDecisionBody struct {
	Scopes []domain.Scope `json:"scopes" enum:"fin:read,fin:sync" nullable:"false" minItems:"1" doc:"Scopes the user approves; must include fin:read and be a subset of the request"`
}

type ConsentRedirect struct {
	RedirectTo string `json:"redirectTo" doc:"URL to send the browser to"`
}

type OAuthGrant struct {
	ID         uuid.UUID      `json:"id"`
	ClientName string         `json:"clientName"`
	ClientHost string         `json:"clientHost"`
	Verified   bool           `json:"verified"`
	Scopes     []domain.Scope `json:"scopes" enum:"fin:read,fin:sync" nullable:"false"`
	CreatedAt  time.Time      `json:"createdAt"`
	LastUsedAt *time.Time     `json:"lastUsedAt"`
	ExpiresAt  time.Time      `json:"expiresAt"`
}

type ConsentPathInput struct {
	RequestID string `path:"requestId" minLength:"43" maxLength:"43" pattern:"^[A-Za-z0-9_-]+$"`
}

type ApproveConsentInput struct {
	RequestID string `path:"requestId" minLength:"43" maxLength:"43" pattern:"^[A-Za-z0-9_-]+$"`
	Body      ConsentDecisionBody
}

type ConsentRequestOutput struct {
	Body ConsentRequest
}

type ConsentRedirectOutput struct {
	Body ConsentRedirect
}

type ListGrantsInput struct {
	PageParams
}

type GrantPathInput struct {
	GrantID string `path:"grantId" format:"uuid"`
}

func (h *OAuthConsentHandler) Register(api huma.API) {
	tags := []string{"oauth"}

	huma.Register(api, huma.Operation{
		OperationID: "get-oauth-request",
		Method:      http.MethodGet,
		Path:        "/oauth/requests/{requestId}",
		Summary:     "Describe a pending app authorization request for the consent page",
		Tags:        tags,
		Security:    cookieAuth,
	}, h.get)

	huma.Register(api, huma.Operation{
		OperationID: "approve-oauth-request",
		Method:      http.MethodPost,
		Path:        "/oauth/requests/{requestId}/approve",
		Summary:     "Approve an app authorization request",
		Tags:        tags,
		Security:    cookieAuth,
	}, h.approve)

	huma.Register(api, huma.Operation{
		OperationID: "deny-oauth-request",
		Method:      http.MethodPost,
		Path:        "/oauth/requests/{requestId}/deny",
		Summary:     "Deny an app authorization request",
		Tags:        tags,
		Security:    cookieAuth,
	}, h.deny)

	huma.Register(api, huma.Operation{
		OperationID: "list-oauth-grants",
		Method:      http.MethodGet,
		Path:        "/oauth/grants",
		Summary:     "Apps the user has connected",
		Tags:        tags,
		Security:    cookieAuth,
	}, h.listGrants)

	huma.Register(api, huma.Operation{
		OperationID:   "revoke-oauth-grant",
		Method:        http.MethodDelete,
		Path:          "/oauth/grants/{grantId}",
		Summary:       "Disconnect an app and revoke all of its tokens",
		Tags:          tags,
		Security:      cookieAuth,
		DefaultStatus: http.StatusNoContent,
	}, h.revokeGrant)
}

func (h *OAuthConsentHandler) get(ctx context.Context, input *ConsentPathInput) (*ConsentRequestOutput, error) {
	request, err := h.service.PendingRequest(ctx, input.RequestID)
	if err != nil {
		return nil, problem(ctx, err)
	}

	return &ConsentRequestOutput{Body: ConsentRequest{
		ClientName:   request.Client.ClientName,
		ClientHost:   clientHost(request.Client),
		Verified:     request.Client.Source == domain.ClientSourceMetadataDocument,
		LogoURL:      verifiedLogo(request.Client),
		RedirectHost: redirectHost(request.RedirectURI),
		Scopes:       request.Scopes,
		ExpiresAt:    request.ExpiresAt,
	}}, nil
}

func (h *OAuthConsentHandler) approve(ctx context.Context, input *ApproveConsentInput) (*ConsentRedirectOutput, error) {
	redirectTo, err := h.service.Approve(ctx, UserFrom(ctx).ID, input.RequestID, input.Body.Scopes)
	if err != nil {
		return nil, problem(ctx, err)
	}

	return &ConsentRedirectOutput{Body: ConsentRedirect{RedirectTo: redirectTo}}, nil
}

func (h *OAuthConsentHandler) deny(ctx context.Context, input *ConsentPathInput) (*ConsentRedirectOutput, error) {
	redirectTo, err := h.service.Deny(ctx, input.RequestID)
	if err != nil {
		return nil, problem(ctx, err)
	}

	return &ConsentRedirectOutput{Body: ConsentRedirect{RedirectTo: redirectTo}}, nil
}

func (h *OAuthConsentHandler) listGrants(ctx context.Context, input *ListGrantsInput) (*PageOutput[OAuthGrant], error) {
	grants, total, err := h.service.ListGrants(ctx, UserFrom(ctx).ID, input.Limit(), input.Offset())
	if err != nil {
		return nil, problem(ctx, err)
	}

	return &PageOutput[OAuthGrant]{Body: NewPage(mapItems(grants, newOAuthGrant), input.PageParams, total)}, nil
}

func (h *OAuthConsentHandler) revokeGrant(ctx context.Context, input *GrantPathInput) (*struct{}, error) {
	id, err := parseID(input.GrantID)
	if err != nil {
		return nil, err
	}

	if err := h.service.RevokeGrant(ctx, UserFrom(ctx).ID, id); err != nil {
		return nil, problem(ctx, err)
	}

	return nil, nil
}

func newOAuthGrant(grant domain.OAuthGrant) OAuthGrant {
	client := domain.OAuthClient{ID: grant.ClientID, Source: grant.ClientSource}

	return OAuthGrant{
		ID:         grant.ID,
		ClientName: grant.ClientName,
		ClientHost: clientHost(client),
		Verified:   grant.ClientSource == domain.ClientSourceMetadataDocument,
		Scopes:     parsedScopes(grant.Scope),
		CreatedAt:  grant.CreatedAt,
		LastUsedAt: grant.LastUsedAt,
		ExpiresAt:  grant.ExpiresAt,
	}
}

func clientHost(client domain.OAuthClient) string {
	if client.Source == domain.ClientSourceMetadataDocument {
		return redirectHost(client.ID)
	}

	if client.ClientURI != nil {
		return redirectHost(*client.ClientURI)
	}

	return ""
}

func verifiedLogo(client domain.OAuthClient) *string {
	if client.Source != domain.ClientSourceMetadataDocument {
		return nil
	}

	return client.LogoURI
}

func redirectHost(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil {
		return ""
	}

	if parsed.Host != "" {
		return parsed.Host
	}

	return parsed.Scheme + ":"
}

func parsedScopes(raw string) []domain.Scope {
	scopes, err := oauth.ParseScopes(raw)
	if err != nil || scopes == nil {
		return []domain.Scope{}
	}

	return scopes
}
