---
title: "fpack"
description: "One command turns a Flutter project into release artifacts for every platform."
---

<div class="hero-badges">

[![pub package](https://img.shields.io/pub/v/fpack.svg)](https://pub.dev/packages/fpack) [![CI](https://github.com/Matkurban/fpack/actions/workflows/ci.yml/badge.svg)](https://github.com/Matkurban/fpack/actions/workflows/ci.yml) [![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](https://github.com/Matkurban/fpack/blob/main/LICENSE)

</div>

**Android** (APK / per-ABI APKs / AAB) · **iOS** (IPA) · **macOS** (.app zip / DMG / pkg, signed + notarized) · **Windows** (zip / Inno Setup installer / MSIX) · **Linux** (tar.gz / deb / rpm / AppImage) · **Web** (zip)

```bash
dart pub global activate fpack
cd my_flutter_app
fpack doctor              # what can this machine build? what is missing and how to install it?
fpack build apk aab       # → dist/1.0.0+1/my_app-1.0.0+1-android-universal.apk …
fpack build --all         # everything this machine can build; the rest is skipped with a reason
```

## Why fpack

- **Zero configuration** – reads `pubspec.yaml`, Gradle and the Xcode projects; `fpack.yaml` is optional.
- **Never modifies your project** – signing data is injected through environment variables, Gradle and Xcode files are never edited. Artifacts go to `dist/`, temporary files to `build/fpack/`.
- **Native core** – a Go binary (~10 ms startup, no runtime dependencies) behind a thin Dart launcher that downloads the matching, SHA-256 verified core.
- **Everything is configurable** – 188 validated `fpack.yaml` keys with a JSON Schema for editor completion, `${VAR}` interpolation and dedicated `FPACK_*` variables.
- **Built for humans and CI** – clear errors with the key excerpt and a fix, `--dry-run` shows every command, `--json` for machines, English and Chinese output.
- **Signing and notarization** – Android keystores, iOS export options, macOS Developer ID + notarization (with resumable waits), Windows Authenticode.

## Where to go next

| | |
| --- | --- |
| 🚀 [Getting started](/getting-started) | install fpack and build your first artifacts |
| ⌨️ [Commands](/commands) | every command and option |
| ⚙️ [Configuration](/configuration) | all `fpack.yaml` keys |
| 📦 [Targets](/targets) | what each target produces and needs |
| 🔐 [macOS signing & notarization](/platforms/macos) | Developer ID, notarytool, credentials for CI |
| 🤖 [CI recipes](/ci) | GitHub Actions for every platform |
| 🧰 [Troubleshooting](/troubleshooting) | common problems and exit codes |

A complete example project with a commented `fpack.yaml` lives in [`example/`](https://github.com/Matkurban/fpack/tree/main/example).
