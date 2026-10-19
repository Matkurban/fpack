---
title: "第一次构建"
description: "十分钟从 Flutter 项目到第一个产物：doctor、预演、构建、看懂结果。"
---

本页一步一步带你从一个普通的 Flutter 项目走到发布文件。你需要已[安装](/installation)的 fpack 和一个 Flutter 项目；示例使用名为 `my_app` 的项目。不需要任何配置文件。

## 1. 进入你的项目

```bash
cd my_app
```

fpack 从当前目录向上查找，直到看到 `pubspec.yaml` 为止。你也可以在任何位置用 `fpack -C path/to/my_app …` 运行。

## 2. 问问这台机器能构建什么

```bash
fpack doctor
```

```text
fpack doctor  ·  fpack 1.1.16  ·  linux/amd64

General
  ✓ Flutter 3.47.7 (stable) · /home/me/flutter  [PATH]
  ✓ Dart 3.13.5
  ✓ project my_app 1.0.0+1 · /home/me/my_app
  • no fpack.yaml – zero-config defaults are used

Android  (apk, aab)
  ✓ Android SDK · /home/me/android-sdk
  ! no Java (JDK 17+) found
  → install Android Studio (bundles a JDK) or a JDK 17/21, then: flutter config --jdk-dir <path>

iOS  (ipa)
  – needs macOS (this is Linux)
…
Web  (web)
  ✓ no extra tools needed

✓ ready to build here: web
```

怎么看：`✓` 正常，`!` 警告（fpack 仍会尝试），`✗` 问题（该目标无法构建），`–` 在这个操作系统上不可能。每个问题后面有一行 `→`，给出修复命令。把你要的目标需要的东西修好，再运行一次 `doctor`。`fpack list` 用一张表显示同样的信息，包含所有目标。

::: tip
doctor 不会构建任何东西，随时运行都安全；每次构建前也会做同样的检查。
:::

## 3. 先预演一次

```bash
fpack build web --dry-run
```

```text
Plan (dry run – nothing will be executed)

▶ web  Web build (zip of build/web, ready to deploy)
  $ /home/me/flutter/bin/flutter build web --release
  · zip build/web
  · move to dist/1.0.0+1/my_app-1.0.0+1-web.zip
  = dist/1.0.0+1/my_app-1.0.0+1-web.zip  [Web (zip)]
```

计划列出每条命令的完整参数和每个将生成的文件，密码显示为 `***`。每次改了配置、想先看效果时，都可以用 `--dry-run`。

## 4. 构建

```bash
fpack build web
```

```text
▶ Web · web  flutter build web --release
  ✓ flutter build web (25.7s)
  ✓ dist/1.0.0+1/my_app-1.0.0+1-web.zip  13.8 MB

Summary
  TARGET  STATUS  FILE                                 SIZE     TIME
  web     ✓ done  dist/1.0.0+1/my_app-1.0.0+1-web.zip  13.8 MB  26.2s

Output:    dist/1.0.0+1
Checksums: dist/1.0.0+1/SHA256SUMS
Logs:      build/fpack/logs
```

把 `web` 换成你的机器已就绪的任何目标（`apk`、`macos`、`windows`、`deb` 等），或者列出多个：`fpack build apk aab`。每个目标都有自己的教程，链接在本页底部。

## 5. 看懂结果

一切都放在 `dist/<版本>+<构建号>/`，旁边还有校验和文件：

```text
dist/1.0.0+1/
  my_app-1.0.0+1-android-universal.apk
  my_app-1.0.0+1-android.aab
  my_app-1.0.0+1-macos-universal.dmg
  my_app-1.0.0+1-web.zip
  SHA256SUMS
```

版本来自 `pubspec.yaml`（`version: 1.0.0+1`：版本名 `1.0.0`，构建号 `1`）。随时可以验证文件：

::: tabs
@@ macOS
```bash
cd dist/1.0.0+1 && shasum -a 256 -c SHA256SUMS
```
@@ Linux
```bash
cd dist/1.0.0+1 && sha256sum -c SHA256SUMS
```
@@ Windows
```powershell
cd dist\1.0.0+1; Get-Content SHA256SUMS
Get-FileHash .\my_app-1.0.0+1-web.zip -Algorithm SHA256
```
:::

fpack 从不覆盖已有产物。同一版本构建两次会停在“already exists”：在 `pubspec.yaml` 里提高版本号、传 `--build-number 2`，或者传 `--force` 替换文件。

## 6. 创建 `fpack.yaml`（需要时）

调试签名的 Android 构建和普通桌面包不需要任何配置。需要安装程序元数据、签名、自定义文件名或钩子时：

```bash
fpack init          # 交互式；加 --yes 接受检测到的默认值
```

`fpack init` 会写出 `fpack.yaml`，把**每个**键都以注释列出，并预填从项目里检测到的值（应用 ID、flavor、版本、团队 ID、安装程序 AppId、钥匙串里的签名证书）。只取消注释你需要的；`fpack doctor` 和 `fpack build` 会带着行号和建议报告错误。以后再运行 `fpack init`，它会写 `fpack.yaml.new` 并显示差异，而不会动你的文件。

一个设置显示名称和默认目标的最小文件：

```yaml
app:
  display_name: My App
build:
  targets: [apk, aab, web]    # 不带参数的 `fpack build` 构建这些
```

现在直接运行 `fpack build` 就会构建这三个目标。

## 7. 试试示例项目

仓库里有一个完整的示例应用，带完整注释的 `fpack.yaml`（flavor、通过环境变量签名、DMG/pkg、Inno Setup/MSIX、deb/rpm/AppImage、web）：

```bash
git clone https://github.com/Matkurban/fpack
cd fpack/example
flutter pub get
fpack build --dry-run     # fpack 将运行的每一条命令
fpack build               # build.targets 里的目标
```

更多说明见[示例 README](https://github.com/Matkurban/fpack/tree/main/example)。

## 接下来

| 我想… | 去 |
| --- | --- |
| 发布 Android 应用（已签名的 APK / AAB，上架 Google Play） | [Android](/platforms/android) |
| 发布 iOS 应用（App Store、TestFlight、Ad Hoc） | [iOS](/platforms/ios) |
| 在 App Store 之外发布 Mac 应用（已签名、已公证的 DMG） | [macOS](/platforms/macos) |
| 制作 Windows 安装程序 | [Windows](/platforms/windows) |
| 构建 deb / rpm / AppImage / Snap | [Linux](/platforms/linux) |
| 部署 Web 构建 | [Web](/platforms/web) |
| 修改名称、版本或输出目录 | [名称、版本与输出](/guides/project-and-output) |
| 在 CI 里自动化 | [CI 示例](/ci) |
| 出错了 | [故障排查](/troubleshooting) |
