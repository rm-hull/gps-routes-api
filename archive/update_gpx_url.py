import json
from tqdm import tqdm

from utils import walk_files

for file in tqdm(list(walk_files("../data/backup")), desc="Updating GPX URL", unit="record"):
    with open(file, "r") as fp:
        record = json.load(fp)

    object_id = record["objectID"]

    if "gpx_url" in record and not record["gpx_url"].startswith("https://raw.githubusercontent.com/rm-hull/gps-routes-api"):
        record["gpx_url"] = (
            f"https://raw.githubusercontent.com/rm-hull/gps-routes-api/refs/heads/main/data/routes/{object_id[0]}/{object_id}.gpx"
        )

        with open(file, "w") as fp:
            json.dump(record, fp, indent=2)
