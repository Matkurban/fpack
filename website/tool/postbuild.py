#!/usr/bin/env python3
"""Post-process the static Jaspr build (build/jaspr) for GitHub Pages.

- drops development-only files (packages/, .dart_tool/)
- gives Chinese headings readable anchors and translates the TOC title
- writes search-index.json (used by web/search.js), sitemap.xml, 404.html
- rewrites absolute links for a sub-path deployment (--base /fpack/)

Usage: python3 tool/postbuild.py [--base /fpack/] [--site-url https://matkurban.github.io/fpack/]
"""
import argparse, html, json, os, re, shutil, sys

ap = argparse.ArgumentParser()
ap.add_argument('--dir', default='build/jaspr')
ap.add_argument('--base', default='/')
ap.add_argument('--site-url', default='')
a = ap.parse_args()
root = a.dir
base = a.base if a.base.endswith('/') else a.base + '/'
if not base.startswith('/'):
    sys.exit('--base must start with /')

for junk in ['packages', '.dart_tool', '.build.manifest']:
    p = os.path.join(root, junk)
    if os.path.isdir(p):
        shutil.rmtree(p)
    elif os.path.exists(p):
        os.remove(p)

def strip_tags(s):
    s = re.sub(r'<(script|style)\b.*?</\1>', ' ', s, flags=re.S)
    s = re.sub(r'<[^>]+>', ' ', s)
    return re.sub(r'\s+', ' ', html.unescape(s)).strip()

def slug(text, used):
    s = re.sub(r'[^\w\s-]', '', text.lower(), flags=re.U)
    s = re.sub(r'[\s_]+', '-', s).strip('-') or 'section'
    out, n = s, 1
    while out in used:
        out = f'{s}-{n}'; n += 1
    used.add(out)
    return out

HEADING = re.compile(r'<h([23]) id="([^"]*)" anchor="true"><span>(.*?)</span><a href="([^"#]*)#[^"]*">#</a></h\1>', re.S)
pages, index = [], []
for dirpath, _, files in os.walk(root):
    for f in files:
        if f == 'index.html':
            pages.append(os.path.join(dirpath, f))
pages.sort()

for path in pages:
    rel = os.path.relpath(os.path.dirname(path), root).replace(os.sep, '/')
    url = '/' if rel == '.' else '/' + rel
    h = open(path, encoding='utf-8').read()
    zh = 'lang="zh-CN"' in h[:300]

    if zh:
        used, mapping = set(), []
        def fix(m):
            new = slug(strip_tags(m.group(3)), used)
            mapping.append((m.group(2), strip_tags(m.group(3)), new))
            return f'<h{m.group(1)} id="{new}" anchor="true"><span>{m.group(3)}</span><a href="{m.group(4)}#{new}">#</a></h{m.group(1)}>'
        h = HEADING.sub(fix, h)
        for old, text, new in mapping:
            h = h.replace(f'#{old}">{html.escape(text, quote=False)}</a>', f'#{new}">{html.escape(text, quote=False)}</a>', 1)
        h = h.replace('<h3>On this page</h3>', '<h3>本页内容</h3>')

    # search index: one entry per page section
    title_m = re.search(r'<div class="content-header">\s*<h1>(.*?)</h1>', h, re.S)
    title = strip_tags(title_m.group(1)) if title_m else 'fpack'
    start = h.find('<div class="content-header">')
    end = h.find('<aside class="toc">')
    body = h[start:end] if start >= 0 and end > start else ''
    parts = re.split(r'(<h[23] id="[^"]*" anchor="true">.*?</h[23]>)', body, flags=re.S)
    section, anchor, buf = '', '', []
    def flush():
        text = strip_tags(' '.join(buf)).replace(' #', '')
        if text:
            index.append({'lang': 'zh' if zh else 'en', 'url': url + (f'#{anchor}' if anchor else ''),
                          'title': title, 'section': section, 'text': text[:2000]})
    for part in parts:
        m = re.match(r'<h[23] id="([^"]*)"[^>]*><span>(.*?)</span>', part, re.S)
        if m:
            flush(); buf = []
            anchor, section = m.group(1), strip_tags(m.group(2))
        else:
            buf.append(part)
    flush()

    if base != '/':
        h = re.sub(r'(\s(?:href|src))="/(?!/)', lambda m: f'{m.group(1)}="{base}', h)
        h = h.replace('<base href="/"/>', f'<base href="{base}"/>').replace('<base href="/">', f'<base href="{base}">')
    open(path, 'w', encoding='utf-8').write(h)

with open(os.path.join(root, 'search-index.json'), 'w', encoding='utf-8') as f:
    json.dump(index, f, ensure_ascii=False, separators=(',', ':'))

if a.site_url:
    site = a.site_url if a.site_url.endswith('/') else a.site_url + '/'
    urls = []
    for path in pages:
        rel = os.path.relpath(os.path.dirname(path), root).replace(os.sep, '/')
        urls.append(site + ('' if rel == '.' else rel))
    with open(os.path.join(root, 'sitemap.xml'), 'w') as f:
        f.write('<?xml version="1.0" encoding="UTF-8"?>\n<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">\n')
        for u in urls:
            f.write(f'  <url><loc>{html.escape(u)}</loc></url>\n')
        f.write('</urlset>\n')
    with open(os.path.join(root, 'robots.txt'), 'w') as f:
        f.write(f'User-agent: *\nAllow: /\nSitemap: {site}sitemap.xml\n')

with open(os.path.join(root, '404.html'), 'w', encoding='utf-8') as f:
    f.write(f'''<!DOCTYPE html><html lang="en"><head><meta charset="utf-8"><title>Not found · fpack</title>
<meta name="viewport" content="width=device-width, initial-scale=1"><link rel="icon" type="image/svg+xml" href="{base}images/logo.svg">
<style>body{{font-family:system-ui,sans-serif;display:grid;place-items:center;min-height:90vh;margin:0;color:#334155}}
@media(prefers-color-scheme:dark){{body{{background:#09090b;color:#e4e4e7}}}}a{{color:#0ea5e9}}</style></head>
<body><div style="text-align:center"><img src="{base}images/logo.svg" width="64" alt=""><h1>404</h1>
<p>This page does not exist. · 页面不存在。</p><p><a href="{base}">fpack docs</a> · <a href="{base}zh/">中文文档</a></p></div></body></html>
''')
open(os.path.join(root, '.nojekyll'), 'w').close()
print(f'postbuild: {len(pages)} pages, {len(index)} search entries, base {base}')
