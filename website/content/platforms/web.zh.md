---
title: "Web"
description: "Web 构建、base href、WebAssembly 与部署。"
---

`fpack build web` 执行 `flutter build web`，并把 `build/web` 打包为 `-web.zip`，可直接部署到任意静态托管。任何操作系统都能构建 Web。

```bash
fpack build web
fpack build web --base-href /app/     # 部署在子路径下
fpack build web --wasm                # WebAssembly 构建
```

## 设置

| 设置 | fpack.yaml | 参数 |
| --- | --- | --- |
| Base href | `web.base_href` | `--base-href` |
| WebAssembly | `web.wasm` | `--wasm` |
| Source map | `web.source_maps` | – |
| 兼容内容安全策略（CSP）的输出 | `web.csp` | – |
| dart2js 优化级别（0–4） | `web.optimization_level` | – |
| 静态资源 CDN | `web.static_assets_url` | – |
| 从 Google CDN 加载 CanvasKit | `web.web_resources_cdn` | – |
| `--web-define` 值 | `web.web_define` | – |

Web 构建不使用 flavor（Flutter 只在 Android、iOS、macOS 上支持）；请改用 `--dart-define` / `build.dart_define_from_file`。

## 部署

把产物解压到服务器即可，也可以在 `post_package.web` 钩子中部署，例如用 `peaceiris/actions-gh-pages` 部署到 GitHub Pages，或用 `rsync` 部署到任意服务器。
