package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"golang.org/x/oauth2"

	"github.com/tahminator/fin/internal/domain"
	"github.com/tahminator/fin/internal/identity"
	"github.com/tahminator/fin/internal/metrics"
	"github.com/tahminator/fin/internal/repository"
)

const tokenBytes = 32

var (
	ErrInvalidState    = errors.New("login state mismatch")
	ErrUnauthenticated = errors.New("unauthenticated")
)

type PendingLogin struct {
	State    string
	Nonce    string
	Verifier string
}

type LoginStart struct {
	Pending PendingLogin
	URL     string
}

type LoginCallback struct {
	Code    string
	State   string
	Pending PendingLogin
}

type AuthService struct {
	users      repository.UserRepository
	sessions   repository.SessionRepository
	identity   identity.IdentityProvider
	recorder   metrics.Recorder
	sessionTTL time.Duration
	now        func() time.Time
}

func NewAuthService(
	users repository.UserRepository,
	sessions repository.SessionRepository,
	identityProvider identity.IdentityProvider,
	recorder metrics.Recorder,
	sessionTTL time.Duration,
) *AuthService {
	return &AuthService{
		users:      users,
		sessions:   sessions,
		identity:   identityProvider,
		recorder:   recorder,
		sessionTTL: sessionTTL,
		now:        time.Now,
	}
}

func (s *AuthService) SessionTTL() time.Duration {
	return s.sessionTTL
}

func (s *AuthService) BeginLogin() LoginStart {
	pending := PendingLogin{
		State:    randomToken(),
		Nonce:    randomToken(),
		Verifier: oauth2.GenerateVerifier(),
	}

	return LoginStart{
		Pending: pending,
		URL:     s.identity.AuthCodeURL(pending.State, pending.Nonce, pending.Verifier),
	}
}

func (s *AuthService) CompleteLogin(ctx context.Context, callback LoginCallback) (string, error) {
	token, err := s.completeLogin(ctx, callback)

	s.recorder.LoginAttempted(loginOutcomeOf(err))

	return token, err
}

func (s *AuthService) completeLogin(ctx context.Context, callback LoginCallback) (string, error) {
	if callback.Pending.State == "" || callback.State != callback.Pending.State {
		return "", ErrInvalidState
	}

	claims, err := s.identity.Exchange(ctx, callback.Code, callback.Pending.Verifier, callback.Pending.Nonce)
	if err != nil {
		return "", err
	}

	if !claims.EmailVerified {
		return "", fmt.Errorf("%w: email %s is not verified", domain.ErrNotAllowed, claims.Email)
	}

	user, err := s.users.FindByEmail(ctx, claims.Email)
	if errors.Is(err, domain.ErrNotFound) {
		return "", fmt.Errorf("%w: %s is not on the allow-list", domain.ErrNotAllowed, claims.Email)
	}

	if err != nil {
		return "", fmt.Errorf("find user: %w", err)
	}

	if user.GoogleSub != nil && *user.GoogleSub != claims.Subject {
		return "", fmt.Errorf("%w: google account does not match %s", domain.ErrNotAllowed, claims.Email)
	}

	if err := s.users.SetGoogleSub(ctx, user.ID, claims.Subject, claims.Name); err != nil {
		return "", fmt.Errorf("store google subject: %w", err)
	}

	if err := s.sessions.DeleteExpired(ctx); err != nil {
		return "", fmt.Errorf("delete expired sessions: %w", err)
	}

	token := randomToken()

	if err := s.sessions.Create(ctx, user.ID, HashToken(token), s.now().Add(s.sessionTTL)); err != nil {
		return "", fmt.Errorf("create session: %w", err)
	}

	return token, nil
}

func (s *AuthService) ResolveSession(ctx context.Context, token string) (domain.User, error) {
	if token == "" {
		return domain.User{}, ErrUnauthenticated
	}

	session, err := s.sessions.FindValidByTokenHash(ctx, HashToken(token))
	if errors.Is(err, domain.ErrNotFound) {
		return domain.User{}, ErrUnauthenticated
	}

	if err != nil {
		return domain.User{}, fmt.Errorf("find session: %w", err)
	}

	if !session.ExpiresAt.After(s.now()) {
		return domain.User{}, ErrUnauthenticated
	}

	user, err := s.users.FindByID(ctx, session.UserID)
	if errors.Is(err, domain.ErrNotFound) {
		return domain.User{}, ErrUnauthenticated
	}

	if err != nil {
		return domain.User{}, fmt.Errorf("find user: %w", err)
	}

	return user, nil
}

func (s *AuthService) Logout(ctx context.Context, token string) error {
	if token == "" {
		return nil
	}

	return s.sessions.Delete(ctx, HashToken(token))
}

func HashToken(token string) []byte {
	sum := sha256.Sum256([]byte(token))

	return sum[:]
}

func randomToken() string {
	buf := make([]byte, tokenBytes)

	_, _ = rand.Read(buf)

	return base64.RawURLEncoding.EncodeToString(buf)
}

func loginOutcomeOf(err error) metrics.LoginOutcome {
	switch {
	case err == nil:
		return metrics.LoginSuccess
	case errors.Is(err, ErrInvalidState):
		return metrics.LoginInvalidState
	case errors.Is(err, domain.ErrNotAllowed):
		return metrics.LoginNotAllowed
	}

	return metrics.LoginError
}
