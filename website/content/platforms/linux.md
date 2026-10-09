---
title: "Linux"
description: "tar.gz, deb, rpm and AppImage packages."
---

deb/rpm/AppImage install into `linux.prefix` (default `/opt/<package>`, with a symlink in `/usr/bin`); the `.desktop` file has `Name`/`GenericName`/`Comment`/`Categories`/`Keywords`/`MimeType`/`StartupWMClass`, the icon is scaled into the hicolor theme per `icon_sizes`, and an AppStream `metainfo` and `copyright` can be included. deb supports Depends/Recommends/Suggests/Conflicts/Section/Priority and maintainer scripts, rpm supports Requires/Group/License/URL and `%pre/%post/%preun/%postun`, and AppImage supports embedded update information (`update_information`, with the `.zsync` file written to the output directory).

## Tools

```bash
# Debian/Ubuntu
sudo apt-get install -y clang cmake ninja-build pkg-config libgtk-3-dev rpm
# appimagetool (for the appimage target)
wget -O ~/.local/bin/appimagetool https://github.com/AppImage/appimagetool/releases/download/continuous/appimagetool-x86_64.AppImage
chmod +x ~/.local/bin/appimagetool
```

`linux`, `deb`, `rpm` and `appimage` share one `flutter build linux`. `fpack build --all` skips `appimage` when appimagetool is missing.

## Package example

```yaml
linux:
  package_name: my-app            # default: app name in kebab-case
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

Check the result with `dpkg-deb -c`, `rpm -qlp` or by running the AppImage.
