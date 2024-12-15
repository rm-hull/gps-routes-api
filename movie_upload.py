import asyncio
import os
import requests
from algoliasearch.search.client import SearchClientSync
from dotenv import load_dotenv

load_dotenv()


async def main():
    APP_ID = os.getenv("ALGOLIA_APP_ID")
    API_KEY = os.getenv("ALGOLIA_API_KEY")

    # Fetch sample dataset
    url = "https://dashboard.algolia.com/sample_datasets/movie.json"
    movies = requests.get(url).json()

    # Connect and authenticate with your Algolia app
    _client = SearchClientSync(APP_ID, API_KEY)

    # Save records in Algolia index
    _client.save_objects(
        index_name="movies_index",
        objects=movies,
    )


asyncio.run(main())
