---
title: "故障排查"
description: "按症状查找：原因与修复办法，涵盖安装、环境、Android、iOS、macOS、Windows、Linux。"
---

## 先读失败信息

构建失败时，fpack 会输出关键错误摘录（Gradle 的 “What went wrong”、Xcode 错误等）、**一条具体的修复办法**，以及 `build/fpack/logs/` 里完整日志的路径。三个工具每次都有用：

| 工具 | 用途 |
| --- | --- |
| `fpack doctor` | 检查每个前置条件，并给出对应的修复命令 |
| `fpack build … --dry-run` | 显示每条命令但不运行（密码显示为 `***`） |
| `-v` / `--verbose` | 输出完整的工具输出，而不是摘录 |

退出码的含义见[退出码与 JSON](/exit-codes)。

## 安装与启动

| 症状 | 原因与修复 |
| --- | --- |
| `command not found: fpack` | `~/.pub-cache/bin`（Windows 为 `%LOCALAPPDATA%\Pub\Cache\bin`）不在 `PATH` 里：[安装](/installation#2-加入-path) |
| 第一次运行提示回退到本机编译 | 核心下载失败；检查网络、`FPACK_DOWNLOAD_URL`，或使用内置/离线安装（[核心如何被找到](/installation#launcher-zh)） |
| macOS 提示无法验证开发者 | fpack 安装核心时会移除隔离属性；如果仍被拦截，运行 `FPACK_REBUILD=1 fpack --version`，用本机 Go 编译 |
| 本地路径安装每次都输出 “Resolving dependencies…” | pub 对路径激活的包的固定行为；改用 pub.dev 或 git 安装，或在处理 `--json` 时过滤这些行 |
| `no Flutter project found in … or its parents` | `cd` 到项目里，或用 `fpack -C <目录>`；在仓库根目录 fpack 会列出下面的 Flutter 应用 |
| `Flutter SDK not found` | 安装 Flutter，或用 `--flutter <sdk>` / `FPACK_FLUTTER`；FVM 项目会被自动检测（[用哪个 Flutter？](/guides/build-variants#用哪个-flutter)） |
| 缺少 path 依赖 | fpack 构建前会检查并指出包名；把它放在期望的相对路径 |
| `which targets?` | 写出目标（`fpack build apk`）、用 `--all`，或设置 `build.targets` |
| `invalid configuration: …` | fpack.yaml 有错误；信息会指出行号并提示可能的键。[参考](/configuration) |

## 输出文件

| 症状 | 原因与修复 |
| --- | --- |
| `already exists (a real run stops unless --force)` | 这个版本的产物已存在：提高版本号、`--build-number N`，或 `--force` |
| `needs macOS` / `needs Windows` / `needs Linux` | Flutter 不能跨系统编译；在对应系统的 runner 上构建（[概念](/concepts#目标与产物)） |
| 某个目标被 `--all` 跳过 | 它需要的工具缺失；`fpack doctor` 会告诉你是哪个 |

## Android

| 症状 | 原因与修复 |
| --- | --- |
| `no Java (JDK 17+) found` | 安装 Android Studio（自带 JDK）或 JDK 17/21，然后 `flutter config --jdk-dir <路径>` |
| `keystore password was incorrect` / `alias … does not exist` | 检查 `FPACK_ANDROID_KEYSTORE_PASSWORD` / `FPACK_ANDROID_KEY_ALIAS`；fpack 构建前会用 keytool 验证 |
| `… defines productFlavors … but no flavor is set` | 传 `--flavor` 或设置 `build.flavor` |
| `flavor "x" not found` | 使用列出的 flavor 之一；`fpack doctor` 会显示检测到的 flavor |
| release 构建用 DEBUG 密钥签名 | 配置签名（[Android 第 4 步](/platforms/android#4-告诉-fpack)）；Google Play 会拒绝调试签名的文件 |
| `store_file is set but missing store_password` | 设置 `FPACK_ANDROID_KEYSTORE_PASSWORD` 或 `android.signing.store_password` |

## iOS

| 症状 | 原因与修复 |
| --- | --- |
| 没有签名证书 / `No Accounts` | 在 Xcode 里登录，或导入证书和描述文件 |
| `No profiles for '…' were found` | 手动签名时，在 `ios.provisioning_profiles` 里列出 bundle id（以及扩展） |
| 信息里的团队不对 | 设置 `ios.team_id`；它优先于项目里的 `DEVELOPMENT_TEAM` |
| `pod install` 失败 | `cd ios && pod install` 查看错误；`pod repo update` |
| 上传时构建号重复 | 提高 `+N` 或使用 `--build-number` |

见 [iOS](/platforms/ios#出问题时)。

## macOS

| 症状 | 原因与修复 |
| --- | --- |
| `signing identity … not found in the keychain` | 按列出的名字原样使用某个 Developer ID 证书，或用 `--no-sign` 构建未签名版本 |
| 公证结果 `Invalid` | fpack 会下载 Apple 的日志（`notary-log-<文件>.json`）并总结问题；通常是嵌套代码未签名或缺少 Hardened Runtime |
| 公证时上传超时 | fpack 会重试两次上传；否则检查网络并重新构建（什么都没提交） |
| 公证时间很长，你按了 Ctrl-C | Apple 会继续处理：先 `fpack notarize status`，再 `fpack notarize finish` |
| 已签名的应用打不开（`RBSRequestErrorDomain Code=5`、POSIX 163、“Launch failed”） | 受限权限（推送、iCloud 等）需要 Developer ID 描述文件：[描述文件](/platforms/macos-profiles)；`--smoke-test` 能在公证前发现 |
| `uses restricted entitlements … only with a provisioning profile`、`embeds Xcode's development profile`、`is a development/ad hoc profile`、`is not authorized by` | 见[描述文件](/platforms/macos-profiles#3-fpack-检查什么)里的校验表 |
| macOS 日志：`Unsatisfied entitlements` / `No matching profile found` | 描述文件与签名不匹配（证书、团队、App ID 或权限值不对）；`fpack doctor` 会指出是哪一项 |
| `the app was not running … after launch`（冒烟测试） | 运行打印出来的 `log show` 命令；通常是未授权的权限或启动时崩溃 |
| pkg 未签名 | 设置 `macos.sign.installer_identity`（Developer ID *Installer* 证书） |

## Windows

| 症状 | 原因与修复 |
| --- | --- |
| `Inno Setup (ISCC.exe) not found` | `winget install --id JRSoftware.InnoSetup -e`，或设置 `windows.inno_setup.iscc` |
| `Unable to read file … app.dill` | 项目路径含有非 ASCII 字符：移动项目，或在 Windows 区域设置里启用 UTF-8 |
| 找不到 signtool | 安装 Windows SDK（`winget install Microsoft.WindowsSDK.10.0.26100`），或设置 `windows.sign.signtool` |
| 签名验证警告 | 自签名证书：只是警告；设置 `windows.sign.verify: false` 可消除 |

## Linux

| 症状 | 原因与修复 |
| --- | --- |
| `Linux desktop toolchain incomplete, missing: …` | `sudo apt-get install -y clang cmake ninja-build pkg-config libgtk-3-dev` |
| 缺少 `dpkg-deb` / `rpmbuild` / `appimagetool` / `snapcraft` | 安装它们（[Linux](/platforms/linux)）；fpack 会一次列出所有缺失的工具 |
| snap 需要 sudo | 先运行 `sudo -v`（或像 CI 上那样免密 sudo），或使用 `linux.snap.build_mode: lxd` |
| snap：LXD 未初始化 | `sudo lxd init --auto`，并把你的用户加入 `lxd` 组 |
| snap 与目标 CPU 不匹配 | 它包含的是在本机构建的应用：arm64 的 snap 请在 arm64 上构建 |

## 还是解决不了？

在 [github.com/Matkurban/fpack/issues](https://github.com/Matkurban/fpack/issues) 提交 issue，附上 `fpack doctor`、`fpack --version` 的输出以及错误里提到的日志文件（先去掉机密；fpack 已经对密码做了脱敏）。
