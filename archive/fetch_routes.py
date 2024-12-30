import json
import os
from tqdm import tqdm
import requests

from utils import walk_files


for file in tqdm(list(walk_files("data/backup")), desc="Extracting GPX", unit="record"):
    with open(file, "r") as fp:
        record = json.load(fp)

    object_id = record["objectID"]
    base_folder = f"data/routes/{object_id[0]}"
    if not os.path.exists(base_folder):
        os.mkdir(base_folder)

    filename = f"{base_folder}/{object_id}.gpx"
    if os.path.exists(filename):
        continue

    gpx_url = record.get("gpx_url")
    if gpx_url:
        resp = requests.get(gpx_url)
        if resp.status_code == 200:
            with open(filename, "w") as fp:
                fp.write(resp.text)
