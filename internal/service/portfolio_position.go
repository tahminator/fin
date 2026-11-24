package service

import (
	"math/big"
	"slices"
	"time"
	"uuid"

	"github.com/tahminator/fin/internal/domain"
)

type positionChange struct {
	day      time.Time
	quantity *big.Rat
	amount   *big.Rat
}

type position struct {
	accountID    uuid.UUID
	accountName  string
	securityID   uuid.UUID
	name         *string
	ticker       *string
	securityType *string
	cash         bool
	quantity     *big.Rat
	price        *big.Rat
	costBasis    *string
	changes      []positionChange
	values       []*big.Rat
}

type positionKey struct {
	accountID  uuid.UUID
	securityID uuid.UUID
}

func inPortfolioCurrency(currency *string) bool {
	return currency == nil || *currency == domain.PortfolioCurrency
}

func isCash(securityType *string) bool {
	return securityType != nil && *securityType == cashSecurityType
}

func optionalDecimal(value *string) *big.Rat {
	if value == nil {
		return new(big.Rat)
	}

	return parseDecimal(*value)
}

func buildPositions(holdings []domain.InvestmentHolding, transactions []domain.InvestmentTransaction, filter domain.PortfolioFilter) []*position {
	positions := make(map[positionKey]*position)

	for _, holding := range holdings {
		if !inPortfolioCurrency(holding.Currency) || (filter.SecurityID != nil && holding.SecurityID != *filter.SecurityID) {
			continue
		}

		positions[positionKey{holding.AccountID, holding.SecurityID}] = &position{
			accountID:    holding.AccountID,
			accountName:  holding.AccountName,
			securityID:   holding.SecurityID,
			name:         holding.SecurityName,
			ticker:       holding.Ticker,
			securityType: holding.SecurityType,
			cash:         isCash(holding.SecurityType),
			quantity:     parseDecimal(holding.Quantity),
			price:        parseDecimal(holding.InstitutionPrice),
			costBasis:    holding.CostBasis,
		}
	}

	var accountTransactions []domain.InvestmentTransaction

	for _, transaction := range transactions {
		if !inPortfolioCurrency(transaction.Currency) || (filter.AccountID != nil && transaction.AccountID != *filter.AccountID) {
			continue
		}

		accountTransactions = append(accountTransactions, transaction)

		if transaction.SecurityID == nil || isCash(transaction.SecurityType) || (filter.SecurityID != nil && *transaction.SecurityID != *filter.SecurityID) {
			continue
		}

		p := securityPosition(positions, transaction)
		p.changes = append(p.changes, positionChange{
			day:      dayOf(transaction.PostedOn),
			quantity: optionalDecimal(transaction.Quantity),
			amount:   parseDecimal(transaction.Amount),
		})
	}

	result := make([]*position, 0, len(positions))

	for _, p := range positions {
		if p.cash {
			p.changes = cashChanges(p.accountID, accountTransactions)
		}

		result = append(result, p)
	}

	return result
}

func securityPosition(positions map[positionKey]*position, transaction domain.InvestmentTransaction) *position {
	key := positionKey{transaction.AccountID, *transaction.SecurityID}

	p, ok := positions[key]
	if !ok {
		p = &position{
			accountID:    transaction.AccountID,
			accountName:  transaction.AccountName,
			securityID:   *transaction.SecurityID,
			name:         transaction.SecurityName,
			ticker:       transaction.Ticker,
			securityType: transaction.SecurityType,
			quantity:     new(big.Rat),
			price:        new(big.Rat),
		}
		positions[key] = p
	}

	if p.price.Sign() == 0 && transaction.Price != nil {
		p.price = parseDecimal(*transaction.Price)
	}

	return p
}

func cashChanges(accountID uuid.UUID, transactions []domain.InvestmentTransaction) []positionChange {
	var changes []positionChange

	for _, transaction := range transactions {
		if transaction.AccountID == accountID {
			changes = append(changes, positionChange{
				day:      dayOf(transaction.PostedOn),
				quantity: parseDecimal(transaction.Amount),
				amount:   new(big.Rat),
			})
		}
	}

	return changes
}

func (p *position) quantityAt(t time.Time) *big.Rat {
	day := dayOf(t)
	quantity := new(big.Rat).Set(p.quantity)

	for _, change := range p.changes {
		if change.day.After(day) {
			quantity.Sub(quantity, change.quantity)
		}
	}

	if quantity.Sign() < 0 {
		return new(big.Rat)
	}

	return quantity
}

func (p *position) heldIn(startDay time.Time) bool {
	if slices.ContainsFunc(p.values, func(value *big.Rat) bool { return value.Sign() != 0 }) {
		return true
	}

	return slices.ContainsFunc(p.changes, func(change positionChange) bool {
		return change.day.After(startDay)
	})
}

func (p *position) flowsAfter(startDay time.Time) (*big.Rat, *big.Rat) {
	flows, purchases := new(big.Rat), new(big.Rat)

	for _, change := range p.changes {
		if !change.day.After(startDay) {
			continue
		}

		flows.Add(flows, change.amount)

		if change.amount.Sign() < 0 {
			purchases.Sub(purchases, change.amount)
		}
	}

	return flows, purchases
}

func (p *position) gain(startDay time.Time) (*big.Rat, *big.Rat) {
	if p.cash {
		return new(big.Rat), new(big.Rat)
	}

	start, end := p.values[0], p.values[len(p.values)-1]
	flows, purchases := p.flowsAfter(startDay)

	gain := new(big.Rat).Sub(end, start)
	gain.Add(gain, flows)

	return gain, new(big.Rat).Add(start, purchases)
}

func (p *position) symbol() string {
	return marketSymbol(p.ticker, p.securityType)
}

func (p *position) marketPriceChange(prices marketPrices, start, end time.Time, seriesEnd seriesEnd) (*string, *string) {
	startPrice, ok := p.marketPriceAt(prices, start, seriesEnd)
	if !ok {
		return nil, nil
	}

	endPrice, ok := p.marketPriceAt(prices, end, seriesEnd)
	if !ok {
		return nil, nil
	}

	change, percent := changeOf(startPrice, endPrice)

	return &change, percent
}

func (p *position) marketPricePoints(times []time.Time, prices marketPrices, seriesEnd seriesEnd) []domain.PortfolioPoint {
	points := make([]domain.PortfolioPoint, 0, len(times))

	for _, t := range times {
		price, ok := p.marketPriceAt(prices, t, seriesEnd)
		if ok {
			points = append(points, domain.PortfolioPoint{Time: t, Value: formatMoney(price)})
		}
	}

	return points
}

func (p *position) marketPriceAt(prices marketPrices, t time.Time, seriesEnd seriesEnd) (*big.Rat, bool) {
	if prices.status != domain.MarketDataLive {
		return nil, false
	}

	symbol := p.symbol()
	bars := prices.bars[symbol]

	if trade, ok := prices.latest[symbol]; ok && symbol != "" && t.Equal(seriesEnd.time) && !trade.Time.After(t) {
		if len(bars) == 0 || trade.Time.After(bars[len(bars)-1].Time) {
			return parseDecimal(trade.Price), true
		}
	}

	return closeAt(bars, t)
}
