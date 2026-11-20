package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/tahminator/fin/internal/config"
	"github.com/tahminator/fin/internal/domain"
	"github.com/tahminator/fin/internal/encryption"
	"github.com/tahminator/fin/internal/metrics"
	"github.com/tahminator/fin/internal/provider/plaid"
	"github.com/tahminator/fin/internal/repository/postgres"
)

const providerTimeout = 30 * time.Second

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) != 1 {
		return errors.New("usage: sandbox-reset-login <connection-id>")
	}

	connectionID, err := uuid.Parse(args[0])
	if err != nil {
		return fmt.Errorf("connection id: %w", err)
	}

	cfg, err := config.NewFromEnv()
	if err != nil {
		return err
	}

	cipher, err := encryption.New(cfg.SecretKey)
	if err != nil {
		return err
	}

	ctx := context.Background()

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}

	defer pool.Close()

	connection, err := postgres.NewConnectionRepository(pool).FindByID(ctx, connectionID)
	if err != nil {
		return fmt.Errorf("find connection: %w", err)
	}

	credential, err := postgres.NewCredentialRepository(pool).Find(ctx, connection.UserID, connection.Provider)
	if err != nil {
		return fmt.Errorf("find credential: %w", err)
	}

	if credential.Environment != domain.ProviderEnvironmentSandbox {
		return fmt.Errorf("refusing to reset a %s connection; only sandbox is allowed", credential.Environment)
	}

	accessToken, err := cipher.Decrypt(connection.AccessTokenCiphertext)
	if err != nil {
		return err
	}

	provider := plaid.New(&http.Client{Timeout: providerTimeout}, metrics.Noop{}, cfg.PlaidCountryCodes)

	if err := provider.ResetLogin(ctx, credential, accessToken); err != nil {
		return fmt.Errorf("reset login: %w", err)
	}

	fmt.Printf("connection %s now requires re-authentication\n", connectionID)

	return nil
}
