#!/usr/bin/env python3
"""Checks an `fpack build --json` result in CI.

    python3 scripts/e2e_check.py result.json apk web ...
    python3 scripts/e2e_check.py result.json dmg --notarization dmg=stapled
    python3 scripts/e2e_check.py result.json --failed dmg

--notarization TARGET=STATE requires every artifact of TARGET that was sent
to Apple to report that notarization state; --failed TARGET requires TARGET
to have failed.

Fails unless every listed target succeeded, every artifact exists with the
reported size and SHA-256/512, and SHA256SUMS/SHA512SUMS lists them. Leading non-JSON lines
(pub prints "Resolving dependencies..." for path-activated packages) are
ignored.
"""
import hashlib
import json
import os
import sys


def main() -> int:
    # Windows consoles default to cp1252, which can't print ✓/✗.
    for stream in (sys.stdout, sys.stderr):
        if hasattr(stream, "reconfigure"):
            stream.reconfigure(encoding="utf-8", errors="replace")
    path, want, notary, failed = sys.argv[1], [], {}, []
    args = iter(sys.argv[2:])
    for a in args:
        if a == "--notarization":
            t, _, st = next(args).partition("=")
            notary[t] = st
        elif a == "--failed":
            failed.append(next(args))
        else:
            want.append(a)
    text = open(path, encoding="utf-8-sig").read()
    start = text.find("{")
    if start < 0:
        print(f"no JSON in {path}:\n{text}")
        return 1
    result = json.loads(text[start:])
    targets = {t["target"]: t for t in result.get("targets", [])}
    ok = True
    for name in failed:
        t = targets.get(name)
        if t is None or t["status"] != "failed":
            print(f"✗ {name}: expected to fail, got {t and t['status']}")
            ok = False
        else:
            print(f"✓ {name}: failed as expected – {t.get('error')}")
    for name in notary:
        if name not in want:
            want.append(name)
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
            data = open(p, "rb").read()
            good = size == a.get("size")
            algo = "sha512" if a.get("sha512") else "sha256"
            if a.get(algo):
                good &= hashlib.new(algo, data).hexdigest() == a[algo]
            ok &= good
            state = (a.get("notarization") or {}).get("state")
            extra = f"  notarization: {state}" if state else ""
            if name in notary and a["file"].endswith((".dmg", ".zip", ".pkg")) and state != notary[name]:
                print(f"✗ {name}: {a['file']}: notarization state {state!r}, want {notary[name]!r}")
                ok = False
            print(f"{'✓' if good else '✗'} {name}: {a['file']}  {size / 1e6:.1f} MB  {a['kind']}{extra}")
            sums = os.path.join(os.path.dirname(p), algo.upper() + "SUMS")
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
