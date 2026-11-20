package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Pinger struct {
	pool *pgxpool.Pool
}

func NewPinger(pool *pgxpool.Pool) *Pinger {
	return &Pinger{pool: pool}
}

const ping = `
SELECT 1;
`

func (p *Pinger) Ping(ctx context.Context) error {
	var one int

	return p.pool.QueryRow(ctx, ping).Scan(&one)
}
