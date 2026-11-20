package domain

import "time"

type CurrencyBalance struct {
	Currency    string `db:"currency"`
	Assets      string `db:"assets"`
	Liabilities string `db:"liabilities"`
	Net         string `db:"net"`
}

type SummaryCounts struct {
	Connections int `db:"connections"`
	Accounts    int `db:"accounts"`
}

type DailyCashFlow struct {
	Currency string    `db:"currency"`
	Day      time.Time `db:"day"`
	In       string    `db:"money_in"`
	Out      string    `db:"money_out"`
}
