import re
from decimal import Decimal, InvalidOperation
from typing import Optional

# Matches "1.49", "1,49", "1,99 €", "Įprasta kaina: 0,99 €" — a run of
# digits, a comma or dot, then 1-2 more digits, anywhere in the string.
_PRICE_RE = re.compile(r"(\d+)[.,](\d{1,2})\b")

# Matches "-25%", "35 %", "-35"
_PERCENT_RE = re.compile(r"(\d+)\s*%?")


def parse_price_text(text: Optional[str]) -> Optional[Decimal]:
    """Extract a price from free-form text, tolerant of comma or dot as the
    decimal separator (sites mix both, sometimes on the same page)."""
    if not text:
        return None
    match = _PRICE_RE.search(text)
    if not match:
        return None
    integer, fraction = match.groups()
    try:
        return Decimal(f"{integer}.{fraction.ljust(2, '0')}")
    except InvalidOperation:
        return None


def parse_discount_pct(text: Optional[str]) -> Optional[int]:
    if not text:
        return None
    match = _PERCENT_RE.search(text)
    if not match:
        return None
    return int(match.group(1))


def compute_discount_pct(price: Optional[Decimal], old_price: Optional[Decimal]) -> Optional[int]:
    """Fallback for sites that show old/new price but no explicit badge."""
    if price is None or not old_price or old_price <= 0:
        return None
    return round((1 - price / old_price) * 100)
