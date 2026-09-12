"""Integration tests against the real live GraphQL API — see test_varle.py
for why frozen fixtures aren't used elsewhere in this suite; the reasoning
applies just as much to a scraped API response as to scraped HTML.
"""

from app.adapters.topocentras import TopocentrasAdapter, catalog_url

HOME_APPLIANCES_CATEGORY_ID = 510


def test_scrape_listing_returns_real_deals():
    deals = TopocentrasAdapter().scrape(catalog_url(HOME_APPLIANCES_CATEGORY_ID, page_size=10))

    assert len(deals) > 5
    first = deals[0]
    assert first.title
    assert first.price > 0
    assert first.url.startswith("https://www.topocentras.lt/")


def test_scrape_product_matches_a_real_listing_entry():
    deals = TopocentrasAdapter().scrape(catalog_url(HOME_APPLIANCES_CATEGORY_ID, page_size=10))
    listed = deals[0]

    product = TopocentrasAdapter().scrape_product(listed.url)

    assert product is not None
    assert product.title
    assert product.price > 0
