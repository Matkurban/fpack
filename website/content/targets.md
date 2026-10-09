---
title: "Targets and artifacts"
description: "What every target builds, where it runs and which tools it needs."
---

The default file name template is `{app}{-flavor}-{version}{+build}-{platform}{-arch}{-variant}{-mode}` (no mode for release).

| Target | Host OS | Artifact | Tools |
| --- | --- | --- | --- |
| `apk` | any | `-android-universal.apk`, `-android-arm64-v8a.apk` … | flutter, apksigner (signature check) |
| `aab` | any | `-android.aab` | flutter, keytool (signature check) |
| `ipa` | macOS | `-ios-arm64.ipa` (unsigned: `-ios-arm64-unsigned.ipa`) | flutter/xcodebuild, ditto with `--no-codesign` |
| `macos` | macOS | `-macos-universal.zip` | flutter, codesign, ditto, notarytool |
| `dmg` | macOS | `-macos-universal.dmg` | hdiutil or create-dmg, codesign, notarytool, stapler |
| `pkg` | macOS | `-macos.pkg` | pkgbuild + productbuild (installs to /Applications), Developer ID Installer signing, notarytool, stapler |
| `windows` | Windows | `-windows-x64-portable.zip` | flutter |
| `exe` | Windows | `-windows-x64-setup.exe` | Inno Setup (ISCC) |
| `msix` | Windows | `-windows-x64.msix` | the `msix` dev dependency (`dart run msix:create`) |
| `linux` | Linux | `-linux-x64.tar.gz` | flutter |
| `deb` | Linux | `-linux-x64.deb` | dpkg-deb |
| `rpm` | Linux | `-linux-x64.rpm` | rpmbuild |
| `appimage` | Linux | `-linux-x64.AppImage` | appimagetool (skipped by `--all` when missing) |
| `web` | any | `-web.zip` | flutter |

Flutter cannot cross-compile iOS/macOS/Windows/Linux desktop apps. `fpack build --all` skips targets this machine cannot build and says why; naming one explicitly (e.g. `fpack build ipa` on Linux) is an error that points you to a CI runner of that OS. `--dry-run` still shows a reference plan for such targets.

One flutter build is shared per run: `macos` + `dmg` + `pkg` build once, `linux` + `deb` + `rpm` + `appimage` build once, `windows` + `exe` + `msix` build once.

Target aliases work on the command line and in `build.targets`: `bundle`/`appbundle` → `aab`, `android` → `apk`, `ios` → `ipa`, `mac`/`osx`/`app` → `macos`, `win`/`zip`/`portable` → `windows`, `setup`/`installer`/`inno` → `exe`, `tar`/`tgz`/`tar.gz` → `linux`, `debian` → `deb`, `fedora` → `rpm`, `image`/`AppImage` → `appimage`.

`fpack doctor` lists a target as "ready to build" only when nothing it needs is missing; for `apk`/`aab` that includes Java (JDK 17+), even though a build still tries without it (Gradle may find a JDK fpack cannot see).
