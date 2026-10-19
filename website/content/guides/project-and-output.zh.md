---
title: "名称、版本与输出"
description: "应用元数据、版本覆盖、产物放在哪里、如何命名，以及校验和。"
---

本指南的所有内容都是可选的：fpack 会从你的项目读取合理的默认值。当安装程序需要正式的名称、CI 必须设置版本，或者你想换输出目录和文件命名方案时再用它。

## 应用元数据

安装程序和包需要可读的名称、发布者等信息。fpack 从你的项目取得它们，并允许你在 `app` 段覆盖：

```yaml
app:
  display_name: My App                   # 安装程序标题、开始菜单、.desktop 的 Name=
  description: A fast and simple notes app
  publisher: Example Inc                 # Windows 安装程序发布者、msix 发布者显示名称、deb Maintainer 的后备值、rpm Vendor
  identifier: com.example.app            # Inno Setup AppId 种子、msix 标识名称、pkg identifier 的后备值
  homepage: https://example.com
  support_url: https://example.com/support
  maintainer: Example Team <dev@example.com>   # deb Maintainer、rpm Packager
  copyright: © 2026 Example Inc
  license: MIT                           # rpm / deb 的 SPDX 标识
```

| 键 | 用途 | 默认值 |
| --- | --- | --- |
| `app.name` | 文件名中的 `{app}` 部分 | pubspec 的 `name` |
| `app.display_name` | 安装程序标题、开始菜单、`.desktop` 的 `Name=` | macOS 的 `PRODUCT_NAME`，否则是 pubspec 的 `name` |
| `app.description` | deb Description、rpm Summary、`.desktop` Comment、msix 描述 | pubspec 的 `description` |
| `app.publisher` | Windows 发布者、msix 发布者显示名称、deb Maintainer 后备值、rpm Vendor | `windows/runner/Runner.rc` 里的 `CompanyName` |
| `app.identifier` | Inno Setup AppId 种子、msix 标识、pkg identifier 后备值 | Linux 的 `APPLICATION_ID`、Android 的 `applicationId` 或 iOS 的 bundle id |
| `app.homepage`、`app.support_url` | 安装程序网址、deb Homepage、rpm URL | 支持网址默认取主页 |
| `app.maintainer` | deb Maintainer、rpm Packager | `app.publisher` |
| `app.copyright`、`app.license` | 版权行和许可证 | `© <年份> <发布者>`、`Proprietary` |

`fpack init` 会根据你的项目预填这些值。

## 版本

版本来自 `pubspec.yaml`：`version: 1.2.0+5` 表示版本名 `1.2.0`、构建号 `5`。单次运行可以覆盖其中任何一个，例如在 CI 里：

```bash
fpack build apk --build-name 1.2.0 --build-number 42
```

在 `fpack.yaml` 里是 `build.build_name` / `build.build_number`，变量是 `FPACK_BUILD_NAME` / `FPACK_BUILD_NUMBER`。版本会出现在 `dist/<版本>+<构建号>/` 和文件名里（`{version}` 和 `{build}`）。

```bash
fpack build apk --build-number "$GITHUB_RUN_NUMBER"
```

## 产物放在哪里

默认是相对于项目的 `dist/{version}{+build}`。用 `-o`（`--output`）修改：

```bash
fpack build apk -o release/android
```

或者在 `fpack.yaml` 里（`output.dir`，变量 `FPACK_OUTPUT_DIR`）。`output.dir` 支持占位符 `{app}`、`{version}`、`{build}`、`{flavor}` 和 `{mode}`。

```yaml
output:
  dir: "dist/{app}/{version}"
```

## 文件名

每个产物都由模板 `output.name` 命名：

```text
{app}{-flavor}-{version}{+build}-{platform}{-arch}{-variant}{-mode}
→ my_app-1.0.0+1-android-universal.apk
```

可以整体覆盖（`output.name`），也可以按目标覆盖（`output.names`）：

```yaml
output:
  name: "{app}-{version}-{platform}{-arch}"
  names:
    exe: "{app}-setup-{version}"
    web: "{app}-web"
```

| 占位符 | 值 |
| --- | --- |
| `{app}` | `app.name`（默认：pubspec 的 `name`） |
| `{version}` / `{build}` | 版本名 / 构建号 |
| `{flavor}` | flavor（Flutter 不使用的地方为空） |
| `{platform}` | `android`、`ios`、`macos`、`windows`、`linux`、`web` |
| `{arch}` | `arm64-v8a`、`universal`、`x64` 等 |
| `{variant}` | `setup`、`portable` 等 |
| `{mode}` | release 时为空，否则是 `profile` 或 `debug` |
| `{target}` | 目标名 |
| `{date}` | 构建日期 `YYYYMMDD` |

`{-x}` 和 `{+x}` 只在值不为空时才加上分隔符（`-` 或 `+`），所以没有 flavor 的 release 构建不会留下多余的连字符。在 `output.dir` 里，按产物区分的占位符为空。

## 覆盖已有文件

fpack 默认从不覆盖已有产物：同一版本构建两次会停在 *already exists*。可以提高版本号、改构建号，或者允许覆盖：

```bash
fpack build apk --force
```

（`output.overwrite: true`、`FPACK_OVERWRITE`）。预演会显示哪些文件已经存在。

## 校验和

fpack 会在产物旁写入 `SHA256SUMS`。可以改用 SHA-512 或关闭：

```yaml
output:
  checksums: true               # false：不生成校验和文件
  checksum_algorithm: sha512    # 生成 SHA512SUMS
```

校验用 `shasum -a 256 -c SHA256SUMS`（macOS）、`sha256sum -c SHA256SUMS`（Linux）。用 `--json` 时同样的哈希也在结果里（[退出码与 JSON 输出](/exit-codes#json-zh)）。运行 `fpack notarize finish` 之后，这个文件会为已装订的产物重写。

## 下一步

[Flavor、入口与编译变量](/guides/build-variants)，或者在[参考](/configuration#app-zh)里查每一个键。
