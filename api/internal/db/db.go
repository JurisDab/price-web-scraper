package db

import (
	"context"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

func NewPool(ctx context.Context) (*pgxpool.Pool, error) {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		url = "postgres://user:pass@localhost:5434/pricetracker?sslmode=disable"
	}
	return pgxpool.New(ctx, url)
}
