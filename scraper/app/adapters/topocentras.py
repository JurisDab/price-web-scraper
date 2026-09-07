import json
from decimal import Decimal
from urllib.parse import urljoin, urlparse

from app.adapters.base import SiteAdapter
from app.models import ProductDeal
from app.pricing import compute_discount_pct

BASE_URL = "https://www.topocentras.lt/"
MEDIA_BASE = "https://www.topocentras.lt/media/catalog/product/"

# Topocentras is a headless Magento 2 storefront (React frontend, GraphQL
# backend) — there's no server-rendered HTML to parse. Its category pages
# call this named/persisted GraphQL query client-side to fetch product data;
# found by watching real network traffic with Playwright (see
# scripts/capture_requests.py), then confirmed to work identically over
# plain curl with no cookies/session needed. So the `url` this adapter
# expects is the getCatalog query URL itself, not the human-facing category
# page — a deliberate exception to the "url = page the user tracks" pattern
# the other adapters follow, unavoidable given there's no HTML page to point
# at. category IDs are found the same way (Playwright + network capture).
def catalog_url(category_id: int, page_size: int = 40, current_page: int = 1) -> str:
    form_vars = json.dumps({"sort": "bestsellers", "filters": {}}, separators=(",", ":"))
    variables = json.dumps(
        {"categoryId": category_id, "pageSize": page_size, "currentPage": current_page},
        separators=(",", ":"),
    )
    return f"{BASE_URL}graphql?formVars={form_vars}&query=getCatalog&vars={variables}"


def url_resolver_url(url_path: str) -> str:
    """Resolves a normal page path (e.g. "/some-product.html") to its
    entity type and numeric ID — the same lookup Topocentras' own frontend
    does before rendering a page, found the same way as catalog_url()."""
    variables = json.dumps({"urlKey": url_path}, separators=(",", ":"))
    return f"{BASE_URL}graphql?query=urlResolver&vars={variables}"


def product_detail_url(product_id: int) -> str:
    variables = json.dumps({"id": str(product_id)}, separators=(",", ":"))
    return f"{BASE_URL}graphql?query=ROOT_GetProduct&vars={variables}"


class TopocentrasAdapter(SiteAdapter):
    site_type = "topocentras"

    def parse(self, raw_json: str, page_url: str) -> list[ProductDeal]:
        data = json.loads(raw_json)
        items = data.get("data", {}).get("products", {}).get("items", [])

        deals: list[ProductDeal] = []
        for item in items:
            deal = self._parse_item(item)
            if deal is not None:
                deals.append(deal)
        return deals

    @staticmethod
    def _parse_item(item: dict) -> ProductDeal | None:
        url_key = item.get("url_key")
        price_range = item.get("price_range", {}).get("minimum_price")
        if not url_key or not price_range:
            return None

        price = Decimal(str(price_range["final_price"]["value"]))
        regular_value = Decimal(str(price_range["regular_price"]["value"]))
        old_price = None
        if regular_value != price:
            old_price = regular_value

        percent_off = price_range.get("discount", {}).get("percent_off") or 0
        if percent_off:
            discount_pct = int(percent_off)
        else:
            discount_pct = compute_discount_pct(price, old_price)

        image_path = (item.get("small_image") or {}).get("path")
        image_url = None
        if image_path:
            image_url = urljoin(MEDIA_BASE, image_path)

        in_stock = None
        if item.get("stock_status"):
            in_stock = item["stock_status"] == "IN_STOCK"

        return ProductDeal(
            title=item["name"],
            url=urljoin(BASE_URL, f"{url_key}.html"),
            price=price,
            in_stock=in_stock,
            old_price=old_price,
            discount_pct=discount_pct,
            image_url=image_url,
        )

    def scrape_product(self, url: str) -> ProductDeal | None:
        """Overrides the base fetch-then-parse shape: this needs two
        requests, not one — resolve the page path to a numeric product ID,
        then fetch that product's data by ID. `url` is the normal
        human-facing product page a user would actually paste in, unlike
        catalog_url()'s listing endpoint.
        """
        url_path = urlparse(url).path
        resolver_response = self.fetch(url_resolver_url(url_path))
        resolved = json.loads(resolver_response).get("data", {}).get("urlResolver")
        if not resolved or resolved.get("type") != "PRODUCT" or not resolved.get("id"):
            return None

        detail_response = self.fetch(product_detail_url(resolved["id"]))
        data = json.loads(detail_response)
        items = data.get("data", {}).get("productDetail", {}).get("items", [])
        if not items:
            return None

        return self._parse_item(items[0])


if __name__ == "__main__":
    import sys

    # Buitinė technika namams (Home appliances), categoryId 510 — found via
    # scripts/capture_requests.py against https://www.topocentras.lt/buitine-technika.html
    category_id = 510
    if len(sys.argv) > 1:
        category_id = int(sys.argv[1])
    url = catalog_url(category_id, page_size=5)

    adapter = TopocentrasAdapter()
    results = adapter.scrape(url)

    print(f"Parsed {len(results)} deals from category {category_id}\n")
    print(json.dumps([d.model_dump(mode="json") for d in results], indent=2, ensure_ascii=False))
