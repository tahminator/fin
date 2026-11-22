package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"slices"
	"time"
	"uuid"

	"github.com/tahminator/fin/internal/domain"
	"github.com/tahminator/fin/internal/provider"
)

const investmentHistoryYears = 2

func (s *ConnectionService) syncInvestments(
	ctx context.Context,
	connection domain.Connection,
	credential domain.Credential,
	accessToken string,
	accounts []provider.Account,
	accountIDs map[string]uuid.UUID,
) error {
	byExternalID := investmentAccountIDs(accounts, accountIDs)
	if len(byExternalID) == 0 {
		return s.saveInvestmentsStatus(ctx, connection, nil)
	}

	userID := connection.UserID
	ids := slices.Collect(maps.Values(byExternalID))
	capturedAt := s.now().UTC().Truncate(time.Second)

	holdings, err := s.provider.InvestmentHoldings(ctx, credential, accessToken)
	if err != nil {
		return s.investmentsUnavailable(ctx, connection, fmt.Errorf("fetch investment holdings: %w", err))
	}

	if err := s.investments.UpsertSecurities(ctx, userID, toDomainSecurities(holdings.Securities)); err != nil {
		return fmt.Errorf("save holding securities: %w", err)
	}

	if err := s.investments.ReplaceHoldings(ctx, userID, ids, toDomainHoldings(holdings.Holdings, byExternalID)); err != nil {
		return fmt.Errorf("save investment holdings: %w", err)
	}

	if err := s.investments.CaptureSnapshot(ctx, userID, ids, capturedAt); err != nil {
		return fmt.Errorf("save investment snapshot: %w", err)
	}

	end := capturedAt
	start := end.AddDate(-investmentHistoryYears, 0, 0)

	transactions, err := s.provider.InvestmentTransactions(ctx, credential, accessToken, start, end)
	if err != nil {
		return s.investmentsUnavailable(ctx, connection, fmt.Errorf("fetch investment transactions: %w", err))
	}

	if err := s.investments.UpsertSecurities(ctx, userID, toDomainSecurities(transactions.Securities)); err != nil {
		return fmt.Errorf("save transaction securities: %w", err)
	}

	mapped := toDomainInvestmentTransactions(transactions.Transactions, byExternalID)

	if err := s.investments.ReplaceTransactions(ctx, userID, ids, start, mapped); err != nil {
		return fmt.Errorf("save investment transactions: %w", err)
	}

	return s.saveInvestmentsStatus(ctx, connection, new(domain.InvestmentsStatusReady))
}

func (s *ConnectionService) investmentsUnavailable(ctx context.Context, connection domain.Connection, err error) error {
	var unavailable *domain.InvestmentsUnavailableError
	if !errors.As(err, &unavailable) {
		return err
	}

	slog.InfoContext(ctx, "investments unavailable", "connection_id", connection.ID, "status", unavailable.Status, "reason", unavailable.Reason)

	return s.saveInvestmentsStatus(ctx, connection, &unavailable.Status)
}

func (s *ConnectionService) saveInvestmentsStatus(ctx context.Context, connection domain.Connection, status *domain.InvestmentsStatus) error {
	if sameInvestmentsStatus(connection.InvestmentsStatus, status) {
		return nil
	}

	if err := s.connections.UpdateInvestmentsStatus(ctx, connection.UserID, connection.ID, status); err != nil {
		return fmt.Errorf("save investments status: %w", err)
	}

	return nil
}

func sameInvestmentsStatus(a, b *domain.InvestmentsStatus) bool {
	if a == nil || b == nil {
		return a == b
	}

	return *a == *b
}
