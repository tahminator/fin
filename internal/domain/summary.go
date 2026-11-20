package domain

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
