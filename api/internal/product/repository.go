package product

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

const selectColumns = `id, url, site_type, title, status, last_error, current_price, currency, in_stock, last_scraped_at, created_at, updated_at`

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func scanProduct(row interface {
	Scan(dest ...any) error
}) (Product, error) {
	var p Product
	err := row.Scan(&p.ID, &p.URL, &p.SiteType, &p.Title, &p.Status, &p.LastError,
		&p.CurrentPrice, &p.Currency, &p.InStock, &p.LastScrapedAt, &p.CreatedAt, &p.UpdatedAt)
	return p, err
}

func (r *Repository) Create(ctx context.Context, url, siteType string) (Product, error) {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO products (url, site_type)
		VALUES ($1, $2)
		RETURNING `+selectColumns, url, siteType)
	return scanProduct(row)
}

func (r *Repository) Get(ctx context.Context, id string) (Product, error) {
	row := r.pool.QueryRow(ctx, `SELECT `+selectColumns+` FROM products WHERE id = $1`, id)
	return scanProduct(row)
}

func (r *Repository) List(ctx context.Context) ([]Product, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+selectColumns+` FROM products ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	products := []Product{}
	for rows.Next() {
		p, err := scanProduct(rows)
		if err != nil {
			return nil, err
		}
		products = append(products, p)
	}
	return products, rows.Err()
}

func (r *Repository) UpdateAfterScrape(ctx context.Context, id string, title *string, price float64, currency string, inStock bool) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE products
		SET title = COALESCE($2, title),
		    status = '`+StatusActive+`',
		    last_error = NULL,
		    current_price = $3,
		    currency = $4,
		    in_stock = $5,
		    last_scraped_at = now(),
		    updated_at = now()
		WHERE id = $1
	`, id, title, price, currency, inStock)
	return err
}

func (r *Repository) MarkError(ctx context.Context, id string, errMsg string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE products SET status = '`+StatusError+`', last_error = $2, updated_at = now() WHERE id = $1
	`, id, errMsg)
	return err
}
