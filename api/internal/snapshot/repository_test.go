package snapshot_test

import (
	"context"
	"testing"

	"pricetracker/internal/product"
	"pricetracker/internal/snapshot"
	"pricetracker/internal/testdb"
)

func TestCreateAndListByProduct(t *testing.T) {
	pool := testdb.New(t)
	products := product.NewRepository(pool)
	snapshots := snapshot.NewRepository(pool)
	ctx := context.Background()

	p, err := products.Create(ctx, "https://example.com/widget", "varle")
	if err != nil {
		t.Fatalf("Create product: %v", err)
	}

	if _, err := snapshots.Create(ctx, p.ID, 19.99, "EUR", true); err != nil {
		t.Fatalf("Create snapshot 1: %v", err)
	}
	if _, err := snapshots.Create(ctx, p.ID, 17.49, "EUR", true); err != nil {
		t.Fatalf("Create snapshot 2: %v", err)
	}

	list, err := snapshots.ListByProduct(ctx, p.ID, 100)
	if err != nil {
		t.Fatalf("ListByProduct: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("ListByProduct returned %d snapshots, want 2", len(list))
	}
	// Most recent first.
	if list[0].Price != 17.49 {
		t.Errorf("list[0].Price = %v, want 17.49 (most recent first)", list[0].Price)
	}
}

func TestListByProductRespectsLimit(t *testing.T) {
	pool := testdb.New(t)
	products := product.NewRepository(pool)
	snapshots := snapshot.NewRepository(pool)
	ctx := context.Background()

	p, err := products.Create(ctx, "https://example.com/widget", "varle")
	if err != nil {
		t.Fatalf("Create product: %v", err)
	}
	for i := 0; i < 5; i++ {
		if _, err := snapshots.Create(ctx, p.ID, 10.00, "EUR", true); err != nil {
			t.Fatalf("Create snapshot: %v", err)
		}
	}

	list, err := snapshots.ListByProduct(ctx, p.ID, 3)
	if err != nil {
		t.Fatalf("ListByProduct: %v", err)
	}
	if len(list) != 3 {
		t.Fatalf("ListByProduct returned %d snapshots, want 3 (the limit)", len(list))
	}
}

func TestMinPriceReturnsLowestAcrossSnapshots(t *testing.T) {
	pool := testdb.New(t)
	products := product.NewRepository(pool)
	snapshots := snapshot.NewRepository(pool)
	ctx := context.Background()

	p, err := products.Create(ctx, "https://example.com/widget", "varle")
	if err != nil {
		t.Fatalf("Create product: %v", err)
	}
	for _, price := range []float64{19.99, 14.99, 17.49} {
		if _, err := snapshots.Create(ctx, p.ID, price, "EUR", true); err != nil {
			t.Fatalf("Create snapshot: %v", err)
		}
	}

	min, err := snapshots.MinPrice(ctx, p.ID)
	if err != nil {
		t.Fatalf("MinPrice: %v", err)
	}
	if min == nil || *min != 14.99 {
		t.Errorf("MinPrice = %v, want 14.99", min)
	}
}

func TestMinPriceReturnsNilWhenNoSnapshots(t *testing.T) {
	pool := testdb.New(t)
	products := product.NewRepository(pool)
	snapshots := snapshot.NewRepository(pool)
	ctx := context.Background()

	p, err := products.Create(ctx, "https://example.com/widget", "varle")
	if err != nil {
		t.Fatalf("Create product: %v", err)
	}

	min, err := snapshots.MinPrice(ctx, p.ID)
	if err != nil {
		t.Fatalf("MinPrice: %v", err)
	}
	if min != nil {
		t.Errorf("MinPrice = %v, want nil (no snapshots yet)", *min)
	}
}
