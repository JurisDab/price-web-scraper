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
	"pricetracker/internal/scheduler"
	"pricetracker/internal/scrapeclient"
	"pricetracker/internal/scrapeservice"
	"pricetracker/internal/snapshot"
)

func main() {
	ctx, cancelBackground := context.WithCancel(context.Background())
	defer cancelBackground()

	pool, err := db.NewPool(ctx)
	if err != nil {
		log.Fatalf("connecting to database: %v", err)
	}
	defer pool.Close()

	products := product.NewRepository(pool)
	snapshots := snapshot.NewRepository(pool)
	scraper := scrapeclient.New()
	scrapeSvc := scrapeservice.New(products, snapshots, scraper)

	sched := scheduler.New(products, scrapeSvc)
	go sched.Run(ctx)

	mux := http.NewServeMux()
	registerRoutes(mux, products, snapshots, scrapeSvc)

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

	cancelBackground() // stops the scheduler loop

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown error: %v", err)
	}
}

func registerRoutes(mux *http.ServeMux, products *product.Repository, snapshots *snapshot.Repository, scrapeSvc *scrapeservice.Service) {
	mux.HandleFunc("POST /products", createProductHandler(products))
	mux.HandleFunc("GET /products", listProductsHandler(products))
	mux.HandleFunc("GET /products/{id}", getProductHandler(products, snapshots))
	mux.HandleFunc("POST /products/{id}/scrape", scrapeProductHandler(products, scrapeSvc))
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

func scrapeProductHandler(products *product.Repository, scrapeSvc *scrapeservice.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")

		if _, err := products.Get(r.Context(), id); err != nil {
			http.Error(w, "product not found", http.StatusNotFound)
			return
		}

		s, err := scrapeSvc.ScrapeProduct(r.Context(), id)
		if err != nil {
			log.Printf("scrape failed for product %s: %v", id, err)
			http.Error(w, "scrape failed", http.StatusBadGateway)
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
