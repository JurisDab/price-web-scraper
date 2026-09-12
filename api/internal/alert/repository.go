package alert

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

const selectColumns = `id, product_id, type, message, price, deal_score, created_at, read_at`

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Create(ctx context.Context, productID, alertType, message string, price *float64) (Alert, error) {
	var a Alert
	err := r.pool.QueryRow(ctx, `
		INSERT INTO alerts (product_id, type, message, price)
		VALUES ($1, $2, $3, $4)
		RETURNING `+selectColumns, productID, alertType, message, price,
	).Scan(&a.ID, &a.ProductID, &a.Type, &a.Message, &a.Price, &a.DealScore, &a.CreatedAt, &a.ReadAt)
	return a, err
}

func (r *Repository) ListByProduct(ctx context.Context, productID string, limit int) ([]Alert, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT `+selectColumns+`
		FROM alerts
		WHERE product_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`, productID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanAlerts(rows)
}

func (r *Repository) List(ctx context.Context, limit int) ([]Alert, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT `+selectColumns+`
		FROM alerts
		ORDER BY created_at DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanAlerts(rows)
}

func scanAlerts(rows interface {
	Next() bool
	Scan(dest ...any) error
	Err() error
}) ([]Alert, error) {
	alerts := []Alert{}
	for rows.Next() {
		var a Alert
		if err := rows.Scan(&a.ID, &a.ProductID, &a.Type, &a.Message, &a.Price, &a.DealScore, &a.CreatedAt, &a.ReadAt); err != nil {
			return nil, err
		}
		alerts = append(alerts, a)
	}
	return alerts, rows.Err()
}
