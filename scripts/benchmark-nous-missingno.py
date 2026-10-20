#!/usr/bin/env python3
"""Run the one-shot pelican SVG benchmark, preserving response and timing."""
import argparse
import json
import os
from pathlib import Path
import re
import time
import urllib.request
import xml.etree.ElementTree as ET


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--output-dir", default="/tmp/openpaths-missingno-benchmark")
    args = parser.parse_args()
    key = os.environ.get("NOUS_API_KEY")
    if not key:
        parser.error("NOUS_API_KEY must be set; no request was sent")
    prompt = (
        "Draw a pelican riding a bicycle as an animated SVG. Return only a complete "
        "standalone SVG document."
    )
    payload = {
        "model": "stealth/missingno",
        "messages": [
            {"role": "system", "content": "You output only final code. No markdown, no prose. The first character must be <."},
            {"role": "user", "content": prompt},
        ],
        "max_tokens": 12000,
        "reasoning_effort": "medium",
        "stream": False,
    }
    request = urllib.request.Request(
        "https://inference-api.nousresearch.com/v1/chat/completions",
        data=json.dumps(payload).encode(),
        headers={"Authorization": "Bearer " + key, "Content-Type": "application/json"},
    )
    start = time.monotonic()
    with urllib.request.urlopen(request, timeout=600) as response:
        result = json.load(response)
    elapsed = time.monotonic() - start
    out = Path(args.output_dir)
    out.mkdir(parents=True, exist_ok=True)
    (out / "response.json").write_text(json.dumps(result, indent=2) + "\n")
    (out / "request.json").write_text(json.dumps(payload, indent=2) + "\n")
    choice = result["choices"][0]
    content = choice["message"].get("content") or ""
    match = re.search(r"<svg\b[\s\S]*?</svg>", content)
    if not match or choice.get("finish_reason") != "stop":
        raise SystemExit("No complete SVG with finish_reason=stop; raw response saved")
    svg = match.group(0)
    root = ET.fromstring(svg)
    for element in root.iter():
        if element.tag.rsplit("}", 1)[-1].lower() in {"script", "foreignobject"}:
            raise SystemExit("Unsafe SVG element; raw response saved, SVG not published")
        for name, value in element.attrib.items():
            if name.lower().startswith("on") or "javascript:" in value.lower():
                raise SystemExit("Unsafe SVG attribute; raw response saved")
            if name.rsplit("}", 1)[-1] == "href" and not value.startswith("#"):
                raise SystemExit("External SVG reference; raw response saved")
    (out / "missingno.svg").write_text(svg + "\n")
    summary = {"model": result.get("model"), "elapsed_seconds": round(elapsed, 3),
               "usage": result.get("usage"), "finish_reason": choice.get("finish_reason"),
               "visual_review": "pending; this single sample is not a coding benchmark score"}
    (out / "summary.json").write_text(json.dumps(summary, indent=2) + "\n")
    print(json.dumps(summary, indent=2))


if __name__ == "__main__":
    main()
