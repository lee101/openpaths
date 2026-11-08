#!/usr/bin/env python3
import re
import sys

import yaml

prod_path, git_path, out_path = sys.argv[1:4]
prod = open(prod_path).read()
git = open(git_path).read()
ID = re.compile(r'^  - id: "?([^"\n]+)"?\s*$')


def blocks(text):
    lines = text.split("\n")
    start = next(i for i, l in enumerate(lines) if l == "models:")
    end = len(lines)
    for i in range(start + 1, len(lines)):
        if lines[i] and not lines[i][0].isspace() and not lines[i].startswith("#"):
            end = i
            break
    out, cur, cid = {}, [], None
    for l in lines[start + 1:end]:
        m = ID.match(l)
        if m:
            if cid:
                out[cid] = cur
            cid, cur = m.group(1), [l]
        elif cid:
            cur.append(l)
    if cid:
        out[cid] = cur
    return lines, start, end, out


plines, pstart, pend, pblocks = blocks(prod)
_, _, _, gblocks = blocks(git)
added = [i for i in gblocks if i not in pblocks and "    provider: openrouter" not in gblocks[i]]
ins = []
for i in added:
    b = list(gblocks[i])
    while b and not b[-1].strip():
        b.pop()
    ins += [""] + b
tail = pend
while tail > pstart and not plines[tail - 1].strip():
    tail -= 1
merged = plines[:tail] + ins + [""] + plines[tail:] if ins else plines
text = "\n".join(merged)
data = yaml.safe_load(text)
before = len(yaml.safe_load(prod)["models"])
assert len(data["models"]) == before + len(added), "model count mismatch"
open(out_path, "w").write(text)
changed = [i for i in gblocks if i in pblocks and [l for l in gblocks[i] if l.strip()] != [l for l in pblocks[i] if l.strip()]]
print(f"config: prod models {len(pblocks)} added {added} git-differs-kept-prod {len(changed)}", file=sys.stderr)
