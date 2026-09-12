package alert_test

import (
	"testing"

	"pricetracker/internal/alert"
)

func floatPtr(v float64) *float64 { return &v }
func boolPtr(v bool) *bool        { return &v }

func TestDetect(t *testing.T) {
	tests := []struct {
		name            string
		previousPrice   *float64
		previousInStock *bool
		newPrice        float64
		newInStock      bool
		minPriceEver    *float64
		wantTypes       []string
	}{
		{
			name:            "first ever scrape fires nothing",
			previousPrice:   nil,
			previousInStock: nil,
			newPrice:        19.99,
			newInStock:      true,
			minPriceEver:    nil,
			wantTypes:       nil,
		},
		{
			name:            "price drop above threshold fires price_drop",
			previousPrice:   floatPtr(100),
			previousInStock: boolPtr(true),
			newPrice:        90,
			newInStock:      true,
			minPriceEver:    floatPtr(90),
			wantTypes:       []string{alert.TypePriceDrop, alert.TypeAllTimeLow},
		},
		{
			name:            "price drop below threshold fires nothing",
			previousPrice:   floatPtr(100),
			previousInStock: boolPtr(true),
			newPrice:        98,
			newInStock:      true,
			minPriceEver:    floatPtr(95),
			wantTypes:       nil,
		},
		{
			name:            "price increase fires nothing",
			previousPrice:   floatPtr(90),
			previousInStock: boolPtr(true),
			newPrice:        100,
			newInStock:      true,
			minPriceEver:    floatPtr(90),
			wantTypes:       nil,
		},
		{
			name:            "matching the all-time low again still fires",
			previousPrice:   floatPtr(50),
			previousInStock: boolPtr(true),
			newPrice:        40,
			newInStock:      true,
			minPriceEver:    floatPtr(40),
			wantTypes:       []string{alert.TypePriceDrop, alert.TypeAllTimeLow},
		},
		{
			// minPriceEver below newPrice deliberately, so this case
			// isn't also (accidentally) exercising the all_time_low rule.
			name:            "restock fires back_in_stock",
			previousPrice:   floatPtr(50),
			previousInStock: boolPtr(false),
			newPrice:        50,
			newInStock:      true,
			minPriceEver:    floatPtr(40),
			wantTypes:       []string{alert.TypeBackInStock},
		},
		{
			name:            "still out of stock fires nothing",
			previousPrice:   floatPtr(50),
			previousInStock: boolPtr(false),
			newPrice:        50,
			newInStock:      false,
			minPriceEver:    floatPtr(40),
			wantTypes:       nil,
		},
		{
			name:            "going out of stock fires nothing (only restock is an alert)",
			previousPrice:   floatPtr(50),
			previousInStock: boolPtr(true),
			newPrice:        50,
			newInStock:      false,
			minPriceEver:    floatPtr(40),
			wantTypes:       nil,
		},
		{
			name:            "price drop and restock together fire both",
			previousPrice:   floatPtr(100),
			previousInStock: boolPtr(false),
			newPrice:        80,
			newInStock:      true,
			minPriceEver:    floatPtr(90),
			wantTypes:       []string{alert.TypePriceDrop, alert.TypeAllTimeLow, alert.TypeBackInStock},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := alert.Detect(tt.previousPrice, tt.previousInStock, tt.newPrice, tt.newInStock, tt.minPriceEver)

			if len(got) != len(tt.wantTypes) {
				t.Fatalf("Detect() returned %d candidates %v, want %d %v", len(got), typesOf(got), len(tt.wantTypes), tt.wantTypes)
			}
			for i, c := range got {
				if c.Type != tt.wantTypes[i] {
					t.Errorf("candidate[%d].Type = %q, want %q", i, c.Type, tt.wantTypes[i])
				}
				if c.Message == "" {
					t.Errorf("candidate[%d].Message is empty", i)
				}
			}
		})
	}
}

func typesOf(candidates []alert.Candidate) []string {
	types := make([]string, len(candidates))
	for i, c := range candidates {
		types[i] = c.Type
	}
	return types
}
