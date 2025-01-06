import json
import time

import requests
from tqdm import tqdm
from utils import walk_files

for file in tqdm(list(walk_files("../data/backup")), desc="Nominatim", unit="record"):
    with open(file, "r") as fp:
        record = json.load(fp)

    if "nominatim_found" not in record:
        ref = record["ref"]
        object_id = record["objectID"]
        record["nominatim_found"] = False

        if "_geoloc" in record:
            geoloc = record["_geoloc"]

            lat = geoloc["lat"]
            lng = geoloc["lng"]

            url = f"https://nominatim.openstreetmap.org/reverse.php"
            resp = requests.get(
                url,
                params={"lat": lat, "lon": lng, "format": "jsonv2"},
                headers={"User-Agent": "gps-routes-api"},
            )

            if resp.status_code == 200:

                payload = resp.json()
                addr = payload["address"]

                record["nominatim_found"] = True
                record["display_address"] = payload["display_name"]
                record["postcode"] = addr.get("postcode")
                record["district"] = (
                    addr.get("city")
                    or addr.get("town")
                    or addr.get("village")
                )

                record["county"] = addr.get("county")
                record["region"] = addr.get("state_district")
                record["state"] = addr.get("state")
                record["country"] = addr.get("country")

            with open(file, "w") as fp:
                json.dump(record, fp, indent=2)

            time.sleep(1)
