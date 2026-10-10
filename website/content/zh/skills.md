---
title: "AI 智能体技能"
description: "fpack-cli 包技能教 AI 编程助手正确使用 fpack。"
---

fpack 在 pub 包中附带一个名为 **`fpack-cli`** 的[包技能（package skill）](https://dart.dev/tools/pub/package-skills)（`skills/fpack-cli/SKILL.md`）。它为 AI 编程助手（Cursor、Claude Code、Gemini、Copilot、Cline、Codex 等）提供简洁、权威的 fpack 使用说明：安装、`fpack init`、`doctor` 与 `list`、所有构建目标、配置键与 `FPACK_*` 环境变量、Android / macOS / Windows 签名、macOS 公证、CI 和故障排查，并链接回本站的详细文档。

## 在项目中安装技能

技能从项目的依赖中安装，因此先把 fpack 添加为开发依赖（不会影响你的应用），再运行 [`skills`](https://pub.dev/packages/skills) 工具：

```sh
dart pub add dev:fpack
dart run skills@ get --package fpack          # 交互式选择你的 AI 助手
dart run skills@ get --package fpack --agent cursor   # 或直接指定：claude、codex、copilot、cline……
```

技能会被复制到项目的智能体目录（例如 `.agents/skills/fpack-cli/`）；如果团队使用同一种助手，可以把它提交到仓库。升级 fpack 后重新运行该命令即可：技能随包的版本一起更新。

fpack 本身仍然通过全局安装运行（`dart pub global activate fpack`）。

## 助手会怎么用它

当你让助手“构建签名的 APK 和 AAB”“公证 DMG”或“添加发布工作流”时，技能会让它：

- 使用 `fpack build <目标>`（先加 `--dry-run`），而不是手写 `flutter build` 和打包脚本；
- 密钥只放在环境变量里，绝不写进 `fpack.yaml`；
- 只使用真实存在的配置键（`fpack init`、`fpack schema`、[配置](/zh/configuration)）；
- 在 CI 中使用 `--json` 和退出码，缺少依赖时运行 `fpack doctor`。

不安装也可以直接把这个文件交给助手：[skills/fpack-cli/SKILL.md](https://github.com/Matkurban/fpack/blob/main/skills/fpack-cli/SKILL.md)。
