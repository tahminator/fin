package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/tahminator/fin/internal/domain"
	"github.com/tahminator/fin/internal/identity"
	"github.com/tahminator/fin/internal/metrics"
)

const sessionTTL = 12 * time.Hour

var errExchangeFailed = errors.New("exchange failed")

type authFixture struct {
	user     domain.User
	claims   identity.Claims
	users    *fakeUsers
	sessions *fakeSessions
	identity *fakeIdentity
	recorder *loginRecorder
	service  *AuthService

	setGoogleSubCalls int
	createdUserID     uuid.UUID
	createdHash       []byte
	createdExpiresAt  time.Time
	sessionsCreated   int
}

func newAuthFixture(t *testing.T) *authFixture {
	t.Helper()

	f := &authFixture{
		user: domain.User{
			ID:    uuid.New(),
			Email: "alice@example.com",
		},
		claims: identity.Claims{
			Subject:       "google-subject-1",
			Email:         "alice@example.com",
			EmailVerified: true,
			Name:          new("Alice"),
		},
		recorder: &loginRecorder{},
	}

	f.users = &fakeUsers{
		findByEmail: func(_ context.Context, email string) (domain.User, error) {
			if !strings.EqualFold(email, f.user.Email) {
				return domain.User{}, domain.ErrNotFound
			}

			return f.user, nil
		},
		findByID: func(_ context.Context, id uuid.UUID) (domain.User, error) {
			if id != f.user.ID {
				return domain.User{}, domain.ErrNotFound
			}

			return f.user, nil
		},
		setGoogleSub: func(_ context.Context, id uuid.UUID, googleSub string, name *string) error {
			f.setGoogleSubCalls++

			if id != f.user.ID {
				t.Errorf("SetGoogleSub id = %v, want %v", id, f.user.ID)
			}

			if googleSub != f.claims.Subject {
				t.Errorf("SetGoogleSub googleSub = %q, want %q", googleSub, f.claims.Subject)
			}

			if name != f.claims.Name {
				t.Errorf("SetGoogleSub name = %v, want %v", name, f.claims.Name)
			}

			return nil
		},
	}

	f.sessions = &fakeSessions{
		create: func(_ context.Context, userID uuid.UUID, tokenHash []byte, expiresAt time.Time) error {
			f.sessionsCreated++
			f.createdUserID = userID
			f.createdHash = tokenHash
			f.createdExpiresAt = expiresAt

			return nil
		},
		deleteExpired: func(context.Context) error {
			return nil
		},
	}

	f.identity = &fakeIdentity{
		exchange: func(context.Context, string, string, string) (identity.Claims, error) {
			return f.claims, nil
		},
	}

	f.service = NewAuthService(f.users, f.sessions, f.identity, f.recorder, sessionTTL)
	f.service.now = fixedClock

	return f
}

func validCallback() LoginCallback {
	return LoginCallback{
		Code:  "auth-code",
		State: "state-1",
		Pending: PendingLogin{
			State:    "state-1",
			Nonce:    "nonce-1",
			Verifier: "verifier-1",
		},
	}
}

func assertLoginOutcomes(t *testing.T, recorder *loginRecorder, want metrics.LoginOutcome) {
	t.Helper()

	if !slices.Equal(recorder.outcomes, []metrics.LoginOutcome{want}) {
		t.Errorf("LoginAttempted outcomes = %v, want [%v]", recorder.outcomes, want)
	}
}

func TestBeginLoginPassesPendingValuesToIdentityProvider(t *testing.T) {
	f := newAuthFixture(t)

	var gotState, gotNonce, gotVerifier string

	f.identity.authCodeURL = func(state, nonce, verifier string) string {
		gotState, gotNonce, gotVerifier = state, nonce, verifier

		return "https://accounts.example.test/auth"
	}

	start := f.service.BeginLogin()

	if start.URL != "https://accounts.example.test/auth" {
		t.Errorf("URL = %q, want identity provider URL", start.URL)
	}

	if start.Pending.State == "" || start.Pending.Nonce == "" || start.Pending.Verifier == "" {
		t.Fatalf("pending login has empty values: %+v", start.Pending)
	}

	if start.Pending.State == start.Pending.Nonce {
		t.Errorf("state and nonce must differ, both %q", start.Pending.State)
	}

	if gotState != start.Pending.State || gotNonce != start.Pending.Nonce || gotVerifier != start.Pending.Verifier {
		t.Errorf("AuthCodeURL got (%q, %q, %q), want %+v", gotState, gotNonce, gotVerifier, start.Pending)
	}
}

func TestCompleteLoginRejections(t *testing.T) {
	tests := []struct {
		name        string
		mutate      func(f *authFixture, callback *LoginCallback)
		wantErr     error
		wantOutcome metrics.LoginOutcome
	}{
		{
			name: "state mismatch",
			mutate: func(_ *authFixture, callback *LoginCallback) {
				callback.State = "forged-state"
			},
			wantErr:     ErrInvalidState,
			wantOutcome: metrics.LoginInvalidState,
		},
		{
			name: "empty pending state",
			mutate: func(_ *authFixture, callback *LoginCallback) {
				callback.State = ""
				callback.Pending.State = ""
			},
			wantErr:     ErrInvalidState,
			wantOutcome: metrics.LoginInvalidState,
		},
		{
			name: "unverified email",
			mutate: func(f *authFixture, _ *LoginCallback) {
				f.claims.EmailVerified = false
			},
			wantErr:     domain.ErrNotAllowed,
			wantOutcome: metrics.LoginNotAllowed,
		},
		{
			name: "email not on allow-list",
			mutate: func(f *authFixture, _ *LoginCallback) {
				f.claims.Email = "mallory@example.com"
			},
			wantErr:     domain.ErrNotAllowed,
			wantOutcome: metrics.LoginNotAllowed,
		},
		{
			name: "google subject differs from stored subject",
			mutate: func(f *authFixture, _ *LoginCallback) {
				f.user.GoogleSub = new("google-subject-other")
			},
			wantErr:     domain.ErrNotAllowed,
			wantOutcome: metrics.LoginNotAllowed,
		},
		{
			name: "identity exchange failure",
			mutate: func(f *authFixture, _ *LoginCallback) {
				f.identity.exchange = func(context.Context, string, string, string) (identity.Claims, error) {
					return identity.Claims{}, errExchangeFailed
				}
			},
			wantErr:     errExchangeFailed,
			wantOutcome: metrics.LoginError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newAuthFixture(t)
			callback := validCallback()

			tt.mutate(f, &callback)

			token, err := f.service.CompleteLogin(t.Context(), callback)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("CompleteLogin error = %v, want %v", err, tt.wantErr)
			}

			if token != "" {
				t.Errorf("token = %q, want empty", token)
			}

			if f.setGoogleSubCalls != 0 {
				t.Errorf("SetGoogleSub called %d times, want 0", f.setGoogleSubCalls)
			}

			if f.sessionsCreated != 0 {
				t.Errorf("sessions created = %d, want 0", f.sessionsCreated)
			}

			assertLoginOutcomes(t, f.recorder, tt.wantOutcome)
		})
	}
}

func TestCompleteLoginPassesCallbackToIdentityExchange(t *testing.T) {
	f := newAuthFixture(t)
	callback := validCallback()

	var gotCode, gotVerifier, gotNonce string

	f.identity.exchange = func(_ context.Context, code, verifier, nonce string) (identity.Claims, error) {
		gotCode, gotVerifier, gotNonce = code, verifier, nonce

		return f.claims, nil
	}

	if _, err := f.service.CompleteLogin(t.Context(), callback); err != nil {
		t.Fatalf("CompleteLogin: %v", err)
	}

	if gotCode != callback.Code || gotVerifier != callback.Pending.Verifier || gotNonce != callback.Pending.Nonce {
		t.Errorf("Exchange got (%q, %q, %q), want (%q, %q, %q)", gotCode, gotVerifier, gotNonce, callback.Code, callback.Pending.Verifier, callback.Pending.Nonce)
	}
}

func TestCompleteLoginMatchesAllowListedEmailCaseInsensitively(t *testing.T) {
	f := newAuthFixture(t)
	f.claims.Email = "Alice@Example.COM"

	var lookedUp []string

	allowListed := f.users.findByEmail
	f.users.findByEmail = func(ctx context.Context, email string) (domain.User, error) {
		lookedUp = append(lookedUp, email)

		return allowListed(ctx, email)
	}

	token, err := f.service.CompleteLogin(t.Context(), validCallback())
	if err != nil {
		t.Fatalf("CompleteLogin: %v", err)
	}

	if token == "" {
		t.Error("token is empty")
	}

	if !slices.Equal(lookedUp, []string{"Alice@Example.COM"}) {
		t.Errorf("FindByEmail calls = %q, want the Google email as given", lookedUp)
	}

	if f.createdUserID != f.user.ID {
		t.Errorf("session user = %v, want %v", f.createdUserID, f.user.ID)
	}

	assertLoginOutcomes(t, f.recorder, metrics.LoginSuccess)
}

func TestCompleteLoginStoresGoogleSubject(t *testing.T) {
	tests := []struct {
		name      string
		storedSub *string
	}{
		{name: "first login with no stored subject", storedSub: nil},
		{name: "returning login with matching subject", storedSub: new("google-subject-1")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newAuthFixture(t)
			f.user.GoogleSub = tt.storedSub

			if _, err := f.service.CompleteLogin(t.Context(), validCallback()); err != nil {
				t.Fatalf("CompleteLogin: %v", err)
			}

			if f.setGoogleSubCalls != 1 {
				t.Errorf("SetGoogleSub called %d times, want 1", f.setGoogleSubCalls)
			}

			assertLoginOutcomes(t, f.recorder, metrics.LoginSuccess)
		})
	}
}

func TestCompleteLoginStoresHashOfTokenNeverTheToken(t *testing.T) {
	f := newAuthFixture(t)

	token, err := f.service.CompleteLogin(t.Context(), validCallback())
	if err != nil {
		t.Fatalf("CompleteLogin: %v", err)
	}

	if f.sessionsCreated != 1 {
		t.Fatalf("sessions created = %d, want 1", f.sessionsCreated)
	}

	want := sha256.Sum256([]byte(token))

	if !bytes.Equal(f.createdHash, want[:]) {
		t.Errorf("stored token hash = %x, want sha256(token) = %x", f.createdHash, want)
	}

	if bytes.Contains(f.createdHash, []byte(token)) {
		t.Error("stored token hash contains the raw token")
	}

	if f.createdUserID != f.user.ID {
		t.Errorf("session user = %v, want %v", f.createdUserID, f.user.ID)
	}

	if !f.createdExpiresAt.Equal(fixedNow.Add(sessionTTL)) {
		t.Errorf("session expires at %v, want %v", f.createdExpiresAt, fixedNow.Add(sessionTTL))
	}
}

func TestCompleteLoginIssuesDistinctTokens(t *testing.T) {
	f := newAuthFixture(t)

	first, err := f.service.CompleteLogin(t.Context(), validCallback())
	if err != nil {
		t.Fatalf("first CompleteLogin: %v", err)
	}

	second, err := f.service.CompleteLogin(t.Context(), validCallback())
	if err != nil {
		t.Fatalf("second CompleteLogin: %v", err)
	}

	if first == second {
		t.Errorf("both logins returned token %q", first)
	}
}

func TestResolveSessionUnauthenticated(t *testing.T) {
	tests := []struct {
		name    string
		token   string
		session func(f *authFixture) (domain.Session, error)
	}{
		{
			name:  "empty token",
			token: "",
		},
		{
			name:  "session not found",
			token: "cookie-token",
			session: func(*authFixture) (domain.Session, error) {
				return domain.Session{}, domain.ErrNotFound
			},
		},
		{
			name:  "session expires exactly now",
			token: "cookie-token",
			session: func(f *authFixture) (domain.Session, error) {
				return domain.Session{UserID: f.user.ID, ExpiresAt: fixedNow}, nil
			},
		},
		{
			name:  "session expired in the past",
			token: "cookie-token",
			session: func(f *authFixture) (domain.Session, error) {
				return domain.Session{UserID: f.user.ID, ExpiresAt: fixedNow.Add(-time.Minute)}, nil
			},
		},
		{
			name:  "session user no longer exists",
			token: "cookie-token",
			session: func(*authFixture) (domain.Session, error) {
				return domain.Session{UserID: uuid.New(), ExpiresAt: fixedNow.Add(time.Hour)}, nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newAuthFixture(t)

			f.sessions.findValidByTokenHash = func(context.Context, []byte) (domain.Session, error) {
				return tt.session(f)
			}

			user, err := f.service.ResolveSession(t.Context(), tt.token)
			if !errors.Is(err, ErrUnauthenticated) {
				t.Fatalf("ResolveSession error = %v, want %v", err, ErrUnauthenticated)
			}

			if user != (domain.User{}) {
				t.Errorf("user = %+v, want zero value", user)
			}
		})
	}
}

func TestResolveSessionReturnsSessionUser(t *testing.T) {
	f := newAuthFixture(t)
	token := "cookie-token"

	var gotHash []byte

	f.sessions.findValidByTokenHash = func(_ context.Context, tokenHash []byte) (domain.Session, error) {
		gotHash = tokenHash

		return domain.Session{UserID: f.user.ID, ExpiresAt: fixedNow.Add(time.Second)}, nil
	}

	user, err := f.service.ResolveSession(t.Context(), token)
	if err != nil {
		t.Fatalf("ResolveSession: %v", err)
	}

	if user.ID != f.user.ID {
		t.Errorf("user = %v, want %v", user.ID, f.user.ID)
	}

	want := sha256.Sum256([]byte(token))

	if !bytes.Equal(gotHash, want[:]) {
		t.Errorf("looked up hash %x, want %x", gotHash, want)
	}
}

func TestLogoutDeletesSessionByTokenHash(t *testing.T) {
	f := newAuthFixture(t)
	token := "cookie-token"

	var deleted [][]byte

	f.sessions.delete = func(_ context.Context, tokenHash []byte) error {
		deleted = append(deleted, tokenHash)

		return nil
	}

	if err := f.service.Logout(t.Context(), token); err != nil {
		t.Fatalf("Logout: %v", err)
	}

	want := sha256.Sum256([]byte(token))

	if len(deleted) != 1 || !bytes.Equal(deleted[0], want[:]) {
		t.Errorf("deleted hashes = %x, want [%x]", deleted, want)
	}
}

func TestLogoutWithoutTokenDeletesNothing(t *testing.T) {
	f := newAuthFixture(t)

	if err := f.service.Logout(t.Context(), ""); err != nil {
		t.Fatalf("Logout: %v", err)
	}
}
