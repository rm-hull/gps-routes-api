import functools
import hashlib
import os
from dotenv import load_dotenv
from archive.osdatahub_names import OSDataHub
from algoliasearch.search.client import SearchClientSync

load_dotenv()


def get_object_id(ref: str) -> str:
    m = hashlib.md5()
    m.update(ref.encode("utf-8"))
    return m.hexdigest()


ROUTES_INDEX = "routes_index"


@functools.cache
def get_search_client():
    ALGOLIA_APP_ID = os.environ["ALGOLIA_APP_ID"]
    ALGOLIA_API_KEY = os.environ["ALGOLIA_API_KEY"]
    return SearchClientSync(ALGOLIA_APP_ID, ALGOLIA_API_KEY)


@functools.cache
def get_datahub_client():
    OS_DATAHUB_API_KEY = os.environ["OS_DATAHUB_API_KEY"]
    return OSDataHub(api_key=OS_DATAHUB_API_KEY)


def walk_files(directory):
    for root, _, files in os.walk(directory):
        for file in files:
            yield os.path.join(root, file)
