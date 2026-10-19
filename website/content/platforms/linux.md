---
title: "Linux"
description: "Build tar.gz, deb, rpm, AppImage and Snap packages, including Snap builds in LXD or Multipass."
---

This tutorial builds Linux packages: a tar.gz bundle (`linux`), Debian/Ubuntu packages (`deb`), Fedora/RHEL/openSUSE packages (`rpm`), an AppImage (`appimage`) and a Snap (`snap`). It builds on Linux only; the five targets share one `flutter build linux`.

## What you need

```bash
# Debian / Ubuntu: Flutter's Linux toolchain plus rpm for the rpm target
sudo apt-get install -y clang cmake ninja-build pkg-config libgtk-3-dev rpm
```

| Target | Extra tool | Install |
| --- | --- | --- |
| `deb` | `dpkg-deb` | part of `dpkg` (preinstalled on Debian and Ubuntu) |
| `rpm` | `rpmbuild` | `sudo apt-get install -y rpm` |
| `appimage` | `appimagetool` | download the AppImage, `chmod +x`, put it on `PATH` (below) |
| `snap` | `snapcraft` | `sudo snap install snapcraft --classic` (plus LXD or Multipass unless you build on Ubuntu 24.04) |

```bash
wget -O ~/.local/bin/appimagetool https://github.com/AppImage/appimagetool/releases/download/continuous/appimagetool-x86_64.AppImage
chmod +x ~/.local/bin/appimagetool
```

## 1. Check the machine

```bash
fpack doctor linux deb rpm appimage snap
```

Doctor lists every missing tool at once with an install command. `fpack build --all` skips `appimage` and `snap` when their tools are missing (naming them explicitly makes a missing tool an error).

## 2. Build the tar.gz

```bash
fpack build linux
```

`dist/1.0.0+1/my_app-1.0.0+1-linux-x64.tar.gz` contains the Flutter bundle; users extract it and run the executable.

## 3. Describe your app once

deb, rpm, AppImage and Snap share these settings. Everything has a default taken from `pubspec.yaml` and your project, so a first build needs none of it:

```yaml
linux:
  package_name: my-app              # default: the app name in kebab-case
  prefix: /opt/my-app               # install location (default /opt/<package>, with a symlink in /usr/bin)
  icon: assets/icon.png             # scaled into the hicolor theme per icon_sizes
  categories: [Office, Utility]     # .desktop Categories=
  keywords: [notes, todo]
  mime_types: [text/markdown]
  startup_wm_class: my_app
  metainfo: linux/com.example.app.metainfo.xml    # AppStream
```

The generated `.desktop` file gets `Name`, `GenericName`, `Comment`, `Categories`, `Keywords`, `MimeType` and `StartupWMClass`; `app.display_name`, `app.description`, `app.maintainer`, `app.homepage`, `app.license` fill the package metadata ([Names, versions and output](/guides/project-and-output)).

## 4. Debian and Ubuntu packages

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
    postinst: linux/postinst        # also preinst, prerm, postrm
```

Check the result: `dpkg-deb -c dist/…/my_app-…-linux-x64.deb`, install with `sudo apt install ./my_app-….deb`.

## 5. RPM packages

```bash
fpack build rpm
```

```yaml
linux:
  rpm:
    requires: [gtk3]
    license: MIT
    group: Applications/Productivity
    post: linux/rpm-post            # also pre, preun, postun (%pre/%post/%preun/%postun)
```

Check: `rpm -qlp dist/…/my_app-….rpm`, install with `sudo dnf install ./my_app-….rpm`.

## 6. AppImage

```bash
fpack build appimage
```

```yaml
linux:
  appimage:
    update_information: "gh-releases-zsync|me|my-app|latest|my-app-*-x86_64.AppImage.zsync"
```

With `update_information` fpack embeds update info and writes the `.zsync` file next to the AppImage. Run it directly: `chmod +x my_app-….AppImage && ./my_app-….AppImage`.

## 7. Snap

```bash
fpack build snap
```

fpack generates `snap/snapcraft.yaml` and packs the bundle Flutter already built, so Flutter is not rebuilt inside snapcraft. The generated snap uses base `core24`, `strict` confinement, the `gnome` extension for GTK, plugs `desktop`, `desktop-legacy`, `wayland`, `x11`, `opengl`, `audio-playback`, `network` and `home`, a `dump` part over the bundle, and `snap/gui/` with the `.desktop` file and icon.

```yaml
linux:
  snap:
    summary: My app for your desktop
    plugs: [network, home, removable-media]
    build_mode: destructive         # see below
```

Name, summary, description, base, grade, confinement, license, plugs, extensions, stage packages, extra apps and the `.desktop` file can all be set under `linux.snap`; `linux.snap.template` points to your own `snapcraft.yaml` instead.

### Choose how snapcraft builds

`linux.snap.build_mode` (or `FPACK_SNAP_BUILD_MODE`) chooses the environment:

| Mode | How | Use it when |
| --- | --- | --- |
| `destructive` | directly on this machine | you are on **Ubuntu 24.04** (the release of base `core24`), typically CI. Fastest. snapcraft installs build dependencies here, so fpack runs it with `sudo -n` unless you are root: passwordless sudo as on CI, or run `sudo -v` first |
| `lxd` | in an LXD container | any other Linux. `sudo snap install lxd && sudo lxd init --auto && sudo usermod -aG lxd $USER`, then log in again |
| `multipass` | in a Multipass VM | any other Linux (or when you prefer VMs). `sudo snap install multipass` |
| `auto` (default) | destructive on a matching Ubuntu, otherwise lxd | you do not care |

::: warning
The snap contains the bundle Flutter built on **this** machine, so it is for this machine's architecture (amd64 or arm64), whatever the build mode. LXD and Multipass isolate the build; they do not cross-compile Flutter. Build an arm64 snap on an arm64 Linux machine or runner (for example GitHub's `ubuntu-24.04-arm`).
:::

Test the snap locally:

```bash
sudo snap install --dangerous dist/1.0.0+1/my_app-1.0.0+1-linux-x64.snap
snap run my-app
sudo snap remove my-app
```

## In CI

Ubuntu runners cover everything; use `destructive` for snaps on `ubuntu-24.04`. See [CI recipes](/ci#linux-and-web).

## If something goes wrong

| Message | Fix |
| --- | --- |
| Linux desktop toolchain incomplete, missing: … | `sudo apt-get install -y clang cmake ninja-build pkg-config libgtk-3-dev` |
| `rpmbuild` / `appimagetool` / `snapcraft` missing | install as above; doctor prints the command |
| snap: needs sudo / passwordless sudo | run `sudo -v` first, or use `linux.snap.build_mode: lxd` |
| snap: LXD not initialised | `sudo lxd init --auto`, add your user to the `lxd` group |

More on the [troubleshooting page](/troubleshooting#linux).
