#!/usr/bin/env python3
"""Check internal links and #anchors of the built site.

Every page renders all locales; each locale lives in an element with
data-l10n="<lang>". A link inside a locale block must point at an existing
page, and its #anchor must exist in the same locale (or outside any locale
block), otherwise readers of that language land on a hidden heading.

usage: check_links.py --dir build/jaspr --base /fpack/
"""
import argparse
import os
import sys
from html.parser import HTMLParser
from urllib.parse import unquote, urlsplit

VOID = {"area", "base", "br", "col", "embed", "hr", "img", "input", "link", "meta", "source", "track", "wbr"}


class Page(HTMLParser):
    def __init__(self):
        super().__init__(convert_charrefs=True)
        self.stack = []  # (tag, lang or None)
        self.ids = {}  # id -> set of langs (None = all)
        self.links = []  # (href, lang)

    def lang(self):
        for _, l in reversed(self.stack):
            if l:
                return l
        return None

    def handle_starttag(self, tag, attrs):
        a = dict(attrs)
        lang = a.get("data-l10n")
        if tag not in VOID:
            self.stack.append((tag, lang))
        cur = lang or self.lang()
        if a.get("id"):
            self.ids.setdefault(a["id"], set()).add(cur)
        if tag == "a" and a.get("href"):
            self.links.append((a["href"], cur))

    def handle_startendtag(self, tag, attrs):
        self.handle_starttag(tag, attrs)
        if tag not in VOID and self.stack:
            self.stack.pop()

    def handle_endtag(self, tag):
        for i in range(len(self.stack) - 1, -1, -1):
            if self.stack[i][0] == tag:
                del self.stack[i:]
                break


def page_file(root, path):
    path = path.strip("/")
    cand = [os.path.join(root, path, "index.html"), os.path.join(root, path)]
    if not path:
        cand = [os.path.join(root, "index.html")]
    for c in cand:
        if os.path.isfile(c):
            return c
    return None


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--dir", required=True)
    ap.add_argument("--base", default="/")
    args = ap.parse_args()
    root, base = args.dir, "/" + args.base.strip("/") + "/" if args.base.strip("/") else "/"
    parsed = {}

    def load(f):
        if f not in parsed:
            p = Page()
            with open(f, encoding="utf-8") as h:
                p.feed(h.read())
            parsed[f] = p
        return parsed[f]

    pages = []
    for d, _, files in os.walk(root):
        for n in files:
            if n == "index.html":
                pages.append(os.path.join(d, n))
    errors, checked = [], 0
    for f in sorted(pages):
        rel = "/" + os.path.relpath(os.path.dirname(f), root).replace(os.sep, "/").strip(".")
        for href, lang in load(f).links:
            u = urlsplit(href)
            if u.scheme or u.netloc or href.startswith("mailto:"):
                continue
            if rel == "/changelog" and not href.startswith(("/", "#")):
                continue  # relative links copied verbatim from CHANGELOG.md
            path = unquote(u.path)
            if not path:
                target = f
            else:
                if not path.startswith(base):
                    errors.append(f"{rel}: link outside the site base: {href}")
                    continue
                if os.path.splitext(path)[1] not in ("", ".html"):
                    if not os.path.exists(os.path.join(root, path[len(base):])):
                        errors.append(f"{rel}: missing file {href}")
                    continue
                target = page_file(root, path[len(base):])
                if not target:
                    errors.append(f"{rel} [{lang or '*'}]: broken link {href}")
                    continue
            checked += 1
            frag = unquote(u.fragment)
            if not frag:
                continue
            langs = load(target).ids.get(frag)
            if not langs:
                errors.append(f"{rel} [{lang or '*'}]: missing anchor {href}")
            elif lang and None not in langs and lang not in langs:
                errors.append(f"{rel} [{lang}]: anchor {href} exists only in {sorted(langs)}")
    for e in errors:
        print(e)
    print(f"check_links: {checked} internal links in {len(pages)} pages, {len(errors)} problems")
    sys.exit(1 if errors else 0)


if __name__ == "__main__":
    main()
