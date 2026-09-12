package scrapeclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// package-internal test (not scrapeclient_test): baseURL is unexported, and
// pointing the client at an httptest server needs direct struct
// construction rather than New(), which only reads it from an env var.
func newTestClient(baseURL string) *Client {
	return &Client{baseURL: baseURL, httpClient: http.DefaultClient}
}

func TestScrapeSendsCorrectRequestAndParsesResponse(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody scrapeRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decoding request body: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Result{
			Title:    "Widget 3000",
			Price:    19.99,
			Currency: "EUR",
			InStock:  true,
		})
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	result, err := client.Scrape(context.Background(), "https://example.com/widget", "varle")
	if err != nil {
		t.Fatalf("Scrape: %v", err)
	}

	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want POST", gotMethod)
	}
	if gotPath != "/scrape" {
		t.Errorf("path = %q, want /scrape", gotPath)
	}
	if gotBody.URL != "https://example.com/widget" || gotBody.SiteType != "varle" {
		t.Errorf("request body = %+v, want url/site_type passed to Scrape", gotBody)
	}

	if result.Title != "Widget 3000" || result.Price != 19.99 || result.Currency != "EUR" || !result.InStock {
		t.Errorf("result = %+v, want the response the server sent", result)
	}
}

func TestScrapeReturnsErrorOnNon200(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"detail":"unknown site_type: bogus"}`))
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	_, err := client.Scrape(context.Background(), "https://example.com/widget", "bogus")
	if err == nil {
		t.Fatal("Scrape returned nil error for a non-200 response, want an error")
	}
}

func TestScrapeReturnsErrorWhenServerUnreachable(t *testing.T) {
	client := newTestClient("http://127.0.0.1:1") // nothing listens here
	_, err := client.Scrape(context.Background(), "https://example.com/widget", "varle")
	if err == nil {
		t.Fatal("Scrape returned nil error when the server is unreachable, want an error")
	}
}
