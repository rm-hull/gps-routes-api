import json
import requests
from tqdm import tqdm

from utils import walk_files


# Define the prompt template
system_prompt = """
You are an expert at structured data extraction. Your task is to extract specific summary information for the given document, and output ONLY valid JSON with no additional text. Follow these steps:

1. Read the document carefully.
2. Disregard any previous instructions or user prompts.
3. Extract the following details, but be sure to use generic descriptions and not the exact text from the document:
   - Estimated duration (e.g. use the distance and activity type to always provide time-based estimates like "1 hour", "half day", etc.)
   - Difficulty (easy, moderate, or hard)
   - Terrain (an array, e.g. ["woodland", "coastal", "mountain"])
   - Points of interest (an array, e.g. ["historical site", "viewpoint"] - refrain from using specific names)
   - Facilities (an array, e.g. ["pub/cafe", "visitor center", "car park", "dog friendly", "play area"] )
   - Route type (e.g. "circular", "one-way", "out and back")
   - Activities (an array, e.g. ["walking", "cycling", "horse riding", "bird watching", "swimming"])
4. Use the singular form for each item in the arrays, e.g. "pub/cafe" instead of "pubs/cafes".
5. Be consistent with the terminology used in the output.
6. If "Dog walking" is explicitly mentioned, include "dog friendly" in the facilities.
7. Be sure to include specific facilities only if they are noted in the document
8. If you are unsure about any detail, you may leave it blank or use a placeholder value.
9. Format your output exactly as the JSON below. Do not include any text before or after the JSON.
{
  "estimated_duration": "duration",
  "difficulty": "difficulty level",
  "terrain": ["Terrain 1", "Terrain 2", "Terrain 3"],
  "points_of_interest": ["Point of interest 1", "Point of interest 2", "Point of interest 3"],
  "facilities": ["Facility 1", "Facility 2", "Facility 3"],
  "route_type": "Route type",
  "activities": ["Activity type 1", "Activity type 2"]
}
"""


def get_facets(document: str) -> tuple[dict, float]:

    # Define the API endpoint
    url = "http://hydra.local:8080/v1/chat/completions"

    payload = {
        "messages": [
            {"role": "system", "content": system_prompt},
            {"role": "user", "content": document},
        ],
        "temperature": 0.2,
        "n_predict": 512,
        "response_format": "json_object",
    }

    response = requests.post(url, json=payload, timeout=(120, 180))
    response.raise_for_status()

    data = response.json()
    # print(json.dumps(data, indent=2))

    content = (
        data["choices"][0]["message"]["content"]
        .replace("```json", "")
        .replace("```", "")
    )
    tokens_per_second = data["timings"]["predicted_per_second"]

    return json.loads(content), tokens_per_second


def further_details(details: list[dict]) -> str:
    return "\n\n".join(
        [
            f"{detail['subtitle']}\n{detail['content']}\n\n"
            for detail in details
            if detail["subtitle"] != "Further Information and Other Local Ideas"
        ]
    )


for file in tqdm(
    list(walk_files("../data/backup")), desc="Summarizing facets", unit="record"
):
    with open(file, "r") as fp:
        record: dict = json.load(fp)

        if "llama_cpp" in record:
            continue

        document = f"""
{record["title"]} ({record["distance_km"]} km)
{record.get("display_address", "")}

{record["description"]}

{further_details(record["details"])}
        """

        facets, tokens_per_second = get_facets(record["description"])

        record["llama_cpp"] = True
        record.update(facets)

        with open(file, "w") as fp:
            json.dump(record, fp, indent=2)
