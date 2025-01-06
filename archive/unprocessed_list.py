import json
from utils import walk_files

search_refs = set()
targetted = []

for file in walk_files("../data/backup"):
    with open(file, "r") as fp:
        record = json.load(fp)
        search_refs.add(record["ref"])

with open("../data/full-list.txt", "r") as fp:
    for line in fp.read().splitlines():
        ref = line.split("/")[-1]
        if ref not in search_refs:
            targetted.append(line)

with open("../data/unprocessed-list.txt", "w") as fp:
    for line in targetted:
        fp.write(line + "\n")
