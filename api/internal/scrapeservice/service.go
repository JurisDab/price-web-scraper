package scrapeservice

import (
	"context"
	"log"
	"time"

	"pricetracker/internal/alert"
	"pricetracker/internal/product"
	"pricetracker/internal/scrapeclient"
	"pricetracker/internal/snapshot"
)

type Service struct {
	products  *product.Repository
	snapshots *snapshot.Repository
	alerts    *alert.Repository
	scraper   *scrapeclient.Client
}

func New(products *product.Repository, snapshots *snapshot.Repository, alerts *alert.Repository, scraper *scrapeclient.Client) *Service {
	return &Service{products: products, snapshots: snapshots, alerts: alerts, scraper: scraper}
}

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

	minPriceEver, err := s.snapshots.MinPrice(ctx, id)
	if err != nil {
		return snapshot.Snapshot{}, err
	}

	for _, c := range alert.Detect(p.CurrentPrice, p.InStock, result.Price, result.InStock, minPriceEver) {
		if _, err := s.alerts.Create(ctx, id, c.Type, c.Message, c.Price); err != nil {
			log.Printf("scrapeservice: creating %s alert for product %s: %v", c.Type, id, err)
		}
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
