import json

with open("orlando_segunda_division.json", "r") as f:
    data = json.load(f)
    print("Total matches:", len(data))
    knockout_matches = [m for m in data if m.get("stage") != "group"]
    print("Knockout matches:", len(knockout_matches))
