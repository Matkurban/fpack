# fpack 配置参考

本文列出 fpack 1.0.0 的**全部**配置方式：`fpack.yaml` 的每个键、每个 `FPACK_*` 环境变量、每个命令行参数，以及它们的类型、默认值、影响的目标和作用。

- 所有配置都是可选的：不写 `fpack.yaml` 也能直接 `fpack build apk`。
- `fpack init` 会在项目根目录生成一份带注释的 `fpack.yaml`（这是 fpack 唯一会写入项目的文件）。
- fpack **不会修改**项目里的任何其他文件（`pubspec.yaml`、`key.properties`、Gradle、Xcode 工程等都只读）。

目录：

1. [优先级规则](#1-优先级规则)
2. [fpack.yaml 键参考](#2-fpackyaml-键参考)
3. [环境变量](#3-环境变量)
4. [命令行参数](#4-命令行参数)
5. [文件名模板](#5-文件名模板)
6. [完整示例：类似 xue_hua_im 的项目](#6-完整示例类似-xue_hua_im-的项目)

---

## 1. 优先级规则

同一个设置有多个来源时，**高优先级覆盖低优先级**：

```
命令行参数  >  FPACK_* 环境变量  >  fpack.yaml  >  项目自身配置  >  fpack 内置默认值
```

具体说明：

| 设置 | 来源（从低到高） |
| --- | --- |
| 一般构建参数（mode、flavor、输出目录…） | 内置默认值 → `fpack.yaml` → 环境变量 → 命令行 |
| Flutter SDK | 常见安装位置 → `PATH` → `FLUTTER_ROOT` → FVM（`.fvm/flutter_sdk`、`.fvmrc`）→ `flutter.sdk` → `FPACK_FLUTTER` → `--flutter` |
| 配置文件位置 | 项目根目录的 `fpack.yaml` / `fpack.yml` → `FPACK_CONFIG` → `--config` |
| 版本号 / 构建号 | `pubspec.yaml` 的 `version:` → `build.build_name` / `build.build_number` → `FPACK_BUILD_NAME` / `FPACK_BUILD_NUMBER` → `--build-name` / `--build-number` |
| Android release 签名 | 项目自己的 `android/key.properties` + `signingConfigs`（fpack 不配置签名时使用）→ `android.signing` → `FPACK_ANDROID_*` → （没有命令行参数，密码不应出现在命令行中） |
| macOS Developer ID 签名 / 公证 | `macos.sign` → `FPACK_MACOS_*` → `--sign/--no-sign`、`--sign-identity`、`--notarize/--no-notarize`、`--notary-profile` |
| 输出语言 | 系统语言（`LC_ALL` → `LC_MESSAGES` → `LANG` → macOS/Windows 系统设置）→ `FPACK_LANG` → `--lang` |

补充规则：

- **列表类参数是追加，不是替换**：`--dart-define`、`--dart-define-from-file` 会追加到 `fpack.yaml` 中已有的值之后。
- **`extra_args` 的拼接顺序**：`build.extra_args` → 平台的 `extra_args`（如 `android.extra_args`）→ 命令行 `--` 之后的参数。都会原样追加到 `flutter build …` 的末尾。
- **macOS 签名的联动规则**：设置了证书（`identity`）即视为启用签名；设置了公证配置名（`notary_profile`）即视为启用公证；关闭签名（`--no-sign` / `enabled: false`）会同时关闭公证；`enabled: false` 与 `notarize: true` 同时出现是配置错误。
- **macOS 签名只来自 fpack 自己的配置**：fpack 不读取 `pubspec.yaml` 中其他插件的配置（例如 [`dmg`](https://pub.dev/packages/dmg) 包的 `dmg:` 段）。什么都不配置时，.app 保留 Xcode 工程自己的签名，DMG 不签名。
- **`fpack.yaml` 中的环境变量引用**：任何字符串值都可以写 `${VAR}` 或 `${VAR:-默认值}`。未设置且没有默认值的变量会被替换为空字符串，并在运行时给出警告。
- **路径**：`fpack.yaml` 中的相对路径都相对于 **项目根目录**（`pubspec.yaml` 所在目录）；支持 `~/`。
- **未知键是错误**：拼错的键会报错并提示「你是不是想写 …」，不会被静默忽略。
- **布尔值**：`true/false`、`yes/no`、`on/off`、`1/0` 都可以（环境变量同样适用）。

---

## 2. fpack.yaml 键参考

表头说明：**类型**中 `list` 表示既可以写 YAML 列表，也可以写单个字符串；**目标**指受影响的 `fpack build` 目标，「全部」表示所有目标。

### 2.1 `app` — 应用信息（安装包元数据）

| 键 | 类型 | 默认值 | 目标 | 说明 |
| --- | --- | --- | --- | --- |
| `app.name` | string | `pubspec.yaml` 的 `name` | 全部 | 产物文件名中的 `{app}` 部分，例如 `xue_hua_im-1.0.0+1-android.aab`。 |
| `app.display_name` | string | `pubspec.yaml` 的 `name` | exe, deb, rpm, appimage, linux | 给人看的应用名：Windows 安装程序标题与开始菜单、Linux `.desktop` 文件的 `Name=`。 |
| `app.description` | string | `pubspec.yaml` 的 `description` | deb, rpm, appimage, linux | 软件包描述（deb `Description:`、rpm `Summary`、`.desktop` 的 `Comment=`）。 |
| `app.publisher` | string | 空 | exe, deb | Windows 安装程序的发布者；deb 未设置 `maintainer` 时用作维护者。 |
| `app.identifier` | string | 依次取 Linux `APPLICATION_ID`、Android `applicationId`、iOS Bundle ID，都没有则 `com.example.<name>` | exe | 反向域名标识。未设置 `windows.inno_setup.app_id` 时，用它生成稳定的安装程序 GUID（升级安装时会识别为同一个应用）。 |
| `app.homepage` | string | 空 | exe, deb | 主页 URL（Inno Setup `AppPublisherURL`、deb `Homepage:`）。 |
| `app.maintainer` | string | `publisher`，再没有则 `<包名> maintainers <noreply@example.com>` | deb | deb 的 `Maintainer:`，格式 `名字 <邮箱>`。 |

### 2.2 `flutter` — Flutter SDK

| 键 | 类型 | 默认值 | 目标 | 说明 |
| --- | --- | --- | --- | --- |
| `flutter.sdk` | path | 自动查找（见优先级规则） | 全部 | Flutter SDK 根目录（包含 `bin/flutter` 的目录）。一般不需要设置；项目使用 FVM 时会自动使用 FVM 的版本。 |

### 2.3 `build` — 所有 flutter build 共用的参数

| 键 | 类型 | 默认值 | 目标 | 说明 |
| --- | --- | --- | --- | --- |
| `build.targets` | list | 无 | — | 执行 `fpack build` 且不带目标时要构建的目标，例如 `[apk, aab, ipa, dmg]`。命令行给了目标或 `--all` 时忽略。 |
| `build.mode` | `release` \| `profile` \| `debug` | `release` | 全部 | 构建模式。非 release 时文件名会带上 `-profile` / `-debug`。 |
| `build.flavor` | string | 无 | apk, aab, ipa, macos, dmg（Flutter 支持 flavor 的平台） | 对应 `flutter build --flavor`：Android productFlavor / Xcode scheme。文件名中出现 `-<flavor>`。 |
| `build.target` | path | `lib/main.dart` | 全部 | 入口文件（`flutter build -t`），例如 `lib/main_prod.dart`。 |
| `build.dart_define` | map 或 `KEY=VALUE` 列表 | 无 | 全部 | 编译期常量（`--dart-define`）。写成 map：`{ API_URL: https://… }`，或列表：`[API_URL=https://…]`。 |
| `build.dart_define_from_file` | list | 无 | 全部 | JSON / `.env` 文件（`--dart-define-from-file`），可多个。 |
| `build.build_name` | string | `pubspec.yaml` 版本号的 `+` 之前部分 | 全部 | 覆盖版本名（`--build-name`），影响文件名中的 `{version}`。 |
| `build.build_number` | string/数字 | `pubspec.yaml` 版本号的 `+` 之后部分 | 全部 | 覆盖构建号（`--build-number`），影响文件名中的 `{build}`。 |
| `build.obfuscate` | bool | `false` | apk, aab, ipa, macos, dmg, windows, exe, msix, linux, deb, rpm, appimage | 混淆 Dart 代码（`--obfuscate`）。需要符号目录，未设置 `split_debug_info` 时自动使用 `<输出目录>/debug-info/<平台>`。web 不支持（会给出提示）。 |
| `build.split_debug_info` | path | `<输出目录>/debug-info` | 同上 | 调试符号保存位置；每个平台一个子目录。崩溃符号化需要它，请和产物一起归档。 |
| `build.extra_args` | list | 无 | 全部 | 原样追加到每一个 `flutter build` 命令末尾的参数。 |

### 2.4 `output` — 产物位置与命名

| 键 | 类型 | 默认值 | 目标 | 说明 |
| --- | --- | --- | --- | --- |
| `output.dir` | 模板 | `dist/{version}{+build}` | 全部 | 产物目录，支持[文件名模板](#5-文件名模板)中的占位符，例如 `dist/{version}`。相对于项目根目录。 |
| `output.name` | 模板 | `{app}{-flavor}-{version}{+build}-{platform}{-arch}{-variant}{-mode}` | 全部 | 产物文件名（不含扩展名）。 |
| `output.overwrite` | bool | `false` | 全部 | 产物已存在时是否覆盖。默认会跳过并提示（退出码 3），避免误覆盖已发布的文件。等同 `--force`。 |
| `output.checksums` | bool | `true` | 全部 | 在输出目录写入 `SHA256SUMS`（`sha256sum -c` 格式）。 |

### 2.5 `android`

| 键 | 类型 | 默认值 | 目标 | 说明 |
| --- | --- | --- | --- | --- |
| `android.split_per_abi` | `false` \| `true` \| `both` | `false` | apk | `false`：一个通用 APK；`true`：每个 ABI 一个 APK；`both`：两者都要（会执行两次 flutter build）。也接受 `universal`/`split`/`all` 等写法。 |
| `android.abis` | list | `[armeabi-v7a, arm64-v8a, x86_64]` | apk, aab | 要包含的 ABI（`--target-platform`），可选值：`armeabi-v7a`、`arm64-v8a`、`x86_64`。 |
| `android.signing.store_file` | path | 无 | apk, aab | release keystore（.jks / .keystore）。**设置后 fpack 会注入签名**；不设置则使用项目自己的 `key.properties` 配置（或 Flutter 默认的 debug 签名，此时会警告）。 |
| `android.signing.store_password` | string | 无 | apk, aab | keystore 密码。请用 `${FPACK_ANDROID_KEYSTORE_PASSWORD}` 引用环境变量，不要把密码写进文件。设置了 `store_file` 时必填。 |
| `android.signing.key_alias` | string | 无 | apk, aab | key 别名。设置了 `store_file` 时必填。 |
| `android.signing.key_password` | string | 同 `store_password` | apk, aab | key 密码。 |
| `android.extra_args` | list | 无 | apk, aab | 只追加到 Android 的 flutter build 命令。 |

签名注入方式：通过 Android Gradle 插件标准的 `android.injected.signing.*` 属性（`ORG_GRADLE_PROJECT_*` 环境变量）传入，不修改 Gradle 文件；密码不会出现在命令行、日志或 `--dry-run` 输出中。构建前用 `keytool` 校验密码和别名，构建后用 `apksigner` / `keytool -printcert` 校验产物并显示签名者。

### 2.6 `ios`

| 键 | 类型 | 默认值 | 目标 | 说明 |
| --- | --- | --- | --- | --- |
| `ios.export_method` | `app-store-connect` \| `app-store` \| `release-testing` \| `ad-hoc` \| `development` \| `debugging` \| `enterprise` | Flutter 默认（App Store Connect） | ipa | IPA 导出方式（`--export-method`）。`ad-hoc`/`release-testing` 用于测试设备分发，`development` 只需要开发证书。导出方式会写入产物类型说明。 |
| `ios.export_options_plist` | path | 无 | ipa | 自定义 `ExportOptions.plist`（`--export-options-plist`），**优先于** `export_method`。 |
| `ios.codesign` | bool | `true` | ipa | `false` 时构建未签名 IPA（`--no-codesign`，产物名带 `-unsigned`），用于之后重签名。 |
| `ios.extra_args` | list | 无 | ipa | 只追加到 `flutter build ipa`。 |

### 2.7 `macos`

| 键 | 类型 | 默认值 | 目标 | 说明 |
| --- | --- | --- | --- | --- |
| `macos.sign.enabled` | bool | 设置了 `identity` 时为 `true`，否则 `false` | macos, dmg | 是否用 Developer ID 重新签名 .app（Hardened Runtime + entitlements）和 DMG。为 `true` 但没有 `identity` 时，自动使用钥匙串中第一个 “Developer ID Application” 证书。为 `false`（或 `--no-sign`）时 .app 保留 Xcode 工程自己的签名，DMG 不签名，同时关闭公证。 |
| `macos.sign.identity` | string | 无 | macos, dmg | codesign 证书全名，例如 `Developer ID Application: Your Name (TEAMID)`。**设置后即启用签名**。证书不存在、已吊销或钥匙串中没有 Developer ID 时，构建前就会报错并说明如何导入 .p12。`fpack init` 会在注释中列出本机钥匙串里的 Developer ID 证书。 |
| `macos.sign.entitlements` | path | `macos/Runner/Release.entitlements`（非 release 模式用 `DebugProfile.entitlements`） | macos, dmg | 重新签名 .app 时使用的 entitlements 文件。 |
| `macos.sign.notarize` | bool | 设置了 `notary_profile` 时为 `true`，否则 `false` | macos, dmg | 是否提交 Apple 公证（`xcrun notarytool submit --wait`）并装订（`stapler staple`），zip 与 DMG 都会公证。会把文件上传到 Apple。需要签名；本地测试可用 `--no-notarize`。 |
| `macos.sign.notary_profile` | string | 无（开启公证但未设置时为 `NotaryProfile`） | macos, dmg | `xcrun notarytool store-credentials <名字>` 创建的钥匙串配置名。**设置后即启用公证**（前提是已启用签名）。 |
| `macos.dmg.tool` | `auto` \| `hdiutil` \| `create-dmg` | `auto` | dmg | 制作 DMG 的工具。`auto`：装了 [create-dmg](https://github.com/create-dmg/create-dmg) 就用它（窗口布局更好看），否则用系统自带的 `hdiutil`。 |
| `macos.dmg.volume_name` | string | .app 名称（如 `XueHua`） | dmg | 挂载 DMG 后显示的卷名。 |
| `macos.dmg.background` | path | 无 | dmg | DMG 窗口背景图，仅 `create-dmg` 支持。 |
| `macos.extra_args` | list | 无 | macos, dmg | 只追加到 `flutter build macos`。 |

fpack 不读取 `pubspec.yaml` 中 [`dmg`](https://pub.dev/packages/dmg) 包的 `dmg:` 段；需要签名/公证时请在 `fpack.yaml` 的 `macos.sign` 中配置（或使用 `FPACK_MACOS_*` 环境变量、命令行参数）。

### 2.8 `windows`

| 键 | 类型 | 默认值 | 目标 | 说明 |
| --- | --- | --- | --- | --- |
| `windows.inno_setup.app_id` | string（GUID） | 由 `app.identifier` 计算出的稳定 GUID | exe | Inno Setup `AppId`。发布后不要更改，否则用户升级时会被当成另一个应用。 |
| `windows.inno_setup.script` | path | 无（使用 fpack 内置脚本） | exe | 自定义 `.iss` 脚本。fpack 会以 `/D` 传入 `AppName`、`AppVersion`、`AppPublisher`、`AppExeName`、`SourceDir`、`AppId`、`AppURL` 等定义。 |
| `windows.inno_setup.iscc` | path | 自动查找 `ISCC.exe`（PATH 与默认安装位置） | exe | Inno Setup 编译器路径。 |
| `windows.msix.extra_args` | list | 无 | msix | 追加到 `dart run msix:create` 的参数。msix 目标需要项目把 [`msix`](https://pub.dev/packages/msix) 加为 dev 依赖；其余 MSIX 设置写在 pubspec 的 `msix_config:` 中。 |
| `windows.extra_args` | list | 无 | windows, exe, msix | 只追加到 `flutter build windows`。 |

### 2.9 `linux`

| 键 | 类型 | 默认值 | 目标 | 说明 |
| --- | --- | --- | --- | --- |
| `linux.package_name` | string | 应用名转小写、`_` 换成 `-`（如 `xue-hua-im`） | deb, rpm, appimage, linux | 软件包名，也用于安装路径和 `.desktop` 文件名。 |
| `linux.icon` | path（PNG） | `flutter_launcher_icons` 配置的图标 → `web/icons/Icon-512.png` → `Icon-192.png` | deb, rpm, appimage, linux | 菜单 / AppImage 图标。AppImage 没有图标时会报错。 |
| `linux.categories` | string | `Utility;` | deb, rpm, appimage, linux | freedesktop 分类，例如 `Network;Chat;`。 |
| `linux.deb.depends` | list | 无 | deb | deb 的 `Depends:`，例如 `["libgtk-3-0 \| libgtk-3-0t64"]`。 |
| `linux.rpm.requires` | list | 无 | rpm | rpm 的 `Requires:`，例如 `[gtk3]`。 |
| `linux.appimagetool` | path | 自动查找 `appimagetool` | appimage | appimagetool 路径。`--all` 时找不到会跳过 appimage。 |
| `linux.extra_args` | list | 无 | linux, deb, rpm, appimage | 只追加到 `flutter build linux`。 |

### 2.10 `web`

| 键 | 类型 | 默认值 | 目标 | 说明 |
| --- | --- | --- | --- | --- |
| `web.base_href` | string | Flutter 默认（`/`） | web | 部署在子路径时设置，例如 `/app/`（必须以 `/` 开头和结尾）。 |
| `web.wasm` | bool | `false` | web | 用 WebAssembly 构建（`--wasm`）。 |
| `web.extra_args` | list | 无 | web | 只追加到 `flutter build web`。 |

---

## 3. 环境变量

### 3.1 构建设置（覆盖 fpack.yaml，被命令行覆盖）

| 变量 | 类型 | 对应 fpack.yaml | 目标 | 说明 |
| --- | --- | --- | --- | --- |
| `FPACK_FLUTTER` | path | `flutter.sdk` | 全部 | Flutter SDK 根目录。 |
| `FPACK_CONFIG` | path | — | 全部 | 配置文件路径（代替项目根目录的 `fpack.yaml`）。 |
| `FPACK_MODE` | `release`\|`profile`\|`debug` | `build.mode` | 全部 | 构建模式。 |
| `FPACK_FLAVOR` | string | `build.flavor` | 支持 flavor 的目标 | flavor。 |
| `FPACK_ENTRY` | path | `build.target` | 全部 | 入口文件。 |
| `FPACK_BUILD_NAME` | string | `build.build_name` | 全部 | 版本名。 |
| `FPACK_BUILD_NUMBER` | string | `build.build_number` | 全部 | 构建号（CI 中常用 `${{ github.run_number }}`）。 |
| `FPACK_OUTPUT_DIR` | 模板 | `output.dir` | 全部 | 产物目录。 |
| `FPACK_OVERWRITE` | bool | `output.overwrite` | 全部 | 覆盖已存在的产物。 |
| `FPACK_OBFUSCATE` | bool | `build.obfuscate` | 原生目标 | 混淆 Dart 代码。 |
| `FPACK_SPLIT_PER_ABI` | `false`\|`true`\|`both` | `android.split_per_abi` | apk | APK 拆分方式。 |
| `FPACK_ANDROID_KEYSTORE` | path | `android.signing.store_file` | apk, aab | keystore 路径。 |
| `FPACK_ANDROID_KEYSTORE_BASE64` | base64 | — | apk, aab | keystore 文件内容的 base64（适合 CI secrets）。仅在没有设置 keystore 路径时使用；fpack 会写入 `build/fpack/secrets/`（权限 0600），构建结束后删除。 |
| `FPACK_ANDROID_KEYSTORE_PASSWORD` | string | `android.signing.store_password` | apk, aab | keystore 密码。 |
| `FPACK_ANDROID_KEY_ALIAS` | string | `android.signing.key_alias` | apk, aab | key 别名。 |
| `FPACK_ANDROID_KEY_PASSWORD` | string | `android.signing.key_password` | apk, aab | key 密码（默认同 keystore 密码）。 |
| `FPACK_IOS_EXPORT_METHOD` | string | `ios.export_method` | ipa | 导出方式。 |
| `FPACK_IOS_EXPORT_OPTIONS_PLIST` | path | `ios.export_options_plist` | ipa | ExportOptions.plist。 |
| `FPACK_IOS_CODESIGN` | bool | `ios.codesign` | ipa | `false` = 未签名 IPA。 |
| `FPACK_MACOS_SIGN` | bool | `macos.sign.enabled` | bool | 设置了 `identity` 时为 `true`，否则 `false` | macos, dmg | 是否用 Developer ID 重新签名 .app（Hardened Runtime + entitlements）和 DMG。为 `true` 但没有 `identity` 时，自动使用钥匙串中第一个 “Developer ID Application” 证书。为 `false`（或 `--no-sign`）时 .app 保留 Xcode 工程自己的签名，DMG 不签名，同时关闭公证。 |
| `FPACK_MACOS_SIGN_IDENTITY` | string | `macos.sign.identity` | string | 无 | macos, dmg | codesign 证书全名，例如 `Developer ID Application: Your Name (TEAMID)`。**设置后即启用签名**。证书不存在、已吊销或钥匙串中没有 Developer ID 时，构建前就会报错并说明如何导入 .p12。`fpack init` 会在注释中列出本机钥匙串里的 Developer ID 证书。 |
| `FPACK_MACOS_NOTARIZE` | bool | `macos.sign.notarize` | bool | 设置了 `notary_profile` 时为 `true`，否则 `false` | macos, dmg | 是否提交 Apple 公证（`xcrun notarytool submit --wait`）并装订（`stapler staple`），zip 与 DMG 都会公证。会把文件上传到 Apple。需要签名；本地测试可用 `--no-notarize`。 |
| `FPACK_MACOS_NOTARY_PROFILE` | string | `macos.sign.notary_profile` | string | 无（开启公证但未设置时为 `NotaryProfile`） | macos, dmg | `xcrun notarytool store-credentials <名字>` 创建的钥匙串配置名。**设置后即启用公证**（前提是已启用签名）。 |
| `FPACK_DMG_TOOL` | `auto`\|`hdiutil`\|`create-dmg` | `macos.dmg.tool` | dmg | DMG 工具。 |

### 3.2 输出与界面

| 变量 | 默认值 | 说明 |
| --- | --- | --- |
| `FPACK_LANG` | 根据系统语言 | 输出语言：`zh` 或 `en`。 |
| `NO_COLOR` / `FPACK_NO_COLOR` | 未设置 | 设置为任意值即关闭颜色。`TERM=dumb` 也会关闭颜色。 |
| `FORCE_COLOR` | 未设置 | 在非终端环境（如 CI 日志）中也输出颜色。 |

### 3.3 启动器（Dart 包装层，决定使用哪个原生核心）

| 变量 | 默认值 | 说明 |
| --- | --- | --- |
| `FPACK_CORE` | 未设置 | 直接指定 `fpack-core` 可执行文件（开发 fpack 本身时用）。 |
| `FPACK_HOME` | macOS `~/Library/Caches/fpack`；Linux `$XDG_CACHE_HOME/fpack` 或 `~/.cache/fpack`；Windows `%LOCALAPPDATA%\fpack` | 原生核心的缓存目录。 |
| `FPACK_REBUILD` | 未设置 | 设为 `1` 时强制用本机 Go 从包内源码重新编译核心。 |
| `FPACK_GO` | 自动查找（PATH、`/usr/local/go/bin`、`/opt/homebrew/bin` 等） | Go 可执行文件路径；设为 `none` 表示不使用 Go 编译。 |
| `FPACK_NO_DOWNLOAD` | 未设置 | 设为 `1` 时禁止从 GitHub Release 下载核心。 |
| `FPACK_DOWNLOAD_URL` | `https://github.com/Matkurban/fpack/releases/download/v<版本>/` | 核心下载地址（内网镜像）。目录下需要 `checksums.txt` 和 `fpack-core-<os>-<arch>[.exe]`，下载后会校验 SHA-256。 |

核心的查找顺序：`FPACK_CORE` → 缓存（版本与校验和匹配时）→ 包内预编译二进制（按 `prebuilt/manifest.json` 校验）→ 用 Go 编译包内源码 → 从 GitHub Release 下载并校验。

### 3.4 fpack 读取的其他变量

| 变量 | 用途 |
| --- | --- |
| `FLUTTER_ROOT` | 查找 Flutter SDK。 |
| `ANDROID_HOME` / `ANDROID_SDK_ROOT` | 查找 `apksigner` 等 Android SDK 工具。 |
| `JAVA_HOME` | 查找 `keytool`（以及 `flutter config --jdk-dir` 的设置）。 |

---

## 4. 命令行参数

### 4.1 全局参数（所有命令可用）

| 参数 | 类型 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `-h`, `--help` | 开关 | — | 显示帮助（`fpack help <命令>` 同理）。 |
| `-V`, `--version` | 开关 | — | 显示版本（同 `fpack version`）。 |
| `-C`, `--project DIR` | path | 当前目录或其上级中第一个含 `pubspec.yaml` 的目录 | Flutter 项目目录。 |
| `--config FILE` | path | 项目中的 `fpack.yaml` | 配置文件。 |
| `--flutter SDK` | path | 自动查找 | Flutter SDK 根目录。 |
| `--lang zh\|en` | string | 根据系统语言 | 输出语言。 |
| `-v`, `--verbose` | 开关 | 关 | 实时输出 flutter / 工具的完整日志。 |
| `--json` | 开关 | 关 | stdout 只输出机器可读的 JSON 结果，其他输出走 stderr。 |
| `--no-color` | 开关 | 关 | 关闭颜色。 |
| `-y`, `--yes` | 开关 | 关 | 所有提问自动使用默认值（`init`、`clean --dist`）。 |

仅启动器支持：`fpack --wrapper-info` 显示启动器使用的核心路径与来源（排查安装问题用）。

### 4.2 `fpack build [目标…] [参数] [-- flutter 参数]`

目标：`apk`、`aab`、`ipa`、`macos`、`dmg`、`windows`、`exe`、`msix`、`linux`、`deb`、`rpm`、`appimage`、`web`。

| 参数 | 类型 | 对应 fpack.yaml | 目标 | 说明 |
| --- | --- | --- | --- | --- |
| `-a`, `--all` | 开关 | — | — | 构建本机能构建的所有目标；其他目标会被跳过并说明原因。 |
| `-n`, `--dry-run` | 开关 | — | 全部 | 只显示计划和将执行的准确命令（与真实运行使用同一套代码），什么都不执行。 |
| `-m`, `--mode MODE` | string | `build.mode` | 全部 | `release`（默认）/ `profile` / `debug`。 |
| `--release` / `--profile` / `--debug` | 开关 | `build.mode` | 全部 | 等同 `--mode …`，只能选一个。 |
| `--flavor NAME` | string | `build.flavor` | 支持 flavor 的目标 | 构建 flavor。 |
| `-t`, `--target FILE` | path | `build.target` | 全部 | 入口文件。 |
| `--dart-define K=V` | 可重复 | `build.dart_define`（追加） | 全部 | 编译期变量。 |
| `--dart-define-from-file FILE` | 可重复 | `build.dart_define_from_file`（追加） | 全部 | define 文件。 |
| `--build-name X.Y.Z` | string | `build.build_name` | 全部 | 版本名。 |
| `--build-number N` | string | `build.build_number` | 全部 | 构建号。 |
| `--split-per-abi[=true\|both]` | 可选值 | `android.split_per_abi` | apk | 不带值等于 `true`。 |
| `--abis LIST` | 逗号分隔 | `android.abis` | apk, aab | 例如 `arm64-v8a,armeabi-v7a`。 |
| `--obfuscate` | 开关 | `build.obfuscate` | 原生目标 | 混淆 Dart 代码。 |
| `--split-debug-info DIR` | path | `build.split_debug_info` | 原生目标 | 调试符号目录。 |
| `-o`, `--output DIR` | 模板 | `output.dir` | 全部 | 产物目录。 |
| `-f`, `--force` | 开关 | `output.overwrite` | 全部 | 覆盖已存在的产物。 |
| `--export-method M` | string | `ios.export_method` | ipa | 导出方式。 |
| `--export-options-plist FILE` | path | `ios.export_options_plist` | ipa | ExportOptions.plist。 |
| `--no-codesign` | 开关 | `ios.codesign: false` | ipa | 未签名 IPA。 |
| `--sign` | 开关 | `macos.sign.enabled: true` | macos, dmg | Developer ID 签名。 |
| `--no-sign` | 开关 | `macos.sign.enabled: false` | macos, dmg | 不签名（同时关闭公证）。 |
| `--sign-identity ID` | string | `macos.sign.identity` | string | 无 | macos, dmg | codesign 证书全名，例如 `Developer ID Application: Your Name (TEAMID)`。**设置后即启用签名**。证书不存在、已吊销或钥匙串中没有 Developer ID 时，构建前就会报错并说明如何导入 .p12。`fpack init` 会在注释中列出本机钥匙串里的 Developer ID 证书。 |
| `--notarize` | 开关 | `macos.sign.notarize: true` | macos, dmg | 公证并装订（zip 与 DMG）。 |
| `--no-notarize` | 开关 | `macos.sign.notarize: false` | macos, dmg | 跳过公证（不上传到 Apple，本地构建更快）。 |
| `--notary-profile NAME` | string | `macos.sign.notary_profile` | string | 无（开启公证但未设置时为 `NotaryProfile`） | macos, dmg | `xcrun notarytool store-credentials <名字>` 创建的钥匙串配置名。**设置后即启用公证**（前提是已启用签名）。 |
| `--dmg-tool T` | string | `macos.dmg.tool` | dmg | `auto` / `hdiutil` / `create-dmg`。 |
| `--base-href PATH` | string | `web.base_href` | web | base href。 |
| `--wasm` | 开关 | `web.wasm: true` | web | WebAssembly 构建。 |
| `-- …` | — | `*.extra_args` 之后 | 全部 | `--` 之后的所有参数原样传给 `flutter build`。 |

### 4.3 其他命令

| 命令 | 参数 | 说明 |
| --- | --- | --- |
| `fpack doctor [目标…]` | — | 按目标检查前置条件（Flutter、Android SDK/JDK、签名、Xcode、证书、CocoaPods、Inno Setup、dpkg-deb…），每个问题都给出修复方法。不指定目标时检查所有目标。 |
| `fpack list`（`ls`、`targets`） | — | 列出所有目标、产物格式，以及本机能否构建。 |
| `fpack init` | `-f`, `--force`：覆盖已有的 fpack.yaml；`-y`：不提问，使用检测到的默认值 | 生成带注释的 `fpack.yaml`。 |
| `fpack clean` | `--dist`：同时删除输出目录（会先确认，`-y` 跳过确认）；`--flutter-clean`：同时运行 `flutter clean`；`--all`：等同 `--dist --flutter-clean`；`-n`, `--dry-run`：只显示将删除什么 | 默认只删除 fpack 的工作目录 `build/fpack/`（临时文件与日志）。 |
| `fpack version` | — | 显示 fpack、原生核心与启动器的版本。 |

### 4.4 退出码

| 码 | 含义 |
| --- | --- |
| 0 | 全部成功（`--dry-run`：计划生成成功） |
| 1 | 至少一个目标在构建/打包时失败 |
| 2 | 用法或配置错误（未知参数、未知目标、fpack.yaml 有误） |
| 3 | 前置条件不满足 / 产物已存在 / 没有可构建的目标 / 找不到 Flutter |
| 130 | 被 Ctrl-C 中断 |

---

## 5. 文件名模板

`output.dir` 与 `output.name` 支持以下占位符：

| 占位符 | 值 | 示例 |
| --- | --- | --- |
| `{app}` | `app.name`（默认 pubspec name） | `xue_hua_im` |
| `{version}` | 版本名 | `1.0.0` |
| `{build}` | 构建号 | `1` |
| `{platform}` | `android` / `ios` / `macos` / `windows` / `linux` / `web` | `android` |
| `{arch}` | `universal`、`arm64-v8a`、`arm64`、`x64` … | `arm64-v8a` |
| `{variant}` | `setup`、`portable`、`unsigned` … | `setup` |
| `{mode}` | 构建模式；**release 时为空** | `profile` |
| `{flavor}` | flavor | `prod` |

在花括号内加前缀 `-`、`_`、`.`、`+`（如 `{-flavor}`、`{+build}`）表示：值非空时先插入这个分隔符再插入值，值为空时整体省略。文件名中不安全的字符会被替换。

默认值的效果：

```
dist/1.0.0+1/xue_hua_im-1.0.0+1-android-universal.apk
dist/1.0.0+1/xue_hua_im-prod-1.0.0+1-ios-arm64.ipa          # --flavor prod
dist/1.0.0+1/xue_hua_im-1.0.0+1-macos-universal-profile.dmg # --profile
```

---

## 6. 完整示例：类似 xue_hua_im 的项目

下面是一个 IM 应用（Android + iOS + macOS）的完整 `fpack.yaml`，每一项都有注释。只需保留你要改的项——删除的项会使用默认值。

```yaml
# fpack.yaml —— 放在项目根目录（与 pubspec.yaml 同级）
# 所有键都可选；字符串中可用 ${VAR} / ${VAR:-默认值} 引用环境变量。

app:
  name: xue_hua_im                 # 产物文件名前缀（默认：pubspec 的 name）
  display_name: 雪花IM              # 安装程序 / 菜单中显示的名称
  description: 雪花即时通讯          # deb/rpm/.desktop 描述（默认：pubspec 的 description）
  publisher: XueHua                # Windows 安装程序发布者
  identifier: com.kurban.im        # 反向域名标识（默认：从 Android/iOS 工程读取）
  homepage: https://example.com

build:
  # `fpack build` 不带目标时构建这些（命令行指定目标时忽略）
  targets: [apk, aab, ipa, dmg]
  mode: release                    # release | profile | debug
  # flavor: prod                   # Android productFlavor / Xcode scheme
  # target: lib/main_prod.dart     # 入口文件
  dart_define:                     # 编译期常量（--dart-define）
    API_BASE: ${API_BASE:-https://api.example.com}
  # dart_define_from_file: [env/prod.json]
  obfuscate: false                 # true 时符号保存到 dist/<版本>/debug-info/
  # build_number: ${BUILD_NUMBER}  # CI 中用流水线编号作为构建号

output:
  dir: "dist/{version}{+build}"    # → dist/1.0.0+1
  name: "{app}{-flavor}-{version}{+build}-{platform}{-arch}{-variant}{-mode}"
  overwrite: false                 # 产物已存在时跳过（--force 覆盖）
  checksums: true                  # 写入 SHA256SUMS

android:
  split_per_abi: both              # 通用 APK + 每个 ABI 一个 APK
  abis: [arm64-v8a, armeabi-v7a, x86_64]
  # 项目已有 android/key.properties + signingConfigs 时，删除下面整段即可，
  # fpack 会直接使用项目自己的签名配置。
  # 在 CI 中可以不写这段，改用 FPACK_ANDROID_* 环境变量。
  signing:
    store_file: ~/keys/xuehua-upload.jks
    store_password: ${FPACK_ANDROID_KEYSTORE_PASSWORD}   # 密码只放在环境变量里
    key_alias: upload
    key_password: ${FPACK_ANDROID_KEY_PASSWORD:-${FPACK_ANDROID_KEYSTORE_PASSWORD}}

ios:
  # app-store-connect（默认，上传 App Store / TestFlight）| ad-hoc | development | enterprise
  export_method: app-store-connect
  # export_options_plist: ios/ExportOptions.plist   # 设置后优先于 export_method
  codesign: true                   # false = 未签名 IPA（用于之后重签名）

macos:
  # 只有这里（以及 FPACK_MACOS_* / 命令行参数）决定是否签名和公证；
  # 不配置时 .app 保留 Xcode 的签名，DMG 不签名。
  sign:
    enabled: true                  # 用 Developer ID 重新签名 .app 和 DMG
    identity: "Developer ID Application: Your Name (TEAMID)"   # 留空则自动选择
    entitlements: macos/Runner/Release.entitlements
    notarize: true                 # 提交 Apple 公证并装订；本地测试时用 --no-notarize
    notary_profile: XueHua         # xcrun notarytool store-credentials XueHua …
  dmg:
    tool: auto                     # auto | hdiutil | create-dmg
    volume_name: 雪花IM
    # background: assets/dmg_background.png   # 仅 create-dmg
```

常用命令：

```bash
fpack doctor                                   # 先检查本机能构建什么、缺什么
fpack build --dry-run                          # 查看 build.targets 的完整计划
fpack build apk aab                            # Android
fpack build ipa --export-method ad-hoc         # 测试分发的 IPA
fpack build macos dmg --no-notarize            # 本地签名 DMG，不上传公证
fpack build dmg                                # 签名 + 公证 + 装订（发布用）
fpack build --all --json > result.json         # CI：本机能构建的全部目标
```

CI 中的 Android 签名（不需要 fpack.yaml 中的 `android.signing` 段）：

```yaml
- run: fpack build apk aab --split-per-abi=both
  env:
    FPACK_ANDROID_KEYSTORE_BASE64: ${{ secrets.KEYSTORE_BASE64 }}
    FPACK_ANDROID_KEYSTORE_PASSWORD: ${{ secrets.KEYSTORE_PASSWORD }}
    FPACK_ANDROID_KEY_ALIAS: upload
    FPACK_BUILD_NUMBER: ${{ github.run_number }}
```
