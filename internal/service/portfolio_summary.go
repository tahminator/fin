package service

import (
	"cmp"
	"math/big"
	"slices"
	"time"
	"uuid"

	"github.com/tahminator/fin/internal/domain"
)

type performanceTotals struct {
	start *big.Rat
	end   *big.Rat
	gain  *big.Rat
	basis *big.Rat
}

func newPerformanceTotals() *performanceTotals {
	return &performanceTotals{start: new(big.Rat), end: new(big.Rat), gain: new(big.Rat), basis: new(big.Rat)}
}

func (t *performanceTotals) add(p *position, startDay time.Time) {
	gain, basis := p.gain(startDay)

	t.start.Add(t.start, p.values[0])
	t.end.Add(t.end, p.values[len(p.values)-1])
	t.gain.Add(t.gain, gain)
	t.basis.Add(t.basis, basis)
}

func (t *performanceTotals) change() (string, *string) {
	return gainOf(t.gain, t.basis)
}

func gainOf(gain, basis *big.Rat) (string, *string) {
	if basis.Sign() <= 0 {
		return formatMoney(gain), nil
	}

	percent := new(big.Rat).Mul(new(big.Rat).Quo(gain, basis), big.NewRat(100, 1))
	formatted := formatMoney(percent)

	return formatMoney(gain), &formatted
}

func summarize(portfolioRange domain.PortfolioRange, start time.Time, times []time.Time, positions []*position, prices marketPrices) domain.PortfolioPerformance {
	startDay := dayOf(start)
	totals := newPerformanceTotals()
	series := make([]*big.Rat, len(times))

	for i := range series {
		series[i] = new(big.Rat)
	}

	for _, p := range positions {
		totals.add(p, startDay)

		for i, value := range p.values {
			series[i].Add(series[i], value)
		}
	}

	points := make([]domain.PortfolioPoint, len(times))
	for i, t := range times {
		points[i] = domain.PortfolioPoint{Time: t, Value: formatMoney(series[i])}
	}

	change, changePercent := totals.change()

	return domain.PortfolioPerformance{
		Range:         portfolioRange,
		Currency:      domain.PortfolioCurrency,
		Start:         start,
		Points:        points,
		StartValue:    formatMoney(series[0]),
		EndValue:      formatMoney(series[len(series)-1]),
		Change:        change,
		ChangePercent: changePercent,
		Positions:     summarizeSecurities(positions, times, startDay, prices),
		Accounts:      summarizeAccounts(positions, startDay),
		MarketData:    prices.status,
	}
}

func summarizeSecurities(positions []*position, times []time.Time, startDay time.Time, prices marketPrices) []domain.PositionPerformance {
	type aggregate struct {
		performance domain.PositionPerformance
		totals      *performanceTotals
		quantity    *big.Rat
		costBasis   *big.Rat
	}

	bySecurity := make(map[uuid.UUID]*aggregate)
	order := []uuid.UUID{}
	lastTime := times[len(times)-1]

	for _, p := range positions {
		a, ok := bySecurity[p.securityID]
		if !ok {
			a = &aggregate{
				performance: domain.PositionPerformance{SecurityID: p.securityID, Name: p.name, Ticker: p.ticker, SecurityType: p.securityType},
				totals:      newPerformanceTotals(),
				quantity:    new(big.Rat),
			}
			bySecurity[p.securityID] = a
			order = append(order, p.securityID)
		}

		a.performance.MarketPriced = a.performance.MarketPriced || p.marketPriced(prices)
		a.quantity.Add(a.quantity, p.quantityAt(lastTime))
		a.totals.add(p, startDay)

		if costBasis := parseOptionalDecimal(p.costBasis); costBasis != nil {
			if a.costBasis == nil {
				a.costBasis = new(big.Rat)
			}

			a.costBasis.Add(a.costBasis, costBasis)
		}
	}

	result := make([]domain.PositionPerformance, 0, len(order))

	for _, securityID := range order {
		a := bySecurity[securityID]
		performance := a.performance

		performance.Quantity = formatQuantity(a.quantity)
		performance.Value = formatMoney(a.totals.end)
		performance.StartValue = formatMoney(a.totals.start)
		performance.Change, performance.ChangePercent = a.totals.change()
		performance.Price = "0.00"

		if a.quantity.Sign() > 0 {
			performance.Price = formatMoney(new(big.Rat).Quo(a.totals.end, a.quantity))
		}

		if a.costBasis != nil {
			costBasis := formatMoney(a.costBasis)
			performance.CostBasis = &costBasis
		}

		if a.costBasis != nil && a.costBasis.Sign() > 0 && a.quantity.Sign() > 0 {
			averageCost := formatMoney(new(big.Rat).Quo(a.costBasis, a.quantity))
			totalReturn, totalReturnPercent := changeOf(a.costBasis, a.totals.end)
			performance.AverageCost = &averageCost
			performance.TotalReturn = &totalReturn
			performance.TotalReturnPercent = totalReturnPercent
		}

		result = append(result, performance)
	}

	slices.SortStableFunc(result, func(a, b domain.PositionPerformance) int {
		return parseDecimal(b.Value).Cmp(parseDecimal(a.Value))
	})

	return result
}

func summarizeAccounts(positions []*position, startDay time.Time) []domain.AccountPerformance {
	type aggregate struct {
		name   string
		totals *performanceTotals
	}

	byAccount := make(map[uuid.UUID]*aggregate)

	for _, p := range positions {
		a, ok := byAccount[p.accountID]
		if !ok {
			a = &aggregate{name: p.accountName, totals: newPerformanceTotals()}
			byAccount[p.accountID] = a
		}

		a.totals.add(p, startDay)
	}

	result := make([]domain.AccountPerformance, 0, len(byAccount))

	for accountID, a := range byAccount {
		change, changePercent := a.totals.change()

		result = append(result, domain.AccountPerformance{
			AccountID:     accountID,
			Name:          a.name,
			Value:         formatMoney(a.totals.end),
			StartValue:    formatMoney(a.totals.start),
			Change:        change,
			ChangePercent: changePercent,
		})
	}

	slices.SortFunc(result, func(a, b domain.AccountPerformance) int {
		return cmp.Or(parseDecimal(b.Value).Cmp(parseDecimal(a.Value)), cmp.Compare(a.Name, b.Name))
	})

	return result
}
