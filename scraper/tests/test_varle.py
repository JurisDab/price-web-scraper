"""Integration tests: hit the real live site, not saved fixtures. A frozen
HTML snapshot would keep "passing" after Varle changes its markup, which
defeats the point of testing a scraper — so these are slower and can fail
for reasons outside the code (site down, redesign), but that's the more
honest signal for this kind of code.
"""

from app.adapters.varle import VarleAdapter


def test_scrape_listing_returns_real_deals():
    deals = VarleAdapter().scrape("https://www.varle.lt/ispardavimas/")

    assert len(deals) > 5
    first = deals[0]
    assert first.title
    assert first.price > 0
    assert first.url.startswith("https://www.varle.lt/")


def test_scrape_product_matches_a_real_listing_entry():
    deals = VarleAdapter().scrape("https://www.varle.lt/ispardavimas/")
    listed = deals[0]

    product = VarleAdapter().scrape_product(listed.url)

    assert product is not None
    assert product.title
    assert product.price > 0
    assert product.currency == "EUR"
    assert product.in_stock is True
