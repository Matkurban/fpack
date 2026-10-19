---
title: "常见问题"
description: "常见问题的简短解答。"
---

## 关于 fpack

**fpack 会修改我的项目吗？** 不会。fpack 只读取项目文件。`flutter_launcher_icons.yaml`、`flutter_native_splash.yaml`、`distribute_options.yaml`、`package_rename_config.yaml`、`pubspec.yaml` 以及 Gradle/Xcode 文件都不会被修改。它创建的一切都在 `dist/` 和 `build/fpack/` 里。建议把 `dist/` 加入 `.gitignore`。

**一定要有 `fpack.yaml` 吗？** 不需要。零配置就能用。需要元数据、签名或自定义名称时，`fpack init` 可以帮忙（[你的第一次构建](/getting-started#6-创建-fpackyaml需要时)）。

**它与 `flutter build` 有什么不同？** `flutter build` 只生成构建目录。fpack 运行它，然后签名、打包（DMG、pkg、安装程序、deb、rpm、AppImage、Snap、zip）、公证、校验、命名、生成校验和并收集文件，一条命令搞定任意多个目标。

**能在 Linux 上构建 iOS 或 Windows 吗？** 不能：Flutter 不能跨系统编译。请用对应系统的 CI runner（[CI 示例](/ci)）。

**fpack 会把我的代码或密钥发到别处吗？** 不会。它运行的是本地工具。公证会像 `notarytool` 一样把产物上传给 Apple；启动器从 GitHub Release 下载它的核心。

## 使用

**怎么读失败的构建？** fpack 会显示关键错误摘录、一条具体的修复办法和完整日志路径 `build/fpack/logs/…`。加 `-v` 可输出完整信息（[故障排查](/troubleshooting)）。

**怎么只构建一个变体？** 写出目标（`fpack build apk`），项目有 flavor 的话再加 `--flavor`（[Flavor](/guides/build-variants#flavor)）。

**版本在哪里设置？** 在 `pubspec.yaml` 里（`version: 1.2.0+5`），或每次运行用 `--build-name` / `--build-number`（[版本](/guides/project-and-output#版本)）。

**为什么 fpack 拒绝构建（“already exists”）？** 它默认从不覆盖产物。提高版本号、用 `--build-number`，或 `--force`。

**怎么让密码不进 `fpack.yaml`？** 用 `${VAR}` 引用和 `FPACK_*` 变量（[密钥与环境变量](/guides/secrets)）。

**怎么在 CI 里用？** 见 [CI 示例](/ci)。用 `--json` 和退出码做决定。

**怎么切换输出语言？** `--lang zh|en` 或 `FPACK_LANG`（[环境变量](/environment#输出语言)）。

## macOS

**启动 fpack 时 macOS 提示无法验证开发者**：fpack 安装核心时会移除隔离属性；如果仍被拦截，运行 `FPACK_REBUILD=1 fpack --version`，用本机 Go 编译。

**需要付费的 Apple 账号吗？** Developer ID 签名和公证需要（[证书与凭证](/platforms/macos-credentials)）。只在自己的 Mac 上用的未签名构建则什么都不需要。

## 安装

**本地路径安装时，每次运行都以 “Resolving dependencies…” 开头**：这是 pub 对路径激活的包的固定行为（git 和 pub.dev 安装不会）。如果其他程序要读取 `--json` 输出，请从 pub.dev 或 git 安装，或过滤这些行。

**缺少 path 依赖**（例如 `my_sdk: path: ../my_sdk`）：fpack 构建前会检查并指出缺失的包；把它放在相对项目的期望位置。
