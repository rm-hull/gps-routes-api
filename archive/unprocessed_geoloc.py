import json
from utils import walk_files

targetted = []

for file in walk_files("data/backup"):
    with open(file, "r") as fp:
        record = json.load(fp)

        if "_geoloc" not in record:
            targetted.append(
                f"http://www.gps-routes.co.uk/routes/home.nsf/routeslinkswalks/{record['ref']}"
            )
            

with open("data/missing-geoloc.txt", "w") as fp:
    for line in targetted:
        fp.write(line + "\n")
