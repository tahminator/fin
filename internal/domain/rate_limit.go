package domain

import (
	"errors"
	"fmt"
	"time"
)

var ErrThrottled = errors.New("throttled")

type RateLimitScope string

const (
	RateLimitScopeSync           RateLimitScope = "sync"
	RateLimitScopeOAuthRegister  RateLimitScope = "oauth_register"
	RateLimitScopeOAuthToken     RateLimitScope = "oauth_token"
	RateLimitScopeOAuthAuthorize RateLimitScope = "oauth_authorize"
	RateLimitScopeMCPGrant       RateLimitScope = "mcp_grant"
)

func (s RateLimitScope) Valid() bool {
	switch s {
	case RateLimitScopeSync, RateLimitScopeOAuthRegister, RateLimitScopeOAuthToken, RateLimitScopeOAuthAuthorize, RateLimitScopeMCPGrant:
		return true
	}

	return false
}

type RateLimit struct {
	Scope  RateLimitScope
	Max    int
	Window time.Duration
}

func (l RateLimit) Enabled() bool {
	return l.Window > 0 && l.Max > 0
}

type RateLimitHit struct {
	Hits       int
	RetryAfter time.Duration
}

type RateLimitDecision struct {
	Allowed    bool
	RetryAfter time.Duration
}

type ThrottledError struct {
	RetryAfter time.Duration
}

func (e *ThrottledError) Error() string {
	return fmt.Sprintf("%s: try again in %s", ErrThrottled, e.RetryAfter.Round(time.Second))
}

func (e *ThrottledError) Is(target error) bool {
	return target == ErrThrottled
}
