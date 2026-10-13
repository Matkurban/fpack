---
title: "Linux"
description: "tar.gz, deb, rpm, AppImage and Snap packages."
---

deb/rpm/AppImage install into `linux.prefix` (default `/opt/<package>`, with a symlink in `/usr/bin`); the `.desktop` file has `Name`/`GenericName`/`Comment`/`Categories`/`Keywords`/`MimeType`/`StartupWMClass`, the icon is scaled into the hicolor theme per `icon_sizes`, and an AppStream `metainfo` and `copyright` can be included. deb supports Depends/Recommends/Suggests/Conflicts/Section/Priority and maintainer scripts, rpm supports Requires/Group/License/URL and `%pre/%post/%preun/%postun`, and AppImage supports embedded update information (`update_information`, with the `.zsync` file written to the output directory).

The `snap` target packs the already built bundle with snapcraft. fpack generates `snap/snapcraft.yaml` (base `core24`, `strict` confinement, the `gnome` extension for GTK, plugs `desktop`, `desktop-legacy`, `wayland`, `x11`, `opengl`, `audio-playback`, `network`, `home`, and a `dump` part over the bundle, so Flutter is not rebuilt inside snapcraft) plus a `.desktop` file and the icon in `snap/gui/`. Name, summary, description, base, grade, confinement, license, plugs, extensions, stage packages, extra apps and the `.desktop` file can be set under `linux.snap`; `linux.snap.template` uses your own snapcraft.yaml instead. `linux.snap.build_mode` (or `FPACK_SNAP_BUILD_MODE`) chooses how snapcraft builds: `destructive` (directly on this machine, which must run the Ubuntu release of the base, 24.04 for core24; fastest, good for CI; snapcraft installs build dependencies here, so fpack runs it with `sudo -n` unless you are root: passwordless sudo as on CI, or run `sudo -v` first), `lxd` or `multipass` (isolated build environments). The default `auto` uses destructive on a matching Ubuntu and lxd elsewhere. Install snapcraft with `sudo snap install snapcraft --classic`, then test the snap with `sudo snap install --dangerous <file>.snap`. The snap contains the bundle Flutter built on this machine, so it is for this machine's architecture (amd64 or arm64), whatever the build mode: LXD and Multipass isolate the build but do not cross-compile Flutter. Build an arm64 snap on an arm64 Linux machine or runner (for example `ubuntu-24.04-arm`).

## Tools

```bash
# Debian/Ubuntu
sudo apt-get install -y clang cmake ninja-build pkg-config libgtk-3-dev rpm
# appimagetool (for the appimage target)
wget -O ~/.local/bin/appimagetool https://github.com/AppImage/appimagetool/releases/download/continuous/appimagetool-x86_64.AppImage
chmod +x ~/.local/bin/appimagetool
# snapcraft (for the snap target)
sudo snap install snapcraft --classic
```

`linux`, `deb`, `rpm`, `appimage` and `snap` share one `flutter build linux`. `fpack build --all` skips `appimage` and `snap` when appimagetool or snapcraft is missing.

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
  snap:
    summary: My app for your desktop
    plugs: [network, home, removable-media]
    build_mode: destructive       # Ubuntu 24.04 / CI; lxd or multipass elsewhere
```

Check the result with `dpkg-deb -c`, `rpm -qlp`, by running the AppImage, or with `sudo snap install --dangerous` and `snap remove`.
