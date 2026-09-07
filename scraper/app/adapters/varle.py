from decimal import Decimal, InvalidOperation
from urllib.parse import urljoin

from bs4 import BeautifulSoup
from bs4.element import Tag

from app.adapters.base import SiteAdapter
from app.models import ProductDeal
from app.pricing import parse_discount_pct, parse_price_text


class VarleAdapter(SiteAdapter):
    """Parses varle.lt product grids (e.g. /ispardavimas/), where each
    `div.GRID_ITEM` card links to a real per-product detail page — unlike
    the grocery sites, this maps cleanly onto "one tracked URL = one
    product". Price comes straight from schema.org `itemprop="price"`
    microdata rather than parsing text, since the site already provides an
    exact decimal in the `content` attribute.
    """

    site_type = "varle"

    def parse(self, html: str, page_url: str) -> list[ProductDeal]:
        soup = BeautifulSoup(html, "lxml")
        deals: list[ProductDeal] = []

        for card in soup.select("div.GRID_ITEM"):
            deal = self._parse_card(card, page_url)
            if deal is not None:
                deals.append(deal)

        return deals

    def _parse_card(self, card: Tag, page_url: str) -> ProductDeal | None:
        title_link = card.select_one("h3.product-title a[href]")
        price_el = card.select_one('span[itemprop="price"]')
        if title_link is None or price_el is None:
            return None

        price = self._read_price(price_el)
        if price is None:
            return None

        discount_el = card.select_one("div.discount-line span")
        discount_pct = None
        if discount_el is not None:
            discount_pct = parse_discount_pct(discount_el.get_text(strip=True))

        image_el = card.select_one("img.product-img")
        image_url = None
        if image_el is not None:
            image_url = image_el.get("src")

        return ProductDeal(
            title=title_link.get_text(strip=True),
            url=urljoin(page_url, title_link["href"]),
            price=price,
            old_price=None,  # Varle shows a discount % badge but no old price on listing cards
            discount_pct=discount_pct,
            image_url=image_url,
        )

    @staticmethod
    def _read_price(price_el: Tag | None) -> Decimal | None:
        if price_el is None:
            return None
        content = price_el.get("content")
        if not content:
            return None
        try:
            return Decimal(content)
        except InvalidOperation:
            return None

    def parse_product(self, html: str, page_url: str) -> ProductDeal | None:
        """Parses a single product's own detail page (not a listing card).
        Unlike the listing grid, the detail page's schema.org Offer block
        carries currency and stock status too, and its discount-line
        (unlike the listing's) isn't commented out, so the old price is
        available here.
        """
        soup = BeautifulSoup(html, "lxml")

        title_el = soup.select_one('h1.title[itemprop="name"]')
        pricing = soup.select_one("div.PRODUCT_PRICING")
        if title_el is None or pricing is None:
            return None

        offer = pricing.select_one('[itemprop="offers"]')
        if offer is None:
            return None

        price = self._read_price(offer.select_one('meta[itemprop="price"]'))
        if price is None:
            return None

        currency = "EUR"
        currency_el = offer.select_one('meta[itemprop="priceCurrency"]')
        if currency_el is not None and currency_el.get("content"):
            currency = currency_el["content"]

        in_stock = None
        availability_el = offer.select_one('link[itemprop="availability"]')
        if availability_el is not None:
            in_stock = "InStock" in (availability_el.get("href") or "")

        old_price = None
        discount_pct = None
        discount_el = pricing.select_one("div.discount-line")
        if discount_el is not None:
            previous_price_el = discount_el.select_one("span.previous-price")
            if previous_price_el is not None:
                old_price = parse_price_text(previous_price_el.get_text(strip=True))
            pct_el = discount_el.select_one("span.discount")
            if pct_el is not None:
                discount_pct = parse_discount_pct(pct_el.get_text(strip=True))

        return ProductDeal(
            title=title_el.get_text(strip=True),
            url=page_url,
            price=price,
            currency=currency,
            in_stock=in_stock,
            old_price=old_price,
            discount_pct=discount_pct,
        )


if __name__ == "__main__":
    import json
    import sys

    url = "https://www.varle.lt/ispardavimas/"
    if len(sys.argv) > 1:
        url = sys.argv[1]
    adapter = VarleAdapter()
    results = adapter.scrape(url)

    print(f"Parsed {len(results)} deals from {url}\n")
    print(json.dumps([d.model_dump(mode="json") for d in results[:5]], indent=2, ensure_ascii=False))
