# fpack

English · [中文](README.md)

**Package a Flutter app into release files for every platform with one command.**
Android (APK / per-ABI APKs / AAB), iOS (IPA), macOS (.app zip / DMG with signing + notarization), Windows (zip / Inno Setup installer / MSIX), Linux (tar.gz / deb / rpm / AppImage) and Web (zip).

```bash
dart pub global activate fpack    # install (git and local path work too, see below)
cd my_flutter_app
fpack doctor                      # what can this machine build, what is missing, how to fix it
fpack build apk aab               # → dist/1.0.0+1/my_app-1.0.0+1-android-universal.apk …
fpack build --all                 # everything this machine can build; the rest is skipped with a reason
```

- **Zero config**: reads `pubspec.yaml`, Gradle and Xcode projects. `fpack init` writes an optional, commented `fpack.yaml`.
- **Never edits your project**: fpack only runs the Flutter toolchain and packaging tools. Android signing is injected through environment variables (no Gradle edits). The only file it writes in your project is `fpack.yaml` (on `init`); artifacts go to `dist/`, temporary files to `build/fpack/`.
- **Native core**: written in Go and compiled to a native binary (~10 ms startup, no runtime deps). The Dart package is a thin launcher that finds or provisions the exact matching binary.
- **Built for humans**: colors and a spinner (plain output in CI), error excerpt + concrete fix + log path on failure, `--dry-run` showing every exact command, `--json` output, Chinese/English auto-detection.

## Install

Requires Dart 3.8+ (bundled with Flutter).

| Method | Command | Notes |
| --- | --- | --- |
| pub.dev | `dart pub global activate fpack` | recommended (once published) |
| git | `dart pub global activate --source git https://github.com/Matkurban/fpack` | first run compiles the core with your local Go, or downloads it from the GitHub release |
| path | `dart pub global activate --source path /path/to/fpack` | works offline: release archives contain prebuilt binaries for 6 hosts |

Make sure `~/.pub-cache/bin` (Windows: `%LOCALAPPDATA%\Pub\Cache\bin`) is on `PATH`.

### How the launcher finds the native core

On every run `fpack` looks for an `fpack-core` whose version **equals** the Dart package version (checked with `fpack-core --core-version`):

1. `FPACK_CORE` (development override)
2. the user cache: macOS `~/Library/Caches/fpack/<version>/<os>-<arch>/`, Linux `$XDG_CACHE_HOME/fpack/…` (default `~/.cache/fpack`), Windows `%LOCALAPPDATA%\fpack\…` (override with `FPACK_HOME`)
3. the bundled binary `prebuilt/<os>-<arch>/`, verified against the SHA-256 in `prebuilt/manifest.json`, then copied into the cache
4. `go build` from the bundled sources with your local Go (dependencies are vendored, no network)
5. download from the GitHub release, verified against `checksums.txt`; a mismatch is never executed

`fpack --wrapper-info` shows every location and the binary in use.

> **pub.dev size tradeoff**: binaries for all 6 hosts are ~23 MB (~10 MB compressed). Bundling them makes installs work offline but every user downloads every platform's binary; not bundling keeps the package at ~200 KB and the first run compiles the core with Go (~30 s) or downloads the ~4 MB binary for the current host. The release workflow controls this with the `BUNDLE_BINARIES` repository variable; the GitHub release archive `fpack-vX.Y.Z.tar.gz` always contains all binaries for offline/intranet installs.

## Quick start

```bash
fpack doctor                     # check prerequisites with install/fix commands
fpack build apk --dry-run        # show the exact commands, run nothing
fpack build apk                  # build for real
fpack init                       # optional: create fpack.yaml (interactive; --yes for defaults)
```

Artifacts go to `dist/<version>+<build>/` with a `SHA256SUMS` file (`shasum -a 256 -c SHA256SUMS`). Existing artifacts are **never overwritten** unless you bump the version, pass `--build-number N`, or use `--force`.

## Commands

```
fpack build [targets…] [options] [-- extra flutter args]
fpack doctor          check prerequisites per target and show how to fix what is missing
fpack list            list targets, output formats and what this machine can build
fpack init            create fpack.yaml (interactive; -y/--yes for defaults; never overwrites)
fpack clean           remove build/fpack; --dist also dist/; --flutter-clean runs flutter clean; --all
fpack version         version information (also --version / -V)
fpack help <command>
```

Global options: `-C, --project DIR`, `--config FILE`, `--flutter SDK`, `--lang zh|en`, `-v, --verbose`, `--json`, `--no-color`, `-y, --yes`.

Build options: `--all`, `--dry-run`, `--mode`/`--release`/`--profile`/`--debug`, `--flavor`, `-t/--target`, `--dart-define K=V` (repeatable), `--dart-define-from-file`, `--build-name`, `--build-number`, `--split-per-abi[=true|both]`, `--abis`, `--obfuscate`, `--split-debug-info`, `-o/--output`, `-f/--force`, iOS `--export-method`, `--export-options-plist`, `--no-codesign`, macOS `--sign`, `--no-sign`, `--sign-identity`, `--notarize`, `--no-notarize`, `--notary-profile`, `--dmg-tool`, web `--base-href`, `--wasm`, and anything after `--` is passed to `flutter build`.

```bash
fpack build apk aab --flavor prod --dart-define-from-file env/prod.json
fpack build apk --split-per-abi=both
fpack build ipa --export-method ad-hoc
fpack build ipa --no-codesign
fpack build macos dmg              # signs/notarizes per fpack.yaml macos.sign (unsigned if not configured)
fpack build dmg --no-notarize      # fast local build
fpack build --all --json > result.json
fpack -C apps/client build web --base-href /app/
```

**Precedence**: flags > `FPACK_*` env vars > `fpack.yaml` > defaults.
**Ctrl-C** stops flutter/gradle/xcodebuild gracefully (whole process group); a second Ctrl-C forces; exit code 130.
**Monorepos**: fpack walks up to the nearest `pubspec.yaml`; at a repo root it lists the Flutter projects it found — pick one with `-C`.
**Flutter SDK lookup**: `--flutter` → `FPACK_FLUTTER` → `flutter.sdk` in fpack.yaml → FVM (`.fvm/flutter_sdk`, `.fvmrc`) → `FLUTTER_ROOT` → `PATH` → common locations (`~/develop/flutter`, `~/flutter`, …).

## Targets

Name template: `{app}{-flavor}-{version}{+build}-{platform}{-arch}{-variant}{-mode}` (mode omitted for release).

| Target | Host | Artifact | Tools |
| --- | --- | --- | --- |
| `apk` | any | `-android-universal.apk`, `-android-arm64-v8a.apk`, … | flutter, apksigner (verification) |
| `aab` | any | `-android.aab` | flutter, keytool (verification) |
| `ipa` | macOS | `-ios-arm64.ipa` (unsigned: `-ios-arm64-unsigned.ipa`) | flutter/xcodebuild, ditto |
| `macos` | macOS | `-macos-universal.zip` | flutter, codesign, ditto, notarytool |
| `dmg` | macOS | `-macos-universal.dmg` | hdiutil or create-dmg, codesign, notarytool, stapler |
| `windows` | Windows | `-windows-x64-portable.zip` | flutter |
| `exe` | Windows | `-windows-x64-setup.exe` | Inno Setup (ISCC) |
| `msix` | Windows | `-windows-x64.msix` | `msix` dev dependency |
| `linux` | Linux | `-linux-x64.tar.gz` | flutter |
| `deb` | Linux | `-linux-x64.deb` | dpkg-deb |
| `rpm` | Linux | `-linux-x64.rpm` | rpmbuild |
| `appimage` | Linux | `-linux-x64.AppImage` | appimagetool (skipped by `--all` when missing) |
| `web` | any | `-web.zip` | flutter |

Flutter cannot cross-compile desktop/iOS apps: `--all` skips what the host can't build (with the reason); naming such a target explicitly fails with a hint to use a matching CI runner, while `--dry-run` still shows its reference plan. One flutter build is shared per platform in a run (`macos`+`dmg`, `linux`+`deb`+`rpm`+`appimage`, `windows`+`exe`+`msix`).

## Configuration

Full reference (every fpack.yaml key, every `FPACK_*` variable, every flag, precedence rules and a complete commented example, in Chinese): [doc/configuration.md](doc/configuration.md).

See the commented file generated by `fpack init`, or the full example in the [Chinese README](README.md#配置-fpackyaml) (keys are identical). Unknown keys are rejected with a "did you mean" suggestion; `${VAR}` and `${VAR:-default}` read environment variables.

## Signing

### Android
fpack never edits Gradle files. It injects the Android Gradle Plugin's standard `android.injected.signing.*` properties via `ORG_GRADLE_PROJECT_*` env vars; passwords never appear on command lines, logs or dry-run output (`***`).

```bash
export FPACK_ANDROID_KEYSTORE=~/keys/upload-keystore.jks
export FPACK_ANDROID_KEYSTORE_PASSWORD='…'
export FPACK_ANDROID_KEY_ALIAS=upload
export FPACK_ANDROID_KEY_PASSWORD='…'   # defaults to the keystore password
fpack build apk aab
```

`FPACK_ANDROID_KEYSTORE_BASE64` can carry the keystore itself in CI (written with mode 0600 to `build/fpack/secrets/`, deleted afterwards). Password and alias are verified with `keytool` before building; the produced APK/AAB signer is printed afterwards and an Android Debug certificate triggers a warning. Projects with their own `key.properties` signing keep working unchanged.

### iOS
`--export-method` (app-store-connect, release-testing, ad-hoc, development, enterprise), `--export-options-plist` (wins), or `--no-codesign` for an unsigned IPA. `doctor` checks Xcode, CocoaPods, `DEVELOPMENT_TEAM` and certificates; Xcode errors (certificates, profiles, team, pods) are mapped to concrete fixes.

### macOS (Developer ID)
Configured only the fpack way, low → high: `macos.sign` in fpack.yaml (`fpack init` writes a commented section with placeholders and lists the Developer ID identities in your keychain), `FPACK_MACOS_*` env vars, flags. Nothing configured → the .app keeps Xcode's signature and the DMG is unsigned (the result notes say how to configure it). fpack does not read other plugins' pubspec sections such as the [`dmg`](https://pub.dev/packages/dmg) package's `dmg:`. An identity implies signing, a notary profile implies notarization, disabling signing disables notarization. Without an identity the first "Developer ID Application" certificate is used; an unknown identity lists the available ones.

DMG flow: copy .app → `codesign --deep` → re-sign with hardened runtime + `macos/Runner/Release.entitlements` → verify → `hdiutil` (or `create-dmg`) with an Applications link → sign DMG → `notarytool submit --wait` → `stapler staple` → `spctl`. Create the notary profile once: `xcrun notarytool store-credentials <profile> --apple-id … --team-id …`. `fpack build macos` produces a signed zip, notarized too when notarization is on.

## Environment variables, exit codes

See the tables in the [Chinese README](README.md#环境变量). Exit codes: `0` ok, `1` a target failed, `2` usage/config error, `3` prerequisites/conflict/nothing to build, `130` interrupted. With `--json`, stdout carries only the JSON result and human output goes to stderr.

## Troubleshooting

- **Failures** show the key error lines, one concrete fix and the full log under `build/fpack/logs/`; `-v` streams everything.
- **Missing path dependency** (e.g. `path: ../xue_hua_sdk`) is detected before building.
- **Path installs print "Resolving dependencies…"** on every run — that's pub's behavior for path-activated packages (git/pub.dev installs don't). Use a git/pub.dev install when piping `--json` into other programs.
- **macOS Gatekeeper**: the launcher removes the quarantine attribute from the cached core; if it is still blocked run `FPACK_REBUILD=1 fpack --version` to compile it locally with Go.

## Development

```bash
cd go && go vet ./... && go test ./...
dart pub get && dart analyze && dart test
scripts/build_binaries.sh                     # cross-compile 6 hosts → prebuilt/
scripts/package_dist.sh ../fpack-dist.tar.gz  # self-contained offline archive
```

Release: bump the version in `pubspec.yaml`, `lib/src/version.dart` and `go/internal/version/version.go` (`scripts/check_versions.sh` enforces it), update `CHANGELOG.md`, push a `vX.Y.Z` tag.

## License

MIT
