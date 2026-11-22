package marketdata

import (
	"context"
	"time"

	"github.com/tahminator/fin/internal/domain"
)

type Timeframe string

const (
	TimeframeMinute        Timeframe = "1Min"
	TimeframeFiveMinutes   Timeframe = "5Min"
	TimeframeFifteenMinute Timeframe = "15Min"
	TimeframeHour          Timeframe = "1Hour"
	TimeframeDay           Timeframe = "1Day"
)

type Bar struct {
	Time  time.Time
	Close string
}

type Trade struct {
	Time  time.Time
	Price string
}

type Provider interface {
	Bars(ctx context.Context, creds domain.Credential, symbols []string, timeframe Timeframe, start, end time.Time) (map[string][]Bar, error)
	LatestTrades(ctx context.Context, creds domain.Credential, symbols []string) (map[string]Trade, error)
}
