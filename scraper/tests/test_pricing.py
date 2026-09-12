from decimal import Decimal

from app.pricing import compute_discount_pct, parse_discount_pct, parse_price_text


def test_parse_price_text_dot_decimal():
    assert parse_price_text("1.49 €") == Decimal("1.49")


def test_parse_price_text_comma_decimal():
    assert parse_price_text("1,49 €") == Decimal("1.49")


def test_parse_price_text_embedded_in_sentence():
    assert parse_price_text("Įprasta kaina: 0,99 €") == Decimal("0.99")


def test_parse_price_text_single_digit_fraction_padded():
    assert parse_price_text("3,9 €") == Decimal("3.90")


def test_parse_price_text_no_number_returns_none():
    assert parse_price_text("Nemokamas pristatymas") is None


def test_parse_price_text_empty_returns_none():
    assert parse_price_text("") is None
    assert parse_price_text(None) is None


def test_parse_discount_pct_with_minus_and_percent():
    assert parse_discount_pct("-25%") == 25


def test_parse_discount_pct_with_space():
    assert parse_discount_pct("35 %") == 35


def test_parse_discount_pct_no_number_returns_none():
    assert parse_discount_pct("Nauja kaina") is None


def test_compute_discount_pct_normal_case():
    assert compute_discount_pct(Decimal("75.00"), Decimal("100.00")) == 25


def test_compute_discount_pct_no_old_price_returns_none():
    assert compute_discount_pct(Decimal("75.00"), None) is None


def test_compute_discount_pct_zero_old_price_returns_none():
    assert compute_discount_pct(Decimal("75.00"), Decimal("0")) is None
