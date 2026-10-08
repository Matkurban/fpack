#!/usr/bin/env python3
"""Checks an `fpack build --json` result in CI.

    python3 scripts/e2e_check.py result.json apk web ...

Fails unless every listed target succeeded, every artifact exists with the
reported size and SHA-256, and SHA256SUMS lists them. Leading non-JSON lines
(pub prints "Resolving dependencies..." for path-activated packages) are
ignored.
"""
import hashlib
import json
import os
import sys


def main() -> int:
    path, want = sys.argv[1], sys.argv[2:]
    text = open(path, encoding="utf-8-sig").read()
    start = text.find("{")
    if start < 0:
        print(f"no JSON in {path}:\n{text}")
        return 1
    result = json.loads(text[start:])
    targets = {t["target"]: t for t in result.get("targets", [])}
    ok = True
    for name in want:
        t = targets.get(name)
        if t is None:
            print(f"✗ {name}: missing from result")
            ok = False
            continue
        if t["status"] != "success":
            print(f"✗ {name}: {t['status']} – {t.get('reason') or t.get('error')}")
            for line in t.get("errorExcerpt") or []:
                print("    " + line)
            if t.get("hint"):
                print("    hint: " + t["hint"])
            ok = False
            continue
        for a in t.get("artifacts") or []:
            p = a["path"]
            if not os.path.isfile(p):
                print(f"✗ {name}: artifact missing: {p}")
                ok = False
                continue
            size = os.path.getsize(p)
            sha = hashlib.sha256(open(p, "rb").read()).hexdigest()
            good = size == a.get("size") and sha == a.get("sha256", sha)
            ok &= good
            print(f"{'✓' if good else '✗'} {name}: {a['file']}  {size / 1e6:.1f} MB  {a['kind']}")
            sums = os.path.join(os.path.dirname(p), "SHA256SUMS")
            if not os.path.isfile(sums) or a["file"] not in open(sums, encoding="utf-8").read():
                print(f"✗ {name}: {a['file']} not listed in {sums}")
                ok = False
        for n in t.get("notes") or []:
            print(f"    note: {n}")
        for w in t.get("warnings") or []:
            print(f"    warning: {w.splitlines()[0]}")
    return 0 if ok else 1


if __name__ == "__main__":
    sys.exit(main())
