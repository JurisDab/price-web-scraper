from urllib.parse import urljoin

from bs4 import BeautifulSoup
from bs4.element import Tag

from app.adapters.base import SiteAdapter
from app.models import ProductDeal
from app.pricing import compute_discount_pct, parse_price_text


class BaitukasAdapter(SiteAdapter):
    """Parses baitukas.lt's deals listing (OpenCart's
    `index.php?route=product/selection&type=171`), a server-rendered
    `table.table-responsive` — plain HTML, no anti-bot friction. Each
    product row appears twice in the DOM (a hidden list view alongside the
    default grid view, an OpenCart theme quirk), but `tr` only ever wraps
    the table's rows, so selecting on that naturally returns just one copy
    per product.

    No discount-percent badge on the page, so it's computed from
    price/old_price rather than parsed.
    """

    site_type = "baitukas"

    def parse(self, html: str, page_url: str) -> list[ProductDeal]:
        soup = BeautifulSoup(html, "lxml")
        deals: list[ProductDeal] = []

        for row in soup.select("table.table-responsive tr"):
            deal = self._parse_row(row, page_url)
            if deal is not None:
                deals.append(deal)

        return deals

    def _parse_row(self, row: Tag, page_url: str) -> ProductDeal | None:
        link = row.select_one("a[title][href]")
        price_el = row.select_one("span.price-new-1")
        if link is None or price_el is None:
            return None

        price = parse_price_text(price_el.get_text(strip=True))
        if price is None:
            return None

        old_price_el = row.select_one("span.price-old")
        old_price = None
        if old_price_el is not None:
            old_price = parse_price_text(old_price_el.get_text(strip=True))

        image_el = row.select_one("img")
        image_url = None
        if image_el is not None:
            image_url = image_el.get("src")

        return ProductDeal(
            title=link["title"],
            url=urljoin(page_url, link["href"]),
            price=price,
            old_price=old_price,
            discount_pct=compute_discount_pct(price, old_price),
            image_url=image_url,
        )

    def parse_product(self, html: str, page_url: str) -> ProductDeal | None:
        """Parses a single product's own detail page — different markup
        from the listing table (`div.price-new-live` instead of
        `span.price-new-1`), and this page exposes real stock status via
        `#stock_color`'s background color, which the listing doesn't.
        """
        soup = BeautifulSoup(html, "lxml")

        title_el = soup.select_one("h1")
        price_el = soup.select_one("div.price-new-live")
        if title_el is None or price_el is None:
            return None

        price = parse_price_text(price_el.get_text(strip=True))
        if price is None:
            return None

        old_price = None
        old_price_el = soup.select_one("div.price-old-live")
        if old_price_el is not None:
            old_price = parse_price_text(old_price_el.get_text(strip=True))

        in_stock = None
        stock_el = soup.select_one("#stock_color")
        if stock_el is not None:
            in_stock = "40ce66" in (stock_el.get("style") or "").lower()

        return ProductDeal(
            title=title_el.get_text(strip=True),
            url=page_url,
            price=price,
            in_stock=in_stock,
            old_price=old_price,
            discount_pct=compute_discount_pct(price, old_price),
        )


if __name__ == "__main__":
    import json
    import sys

    url = "https://baitukas.lt/index.php?route=product/selection&type=171"
    if len(sys.argv) > 1:
        url = sys.argv[1]
    adapter = BaitukasAdapter()
    results = adapter.scrape(url)

    print(f"Parsed {len(results)} deals from {url}\n")
    print(json.dumps([d.model_dump(mode="json") for d in results[:5]], indent=2, ensure_ascii=False))
