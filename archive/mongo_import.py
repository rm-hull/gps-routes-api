import os
import json
from utils import walk_files
from pymongo import MongoClient
from tqdm import tqdm
from bson import ObjectId

MONGO_URI = os.environ["MONGO_URI"]
DATABASE_NAME = "gps-routes"
COLLECTION_NAME = "routes"

client = MongoClient(MONGO_URI)
db = client[DATABASE_NAME]
collection = db[COLLECTION_NAME]


collection.create_index([("objectID", 1)], unique=True)
collection.create_index([("location", "2dsphere")])
collection.create_index(
    [
        ("title", "text"),
        ("description", "text"),
        ("images.title", "text"),
        ("images.caption", "text"),
        ("nearby.description", "text"),
    ]
)

collection.create_index([("district", 1)])
collection.create_index([("county", 1)])
collection.create_index([("region", 1)])
collection.create_index([("country", 1)])

for filename in tqdm(list(walk_files("../data/backup"))):
    with open(filename, "r") as fp:
        data = json.load(fp)
        if "_geoloc" in data:
            data["location"] = dict(
                type="Point",
                coordinates=[data["_geoloc"]["lng"], data["_geoloc"]["lat"]],
            )
            del data["_geoloc"]

        collection.insert_one(data)
