package domain

import (
	"time"
	"uuid"
)

type InvestmentRange string

const (
	InvestmentRangeDay         InvestmentRange = "1d"
	InvestmentRangeThreeDays   InvestmentRange = "3d"
	InvestmentRangeWeek        InvestmentRange = "1w"
	InvestmentRangeMonth       InvestmentRange = "1m"
	InvestmentRangeThreeMonths InvestmentRange = "3m"
	InvestmentRangeYear        InvestmentRange = "1y"
	InvestmentRangeAll         InvestmentRange = "all"
)

const (
	day                = 24 * time.Hour
	maxAllRangeBuckets = 180
	minAllRangeBucket  = day
)

func (r InvestmentRange) Valid() bool {
	switch r {
	case InvestmentRangeDay,
		InvestmentRangeThreeDays,
		InvestmentRangeWeek,
		InvestmentRangeMonth,
		InvestmentRangeThreeMonths,
		InvestmentRangeYear,
		InvestmentRangeAll:
		return true
	}

	return false
}

func (r InvestmentRange) Start(now time.Time) *time.Time {
	var start time.Time

	switch r {
	case InvestmentRangeDay:
		start = now.Add(-day)
	case InvestmentRangeThreeDays:
		start = now.Add(-3 * day)
	case InvestmentRangeWeek:
		start = now.Add(-7 * day)
	case InvestmentRangeMonth:
		start = now.AddDate(0, -1, 0)
	case InvestmentRangeThreeMonths:
		start = now.AddDate(0, -3, 0)
	case InvestmentRangeYear:
		start = now.AddDate(-1, 0, 0)
	case InvestmentRangeAll:
		return nil
	}

	return &start
}

func (r InvestmentRange) Bucket(earliest, now time.Time) time.Duration {
	switch r {
	case InvestmentRangeDay:
		return 15 * time.Minute
	case InvestmentRangeThreeDays:
		return time.Hour
	case InvestmentRangeWeek:
		return 3 * time.Hour
	case InvestmentRangeMonth:
		return 12 * time.Hour
	case InvestmentRangeThreeMonths:
		return day
	case InvestmentRangeYear:
		return 7 * day
	case InvestmentRangeAll:
	}

	bucket := now.Sub(earliest) / maxAllRangeBuckets

	if bucket <= minAllRangeBucket {
		return minAllRangeBucket
	}

	return bucket.Truncate(day) + day
}

type InvestmentHistoryBounds struct {
	Earliest *time.Time `db:"earliest"`
	Latest   *time.Time `db:"latest"`
}

type InvestmentHistoryRow struct {
	At        time.Time  `db:"at"`
	AccountID *uuid.UUID `db:"account_id"`
	Currency  string     `db:"currency"`
	Value     string     `db:"value"`
}

type InvestmentPoint struct {
	At    time.Time
	Value string
}

type InvestmentSeries struct {
	Currency      string
	Points        []InvestmentPoint
	First         string
	Current       string
	Change        string
	ChangePercent *string
	Limited       bool
}

type AccountInvestmentSeries struct {
	AccountID uuid.UUID
	Series    []InvestmentSeries
}

type InvestmentHistory struct {
	Range    InvestmentRange
	From     *time.Time
	Bounds   InvestmentHistoryBounds
	Total    []InvestmentSeries
	Accounts []AccountInvestmentSeries
}
