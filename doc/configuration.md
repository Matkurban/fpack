# fpack 配置参考

本文列出 fpack 1.1.2 的**全部**配置方式：`fpack.yaml` 的每个键、每个 `FPACK_*` 环境变量、每个命令行参数，以及它们的类型、默认值、影响的目标和作用。

- 所有配置都是可选的：不写 `fpack.yaml` 也能直接 `fpack build apk`。
- `fpack init` 会在项目根目录生成一份带注释的 `fpack.yaml`（这是 fpack 唯一会写入项目的文件）。
- fpack **不会修改**项目里的任何其他文件（`pubspec.yaml`、`key.properties`、Gradle、Xcode 工程等都只读）。
- 概览文档：[README.md](../README.md)（English）· [README.ZH.md](../README.ZH.md)（中文）。macOS 证书与公证凭证的获取方法见 [2.13](#213-获取-macos-签名证书与公证凭证)。

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
| macOS Developer ID 签名 / 公证 | `macos.sign` → `FPACK_MACOS_*` → `--sign/--no-sign`、`--sign-identity`、`--installer-identity`、`--notarize/--no-notarize`、`--notary-profile` |
| 输出语言 | 系统语言（`LC_ALL` → `LC_MESSAGES` → `LANG` → macOS/Windows 系统设置）→ `FPACK_LANG` → `--lang` |

补充规则：

- **列表类参数是追加，不是替换**：`--dart-define`、`--dart-define-from-file` 会追加到 `fpack.yaml` 中已有的值之后。
- **`extra_args` 的拼接顺序**：`build.extra_args` → 平台的 `extra_args`（如 `android.extra_args`）→ 命令行 `--` 之后的参数。都会原样追加到 `flutter build …` 的末尾。
- **macOS 签名的联动规则**：设置了证书（`identity`）即视为启用签名；设置了任一公证凭证（`notary_profile`、API 密钥或 Apple ID）即视为启用公证；关闭签名（`--no-sign` / `enabled: false`）会同时关闭公证；`enabled: false` 与 `notarize: true` 同时出现是配置错误。
- **macOS 签名只来自 fpack 自己的配置**：fpack 不读取 `pubspec.yaml` 中其他插件的配置（例如 [`dmg`](https://pub.dev/packages/dmg) 包的 `dmg:` 段）。什么都不配置时，.app 保留 Xcode 工程自己的签名，DMG 不签名。
- **`fpack.yaml` 中的环境变量引用**：任何字符串值都可以写 `${VAR}` 或 `${VAR:-默认值}`。未设置且没有默认值的变量会被替换为空字符串，并在运行时给出警告。
- **路径**：`fpack.yaml` 中的相对路径都相对于 **项目根目录**（`pubspec.yaml` 所在目录）；支持 `~/`。
- **未知键是错误**：拼错的键会报错并提示「你是不是想写 …」，不会被静默忽略。
- **布尔值**：`true/false`、`yes/no`、`on/off`、`1/0` 都可以（环境变量同样适用）。

---

## 2. fpack.yaml 键参考

表头说明：**类型**中 `list` 表示既可以写 YAML 列表，也可以写单个字符串；**目标**指受影响的 `fpack build` 目标，「全部」表示所有目标。本节表格由 fpack 的键注册表自动生成（与 `fpack init` 模板、`fpack schema` 输出、配置校验使用同一份定义），所以永远与实际行为一致。

**编辑器补全**：`fpack init` 生成的文件第一行是

```yaml
# yaml-language-server: $schema=https://raw.githubusercontent.com/Matkurban/fpack/main/schema/fpack.schema.json
```

VS Code（Red Hat YAML 插件）、IntelliJ/Android Studio 会据此提供键补全、悬停说明（中英文）、可选值提示和类型校验。也可以用 `fpack schema -o fpack.schema.json` 导出到本地。

**校验**：加载配置时会检查未知键（并提示「你是不是想写 …」）、类型错误（会指出行号和键名，例如 `line 3: android.signing.v1: expected true or false, got "maybe"`）、可选值、取值范围（如 `web.optimization_level` 0–4、`web.base_href` 必须以 `/` 开头和结尾、`windows.inno_setup.app_id` 必须是 GUID）、组合错误（如 Apple ID 公证缺少 team id）；构建前还会检查当前目标用到的文件路径是否存在。

<!-- BEGIN GENERATED KEYS -->
### 2.1 `app`

应用信息：用于安装程序、软件包元数据。

| 键 | 类型 | 默认值 | 目标 | 环境变量 / 参数 | 说明 |
| --- | --- | --- | --- | --- | --- |
| `app.name` | string | `pubspec 的 name` | 全部 |  | 产物文件名前缀（output.name 中的 {app}）。 示例：`xue_hua_im` |
| `app.display_name` | string | `macOS 的 PRODUCT_NAME，否则为 pubspec 的 name` | exe, msix, pkg, deb, rpm, appimage, linux |  | 给人看的应用名：安装程序标题、开始菜单、.desktop 的 Name=。 示例：`雪花IM` |
| `app.description` | string | `pubspec 的 description` | deb, rpm, appimage, msix |  | 简短描述：deb 的 Description、rpm 的 Summary、.desktop 的 Comment=、msix 描述。 示例：`A fast and secure messenger` |
| `app.publisher` | string | `windows/runner/Runner.rc 中的 CompanyName` | exe, msix, deb, rpm |  | 公司 / 作者：Windows 安装程序发布者、msix 发布者显示名、deb Maintainer 的后备值、rpm Vendor。 示例：`XueHua Tech` |
| `app.identifier` | string | `Linux APPLICATION_ID、Android applicationId 或 iOS bundle id` | exe, msix, pkg, appimage |  | 反向域名格式的应用 ID：Inno Setup AppId 的种子、msix identity name、pkg identifier 的后备值。 示例：`com.xuehua.im` |
| `app.homepage` | url | — | exe, deb, rpm |  | 官网：Inno Setup 发布者网址、deb 的 Homepage、rpm 的 URL。 示例：`https://xuehua.example.com` |
| `app.support_url` | url | `app.homepage` | exe |  | 技术支持链接（Windows“应用和功能”中显示）。 示例：`https://xuehua.example.com/support` |
| `app.maintainer` | string | `app.publisher` | deb, rpm |  | deb Maintainer 与 rpm Packager 字段，格式 “名字 <邮箱>”。 示例：`XueHua Team <dev@xuehua.example.com>` |
| `app.copyright` | string | `© <年份> <发布者>` | exe, deb, rpm |  | 版权信息：Inno Setup 的 AppCopyright 与版本信息、deb 的 copyright 文件。 示例：`© 2026 XueHua Tech` |
| `app.license` | string | `Proprietary` | rpm, deb |  | 许可证（SPDX 标识）：rpm 的 License、deb 的 copyright 文件。 示例：`MIT` |

### 2.2 `flutter`

Flutter SDK 选择。

| 键 | 类型 | 默认值 | 目标 | 环境变量 / 参数 | 说明 |
| --- | --- | --- | --- | --- | --- |
| `flutter.sdk` | path | `FLUTTER_ROOT、fvm、PATH 中的 flutter` | 全部 | `FPACK_FLUTTER`<br>`--flutter` | 使用的 Flutter SDK 根目录。 示例：`~/fvm/versions/stable` |

### 2.3 `build`

所有 flutter build 共用的选项。

| 键 | 类型 | 默认值 | 目标 | 环境变量 / 参数 | 说明 |
| --- | --- | --- | --- | --- | --- |
| `build.targets` | list（或单个字符串） | — | — |  | 执行 `fpack build` 且不带目标时构建的目标。 示例：`[apk, aab, ipa, dmg]` |
| `build.mode` | `release` \\| `profile` \\| `debug` | `release` | 全部 | `FPACK_MODE`<br>`--mode` | 构建模式。 示例：`release` |
| `build.flavor` | string | — | apk, aab, ipa, macos, dmg, pkg | `FPACK_FLAVOR`<br>`--flavor` | flavor / Xcode scheme（--flavor）；文件名中的 {flavor}。 示例：`prod` |
| `build.target` | path | `lib/main.dart` | 全部 | `FPACK_ENTRY`<br>`-t, --target` | 入口文件（flutter -t）。 示例：`lib/main_prod.dart` |
| `build.dart_define` | map 或 `KEY=VALUE` 列表 | — | 全部 | `--dart-define` | --dart-define 值（map 或 KEY=VALUE 列表）。 示例：`{API_URL: https://api.example.com}` |
| `build.dart_define_from_file` | list（或单个字符串） | — | 全部 | `--dart-define-from-file` | --dart-define-from-file 文件（.json 或 .env）。 示例：`[config/prod.json]` |
| `build.build_name` | string/number | `pubspec 版本号 + 之前的部分` | 全部 | `FPACK_BUILD_NAME`<br>`--build-name` | 版本名（{version}）。 示例：`1.2.0` |
| `build.build_number` | string/number | `pubspec 版本号 + 之后的部分` | 全部 | `FPACK_BUILD_NUMBER`<br>`--build-number` | 构建号（{build}）。 示例：`42` |
| `build.obfuscate` | bool | `false` | apk, aab, ipa, macos, dmg, pkg, windows, exe, msix, linux, deb, rpm, appimage | `FPACK_OBFUSCATE`<br>`--obfuscate` | 混淆 Dart 代码；符号文件保存到 split_debug_info（默认 <输出目录>/debug-info/<平台>）。 示例：`true` |
| `build.split_debug_info` | string | `混淆时为 <输出目录>/debug-info/<平台>` | apk, aab, ipa, macos, dmg, pkg, windows, exe, msix, linux, deb, rpm, appimage | `--split-debug-info` | Dart 调试符号目录（--split-debug-info）。 示例：`build/symbols` |
| `build.tree_shake_icons` | bool | `true` | 全部 |  | false 时传 --no-tree-shake-icons（保留全部图标字体字形）。 示例：`false` |
| `build.extra_args` | list（或单个字符串） | — | 全部 |  | 追加到每个 flutter build 的参数（命令行 -- 之后的参数同理）。 示例：`[--no-pub]` |

### 2.4 `output`

产物输出目录与命名。

| 键 | 类型 | 默认值 | 目标 | 环境变量 / 参数 | 说明 |
| --- | --- | --- | --- | --- | --- |
| `output.dir` | string | `dist/{version}{+build}` | 全部 | `FPACK_OUTPUT_DIR`<br>`-o, --output` | 产物目录（相对于项目，可用占位符）。 示例：`"dist/{version}{+build}"` |
| `output.name` | string | `{app}{-flavor}-{version}{+build}-{platform}{-arch}{-variant}{-mode}` | 全部 |  | 文件名模板（不含扩展名）。占位符：{app} {version} {build} {platform} {arch} {variant} {mode} {flavor} {target} {date}；{-x}/{+x} 表示 x 非空时才加分隔符。 示例：`"{app}-{version}-{platform}{-arch}"` |
| `output.names` | map | — | 全部 |  | 按目标单独设置文件名模板（目标 → 模板），优先于 output.name。 示例：`{exe: "{app}-setup-{version}", web: "{app}-web"}` |
| `output.overwrite` | bool | `false` | 全部 | `FPACK_OVERWRITE`<br>`-f, --force` | 已存在同名产物时覆盖，而不是停止。 示例：`true` |
| `output.checksums` | bool | `true` | 全部 |  | 在产物旁写入校验和文件。 示例：`false` |
| `output.checksum_algorithm` | `sha256` \\| `sha512` | `sha256` | 全部 |  | 校验算法（文件名 SHA256SUMS 或 SHA512SUMS）。 示例：`sha512` |

### 2.5 `hooks`

构建前后执行的 shell 命令（工作目录：项目根目录）。

| 键 | 类型 | 默认值 | 目标 | 环境变量 / 参数 | 说明 |
| --- | --- | --- | --- | --- | --- |
| `hooks.pre_build` | list（或单个字符串） | — | 全部 |  | 第一次 flutter build 之前执行一次；失败则停止构建。 示例：`[dart run build_runner build --delete-conflicting-outputs]` |
| `hooks.post_build` | list（或单个字符串） | — | 全部 |  | 全部目标完成后执行一次；FPACK_ARTIFACTS 为产物列表（每行一个），FPACK_SUCCESS 为 true/false。 示例：`[./scripts/upload.sh]` |
| `hooks.pre_package` | map：目标 → 命令列表 | — | 全部 |  | 按目标（目标 → 命令）在打包步骤之前执行；提供 FPACK_TARGET。 示例：`{apk: [./scripts/check_size.sh]}` |
| `hooks.post_package` | map：目标 → 命令列表 | — | 全部 |  | 按目标在产物生成后执行；提供 FPACK_ARTIFACT（第一个产物）和 FPACK_ARTIFACTS。 示例：`{dmg: [./scripts/upload_dmg.sh "$FPACK_ARTIFACT"]}` |

### 2.6 `android`

Android：apk、aab。

| 键 | 类型 | 默认值 | 目标 | 环境变量 / 参数 | 说明 |
| --- | --- | --- | --- | --- | --- |
| `android.split_per_abi` | `false` \\| `true` \\| `both` | `false` | apk | `FPACK_SPLIT_PER_ABI`<br>`--split-per-abi` | false：一个通用 APK；true：每个 ABI 一个 APK；both：两者都要。 示例：`both` |
| `android.abis` | list（或单个字符串） | `armeabi-v7a、arm64-v8a、x86_64` | apk, aab | `--abis` | 目标 ABI（--target-platform）。 示例：`[arm64-v8a, armeabi-v7a]` |
| `android.project_args` | map | — | apk, aab |  | Gradle 项目属性（flutter -P key=value），build.gradle 中可用 project.findProperty 读取（例如开关 minify/R8）。 示例：`{minify: "true"}` |
| `android.extra_args` | list（或单个字符串） | — | apk, aab |  | 追加到 flutter build apk/appbundle 的参数。 示例：`[--android-skip-build-dependency-validation]` |

#### `android.signing`

发布签名：通过注入方式生效，不修改 Gradle 文件。密码请放在环境变量中。

| 键 | 类型 | 默认值 | 目标 | 环境变量 / 参数 | 说明 |
| --- | --- | --- | --- | --- | --- |
| `android.signing.store_file` | path | `无（使用项目自己的 android/key.properties）` | apk, aab | `FPACK_ANDROID_KEYSTORE` | keystore 文件（.jks/.keystore）。CI 中可用 FPACK_ANDROID_KEYSTORE_BASE64 提供。 示例：`~/keys/upload.jks` |
| `android.signing.store_password` | string | — | apk, aab | `FPACK_ANDROID_KEYSTORE_PASSWORD` | keystore 密码。 示例：`${KEYSTORE_PASSWORD}` |
| `android.signing.key_alias` | string | — | apk, aab | `FPACK_ANDROID_KEY_ALIAS` | key 别名。 示例：`upload` |
| `android.signing.key_password` | string | `store_password` | apk, aab | `FPACK_ANDROID_KEY_PASSWORD` | key 密码。 示例：`${KEY_PASSWORD}` |
| `android.signing.v1` | bool | `apksigner 默认（minSdk < 24 时开启）` | apk |  | APK v1 签名（JAR 签名，Android 7 以下需要）。设置 v1-v4 任意一项时，fpack 会用 apksigner 和 android.signing 重新签名 APK。 示例：`true` |
| `android.signing.v2` | bool | `true` | apk |  | APK v2 签名（Android 7+）。 示例：`true` |
| `android.signing.v3` | bool | `true` | apk |  | APK v3 签名（Android 9+，支持密钥轮换）。 示例：`true` |
| `android.signing.v4` | bool | `false` | apk |  | APK v4 签名（增量安装，Android 11+）；会在 APK 旁生成 <apk>.idsig。 示例：`true` |

### 2.7 `ios`

iOS：ipa。设置 team_id … export_options 中任意一项时，fpack 会自动生成 ExportOptions.plist。

| 键 | 类型 | 默认值 | 目标 | 环境变量 / 参数 | 说明 |
| --- | --- | --- | --- | --- | --- |
| `ios.export_method` | `app-store-connect` \\| `app-store` \\| `release-testing` \\| `ad-hoc` \\| `development` \\| `debugging` \\| `enterprise` | `app-store-connect` | ipa | `FPACK_IOS_EXPORT_METHOD`<br>`--export-method` | IPA 导出方式。 示例：`ad-hoc` |
| `ios.export_options_plist` | path | — | ipa | `FPACK_IOS_EXPORT_OPTIONS_PLIST`<br>`--export-options-plist` | 自己的 ExportOptions.plist；优先于下面所有生成选项。 示例：`ios/ExportOptions.plist` |
| `ios.codesign` | bool | `true` | ipa | `FPACK_IOS_CODESIGN`<br>`--no-codesign` | false：构建未签名 IPA（Payload/ 结构），用于之后重签名。 示例：`false` |
| `ios.team_id` | string | `Xcode 工程中的 DEVELOPMENT_TEAM` | ipa |  | Apple 团队 ID（teamID）。 示例：`ABCDE12345` |
| `ios.signing_style` | `automatic` \\| `manual` | `automatic` | ipa |  | signingStyle：automatic 或 manual（manual 需要 provisioning_profiles）。 示例：`manual` |
| `ios.signing_certificate` | string | — | ipa |  | signingCertificate（手动签名），例如 "Apple Distribution"。 示例：`Apple Distribution` |
| `ios.provisioning_profiles` | map | — | ipa |  | provisioningProfiles：bundle id → 描述文件名称或 UUID（扩展也要列出）。 示例：`{com.xuehua.im: XueHua AdHoc}` |
| `ios.upload_symbols` | bool | `true` | ipa |  | uploadSymbols：是否上传符号表到 App Store Connect。 示例：`false` |
| `ios.manage_app_version_and_build_number` | bool | `true` | ipa |  | manageAppVersionAndBuildNumber：是否由 App Store Connect 自动管理版本号/构建号。 示例：`false` |
| `ios.destination` | `export` \\| `upload` | `export` | ipa |  | export 为本地导出 IPA，upload 为直接上传到 App Store Connect（ExportOptions 的 destination）。 示例：`upload` |
| `ios.thinning` | string | `<none>` | ipa |  | ad-hoc/development/enterprise 导出时的瘦身（thinning）选项。 示例：`<thin-for-all-variants>` |
| `ios.strip_swift_symbols` | bool | `true` | ipa |  | stripSwiftSymbols。 示例：`false` |
| `ios.export_options` | map (any) | — | ipa |  | 其他任意 ExportOptions.plist 键（原样写入）。 示例：`{iCloudContainerEnvironment: Production}` |
| `ios.extra_args` | list（或单个字符串） | — | ipa |  | 追加到 flutter build ipa 的参数。 示例：`[--no-tree-shake-icons]` |

### 2.8 `macos`

macOS：macos（.app zip）、dmg、pkg。

| 键 | 类型 | 默认值 | 目标 | 环境变量 / 参数 | 说明 |
| --- | --- | --- | --- | --- | --- |
| `macos.extra_args` | list（或单个字符串） | — | macos, dmg, pkg |  | 追加到 flutter build macos 的参数。 示例：`[--no-tree-shake-icons]` |

#### `macos.sign`

Developer ID 签名与公证（只来自本文件、FPACK_MACOS_* 和命令行参数）。

| 键 | 类型 | 默认值 | 目标 | 环境变量 / 参数 | 说明 |
| --- | --- | --- | --- | --- | --- |
| `macos.sign.enabled` | bool | `设置了 identity 时为 true` | macos, dmg, pkg | `FPACK_MACOS_SIGN`<br>`--sign / --no-sign` | 用 Developer ID 重新签名 .app（并签名 DMG）。为 true 但未设置 identity 时自动选用第一个 “Developer ID Application” 证书。false（--no-sign）同时关闭 pkg 签名与公证。 示例：`true` |
| `macos.sign.identity` | string | — | macos, dmg, pkg | `FPACK_MACOS_SIGN_IDENTITY`<br>`--sign-identity` | App 的 codesign 证书；设置后即启用签名。 示例：`"Developer ID Application: Your Name (TEAMID)"` |
| `macos.sign.entitlements` | path | `macos/Runner/Release.entitlements` | macos, dmg, pkg |  | 重新签名 App 时使用的 entitlements。 示例：`macos/Runner/Release.entitlements` |
| `macos.sign.hardened_runtime` | bool | `true` | macos, dmg, pkg |  | 使用 Hardened Runtime 签名（公证必需）。 示例：`true` |
| `macos.sign.notarize` | bool | `设置了公证凭证时为 true` | macos, dmg, pkg | `FPACK_MACOS_NOTARIZE`<br>`--notarize / --no-notarize` | 公证并装订 zip、DMG 与已签名的 pkg（会上传到 Apple）。 示例：`true` |
| `macos.sign.notary_profile` | string | — | macos, dmg, pkg | `FPACK_MACOS_NOTARY_PROFILE`<br>`--notary-profile` | `xcrun notarytool store-credentials <名字>` 创建的钥匙串配置名（本地推荐）。 示例：`NotaryProfile` |
| `macos.sign.notary_apple_id` | string | — | macos, dmg, pkg | `FPACK_NOTARY_APPLE_ID` | 公证用 Apple ID（配合 notary_team_id + notary_password），替代钥匙串配置。 示例：`dev@example.com` |
| `macos.sign.notary_team_id` | string | — | macos, dmg, pkg | `FPACK_NOTARY_TEAM_ID` | Apple ID 公证时的团队 ID。 示例：`ABCDE12345` |
| `macos.sign.notary_password` | string | — | macos, dmg, pkg | `FPACK_NOTARY_PASSWORD` | Apple ID 公证用的 App 专用密码。 示例：`${NOTARY_PASSWORD}` |
| `macos.sign.notary_api_key` | path | — | macos, dmg, pkg | `FPACK_NOTARY_API_KEY` | 公证用 App Store Connect API 密钥（.p8，适合 CI）。 示例：`~/keys/AuthKey_ABC123.p8` |
| `macos.sign.notary_api_key_id` | string | — | macos, dmg, pkg | `FPACK_NOTARY_API_KEY_ID` | API 密钥 ID。 示例：`ABC123DEF4` |
| `macos.sign.notary_api_issuer` | string | — | macos, dmg, pkg | `FPACK_NOTARY_API_ISSUER` | API Issuer UUID（个人密钥可省略）。 示例：`69a6de7e-…` |
| `macos.sign.installer_identity` | string | — | pkg | `FPACK_MACOS_INSTALLER_IDENTITY`<br>`--installer-identity` | 签名 .pkg 的证书，与 App 的 “Developer ID Application” 不同。不设置则 pkg 不签名。 示例：`"Developer ID Installer: Your Name (TEAMID)"` |

#### `macos.notarize`

如何等待 Apple 公证服务。每次提交都会记录到 <输出目录>/NOTARIZATION.md 和 notarization.json，其中有可直接复制的查询命令。

| 键 | 类型 | 默认值 | 目标 | 环境变量 / 参数 | 说明 |
| --- | --- | --- | --- | --- | --- |
| `macos.notarize.wait` | bool | `true` | macos, dmg, pkg | `FPACK_NOTARIZE_WAIT`<br>`--notarize-no-wait` | true：一直等待 Apple 返回结果（显示已等待时间；按 Ctrl-C 只是停止等待，Apple 端会继续处理）。false：提交后写入 NOTARIZATION.md / notarization.json 并结束，状态为 “submitted”；之后运行 `fpack notarize finish` 装订。 示例：`false` |

#### `macos.dmg`

DMG 磁盘镜像。窗口/图标布局需要 create-dmg（brew install create-dmg）。

| 键 | 类型 | 默认值 | 目标 | 环境变量 / 参数 | 说明 |
| --- | --- | --- | --- | --- | --- |
| `macos.dmg.tool` | `auto` \\| `hdiutil` \\| `create-dmg` | `auto` | dmg | `FPACK_DMG_TOOL`<br>`--dmg-tool` | auto：装了 create-dmg（或设置了布局键）时用 create-dmg，否则用 hdiutil。 示例：`create-dmg` |
| `macos.dmg.volume_name` | string | `.app 名称` | dmg |  | 挂载 DMG 后显示的卷名。 示例：`雪花IM` |
| `macos.dmg.volume_icon` | path | — | dmg |  | 卷图标（.icns）。需要 create-dmg。 示例：`macos/dmg/volume.icns` |
| `macos.dmg.background` | path | — | dmg |  | 窗口背景图。需要 create-dmg。 示例：`macos/dmg/background.png` |
| `macos.dmg.window_position` | `[x, y]` | `[200, 120]` | dmg |  | 窗口位置 [x, y]。需要 create-dmg。 示例：`[200, 120]` |
| `macos.dmg.window_size` | `[x, y]` | `[660, 400]` | dmg |  | 窗口大小 [宽, 高]。需要 create-dmg。 示例：`[660, 400]` |
| `macos.dmg.icon_size` | int | `128` | dmg |  | 窗口中的图标大小。需要 create-dmg。 示例：`128` |
| `macos.dmg.app_position` | `[x, y]` | `[180, 190]` | dmg |  | App 图标位置 [x, y]。需要 create-dmg。 示例：`[180, 190]` |
| `macos.dmg.applications_position` | `[x, y]` | `[480, 190]` | dmg |  | “应用程序”快捷方式位置 [x, y]。需要 create-dmg。 示例：`[480, 190]` |
| `macos.dmg.format` | `UDZO` \\| `UDBZ` \\| `ULFO` \\| `ULMO` \\| `UDRO` | `UDZO` | dmg |  | 镜像格式：UDZO（zlib）、UDBZ（bzip2）、ULFO（lzfse，macOS 10.11+）、ULMO（lzma，10.15+）、UDRO（只读不压缩）。 示例：`ULFO` |
| `macos.dmg.filesystem` | `HFS+` \\| `APFS` | `HFS+` | dmg |  | 镜像文件系统。 示例：`APFS` |
| `macos.dmg.license` | path | — | dmg |  | 打开 DMG 时显示的许可协议（.txt/.rtf）。需要 create-dmg。 示例：`LICENSE.txt` |

#### `macos.pkg`

安装包（pkgbuild + productbuild）。

| 键 | 类型 | 默认值 | 目标 | 环境变量 / 参数 | 说明 |
| --- | --- | --- | --- | --- | --- |
| `macos.pkg.identifier` | string | `macOS 工程的 bundle id` | pkg |  | 安装包标识（pkgutil --pkgs 中显示）。 示例：`com.xuehua.im` |
| `macos.pkg.version` | string | `版本名` | pkg |  | 安装包版本。 示例：`1.2.0` |
| `macos.pkg.install_location` | string | `/Applications` | pkg |  | App 安装到的绝对路径目录。 示例：`/Applications` |
| `macos.pkg.title` | string | `.app 名称` | pkg |  | 安装器窗口标题。 示例：`雪花IM` |
| `macos.pkg.welcome` | path | — | pkg |  | 欢迎页（.html/.rtf/.txt）。 示例：`macos/installer/welcome.html` |
| `macos.pkg.readme` | path | — | pkg |  | “请先阅读”页（.html/.rtf/.txt）。 示例：`macos/installer/readme.html` |
| `macos.pkg.license` | path | — | pkg |  | 用户必须同意的许可页（.html/.rtf/.txt）。 示例：`macos/installer/license.rtf` |
| `macos.pkg.conclusion` | path | — | pkg |  | 完成页（.html/.rtf/.txt）。 示例：`macos/installer/done.html` |
| `macos.pkg.background` | path | — | pkg |  | 背景图（浅色/深色模式都使用）。 示例：`macos/installer/background.png` |
| `macos.pkg.min_os` | string | `工程的 MACOSX_DEPLOYMENT_TARGET` | pkg |  | 最低 macOS 版本；低于此版本时安装器会拒绝安装。 示例：`10.15` |
| `macos.pkg.preinstall` | path | — | pkg |  | 安装前执行的脚本（自动设为可执行）。 示例：`macos/installer/preinstall.sh` |
| `macos.pkg.postinstall` | path | — | pkg |  | 安装后执行的脚本。 示例：`macos/installer/postinstall.sh` |
| `macos.pkg.relocatable` | bool | `false` | pkg |  | true：App 被移动过时在原位置升级；false：总是安装到 install_location。 示例：`true` |
| `macos.pkg.require_restart` | bool | `false` | pkg |  | 安装完成后要求重启。 示例：`true` |

### 2.9 `windows`

Windows：windows（zip）、exe（Inno Setup）、msix。

| 键 | 类型 | 默认值 | 目标 | 环境变量 / 参数 | 说明 |
| --- | --- | --- | --- | --- | --- |
| `windows.extra_args` | list（或单个字符串） | — | windows, exe, msix |  | 追加到 flutter build windows 的参数。 示例：`[--no-tree-shake-icons]` |

#### `windows.inno_setup`

Inno Setup 安装程序（.exe）。

| 键 | 类型 | 默认值 | 目标 | 环境变量 / 参数 | 说明 |
| --- | --- | --- | --- | --- | --- |
| `windows.inno_setup.app_id` | string | `由 app.identifier 推导的固定 GUID` | exe |  | AppId：请永远保持不变，升级时才能覆盖旧版本。 示例：`8D3B5E6A-1C2D-4E5F-8A9B-0C1D2E3F4A5B` |
| `windows.inno_setup.script` | path | — | exe |  | 自定义 .iss 脚本；fpack 会传入 /DAppName /DAppVersion /DAppPublisher /DAppExeName /DSourceDir /DAppId /DAppURL。 示例：`windows/installer.iss` |
| `windows.inno_setup.iscc` | path | `PATH 或 Program Files 中的 ISCC` | exe |  | ISCC.exe 路径。 示例：`C:/Program Files (x86)/Inno Setup 6/ISCC.exe` |
| `windows.inno_setup.publisher` | string | `app.publisher` | exe |  | AppPublisher（发布者）。 示例：`XueHua Tech` |
| `windows.inno_setup.publisher_url` | url | `app.homepage` | exe |  | AppPublisherURL（发布者网址）。 示例：`https://xuehua.example.com` |
| `windows.inno_setup.support_url` | url | `app.support_url` | exe |  | AppSupportURL（支持网址）。 示例：`https://xuehua.example.com/support` |
| `windows.inno_setup.updates_url` | url | — | exe |  | AppUpdatesURL（更新网址）。 示例：`https://xuehua.example.com/download` |
| `windows.inno_setup.default_dir` | string | `{autopf}\<显示名>` | exe |  | 默认安装目录（可用 Inno 常量）。 示例：`'{autopf}\XueHua'` |
| `windows.inno_setup.group_name` | string | `显示名` | exe |  | 开始菜单文件夹。 示例：`XueHua` |
| `windows.inno_setup.desktop_icon` | `none` \\| `unchecked` \\| `checked` | `unchecked` | exe |  | 桌面快捷方式：none 不提供，unchecked 提供但默认不勾选，checked 默认勾选。 示例：`checked` |
| `windows.inno_setup.run_after_install` | bool | `true` | exe |  | 最后一页提供“运行 <应用>”选项。 示例：`false` |
| `windows.inno_setup.license_file` | path | — | exe |  | 许可协议页（.txt/.rtf）。 示例：`LICENSE.txt` |
| `windows.inno_setup.info_before` | path | — | exe |  | 安装前信息页（.txt/.rtf）。 示例：`docs/before.txt` |
| `windows.inno_setup.info_after` | path | — | exe |  | 安装后信息页（.txt/.rtf）。 示例：`docs/after.txt` |
| `windows.inno_setup.setup_icon` | path | `windows/runner/resources/app_icon.ico` | exe |  | 安装程序图标（.ico）。 示例：`windows/installer/setup.ico` |
| `windows.inno_setup.wizard_image` | path | — | exe |  | 向导大图（.bmp/.png，100% 缩放下 164×314）。 示例：`windows/installer/wizard.bmp` |
| `windows.inno_setup.wizard_small_image` | path | — | exe |  | 向导小图（.bmp/.png，55×55）。 示例：`windows/installer/wizard-small.bmp` |
| `windows.inno_setup.wizard_style` | `modern` \\| `classic` | `modern` | exe |  | 向导样式。 示例：`classic` |
| `windows.inno_setup.languages` | list（或单个字符串） | `[en]` | exe |  | 安装程序语言；第一个为默认，多个时让用户选择。zh-CN/zh-TW 需要 Inno Setup 6.5+（旧版本时 fpack 自带语言文件）。支持：en zh-CN zh-TW ja ko de fr es it pt-BR pt ru uk tr pl nl cs ar he。 示例：`[zh-CN, en]` |
| `windows.inno_setup.privileges` | `user` \\| `admin` \\| `ask` | `ask` | exe |  | user：仅当前用户（无需管理员）；admin：所有用户（需要 UAC）；ask：让用户选择。 示例：`admin` |
| `windows.inno_setup.compression` | string | `lzma2/max` | exe |  | 压缩方式（lzma2/max、lzma2/ultra64、zip、none 等）。 示例：`lzma2/ultra64` |
| `windows.inno_setup.min_version` | string | `10.0` | exe |  | 最低 Windows 版本（MinVersion）。 示例：`10.0.17763` |

#### `windows.sign`

使用 signtool 进行 Authenticode 签名：应用 .exe 与安装程序。

| 键 | 类型 | 默认值 | 目标 | 环境变量 / 参数 | 说明 |
| --- | --- | --- | --- | --- | --- |
| `windows.sign.certificate` | path | — | windows, exe, msix | `FPACK_WINDOWS_CERTIFICATE` | 代码签名证书（.pfx）。设置它（或 thumbprint）后会签名应用 .exe、安装程序和 MSIX。 示例：`C:/certs/codesign.pfx` |
| `windows.sign.password` | string | — | windows, exe, msix | `FPACK_WINDOWS_CERTIFICATE_PASSWORD` | 证书密码。 示例：`${WINDOWS_CERT_PASSWORD}` |
| `windows.sign.thumbprint` | string | — | windows, exe | `FPACK_WINDOWS_CERT_THUMBPRINT` | Windows 证书存储中证书的 SHA-1 指纹（替代 .pfx）。 示例：`1A2B3C…` |
| `windows.sign.timestamp_url` | url | `http://timestamp.digicert.com` | windows, exe |  | RFC 3161 时间戳服务器。 示例：`http://timestamp.sectigo.com` |
| `windows.sign.signtool` | path | `PATH 或 Windows SDK 中的 signtool` | windows, exe |  | signtool.exe 路径。 示例：`C:/Program Files (x86)/Windows Kits/10/bin/10.0.22621.0/x64/signtool.exe` |
| `windows.sign.description` | string | `显示名` | windows, exe |  | UAC 弹窗中显示的描述（/d）。 示例：`XueHua IM` |

#### `windows.msix`

MSIX 包（需要 msix 开发依赖）。这些键会覆盖 pubspec 中的 msix_config。

| 键 | 类型 | 默认值 | 目标 | 环境变量 / 参数 | 说明 |
| --- | --- | --- | --- | --- | --- |
| `windows.msix.display_name` | string | `app.display_name` | msix |  | 显示名称。 示例：`雪花IM` |
| `windows.msix.publisher_display_name` | string | `app.publisher` | msix |  | 发布者显示名称。 示例：`XueHua Tech` |
| `windows.msix.identity_name` | string | `app.identifier` | msix |  | 包标识名称（Identity Name）。 示例：`com.xuehua.im` |
| `windows.msix.publisher` | string | `取自证书` | msix |  | 发布者（证书 Subject）；上架 Store 时必填。 示例：`CN=XueHua Tech, O=XueHua Tech, C=CN` |
| `windows.msix.version` | string | `版本名补齐为 a.b.c.0` | msix |  | MSIX 版本（a.b.c.d）。 示例：`1.2.0.0` |
| `windows.msix.logo` | path | `msix 默认 / 应用图标` | msix |  | Logo 图片（≥ 400×400 的 PNG）。 示例：`windows/msix/logo.png` |
| `windows.msix.description` | string | `app.description` | msix |  | 包描述。 示例：`A fast and secure messenger` |
| `windows.msix.capabilities` | list（或单个字符串） | — | msix |  | 能力声明（capabilities）。 示例：`[internetClient, microphone, webcam]` |
| `windows.msix.languages` | list（或单个字符串） | — | msix |  | 语言。 示例：`[zh-cn, en-us]` |
| `windows.msix.file_extensions` | list（或单个字符串） | — | msix |  | 应用可打开的文件扩展名。 示例：`[.xhim]` |
| `windows.msix.protocol_activation` | list（或单个字符串） | — | msix |  | 可激活应用的 URL 协议。 示例：`[xuehua]` |
| `windows.msix.execution_alias` | string | — | msix |  | 命令行别名。 示例：`xuehua` |
| `windows.msix.start_at_login` | bool | `false` | msix |  | 登录时自动启动。 示例：`true` |
| `windows.msix.os_min_version` | string | `10.0.17763.0` | msix |  | 最低 Windows 版本。 示例：`10.0.19041.0` |
| `windows.msix.store` | bool | `false` | msix |  | 为 Microsoft Store 构建（不签名，由 Store 签名）。 示例：`true` |
| `windows.msix.sign` | bool | `true` | msix |  | 是否签名 MSIX（false 时必须设置 publisher）。 示例：`false` |
| `windows.msix.certificate` | path | `windows.sign.certificate，否则使用 msix 测试证书` | msix |  | 签名 MSIX 用的证书（.pfx）。 示例：`C:/certs/codesign.pfx` |
| `windows.msix.certificate_password` | string | `windows.sign.password` | msix |  | 证书密码。 示例：`${WINDOWS_CERT_PASSWORD}` |
| `windows.msix.extra_args` | list（或单个字符串） | — | msix |  | 追加到 dart run msix:create 的参数。 示例：`[--trim-logo, "false"]` |

### 2.10 `linux`

Linux：linux（tar.gz）、deb、rpm、appimage。

| 键 | 类型 | 默认值 | 目标 | 环境变量 / 参数 | 说明 |
| --- | --- | --- | --- | --- | --- |
| `linux.package_name` | string | `应用名转小写并用 - 连接` | deb, rpm, appimage |  | deb/rpm 包名，以及 /usr/bin 中的命令名。 示例：`xuehua-im` |
| `linux.prefix` | string | `/opt/<package_name>` | deb, rpm |  | 应用文件的安装目录。 示例：`/usr/lib/xuehua-im` |
| `linux.icon` | path | `flutter_launcher_icons 的图片，否则 web/icons/Icon-512.png` | deb, rpm, appimage |  | 菜单与 AppImage 使用的 PNG 图标。 示例：`assets/icon/icon.png` |
| `linux.icon_sizes` | list of int | `[16, 32, 48, 64, 128, 256, 512]` | deb, rpm |  | 安装到 hicolor 主题的图标尺寸（由 linux.icon 缩放，不放大）。 示例：`[48, 128, 256]` |
| `linux.categories` | list（或单个字符串） | `[Utility]` | deb, rpm, appimage |  | freedesktop.org 菜单分类（.desktop 的 Categories=）。 示例：`[Network, InstantMessaging]` |
| `linux.generic_name` | string | — | deb, rpm, appimage |  | .desktop 的 GenericName=。 示例：`Instant Messenger` |
| `linux.keywords` | list（或单个字符串） | — | deb, rpm, appimage |  | .desktop 的 Keywords=（搜索关键词）。 示例：`[chat, im, message]` |
| `linux.mime_types` | list（或单个字符串） | — | deb, rpm, appimage |  | .desktop 的 MimeType=（可打开的文件类型 / URL 协议）。 示例：`[x-scheme-handler/xuehua]` |
| `linux.startup_wm_class` | string | `可执行文件名` | deb, rpm, appimage |  | .desktop 的 StartupWMClass=（让窗口与启动器图标归为一组）。 示例：`xue_hua_im` |
| `linux.metainfo` | path | — | deb, rpm, appimage |  | AppStream metainfo，安装到 /usr/share/metainfo（软件中心展示）。 示例：`linux/packaging/com.xuehua.im.metainfo.xml` |
| `linux.appimagetool` | path | `PATH 中的 appimagetool` | appimage |  | appimagetool 路径。 示例：`~/.local/bin/appimagetool` |
| `linux.extra_args` | list（或单个字符串） | — | linux, deb, rpm, appimage |  | 追加到 flutter build linux 的参数。 示例：`[--no-tree-shake-icons]` |

#### `linux.deb`

Debian/Ubuntu 软件包。

| 键 | 类型 | 默认值 | 目标 | 环境变量 / 参数 | 说明 |
| --- | --- | --- | --- | --- | --- |
| `linux.deb.depends` | list（或单个字符串） | `[libgtk-3-0 \| libgtk-3-0t64]` | deb |  | Depends（依赖）。 示例：`[libgtk-3-0, libsecret-1-0]` |
| `linux.deb.recommends` | list（或单个字符串） | — | deb |  | Recommends（推荐）。 示例：`[gnome-keyring]` |
| `linux.deb.suggests` | list（或单个字符串） | — | deb |  | Suggests（建议）。 示例：`[libnotify-bin]` |
| `linux.deb.conflicts` | list（或单个字符串） | — | deb |  | Conflicts（冲突）。 示例：`[xuehua-im-beta]` |
| `linux.deb.section` | string | `utils` | deb |  | Section（分区）。 示例：`net` |
| `linux.deb.priority` | `required` \\| `important` \\| `standard` \\| `optional` \\| `extra` | `optional` | deb |  | Priority（优先级）。 示例：`optional` |
| `linux.deb.preinst` | path | — | deb |  | 解包前执行的维护脚本。 示例：`linux/packaging/preinst` |
| `linux.deb.postinst` | path | — | deb |  | 安装后执行的维护脚本。 示例：`linux/packaging/postinst` |
| `linux.deb.prerm` | path | — | deb |  | 卸载前执行的维护脚本。 示例：`linux/packaging/prerm` |
| `linux.deb.postrm` | path | — | deb |  | 卸载后执行的维护脚本。 示例：`linux/packaging/postrm` |

#### `linux.rpm`

Fedora/RHEL/openSUSE 软件包。

| 键 | 类型 | 默认值 | 目标 | 环境变量 / 参数 | 说明 |
| --- | --- | --- | --- | --- | --- |
| `linux.rpm.requires` | list（或单个字符串） | `[gtk3]` | rpm |  | Requires（依赖）。 示例：`[gtk3, libsecret]` |
| `linux.rpm.group` | string | `Applications/Internet` | rpm |  | Group（分组）。 示例：`Applications/Communications` |
| `linux.rpm.license` | string | `app.license` | rpm |  | License（许可证）。 示例：`MIT` |
| `linux.rpm.pre` | path | — | rpm |  | %pre 脚本。 示例：`linux/packaging/pre.sh` |
| `linux.rpm.post` | path | — | rpm |  | %post 脚本。 示例：`linux/packaging/post.sh` |
| `linux.rpm.preun` | path | — | rpm |  | %preun 脚本。 示例：`linux/packaging/preun.sh` |
| `linux.rpm.postun` | path | — | rpm |  | %postun 脚本。 示例：`linux/packaging/postun.sh` |

#### `linux.appimage`

AppImage。

| 键 | 类型 | 默认值 | 目标 | 环境变量 / 参数 | 说明 |
| --- | --- | --- | --- | --- | --- |
| `linux.appimage.update_information` | string | — | appimage |  | 内嵌更新信息（供 AppImageUpdate 使用）。 示例：`gh-releases-zsync\|xuehua\|im\|latest\|*x86_64.AppImage.zsync` |
| `linux.appimage.extra_args` | list（或单个字符串） | — | appimage |  | 追加到 appimagetool 的参数。 示例：`[--comp, zstd]` |

### 2.11 `web`

Web：web（zip）。

| 键 | 类型 | 默认值 | 目标 | 环境变量 / 参数 | 说明 |
| --- | --- | --- | --- | --- | --- |
| `web.base_href` | string | `保持 index.html 原样` | web | `--base-href` | <base href>；必须以 / 开头和结尾。 示例：`/app/` |
| `web.wasm` | bool | `false` | web | `--wasm` | 编译为 WebAssembly（带 JS 回退）。 示例：`true` |
| `web.source_maps` | bool | `false` | web |  | 生成 source map。 示例：`true` |
| `web.csp` | bool | `false` | web |  | 不动态生成代码（满足 CSP 限制）。 示例：`true` |
| `web.optimization_level` | int | `4` | web |  | dart2js / dart2wasm 优化级别 0-4（-O）。 示例：`2` |
| `web.static_assets_url` | url | — | web |  | 从其他域名加载静态资源（必须以 / 结尾）。 示例：`https://cdn.example.com/app/` |
| `web.web_resources_cdn` | bool | `true` | web |  | false：打包 CanvasKit，不从 CDN 加载。 示例：`false` |
| `web.web_define` | map | — | web |  | --web-define：web/index.html 中的模板变量。 示例：`{API_URL: https://api.example.com}` |
| `web.extra_args` | list（或单个字符串） | — | web |  | 追加到 flutter build web 的参数。 示例：`[--dump-info]` |

<!-- END GENERATED KEYS -->

### 2.12 补充说明

**Android 签名**：通过 Android Gradle 插件标准的 `android.injected.signing.*` 属性（`ORG_GRADLE_PROJECT_*` 环境变量）注入，不修改 Gradle 文件；密码不会出现在命令行、日志或 `--dry-run` 输出中。构建前用 `keytool` 校验密码和别名，构建后用 `apksigner verify --print-certs` / `keytool -printcert` 校验产物，并在结果中显示签名者与签名方案（v1/v2/v3/v4）。设置了 `android.signing.v1`–`v4` 中任意一项时，fpack 会用 `apksigner sign` 以这些方案重新签名每个 APK（密码通过环境变量传给 apksigner）；`v4: true` 时同时输出 `<apk>.idsig`。

**iOS ExportOptions.plist**：设置了 `ios.team_id`、`signing_style`、`provisioning_profiles`、`upload_symbols`、`manage_app_version_and_build_number`、`destination`、`thinning`、`strip_swift_symbols`、`export_options` 中任意一项（且没有设置 `ios.export_options_plist`）时，fpack 会在构建前生成 `build/fpack/ExportOptions.plist` 并传给 `flutter build ipa --export-options-plist`；`--dry-run` 会显示这一步。`destination: upload` 时 Xcode 直接上传到 App Store Connect，本地不保留 IPA。

**macOS 公证凭证**（三选一，优先级从高到低）：`notary_profile`（钥匙串配置，推荐本机使用）→ `notary_api_key` + `notary_api_key_id` (+ `notary_api_issuer`)（App Store Connect API 密钥，推荐 CI）→ `notary_apple_id` + `notary_team_id` + `notary_password`（App 专用密码，会被隐藏）。设置了任意一种凭证即视为开启公证（前提是已签名）。`hardened_runtime: false` 只适合不公证的内部分发。

**公证等待**（`macos.notarize`）：Apple 公证通常需要几分钟。fpack 先执行 `xcrun notarytool submit <文件> … --output-format json`（不带 `--wait`），拿到提交 ID 后立即写入 `<输出目录>/NOTARIZATION.md`（语言随 `--lang`）和 `notarization.json`，并打印路径。记录内容：产物与 SHA-256、提交 ID、提交时间、使用的凭证（钥匙串配置名 / API 密钥 ID / Apple ID；**不会写入任何密码**，Apple ID 方式的命令中密码写作 `$FPACK_NOTARY_PASSWORD`）、可直接复制的命令（`xcrun notarytool info/wait/log`、`xcrun stapler staple/validate`、`spctl -a -vv`）以及对应的 `fpack notarize status/finish`。

- `wait: true`（默认）：执行 `xcrun notarytool wait <ID>`，每 30 秒显示一次已等待时间，并提示「公证通常需要几分钟。可以按 Ctrl-C 停止等待 —— Apple 端会继续处理；之后可用 <路径> 中的命令查询。」。Ctrl-C 时打印同样的提示，已上传的文件保留在产物路径（尚未装订），退出码 130。结果为 Accepted 时继续装订、`spctl` 评估，记录更新为 `stapled`；结果为 Invalid 时下载 Apple 的日志到 `notary-log-<文件>.json`，总结问题（未用 Developer ID 签名、未启用 Hardened Runtime、缺少安全时间戳、带 `get-task-allow`、内嵌代码未签名……）并给出修复建议。
- `wait: false`（`FPACK_NOTARIZE_WAIT=false`、`--notarize-no-wait`）：提交并写入记录后就结束。汇总表中该产物标注「已提交公证」，`--json` 中 `notarization.state` 为 `submitted`；产物尚未装订（用户首次打开时 Gatekeeper 会联网验证）。
- `fpack notarize status [目录|ID]`：读取记录（默认使用 `dist/` 下最新的 `notarization.json`），用 `notarytool info` 查询每个未完成提交的状态并更新记录。
- `fpack notarize finish [目录]`：等待未完成的提交；Accepted 的 DMG/pkg 直接装订，App zip 会解压、装订 .app、校验后重新压缩；启用校验和时重写 `SHA256SUMS`；Invalid 的下载并总结日志。退出码：0 全部完成，1 有提交被拒绝或查询失败，3 找不到记录，130 被中断。
- `NOTARIZATION.md`、`notarization.json`、`notary-log-*.json` 不会写入校验和文件。

**DMG 布局**：`background`、`volume_icon`、`window_position`、`window_size`、`icon_size`、`app_position`、`applications_position`、`license` 需要 [create-dmg](https://github.com/create-dmg/create-dmg)（`brew install create-dmg`）；`tool: auto` 时装了 create-dmg 就会使用，没装则用 `hdiutil` 并警告这些设置被忽略。`format` / `filesystem` 两种工具都支持。

**pkg 安装包**（`fpack build pkg`）：复制 .app →（配置了 `identity` 时）Developer ID 重新签名 → `pkgbuild --root … --component-plist …`（安装到 `install_location`；`relocatable: false` 时升级总是覆盖该位置；`preinstall`/`postinstall` 会放入 `--scripts` 目录并设为可执行）→ `productbuild --distribution …`（支持 arm64 + x86_64，不提示 Rosetta；`min_os` 写入 `allowed-os-versions`，`require_restart` 写入 `onConclusion="RequireRestart"`；`installer_identity` 存在时 `--sign … --timestamp`）→ `pkgutil --check-signature` →（签名且开启公证时）`notarytool submit` → 等待 → `stapler staple` → `spctl --assess --type install`（公证流程见下文「公证等待」）。同一次运行中 `macos`、`dmg`、`pkg` 共享一次 `flutter build macos`。

fpack 不读取 `pubspec.yaml` 中 [`dmg`](https://pub.dev/packages/dmg) 包的 `dmg:` 段；需要签名/公证时请在 `fpack.yaml` 的 `macos.sign` 中配置（或使用 `FPACK_MACOS_*` 环境变量、命令行参数）。

**Inno Setup**：fpack 生成的脚本包含 `AppId`、发布者/网址、版权、`DefaultDirName`、开始菜单、桌面快捷方式任务、许可/信息页、图标与向导图片、`MinVersion`、权限模式（`privileges`：`user` 免 UAC、`admin` 所有用户、`ask` 让用户选择，静默安装可加 `/CURRENTUSER` 或 `/ALLUSERS`）以及多语言。`languages` 的第一个是默认语言，多个语言时显示语言选择框；`zh-CN`/`zh-TW` 在 Inno Setup 6.5+ 中自带，旧版本时 fpack 会写入自带的官方翻译文件（取自 jrsoftware/issrc）。自定义 `script` 时 fpack 仍通过 `/D` 传入 `AppName`、`AppVersion`、`AppPublisher`、`AppExeName`、`SourceDir`、`AppId`、`AppURL`。

**Windows 签名**：设置 `windows.sign.certificate`（.pfx）或 `thumbprint`（证书存储）后：`flutter build windows` 完成后立即用 signtool 签名 `build/windows/…/<app>.exe`（windows zip、exe、msix 都包含已签名的 exe）；Inno Setup 通过 `SignTool=fpack` 签名安装程序和卸载程序；msix 使用同一证书（`--certificate-path` / `--signtool-options`）。密码不会显示在日志或 `--dry-run` 中。

**MSIX**：`windows.msix.*` 会转换成 `dart run msix:create` 的参数，优先于 pubspec 的 `msix_config:`；未设置的 `display_name`、`publisher_display_name`、`identity_name`、`description` 在 `msix_config` 也没有时取 `app.*`。fpack 始终加 `--install-certificate false`，避免在 CI 中卡在提问。

**Linux 软件包**：安装到 `linux.prefix`（默认 `/opt/<包名>`），`/usr/bin/<包名>` 为符号链接；`.desktop` 文件包含 `Name`、`GenericName`、`Comment`、`Categories`、`Keywords`、`MimeType`、`StartupWMClass`；图标按 `icon_sizes` 缩放到 `/usr/share/icons/hicolor/<N>x<N>/apps/`（不会放大），另放一份到 `/usr/share/pixmaps/`；`metainfo` 安装到 `/usr/share/metainfo/`；`/usr/share/doc/<包名>/copyright` 写入 `app.copyright` 和 `app.license`。deb 的维护脚本、rpm 的 `%pre/%post/%preun/%postun` 来自对应的文件。AppImage 的 `.desktop` 额外包含 `X-AppImage-Version`，`update_information` 会嵌入 AppImage（`appimagetool --updateinformation`）。

**钩子**：`hooks.pre_build` 在第一个 flutter build 之前运行一次（失败则停止构建）；`hooks.pre_package.<目标>` / `hooks.post_package.<目标>` 在该目标打包前后运行；`hooks.post_build` 在最后运行一次（即使有目标失败）。命令在项目根目录用 `sh -c`（Windows 为 `cmd /C`）执行，可用环境变量：`FPACK_PROJECT_ROOT`、`FPACK_OUTPUT_DIR`、`FPACK_VERSION`、`FPACK_BUILD_NUMBER`、`FPACK_MODE`、`FPACK_FLAVOR`；打包钩子另有 `FPACK_TARGET`、`FPACK_ARTIFACT`（第一个产物）、`FPACK_ARTIFACTS`（换行分隔）；`post_build` 另有 `FPACK_ARTIFACTS` 与 `FPACK_SUCCESS`（`1`/`0`）。注意 `${VAR}` 会在加载 fpack.yaml 时被替换，钩子运行时才有的变量请写成 `$FPACK_ARTIFACT`（不带花括号）。`--dry-run` 会列出所有钩子命令。

**Web**：Flutter 3.x 已移除 `--pwa-strategy` 与 `--web-renderer`，因此没有对应的键；`web.wasm`、`source_maps`、`csp`、`optimization_level`、`static_assets_url`、`web_resources_cdn`、`web_define` 分别对应 `flutter build web` 的同名参数。zip 文件名可用 `output.names.web` 修改。


### 2.13 获取 macOS 签名证书与公证凭证

在 App Store 之外分发 macOS 应用需要：**Developer ID Application** 证书（签名 .app / zip / DMG）、可选的 **Developer ID Installer** 证书（签名 pkg），以及一种**公证凭证**。都需要付费的 [Apple Developer Program](https://developer.apple.com/programs/) 账号；Developer ID 证书只能由团队的 **Account Holder** 创建。

**1. 创建 Developer ID 证书（在 Mac 上）**

- Xcode → Settings… → Accounts → 选择团队 → Manage Certificates… → 左下角 **+** → *Developer ID Application*（需要 pkg 时再建 *Developer ID Installer*）。证书和私钥会直接进入「登录」钥匙串。
- 或者在网页上创建：钥匙串访问 → 证书助理 → 从证书颁发机构请求证书…（保存 CSR 到磁盘）→ [developer.apple.com/account/resources/certificates](https://developer.apple.com/account/resources/certificates/list) → **+** → Developer ID Application / Developer ID Installer → 上传 CSR → 下载 `.cer` 并双击导入（必须在生成 CSR 的那台 Mac 上导入，私钥在那里）。
- 检查：

  ```bash
  security find-identity -v -p codesigning   # "Developer ID Application: Your Name (ABCDE12345)"
  security find-identity -v -p basic         # 也会列出 "Developer ID Installer: …"
  ```

- 填入 fpack（引号内的名称与上面输出完全一致；括号中的 10 位字符就是 **Team ID**）：

  | 用途 | fpack.yaml | 环境变量 | 参数 |
  | --- | --- | --- | --- |
  | App 签名证书 | `macos.sign.identity` | `FPACK_MACOS_SIGN_IDENTITY` | `--sign-identity` |
  | pkg 签名证书 | `macos.sign.installer_identity` | `FPACK_MACOS_INSTALLER_IDENTITY` | `--installer-identity` |

**2. 公证凭证（三选一）**

*a) 钥匙串配置（本机推荐）*：Apple ID + App 专用密码 + Team ID，只需保存一次。

1. 在 [account.apple.com](https://account.apple.com) → 登录与安全 → **App 专用密码** → 生成一个密码（形如 `abcd-efgh-ijkl-mnop`）。
2. Team ID：[developer.apple.com/account](https://developer.apple.com/account) → 会员资格详细信息（Membership details），或证书名称括号中的 10 位字符。
3. 保存到钥匙串（不写 `--password` 时会提示输入）：

   ```bash
   xcrun notarytool store-credentials fpack-notary \
     --apple-id you@example.com --team-id ABCDE12345 --password abcd-efgh-ijkl-mnop
   xcrun notarytool history --keychain-profile fpack-notary   # 验证
   ```

4. fpack：`macos.sign.notary_profile: fpack-notary`（或 `FPACK_MACOS_NOTARY_PROFILE`、`--notary-profile`）。

也可以不保存配置，直接给出 Apple ID：`notary_apple_id` / `notary_team_id` / `notary_password`（`FPACK_NOTARY_APPLE_ID`、`FPACK_NOTARY_TEAM_ID`、`FPACK_NOTARY_PASSWORD`）。密码只放在环境变量或 CI secret 中，fpack 在日志、`--dry-run` 和 NOTARIZATION.md 中都会隐藏它。

*b) App Store Connect API 密钥（CI 推荐）*：不依赖个人 Apple ID，可随时吊销。

1. [App Store Connect](https://appstoreconnect.apple.com) → 用户和访问 → 集成 → App Store Connect API → 团队密钥 → **+**，选择 Developer（或更高）权限。
2. 下载 `AuthKey_<KEY_ID>.p8`（**只能下载一次**，请妥善保存）；记下页面上的 **Key ID** 和 **Issuer ID**（个人密钥没有 Issuer ID，可省略）。
3. fpack：

   | 值 | fpack.yaml | 环境变量 |
   | --- | --- | --- |
   | `.p8` 文件路径 | `macos.sign.notary_api_key` | `FPACK_NOTARY_API_KEY` |
   | Key ID | `macos.sign.notary_api_key_id` | `FPACK_NOTARY_API_KEY_ID` |
   | Issuer ID | `macos.sign.notary_api_issuer` | `FPACK_NOTARY_API_ISSUER` |

   验证：`xcrun notarytool history --key AuthKey_ABC123DEF4.p8 --key-id ABC123DEF4 --issuer <Issuer ID>`。

优先级：`notary_profile` → API 密钥 → Apple ID；设置任意一种即开启公证。

**3. 导出证书（.p12）用于 CI**

1. 钥匙串访问 → 「登录」钥匙串 → **我的证书** → 选中 “Developer ID Application: …”（展开能看到私钥，说明私钥在本机）→ 文件 → 导出项目… → 格式选「个人信息交换 (.p12)」→ 设置导出密码。需要 pkg 时对 “Developer ID Installer: …” 重复一次（也可以两个一起选中导出到同一个 .p12）。
2. 转成 base64 存为 CI secret（例如 `MACOS_CERTS_P12_BASE64`、`MACOS_CERTS_P12_PASSWORD`），API 密钥同理（`NOTARY_API_KEY_P8_BASE64`）：

   ```bash
   base64 -i DeveloperID.p12 | pbcopy
   base64 -i AuthKey_ABC123DEF4.p8 | pbcopy
   ```

3. 在 CI 机器上导入到临时钥匙串（GitHub Actions 示例）：

   ```yaml
   - name: import Developer ID certificates
     env:
       P12_BASE64: ${{ secrets.MACOS_CERTS_P12_BASE64 }}
       P12_PASSWORD: ${{ secrets.MACOS_CERTS_P12_PASSWORD }}
       KEYCHAIN_PASSWORD: ${{ secrets.KEYCHAIN_PASSWORD }}   # 任意随机字符串
       API_KEY_BASE64: ${{ secrets.NOTARY_API_KEY_P8_BASE64 }}
     run: |
       KEYCHAIN="$RUNNER_TEMP/signing.keychain-db"
       echo "$P12_BASE64" | base64 --decode > "$RUNNER_TEMP/certs.p12"
       security create-keychain -p "$KEYCHAIN_PASSWORD" "$KEYCHAIN"
       security set-keychain-settings -lut 21600 "$KEYCHAIN"
       security unlock-keychain -p "$KEYCHAIN_PASSWORD" "$KEYCHAIN"
       security import "$RUNNER_TEMP/certs.p12" -k "$KEYCHAIN" -P "$P12_PASSWORD" \
         -T /usr/bin/codesign -T /usr/bin/productbuild -T /usr/bin/pkgbuild
       security set-key-partition-list -S apple-tool:,apple: -s -k "$KEYCHAIN_PASSWORD" "$KEYCHAIN"
       security list-keychains -d user -s "$KEYCHAIN" $(security list-keychains -d user | tr -d '"')
       security find-identity -v -p codesigning "$KEYCHAIN"
       echo "$API_KEY_BASE64" | base64 --decode > "$RUNNER_TEMP/AuthKey.p8"
       rm "$RUNNER_TEMP/certs.p12"
   - name: fpack build macos dmg pkg
     env:
       FPACK_MACOS_SIGN_IDENTITY: "Developer ID Application: Your Name (ABCDE12345)"
       FPACK_MACOS_INSTALLER_IDENTITY: "Developer ID Installer: Your Name (ABCDE12345)"
       FPACK_NOTARY_API_KEY: ${{ runner.temp }}/AuthKey.p8
       FPACK_NOTARY_API_KEY_ID: ${{ secrets.NOTARY_API_KEY_ID }}
       FPACK_NOTARY_API_ISSUER: ${{ secrets.NOTARY_API_ISSUER }}
     run: fpack build macos dmg pkg
   - name: clean up keychain
     if: always()
     run: security delete-keychain "$RUNNER_TEMP/signing.keychain-db" || true
   ```

   `set-key-partition-list` 让 codesign / productbuild 无需弹窗即可使用私钥；`-lut 21600` 让钥匙串 6 小时内不自动锁定。证书、.p12、.p8 与密码都不要提交到仓库。

---

## 3. 环境变量

### 3.1 构建设置（覆盖 fpack.yaml，被命令行覆盖）

每个变量对应一个 fpack.yaml 键（自动生成）。布尔值接受 `true/false/1/0/yes/no`；列表用逗号分隔。

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

另外：

| 变量 | 说明 |
| --- | --- |
| `FPACK_CONFIG` | 配置文件路径（代替项目根目录的 `fpack.yaml`）。 |
| `FPACK_ANDROID_KEYSTORE_BASE64` | keystore 文件内容的 base64（适合 CI secrets）。仅在没有设置 keystore 路径时使用；fpack 会写入 `build/fpack/secrets/`（权限 0600），构建结束后删除。 |

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
| `FPACK_REBUILD` | 未设置 | 设为 `1` 时忽略缓存，先用本机 Go 从包内源码编译核心（开发 fpack 本身时用）。 |
| `FPACK_GO` | 自动查找（PATH、`/usr/local/go/bin`、`/opt/homebrew/bin` 等） | 回退编译时使用的 Go；设为 `none` 表示禁止本机编译（下载失败时直接报错）。 |
| `FPACK_NO_DOWNLOAD` | 未设置 | 设为 `1` 时禁止从 GitHub Release 下载核心（离线/内网：使用包内二进制或直接本机编译）。 |
| `FPACK_DOWNLOAD_URL` | `https://github.com/Matkurban/fpack/releases/download/v<版本>/` | 核心下载地址（内网镜像）。目录下需要 `checksums.txt` 和 `fpack-core-<os>-<arch>[.exe]`，下载后会校验 SHA-256。 |

核心的查找顺序（**优先使用经过校验的预编译二进制**）：`FPACK_CORE` → 缓存（版本匹配时）→ 包内预编译二进制（按 `prebuilt/manifest.json` 校验 SHA-256）→ 从 GitHub Release（或 `FPACK_DOWNLOAD_URL`）下载并按 `checksums.txt` 校验 SHA-256 → **只有下载失败或被禁用时**才用本机 Go 编译包内源码，此时会提示 `fpack: note: no verified prebuilt core available (<原因>); fpack: falling back to a local build from the bundled sources with <Go 版本> (<路径>)`。`FPACK_REBUILD=1` 时先本机编译。

本机编译出的核心会被缓存并在之后的运行中使用，直到版本变化；想换成下载的二进制，删除缓存目录即可（`fpack --wrapper-info` 显示路径、下载地址、Go 版本与查找顺序）。

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

目标：`apk`、`aab`、`ipa`、`macos`、`dmg`、`pkg`、`windows`、`exe`、`msix`、`linux`、`deb`、`rpm`、`appimage`、`web`。

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
| `--sign` | 开关 | `macos.sign.enabled: true` | macos, dmg, pkg | Developer ID 签名。 |
| `--no-sign` | 开关 | `macos.sign.enabled: false` | macos, dmg, pkg | 不签名（App、DMG、pkg 都不签名，同时关闭公证）。 |
| `--sign-identity ID` | string | `macos.sign.identity` | macos, dmg, pkg | App 的 codesign 证书（Developer ID Application）。 |
| `--installer-identity ID` | string | `macos.sign.installer_identity` | pkg | pkg 签名证书（Developer ID Installer）。 |
| `--notarize` | 开关 | `macos.sign.notarize: true` | macos, dmg, pkg | 公证并装订（zip、DMG 与已签名的 pkg）。 |
| `--no-notarize` | 开关 | `macos.sign.notarize: false` | macos, dmg, pkg | 跳过公证（不上传到 Apple，本地构建更快）。 |
| `--notarize-no-wait` | 开关 | `macos.notarize.wait: false` | macos, dmg, pkg | 提交公证后不等待结果（之后用 `fpack notarize finish`）。 |
| `--notary-profile NAME` | string | `macos.sign.notary_profile` | macos, dmg, pkg | notarytool 钥匙串配置名。 |
| `--dmg-tool T` | string | `macos.dmg.tool` | dmg | `auto` / `hdiutil` / `create-dmg`。 |
| `--base-href PATH` | string | `web.base_href` | web | base href。 |
| `--wasm` | 开关 | `web.wasm: true` | web | WebAssembly 构建。 |
| `-- …` | — | `*.extra_args` 之后 | 全部 | `--` 之后的所有参数原样传给 `flutter build`。 |

### 4.3 其他命令

| 命令 | 参数 | 说明 |
| --- | --- | --- |
| `fpack doctor [目标…]` | — | 按目标检查前置条件（Flutter、Android SDK/JDK、签名、Xcode、证书、CocoaPods、Inno Setup、dpkg-deb…），每个问题都给出修复方法。不指定目标时检查所有目标。 |
| `fpack list`（`ls`、`targets`） | — | 列出所有目标、产物格式，以及本机能否构建。 |
| `fpack init` | `-f`, `--force`：覆盖已有的 fpack.yaml；`-y`：不提问，使用检测到的默认值；`--lang zh\|en`：注释语言 | 生成带注释的 `fpack.yaml`：列出**所有**键（可选的保持注释），每个键都有说明、可选值、默认值和示例，并预填从项目检测到的值（应用 ID、flavor、版本、团队 ID、GUID…）。只问几个问题（默认目标、显示名、APK 拆分、keystore、iOS 导出方式、输出目录）。`fpack.yaml` 已存在且没有 `--force` 时不会覆盖，而是写入 `fpack.yaml.new` 并显示差异。 |
| `fpack schema` | `-o FILE`：写入文件 | 输出 fpack.yaml 的 JSON Schema（编辑器补全/校验）。 |
| `fpack notarize status [目录\|ID]` | `--json`：输出记录 | 查询 macOS 公证提交的状态并更新 `notarization.json` / `NOTARIZATION.md`。参数可以是输出目录、`notarization.json` 路径或提交 ID；省略时使用 `dist/` 下最新的记录。 |
| `fpack notarize finish [目录]` | `--json` | 等待未完成的提交，装订通过的 DMG/pkg/App zip，重写 SHA256SUMS，被拒绝时下载并总结 Apple 日志。可在 `--notarize-no-wait` 或 Ctrl-C 之后使用。 |
| `fpack clean` | `--dist`：同时删除输出目录（会先确认，`-y` 跳过确认）；`--flutter-clean`：同时运行 `flutter clean`；`--all`：等同 `--dist --flutter-clean`；`-n`, `--dry-run`：只显示将删除什么 | 默认只删除 fpack 的工作目录 `build/fpack/`（临时文件与日志）。 |
| `fpack version` | — | 显示 fpack、原生核心与启动器的版本。 |

### 4.4 退出码

| 码 | 含义 |
| --- | --- |
| 0 | 全部成功（`--dry-run`：计划生成成功） |
| 1 | 至少一个目标在构建/打包时失败，或钩子失败；`fpack notarize`：有提交被拒绝或查询失败 |
| 2 | 用法或配置错误（未知参数、未知目标、fpack.yaml 有误） |
| 3 | 前置条件不满足 / 产物已存在 / 没有可构建的目标 / 找不到 Flutter；`fpack notarize`：找不到公证记录 |
| 130 | 被 Ctrl-C 中断（公证等待被中断时，Apple 端仍会继续处理） |

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
| `{target}` | fpack 目标名 | `exe` |
| `{date}` | 构建日期（YYYYMMDD） | `20261009` |

`output.names` 可以为单个目标指定不同的模板，例如 `output.names: {exe: "{app}-setup-{version}", web: "{app}-web"}`。两个产物算出同一个文件名时，构建前会报错。

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
# yaml-language-server: $schema=https://raw.githubusercontent.com/Matkurban/fpack/main/schema/fpack.schema.json
# fpack.yaml —— 放在项目根目录（与 pubspec.yaml 同级）
# 所有键都可选；字符串中可用 ${VAR} / ${VAR:-默认值} 引用环境变量。

app:
  name: xue_hua_im                 # 产物文件名前缀（默认：pubspec 的 name）
  display_name: 雪花IM              # 安装程序 / 菜单中显示的名称
  description: 雪花即时通讯          # deb/rpm/.desktop 描述（默认：pubspec 的 description）
  publisher: XueHua                # 安装程序发布者（默认：windows/runner/Runner.rc 的 CompanyName）
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
  # checksum_algorithm: sha512     # 改为写入 SHA512SUMS
  # names: { web: "{app}-web" }    # 按目标单独设置文件名模板

hooks:
  pre_build: [dart run build_runner build --delete-conflicting-outputs]
  # post_package: { dmg: ['./scripts/upload.sh "$FPACK_ARTIFACT"'] }

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
    # CI 中可改用 API 密钥：notary_api_key / notary_api_key_id / notary_api_issuer
    # pkg 的签名证书（与上面的 Developer ID Application 是两张不同的证书）
    installer_identity: "Developer ID Installer: Your Name (TEAMID)"
  notarize:
    wait: true                     # false：只提交，之后 fpack notarize finish（或 --notarize-no-wait）
  pkg:
    # identifier: com.xuehua.im      # 默认取 macOS 工程的 bundle id
    min_os: "11.0"                 # 低于此版本的 macOS 拒绝安装
    install_location: /Applications
    title: 雪花IM
    # license: macos/installer/license.rtf
    # background: macos/installer/background.png
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
fpack build dmg --notarize-no-wait             # 只提交公证，立即结束
fpack notarize status                          # 查询最新一次的公证状态
fpack notarize finish                          # 等待结果、装订、更新 SHA256SUMS
fpack build macos dmg pkg                      # zip + DMG + pkg，只跑一次 flutter build
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
