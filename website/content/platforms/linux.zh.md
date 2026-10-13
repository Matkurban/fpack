---
title: "Linux"
description: "tar.gz、deb、rpm、AppImage 与 Snap 软件包。"
---

deb/rpm/AppImage 安装到 `linux.prefix`（默认 `/opt/<包名>`，并在 `/usr/bin` 放符号链接），`.desktop` 文件包含 `Name`/`GenericName`/`Comment`/`Categories`/`Keywords`/`MimeType`/`StartupWMClass`，图标按 `icon_sizes` 缩放到 hicolor 主题，可附带 AppStream `metainfo` 和 `copyright`。deb 支持 Depends/Recommends/Suggests/Conflicts/Section/Priority 与维护脚本，rpm 支持 Requires/Group/License/URL 与 `%pre/%post/%preun/%postun`，AppImage 支持内嵌更新信息（`update_information`，同时在输出目录生成 `.zsync` 文件）。

`snap` 目标用 snapcraft 打包已经构建好的 bundle。fpack 生成 `snap/snapcraft.yaml`（base `core24`、`strict` 沙箱、提供 GTK 的 `gnome` 扩展、plugs `desktop`、`desktop-legacy`、`wayland`、`x11`、`opengl`、`audio-playback`、`network`、`home`，以及基于 bundle 的 `dump` part，snapcraft 内不会重新构建 Flutter），并在 `snap/gui/` 中放入 `.desktop` 文件和图标。名称、简介、描述、base、grade、confinement、许可证、plugs、扩展、stage packages、额外命令和 `.desktop` 文件都可在 `linux.snap` 中设置；`linux.snap.template` 可改用你自己的 snapcraft.yaml。`linux.snap.build_mode`（或 `FPACK_SNAP_BUILD_MODE`）决定 snapcraft 的构建方式：`destructive`（直接在本机构建，本机必须是该 base 对应的 Ubuntu 版本，core24 即 24.04；最快，适合 CI；snapcraft 会在本机安装构建依赖，因此非 root 时 fpack 用 `sudo -n` 运行它：像 CI 一样配置免密码 sudo，或先运行 `sudo -v`）、`lxd` 或 `multipass`（隔离的构建环境）。默认的 `auto` 在匹配的 Ubuntu 上使用 destructive，否则使用 lxd。用 `sudo snap install snapcraft --classic` 安装 snapcraft，然后用 `sudo snap install --dangerous <文件>.snap` 测试安装。snap 中是 Flutter 在本机构建的 bundle，因此无论哪种构建方式，它都只适用于本机架构（amd64 或 arm64）：LXD 和 Multipass 只是隔离构建环境，并不会交叉编译 Flutter。arm64 的 snap 请在 arm64 的 Linux 机器或 runner（如 `ubuntu-24.04-arm`）上构建。

## 工具

```bash
# Debian/Ubuntu
sudo apt-get install -y clang cmake ninja-build pkg-config libgtk-3-dev rpm
# appimagetool（appimage 目标需要）
wget -O ~/.local/bin/appimagetool https://github.com/AppImage/appimagetool/releases/download/continuous/appimagetool-x86_64.AppImage
chmod +x ~/.local/bin/appimagetool
# snapcraft（snap 目标需要）
sudo snap install snapcraft --classic
```

`linux`、`deb`、`rpm`、`appimage`、`snap` 共用一次 `flutter build linux`。缺少 appimagetool 或 snapcraft 时，`fpack build --all` 会跳过 `appimage` 或 `snap`。

## 软件包示例

```yaml
linux:
  package_name: my-app            # 默认：应用名的短横线形式
  icon: assets/icon.png
  categories: [Office, Utility]
  metainfo: linux/com.example.app.metainfo.xml
  deb:
    depends: ["libgtk-3-0 | libgtk-3-0t64"]
    postinst: linux/postinst
  rpm:
    requires: [gtk3]
    license: MIT
  appimage:
    update_information: "gh-releases-zsync|me|my-app|latest|my-app-*-x86_64.AppImage.zsync"
  snap:
    summary: 桌面上的我的应用
    plugs: [network, home, removable-media]
    build_mode: destructive       # Ubuntu 24.04 / CI；其他系统用 lxd 或 multipass
```

可以用 `dpkg-deb -c`、`rpm -qlp` 检查产物，直接运行 AppImage，或用 `sudo snap install --dangerous` 与 `snap remove` 测试 snap。
