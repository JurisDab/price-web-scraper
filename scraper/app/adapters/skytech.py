from decimal import Decimal
from urllib.parse import urljoin

from bs4 import BeautifulSoup
from bs4.element import Tag

from app.adapters.base import SiteAdapter
from app.models import ProductDeal
from app.pricing import parse_price_text


class SkytechAdapter(SiteAdapter):
    """Parses skytech.lt's discounts page (/akcijos.html), a server-rendered
    `table.productListing` — despite first appearances (its homepage uses a
    different template with no matching markup, which is what led to
    initially misjudging this page as JS-rendered; a headless-browser check
    showed the grid was present on first navigation, meaning it was server
    HTML all along, just under different class names).

    Rows come in two flavors: `tr.productListing.group.*` are category
    separator rows (colspan heading, no product), and plain
    `tr.productListing.even/odd` are products — distinguished here by
    requiring a `td.model` child rather than by class name.
    """

    site_type = "skytech"

    def parse(self, html: str, page_url: str) -> list[ProductDeal]:
        soup = BeautifulSoup(html, "lxml")
        deals: list[ProductDeal] = []

        for row in soup.select("tr.productListing"):
            deal = self._parse_row(row, page_url)
            if deal is not None:
                deals.append(deal)

        return deals

    def _parse_row(self, row: Tag, page_url: str) -> ProductDeal | None:
        if row.select_one("td.model") is None:
            return None  # category separator row

        title_link = row.select_one("td.name a[href]")
        price_el = row.select_one("td.akcija")
        if title_link is None or price_el is None:
            return None

        price = parse_price_text(price_el.get_text(strip=True))
        if price is None:
            return None

        discount_badge = row.select_one("div.icon-list-top3")
        discount_pct = None
        if discount_badge is not None:
            rel = discount_badge.get("rel", "").rstrip("%")
            if rel.isdigit():
                discount_pct = int(rel)

        return ProductDeal(
            title=title_link.get_text(strip=True),
            url=urljoin(page_url, title_link["href"]),
            price=price,
            old_price=None,  # not shown on the listing, only current price + discount badge
            discount_pct=discount_pct,
            image_url=None,  # not present on the listing rows
        )


if __name__ == "__main__":
    import json
    import sys

    url = "https://www.skytech.lt/akcijos.html"
    if len(sys.argv) > 1:
        url = sys.argv[1]
    adapter = SkytechAdapter()
    results = adapter.scrape(url)

    print(f"Parsed {len(results)} deals from {url}\n")
    print(json.dumps([d.model_dump(mode="json") for d in results[:5]], indent=2, ensure_ascii=False))
