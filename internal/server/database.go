package server

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

func initDb(connectionString string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(context.Background(), connectionString)
	if err != nil {
		return nil, fmt.Errorf("connect to database: %w", err)
	}

	return pool, nil
}
