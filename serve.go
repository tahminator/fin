package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/tahminator/fin/internal/cardart"
	"github.com/tahminator/fin/internal/config"
	"github.com/tahminator/fin/internal/encryption"
	"github.com/tahminator/fin/internal/handler"
	"github.com/tahminator/fin/internal/identity"
	"github.com/tahminator/fin/internal/metrics"
	"github.com/tahminator/fin/internal/provider/plaid"
	"github.com/tahminator/fin/internal/repository/postgres"
	"github.com/tahminator/fin/internal/server"
	"github.com/tahminator/fin/internal/service"
)

const (
	providerTimeout   = 30 * time.Second
	cardArtTimeout    = 10 * time.Second
	readHeaderTimeout = 10 * time.Second
	shutdownTimeout   = 15 * time.Second
)

func serve() error {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg, err := config.NewFromEnv()
	if err != nil {
		return err
	}

	cipher, err := encryption.New(cfg.SecretKey)
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}

	defer pool.Close()

	users := postgres.NewUserRepository(pool)
	sessions := postgres.NewSessionRepository(pool)
	credentials := postgres.NewCredentialRepository(pool)
	connections := postgres.NewConnectionRepository(pool)
	accounts := postgres.NewAccountRepository(pool)
	transactions := postgres.NewTransactionRepository(pool)
	summary := postgres.NewSummaryRepository(pool)
	metricsRepo := postgres.NewMetricsRepository(pool)
	rateLimits := postgres.NewRateLimitRepository(pool)
	pinger := postgres.NewPinger(pool)

	recorder := metrics.Prometheus{}
	metrics.WatchDatabase(metricsRepo, pool)

	httpClient := &http.Client{Timeout: providerTimeout}
	accountProvider := plaid.New(httpClient, recorder, cfg.PlaidCountryCodes)

	google, err := identity.NewGoogle(ctx, httpClient, cfg.GoogleClientID, cfg.GoogleSecret, cfg.GoogleRedirectURL())
	if err != nil {
		return err
	}

	rateLimiter := service.NewRateLimiter(rateLimits)
	authService := service.NewAuthService(users, sessions, google, recorder, cfg.SessionTTL)
	cardArtService := service.NewCardArtService(accounts, cardart.NewOpenCard(&http.Client{Timeout: cardArtTimeout}))
	connectionService := service.NewConnectionService(
		service.ConnectionRepositories{
			Credentials:  credentials,
			Connections:  connections,
			Accounts:     accounts,
			Transactions: transactions,
		},
		accountProvider,
		cardArtService,
		cipher,
		recorder,
		service.SyncCooldown{Limiter: rateLimiter, Window: cfg.SyncCooldown},
		cfg.PlaidRedirectURI(),
	)
	notificationService := service.NewNotificationService(connections)
	scheduler := service.NewSyncScheduler(cfg.SyncSchedule, connections, connectionService, recorder, logger)

	mux := http.NewServeMux()
	api := server.NewAPI(mux)

	handlers := server.Handlers{
		Auth:           handler.NewAuthHandler(authService, handler.CookieConfig{Domain: cfg.CookieDomain, Secure: cfg.CookieSecure}),
		Summary:        handler.NewSummaryHandler(summary),
		Connections:    handler.NewConnectionHandler(connections, connectionService),
		Accounts:       handler.NewAccountHandler(accounts),
		CardArt:        handler.NewCardArtHandler(cardArtService),
		Transactions:   handler.NewTransactionHandler(transactions),
		Notifications:  handler.NewNotificationHandler(notificationService),
		Version:        handler.NewVersionHandler(cfg.Version),
		RequireSession: server.RequireSession(api, authService),
	}

	if cfg.StaticDir != "" {
		handlers.SPA = server.NewSPA(cfg.StaticDir)
	}

	server.RegisterRoutes(mux, api, handlers)

	go scheduler.Run(ctx)

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           server.WithOps(server.Observe(mux, recorder, logger), handler.NewHealthHandler(pinger)),
		ReadHeaderTimeout: readHeaderTimeout,
	}

	return runServer(ctx, logger, srv)
}

func runServer(ctx context.Context, logger *slog.Logger, srv *http.Server) error {
	errs := make(chan error, 1)

	go func() {
		logger.Info("listening", "addr", srv.Addr)

		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errs <- fmt.Errorf("serve %s: %w", srv.Addr, err)
		}
	}()

	var serveErr error

	select {
	case <-ctx.Done():
	case serveErr = <-errs:
	}

	logger.Info("shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), shutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		serveErr = errors.Join(serveErr, fmt.Errorf("shutdown %s: %w", srv.Addr, err))
	}

	return serveErr
}
