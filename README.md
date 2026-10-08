# fpack

[English](README.en.md) · 中文

**一条命令，把 Flutter 项目打包成所有平台的发布文件。**
Android（APK / 按 ABI 拆分 APK / AAB）、iOS（IPA）、macOS（.app zip / DMG / pkg 安装包，可签名 + 公证）、Windows（zip / Inno Setup 安装包 / MSIX）、Linux（tar.gz / deb / rpm / AppImage）、Web（zip）。

```bash
dart pub global activate fpack    # 安装（也支持 git / 本地路径，见下文）
cd my_flutter_app
fpack doctor                      # 这台机器能打哪些包？缺什么？怎么装？
fpack build apk aab               # → dist/1.0.0+1/my_app-1.0.0+1-android-universal.apk …
fpack build --all                 # 本机能打的全部打出来，打不了的说明原因并跳过
```

- **零配置可用**：直接读取 `pubspec.yaml`、Gradle、Xcode 工程中的信息；`fpack init` 生成的 `fpack.yaml` 完全可选。
- **绝不修改你的项目文件**：只调用 Flutter 工具链和打包工具。签名信息通过环境变量注入，不改 Gradle。唯一会写入的文件是 `fpack init` 生成的 `fpack.yaml`，产物写到 `dist/`，临时文件写到 `build/fpack/`。
- **原生核心**：核心用 Go 编写并编译为原生二进制（启动约 10 ms，无运行时依赖）；Dart 包只是一个很薄的启动器，负责找到/准备与之版本完全一致的二进制。
- **一切可配置**：约 175 个 fpack.yaml 键（安装包元数据、签名、DMG 布局、Inno Setup 语言/权限、MSIX、deb/rpm 元数据与脚本、钩子、文件名模板……），全部可用 `FPACK_*` 环境变量覆盖；配置会被严格校验（未知键提示「你是不是想写 …」、类型错误指出行号），并提供 JSON Schema 供编辑器补全。
- **为人设计**：彩色输出与进度动画（CI 中自动降级为纯文本）、失败时给出关键错误摘录 + 修复建议 + 完整日志路径、`--dry-run` 精确展示每一条将执行的命令、`--json` 机器可读输出、中英文自动切换。

---

## 目录

- [安装](#安装)
- [快速开始](#快速开始)
- [命令](#命令)
- [目标与产物](#目标与产物)
- [配置 fpack.yaml](#配置-fpackyaml)
- [签名](#签名)（[Android](#android-签名) · [iOS](#ios-导出与签名) · [macOS](#macos-签名与公证)）
- [CI 示例](#ci-示例)
- [环境变量](#环境变量)
- [退出码](#退出码)
- [常见问题](#常见问题)
- [架构](#架构)
- [开发与发布](#开发与发布)

---

## 安装

需要 Dart 3.8+（Flutter 自带）。三种方式任选：

| 方式 | 命令 | 说明 |
| --- | --- | --- |
| pub.dev | `dart pub global activate fpack` | 推荐（发布后） |
| git | `dart pub global activate --source git https://github.com/Matkurban/fpack` | 首次运行时用本机 Go 编译核心，或从 GitHub Release 下载 |
| 本地路径 | `dart pub global activate --source path /path/to/fpack` | 离线可用：发布包内已带 6 个平台的预编译二进制 |

确保 `~/.pub-cache/bin`（Windows：`%LOCALAPPDATA%\Pub\Cache\bin`）在 `PATH` 中：

```bash
echo 'export PATH="$PATH:$HOME/.pub-cache/bin"' >> ~/.zshrc && source ~/.zshrc
fpack --version
```

### 启动器如何找到原生核心

`fpack` 每次运行都会按以下顺序找一个**版本与 Dart 包完全一致**的 `fpack-core`（用 `fpack-core --core-version` 校验）：

1. `FPACK_CORE` 环境变量（开发调试用）
2. 用户缓存：macOS `~/Library/Caches/fpack/<版本>/<os>-<arch>/`，Linux `$XDG_CACHE_HOME/fpack/…`（默认 `~/.cache/fpack`），Windows `%LOCALAPPDATA%\fpack\…`（可用 `FPACK_HOME` 覆盖）
3. 包内预编译二进制 `prebuilt/<os>-<arch>/`：先用 `prebuilt/manifest.json` 中的 SHA-256 校验，再复制到缓存
4. 用本机 Go 从包内源码编译（`go build -trimpath`，依赖已 vendor，无需联网）
5. 从 GitHub Release 下载，并用 `checksums.txt` 校验 SHA-256，校验失败绝不执行

`fpack --wrapper-info` 可查看每一步的路径和最终使用的二进制。

> **pub.dev 包体积取舍**：6 个平台的二进制合计约 23 MB（压缩后约 10 MB）。打包进 pub 包 → 安装即可离线使用，但每个用户都要下载所有平台的二进制；不打包 → 包很小（约 200 KB），首次运行时用本机 Go 编译（约 30 秒）或从 GitHub Release 下载当前平台的约 4 MB 文件。发布工作流用仓库变量 `BUNDLE_BINARIES` 控制；GitHub Release 中的 `fpack-vX.Y.Z.tar.gz` 始终包含全部二进制，适合离线/内网使用。

---

## 快速开始

```bash
cd my_flutter_app
fpack doctor                     # 检查环境，逐项给出安装/修复命令
fpack build apk --dry-run        # 先看看会执行哪些命令（什么都不执行）
fpack build apk                  # 真正打包
fpack init                       # （可选）生成带注释的 fpack.yaml，交互式；--yes 使用默认值
```

产物默认在 `dist/<版本>+<构建号>/`，同时生成 `SHA256SUMS`（可用 `shasum -a 256 -c SHA256SUMS` 校验）：

```
dist/1.0.0+1/
  my_app-1.0.0+1-android-universal.apk
  my_app-1.0.0+1-android-arm64-v8a.apk
  my_app-1.0.0+1-android.aab
  my_app-1.0.0+1-ios-arm64.ipa
  my_app-1.0.0+1-macos-universal.dmg
  my_app-1.0.0+1-macos.pkg
  SHA256SUMS
```

默认**不会覆盖**已存在的产物：请提升版本号、使用 `--build-number N`，或加 `--force`。

---

## 命令

```
fpack build [目标…] [选项] [-- 透传给 flutter 的参数]
fpack doctor          检查每个目标的前置条件，并给出修复方法
fpack list            列出所有目标、产物格式，以及本机能否构建
fpack init            生成列出全部键、带中文/英文注释的 fpack.yaml（-y/--yes 用检测到的默认值；
                      已存在时写入 fpack.yaml.new 并显示差异，--force 才覆盖）
fpack schema          输出 fpack.yaml 的 JSON Schema（-o FILE 写入文件）
fpack notarize status [目录|ID]   查询 macOS 公证提交的状态（读取 dist/…/notarization.json）
fpack notarize finish [目录]      等待公证结果、装订票据、更新 SHA256SUMS
fpack clean           删除 build/fpack 临时文件；--dist 同时删除 dist/；--flutter-clean 运行 flutter clean；--all 全部
fpack version         版本信息（--version / -V 同义）
fpack help <命令>     查看命令帮助
```

全局选项：`-C, --project DIR`、`--config FILE`、`--flutter SDK`、`--lang zh|en`、`-v, --verbose`、`--json`、`--no-color`、`-y, --yes`。

### build 选项

| 选项 | 说明 |
| --- | --- |
| `-a, --all` | 构建本机能构建的全部目标，其余跳过并说明原因 |
| `-n, --dry-run` | 打印完整计划与精确命令，不执行任何操作 |
| `-m, --mode` / `--release` `--profile` `--debug` | 构建模式（默认 release） |
| `--flavor NAME` | Android productFlavor / Xcode scheme |
| `-t, --target FILE` | 入口文件，例如 `lib/main_prod.dart` |
| `--dart-define K=V`（可重复） / `--dart-define-from-file FILE` | 编译期变量 |
| `--build-name X.Y.Z` / `--build-number N` | 覆盖 pubspec 中的版本 |
| `--split-per-abi[=true\|both]` / `--abis LIST` | APK 按 ABI 拆分；`both` 同时保留通用包 |
| `--obfuscate` / `--split-debug-info DIR` | 混淆；符号默认保存到 `<输出目录>/debug-info/<平台>` |
| `-o, --output DIR` / `-f, --force` | 输出目录 / 允许覆盖 |
| `--export-method M` / `--export-options-plist FILE` / `--no-codesign` | iOS 导出 |
| `--sign` `--no-sign` `--sign-identity ID` `--installer-identity ID` `--notarize` `--no-notarize` `--notarize-no-wait` `--notary-profile NAME` `--dmg-tool T` | macOS 签名（App / pkg）、公证（`--notarize-no-wait`：提交后不等待）、DMG 工具 |
| `--base-href PATH` / `--wasm` | Web |
| `-- …` | 之后的参数原样传给 `flutter build`，例如 `-- --no-tree-shake-icons` |

示例：

```bash
fpack build apk aab --flavor prod --dart-define-from-file env/prod.json
fpack build apk --split-per-abi=both
fpack build ipa --export-method ad-hoc
fpack build ipa --no-codesign                # 未签名 IPA（Payload 结构）
fpack build macos dmg                        # 按 fpack.yaml 的 macos.sign 签名/公证（未配置则不签名）
fpack build dmg --no-notarize                # 本地快速出包
fpack build dmg --notarize-no-wait           # 提交公证后立即结束，之后 fpack notarize finish
fpack build macos dmg pkg                    # zip + DMG + pkg 安装包，只跑一次 flutter build
fpack build --all --json > result.json
fpack -C apps/client build web --base-href /app/
```

**优先级**：命令行参数 > `FPACK_*` 环境变量 > `fpack.yaml` > 默认值。

**中断**：按一次 Ctrl-C 会优雅停止正在运行的 flutter/gradle/xcodebuild（整个进程组），再按一次强制结束；退出码 130。

**Monorepo**：在项目子目录中运行会向上查找 `pubspec.yaml`；在仓库根目录运行时会列出找到的 Flutter 项目，用 `-C <目录>` 选择。

**Flutter SDK 查找顺序**：`--flutter` → `FPACK_FLUTTER` → `fpack.yaml` 的 `flutter.sdk` → FVM（`.fvm/flutter_sdk`、`.fvmrc`）→ `FLUTTER_ROOT` → `PATH` → 常见安装位置（如 `~/develop/flutter`、`~/flutter`）。

---

## 目标与产物

文件名模板默认为 `{app}{-flavor}-{version}{+build}-{platform}{-arch}{-variant}{-mode}`（release 模式不带 mode）。

| 目标 | 需要的系统 | 产物 | 使用的工具 |
| --- | --- | --- | --- |
| `apk` | 任意 | `-android-universal.apk`、`-android-arm64-v8a.apk` … | flutter、apksigner（校验签名） |
| `aab` | 任意 | `-android.aab` | flutter、keytool（校验签名） |
| `ipa` | macOS | `-ios-arm64.ipa`（未签名：`-ios-arm64-unsigned.ipa`） | flutter/xcodebuild，`--no-codesign` 时用 ditto |
| `macos` | macOS | `-macos-universal.zip` | flutter、codesign、ditto、notarytool |
| `dmg` | macOS | `-macos-universal.dmg` | hdiutil 或 create-dmg、codesign、notarytool、stapler |
| `pkg` | macOS | `-macos.pkg` | pkgbuild + productbuild（安装到 /Applications）、Developer ID Installer 签名、notarytool、stapler |
| `windows` | Windows | `-windows-x64-portable.zip` | flutter |
| `exe` | Windows | `-windows-x64-setup.exe` | Inno Setup（ISCC） |
| `msix` | Windows | `-windows-x64.msix` | `msix` dev 依赖（`dart run msix:create`） |
| `linux` | Linux | `-linux-x64.tar.gz` | flutter |
| `deb` | Linux | `-linux-x64.deb` | dpkg-deb |
| `rpm` | Linux | `-linux-x64.rpm` | rpmbuild |
| `appimage` | Linux | `-linux-x64.AppImage` | appimagetool（`--all` 时缺失则跳过） |
| `web` | 任意 | `-web.zip` | flutter |

Flutter 不能跨系统编译 iOS/macOS/Windows/Linux 桌面应用。`fpack build --all` 会跳过本机不能构建的目标并说明原因；显式指定时（如在 Linux 上 `fpack build ipa`）会报错并提示用对应系统的 CI 机器。`--dry-run` 下仍会显示这类目标的参考计划。

同一次运行中共享 flutter 构建：`macos` + `dmg` + `pkg` 只构建一次，`linux` + `deb` + `rpm` + `appimage` 只构建一次，`windows` + `exe` + `msix` 只构建一次。

---

## 配置 fpack.yaml

> 📖 **完整配置参考**（每个 fpack.yaml 键、每个 `FPACK_*` 环境变量、每个命令行参数、优先级规则和完整示例）：[doc/configuration.md](doc/configuration.md)

所有键都是可选的，`fpack init` 会生成**列出全部键**的文件：每个键都有中文（`--lang en` 为英文）注释，说明作用、可选值、默认值、示例以及对应的环境变量/命令行参数，并预填从项目检测到的值（应用 ID、flavor、版本号、团队 ID、Inno AppId…）；不需要的键保持注释即可。

- **校验**：未知键（提示「你是不是想写 …」）、类型错误（`line 12: android.signing.v1: expected true or false, got "maybe"`）、可选值与取值范围、组合冲突；构建前检查当前目标用到的文件是否存在。
- **编辑器补全**：生成的文件第一行是 `# yaml-language-server: $schema=…/schema/fpack.schema.json`，VS Code（YAML 插件）/ JetBrains 会提供补全、悬停说明（中英文）和校验。`fpack schema -o fpack.schema.json` 可导出本地副本。
- **环境变量**：`${VAR}` / `${VAR:-默认值}` 在加载时替换；几乎每个键都有对应的 `FPACK_*` 变量（见[完整列表](doc/configuration.md#3-环境变量)）。

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

**钩子**（`hooks`）：`pre_build`（第一次 flutter build 前执行一次，失败则停止）、`pre_package.<目标>` / `post_package.<目标>`（打包前后，提供 `FPACK_TARGET`、`FPACK_ARTIFACT`、`FPACK_ARTIFACTS`）、`post_build`（最后执行一次，提供 `FPACK_ARTIFACTS`、`FPACK_SUCCESS`）。所有钩子都在项目根目录用 `sh -c`（Windows：`cmd /C`）执行，并提供 `FPACK_PROJECT_ROOT`、`FPACK_OUTPUT_DIR`、`FPACK_VERSION`、`FPACK_BUILD_NUMBER`、`FPACK_MODE`、`FPACK_FLAVOR`；钩子运行时才有的变量请写 `$VAR`（`${VAR}` 会在加载配置时被替换）。钩子失败时退出码为 1。

---

## 签名

### Android 签名

fpack **不修改 Gradle 文件**。它通过 Android Gradle 插件标准的 `android.injected.signing.*` 属性（以 `ORG_GRADLE_PROJECT_*` 环境变量传入）为 release 构建签名，密码不会出现在命令行、日志或 `--dry-run` 输出中（显示为 `***`）。

```bash
export FPACK_ANDROID_KEYSTORE=~/keys/upload-keystore.jks
export FPACK_ANDROID_KEYSTORE_PASSWORD='…'
export FPACK_ANDROID_KEY_ALIAS=upload
export FPACK_ANDROID_KEY_PASSWORD='…'       # 省略时与 keystore 密码相同
fpack build apk aab
```

- CI 中可以把 keystore 本身 base64 后放进 `FPACK_ANDROID_KEYSTORE_BASE64`，fpack 会写到 `build/fpack/secrets/`（权限 0600），构建后删除。
- 构建前用 `keytool` 校验密码和别名，错误会直接指出是密码错还是别名不存在。
- 构建后用 `apksigner` / `keytool -printcert` 校验产物签名并显示签名者；如果仍是 Android Debug 证书会给出警告。
- 如果项目自己已有 `key.properties` + `signingConfigs`，不配置 fpack 签名即可，fpack 会使用项目的配置。
- 未配置签名且 release 使用 debug 签名（`flutter create` 的默认值）时会提示：可用于测试，Google Play 会拒绝。

### iOS 导出与签名

```bash
fpack build ipa                                   # Flutter 默认（App Store Connect）
fpack build ipa --export-method ad-hoc
fpack build ipa --export-options-plist ios/ExportOptions.plist
fpack build ipa --no-codesign                     # 未签名 IPA，用于后续重签名
```

`doctor` 会检查 Xcode、CocoaPods（存在 Podfile 时）、工程中的 `DEVELOPMENT_TEAM`、签名证书，并在失败时把 Xcode 的关键错误（证书、描述文件、团队、Pod 等）翻译成具体的修复建议。

### macOS 签名与公证

用于 App Store 之外的分发（Developer ID）。**只**通过 fpack 自己的配置开启（低 → 高）：

1. `fpack.yaml` 的 `macos.sign`（`fpack init` 会生成带占位符的注释段，并在注释中列出本机钥匙串里的 Developer ID 证书）
2. 环境变量 `FPACK_MACOS_SIGN`、`FPACK_MACOS_SIGN_IDENTITY`、`FPACK_MACOS_INSTALLER_IDENTITY`、`FPACK_MACOS_NOTARIZE`、`FPACK_MACOS_NOTARY_PROFILE`
3. 命令行 `--sign/--no-sign`、`--sign-identity`、`--installer-identity`、`--notarize/--no-notarize`、`--notary-profile`

```yaml
macos:
  sign:
    identity: "Developer ID Application: Your Name (TEAMID)"   # 设置后即启用签名
    notary_profile: XueHua                                     # 设置后即启用公证
```

公证凭证三选一（优先级从高到低）：钥匙串配置 `notary_profile`（本机推荐）→ App Store Connect API 密钥 `notary_api_key` + `notary_api_key_id`（+ `notary_api_issuer`，CI 推荐）→ Apple ID `notary_apple_id` + `notary_team_id` + `notary_password`（App 专用密码，日志中隐藏）。签名默认启用 Hardened Runtime（`hardened_runtime`，公证必需）。

规则：设置了证书即启用签名；设置了任一公证凭证即启用公证；关闭签名同时关闭公证；`--sign` 不带证书时自动选用钥匙串中第一个 “Developer ID Application” 证书；指定的证书不存在时会列出可用证书。**什么都不配置时**，.app 保留 Xcode 工程自己的签名、DMG 不签名，产物说明中会提示如何配置。fpack 不读取 `pubspec.yaml` 中其他插件（如 [`dmg`](https://pub.dev/packages/dmg) 包的 `dmg:` 段）的配置。

DMG 流程：复制 .app → `codesign --deep` 签名内嵌代码 → 用 Hardened Runtime + `macos/Runner/Release.entitlements` 重新签名 App → 校验 → `hdiutil`（或 `create-dmg`，支持背景图、窗口大小/位置、图标大小与位置、卷图标、许可协议；`format`/`filesystem` 两种工具都支持）制作 DMG → 签名 DMG → `xcrun notarytool submit` → 等待结果 → `stapler staple` → `spctl` 评估。

**公证等待**：Apple 公证通常需要几分钟。上传完成、拿到提交 ID 后，fpack 立即在输出目录写入 `NOTARIZATION.md`（中英文随 `--lang`）和 `notarization.json`：产物路径与 SHA-256、提交 ID、提交时间、使用的凭证（钥匙串配置名 / API 密钥 ID / Apple ID，绝不写密码），以及可直接复制的命令（`xcrun notarytool info/wait/log …`、`xcrun stapler staple/validate …`、`spctl -a -vv …`、`fpack notarize status/finish`），并在终端打印该文件路径。等待期间显示已等待时间，并提示：*公证通常需要几分钟。可以按 Ctrl-C 停止等待 —— Apple 端会继续处理；之后可用 NOTARIZATION.md 中的命令查询。* 结果出来后文件会更新为最终状态（Accepted / Invalid）；被拒绝时 fpack 自动下载 Apple 的日志（`notary-log-<文件>.json`），总结问题并给出修复建议。

- **不等待**：`macos.notarize.wait: false`（或 `FPACK_NOTARIZE_WAIT=false`、`--notarize-no-wait`）只提交、写入记录后就结束，汇总和 `--json` 中该产物的公证状态为 `submitted`（文件尚未装订）。
- **之后继续**：`fpack notarize status [目录|ID]` 向 Apple 查询所有未完成的提交；`fpack notarize finish [目录]` 等待结果，对通过的 DMG/pkg 装订票据（App zip 会解压、装订 .app 后重新压缩），重写 `SHA256SUMS`，并更新记录；被拒绝的会下载并总结日志。Ctrl-C 中断等待后同样可用。

公证凭证只需创建一次：

```bash
xcrun notarytool store-credentials XueHua --apple-id you@example.com --team-id ABCDE12345
```

`fpack build macos` 产出签名的 zip；开启公证时 zip 也会公证（流程：提交 zip → 装订到 .app → 重新压缩）。

### macOS 安装包（pkg）

`fpack build pkg` 用 `pkgbuild` 生成把 App 安装到 `/Applications` 的组件包（不可重定位，升级时总是覆盖 /Applications 中的版本），再用 `productbuild` 生成分发包（支持 Apple silicon 与 Intel，不会提示安装 Rosetta）。可在 `macos.pkg` 中设置 `identifier`、`install_location`、`title` 以及安装界面的 `welcome` / `readme` / `license` / `conclusion`（.html/.rtf/.txt）和 `background`（图片）。

- **签名**：pkg 需要单独的 **“Developer ID Installer”** 证书（与签名 App 的 “Developer ID Application” 不同），通过 `macos.sign.installer_identity` / `FPACK_MACOS_INSTALLER_IDENTITY` / `--installer-identity` 配置；签名后自动用 `pkgutil --check-signature` 校验。未配置时生成未签名 pkg 并给出说明；`--no-sign` 同样关闭 pkg 签名。
- **App 签名**：配置了 `macos.sign.identity` 时，pkg 中的 App 与 zip/DMG 一样先用 Developer ID 重新签名。
- **公证**：与 DMG 共用公证配置；只有已签名的 pkg 才会提交公证并 `stapler staple`，最后用 `spctl --assess --type install` 评估。
- **更多选项**：`min_os`（低于此版本拒绝安装，默认取工程的 `MACOSX_DEPLOYMENT_TARGET`）、`require_restart`、`relocatable`、`preinstall` / `postinstall` 脚本（自动设为可执行）、`version`。
- `doctor` 会检查 pkgbuild/productbuild，并列出钥匙串中的安装包证书。

### Windows 签名与安装包

设置 `windows.sign.certificate`（.pfx）+ `password`，或 `thumbprint`（证书存储）后：`flutter build windows` 之后立即用 signtool 签名应用 .exe（zip、安装包、MSIX 中都是已签名的 exe），Inno Setup 安装程序和卸载程序通过 `SignTool=` 签名，MSIX 使用同一证书；时间戳服务器默认 `http://timestamp.digicert.com`，signtool 会自动在 Windows SDK 中查找。

Inno Setup 安装包可配置发布者/网址、版权、安装目录、开始菜单、桌面快捷方式、许可/信息页、图标与向导图片、最低 Windows 版本、`privileges`（`user` 免管理员 / `admin` / `ask`）以及多语言（`languages: [zh-CN, en]`，第一个为默认；旧版 Inno Setup 缺少的中文语言文件由 fpack 自带）。MSIX 的显示名、发布者、能力、文件关联、协议、版本等可直接在 `windows.msix` 中设置，优先于 pubspec 的 `msix_config`。

### Linux 软件包

deb/rpm/AppImage 安装到 `linux.prefix`（默认 `/opt/<包名>`，并在 `/usr/bin` 放符号链接），`.desktop` 文件包含 `Name`/`GenericName`/`Comment`/`Categories`/`Keywords`/`MimeType`/`StartupWMClass`，图标按 `icon_sizes` 缩放到 hicolor 主题，可附带 AppStream `metainfo` 和 `copyright`。deb 支持 Depends/Recommends/Suggests/Conflicts/Section/Priority 与维护脚本，rpm 支持 Requires/Group/License/URL 与 `%pre/%post/%preun/%postun`，AppImage 支持内嵌更新信息（`update_information`，同时在输出目录生成 `.zsync` 文件）。

---

## CI 示例

```yaml
jobs:
  android:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: subosito/flutter-action@v2
      - run: dart pub global activate fpack && echo "$HOME/.pub-cache/bin" >> $GITHUB_PATH
      - run: fpack build apk aab --split-per-abi=both
        env:
          FPACK_ANDROID_KEYSTORE_BASE64: ${{ secrets.KEYSTORE_BASE64 }}
          FPACK_ANDROID_KEYSTORE_PASSWORD: ${{ secrets.KEYSTORE_PASSWORD }}
          FPACK_ANDROID_KEY_ALIAS: upload
      - uses: actions/upload-artifact@v4
        with: { name: android, path: dist/ }
  apple:
    runs-on: macos-latest
    steps:
      - uses: actions/checkout@v4
      - uses: subosito/flutter-action@v2
      - run: dart pub global activate fpack && echo "$HOME/.pub-cache/bin" >> $GITHUB_PATH
      - run: fpack build ipa dmg --json > result.json
  windows:
    runs-on: windows-latest
    steps:
      - uses: actions/checkout@v4
      - uses: subosito/flutter-action@v2
      - run: dart pub global activate fpack
      - run: fpack build windows exe
  linux:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: subosito/flutter-action@v2
      - run: sudo apt-get install -y ninja-build libgtk-3-dev rpm
      - run: dart pub global activate fpack && echo "$HOME/.pub-cache/bin" >> $GITHUB_PATH
      - run: fpack build linux deb rpm web
```

非 TTY 环境下自动关闭动画与颜色，每 60 秒输出一次心跳，避免 CI 因长时间无输出而超时。

---

## 环境变量

完整说明（类型、默认值、影响的目标）见 [doc/configuration.md](doc/configuration.md#3-环境变量)。

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
| `FPACK_REBUILD=1` | 强制用 Go 重新编译核心（修改了 Go 源码时） |
| `FPACK_GO` | Go 可执行文件路径（`none` 表示不用 Go） |
| `FPACK_NO_DOWNLOAD=1` / `FPACK_DOWNLOAD_URL` | 禁止下载 / 自定义下载地址（内网镜像） |

---

## 退出码

| 码 | 含义 |
| --- | --- |
| 0 | 成功（`--dry-run`：计划成功生成） |
| 1 | 至少一个目标在构建/打包时失败，或钩子失败；`fpack notarize`：有提交被拒绝或查询失败 |
| 2 | 用法或配置错误（未知参数、未知目标、fpack.yaml 有误） |
| 3 | 前置条件不满足 / 产物已存在 / 没有可构建的目标 / 找不到 Flutter |
| 130 | 被 Ctrl-C 中断（公证等待被中断时 Apple 端仍会继续处理） |

`--json` 时 stdout 只输出 JSON 结果（目标、状态、产物路径、大小、SHA-256/512、公证状态 `notarization.state`、耗时、错误摘录、修复建议），所有人类可读输出走 stderr。

---

## 常见问题

**构建失败了怎么看？** fpack 会显示关键错误摘录（Gradle 的 “What went wrong”、Xcode 的 error 行等）、一条具体的修复建议，以及完整日志路径 `build/fpack/logs/…`。加 `-v` 可实时查看完整输出。

**path 依赖找不到**（例如 `xue_hua_sdk: path: ../xue_hua_sdk`）：fpack 会在构建前检查并指出缺少哪个包，请把它放到相对项目的对应位置。

**macOS 提示无法验证开发者**：fpack 复制核心二进制时会移除 quarantine 属性；如果仍被拦截，运行 `FPACK_REBUILD=1 fpack --version` 用本机 Go 重新编译。

**用本地路径安装时，每次运行开头都有 “Resolving dependencies…”**：这是 pub 对路径安装包的固定行为（git / pub.dev 安装没有）。如果要把 `--json` 输出交给其他程序，请用 git 或 pub.dev 安装，或过滤掉这几行。

**会不会改我的项目？** 不会。fpack 只读取项目文件；`flutter_launcher_icons.yaml`、`flutter_native_splash.yaml`、`distribute_options.yaml`、`package_rename_config.yaml`、`pubspec.yaml` 等都不会被修改。建议把 `dist/` 加入 `.gitignore`。

---

## 架构

```
fpack/
  bin/fpack.dart            Dart 入口（pub executable）
  lib/src/                  启动器：平台识别、核心解析（缓存/预编译/编译/下载）、SHA-256、信号转发
  go/                       原生核心（Go，依赖已 vendor）
    cmd/fpack-core/         main
    internal/cli            参数解析、命令、帮助
    internal/build          构建编排：分类 → 预检 → 共享 flutter 步骤 → 打包 → 校验和 → 汇总
    internal/targets        每个目标的预检、flutter 参数、产物定位、打包命令（dry-run 与真实执行同一份代码）
    internal/config         键注册表（驱动环境变量、校验、JSON Schema、init 模板与文档表格）、fpack.yaml 解析
    internal/project        pubspec / Gradle / Xcode 工程信息读取（只读）
    internal/flutter        Flutter SDK 查找（FVM / PATH / …）
    internal/runner         子进程（进程组、Ctrl-C、日志、密钥脱敏）
    internal/hints          常见错误 → 修复建议
    internal/ui, i18n, pack, host, doctor, version
  schema/                   fpack.schema.json（由键注册表生成）
  prebuilt/                 预编译二进制 + manifest.json（发布产物，不进 git）
  scripts/                  build_binaries.sh、check_versions.sh、package_dist.sh
  .github/workflows/        ci.yml、release.yml
```

## 开发与发布

```bash
cd go && go vet ./... && go test ./...      # Go 核心
FPACK_UPDATE=1 go test ./internal/config    # 修改键注册表后重新生成 schema 与 doc/configuration.md 表格
dart pub get && dart analyze && dart test   # Dart 启动器
scripts/build_binaries.sh                   # 交叉编译 6 个平台 → prebuilt/
scripts/package_dist.sh ../fpack-dist.tar.gz  # 自包含离线安装包
FPACK_CORE=$PWD/prebuilt/darwin-arm64/fpack-core fpack doctor   # 用指定核心调试
```

发布：同时修改 `pubspec.yaml`、`lib/src/version.dart`、`go/internal/version/version.go` 中的版本号（`scripts/check_versions.sh` 会检查），更新 `CHANGELOG.md`，推送 `vX.Y.Z` 标签。`release.yml` 会交叉编译、创建 GitHub Release（二进制 + `checksums.txt` + 离线包），并可选发布到 pub.dev。

## 许可证

MIT
