package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/tahminator/fin/internal/domain"
	"github.com/tahminator/fin/internal/repository/postgres"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	email := flag.String("email", "", "Google account email to allow")
	providerName := flag.String("provider", string(domain.ProviderPlaid), "Credential provider: plaid or alpaca")
	clientID := flag.String("client-id", "", "Plaid client ID or Alpaca API key ID")
	secret := flag.String("secret", "", "Plaid secret or Alpaca API secret key")
	environment := flag.String("env", "sandbox", "Provider environment: sandbox or production")

	flag.Parse()

	provider := domain.Provider(*providerName)
	providerEnv := domain.ProviderEnvironment(*environment)

	if *email == "" || *clientID == "" || *secret == "" || !provider.Valid() || !providerEnv.Valid() {
		flag.Usage()

		return errors.New("email, client-id, secret, a valid provider and a valid env are required")
	}

	ctx := context.Background()

	pool, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}

	defer pool.Close()

	user, err := postgres.NewUserRepository(pool).Upsert(ctx, *email)
	if err != nil {
		return fmt.Errorf("upsert user: %w", err)
	}

	err = postgres.NewCredentialRepository(pool).Upsert(ctx, domain.Credential{
		UserID:      user.ID,
		Provider:    provider,
		Environment: providerEnv,
		ClientID:    *clientID,
		Secret:      *secret,
	})
	if err != nil {
		return fmt.Errorf("upsert credential: %w", err)
	}

	fmt.Printf("allowed %s with %s %s credentials\n", user.Email, provider, providerEnv)

	return nil
}
