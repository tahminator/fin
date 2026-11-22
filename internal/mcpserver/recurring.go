package mcpserver

import (
	"math"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/tahminator/fin/internal/repository"
)

const (
	cadenceTolerance = 0.25
	amountTolerance  = 0.15
	minimumAgreement = 0.7
	hoursPerDay      = 24
)

type cadence struct {
	name string
	days float64
}

var cadences = []cadence{
	{name: "weekly", days: 7},
	{name: "biweekly", days: 14},
	{name: "monthly", days: 30.44},
	{name: "quarterly", days: 91.31},
	{name: "yearly", days: 365.25},
}

type RecurringSeries struct {
	Merchant      string `json:"merchant"`
	Direction     string `json:"direction" jsonschema:"inflow (such as a paycheck) or outflow (such as a subscription or bill)"`
	Currency      string `json:"currency"`
	TypicalAmount string `json:"typical_amount" jsonschema:"Median amount, signed"`
	Cadence       string `json:"cadence"`
	Occurrences   int    `json:"occurrences"`
	FirstSeen     string `json:"first_seen"`
	LastSeen      string `json:"last_seen"`
	NextExpected  string `json:"next_expected"`
}

type seriesKey struct {
	merchant string
	currency string
	outflow  bool
}

type occurrence struct {
	amount  float64
	raw     string
	date    time.Time
	display string
}

func detectRecurring(candidates []repository.RecurringCandidate, minOccurrences int) []RecurringSeries {
	groups := map[seriesKey][]occurrence{}

	var order []seriesKey

	for _, candidate := range candidates {
		amount, err := strconv.ParseFloat(candidate.Amount, 64)
		if err != nil || amount == 0 {
			continue
		}

		key := seriesKey{merchant: normalizeMerchant(candidate.Merchant), currency: candidate.Currency, outflow: amount < 0}

		if _, seen := groups[key]; !seen {
			order = append(order, key)
		}

		groups[key] = append(groups[key], occurrence{amount: amount, raw: candidate.Amount, date: candidate.PostedOn, display: candidate.Merchant})
	}

	var series []RecurringSeries

	for _, key := range order {
		if found, ok := recurringSeries(key, groups[key], minOccurrences); ok {
			series = append(series, found)
		}
	}

	slices.SortFunc(series, func(a, b RecurringSeries) int {
		return strings.Compare(a.NextExpected, b.NextExpected)
	})

	return series
}

func recurringSeries(key seriesKey, occurrences []occurrence, minOccurrences int) (RecurringSeries, bool) {
	if len(occurrences) < minOccurrences {
		return RecurringSeries{}, false
	}

	slices.SortFunc(occurrences, func(a, b occurrence) int {
		return a.date.Compare(b.date)
	})

	gaps := make([]float64, 0, len(occurrences)-1)

	for i := 1; i < len(occurrences); i++ {
		gaps = append(gaps, occurrences[i].date.Sub(occurrences[i-1].date).Hours()/hoursPerDay)
	}

	match, ok := matchCadence(median(gaps))
	if !ok || agreement(gaps, match.days, cadenceTolerance) < minimumAgreement {
		return RecurringSeries{}, false
	}

	amounts := make([]float64, len(occurrences))

	for i, o := range occurrences {
		amounts[i] = math.Abs(o.amount)
	}

	typical := median(amounts)
	if agreement(amounts, typical, amountTolerance) < minimumAgreement {
		return RecurringSeries{}, false
	}

	last := occurrences[len(occurrences)-1]
	direction := "inflow"

	if key.outflow {
		direction = "outflow"
	}

	return RecurringSeries{
		Merchant:      last.display,
		Direction:     direction,
		Currency:      key.currency,
		TypicalAmount: closestRaw(occurrences, typical),
		Cadence:       match.name,
		Occurrences:   len(occurrences),
		FirstSeen:     formatDate(occurrences[0].date),
		LastSeen:      formatDate(last.date),
		NextExpected:  formatDate(last.date.Add(time.Duration(match.days*hoursPerDay) * time.Hour)),
	}, true
}

func matchCadence(days float64) (cadence, bool) {
	for _, c := range cadences {
		if math.Abs(days-c.days) <= c.days*cadenceTolerance {
			return c, true
		}
	}

	return cadence{}, false
}

func agreement(values []float64, target, tolerance float64) float64 {
	if len(values) == 0 {
		return 0
	}

	within := 0

	for _, value := range values {
		if math.Abs(value-target) <= target*tolerance {
			within++
		}
	}

	return float64(within) / float64(len(values))
}

func median(values []float64) float64 {
	sorted := slices.Clone(values)
	slices.Sort(sorted)

	middle := len(sorted) / 2

	if len(sorted)%2 == 1 {
		return sorted[middle]
	}

	return (sorted[middle-1] + sorted[middle]) / 2
}

func closestRaw(occurrences []occurrence, target float64) string {
	best := occurrences[0]

	for _, o := range occurrences[1:] {
		if math.Abs(math.Abs(o.amount)-target) < math.Abs(math.Abs(best.amount)-target) {
			best = o
		}
	}

	return best.raw
}

func normalizeMerchant(merchant string) string {
	return strings.Join(strings.Fields(strings.ToLower(merchant)), " ")
}
