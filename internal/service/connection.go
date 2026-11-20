package service

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"
	"uuid"

	"github.com/tahminator/fin/internal/domain"
	"github.com/tahminator/fin/internal/encryption"
	"github.com/tahminator/fin/internal/metrics"
	"github.com/tahminator/fin/internal/provider"
	"github.com/tahminator/fin/internal/repository"
)

const syncAllLimit = 1000

type SyncSummary struct {
	Synced         int
	ReauthRequired int
	Failed         int
}

type ConnectionRepositories struct {
	Credentials  repository.CredentialRepository
	Connections  repository.ConnectionRepository
	Accounts     repository.AccountRepository
	Transactions repository.TransactionRepository
}

type ConnectionService struct {
	credentials  repository.CredentialRepository
	connections  repository.ConnectionRepository
	accounts     repository.AccountRepository
	transactions repository.TransactionRepository
	provider     provider.Provider
	cipher       *encryption.Cipher
	recorder     metrics.Recorder
	redirectURI  string
	now          func() time.Time
}

func NewConnectionService(
	repos ConnectionRepositories,
	accountProvider provider.Provider,
	cipher *encryption.Cipher,
	recorder metrics.Recorder,
	redirectURI string,
) *ConnectionService {
	return &ConnectionService{
		credentials:  repos.Credentials,
		connections:  repos.Connections,
		accounts:     repos.Accounts,
		transactions: repos.Transactions,
		provider:     accountProvider,
		cipher:       cipher,
		recorder:     recorder,
		redirectURI:  redirectURI,
		now:          time.Now,
	}
}

func (s *ConnectionService) CreateLinkToken(ctx context.Context, userID uuid.UUID) (string, error) {
	credential, err := s.credential(ctx, userID)
	if err != nil {
		return "", err
	}

	return s.provider.CreateLinkToken(ctx, credential, userID.String(), s.redirectURI)
}

func (s *ConnectionService) Link(ctx context.Context, userID uuid.UUID, publicToken string) (domain.Connection, error) {
	credential, err := s.credential(ctx, userID)
	if err != nil {
		return domain.Connection{}, err
	}

	item, err := s.provider.Exchange(ctx, credential, publicToken)
	if err != nil {
		return domain.Connection{}, fmt.Errorf("exchange public token: %w", err)
	}

	connection, err := s.connections.Create(ctx, domain.Connection{
		UserID:                userID,
		Provider:              credential.Provider,
		ExternalID:            item.ExternalID,
		AccessTokenCiphertext: s.cipher.Encrypt(item.AccessToken),
		InstitutionID:         item.InstitutionID,
		InstitutionName:       item.InstitutionName,
		Status:                domain.ConnectionStatusActive,
	})
	if err != nil {
		return domain.Connection{}, fmt.Errorf("create connection: %w", err)
	}

	return s.syncAndReload(ctx, userID, connection.ID, metrics.SyncTriggerLink)
}

func (s *ConnectionService) Sync(ctx context.Context, userID, connectionID uuid.UUID, trigger metrics.SyncTrigger) error {
	started := s.now()

	err := s.sync(ctx, userID, connectionID)

	s.recorder.ConnectionSynced(trigger, syncOutcomeOf(err), s.now().Sub(started))

	return err
}

func (s *ConnectionService) SyncOne(ctx context.Context, userID, connectionID uuid.UUID) (domain.Connection, error) {
	return s.syncAndReload(ctx, userID, connectionID, metrics.SyncTriggerManual)
}

func (s *ConnectionService) SyncAll(ctx context.Context, userID uuid.UUID, trigger metrics.SyncTrigger) (SyncSummary, error) {
	connections, _, err := s.connections.ListByUserAndStatus(ctx, userID, domain.ConnectionStatusActive, syncAllLimit, 0)
	if err != nil {
		return SyncSummary{}, fmt.Errorf("list active connections: %w", err)
	}

	var summary SyncSummary

	for _, connection := range connections {
		err := s.Sync(ctx, userID, connection.ID, trigger)

		switch {
		case err == nil:
			summary.Synced++
		case errors.Is(err, domain.ErrReauthRequired):
			summary.ReauthRequired++
		default:
			summary.Failed++
		}
	}

	return summary, nil
}

func (s *ConnectionService) CreateReauthLinkToken(ctx context.Context, userID, connectionID uuid.UUID) (string, error) {
	connection, err := s.connections.Get(ctx, userID, connectionID)
	if err != nil {
		return "", err
	}

	if connection.Status != domain.ConnectionStatusNeedsReauth {
		return "", fmt.Errorf("%w: connection does not need to be reconnected", domain.ErrConflict)
	}

	credential, err := s.credential(ctx, userID)
	if err != nil {
		return "", err
	}

	accessToken, err := s.cipher.Decrypt(connection.AccessTokenCiphertext)
	if err != nil {
		return "", err
	}

	return s.provider.CreateUpdateLinkToken(ctx, credential, userID.String(), accessToken, s.redirectURI)
}

func (s *ConnectionService) CompleteReauth(ctx context.Context, userID, connectionID uuid.UUID) (domain.Connection, error) {
	if err := s.connections.UpdateStatus(ctx, userID, connectionID, domain.ConnectionStatusActive, nil); err != nil {
		return domain.Connection{}, err
	}

	return s.syncAndReload(ctx, userID, connectionID, metrics.SyncTriggerReauth)
}

func (s *ConnectionService) Remove(ctx context.Context, userID, connectionID uuid.UUID) error {
	connection, err := s.connections.Get(ctx, userID, connectionID)
	if err != nil {
		return err
	}

	credential, err := s.credential(ctx, userID)
	if err != nil {
		return err
	}

	accessToken, err := s.cipher.Decrypt(connection.AccessTokenCiphertext)
	if err != nil {
		return err
	}

	if err := s.provider.Remove(ctx, credential, accessToken); err != nil {
		return fmt.Errorf("remove item: %w", err)
	}

	return s.connections.Delete(ctx, userID, connectionID)
}

func (s *ConnectionService) syncAndReload(ctx context.Context, userID, connectionID uuid.UUID, trigger metrics.SyncTrigger) (domain.Connection, error) {
	err := s.Sync(ctx, userID, connectionID, trigger)
	if errors.Is(err, domain.ErrNotFound) {
		return domain.Connection{}, err
	}

	return s.connections.Get(ctx, userID, connectionID)
}

func (s *ConnectionService) sync(ctx context.Context, userID, connectionID uuid.UUID) error {
	connection, err := s.connections.Get(ctx, userID, connectionID)
	if err != nil {
		return err
	}

	err = s.pull(ctx, connection)

	switch {
	case err == nil:
		return nil
	case errors.Is(err, domain.ErrReauthRequired):
		message := err.Error()

		return errors.Join(err, s.connections.UpdateStatus(ctx, userID, connectionID, domain.ConnectionStatusNeedsReauth, &message))
	default:
		return errors.Join(err, s.connections.UpdateLastError(ctx, userID, connectionID, err.Error()))
	}
}

func (s *ConnectionService) pull(ctx context.Context, connection domain.Connection) error {
	credential, err := s.credential(ctx, connection.UserID)
	if err != nil {
		return err
	}

	accessToken, err := s.cipher.Decrypt(connection.AccessTokenCiphertext)
	if err != nil {
		return err
	}

	accounts, err := s.provider.Accounts(ctx, credential, accessToken)
	if err != nil {
		return fmt.Errorf("fetch accounts: %w", err)
	}

	accountIDs, err := s.accounts.Upsert(ctx, connection.UserID, connection.ID, toDomainAccounts(accounts))
	if err != nil {
		return fmt.Errorf("save accounts: %w", err)
	}

	result, err := s.provider.SyncTransactions(ctx, credential, accessToken, connection.SyncCursor)
	if err != nil {
		return fmt.Errorf("fetch transactions: %w", err)
	}

	changed := slices.Concat(result.Added, result.Modified)

	if err := s.transactions.Upsert(ctx, connection.UserID, toDomainTransactions(changed, accountIDs)); err != nil {
		return fmt.Errorf("save transactions: %w", err)
	}

	if err := s.transactions.DeleteByExternalIDs(ctx, connection.UserID, result.Removed); err != nil {
		return fmt.Errorf("delete removed transactions: %w", err)
	}

	if err := s.connections.UpdateSyncState(ctx, connection.UserID, connection.ID, result.NextCursor, s.now()); err != nil {
		return fmt.Errorf("save sync cursor: %w", err)
	}

	return nil
}

func (s *ConnectionService) credential(ctx context.Context, userID uuid.UUID) (domain.Credential, error) {
	credential, err := s.credentials.Find(ctx, userID, domain.ProviderPlaid)
	if errors.Is(err, domain.ErrNotFound) {
		return domain.Credential{}, fmt.Errorf("%w: no Plaid credentials are configured for your account", domain.ErrNotAllowed)
	}

	if err != nil {
		return domain.Credential{}, fmt.Errorf("find credential: %w", err)
	}

	return credential, nil
}

func syncOutcomeOf(err error) metrics.SyncOutcome {
	switch {
	case err == nil:
		return metrics.SyncOutcomeSuccess
	case errors.Is(err, domain.ErrReauthRequired):
		return metrics.SyncOutcomeReauthRequired
	}

	return metrics.SyncOutcomeError
}
