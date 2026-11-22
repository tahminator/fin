package config

import (
	"errors"
	"fmt"
	"net/url"
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
	DatabaseURL       string
	PublicURL         string
	StaticDir         string
	CookieDomain      string
	CookieSecure      bool
	SessionTTL        time.Duration
	SyncSchedule      cron.Schedule
	SyncCooldown      time.Duration
	GoogleClientID    string
	GoogleSecret      string
	SecretKey         []byte
	PlaidCountryCodes []string
	Version           string
	MCP               MCPConfig
}

type MCPConfig struct {
	Enabled         bool
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
	GrantMaxAge     time.Duration
	AllowedOrigins  []string
}

func NewFromEnv() (Config, error) {
	env := &envReader{}

	cfg := Config{
		AppEnv:            Environment(env.optional("APP_ENV", string(Production))),
		HTTPAddr:          env.optional("HTTP_ADDR", ":8080"),
		DatabaseURL:       env.required("DATABASE_URL"),
		PublicURL:         env.required("PUBLIC_URL"),
		StaticDir:         env.optional("STATIC_DIR", ""),
		CookieDomain:      env.optional("COOKIE_DOMAIN", ""),
		CookieSecure:      env.boolean("COOKIE_SECURE", true),
		SessionTTL:        env.duration("SESSION_TTL", 30*24*time.Hour),
		SyncSchedule:      env.schedule("SYNC_CRON", "0 */6 * * *"),
		SyncCooldown:      env.duration("SYNC_COOLDOWN", 10*time.Minute),
		GoogleClientID:    env.required("GOOGLE_CLIENT_ID"),
		GoogleSecret:      env.required("GOOGLE_CLIENT_SECRET"),
		SecretKey:         []byte(env.required("SECRET_KEY")),
		PlaidCountryCodes: env.list("PLAID_COUNTRY_CODES", []string{"US"}),
		Version:           env.optional("VERSION", ""),
		MCP: MCPConfig{
			Enabled:         env.boolean("MCP_ENABLED", true),
			AccessTokenTTL:  env.duration("MCP_ACCESS_TOKEN_TTL", time.Hour),
			RefreshTokenTTL: env.duration("MCP_REFRESH_TOKEN_TTL", 30*24*time.Hour),
			GrantMaxAge:     env.duration("MCP_GRANT_MAX_AGE", 90*24*time.Hour),
			AllowedOrigins:  env.list("MCP_ALLOWED_ORIGINS", nil),
		},
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

	if c.SyncCooldown < 0 {
		errs = append(errs, errors.New("SYNC_COOLDOWN must not be negative"))
	}

	errs = append(errs, c.validateMCP()...)

	if c.AppEnv == Production && c.CookieDomain == "" {
		errs = append(errs, errors.New("COOKIE_DOMAIN is required in production"))
	}

	if c.AppEnv == Production && !c.CookieSecure {
		errs = append(errs, errors.New("COOKIE_SECURE must be true in production"))
	}

	return errs
}

func (c Config) validateMCP() []error {
	var errs []error

	if c.MCP.AccessTokenTTL <= 0 || c.MCP.AccessTokenTTL > 24*time.Hour {
		errs = append(errs, errors.New("MCP_ACCESS_TOKEN_TTL must be between 1s and 24h"))
	}

	if c.MCP.RefreshTokenTTL <= 0 || c.MCP.GrantMaxAge <= 0 {
		errs = append(errs, errors.New("MCP_REFRESH_TOKEN_TTL and MCP_GRANT_MAX_AGE must be positive"))
	}

	if c.AppEnv != Production || !c.MCP.Enabled {
		return errs
	}

	parsed, err := url.Parse(c.PublicURL)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.Path != "" || parsed.RawQuery != "" {
		errs = append(errs, errors.New("PUBLIC_URL must be an https origin with no path or trailing slash when the MCP server is enabled"))
	}

	return errs
}

func (c Config) OAuthIssuer() string {
	return c.PublicURL
}

func (c Config) MCPResource() string {
	return c.PublicURL + "/mcp"
}

func (c Config) GoogleRedirectURL() string {
	return c.PublicURL + "/auth/google/callback"
}

func (c Config) PlaidRedirectURI() string {
	return c.PublicURL + "/connections/oauth"
}
