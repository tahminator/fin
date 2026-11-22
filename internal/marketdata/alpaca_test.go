package marketdata_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/tahminator/fin/internal/domain"
	"github.com/tahminator/fin/internal/marketdata"
)

var testCreds = domain.Credential{Provider: domain.ProviderAlpaca, ClientID: "key-id", Secret: "secret-key"}

func newAlpaca(t *testing.T, handler http.HandlerFunc) *marketdata.Alpaca {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	baseURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("parse server URL: %v", err)
	}

	return marketdata.NewAlpacaAt(server.Client(), baseURL)
}

func TestBarsFollowsPagesAndSendsCredentials(t *testing.T) {
	var queries []url.Values

	alpaca := newAlpaca(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/stocks/bars" {
			t.Errorf("path = %s, want /v2/stocks/bars", r.URL.Path)
		}

		if r.Header.Get("APCA-API-KEY-ID") != "key-id" || r.Header.Get("APCA-API-SECRET-KEY") != "secret-key" {
			t.Errorf("missing Alpaca credentials in headers %v", r.Header)
		}

		queries = append(queries, r.URL.Query())

		if r.URL.Query().Get("page_token") == "" {
			_, _ = w.Write([]byte(`{"bars":{"AAPL":[{"t":"2026-11-20T15:00:00Z","c":190.12}]},"next_page_token":"next"}`))

			return
		}

		_, _ = w.Write([]byte(`{"bars":{"AAPL":[{"t":"2026-11-20T15:01:00Z","c":190.5}],"MSFT":[{"t":"2026-11-20T15:01:00Z","c":410}]},"next_page_token":null}`))
	})

	start := time.Date(2026, 11, 20, 14, 0, 0, 0, time.UTC)
	end := start.Add(2 * time.Hour)

	bars, err := alpaca.Bars(t.Context(), testCreds, []string{"AAPL", "MSFT"}, marketdata.TimeframeMinute, start, end)
	if err != nil {
		t.Fatalf("Bars: %v", err)
	}

	if len(queries) != 2 || queries[1].Get("page_token") != "next" {
		t.Fatalf("queries = %v, want a second page request with page_token=next", queries)
	}

	first := queries[0]
	if first.Get("symbols") != "AAPL,MSFT" || first.Get("timeframe") != "1Min" || first.Get("feed") != "sip" || first.Get("start") != "2026-11-20T14:00:00Z" || first.Get("end") != "2026-11-20T16:00:00Z" {
		t.Errorf("first query = %v", first)
	}

	if got := bars["AAPL"]; len(got) != 2 || got[0].Close != "190.12" || got[1].Close != "190.5" {
		t.Errorf("AAPL bars = %+v", got)
	}

	if got := bars["MSFT"]; len(got) != 1 || got[0].Close != "410" {
		t.Errorf("MSFT bars = %+v", got)
	}
}

func TestLatestTradesUsesIEX(t *testing.T) {
	alpaca := newAlpaca(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/stocks/trades/latest" || r.URL.Query().Get("feed") != "iex" {
			t.Errorf("request = %s?%s", r.URL.Path, r.URL.RawQuery)
		}

		_, _ = w.Write([]byte(`{"trades":{"AAPL":{"t":"2026-11-20T20:59:59Z","p":191.01}}}`))
	})

	trades, err := alpaca.LatestTrades(t.Context(), testCreds, []string{"AAPL"})
	if err != nil {
		t.Fatalf("LatestTrades: %v", err)
	}

	if trades["AAPL"].Price != "191.01" {
		t.Errorf("AAPL trade = %+v", trades["AAPL"])
	}
}

func TestErrorsAreUnavailable(t *testing.T) {
	alpaca := newAlpaca(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"message":"forbidden"}`))
	})

	_, err := alpaca.LatestTrades(t.Context(), testCreds, []string{"AAPL"})
	if !errors.Is(err, domain.ErrUnavailable) {
		t.Fatalf("error = %v, want ErrUnavailable", err)
	}
}

func TestNoSymbolsMakesNoRequest(t *testing.T) {
	alpaca := newAlpaca(t, func(http.ResponseWriter, *http.Request) {
		t.Error("request made without symbols")
	})

	bars, err := alpaca.Bars(t.Context(), testCreds, nil, marketdata.TimeframeDay, time.Now(), time.Now())
	if err != nil || len(bars) != 0 {
		t.Fatalf("Bars = %v, %v; want empty", bars, err)
	}
}
