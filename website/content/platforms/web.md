---
title: "Web"
description: "Web builds, base href, WebAssembly and deployment."
---

`fpack build web` runs `flutter build web` and zips `build/web` into `-web.zip`, ready to deploy to any static host. Web builds work on every host OS.

```bash
fpack build web
fpack build web --base-href /app/     # served from a sub-path
fpack build web --wasm                # WebAssembly build
```

## Settings

| Setting | fpack.yaml | Flag |
| --- | --- | --- |
| Base href | `web.base_href` | `--base-href` |
| WebAssembly | `web.wasm` | `--wasm` |
| Source maps | `web.source_maps` | – |
| Content Security Policy friendly output | `web.csp` | – |
| dart2js optimization level (0–4) | `web.optimization_level` | – |
| CDN for static assets | `web.static_assets_url` | – |
| Load CanvasKit from Google's CDN | `web.web_resources_cdn` | – |
| `--web-define` values | `web.web_define` | – |

Flavors are not used for web builds (Flutter supports them on Android, iOS and macOS only); use `--dart-define` / `build.dart_define_from_file` instead.

## Deploying

Unzip the artifact to your host, or deploy it from a `post_package.web` hook, e.g. to GitHub Pages with `peaceiris/actions-gh-pages` or to any server with `rsync`.
