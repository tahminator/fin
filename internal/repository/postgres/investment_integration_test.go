//go:build integration

package postgres

import (
	"context"
	"os"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/tahminator/fin/internal/domain"
	"github.com/tahminator/fin/internal/repository"
)

type investmentFixture struct {
	pool       *pgxpool.Pool
	repo       *InvestmentRepository
	userID     uuid.UUID
	brokerage  uuid.UUID
	retirement uuid.UUID
}

func newInvestmentFixture(t *testing.T) investmentFixture {
	t.Helper()

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL is not set")
	}

	ctx := context.Background()

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}

	t.Cleanup(pool.Close)

	fixture := investmentFixture{pool: pool, repo: NewInvestmentRepository(pool)}

	err = pool.QueryRow(ctx, `INSERT INTO users (email) VALUES ($1) RETURNING id;`, "investments-"+uuid.NewV4().String()+"@example.test").Scan(&fixture.userID)
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}

	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1;`, fixture.userID); err != nil {
			t.Errorf("delete user: %v", err)
		}
	})

	var connectionID uuid.UUID

	err = pool.QueryRow(ctx, `
INSERT INTO connections (user_id, provider, external_id, access_token_ciphertext, status)
VALUES ($1, 'plaid', $2, 'ciphertext', 'active')
RETURNING id;`, fixture.userID, uuid.NewV4().String()).Scan(&connectionID)
	if err != nil {
		t.Fatalf("insert connection: %v", err)
	}

	fixture.brokerage = fixture.insertAccount(t, connectionID, "Brokerage", "1500.0000")
	fixture.retirement = fixture.insertAccount(t, connectionID, "Retirement", "900.0000")

	return fixture
}

func (f investmentFixture) insertAccount(t *testing.T, connectionID uuid.UUID, name, balance string) uuid.UUID {
	t.Helper()

	var accountID uuid.UUID

	err := f.pool.QueryRow(context.Background(), `
INSERT INTO accounts (user_id, connection_id, external_id, name, type, current_balance, currency)
VALUES ($1, $2, $3, $4, 'investment', $5::numeric, 'USD')
RETURNING id;`, f.userID, connectionID, uuid.NewV4().String(), name, balance).Scan(&accountID)
	if err != nil {
		t.Fatalf("insert account: %v", err)
	}

	return accountID
}

func (f investmentFixture) setBalance(t *testing.T, accountID uuid.UUID, balance string) {
	t.Helper()

	if _, err := f.pool.Exec(context.Background(), `UPDATE accounts SET current_balance = $2::numeric WHERE id = $1;`, accountID, balance); err != nil {
		t.Fatalf("update balance: %v", err)
	}
}

func (f investmentFixture) countRows(t *testing.T, table string) int {
	t.Helper()

	var total int

	err := f.pool.QueryRow(context.Background(), "SELECT COUNT(*) FROM "+table+" WHERE user_id = $1;", f.userID).Scan(&total)
	if err != nil {
		t.Fatalf("count %s: %v", table, err)
	}

	return total
}

func text(value string) *string {
	return &value
}

func TestInvestmentRepositoryRoundTrip(t *testing.T) {
	f := newInvestmentFixture(t)
	ctx := context.Background()
	accounts := []uuid.UUID{f.brokerage, f.retirement}

	securities := []domain.InvestmentSecurity{
		{ExternalID: "sec-vti", Name: text("Vanguard Total Stock Market ETF"), Ticker: text("VTI"), Type: text("etf")},
		{ExternalID: "sec-aapl", Name: text("Apple Inc."), Ticker: text("AAPL"), Type: text("equity")},
		{ExternalID: "sec-sold", Name: text("Sold Fund"), Ticker: text("SOLD"), Type: text("mutual fund")},
	}

	if err := f.repo.UpsertSecurities(ctx, f.userID, securities); err != nil {
		t.Fatalf("UpsertSecurities: %v", err)
	}

	if err := f.repo.UpsertSecurities(ctx, f.userID, []domain.InvestmentSecurity{{ExternalID: "sec-sold"}}); err != nil {
		t.Fatalf("UpsertSecurities without metadata: %v", err)
	}

	holdings := []domain.InvestmentHolding{
		{AccountID: f.brokerage, SecurityExternalID: "sec-vti", InstitutionPrice: "250.5", InstitutionValue: "1002", Quantity: "4", Currency: text("USD")},
		{AccountID: f.brokerage, SecurityExternalID: "sec-aapl", InstitutionPrice: "200", InstitutionValue: "400", Quantity: "2", Currency: text("USD")},
		{AccountID: f.retirement, SecurityExternalID: "sec-vti", InstitutionPrice: "250.5", InstitutionValue: "751.5", Quantity: "3", Currency: text("USD")},
	}

	if err := f.repo.ReplaceHoldings(ctx, f.userID, accounts, holdings); err != nil {
		t.Fatalf("ReplaceHoldings: %v", err)
	}

	listed, err := f.repo.ListHoldings(ctx, f.userID, repository.InvestmentFilter{})
	if err != nil {
		t.Fatalf("ListHoldings: %v", err)
	}

	if len(listed) != 3 || listed[0].InstitutionValue != "1002.0000" || listed[0].SecurityExternalID != "sec-vti" || *listed[0].Ticker != "VTI" {
		t.Fatalf("ListHoldings = %+v", listed)
	}

	excluded, err := f.repo.ListHoldings(ctx, f.userID, repository.InvestmentFilter{ExcludeAccountID: []uuid.UUID{f.brokerage}})
	if err != nil || len(excluded) != 1 || excluded[0].AccountID != f.retirement {
		t.Fatalf("ListHoldings excluding brokerage = %+v, %v", excluded, err)
	}

	hiddenAll, err := f.repo.ListHoldings(ctx, f.userID, repository.InvestmentFilter{ExcludeAccountID: accounts})
	if err != nil || len(hiddenAll) != 0 {
		t.Fatalf("ListHoldings hiding all = %+v, %v", hiddenAll, err)
	}

	since := time.Date(2024, 11, 22, 0, 0, 0, 0, time.UTC)
	transactions := []domain.InvestmentTransaction{
		{AccountID: f.brokerage, SecurityExternalID: text("sec-sold"), ExternalID: "tx-sell", PostedOn: since.AddDate(0, 6, 0), Name: "SELL Sold Fund", Quantity: "-5", Amount: "510.25", Price: "102.05", Fees: text("1.25"), Type: "sell", Subtype: "sell", Currency: text("USD")},
		{AccountID: f.brokerage, SecurityExternalID: text("sec-aapl"), ExternalID: "tx-buy", PostedOn: since.AddDate(0, 7, 0), Name: "BUY Apple", Quantity: "2", Amount: "-400", Price: "200", Type: "buy", Subtype: "buy", Currency: text("USD")},
		{AccountID: f.retirement, ExternalID: "tx-cash", PostedOn: since.AddDate(0, 8, 0), Name: "Contribution", Quantity: "0", Amount: "0", Price: "0", Type: "cash", Subtype: "contribution", Currency: text("USD")},
	}

	if err := f.repo.ReplaceTransactions(ctx, f.userID, accounts, since, transactions); err != nil {
		t.Fatalf("ReplaceTransactions: %v", err)
	}

	if _, err := f.pool.Exec(ctx, `UPDATE investment_transactions SET posted_on = $2 WHERE user_id = $1 AND external_id = 'tx-cash';`, f.userID, since.AddDate(0, 0, -1)); err != nil {
		t.Fatalf("age transaction: %v", err)
	}

	if err := f.repo.ReplaceTransactions(ctx, f.userID, accounts, since, transactions[:2]); err != nil {
		t.Fatalf("ReplaceTransactions again: %v", err)
	}

	page, total, err := f.repo.ListTransactions(ctx, f.userID, repository.InvestmentFilter{}, 1, 0)
	if err != nil {
		t.Fatalf("ListTransactions: %v", err)
	}

	if total != 2 || len(page) != 1 || page[0].ExternalID != "tx-buy" || page[0].Amount != "-400.0000" {
		t.Fatalf("ListTransactions page 1 = %+v, total %d", page, total)
	}

	sold, _, err := f.repo.ListTransactions(ctx, f.userID, repository.InvestmentFilter{AccountID: &f.brokerage}, 10, 1)
	if err != nil || len(sold) != 1 {
		t.Fatalf("ListTransactions page 2 = %+v, %v", sold, err)
	}

	if *sold[0].Ticker != "SOLD" || *sold[0].SecurityName != "Sold Fund" || sold[0].Amount != "510.2500" || *sold[0].Fees != "1.2500" {
		t.Fatalf("sold transaction lost its security metadata: %+v", sold[0])
	}

	_, hiddenTotal, err := f.repo.ListTransactions(ctx, f.userID, repository.InvestmentFilter{ExcludeAccountID: accounts}, 10, 0)
	if err != nil || hiddenTotal != 0 {
		t.Fatalf("ListTransactions hiding all total = %d, %v", hiddenTotal, err)
	}

	first := time.Now().UTC().Add(-2 * time.Hour).Truncate(time.Second)

	for range 2 {
		if err := f.repo.CaptureSnapshot(ctx, f.userID, accounts, first); err != nil {
			t.Fatalf("CaptureSnapshot: %v", err)
		}
	}

	if got := f.countRows(t, "investment_account_snapshots"); got != 2 {
		t.Fatalf("account snapshots after repeated capture = %d, want 2", got)
	}

	if got := f.countRows(t, "investment_holding_snapshots"); got != 3 {
		t.Fatalf("holding snapshots after repeated capture = %d, want 3", got)
	}

	f.setBalance(t, f.brokerage, "1600")

	if err := f.repo.CaptureSnapshot(ctx, f.userID, accounts, first.Add(5*time.Minute)); err != nil {
		t.Fatalf("CaptureSnapshot within spacing: %v", err)
	}

	if got := f.countRows(t, "investment_account_snapshots"); got != 2 {
		t.Fatalf("account snapshots after superseding capture = %d, want 2", got)
	}

	f.setBalance(t, f.brokerage, "1800")

	second := first.Add(time.Hour)

	if err := f.repo.CaptureSnapshot(ctx, f.userID, accounts, second); err != nil {
		t.Fatalf("CaptureSnapshot later: %v", err)
	}

	bounds, err := f.repo.HistoryBounds(ctx, f.userID, repository.InvestmentFilter{})
	if err != nil || bounds.Earliest == nil || !bounds.Earliest.Equal(first.Add(5*time.Minute)) || !bounds.Latest.Equal(second) {
		t.Fatalf("HistoryBounds = %+v, %v", bounds, err)
	}

	history, err := f.repo.History(ctx, f.userID, repository.InvestmentHistoryQuery{Bucket: 15 * time.Minute})
	if err != nil {
		t.Fatalf("History: %v", err)
	}

	totals := map[int64]string{}

	for _, row := range history {
		if row.AccountID == nil {
			totals[row.At.Unix()] = row.Value
		}
	}

	if totals[first.Add(5*time.Minute).Unix()] != "2500.0000" || totals[second.Unix()] != "2700.0000" || len(history) != 6 {
		t.Fatalf("History rows = %+v", history)
	}

	var vti uuid.UUID

	if err := f.pool.QueryRow(ctx, `SELECT id FROM investment_securities WHERE user_id = $1 AND external_id = 'sec-vti';`, f.userID).Scan(&vti); err != nil {
		t.Fatalf("find security: %v", err)
	}

	selected, err := f.repo.History(ctx, f.userID, repository.InvestmentHistoryQuery{
		Filter:      repository.InvestmentFilter{ExcludeAccountID: []uuid.UUID{f.retirement}},
		SecurityIDs: []uuid.UUID{vti},
		Bucket:      15 * time.Minute,
	})
	if err != nil || len(selected) != 4 || selected[0].Value != "1002.0000" {
		t.Fatalf("History for selected security = %+v, %v", selected, err)
	}

	securitiesList, err := f.repo.ListSecurities(ctx, f.userID, repository.InvestmentFilter{})
	if err != nil || len(securitiesList) != 2 || !securitiesList[0].Held {
		t.Fatalf("ListSecurities = %+v, %v", securitiesList, err)
	}

	old := time.Now().UTC().AddDate(0, 0, -100).Truncate(24 * time.Hour).Add(10 * time.Hour)

	for _, capturedAt := range []time.Time{old, old.Add(2 * time.Hour), second.Add(time.Hour)} {
		if err := f.repo.CaptureSnapshot(ctx, f.userID, []uuid.UUID{f.brokerage}, capturedAt); err != nil {
			t.Fatalf("CaptureSnapshot %v: %v", capturedAt, err)
		}
	}

	var oldSnapshots int

	err = f.pool.QueryRow(ctx, `SELECT COUNT(*) FROM investment_account_snapshots WHERE account_id = $1 AND captured_at < $2;`, f.brokerage, old.Add(24*time.Hour)).Scan(&oldSnapshots)
	if err != nil || oldSnapshots != 1 {
		t.Fatalf("snapshots kept for an old day = %d, %v; want 1", oldSnapshots, err)
	}
}

func TestConnectionInvestmentsStatusRoundTrip(t *testing.T) {
	f := newInvestmentFixture(t)
	ctx := context.Background()
	connections := NewConnectionRepository(f.pool)

	listed, _, err := connections.ListByUser(ctx, f.userID, 10, 0)
	if err != nil || len(listed) != 1 || listed[0].InvestmentsStatus != nil {
		t.Fatalf("ListByUser = %+v, %v", listed, err)
	}

	if err := connections.UpdateInvestmentsStatus(ctx, f.userID, listed[0].ID, new(domain.InvestmentsStatusConsentRequired)); err != nil {
		t.Fatalf("UpdateInvestmentsStatus: %v", err)
	}

	connection, err := connections.Get(ctx, f.userID, listed[0].ID)
	if err != nil || connection.InvestmentsStatus == nil || *connection.InvestmentsStatus != domain.InvestmentsStatusConsentRequired {
		t.Fatalf("Get = %+v, %v", connection, err)
	}
}
