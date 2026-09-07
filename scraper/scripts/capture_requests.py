"""One-off investigation tool: load a page in a real browser and log every
XHR/fetch request it makes, to find the JSON API backing a JS-rendered
product grid. Not part of the adapter pipeline — throwaway by design.

Usage: python scripts/capture_requests.py <url>
"""
import sys

from playwright.sync_api import sync_playwright


def capture(url: str) -> None:
    with sync_playwright() as p:
        browser = p.chromium.launch()
        page = browser.new_page()

        def on_response(response):
            request = response.request
            if request.resource_type in ("xhr", "fetch"):
                print(f"{response.status}  {request.method}  {request.url}")

        page.on("response", on_response)
        page.goto(url, wait_until="networkidle", timeout=30000)

        for text in ("Sutinku", "Accept", "Priimti", "Leisti visus"):
            try:
                page.get_by_text(text, exact=False).first.click(timeout=2000)
                print(f"[clicked cookie button: {text!r}]", file=sys.stderr)
                break
            except Exception:
                continue

        page.wait_for_timeout(3000)
        page.screenshot(path="scripts/_debug_screenshot.png", full_page=True)
        with open("scripts/_debug_rendered.html", "w", encoding="utf-8") as f:
            f.write(page.content())
        browser.close()


if __name__ == "__main__":
    capture(sys.argv[1])
