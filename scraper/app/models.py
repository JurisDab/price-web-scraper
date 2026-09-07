from decimal import Decimal
from typing import Optional

from pydantic import BaseModel


class ProductDeal(BaseModel):
    """A single discounted product as found on a site's deals page."""

    title: str
    url: str
    price: Decimal
    currency: str = "EUR"
    in_stock: Optional[bool] = None
    old_price: Optional[Decimal] = None
    discount_pct: Optional[int] = None
    unit_info: Optional[str] = None
    valid_to: Optional[str] = None
    image_url: Optional[str] = None
