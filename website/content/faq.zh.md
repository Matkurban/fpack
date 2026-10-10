---
title: "常见问题"
description: "常见问题解答。"
---

**构建失败了怎么看？** fpack 会显示关键错误摘录（Gradle 的 “What went wrong”、Xcode 的 error 行等）、一条具体的修复建议，以及完整日志路径 `build/fpack/logs/…`。加 `-v` 可实时查看完整输出。

**path 依赖找不到**（例如 `xue_hua_sdk: path: ../xue_hua_sdk`）：fpack 会在构建前检查并指出缺少哪个包，请把它放到相对项目的对应位置。

**macOS 提示无法验证开发者**：fpack 安装核心二进制时会移除 quarantine 属性；如果仍被拦截，运行 `FPACK_REBUILD=1 fpack --version` 用本机 Go 重新编译。

**用本地路径安装时，每次运行开头都有 “Resolving dependencies…”**：这是 pub 对路径安装包的固定行为（git / pub.dev 安装没有）。如果要把 `--json` 输出交给其他程序，请用 git 或 pub.dev 安装，或过滤掉这几行。

**会不会改我的项目？** 不会。fpack 只读取项目文件；`flutter_launcher_icons.yaml`、`flutter_native_splash.yaml`、`distribute_options.yaml`、`package_rename_config.yaml`、`pubspec.yaml` 等都不会被修改。建议把 `dist/` 加入 `.gitignore`。
