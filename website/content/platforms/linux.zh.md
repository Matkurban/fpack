---
title: "Linux"
description: "构建 tar.gz、deb、rpm、AppImage 和 Snap 包，包括在 LXD 或 Multipass 里构建 Snap。"
---

本教程构建 Linux 包：tar.gz 压缩包（`linux`）、Debian/Ubuntu 包（`deb`）、Fedora/RHEL/openSUSE 包（`rpm`）、AppImage（`appimage`）和 Snap（`snap`）。只能在 Linux 上构建；五个目标共用一次 `flutter build linux`。

## 你需要什么

```bash
# Debian / Ubuntu：Flutter 的 Linux 工具链，加上 rpm 目标需要的 rpm
sudo apt-get install -y clang cmake ninja-build pkg-config libgtk-3-dev rpm
```

| 目标 | 额外工具 | 安装 |
| --- | --- | --- |
| `deb` | `dpkg-deb` | `dpkg` 的一部分（Debian 和 Ubuntu 预装） |
| `rpm` | `rpmbuild` | `sudo apt-get install -y rpm` |
| `appimage` | `appimagetool` | 下载这个 AppImage，`chmod +x`，放进 `PATH`（见下） |
| `snap` | `snapcraft` | `sudo snap install snapcraft --classic`（除非在 Ubuntu 24.04 上构建，否则还需要 LXD 或 Multipass） |

```bash
wget -O ~/.local/bin/appimagetool https://github.com/AppImage/appimagetool/releases/download/continuous/appimagetool-x86_64.AppImage
chmod +x ~/.local/bin/appimagetool
```

## 1. 检查机器

```bash
fpack doctor linux deb rpm appimage snap
```

doctor 会一次列出所有缺少的工具和对应的安装命令。缺少 appimagetool 或 snapcraft 时，`fpack build --all` 会跳过 `appimage` 和 `snap`（明确指定它们时，缺工具则报错）。

## 2. 构建 tar.gz

```bash
fpack build linux
```

`dist/1.0.0+1/my_app-1.0.0+1-linux-x64.tar.gz` 里是 Flutter 的 bundle；用户解压后运行可执行文件即可。

## 3. 一次描述你的应用

deb、rpm、AppImage 和 Snap 共用这些设置。每一项都有来自 `pubspec.yaml` 和项目的默认值，所以第一次构建不需要任何配置：

```yaml
linux:
  package_name: my-app              # 默认：kebab-case 的应用名
  prefix: /opt/my-app               # 安装位置（默认 /opt/<包名>，并在 /usr/bin 里放符号链接）
  icon: assets/icon.png             # 按 icon_sizes 缩放到 hicolor 主题
  categories: [Office, Utility]     # .desktop 的 Categories=
  keywords: [notes, todo]
  mime_types: [text/markdown]
  startup_wm_class: my_app
  metainfo: linux/com.example.app.metainfo.xml    # AppStream
```

生成的 `.desktop` 文件包含 `Name`、`GenericName`、`Comment`、`Categories`、`Keywords`、`MimeType` 和 `StartupWMClass`；`app.display_name`、`app.description`、`app.maintainer`、`app.homepage`、`app.license` 用来填充包的元数据（[名称、版本与输出](/guides/project-and-output)）。

## 4. Debian 和 Ubuntu 包

```bash
fpack build deb
```

```yaml
linux:
  deb:
    depends: ["libgtk-3-0 | libgtk-3-0t64"]
    recommends: []
    section: utils
    priority: optional
    postinst: linux/postinst        # 还有 preinst、prerm、postrm
```

检查结果：`dpkg-deb -c dist/…/my_app-…-linux-x64.deb`，安装用 `sudo apt install ./my_app-….deb`。

## 5. RPM 包

```bash
fpack build rpm
```

```yaml
linux:
  rpm:
    requires: [gtk3]
    license: MIT
    group: Applications/Productivity
    post: linux/rpm-post            # 还有 pre、preun、postun（%pre/%post/%preun/%postun）
```

检查：`rpm -qlp dist/…/my_app-….rpm`，安装用 `sudo dnf install ./my_app-….rpm`。

## 6. AppImage

```bash
fpack build appimage
```

```yaml
linux:
  appimage:
    update_information: "gh-releases-zsync|me|my-app|latest|my-app-*-x86_64.AppImage.zsync"
```

设置了 `update_information` 后，fpack 会嵌入更新信息，并把 `.zsync` 文件写在 AppImage 旁边。直接运行：`chmod +x my_app-….AppImage && ./my_app-….AppImage`。

## 7. Snap

```bash
fpack build snap
```

fpack 生成 `snap/snapcraft.yaml`，并打包 Flutter 已经构建好的 bundle，所以不会在 snapcraft 里重新构建 Flutter。生成的 snap 使用 base `core24`、`strict` 约束、用于 GTK 的 `gnome` 扩展、plug `desktop`、`desktop-legacy`、`wayland`、`x11`、`opengl`、`audio-playback`、`network` 和 `home`、一个覆盖 bundle 的 `dump` part，以及 `snap/gui/` 里的 `.desktop` 文件和图标。

```yaml
linux:
  snap:
    summary: My app for your desktop
    plugs: [network, home, removable-media]
    build_mode: destructive         # 见下
```

名称、摘要、描述、base、grade、约束、许可证、plugs、扩展、stage packages、额外的 app 和 `.desktop` 文件都可以在 `linux.snap` 下设置；`linux.snap.template` 指向你自己的 `snapcraft.yaml`。

### 选择 snapcraft 的构建方式

`linux.snap.build_mode`（或 `FPACK_SNAP_BUILD_MODE`）选择构建环境：

| 模式 | 方式 | 何时使用 |
| --- | --- | --- |
| `destructive` | 直接在本机构建 | 你在 **Ubuntu 24.04**（base `core24` 对应的版本）上，通常是 CI。最快。snapcraft 会在本机安装构建依赖，所以除非你是 root，fpack 会用 `sudo -n` 运行它：像 CI 一样配置免密码 sudo，或先运行 `sudo -v` |
| `lxd` | 在 LXD 容器中 | 其他任何 Linux。`sudo snap install lxd && sudo lxd init --auto && sudo usermod -aG lxd $USER`，然后重新登录 |
| `multipass` | 在 Multipass 虚拟机中 | 其他任何 Linux（或你更喜欢虚拟机）。`sudo snap install multipass` |
| `auto`（默认） | 在匹配的 Ubuntu 上用 destructive，否则用 lxd | 你不想操心 |

::: warning
snap 里是**这台**机器上 Flutter 构建的 bundle，所以无论哪种构建方式，它都只适用于这台机器的架构（amd64 或 arm64）。LXD 和 Multipass 只是隔离构建，并不会交叉编译 Flutter。arm64 的 snap 要在 arm64 的 Linux 机器或 runner 上构建（例如 GitHub 的 `ubuntu-24.04-arm`）。
:::

在本机测试 snap：

```bash
sudo snap install --dangerous dist/1.0.0+1/my_app-1.0.0+1-linux-x64.snap
snap run my-app
sudo snap remove my-app
```

## 在 CI 中

Ubuntu runner 能覆盖所有目标；在 `ubuntu-24.04` 上用 `destructive` 构建 snap。见 [CI 示例](/ci#linux-与-web)。

## 出问题时

| 提示 | 处理 |
| --- | --- |
| Linux desktop toolchain incomplete, missing: … | `sudo apt-get install -y clang cmake ninja-build pkg-config libgtk-3-dev` |
| 缺少 `rpmbuild` / `appimagetool` / `snapcraft` | 按上面安装；doctor 会打印命令 |
| snap：需要 sudo / 免密码 sudo | 先运行 `sudo -v`，或使用 `linux.snap.build_mode: lxd` |
| snap：LXD 未初始化 | `sudo lxd init --auto`，并把你的用户加入 `lxd` 组 |

更多见[故障排查页](/troubleshooting#linux-zh)。
