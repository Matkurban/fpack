---
title: "fpack 如何工作"
description: "目标、构建流程、文件放在哪里，以及 fpack 绝不会动什么。先读一遍，后面的内容都建立在这里。"
---

fpack 是打包工具，不是构建系统。它按正确的顺序驱动你已经在用的工具（`flutter`、Gradle、Xcode、signtool、`dpkg-deb` 等），补上签名、公证和打包，并把结果收集起来。本页解释后面文档会反复用到的几个概念。

## 目标与产物

**目标（target）**是你想发布的一种文件。在命令行里写出目标名：

```bash
fpack build apk aab       # 两个目标
fpack build macos dmg pkg # 三个目标
```

| 平台 | 目标 | 得到什么 |
| --- | --- | --- |
| Android | `apk`、`aab` | APK（通用和/或按 ABI 拆分）、Android App Bundle |
| iOS | `ipa` | 用于 App Store Connect、Ad Hoc、企业分发或未签名的 IPA |
| macOS | `macos`、`dmg`、`pkg` | 压缩的 `.app`、磁盘映像、安装包（已签名、已公证） |
| Windows | `windows`、`exe`、`msix` | 便携 zip、Inno Setup 安装程序、MSIX |
| Linux | `linux`、`deb`、`rpm`、`appimage`、`snap` | tar.gz、Debian 包、RPM、AppImage、Snap |
| Web | `web` | 可部署站点的 zip |

**产物（artifact）**是目标生成的文件。每个目标的详细信息见[目标与产物](/targets)。

::: note
Flutter 不能交叉编译桌面和 iOS 应用：iOS 和 macOS 要在 Mac 上构建，Windows 目标要在 Windows 上构建，Linux 目标要在 Linux 上构建；Android 和 Web 在哪里都能构建。[`fpack build --all`](/guides/many-targets) 会构建*这台*机器能构建的全部目标，其余的注明原因并跳过；明确指定一个本机无法构建的目标会报错，并指引你使用对应系统的 CI 机器。
:::

## 构建流程

每次运行 fpack 都执行同样的步骤，所以失败总是发生得早、也容易定位：

1. **读取**项目（`pubspec.yaml`、`android/app/build.gradle`、Xcode 工程）和 `fpack.yaml`（如果有）。配置无效会在这里停下，并给出行号和建议。
2. **检查**每个目标（与 [`fpack doctor`](/commands#fpack-doctor-zh) 相同）：工具、SDK、证书、密钥库密码、flavor。无法构建的目标会在任何耗时步骤开始*之前*报告。
3. **构建**：每个平台只运行一次 Flutter 构建。`macos` + `dmg` + `pkg` 共用一次 `flutter build macos`；`linux` + `deb` + `rpm` + `appimage` + `snap` 共用一次 `flutter build linux`。
4. **打包与签名**每个目标：复制或重新打包、签名、公证、验证。
5. **校验和与汇总**：在产物旁写入 `SHA256SUMS`，并打印产出文件的表格。

`--dry-run` 在第 2 步之后停下，打印第 3、4 步将运行的确切命令。

## 文件放在哪里

```text
my_app/
  dist/1.0.0+1/            产物和 SHA256SUMS（output.dir）
  build/fpack/logs/        每条命令的完整日志
  build/fpack/secrets/     临时密钥文件，权限 0600，构建后删除
  build/fpack/…            生成的 ExportOptions.plist、snapcraft.yaml 等
```

- **产物**放在 `dist/<版本>+<构建号>/`。文件名遵循模板（[名称与输出](/guides/project-and-output)）。默认**绝不覆盖**已有产物：提高版本号、传 `--build-number N`，或者用 `--force`。
- **fpack 创建的其他一切**都在 `build/fpack/` 下，可以用 [`fpack clean`](/commands#fpack-clean-zh) 删除。
- **你的源码是只读的。** fpack 不会修改 Gradle 文件、Xcode 工程、`pubspec.yaml`、图标或启动页配置。建议把 `dist/` 加入 `.gitignore`。

## 配置：参数、变量、文件

fpack 不需要配置。需要时，配置有四个来源，先设置了值的那个生效：

1. 命令行参数（`--flavor prod`）
2. `FPACK_*` 环境变量（`FPACK_FLAVOR=prod`）
3. 项目根目录的 `fpack.yaml`（`build.flavor: prod`）
4. 默认值（从你的项目读取，或使用 fpack 自己的默认）

`fpack init` 会写出一个列出**所有**键的 `fpack.yaml`，全部注释掉并带说明，需要什么就取消注释什么。文件里的 `${VAR}` 和 `${VAR:-默认值}` 会读取环境变量，密钥就是这样不进文件的（[密钥与环境变量](/guides/secrets)）。

## 安全网

| 你想知道 | 运行 |
| --- | --- |
| 这台机器能构建什么？缺什么？ | `fpack doctor`（或 `fpack list`） |
| 会运行哪些命令？ | `fpack build … --dry-run` |
| 为什么失败？ | 看 fpack 打印的错误摘要和修复建议；会显示完整日志路径；加 `-v` 查看工具的完整输出 |
| 到底产出了什么？ | `dist/<版本>/SHA256SUMS`，或 `--json` |

## 术语

| 术语 | 含义 |
| --- | --- |
| 目标 | 一种产物：`apk`、`dmg`、`deb` 等 |
| flavor | Android productFlavor / Xcode scheme（[Flavor](/guides/build-variants)） |
| 模式 | `release`（默认）、`profile` 或 `debug` |
| 启动器 / 核心 | 你安装的小型 Dart 程序，以及它下载并运行的原生二进制（[安装](/installation#launcher-zh)） |
| 公证 | Apple 对在 App Store 之外分发的 macOS 软件做的恶意软件扫描（[macOS](/platforms/macos)） |

## 下一步

在[下一页](/installation)安装 fpack，然后进行[第一次构建](/getting-started)。
