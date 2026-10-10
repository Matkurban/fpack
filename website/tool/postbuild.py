#!/usr/bin/env python3
"""Post-process the static Jaspr build (build/jaspr) for GitHub Pages.

- drops development-only files (packages/, .dart_tool/)
- writes search-index.json (one entry per locale and section, used by web/site.js)
- turns /404 into 404.html, writes sitemap.xml and robots.txt
- writes redirects from the old /zh/<page> URLs to /<page>?lang=zh
- rewrites absolute links for a sub-path deployment (--base /fpack/)

Usage: python3 tool/postbuild.py [--base /fpack/] [--site-url https://matkurban.github.io/fpack/]
"""
import argparse
import html
import json
import os
import re
import shutil
import sys
from html.parser import HTMLParser

ap = argparse.ArgumentParser()
ap.add_argument('--dir', default='build/jaspr')
ap.add_argument('--base', default='/')
ap.add_argument('--site-url', default='')
args = ap.parse_args()
root = args.dir
base = args.base if args.base.endswith('/') else args.base + '/'
if not base.startswith('/'):
    sys.exit('--base must start with /')

for junk in ['packages', '.dart_tool', '.build.manifest']:
    path = os.path.join(root, junk)
    if os.path.isdir(path):
        shutil.rmtree(path)
    elif os.path.exists(path):
        os.remove(path)

not_found = os.path.join(root, '404', 'index.html')
if os.path.exists(not_found):
    shutil.move(not_found, os.path.join(root, '404.html'))
    shutil.rmtree(os.path.join(root, '404'))


class SearchExtractor(HTMLParser):
    """Collects (locale, title, section, anchor, text) from the page's <main>."""

    SKIP = {'script', 'style', 'button'}
    VOID = {'area', 'base', 'br', 'col', 'embed', 'hr', 'img', 'input', 'link', 'meta', 'source', 'track', 'wbr'}

    def __init__(self):
        super().__init__(convert_charrefs=True)
        self.stack = []          # (tag, locale or None)
        self.in_main = 0
        self.skip = 0
        self.heading = None      # (tag, id, [text]) while inside h1-h3
        self.titles = {}
        self.sections = []       # [locale, section, anchor, [text]]
        self.current = {}

    def locale(self):
        for _, loc in reversed(self.stack):
            if loc:
                return loc
        return None

    def handle_starttag(self, tag, attrs):
        if tag in self.VOID:
            return
        a = dict(attrs)
        self.stack.append((tag, a.get('data-l10n')))
        if tag == 'main':
            self.in_main += 1
        if {'toc', 'site-footer'} & set((a.get('class') or '').split()):
            self.skip += 1
            self.stack[-1] = (tag + ':skip', self.stack[-1][1])
        if tag in self.SKIP:
            self.skip += 1
        if self.in_main and not self.skip and tag in ('h1', 'h2', 'h3'):
            self.heading = (tag, a.get('id', ''), [])

    def handle_endtag(self, tag):
        while self.stack:
            t, _ = self.stack.pop()
            if t.endswith(':skip'):
                self.skip -= 1
                t = t[:-5]
            if t in self.SKIP:
                self.skip -= 1
            if t == 'main':
                self.in_main -= 1
            if self.heading and t == self.heading[0]:
                self.finish_heading()
            if t == tag:
                break

    def finish_heading(self):
        tag, anchor, text = self.heading
        self.heading = None
        loc = self.locale()
        title = clean(''.join(text)).rstrip('#').strip()
        if not loc:
            return
        if tag == 'h1':
            self.titles[loc] = title
        else:
            self.current[loc] = [loc, title, anchor, []]
            self.sections.append(self.current[loc])

    def handle_data(self, data):
        if not self.in_main or self.skip:
            return
        if self.heading:
            self.heading[2].append(data)
            return
        loc = self.locale()
        if not loc:
            return
        if loc not in self.current:
            self.current[loc] = [loc, '', '', []]
            self.sections.append(self.current[loc])
        self.current[loc][3].append(data)


def clean(s):
    return re.sub(r'\s+', ' ', s).strip()


pages = sorted(os.path.join(d, f) for d, _, fs in os.walk(root) for f in fs if f == 'index.html')
index, urls = [], []
for path in pages:
    rel = os.path.relpath(os.path.dirname(path), root).replace(os.sep, '/')
    url = '/' if rel == '.' else '/' + rel
    urls.append(url)
    h = open(path, encoding='utf-8').read()

    parser = SearchExtractor()
    parser.feed(h)
    for loc, section, anchor, text in parser.sections:
        body = clean(' '.join(text))
        if body:
            index.append({'lang': loc, 'url': url + (f'#{anchor}' if anchor else ''),
                          'title': parser.titles.get(loc, 'fpack'), 'section': section, 'text': body[:2000]})

    if base != '/':
        h = re.sub(r'(\s(?:href|src))="/(?!/)', lambda m: f'{m.group(1)}="{base}', h)
    open(path, 'w', encoding='utf-8').write(h)

if base != '/' and os.path.exists(os.path.join(root, '404.html')):
    p404 = os.path.join(root, '404.html')
    h = open(p404, encoding='utf-8').read()
    open(p404, 'w', encoding='utf-8').write(re.sub(r'(\s(?:href|src))="/(?!/)', lambda m: f'{m.group(1)}="{base}', h))

with open(os.path.join(root, 'search-index.json'), 'w', encoding='utf-8') as f:
    json.dump(index, f, ensure_ascii=False, separators=(',', ':'))

# Old URLs: the Chinese pages used to live under /zh/.
for url in urls:
    target = base + (url.strip('/') + '/' if url != '/' else '')
    out = os.path.join(root, 'zh', url.lstrip('/'), 'index.html')
    os.makedirs(os.path.dirname(out), exist_ok=True)
    with open(out, 'w', encoding='utf-8') as f:
        t = html.escape(target + '?lang=zh')
        f.write(f'''<!DOCTYPE html><html lang="zh-CN"><head><meta charset="utf-8"><title>fpack</title>
<meta name="robots" content="noindex"><link rel="canonical" href="{t}"><meta http-equiv="refresh" content="0; url={t}">
<script>location.replace({json.dumps(target + '?lang=zh')} + location.hash)</script></head>
<body><a href="{t}">{t}</a></body></html>
''')

if args.site_url:
    site = args.site_url if args.site_url.endswith('/') else args.site_url + '/'
    with open(os.path.join(root, 'sitemap.xml'), 'w') as f:
        f.write('<?xml version="1.0" encoding="UTF-8"?>\n'
                '<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9" xmlns:xhtml="http://www.w3.org/1999/xhtml">\n')
        for url in urls:
            loc = html.escape(site + url.lstrip('/'))
            f.write(f'  <url><loc>{loc}</loc>'
                    f'<xhtml:link rel="alternate" hreflang="en" href="{loc}"/>'
                    f'<xhtml:link rel="alternate" hreflang="zh-CN" href="{loc}?lang=zh"/></url>\n')
        f.write('</urlset>\n')
    with open(os.path.join(root, 'robots.txt'), 'w') as f:
        f.write(f'User-agent: *\nAllow: /\nSitemap: {site}sitemap.xml\n')

open(os.path.join(root, '.nojekyll'), 'w').close()
print(f'postbuild: {len(pages)} pages, {len(index)} search entries, base {base}')
