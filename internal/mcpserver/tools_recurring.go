package mcpserver

import (
	"context"
	"time"
)

const (
	recurringLookback      = 395 * 24 * time.Hour
	defaultMinOccurrences  = 3
	maxRecurringCandidates = 20000
)

type FindRecurringInput struct {
	From           string `json:"from,omitempty" jsonschema:"Start date (YYYY-MM-DD). Defaults to 13 months before to."`
	To             string `json:"to,omitempty" jsonschema:"End date (YYYY-MM-DD). Defaults to today."`
	MinOccurrences int    `json:"min_occurrences,omitempty" jsonschema:"Minimum payments needed to call something recurring. Default 3."`
}

type FindRecurringOutput struct {
	Series []RecurringSeries `json:"series" jsonschema:"Likely subscriptions, bills and paychecks, soonest next payment first"`
}

func (t *toolset) findRecurring(ctx context.Context, input FindRecurringInput) (FindRecurringOutput, error) {
	from, to, err := dateRange(input.From, input.To, t.deps.Now(), recurringLookback)
	if err != nil {
		return FindRecurringOutput{}, err
	}

	minOccurrences := input.MinOccurrences
	if minOccurrences == 0 {
		minOccurrences = defaultMinOccurrences
	}

	if minOccurrences < 2 || minOccurrences > 52 {
		return FindRecurringOutput{}, invalidParams("min_occurrences must be between 2 and 52")
	}

	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	candidates, err := t.deps.Transactions.RecurringCandidates(ctx, t.userID(), from, to, maxRecurringCandidates)
	if err != nil {
		return FindRecurringOutput{}, err
	}

	series := detectRecurring(candidates, minOccurrences)
	if series == nil {
		series = []RecurringSeries{}
	}

	return FindRecurringOutput{Series: series}, nil
}
