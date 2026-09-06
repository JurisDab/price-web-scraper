CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE products (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    url              TEXT NOT NULL UNIQUE,
    site_type        TEXT NOT NULL,
    title            TEXT,
    status           TEXT NOT NULL DEFAULT 'pending_first_scrape'
                     CHECK (status IN ('pending_first_scrape', 'active', 'error')),
    last_error       TEXT,
    current_price    NUMERIC(10, 2),
    currency         TEXT NOT NULL DEFAULT 'EUR',
    in_stock         BOOLEAN,
    last_scraped_at  TIMESTAMPTZ,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE price_snapshots (
    id          BIGSERIAL PRIMARY KEY,
    product_id  UUID NOT NULL REFERENCES products (id) ON DELETE CASCADE,
    price       NUMERIC(10, 2) NOT NULL,
    currency    TEXT NOT NULL,
    in_stock    BOOLEAN NOT NULL,
    scraped_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_price_snapshots_product_scraped_at
    ON price_snapshots (product_id, scraped_at DESC);

CREATE TABLE alerts (
    id          BIGSERIAL PRIMARY KEY,
    product_id  UUID NOT NULL REFERENCES products (id) ON DELETE CASCADE,
    type        TEXT NOT NULL
                CHECK (type IN ('price_drop', 'all_time_low', 'back_in_stock', 'deal_score')),
    message     TEXT NOT NULL,
    price       NUMERIC(10, 2),
    deal_score  SMALLINT CHECK (deal_score BETWEEN 0 AND 100),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    read_at     TIMESTAMPTZ
);

CREATE INDEX idx_alerts_product_created_at
    ON alerts (product_id, created_at DESC);
