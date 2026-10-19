---
title: "命令"
description: "每个 fpack 命令和选项，附示例。"
---

```text
fpack build [目标…] [选项] [-- 传给 flutter 的参数]
fpack doctor [目标…]         检查每个目标的前置条件，并给出修复办法
fpack list                   列出所有目标、产物格式以及这台机器能否构建
fpack init                   写出列出所有键的 fpack.yaml，带中英文注释
fpack schema                 打印 fpack.yaml 的 JSON Schema
fpack notarize status|finish 查询或完成 macOS 公证提交
fpack clean                  删除 fpack 的临时文件（可选地也删除 dist/ 或 flutter 构建输出）
fpack version                版本信息（等同于 --version / -V）
fpack help <命令>            命令的帮助
```

别名：`build` = `b`，`doctor` = `check`，`list` = `ls` / `targets`。

## 全局选项

对所有命令都有效，写在命令前后都行。

| 选项 | 说明 |
| --- | --- |
| `-C`、`--project 目录` | Flutter 项目目录（默认：当前目录或含有 `pubspec.yaml` 的上级目录） |
| `--config 文件` | 配置文件（默认：项目里的 `fpack.yaml`；`FPACK_CONFIG`） |
| `--flutter SDK` | Flutter SDK 根目录（默认：FVM、`PATH`、`FLUTTER_ROOT` 等；见[用哪个 Flutter？](/guides/build-variants#用哪个-flutter)） |
| `--lang zh\|en` | 输出语言（默认取你的系统语言） |
| `-v`、`--verbose` | 输出工具的完整输出 |
| `--json` | 向 stdout 输出机器可读的 JSON 结果（[格式](/exit-codes#json-zh)） |
| `--no-color` | 关闭颜色 |
| `-y`、`--yes` | 默认回答“是”/使用默认值 |
| `-h`、`--help` | 显示帮助 |
| `-V`、`--version` | 打印版本 |
| `--wrapper-info` | 启动器诊断：使用的是哪个原生核心、来自哪里（[安装](/installation#launcher-zh)） |

**优先级**：命令行参数 > `FPACK_*` 环境变量 > `fpack.yaml` > 默认值。

## `fpack build`

构建并把目标打包到 `dist/`。`fpack build apk aab`、`fpack build --all`，不带目标时使用 `build.targets` 里的列表。

### 构建什么、怎么构建

| 选项 | 说明 | 指南 |
| --- | --- | --- |
| `-a`、`--all` | 构建这台机器能构建的每个目标；其余的注明原因并跳过 | [多目标](/guides/many-targets) |
| `-n`、`--dry-run` | 打印完整计划和确切命令，什么都不运行 | [多目标](/guides/many-targets#预演与查看) |
| `-m`、`--mode MODE`、`--release`、`--profile`、`--debug` | 构建模式（默认 `release`） | [构建模式](/guides/build-variants#构建模式) |
| `--flavor 名称` | Android productFlavor / Xcode scheme。名称未知会报错（退出码 3，并给出建议）；没有检测到 flavor 时只是警告 | [Flavor](/guides/build-variants#flavor) |
| `-t`、`--target 文件` | 入口文件，如 `lib/main_prod.dart` | [入口文件](/guides/build-variants#入口文件) |
| `--dart-define K=V`（可重复）、`--dart-define-from-file 文件`（可重复） | 编译期变量 | [编译期变量](/guides/build-variants#编译期变量) |
| `--build-name X.Y.Z`、`--build-number N` | 覆盖 pubspec 里的版本 | [版本](/guides/project-and-output#版本) |
| `--obfuscate`、`--split-debug-info 目录` | 混淆；符号默认放在 `<输出目录>/debug-info/<平台>` | [混淆](/guides/build-variants#混淆与调试符号) |
| `-o`、`--output 目录`、`-f`、`--force` | 输出目录；允许覆盖 | [输出](/guides/project-and-output#产物放在哪里) |
| `-- …` | `--` 之后的所有内容传给 `flutter build`，如 `-- --no-tree-shake-icons` | [Flutter 选项](/guides/build-variants#图标与其他-flutter-选项) |

### Android

| 选项 | 说明 |
| --- | --- |
| `--split-per-abi[=true\|both]` | 每个 ABI 一个 APK；`both` 同时保留通用 APK |
| `--abis 列表` | 要构建的 ABI，如 `arm64-v8a,armeabi-v7a` |

见 [Android](/platforms/android#apk-变体)。

### iOS

| 选项 | 说明 |
| --- | --- |
| `--export-method M` | `app-store-connect`、`app-store`、`release-testing`、`ad-hoc`、`development`、`debugging`、`enterprise` |
| `--export-options-plist 文件` | 使用你自己的 `ExportOptions.plist` |
| `--no-codesign` | 构建未签名的 IPA |

见 [iOS](/platforms/ios#其他分发方式)。

### macOS

| 选项 | 说明 |
| --- | --- |
| `--sign`、`--no-sign` | 用 Developer ID 给 app / DMG 签名；`--no-sign` 同时关闭公证 |
| `--sign-identity ID` | codesign 证书（Developer ID Application） |
| `--installer-identity ID` | `.pkg` 签名证书（Developer ID Installer） |
| `--notarize`、`--no-notarize` | 公证并装订 zip / DMG / pkg，或跳过（本地构建更快） |
| `--notarize-no-wait` | 提交后不等待就结束（之后用 `fpack notarize finish`） |
| `--notary-profile 名称` | `notarytool` 钥匙串配置 |
| `--provisioning-profile 文件` | 要嵌入的 Developer ID 描述文件 |
| `--smoke-test` | 启动已签名的应用，检查它是否一直在运行 |
| `--dmg-tool T` | `auto`、`hdiutil` 或 `create-dmg` |

见 [macOS](/platforms/macos#3-配置签名与公证)和[描述文件](/platforms/macos-profiles)。

### Web

| 选项 | 说明 |
| --- | --- |
| `--base-href 路径` | `<base href>`，如 `/app/` |
| `--wasm` | 用 WebAssembly 构建 |

见 [Web](/platforms/web)。

### 示例

```bash
fpack build apk aab --flavor prod --dart-define-from-file env/prod.json
fpack build apk --split-per-abi=both
fpack build ipa --export-method ad-hoc
fpack build ipa --no-codesign                # 未签名的 IPA（Payload 布局）
fpack build macos dmg                        # 按 fpack.yaml 的 macos.sign 签名/公证（未配置则不签名）
fpack build dmg --no-notarize                # 快速的本地包
fpack build dmg --notarize-no-wait           # 提交后退出；之后：fpack notarize finish
fpack build macos dmg pkg                    # 一次 flutter 构建，得到 zip + DMG + pkg 安装包
fpack build --all --json > result.json
fpack -C apps/client build web --base-href /app/
```

## `fpack doctor`

```bash
fpack doctor                  # 所有目标
fpack doctor apk ipa dmg      # 只检查这几个
fpack doctor --json
```

按目标检查前置条件，并给出缺失项的修复办法：Flutter、Dart、项目、Android SDK、许可和 Java、签名配置和密钥库、Xcode 和 CocoaPods、iOS 团队和证书、Developer ID 证书、描述文件和 entitlements、Visual Studio、Inno Setup、`signtool`、Linux 工具链和打包工具、带 LXD 或 Multipass 的 snapcraft。发现问题时退出码为 1。

## `fpack list`

```bash
fpack list                    # 也可以：fpack ls、fpack targets
fpack list --json
```

一张表，列出每个目标的平台、输出扩展名，以及这台机器能否构建（以及为什么不能）。

## `fpack init`

```bash
fpack init                    # 交互式
fpack init --yes              # 接受检测到的默认值
fpack init --force            # 覆盖已有的 fpack.yaml
```

写出一个带注释的 `fpack.yaml`，列出每个键，并预填检测到的值。如果 `fpack.yaml` 已存在，则写出 `fpack.yaml.new` 并显示差异；`--force` 覆盖。`-y` / `--yes` 不询问，直接接受默认值。

## `fpack schema`

```bash
fpack schema                              # 打印到 stdout
fpack schema -o .vscode/fpack.schema.json # 写入文件
```

打印 `fpack.yaml` 的 JSON Schema（[JSON Schema](/schema)）。

## `fpack notarize`

```bash
fpack notarize status                  # 最新 notarization.json 里的每个提交
fpack notarize status dist/1.2.0+5     # 指定的输出目录
fpack notarize status 2efe2717-52ef-…  # 单个提交 ID
fpack notarize finish                  # 等待、给通过的文件装订、更新 SHA256SUMS
```

`status` 向 Apple 查询 `<输出目录>/notarization.json`（上传后立即写入）里记录的每个提交的状态。`finish` 等待未完成的提交，给通过的 DMG/pkg/app zip 装订票据，重写校验和文件，并对被拒绝的提交下载并总结 Apple 的日志。凭证来自 `macos.sign`（或记录下来的钥匙串配置、API 密钥；用 Apple ID 提交需要 `FPACK_NOTARY_PASSWORD`）。见 [macOS：等待公证](/platforms/macos#5-等待公证)。

## `fpack clean`

```bash
fpack clean                   # 删除 build/fpack
fpack clean --dist --yes      # 同时不询问地删除 dist/
fpack clean --all             # --dist + --flutter-clean
fpack clean --dry-run         # 显示将删除什么
```

| 选项 | 说明 |
| --- | --- |
| `--dist` | 同时删除输出目录（会先询问） |
| `--flutter-clean` | 同时运行 `flutter clean` |
| `--all` | `--dist` + `--flutter-clean` |
| `-n`、`--dry-run` | 显示将删除什么 |

## `fpack version` 和 `fpack help`

`fpack version`（或 `fpack --version`）打印 fpack 版本以及核心的平台和 Go 版本（例如 `… (core linux/amd64, go1.27.2)`）。`fpack help <命令>` 打印下面的帮助。

## 每个命令的完整帮助

`fpack help <命令>` 的输出（由命令行程序生成，始终与发布版本一致）。

<!-- BEGIN GENERATED HELP -->
### `fpack build`

```text
构建并打包目标到 dist/

用法：
  fpack build [目标...] [选项] [-- 额外的 flutter 参数]

选项：
  -a, --all                     构建本机能构建的所有目标（其余会被跳过并说明原因）
  -n, --dry-run                 只打印计划和具体命令，不执行
  -m, --mode MODE               release（默认）| profile | debug
  --release                     等同 --mode release
  --profile                     等同 --mode profile
  --debug                       等同 --mode debug
  --flavor NAME                 构建 flavor（Android productFlavor / Xcode scheme）
  -t, --target FILE             入口文件，例如 lib/main_prod.dart
  --dart-define K=V             编译期变量（可重复）
  --dart-define-from-file FILE  包含 define 的 JSON/.env 文件（可重复）
  --build-name X.Y.Z            覆盖 pubspec.yaml 中的版本名
  --build-number N              覆盖 pubspec.yaml 中的构建号
  --split-per-abi[=true|both]   APK：按 ABI 拆分；=both 同时保留通用包
  --abis LIST                   Android ABI 列表，例如 arm64-v8a,armeabi-v7a
  --obfuscate                   混淆 Dart 代码（符号文件保存到 <输出目录>/debug-info/<平台>）
  --split-debug-info DIR        调试符号保存目录
  -o, --output DIR              输出目录（默认 dist/{version}{+build}）
  -f, --force                   覆盖已存在的产物
  --export-method M             iOS：app-store | ad-hoc | development | enterprise 等
  --export-options-plist FILE   iOS：ExportOptions.plist
  --no-codesign                 iOS：构建未签名 IPA
  --sign                        macOS：使用 Developer ID 签名 App/DMG
  --no-sign                     macOS：不签名（同时关闭公证）
  --sign-identity ID            macOS：codesign 证书名（Developer ID Application）
  --installer-identity ID       macOS：.pkg 签名证书（Developer ID Installer）
  --notarize                    macOS：公证并装订 zip/DMG/pkg
  --no-notarize                 macOS：跳过公证（本地构建更快）
  --notarize-no-wait            macOS：提交公证后不等待结果直接结束（之后：fpack notarize finish）
  --provisioning-profile FILE   macOS：要嵌入的 Developer ID 描述文件
  --smoke-test                  macOS：启动签名后的 App 并检查能否正常运行
  --notary-profile NAME         macOS：notarytool 钥匙串配置名
  --dmg-tool T                  macOS：auto | hdiutil | create-dmg
  --base-href PATH              web：base href，例如 /app/
  --wasm                        web：使用 WebAssembly 构建

全局选项：
  -h, --help         显示帮助
  -C, --project DIR  Flutter 项目目录（默认：当前目录或其上级）
  --config FILE      配置文件（默认：项目中的 fpack.yaml）
  --flutter SDK      Flutter SDK 根目录（默认：FVM、PATH、FLUTTER_ROOT 等）
  --lang zh|en       输出语言（默认：根据系统语言）
  -v, --verbose      输出完整的工具日志
  --json             向 stdout 输出机器可读的 JSON 结果
  --no-color         关闭颜色
  -y, --yes          所有提问使用默认值 / 自动确认

示例：
  fpack build apk
  fpack build apk aab --flavor prod --dart-define-from-file env/prod.json
  fpack build apk --split-per-abi=both
  fpack build ipa --export-method ad-hoc
  fpack build ipa --no-codesign
  fpack build macos dmg                 # 使用 fpack.yaml 中 macos.sign 的 Developer ID 签名/公证配置
  fpack build dmg --no-notarize
  fpack build macos dmg pkg             # 一次 flutter 构建，三个产物
  fpack build --all --json > result.json
  fpack build web --base-href /app/ -- --no-web-resources-cdn

目标：
  apk       Android APK（通用包和/或按 ABI 拆分）
  aab       Android App Bundle（用于 Google Play）
  ipa       iOS IPA（App Store / Ad Hoc / 企业 / 未签名）
  macos     macOS .app，用 ditto 压缩（可选 Developer ID 签名）
  dmg       macOS 磁盘镜像（hdiutil 或 create-dmg，可签名 + 公证）
  pkg       macOS 安装包（pkgbuild + productbuild，可用 Developer ID Installer 签名 + 公证）
  windows   Windows 绿色版文件夹（zip）
  exe       Windows 安装程序 .exe（Inno Setup）
  msix      Windows MSIX 包（需要 msix 开发依赖）
  linux     Linux 应用目录（tar.gz）
  deb       Debian/Ubuntu 安装包（dpkg-deb）
  rpm       Fedora/RHEL/openSUSE 安装包（rpmbuild）
  appimage  Linux AppImage（appimagetool）
  snap      Snap 包（snapcraft）
  web       Web 构建（build/web 的 zip，可直接部署）
```

### `fpack doctor`

```text
按目标检查环境，并给出缺失项的修复命令

用法：
  fpack doctor [目标...]

全局选项：
  -h, --help         显示帮助
  -C, --project DIR  Flutter 项目目录（默认：当前目录或其上级）
  --config FILE      配置文件（默认：项目中的 fpack.yaml）
  --flutter SDK      Flutter SDK 根目录（默认：FVM、PATH、FLUTTER_ROOT 等）
  --lang zh|en       输出语言（默认：根据系统语言）
  -v, --verbose      输出完整的工具日志
  --json             向 stdout 输出机器可读的 JSON 结果
  --no-color         关闭颜色
  -y, --yes          所有提问使用默认值 / 自动确认

示例：
  fpack doctor
  fpack doctor apk ipa dmg
  fpack doctor --json
```

### `fpack list`

```text
列出所有目标、输出格式，以及本机能构建哪些

用法：
  fpack list

全局选项：
  -h, --help         显示帮助
  -C, --project DIR  Flutter 项目目录（默认：当前目录或其上级）
  --config FILE      配置文件（默认：项目中的 fpack.yaml）
  --flutter SDK      Flutter SDK 根目录（默认：FVM、PATH、FLUTTER_ROOT 等）
  --lang zh|en       输出语言（默认：根据系统语言）
  -v, --verbose      输出完整的工具日志
  --json             向 stdout 输出机器可读的 JSON 结果
  --no-color         关闭颜色
  -y, --yes          所有提问使用默认值 / 自动确认

示例：
  fpack list
  fpack list --json
```

### `fpack init`

```text
为项目生成带注释的 fpack.yaml（交互式，或用 --yes 使用默认值）

用法：
  fpack init [--yes] [--force]

选项：
  -f, --force  覆盖已有的 fpack.yaml

全局选项：
  -h, --help         显示帮助
  -C, --project DIR  Flutter 项目目录（默认：当前目录或其上级）
  --config FILE      配置文件（默认：项目中的 fpack.yaml）
  --flutter SDK      Flutter SDK 根目录（默认：FVM、PATH、FLUTTER_ROOT 等）
  --lang zh|en       输出语言（默认：根据系统语言）
  -v, --verbose      输出完整的工具日志
  --json             向 stdout 输出机器可读的 JSON 结果
  --no-color         关闭颜色
  -y, --yes          所有提问使用默认值 / 自动确认

示例：
  fpack init
  fpack init --yes
```

### `fpack schema`

```text
输出 fpack.yaml 的 JSON Schema（用于编辑器补全与校验）

用法：
  fpack schema [-o 文件]

选项：
  -o, --output FILE  写入 FILE 而不是标准输出

全局选项：
  -h, --help         显示帮助
  -C, --project DIR  Flutter 项目目录（默认：当前目录或其上级）
  --config FILE      配置文件（默认：项目中的 fpack.yaml）
  --flutter SDK      Flutter SDK 根目录（默认：FVM、PATH、FLUTTER_ROOT 等）
  --lang zh|en       输出语言（默认：根据系统语言）
  -v, --verbose      输出完整的工具日志
  --json             向 stdout 输出机器可读的 JSON 结果
  --no-color         关闭颜色
  -y, --yes          所有提问使用默认值 / 自动确认

示例：
  fpack schema -o .vscode/fpack.schema.json
  # fpack init 会在 fpack.yaml 顶部加入下面这行，VS Code / IntelliJ（YAML 插件）会自动使用：
  # yaml-language-server: $schema=https://raw.githubusercontent.com/Matkurban/fpack/main/schema/fpack.schema.json
```

### `fpack notarize`

```text
查询或完成 macOS 公证提交（Ctrl-C 或 --notarize-no-wait 之后）

用法：
  fpack notarize status [目录|ID]
  fpack notarize finish [目录]

全局选项：
  -h, --help         显示帮助
  -C, --project DIR  Flutter 项目目录（默认：当前目录或其上级）
  --config FILE      配置文件（默认：项目中的 fpack.yaml）
  --flutter SDK      Flutter SDK 根目录（默认：FVM、PATH、FLUTTER_ROOT 等）
  --lang zh|en       输出语言（默认：根据系统语言）
  -v, --verbose      输出完整的工具日志
  --json             向 stdout 输出机器可读的 JSON 结果
  --no-color         关闭颜色
  -y, --yes          所有提问使用默认值 / 自动确认

示例：
  fpack notarize status                  # 最新 notarization.json 中的所有提交
  fpack notarize status dist/1.2.0+5     # 指定输出目录
  fpack notarize status 2efe2717-52ef-…  # 指定提交 ID
  fpack notarize finish                  # 等待、装订已通过的文件、更新 SHA256SUMS

status 向 Apple 查询 <输出目录>/notarization.json（fpack build 上传后立即写入）
中记录的每个提交的状态。finish 等待未完成的提交，为已通过的 DMG/pkg/App zip
装订票据（App 会先解压、装订再重新压缩），重写校验和文件；对被拒绝的提交下载并
总结 Apple 的日志。凭证来自 fpack.yaml 的 macos.sign（或记录中的钥匙串配置 /
API 密钥；Apple ID 提交需要设置 FPACK_NOTARY_PASSWORD）。
```

### `fpack clean`

```text
删除 fpack 的临时文件（可选删除 dist/ 或 flutter 构建产物）

用法：
  fpack clean [--dist] [--flutter-clean] [--all]

选项：
  --dist           同时删除输出目录（会先确认）
  --flutter-clean  同时运行 `flutter clean`
  --all            等同 --dist + --flutter-clean
  -n, --dry-run    只显示将删除的内容

全局选项：
  -h, --help         显示帮助
  -C, --project DIR  Flutter 项目目录（默认：当前目录或其上级）
  --config FILE      配置文件（默认：项目中的 fpack.yaml）
  --flutter SDK      Flutter SDK 根目录（默认：FVM、PATH、FLUTTER_ROOT 等）
  --lang zh|en       输出语言（默认：根据系统语言）
  -v, --verbose      输出完整的工具日志
  --json             向 stdout 输出机器可读的 JSON 结果
  --no-color         关闭颜色
  -y, --yes          所有提问使用默认值 / 自动确认

示例：
  fpack clean
  fpack clean --dist --yes
  fpack clean --all
```

### `fpack version`

```text
显示版本信息

用法：
  fpack version

全局选项：
  -h, --help         显示帮助
  -C, --project DIR  Flutter 项目目录（默认：当前目录或其上级）
  --config FILE      配置文件（默认：项目中的 fpack.yaml）
  --flutter SDK      Flutter SDK 根目录（默认：FVM、PATH、FLUTTER_ROOT 等）
  --lang zh|en       输出语言（默认：根据系统语言）
  -v, --verbose      输出完整的工具日志
  --json             向 stdout 输出机器可读的 JSON 结果
  --no-color         关闭颜色
  -y, --yes          所有提问使用默认值 / 自动确认

示例：
  fpack --version
```
<!-- END GENERATED HELP -->
