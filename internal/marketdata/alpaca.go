package marketdata

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/tahminator/fin/internal/domain"
)

const (
	alpacaHost       = "data.alpaca.markets"
	alpacaPageLimit  = "10000"
	maxResponseBytes = 32 << 20
)

type Alpaca struct {
	httpClient *http.Client
	baseURL    *url.URL
}

func NewAlpaca(httpClient *http.Client) *Alpaca {
	return NewAlpacaAt(httpClient, &url.URL{Scheme: "https", Host: alpacaHost})
}

func NewAlpacaAt(httpClient *http.Client, baseURL *url.URL) *Alpaca {
	return &Alpaca{httpClient: httpClient, baseURL: baseURL}
}

type alpacaBar struct {
	Time  time.Time   `json:"t"`
	Close json.Number `json:"c"`
}

type alpacaBarsResponse struct {
	Bars          map[string][]alpacaBar `json:"bars"`
	NextPageToken *string                `json:"next_page_token"`
}

type alpacaTrade struct {
	Time  time.Time   `json:"t"`
	Price json.Number `json:"p"`
}

type alpacaTradesResponse struct {
	Trades map[string]alpacaTrade `json:"trades"`
}

type alpacaError struct {
	Message string `json:"message"`
}

func (a *Alpaca) Bars(ctx context.Context, creds domain.Credential, symbols []string, timeframe Timeframe, start, end time.Time) (map[string][]Bar, error) {
	bars := make(map[string][]Bar, len(symbols))
	if len(symbols) == 0 {
		return bars, nil
	}

	query := url.Values{
		"symbols":    {strings.Join(symbols, ",")},
		"timeframe":  {string(timeframe)},
		"start":      {start.UTC().Format(time.RFC3339)},
		"end":        {end.UTC().Format(time.RFC3339)},
		"limit":      {alpacaPageLimit},
		"adjustment": {"all"},
		"feed":       {"sip"},
		"sort":       {"asc"},
	}

	for {
		var response alpacaBarsResponse

		if err := a.get(ctx, creds, "/v2/stocks/bars", query, &response); err != nil {
			return nil, err
		}

		for symbol, page := range response.Bars {
			for _, bar := range page {
				bars[symbol] = append(bars[symbol], Bar{Time: bar.Time, Close: bar.Close.String()})
			}
		}

		if response.NextPageToken == nil || *response.NextPageToken == "" {
			return bars, nil
		}

		query.Set("page_token", *response.NextPageToken)
	}
}

func (a *Alpaca) LatestTrades(ctx context.Context, creds domain.Credential, symbols []string) (map[string]Trade, error) {
	trades := make(map[string]Trade, len(symbols))
	if len(symbols) == 0 {
		return trades, nil
	}

	query := url.Values{
		"symbols": {strings.Join(symbols, ",")},
		"feed":    {"iex"},
	}

	var response alpacaTradesResponse

	if err := a.get(ctx, creds, "/v2/stocks/trades/latest", query, &response); err != nil {
		return nil, err
	}

	for symbol, trade := range response.Trades {
		trades[symbol] = Trade{Time: trade.Time, Price: trade.Price.String()}
	}

	return trades, nil
}

func (a *Alpaca) get(ctx context.Context, creds domain.Credential, path string, query url.Values, target any) error {
	endpoint := a.baseURL.JoinPath(path)
	endpoint.RawQuery = query.Encode()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return fmt.Errorf("build alpaca request: %w", err)
	}

	request.Header.Set("APCA-API-KEY-ID", creds.ClientID)
	request.Header.Set("APCA-API-SECRET-KEY", creds.Secret)
	request.Header.Set("Accept", "application/json")

	response, err := a.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("%w: alpaca request: %w", domain.ErrUnavailable, err)
	}

	defer func() { _ = response.Body.Close() }()

	body := io.LimitReader(response.Body, maxResponseBytes)

	if response.StatusCode != http.StatusOK {
		var apiErr alpacaError

		_ = json.NewDecoder(body).Decode(&apiErr)

		return fmt.Errorf("%w: alpaca returned status %d: %s", domain.ErrUnavailable, response.StatusCode, apiErr.Message)
	}

	decoder := json.NewDecoder(body)
	decoder.UseNumber()

	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("%w: decode alpaca response: %w", domain.ErrUnavailable, err)
	}

	return nil
}
