from abc import ABC, abstractmethod

import httpx

from app.models import ProductDeal

DEFAULT_HEADERS = {
    "User-Agent": (
        "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 "
        "(KHTML, like Gecko) Chrome/120.0 Safari/537.36"
    )
}


class SiteAdapter(ABC):
    """One adapter per site: knows how to fetch its deals page and parse
    it into a flat list of ProductDeal. Sites vary too much for a shared
    parser, so each adapter owns its own selectors."""

    site_type: str

    def fetch(self, url: str) -> str:
        with httpx.Client(headers=DEFAULT_HEADERS, timeout=15, follow_redirects=True) as client:
            response = client.get(url)
            response.raise_for_status()
            return response.text

    @abstractmethod
    def parse(self, html: str, page_url: str) -> list[ProductDeal]:
        ...

    def scrape(self, url: str) -> list[ProductDeal]:
        html = self.fetch(url)
        return self.parse(html, url)
