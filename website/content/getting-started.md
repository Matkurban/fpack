---
title: "Getting started"
description: "From install to your first release artifacts."
---

fpack turns a Flutter project into release files for every platform with one command. This page takes you from zero to your first artifacts in a few minutes.

## 1. Install

```bash
dart pub global activate fpack
fpack --version
```

Make sure `~/.pub-cache/bin` is on your `PATH` (see [Installation](/installation)).

## 2. Check your machine

```bash
cd my_flutter_app
fpack doctor                     # checks the environment, with an install/fix command per item
fpack build apk --dry-run        # shows the commands that would run (runs nothing)
fpack build apk                  # actually build
fpack init                       # (optional) commented fpack.yaml, interactive; --yes uses the defaults
```

Artifacts go to `dist/<version>+<build>/` by default, together with `SHA256SUMS` (check with `shasum -a 256 -c SHA256SUMS`):

```
dist/1.0.0+1/
  my_app-1.0.0+1-android-universal.apk
  my_app-1.0.0+1-android-arm64-v8a.apk
  my_app-1.0.0+1-android.aab
  my_app-1.0.0+1-ios-arm64.ipa
  my_app-1.0.0+1-macos-universal.dmg
  my_app-1.0.0+1-macos.pkg
  SHA256SUMS
```

Existing artifacts are **never overwritten** by default: bump the version, use `--build-number N`, or pass `--force`.

## 3. Try the example project

The repository contains a complete example app with a fully commented `fpack.yaml` (flavors, signing via environment variables, DMG/pkg, Inno Setup/MSIX, deb/rpm/AppImage, web):

```bash
git clone https://github.com/Matkurban/fpack
cd fpack/example
flutter pub get
fpack build --dry-run     # see every command fpack would run
fpack build               # apk + web (build.targets)
```

See [example/README.md](https://github.com/Matkurban/fpack/tree/main/example) for more.

## 4. Configure (optional)

fpack works without configuration. When you want installer metadata, signing or custom file names, run `fpack init` – it writes an `fpack.yaml` that lists **every** key with a comment, pre-filled with values detected in your project. See [Configuration](/configuration).

## Next steps

- [Commands](/commands) – every command and option
- [Targets and artifacts](/targets) – what each target produces and needs
- Platform guides: [Android](/platforms/android) · [iOS](/platforms/ios) · [macOS](/platforms/macos) · [Windows](/platforms/windows) · [Linux](/platforms/linux) · [Web](/platforms/web)
- [CI recipes](/ci) – GitHub Actions for every platform
