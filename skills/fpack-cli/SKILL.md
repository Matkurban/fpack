---
name: fpack-cli
description: >-
  Use when the user wants to build, package, sign, notarize or release a
  Flutter app (APK, AAB, IPA, macOS zip/DMG/pkg, Windows zip/Inno Setup
  exe/MSIX, Linux tar.gz/deb/rpm/AppImage/Snap, web zip) with the fpack CLI, edit
  fpack.yaml, set FPACK_* environment variables, or run fpack in CI.
---

# Packaging Flutter apps with fpack

fpack is a command-line tool (not a library): never import it from Dart code.
It wraps `flutter build` and the platform packaging tools and writes release
files to `dist/<version>+<build>/`. Full docs: https://matkurban.github.io/fpack/

## Guidelines

* Install it globally: `dart pub global activate fpack` (make sure
  `~/.pub-cache/bin` is on `PATH`). Check with `fpack --version`.
* Run fpack from the Flutter project root (or pass `-C <dir>`).
* Always run `fpack build <targets> --dry-run` first when unsure: it prints the
  plan and every exact command and builds nothing.
* Run `fpack doctor` when a build fails on prerequisites; it lists every missing
  tool with an install hint. `fpack list` shows which targets this machine can build.
* fpack works without configuration. Create `fpack.yaml` only when needed, with
  `fpack init --yes`: every key is listed with a comment; only values detected
  in the project (default targets, flavor, display name…) are active, the rest
  stays commented out – uncomment only what you change. Never invent keys; run `fpack schema` or read
  https://matkurban.github.io/fpack/configuration/ for the authoritative list.
* Precedence: command-line flags > `FPACK_*` environment variables >
  `fpack.yaml` > defaults. `fpack.yaml` values may reference `${VAR}` or `${VAR:-default}`.
* Never write passwords, keystores, certificates or API keys into
  `fpack.yaml` or the repository. Pass them as environment variables
  (CI secrets); fpack masks them in logs and dry runs.
* fpack never edits project files (no changes to `build.gradle`, Xcode
  projects or `pubspec.yaml`); temporary files go to `build/fpack/`.
* Flutter cannot cross-compile: iOS/macOS targets need macOS, Windows targets
  need Windows, Linux targets need Linux. Use one CI runner per OS.
* An existing artifact stops a real build (exit code 3). Bump the version,
  pass `--build-number N`, or use `--force`.
* For scripts and CI use `--json` (JSON result on stdout, human output on
  stderr) and check the exit code: 0 ok, 1 build failed (`fpack doctor`:
  problems found), 2 usage/config error or no Flutter project, 3 prerequisites
  missing, artifact exists or nothing buildable, 130 interrupted.
* Output language follows the OS (Chinese on a Chinese system, else English);
  use `--lang en` when you need to parse English messages.

## Targets

| Platform | Targets |
| --- | --- |
| Android | `apk` (`--split-per-abi=both` keeps the universal APK too), `aab` |
| iOS | `ipa` (`--export-method app-store-connect\|release-testing\|ad-hoc\|enterprise\|debugging` or the older `app-store`/`development`, `--no-codesign`) |
| macOS | `macos` (zipped .app), `dmg`, `pkg` |
| Windows | `windows` (portable zip), `exe` (Inno Setup 6), `msix` (needs the `msix` dev_dependency) |
| Linux | `linux` (tar.gz), `deb`, `rpm`, `appimage`, `snap` |
| Web | `web` (`--base-href /app/`, `--wasm`) |

Targets of one platform share a single `flutter build`. `fpack build` with no
targets builds `build.targets` from fpack.yaml (without it, it stops with exit
code 2 and asks which targets); `--all` builds everything this machine can
build and skips the rest with a reason.

## Examples

### Everyday builds

```sh
fpack build apk aab --flavor prod --dart-define-from-file config/prod.json
fpack build linux deb rpm appimage
fpack build web --base-href /app/
fpack build --all --dry-run
fpack build ipa --export-method ad-hoc
fpack build apk -- --no-tree-shake-icons   # extra flutter args after --
```

### Android signing (environment variables only)

```sh
export FPACK_ANDROID_KEYSTORE=$HOME/keys/upload.jks   # or FPACK_ANDROID_KEYSTORE_BASE64 in CI
export FPACK_ANDROID_KEYSTORE_PASSWORD=...            # from a secret store
export FPACK_ANDROID_KEY_ALIAS=upload
export FPACK_ANDROID_KEY_PASSWORD=...                 # if it differs from the store password
fpack build apk aab
```

fpack validates the keystore with `keytool` before building. Without signing
it warns that the debug key is used (fine for testing, rejected by Google Play).

### macOS signing and notarization

```yaml
# fpack.yaml
macos:
  sign:
    identity: "Developer ID Application: Example Inc (ABCDE12345)"
    installer_identity: "Developer ID Installer: Example Inc (ABCDE12345)"  # for pkg
    notary_profile: fpack-notary   # xcrun notarytool store-credentials fpack-notary …
```

```sh
fpack build macos dmg pkg                 # sign (hardened runtime), notarize, staple, verify
fpack build dmg --no-notarize             # fast local build
fpack build dmg --notarize-no-wait        # submit and exit; state in dist/<ver>/NOTARIZATION.md
fpack notarize status                     # ask Apple
fpack notarize finish                     # wait, staple, update SHA256SUMS
```

Instead of a keychain profile, CI can use an App Store Connect API key:
`FPACK_NOTARY_API_KEY` (path to the .p8), `FPACK_NOTARY_API_KEY_ID`,
`FPACK_NOTARY_API_ISSUER`. For a rejected submission fpack downloads Apple's log and
summarizes it (usually unsigned nested code or a missing hardened runtime).
Certificates and credentials: https://matkurban.github.io/fpack/platforms/macos/

### Snap (Linux)

```yaml
# fpack.yaml – defaults: base core24, strict confinement, gnome extension,
# desktop/wayland/x11/opengl/audio-playback/network/home plugs
linux:
  snap:
    summary: My app            # max 78 characters
    plugs: [network, home]     # [] for none
    build_mode: destructive    # auto | destructive | lxd | multipass
```

`fpack build snap` packs the already built bundle (dump plugin; Flutter is
not rebuilt). Install snapcraft with `sudo snap install snapcraft --classic`.
`auto` builds destructively on Ubuntu 24.04 (snapcraft runs with `sudo -n`,
so sudo must not ask for a password) and with LXD elsewhere. Test with
`sudo snap install --dangerous dist/*/*.snap`.

### Windows signing

Optional. Set `FPACK_WINDOWS_CERTIFICATE` (.pfx) + `FPACK_WINDOWS_CERTIFICATE_PASSWORD`,
or `FPACK_WINDOWS_CERT_THUMBPRINT` / `FPACK_WINDOWS_CERT_SUBJECT` (certificate store),
or `windows.sign.dlib` + `dlib_metadata` (Azure Trusted Signing). fpack signs the
Release `*.exe`/`*.dll`, the installer and the MSIX with signtool (SHA-256 +
timestamp) and verifies them; `fpack doctor` reports signtool (install with
`winget install Microsoft.WindowsSDK.10.0.26100`).

### GitHub Actions

```yaml
jobs:
  android:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v7
      - uses: subosito/flutter-action@v2
      - run: dart pub global activate fpack 1.1.13 && echo "$HOME/.pub-cache/bin" >> $GITHUB_PATH
      - run: fpack build apk aab --json > result.json
        env:
          FPACK_ANDROID_KEYSTORE_BASE64: ${{ secrets.KEYSTORE_BASE64 }}
          FPACK_ANDROID_KEYSTORE_PASSWORD: ${{ secrets.KEYSTORE_PASSWORD }}
          FPACK_ANDROID_KEY_ALIAS: upload
      - uses: actions/upload-artifact@v7
        with: { name: android, path: dist/ }
```

More recipes (macOS, Windows, Linux): https://matkurban.github.io/fpack/ci/

## Troubleshooting

| Message | Fix |
| --- | --- |
| `Flutter SDK not found` | install Flutter, or `--flutter <sdk>` / `FPACK_FLUTTER`; FVM is detected |
| `no Java (JDK 17+) found` | install JDK 17/21, then `flutter config --jdk-dir <path>` |
| `flavor "x" not found` / `… but no flavor is set` | use a flavor listed by `fpack doctor`, pass `--flavor` or set `build.flavor` |
| `signing identity … not found in the keychain` | use a listed Developer ID identity or `--no-sign` |
| `already exists` | bump the version, `--build-number N` or `--force` |
| `needs macOS/Windows/Linux` | build on that OS |
| `Inno Setup (ISCC.exe) not found` / `dpkg-deb`, `rpmbuild`, `appimagetool`, `snapcraft` missing | install the tool (`fpack doctor` lists all) |
| `no Flutter project found in … or its parents` (lists the apps below a repo root) | `cd` into one or pass `fpack -C <dir>` |
| `which targets?` | name targets, use `--all`, or set `build.targets` |

More: https://matkurban.github.io/fpack/troubleshooting/ and
https://matkurban.github.io/fpack/environment/
