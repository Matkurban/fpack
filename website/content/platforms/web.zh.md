---
title: "Web"
description: "构建可部署的 Web zip：base href、WebAssembly、CDN 选项与部署。"
---

`fpack build web` 运行 `flutter build web`，并把 `build/web` 压缩成 `my_app-1.0.0+1-web.zip`，可直接部署到任何静态主机。Web 构建在所有操作系统上都能用，只需要 Flutter。

## 1. 构建

```bash
fpack build web
```

## 2. 部署在子路径

当应用不在域名根路径（例如 `https://example.com/app/`）时，设置 base href。它必须以 `/` 开头和结尾：

```bash
fpack build web --base-href /app/
```

在 `fpack.yaml` 里：`web.base_href`。

## 3. WebAssembly

```bash
fpack build web --wasm            # 编译为 WebAssembly（带 JavaScript 回退）
```

在 `fpack.yaml` 里：`web.wasm: true`。

## 4. 更多选项

其余设置都在 `fpack.yaml` 的 `web` 下：

```yaml
web:
  base_href: /app/
  source_maps: true               # 生成 source map
  csp: true                       # 不做动态代码生成（Content-Security-Policy）
  optimization_level: 2           # dart2js / dart2wasm 的 -O0 … -O4（默认 4）
  static_assets_url: https://cdn.example.com/app/    # 从另一个域名提供静态资源（必须以 / 结尾）
  web_resources_cdn: false        # 把 CanvasKit 打进包里，而不是从 CDN 加载
  web_define: { API_URL: https://api.example.com }   # 给 web/index.html 的 --web-define 值
  extra_args: [--dump-info]
```

其他 flutter 参数写在 `--` 之后：`fpack build web --base-href /app/ -- --no-web-resources-cdn`。

::: note
Web 构建不使用 flavor（Flutter 只在 Android、iOS 和 macOS 上支持）。请使用 `--dart-define` 或 `build.dart_define_from_file`（[Flavor、入口与编译变量](/guides/build-variants)）。
:::

## 5. 部署

把产物解压到你的主机上，或者用[钩子](/guides/hooks)部署，例如让 `post_package.web` 调用 `rsync`，或在 CI 里用 `peaceiris/actions-gh-pages` 发布到 GitHub Pages：

```yaml
hooks:
  post_package:
    web: ['rsync -a --delete build/web/ user@host:/var/www/app/']
```

## 在 CI 中

见 [CI 示例](/ci#linux-与-web)。
