---
title: "环境变量"
description: "所有 FPACK_* 变量以及 fpack 读取的其他变量。"
lang: zh-CN
---

完整说明（类型、默认值、影响的目标）见 [doc/configuration.md](/zh/environment)。

| 变量 | 作用 |
| --- | --- |
| `FPACK_FLUTTER` | Flutter SDK 路径 |
| `FPACK_CONFIG` | 配置文件路径 |
| `FPACK_MODE` `FPACK_FLAVOR` `FPACK_ENTRY` `FPACK_BUILD_NAME` `FPACK_BUILD_NUMBER` | 构建参数 |
| `FPACK_OUTPUT_DIR` `FPACK_OVERWRITE` `FPACK_SPLIT_PER_ABI` `FPACK_OBFUSCATE` | 输出 / Android / 混淆 |
| `FPACK_ANDROID_KEYSTORE` `FPACK_ANDROID_KEYSTORE_BASE64` `FPACK_ANDROID_KEYSTORE_PASSWORD` `FPACK_ANDROID_KEY_ALIAS` `FPACK_ANDROID_KEY_PASSWORD` | Android 签名 |
| `FPACK_IOS_EXPORT_METHOD` `FPACK_IOS_EXPORT_OPTIONS_PLIST` `FPACK_IOS_CODESIGN` | iOS 导出 |
| `FPACK_MACOS_SIGN` `FPACK_MACOS_SIGN_IDENTITY` `FPACK_MACOS_INSTALLER_IDENTITY` `FPACK_MACOS_NOTARIZE` `FPACK_MACOS_NOTARY_PROFILE` `FPACK_NOTARIZE_WAIT` `FPACK_DMG_TOOL` | macOS |
| `FPACK_NOTARY_APPLE_ID` `FPACK_NOTARY_TEAM_ID` `FPACK_NOTARY_PASSWORD` `FPACK_NOTARY_API_KEY` `FPACK_NOTARY_API_KEY_ID` `FPACK_NOTARY_API_ISSUER` | macOS 公证凭证（Apple ID / API 密钥） |
| `FPACK_WINDOWS_CERTIFICATE` `FPACK_WINDOWS_CERTIFICATE_PASSWORD` `FPACK_WINDOWS_CERT_THUMBPRINT` | Windows 代码签名 |
| `FPACK_LANG` | `zh` / `en`（默认依次读取 `LC_ALL`、`LC_MESSAGES`、`LANG`、macOS/Windows 系统语言） |
| `NO_COLOR` / `FPACK_NO_COLOR` / `FORCE_COLOR` | 颜色控制 |
| `FPACK_CORE` | 指定原生核心二进制（开发用） |
| `FPACK_HOME` | 核心缓存目录 |
| `FPACK_REBUILD=1` | 强制先用本机 Go 重新编译核心（修改了 Go 源码时） |
| `FPACK_GO` | 回退编译时使用的 Go（`none` 表示禁止本机编译） |
| `FPACK_NO_DOWNLOAD=1` / `FPACK_DOWNLOAD_URL` | 禁止下载（直接本机编译） / 自定义下载地址（内网镜像）；见[启动器如何找到原生核心](/zh/installation#launcher) |

## 输出语言

fpack（Go 核心和 Dart 启动器）在操作系统语言为中文（任何 `zh` 变体：简体、繁体、香港等）时使用中文，其余情况一律使用英文。所有内容都随之切换：提示信息、帮助、`doctor` / `list` 输出、错误提示、`fpack init` 生成的 `fpack.yaml` 注释，以及 `NOTARIZATION.md`。

优先级（先匹配者生效）：

1. `--lang zh|en`
2. `FPACK_LANG=zh|en`
3. 操作系统语言：
   - **macOS**：系统设置 → 通用 → 语言与地区 中的第一个语言（`defaults read -g AppleLanguages`），其次 `AppleLocale`。终端通常按“地区”设置 `LANG`（即使界面是中文也常为 `en_US.UTF-8`），所以只有读取失败时才使用 `LANG` / `LANGUAGE`；显式设置的 `LC_ALL` 或 `LC_MESSAGES` 仍然优先。结果缓存在 `~/Library/Caches/fpack/os-language`，系统偏好设置变化后自动刷新。
   - **Windows**：用户的显示语言（`GetUserDefaultUILanguage`，不启动子进程）；忽略 Git Bash/MSYS 设置的 `LANG`，显式的 `LC_ALL` / `LC_MESSAGES` 仍然优先。
   - **Linux 等**：依次读取 `LC_ALL`、`LC_MESSAGES`、`LANGUAGE`（第一项）、`LANG`；跳过 `C` / `POSIX`。

## 对应 fpack.yaml 键的变量

由键注册表生成。它们覆盖 `fpack.yaml`，又会被命令行参数覆盖。`fpack.yaml` 中的任何值也都可以用 `${VAR}` / `${VAR:-默认值}` 引用任意环境变量。

<!-- BEGIN GENERATED ENV -->
| 变量 | 对应 fpack.yaml | 类型 | 说明 |
| --- | --- | --- | --- |
| `FPACK_FLUTTER` | `flutter.sdk` | path | 使用的 Flutter SDK 根目录。 |
| `FPACK_MODE` | `build.mode` | `release` \\| `profile` \\| `debug` | 构建模式。 |
| `FPACK_FLAVOR` | `build.flavor` | string | flavor / Xcode scheme（--flavor）；文件名中的 {flavor}。 |
| `FPACK_ENTRY` | `build.target` | path | 入口文件（flutter -t）。 |
| `FPACK_BUILD_NAME` | `build.build_name` | string/number | 版本名（{version}）。 |
| `FPACK_BUILD_NUMBER` | `build.build_number` | string/number | 构建号（{build}）。 |
| `FPACK_OBFUSCATE` | `build.obfuscate` | bool | 混淆 Dart 代码；符号文件保存到 split_debug_info（默认 <输出目录>/debug-info/<平台>）。 |
| `FPACK_OUTPUT_DIR` | `output.dir` | string | 产物目录（相对于项目，可用占位符）。 |
| `FPACK_OVERWRITE` | `output.overwrite` | bool | 已存在同名产物时覆盖，而不是停止。 |
| `FPACK_SPLIT_PER_ABI` | `android.split_per_abi` | `false` \\| `true` \\| `both` | false：一个通用 APK；true：每个 ABI 一个 APK；both：两者都要。 |
| `FPACK_ANDROID_KEYSTORE` | `android.signing.store_file` | path | keystore 文件（.jks/.keystore）。CI 中可用 FPACK_ANDROID_KEYSTORE_BASE64 提供。 |
| `FPACK_ANDROID_KEYSTORE_PASSWORD` | `android.signing.store_password` | string | keystore 密码。 |
| `FPACK_ANDROID_KEY_ALIAS` | `android.signing.key_alias` | string | key 别名。 |
| `FPACK_ANDROID_KEY_PASSWORD` | `android.signing.key_password` | string | key 密码。 |
| `FPACK_IOS_EXPORT_METHOD` | `ios.export_method` | `app-store-connect` \\| `app-store` \\| `release-testing` \\| `ad-hoc` \\| `development` \\| `debugging` \\| `enterprise` | IPA 导出方式。 |
| `FPACK_IOS_EXPORT_OPTIONS_PLIST` | `ios.export_options_plist` | path | 自己的 ExportOptions.plist；优先于下面所有生成选项。 |
| `FPACK_IOS_CODESIGN` | `ios.codesign` | bool | false：构建未签名 IPA（Payload/ 结构），用于之后重签名。 |
| `FPACK_MACOS_SIGN` | `macos.sign.enabled` | bool | 用 Developer ID 重新签名 .app（并签名 DMG）。为 true 但未设置 identity 时自动选用第一个 “Developer ID Application” 证书。false（--no-sign）同时关闭 pkg 签名与公证。 |
| `FPACK_MACOS_SIGN_IDENTITY` | `macos.sign.identity` | string | App 的 codesign 证书；设置后即启用签名。 |
| `FPACK_MACOS_NOTARIZE` | `macos.sign.notarize` | bool | 公证并装订 zip、DMG 与已签名的 pkg（会上传到 Apple）。 |
| `FPACK_MACOS_NOTARY_PROFILE` | `macos.sign.notary_profile` | string | `xcrun notarytool store-credentials <名字>` 创建的钥匙串配置名（本地推荐）。 |
| `FPACK_NOTARY_APPLE_ID` | `macos.sign.notary_apple_id` | string | 公证用 Apple ID（配合 notary_team_id + notary_password），替代钥匙串配置。 |
| `FPACK_NOTARY_TEAM_ID` | `macos.sign.notary_team_id` | string | Apple ID 公证时的团队 ID。 |
| `FPACK_NOTARY_PASSWORD` | `macos.sign.notary_password` | string | Apple ID 公证用的 App 专用密码。 |
| `FPACK_NOTARY_API_KEY` | `macos.sign.notary_api_key` | path | 公证用 App Store Connect API 密钥（.p8，适合 CI）。 |
| `FPACK_NOTARY_API_KEY_ID` | `macos.sign.notary_api_key_id` | string | API 密钥 ID。 |
| `FPACK_NOTARY_API_ISSUER` | `macos.sign.notary_api_issuer` | string | API Issuer UUID（个人密钥可省略）。 |
| `FPACK_MACOS_INSTALLER_IDENTITY` | `macos.sign.installer_identity` | string | 签名 .pkg 的证书，与 App 的 “Developer ID Application” 不同。不设置则 pkg 不签名。 |
| `FPACK_NOTARIZE_WAIT` | `macos.notarize.wait` | bool | true：一直等待 Apple 返回结果（显示已等待时间；按 Ctrl-C 只是停止等待，Apple 端会继续处理）。false：提交后写入 NOTARIZATION.md / notarization.json 并结束，状态为 “submitted”；之后运行 `fpack notarize finish` 装订。 |
| `FPACK_DMG_TOOL` | `macos.dmg.tool` | `auto` \\| `hdiutil` \\| `create-dmg` | auto：装了 create-dmg（或设置了布局键）时用 create-dmg，否则用 hdiutil。 |
| `FPACK_WINDOWS_CERTIFICATE` | `windows.sign.certificate` | path | 代码签名证书（.pfx）。设置它（或 thumbprint）后会签名应用 .exe、安装程序和 MSIX。 |
| `FPACK_WINDOWS_CERTIFICATE_PASSWORD` | `windows.sign.password` | string | 证书密码。 |
| `FPACK_WINDOWS_CERT_THUMBPRINT` | `windows.sign.thumbprint` | string | Windows 证书存储中证书的 SHA-1 指纹（替代 .pfx）。 |
<!-- END GENERATED ENV -->
