---
title: "Web"
description: "Build a deployable web zip: base href, WebAssembly, CDN options and deployment."
---

`fpack build web` runs `flutter build web` and zips `build/web` into `my_app-1.0.0+1-web.zip`, ready for any static host. Web builds work on every operating system and need only Flutter.

## 1. Build

```bash
fpack build web
```

## 2. Serve from a sub-path

When the app is not at the root of the domain (for example `https://example.com/app/`), set the base href. It must start and end with `/`:

```bash
fpack build web --base-href /app/
```

In `fpack.yaml`: `web.base_href`.

## 3. WebAssembly

```bash
fpack build web --wasm            # compile to WebAssembly (with JavaScript fallback)
```

In `fpack.yaml`: `web.wasm: true`.

## 4. More options

Everything else lives in `fpack.yaml` under `web`:

```yaml
web:
  base_href: /app/
  source_maps: true               # generate source maps
  csp: true                       # no dynamic code generation (Content-Security-Policy)
  optimization_level: 2           # dart2js / dart2wasm -O0 … -O4 (default 4)
  static_assets_url: https://cdn.example.com/app/    # serve assets from another domain (must end with /)
  web_resources_cdn: false        # bundle CanvasKit instead of loading it from the CDN
  web_define: { API_URL: https://api.example.com }   # --web-define values for web/index.html
  extra_args: [--dump-info]
```

Pass other flutter arguments after `--`: `fpack build web --base-href /app/ -- --no-web-resources-cdn`.

::: note
Flavors are not used for web builds (Flutter supports them on Android, iOS and macOS only). Use `--dart-define` or `build.dart_define_from_file` ([Flavors, entry points and defines](/guides/build-variants)).
:::

## 5. Deploy

Unzip the artifact on your host, or deploy it from a [hook](/guides/hooks), for example `post_package.web` calling `rsync`, or in CI with `peaceiris/actions-gh-pages` for GitHub Pages:

```yaml
hooks:
  post_package:
    web: ['rsync -a --delete build/web/ user@host:/var/www/app/']
```

## In CI

See [CI recipes](/ci#linux-and-web).
