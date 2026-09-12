package alert

import "time"

const (
	TypePriceDrop   = "price_drop"
	TypeAllTimeLow  = "all_time_low"
	TypeBackInStock = "back_in_stock"
	TypeDealScore   = "deal_score"
)

type Alert struct {
	ID        int64      `json:"id"`
	ProductID string     `json:"product_id"`
	Type      string     `json:"type"`
	Message   string     `json:"message"`
	Price     *float64   `json:"price,omitempty"`
	DealScore *int       `json:"deal_score,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	ReadAt    *time.Time `json:"read_at,omitempty"`
}
