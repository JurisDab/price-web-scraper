package alert_test

import (
	"context"
	"testing"

	"pricetracker/internal/alert"
	"pricetracker/internal/product"
	"pricetracker/internal/testdb"
)

func TestCreateAndListByProduct(t *testing.T) {
	pool := testdb.New(t)
	products := product.NewRepository(pool)
	alerts := alert.NewRepository(pool)
	ctx := context.Background()

	p, err := products.Create(ctx, "https://example.com/widget", "varle")
	if err != nil {
		t.Fatalf("Create product: %v", err)
	}

	price := 14.99
	created, err := alerts.Create(ctx, p.ID, alert.TypePriceDrop, "Price dropped 25% to €14.99 (was €19.99)", &price)
	if err != nil {
		t.Fatalf("Create alert: %v", err)
	}
	if created.Type != alert.TypePriceDrop {
		t.Errorf("Type = %q, want %q", created.Type, alert.TypePriceDrop)
	}
	if created.Price == nil || *created.Price != price {
		t.Errorf("Price = %v, want %v", created.Price, price)
	}
	if created.ReadAt != nil {
		t.Errorf("ReadAt = %v, want nil for a newly created alert", created.ReadAt)
	}

	list, err := alerts.ListByProduct(ctx, p.ID, 100)
	if err != nil {
		t.Fatalf("ListByProduct: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("ListByProduct returned %d alerts, want 1", len(list))
	}
	if list[0].ID != created.ID {
		t.Errorf("ListByProduct returned alert %d, want %d", list[0].ID, created.ID)
	}
}

func TestListReturnsAlertsAcrossProducts(t *testing.T) {
	pool := testdb.New(t)
	products := product.NewRepository(pool)
	alerts := alert.NewRepository(pool)
	ctx := context.Background()

	p1, err := products.Create(ctx, "https://example.com/a", "varle")
	if err != nil {
		t.Fatalf("Create product a: %v", err)
	}
	p2, err := products.Create(ctx, "https://example.com/b", "skytech")
	if err != nil {
		t.Fatalf("Create product b: %v", err)
	}

	if _, err := alerts.Create(ctx, p1.ID, alert.TypeBackInStock, "Back in stock", nil); err != nil {
		t.Fatalf("Create alert for a: %v", err)
	}
	if _, err := alerts.Create(ctx, p2.ID, alert.TypeAllTimeLow, "New all-time low: €9.99", nil); err != nil {
		t.Fatalf("Create alert for b: %v", err)
	}

	list, err := alerts.List(ctx, 100)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("List returned %d alerts, want 2 (across both products)", len(list))
	}
}

func TestListByProductOnlyReturnsThatProductsAlerts(t *testing.T) {
	pool := testdb.New(t)
	products := product.NewRepository(pool)
	alerts := alert.NewRepository(pool)
	ctx := context.Background()

	p1, err := products.Create(ctx, "https://example.com/a", "varle")
	if err != nil {
		t.Fatalf("Create product a: %v", err)
	}
	p2, err := products.Create(ctx, "https://example.com/b", "skytech")
	if err != nil {
		t.Fatalf("Create product b: %v", err)
	}

	if _, err := alerts.Create(ctx, p1.ID, alert.TypeBackInStock, "Back in stock", nil); err != nil {
		t.Fatalf("Create alert for a: %v", err)
	}
	if _, err := alerts.Create(ctx, p2.ID, alert.TypeAllTimeLow, "New all-time low: €9.99", nil); err != nil {
		t.Fatalf("Create alert for b: %v", err)
	}

	list, err := alerts.ListByProduct(ctx, p1.ID, 100)
	if err != nil {
		t.Fatalf("ListByProduct: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("ListByProduct(p1) returned %d alerts, want 1 (only p1's)", len(list))
	}
	if list[0].ProductID != p1.ID {
		t.Errorf("ListByProduct(p1) returned an alert for product %q", list[0].ProductID)
	}
}
