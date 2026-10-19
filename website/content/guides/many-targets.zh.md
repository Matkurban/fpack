---
title: "多目标一条命令"
description: "构建这台机器能构建的一切、预览计划、获取 JSON 结果、在 monorepo 中工作并清理。"
---

## 选择目标

```bash
fpack build apk aab ipa          # 写出名字
fpack build --all                # 这台机器能构建的每个目标
fpack build                      # build.targets 里的目标
```

```yaml
build:
  targets: [apk, aab, ipa, dmg]
```

没有目标也没有 `build.targets` 时，fpack 以退出码 2 停止并询问要构建哪些目标。别名在命令行和 `build.targets` 里都可以用：

| 别名 | 目标 |
| --- | --- |
| `bundle`、`appbundle` | `aab` |
| `android` | `apk` |
| `ios` | `ipa` |
| `mac`、`osx`、`app` | `macos` |
| `win`、`zip`、`portable` | `windows` |
| `setup`、`installer`、`inno` | `exe` |
| `tar`、`tgz`、`tar.gz` | `linux` |
| `debian` | `deb` |
| `fedora` | `rpm` |
| `image`、`AppImage` | `appimage` |
| `snapcraft` | `snap` |

## `--all` 会跳过什么

`--all` 构建这台机器*能*构建的目标，其余的跳过并给出原因（例如“needs macOS”）。需要缺失的可选工具的目标（没有 appimagetool 的 `appimage`、没有 snapcraft 的 `snap`）也会被跳过。明确指定目标则更严格：指定一个本机无法构建的目标是错误（退出码 3），并指向对应系统的 CI 机器。

## 共享构建

每次运行中，每个平台只做一次 `flutter build`：`macos` + `dmg` + `pkg` 只构建一次，`linux` + `deb` + `rpm` + `appimage` + `snap` 只构建一次，`windows` + `exe` + `msix` 只构建一次。同一平台要多种格式，几乎不增加时间。

## 预演与查看

```bash
fpack build --all --dry-run     # 打印计划和每条命令，什么都不运行
fpack list                      # 每个目标：平台、扩展名、这台机器能否构建
fpack doctor apk ipa dmg        # 只检查这几个目标的前置条件
```

## 机器可读的结果

```bash
fpack build --all --json > result.json
```

使用 `--json` 时，stdout 只输出 JSON 结果（目标、状态、产物路径、大小、哈希、公证状态、耗时、错误摘要和修复建议）；人类可读的输出走 stderr。字段和示例：[退出码与 JSON 输出](/exit-codes#json-zh)。

## 中断

一次 Ctrl-C 会优雅地停止正在运行的 flutter、Gradle 或 xcodebuild 进程组；第二次则强制结束。退出码是 130。被中断的公证等待会在 Apple 那边继续处理，用 `fpack notarize finish` 完成它。

## Monorepo 与其他目录

fpack 从当前目录向上找到 `pubspec.yaml`。在含有多个 Flutter 项目的仓库根目录，它会列出这些项目；用 `-C` 选择其一：

```bash
fpack -C apps/client build web --base-href /app/
fpack --config ci/fpack.ci.yaml build apk      # 另一个配置文件
```

（`--project 目录` / `-C`、`--config 文件`、`FPACK_CONFIG`。）

## 详细输出、颜色与语言

| 选项 | 作用 |
| --- | --- |
| `-v`、`--verbose` | 输出每个工具的完整输出 |
| `--no-color` | 不使用颜色（也可用 `NO_COLOR`、`FPACK_NO_COLOR`；`FORCE_COLOR` 强制使用颜色） |
| `--lang zh\|en` | 输出语言（也可用 `FPACK_LANG`；默认取你的系统语言） |
| `-y`、`--yes` | 默认回答“是”/使用默认值（例如 `fpack init`、`fpack clean`） |

没有 TTY 时，进度动画和颜色会关闭，fpack 每 60 秒输出一行心跳，避免 CI 因长时间无输出而超时。

## 清理

```bash
fpack clean                     # 删除 build/fpack（日志、临时文件）
fpack clean --dist              # 同时删除输出目录（会先询问）
fpack clean --flutter-clean     # 同时运行 `flutter clean`
fpack clean --all --dry-run     # 显示将删除什么
```

## 下一步

每个命令和选项的[参考](/commands)，以及 [CI 示例](/ci)。
