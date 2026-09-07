package snapshot

import "time"

type Snapshot struct {
	ID        int64     `json:"id"`
	ProductID string    `json:"product_id"`
	Price     float64   `json:"price"`
	Currency  string    `json:"currency"`
	InStock   bool      `json:"in_stock"`
	ScrapedAt time.Time `json:"scraped_at"`
}
