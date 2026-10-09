---
title: "命令"
description: "fpack 的所有命令与参数。"
lang: zh-CN
---

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

## build 选项

| 选项 | 说明 |
| --- | --- |
| `-a, --all` | 构建本机能构建的全部目标，其余跳过并说明原因 |
| `-n, --dry-run` | 打印完整计划与精确命令，不执行任何操作 |
| `-m, --mode` / `--release` `--profile` `--debug` | 构建模式（默认 release） |
| `--flavor NAME` | Android productFlavor / Xcode scheme。名称不在检测到的 flavor 中时报错（退出码 3，并给出建议）；如果项目中未检测到任何 flavor，则只给出警告 |
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

## 每个命令的完整帮助

`fpack help <命令>` 的输出（由命令行程序生成，始终与发布版本一致）。

<!-- BEGIN GENERATED HELP -->
### `fpack build`

```text
构建并打包目标到 dist/

用法：
  fpack build [targets...] [options] [-- extra flutter args]

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
  --obfuscate                   混淆 Dart 代码（符号文件保存到 <输出目录>/debug-info）
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
  web       Web 构建（build/web 的 zip，可直接部署）
```

### `fpack doctor`

```text
按目标检查环境，并给出缺失项的修复命令

用法：
  fpack doctor [targets...]

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
  fpack schema [-o FILE]

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
  fpack notarize status [DIR|ID]
  fpack notarize finish [DIR]

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
