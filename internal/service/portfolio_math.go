package service

import (
	"math/big"
	"slices"
	"strings"
	"time"

	"github.com/tahminator/fin/internal/marketdata"
)

var marketPricedTypes = []string{"equity", "etf"}

func parseDecimal(value string) *big.Rat {
	parsed, ok := new(big.Rat).SetString(value)
	if !ok {
		return new(big.Rat)
	}

	return parsed
}

func parseOptionalDecimal(value *string) *big.Rat {
	if value == nil {
		return nil
	}

	parsed, ok := new(big.Rat).SetString(*value)
	if !ok {
		return nil
	}

	return parsed
}

func formatMoney(value *big.Rat) string {
	formatted := value.FloatString(2)
	if formatted == "-0.00" {
		return "0.00"
	}

	return formatted
}

func formatQuantity(value *big.Rat) string {
	formatted := strings.TrimRight(strings.TrimRight(value.FloatString(6), "0"), ".")
	if formatted == "" || formatted == "-0" {
		return "0"
	}

	return formatted
}

func changeOf(start, end *big.Rat) (string, *string) {
	change := new(big.Rat).Sub(end, start)

	if start.Sign() <= 0 {
		return formatMoney(change), nil
	}

	percent := new(big.Rat).Mul(new(big.Rat).Quo(change, start), big.NewRat(100, 1))
	formatted := formatMoney(percent)

	return formatMoney(change), &formatted
}

func marketSymbol(ticker, securityType *string) string {
	if ticker == nil || securityType == nil || !slices.Contains(marketPricedTypes, *securityType) {
		return ""
	}

	symbol := strings.ToUpper(strings.TrimSpace(*ticker))
	if symbol == "" || strings.ContainsAny(symbol, ": /") {
		return ""
	}

	return symbol
}

func dayOf(t time.Time) time.Time {
	t = t.UTC()

	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

func closeAt(bars []marketdata.Bar, t time.Time) (*big.Rat, bool) {
	index, found := slices.BinarySearchFunc(bars, t, func(bar marketdata.Bar, target time.Time) int {
		return bar.Time.Compare(target)
	})

	if found {
		return parseDecimal(bars[index].Close), true
	}

	if index == 0 {
		if len(bars) == 0 {
			return nil, false
		}

		return parseDecimal(bars[0].Close), true
	}

	return parseDecimal(bars[index-1].Close), true
}

func seriesTimes(start, now time.Time, bars map[string][]marketdata.Bar) []time.Time {
	times := []time.Time{start}

	for _, symbolBars := range bars {
		for _, bar := range symbolBars {
			if bar.Time.After(start) && bar.Time.Before(now) {
				times = append(times, bar.Time)
			}
		}
	}

	times = append(times, now)

	slices.SortFunc(times, func(a, b time.Time) int {
		return a.Compare(b)
	})

	return slices.CompactFunc(times, func(a, b time.Time) bool {
		return a.Equal(b)
	})
}
