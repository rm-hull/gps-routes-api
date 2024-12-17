import json
import os
import pyproj
import requests
import dotenv

dotenv.load_dotenv()


class OSDataHub:

    def __init__(self, api_key: str):
        self.api_key = api_key

    def nearby(self, lat: float, lng: float, radius: int = 1000):
        easting, northing = to_bng(lat, lng)
        resp = requests.get(
            f"https://api.os.uk/search/names/v1/nearest",
            params={
                "key": self.api_key,
                "radius": radius,
                "point": f"{easting:0.0f},{northing:0.0f}",
                # "fq": f"LOCAL_TYPE:Postcode"
            },
        ).json()

        if resp["header"]["totalresults"] != 1:
            return None

        # return (resp["results"][0]["GAZETTEER_ENTRY"],)
        return pick_attributes(
            resp["results"][0]["GAZETTEER_ENTRY"],
            keys=[
                "NAME1",
                "LOCAL_TYPE",
                "POPULATED_PLACE",
                "COUNTY_UNITARY",
                "REGION",
                "COUNTRY",
            ],
        )


def pick_attributes(dictionary, keys):
    return {k: dictionary[k] for k in keys if k in dictionary}


def to_bng(lat: float, lng: float) -> tuple[float, float]:
    wgs84 = pyproj.CRS("EPSG:4326")  # WGS84 (standard lat/lon)
    bng = pyproj.CRS("EPSG:27700")  # British National Grid

    transformer = pyproj.Transformer.from_crs(wgs84, bng, always_xy=True)
    easting, northing = transformer.transform(lng, lat)

    return easting, northing


if __name__ == "__main__":
    datahub = OSDataHub(os.getenv("OS_DATAHUB_API_KEY"))

    print(json.dumps(datahub.nearby(56.3779128789, -3.9830157992), indent=2))
    print(json.dumps(datahub.nearby(53.2266182629, -4.0030570596), indent=2))
    print(json.dumps(datahub.nearby(53.9934096, -1.5331361), indent=2))
    print(json.dumps(datahub.nearby(53.9653160,-1.5574571), indent=2))
    print(json.dumps(datahub.nearby(53.9771144, -1.6432486), indent=2))
    print(json.dumps(datahub.nearby(53.9358396, -2.5691932), indent=2))
    print(json.dumps(datahub.nearby(53.4656173, -2.3283789), indent=2))





