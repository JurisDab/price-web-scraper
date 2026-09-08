# Price Tracker

Tracks prices and discounts across Lithuanian retailers: a Go API schedules
and orchestrates, a Python service does the actual scraping, Postgres holds
the data.

## Architecture

```
[User adds a product URL] → [Go API] → [Postgres: products]
                                  ↓
                    [Go dispatches a scrape job over HTTP]
                                  ↓
              [Python: fetch the site, parse title/price/stock]
                                  ↓
   [Go: store price_snapshots row, update product, check alert rules]

        (meanwhile, a Go scheduler re-dispatches scrape jobs
         for every tracked product on a timer, capped per domain)
```

## Setup

**Database** — migrations are plain SQL, named for
[golang-migrate](https://github.com/golang-migrate/migrate):

```sh
migrate -database "postgres://user:pass@localhost:5434/pricetracker?sslmode=disable" \
        -path db/migrations up
```

Port **5434**, not the default 5432 — see `docker-compose.yml`. Remapped
because a native Postgres install on the dev machine was already bound to
5432, and the two silently conflicted rather than erroring, which took a
while to track down.

**Python scraper**:

```sh
cd scraper
python -m venv .venv
.venv/Scripts/pip install -e .        # Windows
.venv/bin/pip install -e .            # macOS/Linux
.venv/Scripts/python -m uvicorn app.main:app --port 8000
```

**Go API**:

```sh
cd api
DATABASE_URL="postgres://user:pass@localhost:5434/pricetracker?sslmode=disable" go run ./cmd/server
```

Defaults to that same connection string (matching `docker-compose.yml`) if
`DATABASE_URL` isn't set. Listens on `:8080`.

With Postgres, the Python service, and the Go API all running, the full
loop works end-to-end: add a product's own page URL (not a listing/deals
page — see "Adapters" below for why the distinction matters), trigger a
scrape, and the product's `current_price`/`in_stock`/`status` update from a
real live fetch, with the result also stored in `price_snapshots`.
Verified against all four site_types (`varle`, `skytech`, `baitukas`,
`topocentras`). If the Python service isn't running, the failure path is
exercised correctly too: the connection error gets stored as the product's
`last_error` with `status` set to `error`.

If Python output looks garbled in a Windows terminal (mojibake on
Lithuanian characters), that's the console codepage, not the data — the
scraped text is correct UTF-8 (verified by writing to a file and reading it
back). Set `PYTHONIOENCODING=utf-8` before running if you want clean
terminal output.

## Schema

- **products** — one row per tracked URL. `status` tracks scrape lifecycle
  (`pending_first_scrape` → `active`, or `error` with `last_error` set).
  `current_price` / `in_stock` / `last_scraped_at` are denormalized from the
  latest snapshot so the dashboard's product list doesn't need a join.
- **price_snapshots** — append-only history, one row per scrape. Drives the
  price chart and all alert logic (comparisons against previous/lowest price).
- **alerts** — generated when a snapshot triggers a rule (`price_drop`,
  `all_time_low`, `back_in_stock`) or an LLM deal-score verdict (`deal_score`,
  with `deal_score` 0-100). `read_at` lets the dashboard mark alerts as seen.

## API endpoints

- `POST /products` — `{url, site_type}` → creates a product (`status: pending_first_scrape`)
- `GET /products` — list tracked products
- `GET /products/{id}` — one product + its price history (snapshots)
- `POST /products/{id}/scrape` — dispatch a scrape job, store the result as a
  new snapshot, update the product's denormalized price/stock fields
- `POST /scrape` (Python service, port 8000) — `{url, site_type}` →
  `{title, price, currency, in_stock}`. What the Go API's `scrapeclient` calls.

## Scheduler

`api/internal/scheduler` sweeps all tracked products — once immediately on
startup, then every `DefaultInterval` (6h). Within a sweep, different
domains scrape concurrently (Varle and Skytech products don't wait on each
other), but each domain is capped at `DefaultPerDomainConcurrency` (2)
requests in flight at once, so no single site gets hit with a burst. Both
constants are hardcoded in `internal/scheduler/scheduler.go` for now — real
config (env vars) can come later if needed, no point adding that surface
before anything actually needs to tune it.

The manual `POST /products/{id}/scrape` endpoint and the scheduler share
the exact same scrape-one-product logic (`internal/scrapeservice`) rather
than each having their own copy — a manual trigger and a scheduled one
behave identically, including the same error handling.

Verified live: seeded two products (different domains) directly in
Postgres, started the server, and confirmed both went from untouched rows
to `status: active` with real scraped prices within about a second of
startup — no manual `/scrape` call involved.

## Adapters

One adapter per site (`scraper/app/adapters/<site>.py`). Each does two
things: `parse()`/`scrape()` for a deals-listing page (many `ProductDeal`s
at once), and `parse_product()`/`scrape_product()` for one product's own
page (a single `ProductDeal`, matching "one tracked URL = one product" the
way the DB/API are designed) — the `POST /scrape` HTTP endpoint uses the
latter. The two aren't always the same page for a given site. Sites vary
too much in markup to share a parser, so there's no attempt at a generic
one.

## What's not yet implemented

- **Alert logic** — threshold-based rules (`price_drop`, `all_time_low`,
  `back_in_stock`) triggered from a new scrape result, persisted to the
  `alerts` table. Planned as the next piece.
- **LLM deal-score** — a follow-up to alert logic: instead of (or
  alongside) fixed thresholds, feed price history to a local LLM (Ollama)
  for a "genuine deal vs. noise" verdict. Needs Ollama running + prompt
  design; deliberately scoped as a separate step after plain thresholds
  are working.
- **React dashboard** — product list, add-product form, price history
  chart, alerts view. Not started.
- **Automated test suite** — currently the only way to check an adapter
  or the API is running it manually against the live site/DB. No pytest
  suite (with saved HTML/JSON fixtures) or Go tests exist yet.
- **docker-compose for the full stack** — currently only runs Postgres;
  doesn't yet build/run the Go API, the Python service, or (once it
  exists) the dashboard together.
