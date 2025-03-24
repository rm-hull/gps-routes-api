import json
import requests
from tqdm import tqdm

from utils import walk_files

# Define the API endpoint
url = "http://hydra.local:8080/v1/chat/completions"

# Define the prompt template
system_prompt = """
You are an expert at structured data extraction. Your task is to extract specific summary information for the given document, and output ONLY valid JSON with no additional text. Follow these steps:

1. Read the document carefully.
2. Extract the following details, but be sure to use generic descriptions and not the exact text from the document:
   - Estimated duration (e.g. use the distance and activity type to always provide time-based estimates like "1 hour", "half day", etc.)
   - Difficulty (easy, moderate, or hard)
   - Terrain (an array, e.g. ["woodland", "coastal", "mountain"])
   - Points of interest (an array, e.g. ["historical sites", "viewpoints"] - refrain from using specific names)
   - Facilities (an array, e.g. ["pub/cafe", "visitor center", "car park", "dog friendly", "wheelchair accessible", "play area"] )
   - Route type (e.g. "circular", "one-way", "out and back")
   - Activity type (an array, e.g. ["walking", "cycling", "horse riding", "bird watching", "swimming"])
3. be sure to include specific facilities only if they are noted in the document
4. If you are unsure about any detail, you may leave it blank or use a placeholder value.
5. Format your output exactly as the JSON below. Do not include any text before or after the JSON.
{
  "estimated_duration": "duration",
  "difficulty": "difficulty level",
  "terrain": ["Terrain 1", "Terrain 2", "Terrain 3"],
  "points_of_interest": ["Point of interest 1", "Point of interest 2", "Point of interest 3"],
  "facilities": ["Facility 1", "Facility 2", "Facility 3"],
  "route_type": "Route type",
  "activity_type": ["Activity type 1", "Activity type 2"]
}
Output only JSON. Do not include any other text or characters.
"""


def get_facets(document: str) -> tuple[dict, float]:

    payload = {
        "messages": [
            {"role": "system", "content": system_prompt},
            {"role": "user", "content": document},
        ],
        "temperature": 0.2,
        "n_predict": 512,
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


# # Example document
# documents = [
#     """
#     Blakeney Point Seals Walk Route (8.3 km)
# Beach Road, Cley Next The Sea, Cley next the Sea, North Norfolk, Norfolk, England, NR25 7RY, United Kingdom

# This beautiful National Nature Reserve on the Norfolk coast is a four-mile-long sand and shingle spit with sand dunes, salt marshes, tidal mudflats and farmland. There is a mixed colony of around 500 seals which can be seen on the beach or from boat trips departing from Morston Quay to Blakeney Point. This is a very popular walk, running for just over 7 miles along the beach and shingle reach. Start the walk from the good sized car park at Cley Beach at the end of Beach Road. It's located just to the north of the little village of Cley-Next-the-Sea. From the car park head west along the beach to the reserve where you will find a variety of rare flora and fauna. Look out for interesting plantlife including Sea Lavenders, Yellow-Horned Poppy and the white petals of Sea Campion. There is also an abundance of wildlife with migrant terns, the resident seals, wintering wildfowl and the occasional otter. The walk below takes you along the soft shingle beach and then on to the lifeboat house. You can also catch boat trips to Blakeney Point from Morston Quay which has an excellent visitor centre. It has a wealth of information about the area and you can catch a boat from the pretty quay to the reserve. Morston is located just to the west of both Blakeney and Cley. The Peddars Way and Norfolk Coast Path runs past the reserve so you have the option of following this trail to some other lovely locations in the area. On the trail to the west is the Stiffkey Salt Marsh where you will find a vast open expanse of salt marshes which attracts large numbers of birdlife including waders and wintering wildfowl. If you head east along the trail you will pass Blakeney with its pretty key and then on to the Cley Marshes Nature Reserve. This 430 acre reserve contains reed beds, freshwater marsh, pools and wet meadows.

# Pubs/Cafes
# Head to the 18th century Kings Arms in Blakeney for some post walk refreshments. The Georgian Inn has an interior with bags of character including cosy little alcoves and interesting pictures, photos and posters on the walls. Outside there's a nice garden area where you can enjoy your meal on a fine day. The pub is located in a wonderful spot just yards from Blakeney Quay. You can find the pub on Westgate Street at a postcode of NR25 7NQ for your sat navs. The pub is also dog friendly if you have your canine friend with you.
# """,
#     """
# Edinburgh Coastal Beach Walk Route (6.7 km)
# The Prom, Westbank, Portobello, City of Edinburgh, Alba / Scotland, EH15 1DT, United Kingdom

# This walk takes you along the beach and promenade in Portobello on the eastern side of the city of Edinburgh. The two mile stretch of sandy beach is a lovely place for a coastal walk in the city. Along the way you can enjoy splendid views over the Firth of Forth to Inchkeith Island. The village is also very attractive with some fine Georgian and Victorian architecture to see. Other notable features include a Victorian swimming pool, featuring an original Aerotone (the forerunner to the modern Jacuzzi) and an authentic Turkish Bath. You can park at the public car park on the sea front and then enjoy a circular walk by following the promenade path before heading onto the wide sandy beach. The area is dotted with Ice cream parlours, arcades and beach-side cafes and bars so there's plenty of options for refreshments. You could extend the walk by continuing east from Joppa to visit nearby Musselburgh. In this area there's another nice beach, riverside trails along the Esk and the National Trust's delightful Inveresk Gardens to see.

# Pubs/Cafes
# There's several nice cafes on the prom where you can stop and enjoy some refreshments. Crumbs of Portobello serves haggis and crepes with outdoor seating available. You can find them at postcode EH15 1HJ for your sat navs.

# Dog Walking
# The beach is a popular place for dog walking so you'll probably see other owners on a fine day. Crumbs of Portobello mentioned above is also dog friendly with treats available.
# """,
#     """
# Battle of the Boyne Walk Route (4.3 km)
# Canal Towpath, Saint Mary's ED, The Municipal District of Laytown — Bettystown, County Meath, Leinster, A92 NY2V, Éire / Ireland

# This walk visits the site of the 17th century Battle of the Boyne in County Meath. There are miles of free walkways to follow around the historic area which is the site of a noteworthy battle for the British throne. The area also includes nice waterside trails along the Boyne canal and the river. You can park at the visitor centre before enjoying a stroll through the adjacent pretty gardens. Visitors also have the option of self-guided walks through the core battle site and Oldbridge Estate. These walks are freely accessible, with parking provided in the Main Car Park. At the start and various access points of the walks, you'll find several orientation panels and maps. The routes are colour-coded on information panels, and the tops of the way markers on the walks are painted to correspond with these colours. Each walk is both timed and measured from its starting point. It's recommended that visitors wear appropriate footwear, as the paths are grassy, with the exception of the Boyne Canal towpath which is gravel-surfaced and adjoins the site. You can reach the site from Drogheda by following the Drogheda Boyne Greenway west from the town. The shared cycling and walking trail utilises sections of boardwalk and riverside paths along the River Boyne. It's a pleasant way to reach the site without a car. To continue your historical walking in the area head west to Slane Castle. The castle is well worth a visit with an interesting history, a whiskey distillery, woodland trails and a natural amphitheatre which has been used for many concerts. You can also pick up the lovely Boyne Valley Walk in this area.

# Dog Walking
# Dogs are welcome on site but owners must keep them on leads at all times and pick up after them. Please note that dogs are not permitted in the Walled Garden or the Visitor Centre with the exception of Guide Dogs.
# """,
#     """
# Lanty's Tarn Circular Walk Route (5 km)
# A592, Glenridding, Patterdale, Westmorland and Furness, England, CA11 0PA, United Kingdom

# This circular walk visits Lanty's Tarn from the village of Glenridding in the Lake District National Park. The pretty tarn sits just to the south of the village. You can follow footpaths up the site where there's a pleasant footpath along the secluded and peaceful tarn. The walk then heads west through the Brownend plantation towards the pretty Mires Beck. Along the way you can enjoy splendid views over Ullswater Lake and Grisedale. You then follow the Mires Beck north to Rattlebeck Bridge where you can pick up another waterside footpath along the Glenridding beck to return you to the village. To extend the walk you could head west to visit Red Tarn on a longer, more challenging walk. To continue your walking in the Glenridding area try the Glenridding to Aira Force Walk which visits a beautiful waterfall above the lake.
# """,
#     """
# Nidd Gorge Walking Route (11 km)
# Conyngham Hall, Beryl Burton Cycleway, Tentergate, Calcutt, Knaresborough, North Yorkshire, York and North Yorkshire, England, HG5 9AY, United Kingdom

# Explore this beautiful river gorge on this waterside walk in Knaresborough. This circular walk takes you through the wooded gorge before crossing the Nidd Viaduct and returning to Knaresborough through the countryside around Old Bilton. It makes use of the Harrogate Ringway long distance path for part of the route. The walk starts in Knaresborough at the Conyngham Hall car park near the town centre and train station. You then head along Harrogate Road and High Bond End Road before turning down Lands Lane towards the river. The trail then weaves its way through the ancient woodland to Viaduct Wood and the Nidd Viaduct. Look out for a variety wildlife such as tawny owl, roe deer, woodpeckers and herons on the water. You then cross the Nidd Viaduct and head through the village of Old Bilton. The final section takes you through the countryside along Bilton Lane to the finish point back at the car park. Although this route is designed for walkers the section from the village of Old Bilton to Knaresborough follows the Beryl Burton Cycleway so cyclists can enjoy a nice traffic free path in the area too. The Knaresborough Round passes through the gorge so you could pick up this 20 mile circular trail to extend your walk. There's also the shorter Knaresborough River Walk which starts from the nearby 12th century Knaresborough Castle and explores the pretty grounds of Conyngham Hall.

# Pubs/Cafes
# In the little village of Old Bilton you could stop for some refreshments at the Gardeners Arms. The classic old pub has some features like wainscotting, stone flags, and stone fireplaces still remaining. There's also a nice large garden area to sit out in when the weather is fine. You can find the pub on Bilton Lane with a postcode of HG1 4DH for your sat navs. Just north of the gorge you'll find the little village of Scotton. Here you can pay a visit to the noteworthy Guy Fawkes Arms. Originally constructed in the 17th century, the pub takes its name from Guy Fawkes who used to live in the village. They do excellent food and can be found on the Main Street at postcode HG5 9HU.

# Dog Walking
# The gorge is a fine place for a dog walk and there is also the opportunity for a paddle in the quieter parts of the river. The Guy Fawkes Arms mentioned above is also dog friendly.
# """,
#     """
# Harrogate Ringway Walking Route (32 km)
# 45, Station Road, Spacey Houses, Pannal and Burn Bridge, Pannal, North Yorkshire, York and North Yorkshire, England, HG3 1JS, United Kingdom

# This is a 20 mile circular walk around Harrogate. There's much to enjoy on this route including the RHS Harlow Carr Gardens. With lakes, woodland and a wildflower meadow it is well worth spending some time in. It's also a short stroll through the Pinewoods to the nearby Valley Gardens, another highlight of the town. The 17 acre gardens also include woodland trails, several mineral springs and historic buildings including the Sun Pavilion and colonnades. The path also includes a lovely long stretch along the River Nidd to the delightful market town of Knaresborough. You'll pass through the Nidd Gorge, a peaceful wooded gorge wth lots of wildlife to look out for such as tawny owl, roe deer, woodpeckers and herons on the water. Another waterside stretch along the River Crimple follows soon after Knaresborough with splendid views of the Yorkshire countryside a further attraction on this challenging walk. This trail joins with the Knaresborough Round around the historic town of Knaresborough. You can pick up this 20 mile circular trail to further explore the countryside and villages of the area. The route also passes close to the splendid Plumpton Rocks. It's well worth a small detour to visit this delightful hidden gem. Almscliffe Crag is also just off the route near North Rigton, providing splendid views over the Wharfe Valley countryside.

# Pubs/Cafes
# In the little village of Old Bilton you could stop for some refreshments at the Gardeners Arms. The classic old pub has some features like wainscotting, stone flags, and stone fireplaces still remaining. There's also a nice large garden area to sit out in when the weather is fine. You can find the pub on Bilton Lane with a postcode of HG1 4DH for your sat navs.
# """,
#     """
# Cley Marshes Nature Reserve Walking Route (8 km)
# The Quay, Cley Next The Sea, Cley next the Sea, North Norfolk, Norfolk, England, NR25 7RP, United Kingdom

# This walk takes you around the stunning Cley Marshes on the Norfolk coast at Cley next the Sea. You start at the windmill at Cley next the Sea and head through the reserve to the coast, before following a walking trail and country lanes through the countryside and returning to the windmill. Cley Marshes contains 430 acres of reed beds, freshwater marsh, pools and wet meadows. An abundance of rare birdlife can be seen at the site, including pied avocets on the islands, western marsh harriers, Eurasian bitterns and bearded reedlings. There are five bird hides and an excellent visitor centre with a cafe, shop, viewing areas (including viewing from a camera on the reserve) and an exhibition area. Plantlife at the reserve includes biting stonecrop, sea campion, yellow horned poppy, sea thrift, bird's foot trefoil and sea beet. Wilidlife includes Water Voles, hares and otters. If you would like to continue your walk the Peddars Way and Norfolk Coast Path runs through the reserve so you could follow this trail west to Morston Quay and catch a boat to Blakeney Point Nature Reserve where can you go seal watching! A little further on is Stiffkey Salt Marsh where you will find a vast open expanse of salt marshes which attracts large numbers of birdlife including waders and wintering wildfowl.
# """,
#     """
# London Bridge to Tower Bridge Walk Route (2.5 km)
# London Bridge, Monument, London Borough of Southwark, City of London, Greater London, England, EC4R 3AE, United Kingdom

# This walk takes you from London Bridge to Tower Bridge in the City of London. It's just over half a mile between the two iconic bridges so the walk should take around 10-15 minutes. This circular route crosses both bridges, passing along both sides of the River Thames on the Thames Path National Trail. It extends the walk to a distance of 1.5 miles and includes two bridge crossings where you can enjoy fine views down the river to some iconic London highlights. On the way you will pass the famous Tower of London where you can see the Crown Jewels of England and the notorious prison. There are also great views of London's more modern buildings including the Gherkin, City Hall and the Shard. You can extend your walk by heading west to visit some of London's beautiful parks including St James's Park, Hyde Park and Kensington Gardens. The long distance Diana Princess of Wales Memorial Walk is a popular way of exploring this area on foot. You could also head east on the Tower Bridge to Greenwich Walk which will take you to Greenwich Park where you will find the Royal Observatory, the home of Greenwich Mean Time, the Prime Meridian and London's only Planetarium.
# """,
#     """
# Coire an t-Sneachda Walking Route (5.5 km)
# Northern Corries Path, Highland, Alba / Scotland, PH22 1RB, United Kingdom

# Follow a good path to this stunning glacial corrie in the Cairngorms. You start off from the Cairngorm Ski Centre car park and soon pick up the well maintained path to this spectacular corrie. As you climb you will see wonderful views of the Rothiemurchus Forest and Loch Morlich while crossing pretty streams on huge stepping stones. The surrounding glacial cliffs and huge boulders add to the dramatic nature of this stunning area. In the colder months you may see ice climbers attempting Magic Crack. The climb to Cairn Gorm also starts from the same car park so you can continue your walking in the area on this path.
# """,
#     """
# Peddars Way and Norfolk Coast Path National Trail Walking Route (150 km)
# Welcome to Knettishall Heath Nature Reserve, Spalding's Chair Hill, West Suffolk, Suffolk, England, IP24 2SH, United Kingdom

# The Peddars Way & Norfolk Coast Path begins at Knettishall Heath Country Park in Suffolk and takes you to Holme next the sea on the Norfolk coast along designated footpaths. Some wonderful coastal scenery then follows as you head east along the Norfolk coast path from Hunstanton to Cromer.

# Pubs/Cafes
# In Blakeney you could enjoy a pit stop at the 18th century Kings Arms. The Georgian Inn has an interior with bags of character including cosy little alcoves and interesting pictures, photos and posters on the walls. Outside there's a nice garden area where you can enjoy your meal on a fine day. The pub is located in a wonderful spot just yards from Blakeney Quay. You can find the pub on Westgate Street at a postcode of NR25 7NQ for your sat navs. The pub is also dog friendly if you have your canine friend with you. The Dun Cow in Salthouse is another great choice. They do good food and have an interesting interior with wood beams and an exposed brick fireplace. Outside there's a lovely garden area with fine views over the Salthouse Marshes. You can find the inn on Purdy Street at a postcode of NR25 7XA. The pub is also dog friendly. In the attractive village of Great Massingham there's the Dabbling Duck for to consider. The fine country pub serves good quality food and also provides accommodation if you'd like to stay over. Inside there's roaring fires in the winter and a garden to enjoy in the summer. You can find them in a lovely spot between the two ponds at 11 Abbey Road, with postcode PE32 2HN for your sat navs. The pub has been included in The Sunday Times list of the best 80 places to stay in Britain in 2020.
# """,
# ]

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

        # print(document)
        # print("======================================")
        facets, tokens_per_second = get_facets(record["description"])

        # print(json.dumps(facets, indent=2))
        # print(f"Tokens per second: {tokens_per_second:.2f}")
        # print("======================================")
        record["llama_cpp"] = True
        record.update(facets)

        with open(file, "w") as fp:
            json.dump(record, fp, indent=2)
