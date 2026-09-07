package snapshot

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Create(ctx context.Context, productID string, price float64, currency string, inStock bool) (Snapshot, error) {
	var s Snapshot
	err := r.pool.QueryRow(ctx, `
		INSERT INTO price_snapshots (product_id, price, currency, in_stock)
		VALUES ($1, $2, $3, $4)
		RETURNING id, product_id, price, currency, in_stock, scraped_at
	`, productID, price, currency, inStock).Scan(&s.ID, &s.ProductID, &s.Price, &s.Currency, &s.InStock, &s.ScrapedAt)
	return s, err
}

func (r *Repository) ListByProduct(ctx context.Context, productID string, limit int) ([]Snapshot, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, product_id, price, currency, in_stock, scraped_at
		FROM price_snapshots
		WHERE product_id = $1
		ORDER BY scraped_at DESC
		LIMIT $2
	`, productID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	snapshots := []Snapshot{}
	for rows.Next() {
		var s Snapshot
		if err := rows.Scan(&s.ID, &s.ProductID, &s.Price, &s.Currency, &s.InStock, &s.ScrapedAt); err != nil {
			return nil, err
		}
		snapshots = append(snapshots, s)
	}
	return snapshots, rows.Err()
}
