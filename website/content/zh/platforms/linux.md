---
title: "Linux"
description: "tar.gz、deb、rpm 与 AppImage 软件包。"
lang: zh-CN
---

deb/rpm/AppImage 安装到 `linux.prefix`（默认 `/opt/<包名>`，并在 `/usr/bin` 放符号链接），`.desktop` 文件包含 `Name`/`GenericName`/`Comment`/`Categories`/`Keywords`/`MimeType`/`StartupWMClass`，图标按 `icon_sizes` 缩放到 hicolor 主题，可附带 AppStream `metainfo` 和 `copyright`。deb 支持 Depends/Recommends/Suggests/Conflicts/Section/Priority 与维护脚本，rpm 支持 Requires/Group/License/URL 与 `%pre/%post/%preun/%postun`，AppImage 支持内嵌更新信息（`update_information`，同时在输出目录生成 `.zsync` 文件）。

## 工具

```bash
# Debian/Ubuntu
sudo apt-get install -y clang cmake ninja-build pkg-config libgtk-3-dev rpm
# appimagetool（appimage 目标需要）
wget -O ~/.local/bin/appimagetool https://github.com/AppImage/appimagetool/releases/download/continuous/appimagetool-x86_64.AppImage
chmod +x ~/.local/bin/appimagetool
```

`linux`、`deb`、`rpm`、`appimage` 共用一次 `flutter build linux`。缺少 appimagetool 时，`fpack build --all` 会跳过 `appimage`。

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
```

可以用 `dpkg-deb -c`、`rpm -qlp` 检查产物，或直接运行 AppImage。
