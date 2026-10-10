---
title: "快速开始"
description: "从安装到第一批发布产物。"
---

fpack 用一条命令把 Flutter 项目打包成所有平台的发布文件。本页带你在几分钟内打出第一批产物。

## 1. 安装

```bash
dart pub global activate fpack
fpack --version
```

确保 `~/.pub-cache/bin` 在 `PATH` 中（见[安装](/installation)）。

## 2. 检查本机环境

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

## 3. 试试示例项目

仓库中有一个完整的示例应用，带一份详细注释的 `fpack.yaml`（flavor、通过环境变量签名、DMG/pkg、Inno Setup/MSIX、deb/rpm/AppImage、web）：

```bash
git clone https://github.com/Matkurban/fpack
cd fpack/example
flutter pub get
fpack build --dry-run     # 查看 fpack 将执行的每条命令
fpack build               # apk + web（build.targets）
```

更多说明见 [example/README.md](https://github.com/Matkurban/fpack/tree/main/example)。

## 4. 配置（可选）

fpack 无需配置即可使用。需要安装包元数据、签名或自定义文件名时，运行 `fpack init`——它会生成一份列出**全部**键并带注释的 `fpack.yaml`，并预填从项目中检测到的值。见[配置](/configuration)。

## 下一步

- [命令](/commands)——所有命令与参数
- [目标与产物](/targets)——每个目标产出什么、需要什么
- 平台指南：[Android](/platforms/android) · [iOS](/platforms/ios) · [macOS](/platforms/macos) · [Windows](/platforms/windows) · [Linux](/platforms/linux) · [Web](/platforms/web)
- [CI 示例](/ci)——各平台的 GitHub Actions 配置
