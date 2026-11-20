package config

import (
	"errors"
	"fmt"
	"time"

	"github.com/robfig/cron/v3"
)

type Environment string

const (
	Development Environment = "development"
	Production  Environment = "production"
)

type Config struct {
	AppEnv            Environment
	HTTPAddr          string
	OpsAddr           string
	DatabaseURL       string
	PublicURL         string
	StaticDir         string
	CookieDomain      string
	CookieSecure      bool
	SessionTTL        time.Duration
	SyncSchedule      cron.Schedule
	GoogleClientID    string
	GoogleSecret      string
	SecretKey         []byte
	PlaidCountryCodes []string
	Version           string
}

func NewFromEnv() (Config, error) {
	env := &envReader{}

	cfg := Config{
		AppEnv:            Environment(env.optional("APP_ENV", string(Production))),
		HTTPAddr:          env.optional("HTTP_ADDR", ":8080"),
		OpsAddr:           env.optional("OPS_ADDR", ":9090"),
		DatabaseURL:       env.required("DATABASE_URL"),
		PublicURL:         env.required("PUBLIC_URL"),
		StaticDir:         env.optional("STATIC_DIR", ""),
		CookieDomain:      env.optional("COOKIE_DOMAIN", ""),
		CookieSecure:      env.boolean("COOKIE_SECURE", true),
		SessionTTL:        env.duration("SESSION_TTL", 30*24*time.Hour),
		SyncSchedule:      env.schedule("SYNC_CRON", "0 */6 * * *"),
		GoogleClientID:    env.required("GOOGLE_CLIENT_ID"),
		GoogleSecret:      env.required("GOOGLE_CLIENT_SECRET"),
		SecretKey:         []byte(env.required("SECRET_KEY")),
		PlaidCountryCodes: env.list("PLAID_COUNTRY_CODES", []string{"US"}),
		Version:           env.optional("VERSION", ""),
	}

	env.errs = append(env.errs, cfg.validate()...)

	if err := errors.Join(env.errs...); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func (c Config) validate() []error {
	var errs []error

	if c.AppEnv != Development && c.AppEnv != Production {
		errs = append(errs, fmt.Errorf("APP_ENV must be %q or %q", Development, Production))
	}

	if len(c.SecretKey) != 32 {
		errs = append(errs, errors.New("SECRET_KEY must be exactly 32 characters"))
	}

	if c.AppEnv == Production && c.CookieDomain == "" {
		errs = append(errs, errors.New("COOKIE_DOMAIN is required in production"))
	}

	if c.AppEnv == Production && !c.CookieSecure {
		errs = append(errs, errors.New("COOKIE_SECURE must be true in production"))
	}

	return errs
}

func (c Config) GoogleRedirectURL() string {
	return c.PublicURL + "/auth/google/callback"
}

func (c Config) PlaidRedirectURI() string {
	return c.PublicURL + "/connections/oauth"
}
