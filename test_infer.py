import json

with open("orlando_segunda_division.json", "r") as f:
    data = json.load(f)

# Find group matches to build "groups" to players mapping
group_players = {}
for m in data:
    if m.get("stage") == "group":
        g_id = m.get("group_id", "unknown")
        if g_id not in group_players:
            group_players[g_id] = set()
        if m.get("team_a_player_1_id"): group_players[g_id].add(m.get("team_a_player_1_id"))
        if m.get("team_b_player_1_id"): group_players[g_id].add(m.get("team_b_player_1_id"))

# Find knockout players
ko_players = set()
for m in data:
    if m.get("stage") not in ["group", "loser_bracket"] and not m.get("stage").startswith("tier"):
        if m.get("team_a_player_1_id"): ko_players.add(m.get("team_a_player_1_id"))
        if m.get("team_b_player_1_id"): ko_players.add(m.get("team_b_player_1_id"))

print("Knockout players:", len(ko_players))

# Count how many ko_players are in each group
max_advancing = 0
for g_id, players in group_players.items():
    advancing = len(players.intersection(ko_players))
    print(f"Group {g_id} has {advancing} advancing players")
    if advancing > max_advancing:
        max_advancing = advancing

print("Inferred PassCount:", max_advancing)
