---
title: "目标与产物"
description: "每个目标产出什么、在哪里构建、需要哪些工具。"
---

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

命令行和 `build.targets` 都接受目标别名：`bundle`/`appbundle` → `aab`，`android` → `apk`，`ios` → `ipa`，`mac`/`osx`/`app` → `macos`，`win`/`zip`/`portable` → `windows`，`setup`/`installer`/`inno` → `exe`，`tar`/`tgz`/`tar.gz` → `linux`，`debian` → `deb`，`fedora` → `rpm`，`image`/`AppImage` → `appimage`。

`fpack doctor` 只有在所需工具齐全时才把目标列为“当前可构建”；对 `apk`/`aab` 来说包括 Java（JDK 17+）——不过没有检测到 Java 时构建仍会尝试（Gradle 可能找到 fpack 看不到的 JDK）。
