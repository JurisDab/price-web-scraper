// Package scheduler periodically re-scrapes every tracked product.
// Concurrency is capped per domain (not globally): products all get
// scraped promptly since different sites run in parallel, but no single
// site gets hit with more than a handful of simultaneous requests.
package scheduler

import (
	"context"
	"log"
	"net/url"
	"sync"
	"time"

	"pricetracker/internal/product"
	"pricetracker/internal/scrapeservice"
)

const (
	DefaultInterval             = 6 * time.Hour
	DefaultPerDomainConcurrency = 2
)

type Scheduler struct {
	products             *product.Repository
	scraper              *scrapeservice.Service
	interval             time.Duration
	perDomainConcurrency int
}

func New(products *product.Repository, scraper *scrapeservice.Service) *Scheduler {
	return &Scheduler{
		products:             products,
		scraper:              scraper,
		interval:             DefaultInterval,
		perDomainConcurrency: DefaultPerDomainConcurrency,
	}
}

// Run sweeps once immediately, then again on every tick, until ctx is
// canceled. Intended to run in its own goroutine alongside the HTTP server.
func (s *Scheduler) Run(ctx context.Context) {
	s.sweep(ctx)

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.sweep(ctx)
		}
	}
}

func (s *Scheduler) sweep(ctx context.Context) {
	products, err := s.products.List(ctx)
	if err != nil {
		log.Printf("scheduler: listing products: %v", err)
		return
	}

	byDomain := make(map[string][]product.Product)
	for _, p := range products {
		domain := domainOf(p.URL)
		byDomain[domain] = append(byDomain[domain], p)
	}

	var wg sync.WaitGroup
	for domain, items := range byDomain {
		wg.Add(1)
		go func(domain string, items []product.Product) {
			defer wg.Done()
			s.sweepDomain(ctx, items)
		}(domain, items)
	}
	wg.Wait()
}

func (s *Scheduler) sweepDomain(ctx context.Context, items []product.Product) {
	sem := make(chan struct{}, s.perDomainConcurrency)
	var wg sync.WaitGroup

	for _, p := range items {
		wg.Add(1)
		sem <- struct{}{}
		go func(p product.Product) {
			defer wg.Done()
			defer func() { <-sem }()

			if _, err := s.scraper.ScrapeProduct(ctx, p.ID); err != nil {
				log.Printf("scheduler: scraping product %s (%s): %v", p.ID, p.URL, err)
			}
		}(p)
	}
	wg.Wait()
}

func domainOf(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	return u.Hostname()
}
