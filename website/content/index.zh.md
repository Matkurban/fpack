---
title: "fpack"
description: "一条命令，把 Flutter 项目打包成所有平台的发布文件。"
---

<div class="hero-badges">

[![pub package](https://img.shields.io/pub/v/fpack.svg)](https://pub.dev/packages/fpack) [![CI](https://github.com/Matkurban/fpack/actions/workflows/ci.yml/badge.svg)](https://github.com/Matkurban/fpack/actions/workflows/ci.yml) [![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](https://github.com/Matkurban/fpack/blob/main/LICENSE)

</div>

**Android**（APK / 按 ABI 拆分 APK / AAB）· **iOS**（IPA）· **macOS**（.app zip / DMG / pkg，可签名 + 公证）· **Windows**（zip / Inno Setup 安装包 / MSIX）· **Linux**（tar.gz / deb / rpm / AppImage）· **Web**（zip）

```bash
dart pub global activate fpack
cd my_flutter_app
fpack doctor              # 这台机器能打哪些包？缺什么？怎么装？
fpack build apk aab       # → dist/1.0.0+1/my_app-1.0.0+1-android-universal.apk …
fpack build --all         # 本机能打的全部打出来，打不了的说明原因并跳过
```

## 为什么选择 fpack

- **零配置可用**——直接读取 `pubspec.yaml`、Gradle 和 Xcode 工程；`fpack.yaml` 完全可选。
- **绝不修改你的项目**——签名信息通过环境变量注入，不修改 Gradle 和 Xcode 文件。产物写到 `dist/`，临时文件写到 `build/fpack/`。
- **原生核心**——Go 编写的二进制（启动约 10 ms，无运行时依赖），由轻量的 Dart 启动器下载版本一致、经 SHA-256 校验的核心。
- **一切可配置**——188 个经过校验的 `fpack.yaml` 键，提供 JSON Schema 供编辑器补全，支持 `${VAR}` 插值和专门的 `FPACK_*` 变量。
- **为人和 CI 设计**——清晰的错误摘录与修复建议、`--dry-run` 展示每条命令、`--json` 机器可读输出、中英文输出。
- **签名与公证**——Android 密钥库、iOS 导出选项、macOS Developer ID + 公证（可稍后继续等待）、Windows Authenticode。

## 接下来

| | |
| --- | --- |
| 🚀 [快速开始](/getting-started) | 安装 fpack 并打出第一批产物 |
| ⌨️ [命令](/commands) | 所有命令与参数 |
| ⚙️ [配置](/configuration) | `fpack.yaml` 的全部键 |
| 📦 [目标](/targets) | 每个目标产出什么、需要什么 |
| 🔐 [macOS 签名与公证](/platforms/macos) | Developer ID、notarytool、CI 凭证 |
| 🤖 [CI 示例](/ci) | 各平台的 GitHub Actions |
| 🧰 [故障排查](/troubleshooting) | 常见问题与退出码 |

带注释 `fpack.yaml` 的完整示例项目位于 [`example/`](https://github.com/Matkurban/fpack/tree/main/example)。
