import asyncio
from pydoc import text
from urllib.parse import parse_qs, urlparse
from bs4 import BeautifulSoup
import requests


def check(text):
    print(f"check: {text}")
    return text and "Next" in text


def extract_next_link(soup: BeautifulSoup) -> str | None:
    next_text = soup.find(string="Next")
    if next_text:
        parent_a_tag = next_text.find_parent("a", href=True)
        if parent_a_tag:
            return parent_a_tag.get("href")


def extract_links(soup: BeautifulSoup) -> list[str]:
    results = []
    target_div = soup.find(
        "div",
        align="center",
        string=lambda text: text and "RoutesLinksWalks" in text,
    )
    if target_div:
        target_table = target_div.find_next("table")
        if target_table:
            td_elements = target_table.find_all("td")
            for td in td_elements:
                links = td.find_all("a")
                for link in links:
                    href = link.get("href")
                    if href:
                        results.append(href)

    return results


async def main():

    # Fetch sample page
    # markup = requests.get(url).text
    # soup = BeautifulSoup(markup, "lxml")

    test_file = "example-data/paginated-results.html"

    hrefs = []

    base_url = "https://www.gps-routes.co.uk"
    path = "/A55CD9/home.nsf/RoutesLinksWalks?OpenView&Start=1"
    prev = None

    while path != prev:
        print(f"checking path {path}, current={len(hrefs)}")

        markup = requests.get(f"{base_url}{path}").text
        soup = BeautifulSoup(markup, "lxml")

        prev = path
        path = extract_next_link(soup)
        hrefs.extend(extract_links(soup))

    for href in hrefs:
        print(href)


asyncio.run(main())
