package product

import "time"

const (
	StatusPendingFirstScrape = "pending_first_scrape"
	StatusActive             = "active"
	StatusError              = "error"
)

type Product struct {
	ID            string     `json:"id"`
	URL           string     `json:"url"`
	SiteType      string     `json:"site_type"`
	Title         *string    `json:"title"`
	Status        string     `json:"status"`
	LastError     *string    `json:"last_error,omitempty"`
	CurrentPrice  *float64   `json:"current_price"`
	Currency      string     `json:"currency"`
	InStock       *bool      `json:"in_stock"`
	LastScrapedAt *time.Time `json:"last_scraped_at"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}
