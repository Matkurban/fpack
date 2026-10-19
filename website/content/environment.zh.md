---
title: "环境变量"
description: "所有 FPACK_* 变量以及 fpack 读取的其他变量。"
---

环境变量在[优先级](/guides/secrets#哪个设置优先)中位于命令行参数和 `fpack.yaml` 之间：参数覆盖变量，变量覆盖文件。本页按用途列出 fpack 读取的每个变量。如何用它们保管密钥，见[密钥与环境变量](/guides/secrets)。

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
| `FPACK_OUTPUT_DIR` | `output.dir` | string | 产物目录（相对于项目）。占位符：{app} {version} {build} {flavor} {mode}。 |
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
| `FPACK_MACOS_PROVISIONING_PROFILE` | `macos.sign.provisioning_profile` | path | Developer ID 描述文件，签名前复制为 Contents/embedded.provisionprofile。entitlements 使用受限键（aps-environment、iCloud、associated domains、keychain access groups 等）时必需。 |
| `FPACK_MACOS_PROFILE_CHECK` | `macos.sign.profile_check` | bool | 签名前检查描述文件与受限 entitlements（Developer ID 类型、团队、App ID、有效期、已授权的 entitlements），并拒绝开发描述文件。false 跳过检查。 |
| `FPACK_NOTARIZE_WAIT` | `macos.notarize.wait` | bool | true：一直等待 Apple 返回结果（显示已等待时间；按 Ctrl-C 只是停止等待，Apple 端会继续处理）。false：提交后写入 NOTARIZATION.md / notarization.json 并结束，状态为 “submitted”；之后运行 `fpack notarize finish` 装订。 |
| `FPACK_MACOS_SMOKE_TEST` | `macos.smoke_test` | bool | 签名后启动 App，检查 smoke_test_seconds 秒后仍在运行，然后退出（可发现未授权 entitlements 等启动失败）。 |
| `FPACK_DMG_TOOL` | `macos.dmg.tool` | `auto` \\| `hdiutil` \\| `create-dmg` | auto：装了 create-dmg（或设置了布局键）时用 create-dmg，否则用 hdiutil。 |
| `FPACK_WINDOWS_CERTIFICATE` | `windows.sign.certificate` | path | 代码签名证书（.pfx）。设置它（或 thumbprint）后会签名应用 .exe、安装程序和 MSIX。 |
| `FPACK_WINDOWS_CERTIFICATE_PASSWORD` | `windows.sign.password` | string | 证书密码。 |
| `FPACK_WINDOWS_CERT_THUMBPRINT` | `windows.sign.thumbprint` | string | Windows 证书存储中证书的 SHA-1 指纹（/sha1），替代 .pfx。 |
| `FPACK_WINDOWS_CERT_SUBJECT` | `windows.sign.subject` | string | 证书存储中证书的主题名称（/n），替代指纹。 |
| `FPACK_WINDOWS_SIGN_DLIB` | `windows.sign.dlib` | path | 用于云签名的 signtool 插件（/dlib），如 Azure Trusted Signing；需同时设置 dlib_metadata。 |
| `FPACK_WINDOWS_SIGN_DLIB_METADATA` | `windows.sign.dlib_metadata` | path | dlib 的元数据文件（/dmdf），如 Trusted Signing 的账户和证书配置。 |
| `FPACK_SNAP_BUILD_MODE` | `linux.snap.build_mode` | `auto` \\| `destructive` \\| `lxd` \\| `multipass` | snapcraft 的构建方式：destructive（用 sudo 在本机构建，最快，适合 CI）、lxd 或 multipass（隔离的构建环境）。 |
<!-- END GENERATED ENV -->

## 按用途分类的变量

| 变量 | 作用 |
| --- | --- |
| `FPACK_FLUTTER` | Flutter SDK 路径 |
| `FPACK_CONFIG` | 配置文件路径 |
| `FPACK_MODE` `FPACK_FLAVOR` `FPACK_ENTRY` `FPACK_BUILD_NAME` `FPACK_BUILD_NUMBER` | 构建参数 |
| `FPACK_OUTPUT_DIR` `FPACK_OVERWRITE` `FPACK_SPLIT_PER_ABI` `FPACK_OBFUSCATE` | 输出 / Android / 混淆 |
| `FPACK_ANDROID_KEYSTORE` `FPACK_ANDROID_KEYSTORE_BASE64` `FPACK_ANDROID_KEYSTORE_PASSWORD` `FPACK_ANDROID_KEY_ALIAS` `FPACK_ANDROID_KEY_PASSWORD` | Android 签名 |
| `FPACK_IOS_EXPORT_METHOD` `FPACK_IOS_EXPORT_OPTIONS_PLIST` `FPACK_IOS_CODESIGN` | iOS 导出 |
| `FPACK_MACOS_SIGN` `FPACK_MACOS_SIGN_IDENTITY` `FPACK_MACOS_INSTALLER_IDENTITY` `FPACK_MACOS_NOTARIZE` `FPACK_MACOS_NOTARY_PROFILE` `FPACK_NOTARIZE_WAIT` `FPACK_MACOS_PROVISIONING_PROFILE` `FPACK_MACOS_PROFILE_CHECK` `FPACK_MACOS_SMOKE_TEST` `FPACK_DMG_TOOL` | macOS |
| `FPACK_NOTARY_APPLE_ID` `FPACK_NOTARY_TEAM_ID` `FPACK_NOTARY_PASSWORD` `FPACK_NOTARY_API_KEY` `FPACK_NOTARY_API_KEY_ID` `FPACK_NOTARY_API_ISSUER` | macOS 公证凭证（Apple ID / API 密钥） |
| `FPACK_WINDOWS_CERTIFICATE` `FPACK_WINDOWS_CERTIFICATE_PASSWORD` `FPACK_WINDOWS_CERT_THUMBPRINT` `FPACK_WINDOWS_CERT_SUBJECT` `FPACK_WINDOWS_SIGN_DLIB` `FPACK_WINDOWS_SIGN_DLIB_METADATA` | Windows 代码签名（.pfx、证书存储，或 Azure Trusted Signing 等 signtool dlib） |
| `FPACK_LANG` | `zh` / `en`（默认跟随系统语言，见下文「输出语言」） |
| `NO_COLOR` / `FPACK_NO_COLOR` / `FORCE_COLOR` | 颜色控制 |
| `FPACK_CORE` | 指定原生核心二进制（开发用） |
| `FPACK_HOME` | 核心缓存目录 |
| `FPACK_REBUILD=1` | 强制先用本机 Go 重新编译核心（修改了 Go 源码时） |
| `FPACK_GO` | 回退编译时使用的 Go（`none` 表示禁止本机编译） |
| `FPACK_NO_DOWNLOAD=1` / `FPACK_DOWNLOAD_URL` | 禁止下载（直接本机编译） / 自定义下载地址（内网镜像）；见[启动器如何找到原生核心](/installation#launcher-zh) |

## 输出语言

fpack（Go 核心和 Dart 启动器）在操作系统语言为中文（任何 `zh` 变体：简体、繁体、香港等）时使用中文，其余情况一律使用英文。所有内容都随之切换：提示信息、帮助、`doctor` / `list` 输出、错误提示、`fpack init` 生成的 `fpack.yaml` 注释，以及 `NOTARIZATION.md`。

优先级（先匹配者生效）：

1. `--lang zh|en`
2. `FPACK_LANG=zh|en`
3. 操作系统语言：
   - **macOS**：系统设置 → 通用 → 语言与地区 中的第一个语言（`defaults read -g AppleLanguages`），其次 `AppleLocale`。终端通常按“地区”设置 `LANG`（即使界面是中文也常为 `en_US.UTF-8`），所以只有读取失败时才使用 `LANG` / `LANGUAGE`；显式设置的 `LC_ALL` 或 `LC_MESSAGES` 仍然优先。结果缓存在 `~/Library/Caches/fpack/os-language`，系统偏好设置变化后自动刷新。
   - **Windows**：用户的显示语言（`GetUserDefaultUILanguage`，不启动子进程）；忽略 Git Bash/MSYS 设置的 `LANG`，显式的 `LC_ALL` / `LC_MESSAGES` 仍然优先。
   - **Linux 等**：依次读取 `LC_ALL`、`LC_MESSAGES`、`LANGUAGE`（第一项）、`LANG`；跳过 `C` / `POSIX`。

## fpack 读取的其他变量

| 变量 | 用途 |
| --- | --- |
| `FLUTTER_ROOT`、`FVM_CACHE_PATH` | 查找 Flutter SDK（在 `--flutter`、`FPACK_FLUTTER`、`flutter.sdk` 与项目的 FVM 链接之后） |
| `ANDROID_HOME` / `ANDROID_SDK_ROOT` | Android SDK 工具（`apksigner`），顺序与 Flutter 相同：先 `flutter config --android-sdk`，再这两个变量，最后默认安装目录 |
| `JAVA_HOME` | `keytool` / Java 版本检查：先 `flutter config --jdk-dir` 与 Android Studio 自带的 JDK，再 `JAVA_HOME`，最后 `PATH` |
| `XDG_CONFIG_HOME`、`APPDATA` | 读取 Flutter 设置文件的位置（`--android-sdk` / `--jdk-dir`） |
| `LOCALAPPDATA`、`ProgramFiles`、`ProgramFiles(x86)` | Windows：Android SDK、Inno Setup、FVM 与 Flutter 的默认安装位置 |
| `LC_ALL`、`LC_MESSAGES`、`LANGUAGE`、`LANG` | 输出语言（见上文） |
| `NO_COLOR`、`FPACK_NO_COLOR`、`TERM=dumb`、`FORCE_COLOR` | 颜色；`TERM=dumb` 同时关闭进度动画 |
| `COLUMNS` | 无法获取终端宽度时的表格宽度 |
| `XDG_CACHE_HOME` | 启动器：Linux 上的缓存根目录（可被 `FPACK_HOME` 覆盖） |

## fpack 为钩子设置的变量

fpack 把这些变量传给[钩子](/guides/hooks)；它们是输出，不是设置。

| 变量 | 值 |
| --- | --- |
| `FPACK_PROJECT_ROOT` | 项目目录 |
| `FPACK_OUTPUT_DIR` | 产物目录 |
| `FPACK_VERSION`、`FPACK_BUILD_NUMBER` | 版本名和构建号 |
| `FPACK_MODE`、`FPACK_FLAVOR` | 构建模式和 flavor |
| `FPACK_TARGET` | 目标（`pre_package` / `post_package`） |
| `FPACK_ARTIFACT`、`FPACK_ARTIFACTS` | 产物（`post_package`、`post_build`） |
| `FPACK_SUCCESS` | `1` 或 `0`（`post_build`） |
