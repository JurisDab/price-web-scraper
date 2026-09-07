package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"pricetracker/internal/db"
	"pricetracker/internal/product"
	"pricetracker/internal/scrapeclient"
	"pricetracker/internal/snapshot"
)

func main() {
	ctx := context.Background()

	pool, err := db.NewPool(ctx)
	if err != nil {
		log.Fatalf("connecting to database: %v", err)
	}
	defer pool.Close()

	products := product.NewRepository(pool)
	snapshots := snapshot.NewRepository(pool)
	scraper := scrapeclient.New()

	mux := http.NewServeMux()
	registerRoutes(mux, products, snapshots, scraper)

	srv := &http.Server{Addr: ":8080", Handler: mux}

	go func() {
		log.Printf("listening on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown error: %v", err)
	}
}

func registerRoutes(mux *http.ServeMux, products *product.Repository, snapshots *snapshot.Repository, scraper *scrapeclient.Client) {
	mux.HandleFunc("POST /products", createProductHandler(products))
	mux.HandleFunc("GET /products", listProductsHandler(products))
	mux.HandleFunc("GET /products/{id}", getProductHandler(products, snapshots))
	mux.HandleFunc("POST /products/{id}/scrape", scrapeProductHandler(products, snapshots, scraper))
}

func createProductHandler(products *product.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			URL      string `json:"url"`
			SiteType string `json:"site_type"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		if req.URL == "" || req.SiteType == "" {
			http.Error(w, "url and site_type are required", http.StatusBadRequest)
			return
		}

		p, err := products.Create(r.Context(), req.URL, req.SiteType)
		if err != nil {
			log.Printf("creating product: %v", err)
			http.Error(w, "failed to create product", http.StatusInternalServerError)
			return
		}

		writeJSON(w, http.StatusCreated, p)
	}
}

func listProductsHandler(products *product.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		list, err := products.List(r.Context())
		if err != nil {
			log.Printf("listing products: %v", err)
			http.Error(w, "failed to list products", http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, list)
	}
}

func getProductHandler(products *product.Repository, snapshots *snapshot.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")

		p, err := products.Get(r.Context(), id)
		if err != nil {
			http.Error(w, "product not found", http.StatusNotFound)
			return
		}

		history, err := snapshots.ListByProduct(r.Context(), id, 100)
		if err != nil {
			log.Printf("listing snapshots: %v", err)
			http.Error(w, "failed to load price history", http.StatusInternalServerError)
			return
		}

		writeJSON(w, http.StatusOK, struct {
			product.Product
			PriceHistory []snapshot.Snapshot `json:"price_history"`
		}{p, history})
	}
}

func scrapeProductHandler(products *product.Repository, snapshots *snapshot.Repository, scraper *scrapeclient.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")

		p, err := products.Get(r.Context(), id)
		if err != nil {
			http.Error(w, "product not found", http.StatusNotFound)
			return
		}

		scrapeCtx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()

		result, err := scraper.Scrape(scrapeCtx, p.URL, p.SiteType)
		if err != nil {
			log.Printf("scrape failed for product %s: %v", id, err)
			if markErr := products.MarkError(r.Context(), id, err.Error()); markErr != nil {
				log.Printf("marking product error: %v", markErr)
			}
			http.Error(w, "scrape failed", http.StatusBadGateway)
			return
		}

		s, err := snapshots.Create(r.Context(), id, result.Price, result.Currency, result.InStock)
		if err != nil {
			log.Printf("storing snapshot: %v", err)
			http.Error(w, "failed to store scrape result", http.StatusInternalServerError)
			return
		}

		var title *string
		if result.Title != "" {
			title = &result.Title
		}
		if err := products.UpdateAfterScrape(r.Context(), id, title, result.Price, result.Currency, result.InStock); err != nil {
			log.Printf("updating product after scrape: %v", err)
			http.Error(w, "failed to update product", http.StatusInternalServerError)
			return
		}

		writeJSON(w, http.StatusCreated, s)
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
