package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

func count(ctx context.Context, pool *pgxpool.Pool, query string, args ...any) (int, error) {
	var total int

	err := pool.QueryRow(ctx, query, args...).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("count: %w", err)
	}

	return total, nil
}
