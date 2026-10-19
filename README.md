# fpack

**English** · [中文](https://github.com/Matkurban/fpack/blob/main/README.ZH.md)

### 📖 [Documentation → matkurban.github.io/fpack](https://matkurban.github.io/fpack/)

Guides for every platform (signing, notarization, installers), every command and config key, CI recipes and troubleshooting. A complete sample app with a commented `fpack.yaml` lives in [`example/`](https://github.com/Matkurban/fpack/tree/main/example).

**One command turns a Flutter project into release artifacts for every platform.** It works with zero configuration, reads `pubspec.yaml`, Gradle and the Xcode projects, and never modifies your project files.

## Install

Requires Dart 3.8+ (bundled with Flutter).

```bash
dart pub global activate fpack
```

Make sure `~/.pub-cache/bin` (Windows: `%LOCALAPPDATA%\Pub\Cache\bin`) is on your `PATH`, then check:

```bash
fpack --version
```

The first run downloads the matching native core (≈4 MB, SHA-256 verified). Offline and mirror options: [Installation](https://matkurban.github.io/fpack/installation/).

## Quick start

```bash
cd my_flutter_app
fpack init                       # (optional) commented fpack.yaml; --yes uses the detected defaults
fpack doctor                     # what can this machine build? what is missing and how to install it?
fpack build apk --dry-run        # shows the commands that would run (runs nothing)
fpack build apk                  # first build → dist/1.0.0+1/my_app-1.0.0+1-android-universal.apk
fpack build --all                # everything this machine can build; the rest is skipped with a reason
```

Artifacts go to `dist/<version>+<build>/` together with `SHA256SUMS`. Existing artifacts are never overwritten: bump the version, use `--build-number N`, or pass `--force`.

## A minimal fpack.yaml

Every key is optional. This is enough to build and sign an Android release:

```yaml
app:
  display_name: My App
build:
  targets: [apk, aab]                # built by `fpack build` without targets
android:
  signing:
    store_file: ~/keys/upload.jks
    store_password: ${FPACK_ANDROID_KEYSTORE_PASSWORD}   # never write passwords into the file
    key_alias: upload
```

## Supported targets

Android: `apk` · `aab` — iOS: `ipa` — macOS: `macos` · `dmg` · `pkg` — Windows: `windows` · `exe` · `msix` — Linux: `linux` · `deb` · `rpm` · `appimage` · `snap` — Web: `web`.

`fpack list` shows every target and whether this machine can build it. Details and artifact names: [Targets](https://matkurban.github.io/fpack/targets/).

## More docs

Everything else lives on the documentation site: **[matkurban.github.io/fpack](https://matkurban.github.io/fpack/)**

- [Getting started](https://matkurban.github.io/fpack/getting-started/) · [Installation](https://matkurban.github.io/fpack/installation/) · [Commands](https://matkurban.github.io/fpack/commands/) · [Targets](https://matkurban.github.io/fpack/targets/)
- [Configuration reference](https://matkurban.github.io/fpack/configuration/) · [Environment variables](https://matkurban.github.io/fpack/environment/)
- Platform guides (signing, notarization, installers): [Android](https://matkurban.github.io/fpack/platforms/android/) · [iOS](https://matkurban.github.io/fpack/platforms/ios/) · [macOS](https://matkurban.github.io/fpack/platforms/macos/) · [Windows](https://matkurban.github.io/fpack/platforms/windows/) · [Linux](https://matkurban.github.io/fpack/platforms/linux/) · [Web](https://matkurban.github.io/fpack/platforms/web/)
- [CI examples](https://matkurban.github.io/fpack/ci/) · [Troubleshooting](https://matkurban.github.io/fpack/troubleshooting/) · [FAQ](https://matkurban.github.io/fpack/faq/)
- [AI agent skill](https://matkurban.github.io/fpack/skills/) · [Architecture](https://matkurban.github.io/fpack/architecture/) · [Changelog](https://matkurban.github.io/fpack/changelog/)
- Source and issues: [GitHub](https://github.com/Matkurban/fpack) · [pub.dev](https://pub.dev/packages/fpack)

## License

MIT
