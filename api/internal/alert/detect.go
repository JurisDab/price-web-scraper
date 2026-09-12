package alert

import "fmt"

// PriceDropThresholdPct is the naive fixed-percentage baseline for a
// price_drop alert. The planned LLM deal-score step replaces/augments this
// with real judgment about whether a drop is a genuine deal, rather than
// an arbitrary cutoff — this threshold is deliberately simple until then.
const PriceDropThresholdPct = 5.0

// Candidate is an alert Detect wants created, before it has an ID or
// timestamp — the caller (which owns the DB) turns it into a real Alert.
type Candidate struct {
	Type    string
	Message string
	Price   *float64
}

// Detect compares a new scrape result against the product's previous state
// and returns every alert that should fire. previousPrice/previousInStock
// are nil on a product's first-ever scrape, in which case nothing fires —
// there's nothing yet to compare against. minPriceEver is the lowest price
// recorded across all prior snapshots (nil if there are none).
func Detect(previousPrice *float64, previousInStock *bool, newPrice float64, newInStock bool, minPriceEver *float64) []Candidate {
	var candidates []Candidate

	if previousPrice != nil && *previousPrice > 0 {
		dropPct := (*previousPrice - newPrice) / *previousPrice * 100
		if dropPct >= PriceDropThresholdPct {
			candidates = append(candidates, Candidate{
				Type:    TypePriceDrop,
				Message: fmt.Sprintf("Price dropped %.0f%% to €%.2f (was €%.2f)", dropPct, newPrice, *previousPrice),
				Price:   &newPrice,
			})
		}
	}

	if minPriceEver != nil && newPrice <= *minPriceEver {
		candidates = append(candidates, Candidate{
			Type:    TypeAllTimeLow,
			Message: fmt.Sprintf("New all-time low: €%.2f", newPrice),
			Price:   &newPrice,
		})
	}

	if previousInStock != nil && !*previousInStock && newInStock {
		candidates = append(candidates, Candidate{
			Type:    TypeBackInStock,
			Message: "Back in stock",
			Price:   &newPrice,
		})
	}

	return candidates
}
