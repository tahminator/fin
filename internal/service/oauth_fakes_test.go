package service

import (
	"bytes"
	"context"
	"time"
	"uuid"

	"github.com/tahminator/fin/internal/domain"
	"github.com/tahminator/fin/internal/metrics"
	"github.com/tahminator/fin/internal/oauth"
	"github.com/tahminator/fin/internal/repository"
)

type memoryOAuth struct {
	clients        map[string]domain.OAuthClient
	authorizations []*memoryAuthorization
	grants         map[uuid.UUID]domain.OAuthGrant
	tokens         []*memoryToken
	users          map[uuid.UUID]domain.User
	now            func() time.Time
}

type memoryAuthorization struct {
	domain.OAuthAuthorization
	requestHash   []byte
	codeHash      []byte
	codeExpiresAt time.Time
	consumed      bool
}

type memoryToken struct {
	repository.NewToken
	used bool
}

func newMemoryOAuth() *memoryOAuth {
	return &memoryOAuth{
		clients: map[string]domain.OAuthClient{},
		grants:  map[uuid.UUID]domain.OAuthGrant{},
		users:   map[uuid.UUID]domain.User{},
		now:     fixedClock,
	}
}

func (m *memoryOAuth) repos() OAuthRepositories {
	return OAuthRepositories{
		Clients:        memoryClients{m},
		Authorizations: memoryAuthorizations{m},
		Grants:         memoryGrants{m},
		Tokens:         memoryTokens{m},
	}
}

type memoryClients struct{ m *memoryOAuth }

func (c memoryClients) Find(_ context.Context, id string) (domain.OAuthClient, error) {
	client, ok := c.m.clients[id]
	if !ok {
		return domain.OAuthClient{}, domain.ErrNotFound
	}

	return client, nil
}

func (c memoryClients) Create(_ context.Context, client domain.OAuthClient) error {
	c.m.clients[client.ID] = client

	return nil
}

func (c memoryClients) Upsert(_ context.Context, client domain.OAuthClient) error {
	c.m.clients[client.ID] = client

	return nil
}

func (memoryClients) DeleteUnusedRegistered(context.Context, time.Time) error {
	return nil
}

type memoryAuthorizations struct{ m *memoryOAuth }

func (a memoryAuthorizations) Create(_ context.Context, authorization repository.NewAuthorization) error {
	a.m.authorizations = append(a.m.authorizations, &memoryAuthorization{
		OAuthAuthorization: domain.OAuthAuthorization{
			ID:            uuid.New(),
			ClientID:      authorization.ClientID,
			RedirectURI:   authorization.RedirectURI,
			Scope:         authorization.Scope,
			Resource:      authorization.Resource,
			State:         authorization.State,
			CodeChallenge: authorization.CodeChallenge,
			ExpiresAt:     authorization.ExpiresAt,
		},
		requestHash: authorization.RequestTokenHash,
	})

	return nil
}

func (a memoryAuthorizations) FindPending(_ context.Context, hash []byte) (domain.OAuthAuthorization, error) {
	for _, auth := range a.m.authorizations {
		if bytes.Equal(auth.requestHash, hash) && auth.UserID == nil && auth.ExpiresAt.After(a.m.now()) {
			return auth.OAuthAuthorization, nil
		}
	}

	return domain.OAuthAuthorization{}, domain.ErrNotFound
}

func (a memoryAuthorizations) Approve(_ context.Context, approval repository.AuthorizationApproval) error {
	for _, auth := range a.m.authorizations {
		if auth.ID == approval.ID && auth.UserID == nil {
			auth.UserID = &approval.UserID
			auth.Scope = approval.Scope
			auth.GrantID = &approval.GrantID
			auth.codeHash = approval.CodeHash
			auth.codeExpiresAt = approval.CodeExpiresAt

			return nil
		}
	}

	return domain.ErrNotFound
}

func (a memoryAuthorizations) DeletePending(_ context.Context, id uuid.UUID) error {
	for i, auth := range a.m.authorizations {
		if auth.ID == id && auth.UserID == nil {
			a.m.authorizations = append(a.m.authorizations[:i], a.m.authorizations[i+1:]...)

			return nil
		}
	}

	return domain.ErrNotFound
}

func (a memoryAuthorizations) ConsumeCode(_ context.Context, hash []byte) (domain.OAuthAuthorization, error) {
	for _, auth := range a.m.authorizations {
		if auth.codeHash != nil && bytes.Equal(auth.codeHash, hash) && !auth.consumed && auth.codeExpiresAt.After(a.m.now()) {
			auth.consumed = true

			return auth.OAuthAuthorization, nil
		}
	}

	return domain.OAuthAuthorization{}, domain.ErrNotFound
}

func (a memoryAuthorizations) FindConsumedGrantID(_ context.Context, hash []byte) (uuid.UUID, error) {
	for _, auth := range a.m.authorizations {
		if auth.codeHash != nil && bytes.Equal(auth.codeHash, hash) && auth.consumed && auth.GrantID != nil {
			return *auth.GrantID, nil
		}
	}

	return uuid.UUID{}, domain.ErrNotFound
}

func (memoryAuthorizations) DeleteExpired(context.Context) error {
	return nil
}

type memoryGrants struct{ m *memoryOAuth }

func (g memoryGrants) Upsert(_ context.Context, userID uuid.UUID, clientID, scope, resource string, expiresAt time.Time) (uuid.UUID, error) {
	for id, grant := range g.m.grants {
		if grant.UserID == userID && grant.ClientID == clientID {
			grant.Scope, grant.Resource, grant.ExpiresAt = scope, resource, expiresAt
			g.m.grants[id] = grant

			return id, nil
		}
	}

	id := uuid.New()
	g.m.grants[id] = domain.OAuthGrant{ID: id, UserID: userID, ClientID: clientID, Scope: scope, Resource: resource, ExpiresAt: expiresAt}

	return id, nil
}

func (g memoryGrants) ListByUser(_ context.Context, userID uuid.UUID, _, _ int) ([]domain.OAuthGrant, int, error) {
	var grants []domain.OAuthGrant

	for _, grant := range g.m.grants {
		if grant.UserID == userID {
			grants = append(grants, grant)
		}
	}

	return grants, len(grants), nil
}

func (g memoryGrants) Delete(ctx context.Context, userID, id uuid.UUID) error {
	grant, ok := g.m.grants[id]
	if !ok || grant.UserID != userID {
		return domain.ErrNotFound
	}

	return g.DeleteByID(ctx, id)
}

func (g memoryGrants) DeleteByID(_ context.Context, id uuid.UUID) error {
	delete(g.m.grants, id)

	kept := g.m.tokens[:0]

	for _, token := range g.m.tokens {
		if token.GrantID != id {
			kept = append(kept, token)
		}
	}

	g.m.tokens = kept

	return nil
}

func (memoryGrants) TouchLastUsed(context.Context, uuid.UUID) error {
	return nil
}

type memoryTokens struct{ m *memoryOAuth }

func (t memoryTokens) Create(_ context.Context, tokens ...repository.NewToken) error {
	for _, token := range tokens {
		t.m.tokens = append(t.m.tokens, &memoryToken{NewToken: token})
	}

	return nil
}

func (t memoryTokens) find(hash []byte) *memoryToken {
	for _, token := range t.m.tokens {
		if bytes.Equal(token.TokenHash, hash) {
			return token
		}
	}

	return nil
}

func (t memoryTokens) FindAccess(_ context.Context, hash []byte) (repository.AccessTokenRecord, error) {
	token := t.find(hash)
	if token == nil || token.Kind != domain.TokenKindAccess || !token.ExpiresAt.After(t.m.now()) {
		return repository.AccessTokenRecord{}, domain.ErrNotFound
	}

	grant, ok := t.m.grants[token.GrantID]
	if !ok || !grant.ExpiresAt.After(t.m.now()) {
		return repository.AccessTokenRecord{}, domain.ErrNotFound
	}

	return repository.AccessTokenRecord{
		User:       t.m.users[token.UserID],
		GrantID:    grant.ID,
		ClientID:   grant.ClientID,
		TokenScope: token.Scope,
		GrantScope: grant.Scope,
		Resource:   grant.Resource,
		ExpiresAt:  token.ExpiresAt,
	}, nil
}

func (t memoryTokens) ConsumeRefresh(_ context.Context, hash []byte) (repository.RefreshTokenRecord, error) {
	token := t.find(hash)
	if token == nil || token.Kind != domain.TokenKindRefresh || token.used {
		return repository.RefreshTokenRecord{}, domain.ErrNotFound
	}

	grant, ok := t.m.grants[token.GrantID]
	if !ok {
		return repository.RefreshTokenRecord{}, domain.ErrNotFound
	}

	token.used = true

	return repository.RefreshTokenRecord{
		GrantID:    grant.ID,
		UserID:     token.UserID,
		ClientID:   grant.ClientID,
		TokenScope: token.Scope,
		GrantScope: grant.Scope,
		Resource:   grant.Resource,
	}, nil
}

func (t memoryTokens) FindOwner(_ context.Context, hash []byte) (repository.TokenOwner, error) {
	token := t.find(hash)
	if token == nil {
		return repository.TokenOwner{}, domain.ErrNotFound
	}

	return repository.TokenOwner{GrantID: token.GrantID, ClientID: t.m.grants[token.GrantID].ClientID, Kind: token.Kind, Used: token.used}, nil
}

func (t memoryTokens) DeleteByHash(_ context.Context, hash []byte) error {
	kept := t.m.tokens[:0]

	for _, token := range t.m.tokens {
		if !bytes.Equal(token.TokenHash, hash) {
			kept = append(kept, token)
		}
	}

	t.m.tokens = kept

	return nil
}

func (memoryTokens) DeleteExpired(context.Context) error {
	return nil
}

type unusedFetcher struct{}

func (unusedFetcher) Fetch(context.Context, string) (oauth.FetchedClient, error) {
	return oauth.FetchedClient{}, oauth.NewError(oauth.ErrInvalidClient, "no network in tests")
}

type oauthRecorder struct {
	metrics.Noop
	tokens   []metrics.OAuthOutcome
	consents []metrics.ConsentDecision
}

func (r *oauthRecorder) OAuthTokenRequested(_ metrics.OAuthGrantType, outcome metrics.OAuthOutcome) {
	r.tokens = append(r.tokens, outcome)
}

func (r *oauthRecorder) OAuthConsentDecided(decision metrics.ConsentDecision) {
	r.consents = append(r.consents, decision)
}
