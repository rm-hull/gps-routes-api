import asyncio
import functools
import json
import random
from bs4 import BeautifulSoup
import requests

from archive.detail_extractor import DetailExtractor
from utils import get_datahub_client, get_object_id, get_search_client, ROUTES_INDEX
from algoliasearch.http.exceptions import RequestException


algolia_client = get_search_client()
datahub = get_datahub_client()


def store_objects(records: list):
    return algolia_client.save_objects(
        index_name=ROUTES_INDEX,
        objects=records,
    )


def gazetteer_info(record: dict) -> dict:
    if "_geoloc" not in record:
        return {"gazetteer_found": False}

    gazetteer = datahub.nearby(record["_geoloc"]["lat"], record["_geoloc"]["lng"])
    if gazetteer is None:
        return {"gazetteer_found": False}

    result = {}
    if gazetteer["LOCAL_TYPE"] == "Postcode":
        result["postcode"] = gazetteer["NAME1"]

    if "POPULATED_PLACE" in gazetteer:
        result["district"] = gazetteer["POPULATED_PLACE"]

    if "COUNTY_UNITARY" in gazetteer:
        result["county"] = gazetteer["COUNTY_UNITARY"]

    if "REGION" in gazetteer:
        result["region"] = gazetteer["REGION"]

    if "COUNTRY" in gazetteer:
        result["country"] = gazetteer["COUNTRY"]

    result["gazetteer_found"] = bool(result)

    return result


def oversized(record: dict) -> bool:
    return len(json.dumps(record)) > 10000


@functools.cache
def load_all_routes() -> list[str]:
    with open("data/full-list.txt", mode="r") as fp:
        return fp.read().splitlines()


def pick_random_route() -> str:
    return random.choice(load_all_routes())


def select_unprocessed_route() -> tuple[int, str | None]:
    for attempt in range(20):
        route = pick_random_route()
        ref = route.split("/")[-1]
        try:
            resp = algolia_client.get_object(
                index_name=ROUTES_INDEX,
                object_id=get_object_id(ref),
                attributes_to_retrieve=["country", "_geoloc", "gazetteer_found"],
            )
            if "_geoloc" not in resp:
                continue

            if "gazetter_found" in resp:
                continue

            if "country" not in resp:
                return attempt, route

        except RequestException as ex:
            if ex.status_code == 404:
                return attempt, route
            else:
                continue

    return attempt, None


def random_page_crawl():
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
        record = DetailExtractor(markup).process()
        record.update(gazetteer_info(record))

        if oversized(record):
            print(f"WARN: {url} is oversized")
        else:
            records.append(record)

    store_objects(records)


def unprocessed_entries_crawl():
    records = []
    limit = 30

    for index in range(limit):
        num_attempts, url = select_unprocessed_route()
        if not url:
            print(
                f"WARN! couldnt find any unprocessed routes after {num_attempts} attempts"
            )
            continue

        print(f"[{index+1:02d}/{limit}] found after {num_attempts} attempts: {url}")
        markup = requests.get(url).text
        record = DetailExtractor(markup).process()
        record.update(gazetteer_info(record))

        if oversized(record):
            print(f"WARN: {url} is oversized")
        else:
            records.append(record)

    store_objects(records)


async def main():
    # random_page_crawl()
    unprocessed_entries_crawl()


asyncio.run(main())
