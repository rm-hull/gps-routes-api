import json
import os
from utils import get_search_client, ROUTES_INDEX
from tqdm import tqdm

client = get_search_client()

batch_size = 10
cursor = None
progress = None

while True:
    params = dict(length=batch_size)
    if cursor:
        params["cursor"] = cursor

    response = client.browse(index_name=ROUTES_INDEX, browse_params=params)

    if not progress:
        progress = tqdm(total=response.nb_hits, desc="Downloading", unit="record")

    for record in response.hits:
        progress.update()

        base_folder = f"data/backup/{record.object_id[0]}"
        if not os.path.exists(base_folder):
            os.mkdir(base_folder)

        with open(f"{base_folder}/{record.object_id}.json", "w") as fp:
            json.dump(record.to_dict(), fp, indent=2)

    cursor = response.cursor
    if not cursor:
        break
