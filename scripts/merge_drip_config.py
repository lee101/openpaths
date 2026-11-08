#!/usr/bin/env python3
import json
import sys

prod_path, git_path, out_path = sys.argv[1:4]
prod = json.load(open(prod_path))
git = json.load(open(git_path))
by_id = {e["id"]: i for i, e in enumerate(prod["emails"])}
added, updated = [], []
for entry in git["emails"]:
    i = by_id.get(entry["id"])
    if i is None:
        prod["emails"].append(entry)
        by_id[entry["id"]] = len(prod["emails"]) - 1
        added.append(entry["id"])
    elif prod["emails"][i] != entry:
        prod["emails"][i] = entry
        updated.append(entry["id"])
for key, value in git.items():
    if key != "emails" and key not in prod:
        prod[key] = value
ids = [e["id"] for e in prod["emails"]]
assert len(ids) == len(set(ids)), "duplicate email ids"
for e in prod["emails"]:
    if e.get("kind") == "model_launch":
        assert e.get("launch_date") and e.get("max_age_days", 0) > 0, f"email {e['id']} needs launch_date and max_age_days"
with open(out_path, "w") as f:
    json.dump(prod, f, indent=2, ensure_ascii=False)
    f.write("\n")
held = [e["id"] for e in prod["emails"] if e.get("hold")]
print(f"drip_config: added {added} updated {updated} held {held} prod-only kept {[i for i in ids if i not in {e['id'] for e in git['emails']}]}", file=sys.stderr)
