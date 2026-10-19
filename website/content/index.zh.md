---
title: "fpack"
description: "一条命令，把 Flutter 项目打包成所有平台的已签名发布文件。"
---

<div class="hero-badges">

[![pub package](https://img.shields.io/pub/v/fpack.svg)](https://pub.dev/packages/fpack) [![CI](https://github.com/Matkurban/fpack/actions/workflows/ci.yml/badge.svg)](https://github.com/Matkurban/fpack/actions/workflows/ci.yml) [![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](https://github.com/Matkurban/fpack/blob/main/LICENSE)

</div>

`flutter build` 给你的是一个构建目录，**fpack 给你的是可以直接发布的文件**：已签名的 APK 和 AAB、IPA、已公证的 DMG、Windows 安装程序、`.deb` / `.rpm` / AppImage / Snap 包和 Web 压缩包。文件名统一、带校验和，全部收集在一个 `dist/` 目录里。

```bash
dart pub global activate fpack
cd my_app
fpack doctor              # 这台机器能构建什么？缺什么、怎么装？
fpack build apk aab       # → dist/1.0.0+1/my_app-1.0.0+1-android-universal.apk、… .aab
fpack build --all         # 构建这台机器能构建的全部目标，其余的会注明原因并跳过
```

## 从这里开始

第一次使用 fpack？按顺序读下面四页，大约二十分钟，最后你会得到一个真实的产物。

::: cards
- [**1 · fpack 如何工作**](/concepts) 目标、构建流程、文件放在哪里、fpack 绝不会动什么
- [**2 · 安装**](/installation) 一条命令，以及找不到 `fpack` 时怎么办
- [**3 · 第一次构建**](/getting-started) `doctor`、预演、真正构建、看懂结果
- [**4 · 选择你的平台**](#平台教程) 每个平台一份完整教程
:::

## 平台教程

每份教程都从前置条件一路走到可以发布、并且验证过的已签名文件。

::: cards
- [**Android**](/platforms/android) 密钥库、已签名的 APK / AAB、按 ABI 拆分、Google Play
- [**iOS**](/platforms/ios) 团队、导出方式、Ad Hoc、未签名、App Store Connect
- [**macOS**](/platforms/macos) zip / DMG / pkg、Developer ID 签名、公证
- [**Windows**](/platforms/windows) 便携 zip、Inno Setup 安装程序、MSIX、代码签名
- [**Linux**](/platforms/linux) tar.gz、deb、rpm、AppImage、Snap（含 LXD / Multipass 构建）
- [**Web**](/platforms/web) 可部署的 zip、base href、WebAssembly
:::

## 为你的项目配置

fpack 不配置也能用。需要更多控制时，指南按“你想做什么”来组织：

::: cards
- [**名称、版本与输出**](/guides/project-and-output) 应用元数据、版本覆盖、文件名、校验和
- [**Flavor、入口与编译变量**](/guides/build-variants) `--flavor`、`-t`、`--dart-define`、混淆
- [**密钥与环境变量**](/guides/secrets) 让密码不进文件；参数、变量、配置文件的优先级
- [**钩子**](/guides/hooks) 在构建或打包前后运行你自己的脚本
- [**多目标一条命令**](/guides/many-targets) `--all`、预演、JSON、monorepo
:::

## 查阅

::: cards
- [**命令**](/commands) 每个命令和选项
- [**fpack.yaml 参考**](/configuration) 每个键的类型、默认值和示例
- [**环境变量**](/environment) 每个 `FPACK_*` 变量
- [**目标与产物**](/targets) 名称、别名、产物、所需工具
- [**退出码与 JSON 输出**](/exit-codes) 给脚本和 CI 用
- [**JSON Schema**](/schema) 编辑器里 `fpack.yaml` 的补全
- [**CI 示例**](/ci) 各平台的 GitHub Actions
- [**故障排查**](/troubleshooting) 按症状查找
:::

## 为什么选择 fpack

- **零配置。** 读取 `pubspec.yaml`、Gradle 和 Xcode 工程，`fpack.yaml` 可选。
- **绝不修改你的项目。** 签名信息通过环境变量和临时文件注入；产物放进 `dist/`，临时文件放进 `build/fpack/`。
- **可以放心试。** `fpack doctor` 告诉你缺什么、怎么装；`--dry-run` 打印它将运行的每一条命令。
- **内置签名与公证。** Android 密钥库（v1–v4）、iOS 导出选项、macOS Developer ID 与可恢复的公证、Windows Authenticode（pfx、证书库、Azure Trusted Signing）。
- **一切皆可配置。** 216 个经过校验的 `fpack.yaml` 键、用于编辑器补全的 JSON Schema、`${VAR}` 插值和专用的 `FPACK_*` 变量。
- **为 CI 而生。** 稳定的退出码、`--json` 结果、日志中的密钥脱敏、中英文输出。
- **快且无依赖。** 原生核心（启动约 10 毫秒）加一个轻量的 Dart 启动器，后者会下载版本匹配、经 SHA-256 校验的二进制。

完整的示例项目和带注释的 `fpack.yaml` 在 [`example/`](https://github.com/Matkurban/fpack/tree/main/example)。
