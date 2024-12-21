import json
from operator import index
from tqdm import tqdm
from utils import get_search_client, ROUTES_INDEX, walk_files


client = get_search_client()

files = list(walk_files("data/backup"))
batch = []

for file in tqdm(files, desc="Uploading", unit="record"):
    with open(file, "r") as fp:
        record = json.load(fp)
        batch.append(record)

        if len(batch) == 100:
            client.save_objects(index_name=ROUTES_INDEX, objects=batch)
            batch = []

if batch:
    client.save_objects(index_name=ROUTES_INDEX, objects=batch)
