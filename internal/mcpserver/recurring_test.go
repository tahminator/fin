package mcpserver

import (
	"testing"
	"time"

	"github.com/tahminator/fin/internal/repository"
)

func series(merchant, amount string, start time.Time, step func(time.Time) time.Time, count int) []repository.RecurringCandidate {
	candidates := make([]repository.RecurringCandidate, 0, count)

	for date := start; len(candidates) < count; date = step(date) {
		candidates = append(candidates, repository.RecurringCandidate{Merchant: merchant, Amount: amount, Currency: "USD", PostedOn: date})
	}

	return candidates
}

func TestDetectRecurringCadences(t *testing.T) {
	start := time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC)
	monthly := series("Netflix", "-15.49", start, func(d time.Time) time.Time { return d.AddDate(0, 1, 0) }, 6)
	biweekly := series("ACME PAYROLL", "2500.00", start, func(d time.Time) time.Time { return d.AddDate(0, 0, 14) }, 8)
	yearly := series("Domain renewal", "-12.00", start, func(d time.Time) time.Time { return d.AddDate(1, 0, 0) }, 3)

	found := detectRecurring(append(append(monthly, biweekly...), yearly...), 3)

	cadences := map[string]string{}
	for _, s := range found {
		cadences[s.Merchant] = s.Cadence
	}

	want := map[string]string{"Netflix": "monthly", "ACME PAYROLL": "biweekly", "Domain renewal": "yearly"}
	for merchant, cadence := range want {
		if cadences[merchant] != cadence {
			t.Errorf("%s cadence = %q, want %q (all: %v)", merchant, cadences[merchant], cadence, cadences)
		}
	}
}

func TestDetectRecurringToleratesJitterButNotAmountDrift(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	jitter := []int{0, 31, 59, 92, 120, 151}

	var jittered, drifting []repository.RecurringCandidate

	for i, day := range jitter {
		jittered = append(jittered, repository.RecurringCandidate{Merchant: "Gym", Amount: "-40.00", Currency: "USD", PostedOn: start.AddDate(0, 0, day)})
		drifting = append(drifting, repository.RecurringCandidate{Merchant: "Grocer", Amount: []string{"-12.00", "-80.00", "-35.00", "-150.00", "-9.00", "-60.00"}[i], Currency: "USD", PostedOn: start.AddDate(0, 0, day)})
	}

	found := detectRecurring(append(jittered, drifting...), 3)

	if len(found) != 1 || found[0].Merchant != "Gym" || found[0].NextExpected != "2026-07-01" {
		t.Errorf("found = %+v, want only Gym, next expected 2026-07-01", found)
	}
}

func TestDetectRecurringNeedsMinimumOccurrences(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	two := series("Spotify", "-9.99", start, func(d time.Time) time.Time { return d.AddDate(0, 1, 0) }, 2)

	if found := detectRecurring(two, 3); len(found) != 0 {
		t.Errorf("found = %+v, want nothing below min_occurrences", found)
	}
}
