import json

from bs4 import BeautifulSoup
import requests
from tqdm import tqdm
from utils import walk_files

for file in tqdm(
    list(walk_files("data/backup")), desc="Augmenting GPX URL", unit="record"
):
    with open(file, "r") as fp:
        record = json.load(fp)

    if "gpx_url" in record:
        continue

    ref = record["ref"]
    object_id = record["objectID"]
    mobile_url = (
        f"https://www.gps-routes.co.uk/routes/home.nsf/osmapdisp?openform&route={ref}"
    )
    resp = requests.get(mobile_url)
    soup = BeautifulSoup(resp.text, "lxml")

    routename = soup.find("input", {"name": "routename"})["value"]
    viewname = soup.find("input", {"name": "viewname"})["value"]
    filename = soup.find("input", {"name": "filename"})["value"]

    gpx_url = f"https://www.gps-routes.co.uk/A55CD9/home.nsf/{viewname}/{routename}/$FILE/{filename}"
    record["gpx_url"] = gpx_url

    with open(file, "w") as fp:
        json.dump(record, fp, indent=2)
