---
title: "配置"
description: "fpack.yaml：校验、环境变量插值、钩子以及全部键。"
lang: zh-CN
---

所有键都是可选的，`fpack init` 会生成**列出全部键**的文件：每个键都有中文（`--lang en` 为英文）注释，说明作用、可选值、默认值、示例以及对应的环境变量/命令行参数，并预填从项目检测到的值（应用 ID、flavor、版本号、团队 ID、Inno AppId…）；不需要的键保持注释即可。

- **校验**：未知键（提示「你是不是想写 …」）、类型错误（`line 12: android.signing.v1: expected true or false, got "maybe"`）、可选值与取值范围、组合冲突；构建前检查当前目标用到的文件是否存在。
- **编辑器补全**：生成的文件第一行是 `# yaml-language-server: $schema=…/schema/fpack.schema.json`，VS Code（YAML 插件）/ JetBrains 会提供补全、悬停说明（中英文）和校验。`fpack schema -o fpack.schema.json` 可导出本地副本。
- **环境变量**：任何值中的 `${VAR}` / `${VAR:-默认值}` 都会在加载时替换（`obfuscate: ${OBF:-false}` 也可以）；构建、签名、公证相关设置还有专门的 `FPACK_*` 变量（见[完整列表](/zh/environment)）。

```yaml
# yaml-language-server: $schema=https://raw.githubusercontent.com/Matkurban/fpack/main/schema/fpack.schema.json
app:
  display_name: 雪花IM
  publisher: XueHua Tech             # 默认：windows/runner/Runner.rc 的 CompanyName
  homepage: https://xuehua.example.com
  license: MIT

build:
  targets: [apk, aab, ipa, dmg]      # `fpack build` 不带目标时构建这些
  flavor: prod
  dart_define_from_file: [config/prod.json]

output:
  dir: "dist/{version}{+build}"
  names: { exe: "{app}-setup-{version}", web: "{app}-web" }   # 按目标单独命名；还可用 {target} {date}
  checksum_algorithm: sha512         # SHA512SUMS

hooks:
  pre_build: [dart run build_runner build --delete-conflicting-outputs]
  post_package: { dmg: ['./scripts/upload.sh "$FPACK_ARTIFACT"'] }

android:
  split_per_abi: both
  signing:
    store_file: ~/keys/upload.jks
    store_password: ${FPACK_ANDROID_KEYSTORE_PASSWORD}
    key_alias: upload
    v4: true                         # 设置 v1–v4 任意一项时用 apksigner 重新签名，v4 额外输出 .idsig

ios:
  export_method: ad-hoc
  team_id: ABCDE12345                # 设置后 fpack 自动生成 ExportOptions.plist
  provisioning_profiles: { com.xuehua.im: XueHua AdHoc }

macos:
  sign:
    identity: "Developer ID Application: XueHua Tech (ABCDE12345)"
    notary_profile: XueHua           # 或 notary_api_key/_id/_issuer，或 notary_apple_id/_team_id/_password
  notarize:
    wait: true                       # false：提交后不等待（fpack notarize finish 稍后装订）
  dmg: { background: macos/dmg/bg.png, window_size: [660, 400], format: ULFO }
  pkg: { min_os: "11.0", postinstall: macos/installer/postinstall.sh }

windows:
  inno_setup: { languages: [zh-CN, en], privileges: ask, desktop_icon: checked }
  sign: { certificate: C:/certs/codesign.pfx, password: ${WINDOWS_CERT_PASSWORD} }
  msix: { publisher: "CN=XueHua Tech", capabilities: [internetClient] }

linux:
  categories: [Network, InstantMessaging]
  metainfo: linux/com.xuehua.im.metainfo.xml
  deb: { depends: ["libgtk-3-0 | libgtk-3-0t64"], postinst: linux/postinst }
  rpm: { requires: [gtk3], license: MIT }

web:
  base_href: /app/
  source_maps: true
```

**钩子**（`hooks`）：`pre_build`（第一次 flutter build 前执行一次，失败则停止）、`pre_package.<目标>` / `post_package.<目标>`（打包前后，提供 `FPACK_TARGET`、`FPACK_ARTIFACT`、`FPACK_ARTIFACTS`）、`post_build`（最后执行一次，提供 `FPACK_ARTIFACTS`、`FPACK_SUCCESS` = `1`/`0`；构建失败后也会执行，但如果所有目标都未通过构建前检查（例如缺少工具、flavor 不存在）则跳过）。所有钩子都在项目根目录用 `sh -c`（Windows：`cmd /C`）执行，并提供 `FPACK_PROJECT_ROOT`、`FPACK_OUTPUT_DIR`、`FPACK_VERSION`、`FPACK_BUILD_NUMBER`、`FPACK_MODE`、`FPACK_FLAVOR`；钩子运行时才有的变量请写 `$VAR`（`${VAR}` 会在加载配置时被替换）。钩子失败时退出码为 1。

## 文件名模板

`output.name` 和 `output.names.<目标>` 支持以下占位符；`{-x}` / `{+x}` 只在值非空时才加上分隔符：

| 占位符 | 值 |
| --- | --- |
| `{app}` | `app.name`（默认：pubspec 的 `name`） |
| `{version}` / `{build}` | 版本号 / 构建号 |
| `{flavor}` | flavor（Flutter 不使用 flavor 的平台为空） |
| `{platform}` | `android`、`ios`、`macos`、`windows`、`linux`、`web` |
| `{arch}` | `arm64-v8a`、`universal`、`x64` … |
| `{variant}` | `setup`、`portable` … |
| `{mode}` | release 时为空，否则为 `profile` / `debug` |
| `{target}` | 目标名 |
| `{date}` | 构建日期 `YYYYMMDD` |

## 键参考

`fpack.yaml` 的全部键（由 fpack 的键注册表生成——与校验、JSON Schema、`fpack init` 同源）。

<!-- BEGIN GENERATED KEYS -->
### `app`

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

### `flutter`

Flutter SDK 选择。

| 键 | 类型 | 默认值 | 目标 | 环境变量 / 参数 | 说明 |
| --- | --- | --- | --- | --- | --- |
| `flutter.sdk` | path | `FLUTTER_ROOT、fvm、PATH 中的 flutter` | 全部 | `FPACK_FLUTTER`<br>`--flutter` | 使用的 Flutter SDK 根目录。 示例：`~/fvm/versions/stable` |

### `build`

所有 flutter build 共用的选项。

| 键 | 类型 | 默认值 | 目标 | 环境变量 / 参数 | 说明 |
| --- | --- | --- | --- | --- | --- |
| `build.targets` | list（或单个字符串） | — | — |  | 执行 `fpack build` 且不带目标时构建的目标。命令行接受的别名（如 `bundle`、`ios`、`setup`）这里同样可用。 示例：`[apk, aab, ipa, dmg]` |
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

### `output`

产物输出目录与命名。

| 键 | 类型 | 默认值 | 目标 | 环境变量 / 参数 | 说明 |
| --- | --- | --- | --- | --- | --- |
| `output.dir` | string | `dist/{version}{+build}` | 全部 | `FPACK_OUTPUT_DIR`<br>`-o, --output` | 产物目录（相对于项目，可用占位符）。 示例：`"dist/{version}{+build}"` |
| `output.name` | string | `{app}{-flavor}-{version}{+build}-{platform}{-arch}{-variant}{-mode}` | 全部 |  | 文件名模板（不含扩展名）。占位符：{app} {version} {build} {platform} {arch} {variant} {mode} {flavor} {target} {date}；{-x}/{+x} 表示 x 非空时才加分隔符。 示例：`"{app}-{version}-{platform}{-arch}"` |
| `output.names` | map | — | 全部 |  | 按目标单独设置文件名模板（目标 → 模板），优先于 output.name。 示例：`{exe: "{app}-setup-{version}", web: "{app}-web"}` |
| `output.overwrite` | bool | `false` | 全部 | `FPACK_OVERWRITE`<br>`-f, --force` | 已存在同名产物时覆盖，而不是停止。 示例：`true` |
| `output.checksums` | bool | `true` | 全部 |  | 在产物旁写入校验和文件。 示例：`false` |
| `output.checksum_algorithm` | `sha256` \\| `sha512` | `sha256` | 全部 |  | 校验算法（文件名 SHA256SUMS 或 SHA512SUMS）。 示例：`sha512` |

### `hooks`

构建前后执行的 shell 命令（工作目录：项目根目录）。

| 键 | 类型 | 默认值 | 目标 | 环境变量 / 参数 | 说明 |
| --- | --- | --- | --- | --- | --- |
| `hooks.pre_build` | list（或单个字符串） | — | 全部 |  | 第一次 flutter build 之前执行一次；失败则停止构建。 示例：`[dart run build_runner build --delete-conflicting-outputs]` |
| `hooks.post_build` | list（或单个字符串） | — | 全部 |  | 全部目标完成后执行一次（构建失败后也会执行，但所有目标都未通过构建前检查时跳过）；FPACK_ARTIFACTS 为产物列表（每行一个），FPACK_SUCCESS 为 1/0。 示例：`[./scripts/upload.sh]` |
| `hooks.pre_package` | map：目标 → 命令列表 | — | 全部 |  | 按目标（目标 → 命令）在打包步骤之前执行；提供 FPACK_TARGET。 示例：`{apk: [./scripts/check_size.sh]}` |
| `hooks.post_package` | map：目标 → 命令列表 | — | 全部 |  | 按目标在产物生成后执行；提供 FPACK_ARTIFACT（第一个产物）和 FPACK_ARTIFACTS。 示例：`{dmg: [./scripts/upload_dmg.sh "$FPACK_ARTIFACT"]}` |

### `android`

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

### `ios`

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

### `macos`

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

### `windows`

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

### `linux`

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

### `web`

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
