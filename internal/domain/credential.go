package domain

import (
	"time"
	"uuid"
)

type Provider string

const ProviderPlaid Provider = "plaid"

func (p Provider) Valid() bool {
	return p == ProviderPlaid
}

type ProviderEnvironment string

const (
	ProviderEnvironmentSandbox    ProviderEnvironment = "sandbox"
	ProviderEnvironmentProduction ProviderEnvironment = "production"
)

func (e ProviderEnvironment) Valid() bool {
	return e == ProviderEnvironmentSandbox || e == ProviderEnvironmentProduction
}

type Credential struct {
	UserID      uuid.UUID           `db:"user_id"`
	Provider    Provider            `db:"provider"`
	Environment ProviderEnvironment `db:"environment"`
	ClientID    string              `db:"client_id"`
	Secret      string              `db:"secret"`
	CreatedAt   time.Time           `db:"created_at"`
}
