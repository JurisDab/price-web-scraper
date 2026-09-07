from fastapi import FastAPI, HTTPException
from pydantic import BaseModel

from app.adapters.baitukas import BaitukasAdapter
from app.adapters.skytech import SkytechAdapter
from app.adapters.topocentras import TopocentrasAdapter
from app.adapters.varle import VarleAdapter

app = FastAPI()

ADAPTERS = {
    "varle": VarleAdapter(),
    "skytech": SkytechAdapter(),
    "baitukas": BaitukasAdapter(),
    "topocentras": TopocentrasAdapter(),
}


class ScrapeRequest(BaseModel):
    url: str
    site_type: str


class ScrapeResponse(BaseModel):
    title: str
    price: float
    currency: str
    in_stock: bool


@app.post("/scrape", response_model=ScrapeResponse)
def scrape(req: ScrapeRequest) -> ScrapeResponse:
    adapter = ADAPTERS.get(req.site_type)
    if adapter is None:
        raise HTTPException(status_code=400, detail=f"unknown site_type: {req.site_type}")

    try:
        deal = adapter.scrape_product(req.url)
    except Exception as exc:
        raise HTTPException(status_code=502, detail=f"scrape failed: {exc}") from exc

    if deal is None:
        raise HTTPException(status_code=502, detail="could not parse product from page")

    # Go's Result.InStock isn't nullable; sites that don't expose stock
    # status leave ProductDeal.in_stock as None, so default to available
    # rather than surface a false "out of stock" warning.
    in_stock = deal.in_stock
    if in_stock is None:
        in_stock = True

    return ScrapeResponse(
        title=deal.title,
        price=float(deal.price),
        currency=deal.currency,
        in_stock=in_stock,
    )
