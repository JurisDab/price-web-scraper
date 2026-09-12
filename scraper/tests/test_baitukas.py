"""Integration tests against the real live site — see test_varle.py for why
fixtures aren't used here.
"""

from app.adapters.baitukas import BaitukasAdapter

LISTING_URL = "https://baitukas.lt/index.php?route=product/selection&type=171"


def test_scrape_listing_returns_real_deals():
    deals = BaitukasAdapter().scrape(LISTING_URL)

    assert len(deals) > 5
    first = deals[0]
    assert first.title
    assert first.price > 0
    assert first.url.startswith("https://baitukas.lt/")


def test_scrape_product_matches_a_real_listing_entry():
    deals = BaitukasAdapter().scrape(LISTING_URL)
    listed = deals[0]

    product = BaitukasAdapter().scrape_product(listed.url)

    assert product is not None
    assert product.title
    assert product.price > 0
