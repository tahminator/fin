package domain

import (
	"errors"
	"fmt"
	"time"
)

var ErrThrottled = errors.New("throttled")

type RateLimitScope string

const (
	RateLimitScopeSync RateLimitScope = "sync"
)

func (s RateLimitScope) Valid() bool {
	return s == RateLimitScopeSync
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
