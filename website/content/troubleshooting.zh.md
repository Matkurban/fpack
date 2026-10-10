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
| 找不到 `ISCC.exe` | 安装 Inno Setup 6，或设置 `windows.inno_setup.iscc` |
| 缺少 `dpkg-deb` / `rpmbuild` / `appimagetool` | 安装它们（见 [Linux](/platforms/linux)）——fpack 会一次列出所有缺少的工具 |
| 在仓库根目录：“找到了 Flutter 项目 …” | 用 `fpack -C <目录>` 选择其中一个 |

## 退出码

| 码 | 含义 |
| --- | --- |
| 0 | 成功（`--dry-run`：计划成功生成） |
| 1 | 至少一个目标在构建/打包时失败，或钩子失败；`fpack notarize`：有提交被拒绝或查询失败 |
| 2 | 用法或配置错误（未知参数、未知目标、fpack.yaml 有误） |
| 3 | 前置条件不满足 / 产物已存在 / 没有可构建的目标 / 找不到 Flutter |
| 130 | 被 Ctrl-C 中断（公证等待被中断时 Apple 端仍会继续处理） |

`--json` 时 stdout 只输出 JSON 结果（目标、状态、产物路径、大小、SHA-256/512、公证状态 `notarization.state`、耗时、错误摘录、修复建议），所有人类可读输出走 stderr。
