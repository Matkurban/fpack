---
title: "安装"
description: "安装 fpack、配置 PATH，并了解它如何找到原生核心。"
---

fpack 只需要 **Dart 3.8+**（Flutter 自带）。你要构建的目标需要各自常用的工具链（见[各目标的要求](#各目标的要求)）；`fpack doctor` 会帮你检查。

## 1. 安装

```bash
dart pub global activate fpack
```

这会从 [pub.dev](https://pub.dev/packages/fpack) 安装 `fpack` 命令。其他安装方式：

| 方式 | 命令 | 说明 |
| --- | --- | --- |
| pub.dev | `dart pub global activate fpack` | 推荐 |
| git | `dart pub global activate --source git https://github.com/Matkurban/fpack` | 首次运行从 GitHub Release 下载核心（经 SHA-256 校验）；只有下载不可用时才用本机 Go 构建 |
| 本地路径 | `dart pub global activate --source path /path/to/fpack` | 可离线使用：发布压缩包内含全部 6 个平台的预编译二进制 |

## 2. 加入 PATH

Dart 把全局工具装在 pub 缓存目录里，这个目录必须在 `PATH` 中：

::: tabs
@@ macOS
```bash
echo 'export PATH="$PATH:$HOME/.pub-cache/bin"' >> ~/.zshrc && source ~/.zshrc
```
@@ Linux
```bash
echo 'export PATH="$PATH:$HOME/.pub-cache/bin"' >> ~/.bashrc && source ~/.bashrc
```
@@ Windows
把 `%LOCALAPPDATA%\Pub\Cache\bin` 加入用户 `PATH`（设置 → 系统 → 关于 → 高级系统设置 → 环境变量），然后打开新的终端。
:::

## 3. 检查

```bash
fpack --version
```

会打印已安装的版本号。如果 shell 提示 `command not found`，说明上面的 PATH 步骤没做。然后试试[第一次构建](/getting-started)。

::: tip
`fpack --wrapper-info` 会显示使用的是哪个原生核心、来自哪里。安装看起来不对劲时，先运行它。
:::

## 升级

```bash
dart pub global activate fpack    # 安装最新版本
fpack --version
```

每个 fpack 版本使用自己的原生核心，每个版本下载一次并缓存。

## 各目标的要求

fpack 本身只需要 Dart。每个目标需要各自常用的工具链：

| 目标 | 需要 |
| --- | --- |
| `apk`、`aab` | Flutter、Android SDK、Java 17+（Android Studio 自带） |
| `ipa`、`macos`、`dmg`、`pkg` | 装有 Xcode 的 macOS（工程有 Podfile 时还需要 CocoaPods） |
| `windows`、`exe`、`msix` | 装有 Visual Studio（C++ 桌面开发工作负载）的 Windows；`exe` 需要 Inno Setup 6；`msix` 需要 `msix` 开发依赖 |
| `linux`、`deb`、`rpm`、`appimage`、`snap` | Linux，需要 clang、cmake、ninja、pkg-config、GTK 3 头文件；`dpkg-deb`、`rpmbuild`、`appimagetool`、`snapcraft`（除非在 Ubuntu 24.04 上构建，否则还需要 LXD 或 Multipass） |
| `web` | 只需要 Flutter |

在项目里运行 `fpack doctor`，可以看到*你的*机器具体缺什么，每一项都附带安装命令。

<a id="launcher-zh"></a>

## 启动器如何找到原生核心

`fpack` 命令是一个小型 Dart 启动器，真正干活的是 `fpack-core`，一个原生 Go 二进制（约 4 MB，启动约 10 毫秒）。每次运行，启动器按下面的顺序查找版本与 Dart 包**完全一致**的核心（用 `fpack-core --core-version` 检查）。**经过校验的预编译二进制永远优先，本地构建只是最后的手段。**

1. 环境变量 `FPACK_CORE`（开发用）
2. 用户缓存：macOS `~/Library/Caches/fpack/<版本>/<os>-<arch>/`，Linux `$XDG_CACHE_HOME/fpack/…`（默认 `~/.cache/fpack`），Windows `%LOCALAPPDATA%\fpack\…`（可用 `FPACK_HOME` 覆盖）
3. 随包的预编译二进制 `prebuilt/<os>-<arch>/`（如果有）：用 `prebuilt/manifest.json` 里的 SHA-256 校验后复制到缓存
4. 从 GitHub Release 下载本平台的二进制，用 `checksums.txt` 校验：校验不一致的文件绝不会被执行
5. **只有下载失败或被禁用时**：用随包的 Go 源码本地构建（`go build -trimpath`，依赖已 vendor，无需网络），并明确提示已回退到本地构建以及使用的 Go：

   ```text
   fpack: note: no verified prebuilt core available (download: …connection refused…);
   fpack: falling back to a local build from the bundled sources with go1.27.2 (/usr/local/go/bin/go) – first run only, ~30s…
   ```

`fpack --wrapper-info` 会打印每一步的路径和最终使用的二进制。缓存的核心在版本变化前一直复用；想用下载的二进制替换本地构建的核心，删除缓存目录即可。

### 离线、镜像与开发

| 变量 | 作用 |
| --- | --- |
| `FPACK_NO_DOWNLOAD=1` | 离线 / 隔离网络：从不下载；使用随包二进制或用本机 Go 构建 |
| `FPACK_DOWNLOAD_URL` | 从镜像下载（一个目录，包含 `checksums.txt` 和 `fpack-core-<os>-<arch>`（Windows 为 `.exe`），布局与 GitHub Release 相同） |
| `FPACK_GO=/path/to/go` / `FPACK_GO=none` | 回退构建使用的 Go / 永不本地构建 |
| `FPACK_REBUILD=1` | 开发：忽略缓存，先用本机 Go 构建（修改 Go 源码之后） |
| `FPACK_HOME` | 缓存目录 |
| `FPACK_CORE` | 使用指定的核心二进制 |

::: note
**包体积的取舍。** 6 个平台的二进制约 23 MB（压缩后约 10 MB）。把它们打进 pub 包，安装后即可离线使用，但每个用户都要下载所有平台的二进制；不打包则包很小（约 350 KB），首次运行时下载本平台的二进制。由发布工作流的仓库变量 `BUNDLE_BINARIES` 决定；GitHub Release 上的 `fpack-vX.Y.Z.tar.gz` 始终包含全部二进制，供离线和内网使用。
:::

## 下一步

[第一次构建](/getting-started)：`doctor`、预演和真正的构建。
