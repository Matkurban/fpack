---
title: "Flavor、入口与编译变量"
description: "构建变体：flavor、入口文件、编译期变量、构建模式、混淆和传给 Flutter 的额外参数。"
---

本指南涵盖所有改变*编译什么*的设置：flavor、入口文件、`--dart-define` 值、构建模式、混淆，以及直接传给 Flutter 的参数。如果你只构建一个普通变体，可以跳过。

## Flavor

flavor 是应用的一个命名变体。在 Android 上它是 Gradle 的 `productFlavor`；在 iOS 和 macOS 上是 Xcode scheme。只有项目定义了 flavor 才需要这个参数。

```bash
fpack build apk aab --flavor prod
```

在 `fpack.yaml` 里是 `build.flavor`，变量是 `FPACK_FLAVOR`。

- fpack 从 `android/app/build.gradle(.kts)` 和 Xcode scheme（共享、用户级和 workspace）读取 flavor。
- 名称不在检测到的 flavor 里会报错（退出码 3）并给出建议；如果没有检测到任何 flavor，则只是警告。
- 项目定义了 flavor 却没有选择时，fpack 会在完整的 Gradle 构建失败*之前*停下。
- 产物名里带有 flavor（`{-flavor}`、`{flavor}`）。
- 只存在于 Android 的 flavor，在 Xcode 工程没有自定义 scheme 时不会传给 iOS 或 macOS 构建（并给出提示）。
- Web 构建不使用 flavor。

## 入口文件

```bash
fpack build apk -t lib/main_prod.dart       # --target 文件
```

`fpack.yaml` 里是 `build.target`，变量是 `FPACK_ENTRY`。可以与 flavor 组合：`fpack build apk --flavor prod -t lib/main_prod.dart`。

## 编译期变量

用 `--dart-define`（可重复）或文件把值传进应用：

```bash
fpack build apk --dart-define API_URL=https://api.example.com --dart-define DEBUG_MENU=false
fpack build apk --dart-define-from-file config/prod.json
fpack build apk --dart-define-from-file config/base.json --dart-define-from-file config/prod.env
```

```yaml
build:
  dart_define:
    API_URL: https://api.example.com
  dart_define_from_file: [config/prod.json]     # .json 或 .env 文件
```

Web 目标还有 `web.web_define`，用于 `web/index.html` 里的值（[Web](/platforms/web)）。

::: warning
`--dart-define` 的值会被编译进应用，可以被提取出来。不要把真正的密钥放进去。
:::

## 构建模式

```bash
fpack build apk --release     # 默认；等同于 -m release / --mode release
fpack build apk --profile     # profile 模式，用于性能测试
fpack build apk --debug       # debug 构建
```

`build.mode`、`FPACK_MODE`。非 release 模式会在文件名里加上 `{mode}`（`{-mode}`）。

## 混淆与调试符号

```bash
fpack build apk aab --obfuscate
fpack build apk --obfuscate --split-debug-info build/symbols
```

`--obfuscate` 会隐藏 Dart 符号名。fpack 默认把符号文件存到 `<输出目录>/debug-info/<平台>`；`--split-debug-info 目录` 可以选别的目录。**请保管好符号文件**：读取混淆后发布版的崩溃堆栈时需要它们。设置：`build.obfuscate`、`build.split_debug_info`、`FPACK_OBFUSCATE`。

## 图标与其他 flutter 选项

```yaml
build:
  tree_shake_icons: false          # 等同于 --no-tree-shake-icons
  extra_args: [--no-pub]           # 追加到每次 flutter build
```

命令行里 `--` 之后的参数同样会传给 `flutter build`：

```bash
fpack build apk -- --no-tree-shake-icons
```

各平台专用的额外参数在各自的段下：`android.extra_args`、`ios.extra_args`、`macos.extra_args`、`windows.extra_args`、`linux.extra_args`、`web.extra_args`（以及 `windows.msix.extra_args`、`linux.appimage.extra_args`、`linux.snap.extra_args`）。

## 用哪个 Flutter？

fpack 按这个顺序查找 Flutter SDK：`--flutter` → `FPACK_FLUTTER` → `fpack.yaml` 的 `flutter.sdk` → FVM（`.fvm/flutter_sdk`、`.fvmrc`）→ `FLUTTER_ROOT` → `PATH` → 常见安装位置（`~/develop/flutter`、`~/flutter` 等）。

```bash
fpack build apk --flutter ~/sdks/flutter-3.24
```

## 下一步

[密钥与环境变量](/guides/secrets)。
