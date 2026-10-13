---
title: "故障排查"
description: "如何阅读失败信息、常见问题与退出码。"
---

构建失败时，fpack 会给出关键错误摘录（Gradle 的 “What went wrong”、Xcode 错误等）、一条具体的修复建议，以及 `build/fpack/logs/` 中完整日志的路径。加 `-v` 可实时显示完整的工具输出，`--dry-run` 可查看每条命令而不执行。

## 常见问题

| 提示 | 解决 |
| --- | --- |
| `找不到 Flutter SDK` | 安装 Flutter，或使用 `--flutter <SDK路径>` / 设置 `FPACK_FLUTTER`；FVM 项目会被自动识别 |
| `未找到 Java（需要 JDK 17+）` | 安装 Android Studio（自带 JDK）或 JDK 17/21，然后执行 `flutter config --jdk-dir <路径>` |
| `找不到 flavor "x"` | 使用列出的 flavor 之一；`fpack doctor` 会显示检测到的 flavor |
| `定义了 productFlavors，但没有指定 flavor` | 使用 `--flavor` 或设置 `build.flavor` |
| 密钥库密码错误 / 别名不存在 | 检查 `FPACK_ANDROID_KEYSTORE_PASSWORD` / `FPACK_ANDROID_KEY_ALIAS`——fpack 会在构建前用 keytool 校验 |
| 钥匙串中找不到签名证书 | 使用列出的 Developer ID 证书之一，或用 `--no-sign` 构建未签名包 |
| 公证结果 `Invalid` | fpack 会下载 Apple 的日志（`notary-log-<文件>.json`）并汇总问题；通常是嵌套代码未签名或缺少 Hardened Runtime |
| 公证上传超时 | fpack 会自动重试两次；仍失败时请检查网络后重新构建——此时什么都没有提交 |
| `已存在（实际运行会停止，除非 --force）` | 提升版本号、使用 `--build-number N` 或加 `--force` |
| `需要 macOS/Windows/Linux` | Flutter 无法交叉编译桌面/iOS 应用；请在对应系统的机器上构建 |
| `找不到 Inno Setup（ISCC.exe）` | 安装 Inno Setup 6（`winget install --id JRSoftware.InnoSetup -e`），或设置 `windows.inno_setup.iscc` |
| 缺少 `dpkg-deb` / `rpmbuild` / `appimagetool` / `snapcraft` | 安装它们（见 [Linux](/platforms/linux)）——fpack 会一次列出所有缺少的工具 |
| `在 … 及其上级目录中找不到 Flutter 项目`（在仓库根目录时还会列出下层的 Flutter 应用） | 进入其中一个目录，或用 `fpack -C <目录>` 指定 |
| 已签名并公证的 macOS App 无法启动（`RBSRequestErrorDomain Code=5`、POSIX 163、“Launch failed”） | entitlements 使用了只被 Xcode 开发描述文件授权的受限键（推送、iCloud 等）：配置 Developer ID 的 `macos.sign.provisioning_profile`，见 [描述文件](/platforms/macos#描述文件)；`--smoke-test` 可在公证前发现此问题 |
| `使用了受限 entitlements … 必须配合描述文件` / `嵌入了 Xcode 的开发描述文件` / `是开发/Ad Hoc 描述文件` / `未授权 entitlement` | 见 [macOS 页面](/platforms/macos#描述文件) 中的校验信息表 |
| macOS 日志：`Unsatisfied entitlements` / `No matching profile found`（amfid、taskgated-helper） | 描述文件与签名不匹配：证书、团队、App ID 或 entitlement 值不对——`fpack doctor` 会指出具体哪一项 |
| `启动 … 后 App 已不在运行`（启动测试） | 运行提示中的 `log show` 命令查看原因；通常是 entitlements 未授权或启动时崩溃 |
| `要构建哪些目标？` | 指定目标（`fpack build apk`）、使用 `--all`，或在 fpack.yaml 中设置 `build.targets` |

## 退出码

<!-- BEGIN GENERATED EXIT -->
| 退出码 | 含义 |
| --- | --- |
| 0 | 成功（`--dry-run`：计划成功生成） |
| 1 | 至少一个目标在构建/打包时失败，或钩子失败；`fpack doctor`：发现问题；`fpack notarize`：有提交被拒绝或查询失败 |
| 2 | 用法或配置错误（未知参数或目标、未指定目标且没有 `build.targets`、fpack.yaml 有误、找不到 Flutter 项目） |
| 3 | 前置条件不满足 / 产物已存在 / 没有可构建的目标 / 找不到 Flutter SDK；`fpack notarize`：找不到公证记录 |
| 130 | 被 Ctrl-C 中断（公证等待被中断时，Apple 端仍会继续处理） |
<!-- END GENERATED EXIT -->

`--json` 时 stdout 只输出 JSON 结果（目标、状态、产物路径、大小、SHA-256/512、公证状态 `notarization.state`、耗时、错误摘录、修复建议），所有人类可读输出走 stderr。
