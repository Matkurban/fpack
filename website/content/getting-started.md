---
title: "Your first build"
description: "From a Flutter project to your first artifact in ten minutes: doctor, dry run, build, read the result."
---

This page takes you from a plain Flutter project to a release file, step by step. You need fpack [installed](/installation) and a Flutter project; the examples use one called `my_app`. No configuration file is required.

## 1. Go to your project

```bash
cd my_app
```

fpack finds the project by walking up from the current directory until it sees a `pubspec.yaml`. You can run it from anywhere with `fpack -C path/to/my_app …`.

## 2. Ask what this machine can build

```bash
fpack doctor
```

```text
fpack doctor  ·  fpack 1.1.17  ·  linux/amd64

General
  ✓ Flutter 3.47.7 (stable) · /home/me/flutter  [PATH]
  ✓ Dart 3.13.5
  ✓ project my_app 1.0.0+1 · /home/me/my_app
  • no fpack.yaml – zero-config defaults are used

Android  (apk, aab)
  ✓ Android SDK · /home/me/android-sdk
  ! no Java (JDK 17+) found
  → install Android Studio (bundles a JDK) or a JDK 17/21, then: flutter config --jdk-dir <path>

iOS  (ipa)
  – needs macOS (this is Linux)
…
Web  (web)
  ✓ no extra tools needed

✓ ready to build here: web
```

How to read it: `✓` ok, `!` a warning (fpack can still try), `✗` a problem (the target cannot be built), `–` not possible on this operating system. Each problem is followed by an `→` line with the command that fixes it. Fix what you need for the targets you want, then run `doctor` again. `fpack list` shows the same information as one table with every target.

::: tip
Doctor does not build anything. It is safe to run any time, and the same checks run before every build.
:::

## 3. Rehearse with a dry run

```bash
fpack build web --dry-run
```

```text
Plan (dry run – nothing will be executed)

▶ web  Web build (zip of build/web, ready to deploy)
  $ /home/me/flutter/bin/flutter build web --release
  · zip build/web
  · move to dist/1.0.0+1/my_app-1.0.0+1-web.zip
  = dist/1.0.0+1/my_app-1.0.0+1-web.zip  [Web (zip)]
```

The plan lists every command with its full arguments and every file that would be created. Passwords are shown as `***`. Use `--dry-run` whenever you change something and want to see the effect first.

## 4. Build

```bash
fpack build web
```

```text
▶ Web · web  flutter build web --release
  ✓ flutter build web (25.7s)
  ✓ dist/1.0.0+1/my_app-1.0.0+1-web.zip  13.8 MB

Summary
  TARGET  STATUS  FILE                                 SIZE     TIME
  web     ✓ done  dist/1.0.0+1/my_app-1.0.0+1-web.zip  13.8 MB  26.2s

Output:    dist/1.0.0+1
Checksums: dist/1.0.0+1/SHA256SUMS
Logs:      build/fpack/logs
```

Replace `web` with any target your machine is ready for (`apk`, `macos`, `windows`, `deb`, …) or list several: `fpack build apk aab`. Each target has its own tutorial, linked at the bottom of this page.

## 5. Read the result

Everything goes to `dist/<version>+<build>/`, together with a checksum file:

```text
dist/1.0.0+1/
  my_app-1.0.0+1-android-universal.apk
  my_app-1.0.0+1-android.aab
  my_app-1.0.0+1-macos-universal.dmg
  my_app-1.0.0+1-web.zip
  SHA256SUMS
```

The version comes from `pubspec.yaml` (`version: 1.0.0+1`: name `1.0.0`, build number `1`). Verify the files any time:

::: tabs
@@ macOS
```bash
cd dist/1.0.0+1 && shasum -a 256 -c SHA256SUMS
```
@@ Linux
```bash
cd dist/1.0.0+1 && sha256sum -c SHA256SUMS
```
@@ Windows
```powershell
cd dist\1.0.0+1; Get-Content SHA256SUMS
Get-FileHash .\my_app-1.0.0+1-web.zip -Algorithm SHA256
```
:::

fpack never overwrites an existing artifact. Building the same version twice stops with "already exists": bump the version in `pubspec.yaml`, pass `--build-number 2`, or pass `--force` to replace the files.

## 6. Create `fpack.yaml` (when you need it)

Zero configuration is enough for debug-signed Android builds and plain desktop bundles. When you need installer metadata, signing, custom file names or hooks:

```bash
fpack init          # interactive; add --yes to accept the detected defaults
```

`fpack init` writes `fpack.yaml` listing **every** key as a comment, pre-filled with values detected in your project (application ID, flavors, version, team ID, installer AppId, signing certificates found in your keychain). Uncomment only what you need; `fpack doctor` and `fpack build` report mistakes with the line number and a suggestion. Run `fpack init` again later and it writes `fpack.yaml.new` plus a diff instead of touching your file.

A minimal file that sets the display name and the default targets:

```yaml
app:
  display_name: My App
build:
  targets: [apk, aab, web]    # what `fpack build` builds without arguments
```

Now `fpack build` alone builds those three targets.

## 7. Try the example project

The repository has a complete example app with a fully commented `fpack.yaml` (flavors, signing through environment variables, DMG/pkg, Inno Setup/MSIX, deb/rpm/AppImage, web):

```bash
git clone https://github.com/Matkurban/fpack
cd fpack/example
flutter pub get
fpack build --dry-run     # every command fpack would run
fpack build               # the targets in build.targets
```

See the [example README](https://github.com/Matkurban/fpack/tree/main/example) for more.

## What next?

| I want to… | Go to |
| --- | --- |
| ship an Android app (signed APK / AAB for Google Play) | [Android](/platforms/android) |
| ship an iOS app (App Store, TestFlight, ad hoc) | [iOS](/platforms/ios) |
| ship a Mac app outside the App Store (signed, notarized DMG) | [macOS](/platforms/macos) |
| build a Windows installer | [Windows](/platforms/windows) |
| build deb / rpm / AppImage / Snap | [Linux](/platforms/linux) |
| deploy a web build | [Web](/platforms/web) |
| change names, versions or the output folder | [Names, versions and output](/guides/project-and-output) |
| automate in CI | [CI recipes](/ci) |
| something failed | [Troubleshooting](/troubleshooting) |
