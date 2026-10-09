---
title: "安装"
description: "安装 fpack，以及启动器如何找到原生核心。"
lang: zh-CN
---

需要 Dart 3.8+（Flutter 自带）。三种方式任选：

| 方式 | 命令 | 说明 |
| --- | --- | --- |
| pub.dev | `dart pub global activate fpack` | 推荐 |
| git | `dart pub global activate --source git https://github.com/Matkurban/fpack` | 首次运行时从 GitHub Release 下载核心（SHA-256 校验）；下载不可用时用本机 Go 编译 |
| 本地路径 | `dart pub global activate --source path /path/to/fpack` | 离线可用：发布包内已带 6 个平台的预编译二进制 |

确保 `~/.pub-cache/bin`（Windows：`%LOCALAPPDATA%\Pub\Cache\bin`）在 `PATH` 中：

```bash
echo 'export PATH="$PATH:$HOME/.pub-cache/bin"' >> ~/.zshrc && source ~/.zshrc
fpack --version
```

<a id="launcher"></a>

## 启动器如何找到原生核心

`fpack` 每次运行都会按以下顺序找一个**版本与 Dart 包完全一致**的 `fpack-core`（用 `fpack-core --core-version` 校验）。**优先使用经过校验的预编译二进制，本机编译只是最后的退路**：

1. `FPACK_CORE` 环境变量（开发调试用）
2. 用户缓存：macOS `~/Library/Caches/fpack/<版本>/<os>-<arch>/`，Linux `$XDG_CACHE_HOME/fpack/…`（默认 `~/.cache/fpack`），Windows `%LOCALAPPDATA%\fpack\…`（可用 `FPACK_HOME` 覆盖）
3. 包内预编译二进制 `prebuilt/<os>-<arch>/`（存在时）：先用 `prebuilt/manifest.json` 中的 SHA-256 校验，再复制到缓存
4. 从 GitHub Release 下载当前平台的二进制（约 4 MB），用 `checksums.txt` 校验 SHA-256，校验失败绝不执行
5. **只有在下载失败或被禁用时**，才用本机 Go 从包内源码编译（`go build -trimpath`，依赖已 vendor，无需联网），并明确提示「已回退到本机编译」以及使用的 Go 版本，例如：

   ```
   fpack: note: no verified prebuilt core available (download: …connection refused…);
   fpack: falling back to a local build from the bundled sources with go1.27.2 (/usr/local/go/bin/go) – first run only, ~30s…
   ```

控制变量：

| 变量 | 作用 |
| --- | --- |
| `FPACK_NO_DOWNLOAD=1` | 离线/内网：不下载，直接用本机 Go 编译（或使用包内二进制） |
| `FPACK_DOWNLOAD_URL` | 从镜像下载（目录中需包含 `checksums.txt` 与 `fpack-core-<os>-<arch>`，与 GitHub Release 相同） |
| `FPACK_GO=/path/to/go` / `FPACK_GO=none` | 指定编译用的 Go / 完全禁止本机编译 |
| `FPACK_REBUILD=1` | 开发用：忽略缓存，先用本机 Go 编译（修改 Go 源码后） |
| `FPACK_HOME` | 缓存目录 |

缓存中的核心会一直使用到版本变化；想改用下载的二进制替换本机编译的核心，删除缓存目录即可（`fpack --wrapper-info` 会显示路径）。

`fpack --wrapper-info` 可查看每一步的路径和最终使用的二进制。

> **pub.dev 包体积取舍**：6 个平台的二进制合计约 23 MB（压缩后约 10 MB）。打包进 pub 包 → 安装即可离线使用，但每个用户都要下载所有平台的二进制；不打包 → 包很小（约 350 KB），首次运行时从 GitHub Release 下载当前平台约 4 MB 的二进制（下载不可用时才用本机 Go 编译，约 30 秒）。发布工作流用仓库变量 `BUNDLE_BINARIES` 控制；GitHub Release 中的 `fpack-vX.Y.Z.tar.gz` 始终包含全部二进制，适合离线/内网使用。

## 各目标的环境要求

fpack 本身只需要 Dart。各目标需要各自的工具链——运行 `fpack doctor` 可以看到本机缺什么，以及每一项的安装命令：

| 目标 | 需要 |
| --- | --- |
| `apk`、`aab` | Flutter、Android SDK、Java 17+（Android Studio 自带） |
| `ipa`、`macos`、`dmg`、`pkg` | 装有 Xcode 的 macOS（项目有 Podfile 时还需 CocoaPods） |
| `windows`、`exe`、`msix` | 装有 Visual Studio（C++ 桌面开发）的 Windows；`exe` 需要 Inno Setup 6；`msix` 需要 `msix` dev 依赖 |
| `linux`、`deb`、`rpm`、`appimage` | 装有 clang、cmake、ninja、pkg-config、GTK 3 开发包的 Linux；以及 `dpkg-deb`、`rpmbuild`、`appimagetool` |
| `web` | 只需要 Flutter |

## 升级

```bash
dart pub global activate fpack    # 安装最新版本
fpack --version
```

每个 fpack 版本使用各自的原生核心（每个版本下载一次并缓存）。
