import asyncio
import json
import os
import random
from bs4 import BeautifulSoup
import requests
from dotenv import load_dotenv
from algoliasearch.search.client import SearchClientSync

from detail_extractor import DetailExtractor


load_dotenv()


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


def store_objects(records: list):
    APP_ID = os.getenv("ALGOLIA_APP_ID")
    API_KEY = os.getenv("ALGOLIA_API_KEY")

    client = SearchClientSync(APP_ID, API_KEY)
    return client.save_objects(
        index_name="routes_index",
        objects=records,
    )


async def main():

    start = 1 + random.randint(0, 243) * 29
    url = f"https://www.gps-routes.co.uk/A55CD9/home.nsf/RoutesLinksWalks?OpenView&Start={start}"

    print(f"Fetching index page: {url}")

    markup = requests.get(url).text
    soup = BeautifulSoup(markup, "lxml")

    pages = extract_links(soup)
    records = []

    for index, url in enumerate(pages):
        print(f"[{index+1:02d}/{len(pages)}] processing detail page: {url}")
        markup = requests.get(url).text
        records.append(DetailExtractor(markup).process())

    store_objects(records)

    # base_url = "https://www.gps-routes.co.uk"
    # path = "/A55CD9/home.nsf/RoutesLinksWalks?OpenView&Start=1"
    # prev = None

    # while path != prev:
    #     print(f"checking path {path}, current={len(hrefs)}")

    #     markup = requests.get(f"{base_url}{path}").text
    #     soup = BeautifulSoup(markup, "lxml")

    #     prev = path
    #     path = extract_next_link(soup)
    #     hrefs.extend(extract_links(soup))

    # for href in hrefs:
    #     print(href)


asyncio.run(main())
