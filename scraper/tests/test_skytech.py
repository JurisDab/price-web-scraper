"""Integration tests against the real live site — see test_varle.py for why
fixtures aren't used here.
"""

from app.adapters.skytech import SkytechAdapter


def test_scrape_listing_returns_real_deals():
    deals = SkytechAdapter().scrape("https://www.skytech.lt/akcijos.html")

    assert len(deals) > 5
    first = deals[0]
    assert first.title
    assert first.price > 0
    assert first.url.startswith("https://www.skytech.lt/")


def test_scrape_product_matches_a_real_listing_entry():
    deals = SkytechAdapter().scrape("https://www.skytech.lt/akcijos.html")
    listed = deals[0]

    product = SkytechAdapter().scrape_product(listed.url)

    assert product is not None
    assert product.title
    assert product.price > 0
