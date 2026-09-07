package scrapeclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

type Result struct {
	Title    string  `json:"title"`
	Price    float64 `json:"price"`
	Currency string  `json:"currency"`
	InStock  bool    `json:"in_stock"`
}

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func New() *Client {
	base := os.Getenv("SCRAPER_SERVICE_URL")
	if base == "" {
		base = "http://localhost:8000"
	}
	return &Client{
		baseURL:    base,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

type scrapeRequest struct {
	URL      string `json:"url"`
	SiteType string `json:"site_type"`
}

func (c *Client) Scrape(ctx context.Context, url, siteType string) (Result, error) {
	body, err := json.Marshal(scrapeRequest{URL: url, SiteType: siteType})
	if err != nil {
		return Result{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/scrape", bytes.NewReader(body))
	if err != nil {
		return Result{}, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return Result{}, fmt.Errorf("scraper service request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return Result{}, fmt.Errorf("scraper service returned %d: %s", resp.StatusCode, respBody)
	}

	var result Result
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return Result{}, fmt.Errorf("decoding scraper response: %w", err)
	}
	return result, nil
}
