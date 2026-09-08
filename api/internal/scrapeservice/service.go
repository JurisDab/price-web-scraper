// Package scrapeservice holds the "scrape one product" logic shared by the
// manual POST /products/{id}/scrape endpoint and the scheduler's periodic
// sweep, so there's one code path for it rather than two copies drifting
// apart.
package scrapeservice

import (
	"context"
	"time"

	"pricetracker/internal/product"
	"pricetracker/internal/scrapeclient"
	"pricetracker/internal/snapshot"
)

type Service struct {
	products  *product.Repository
	snapshots *snapshot.Repository
	scraper   *scrapeclient.Client
}

func New(products *product.Repository, snapshots *snapshot.Repository, scraper *scrapeclient.Client) *Service {
	return &Service{products: products, snapshots: snapshots, scraper: scraper}
}

// ScrapeProduct fetches the product's current price/stock via the Python
// scraper service, stores the result as a new snapshot, and updates the
// product's denormalized fields. On scrape failure, the product is marked
// status=error with the failure recorded in last_error rather than left
// stale with no explanation.
func (s *Service) ScrapeProduct(ctx context.Context, id string) (snapshot.Snapshot, error) {
	p, err := s.products.Get(ctx, id)
	if err != nil {
		return snapshot.Snapshot{}, err
	}

	scrapeCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	result, err := s.scraper.Scrape(scrapeCtx, p.URL, p.SiteType)
	if err != nil {
		if markErr := s.products.MarkError(ctx, id, err.Error()); markErr != nil {
			return snapshot.Snapshot{}, markErr
		}
		return snapshot.Snapshot{}, err
	}

	snap, err := s.snapshots.Create(ctx, id, result.Price, result.Currency, result.InStock)
	if err != nil {
		return snapshot.Snapshot{}, err
	}

	var title *string
	if result.Title != "" {
		title = &result.Title
	}
	if err := s.products.UpdateAfterScrape(ctx, id, title, result.Price, result.Currency, result.InStock); err != nil {
		return snap, err
	}

	return snap, nil
}
