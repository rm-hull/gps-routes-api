# gps-routes-api [![Go Report Card](https://goreportcard.com/badge/github.com/rm-hull/gps-routes-api)](https://goreportcard.com/report/github.com/rm-hull/gps-routes-api) [![Coverage Status](https://coveralls.io/repos/github/rm-hull/gps-routes-api/badge.svg)](https://coveralls.io/github/rm-hull/gps-routes-api)

## Local setup

```console
$ python3 -m venv .venv
$ source .venv/bin/activate
$ pip install --upgrade pip
$ pip install -r requirements.txt
$ cp .env.example .env
$ vi .env  # <== edit the API keys
```

## Data Cleansing

To remove escaped new-lines & non-breaking spaces:

```console
find data/backup -type f -exec perl -i -pe 's/\ / /g' {} +
find data/backup -type f -exec perl -i -pe 's/\\n/ /g' {} +
find data/backup -type f -exec perl -i -pe 's/ , /,  /g' {} +
find data/backup -type f -exec perl -i -pe 's/ \. /. /g' {} +
```
