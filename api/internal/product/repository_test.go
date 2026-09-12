package product_test

import (
	"context"
	"testing"

	"pricetracker/internal/product"
	"pricetracker/internal/testdb"
)

func TestCreateAndGet(t *testing.T) {
	pool := testdb.New(t)
	repo := product.NewRepository(pool)
	ctx := context.Background()

	created, err := repo.Create(ctx, "https://example.com/widget", "varle")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created.Status != product.StatusPendingFirstScrape {
		t.Errorf("status = %q, want %q", created.Status, product.StatusPendingFirstScrape)
	}
	if created.URL != "https://example.com/widget" {
		t.Errorf("url = %q, want the URL passed to Create", created.URL)
	}

	fetched, err := repo.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if fetched.ID != created.ID {
		t.Errorf("Get returned id %q, want %q", fetched.ID, created.ID)
	}
}

func TestList(t *testing.T) {
	pool := testdb.New(t)
	repo := product.NewRepository(pool)
	ctx := context.Background()

	if _, err := repo.Create(ctx, "https://example.com/a", "varle"); err != nil {
		t.Fatalf("Create a: %v", err)
	}
	if _, err := repo.Create(ctx, "https://example.com/b", "skytech"); err != nil {
		t.Fatalf("Create b: %v", err)
	}

	list, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("List returned %d products, want 2", len(list))
	}
}

func TestUpdateAfterScrapeSetsActiveAndPrice(t *testing.T) {
	pool := testdb.New(t)
	repo := product.NewRepository(pool)
	ctx := context.Background()

	created, err := repo.Create(ctx, "https://example.com/widget", "varle")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	title := "Widget 3000"
	if err := repo.UpdateAfterScrape(ctx, created.ID, &title, 19.99, "EUR", true); err != nil {
		t.Fatalf("UpdateAfterScrape: %v", err)
	}

	updated, err := repo.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if updated.Status != product.StatusActive {
		t.Errorf("status = %q, want %q", updated.Status, product.StatusActive)
	}
	if updated.Title == nil || *updated.Title != title {
		t.Errorf("title = %v, want %q", updated.Title, title)
	}
	if updated.CurrentPrice == nil || *updated.CurrentPrice != 19.99 {
		t.Errorf("current_price = %v, want 19.99", updated.CurrentPrice)
	}
	if updated.InStock == nil || !*updated.InStock {
		t.Errorf("in_stock = %v, want true", updated.InStock)
	}
}

func TestMarkErrorSetsErrorStatus(t *testing.T) {
	pool := testdb.New(t)
	repo := product.NewRepository(pool)
	ctx := context.Background()

	created, err := repo.Create(ctx, "https://example.com/widget", "varle")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := repo.MarkError(ctx, created.ID, "connection refused"); err != nil {
		t.Fatalf("MarkError: %v", err)
	}

	updated, err := repo.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if updated.Status != product.StatusError {
		t.Errorf("status = %q, want %q", updated.Status, product.StatusError)
	}
	if updated.LastError == nil || *updated.LastError != "connection refused" {
		t.Errorf("last_error = %v, want %q", updated.LastError, "connection refused")
	}
}
