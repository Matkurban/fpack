---
title: "CI recipes"
description: "GitHub Actions workflows for every platform, caching, secrets and machine-readable results."
---

fpack is built for CI: no prompts, stable exit codes, JSON output, redacted logs. The recipes below use GitHub Actions; the commands are the same anywhere.

## Principles

- **Pin the version** so CI is reproducible: `dart pub global activate fpack 1.1.18`.
- **Build each platform on its own OS.** iOS and macOS on `macos-*`, Windows on `windows-*`, Linux on `ubuntu-*`; Android and web run anywhere ([why](/concepts#targets-and-artifacts)).
- **Secrets through environment variables.** Never in `fpack.yaml` ([Secrets and variables](/guides/secrets)).
- **Upload `dist/`.** It contains the artifacts and `SHA256SUMS`.
- **Use `--json` and exit codes** to decide what happens next ([Exit codes and JSON](/exit-codes)).

Without a TTY, spinners and colors are turned off and a heartbeat line is printed every 60 seconds so CI does not time out on silence.

## Android

```yaml
jobs:
  android:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v7
      - uses: subosito/flutter-action@v2
      - run: dart pub global activate fpack 1.1.18 && echo "$HOME/.pub-cache/bin" >> $GITHUB_PATH
      - run: fpack build apk aab --split-per-abi=both
        env:
          FPACK_ANDROID_KEYSTORE_BASE64: ${{ secrets.KEYSTORE_BASE64 }}
          FPACK_ANDROID_KEYSTORE_PASSWORD: ${{ secrets.KEYSTORE_PASSWORD }}
          FPACK_ANDROID_KEY_ALIAS: upload
      - uses: actions/upload-artifact@v7
        with: { name: android, path: dist/ }
```

Create the secret with `base64 -i upload.jks | pbcopy` (Linux: `base64 -w0 upload.jks`). Set `FPACK_ANDROID_KEY_PASSWORD` too if it differs from the keystore password. fpack writes the keystore to `build/fpack/secrets/` with mode 0600 and deletes it after the build. Details: [Android](/platforms/android#in-ci).

## iOS and macOS

```yaml
  apple:
    runs-on: macos-latest
    steps:
      - uses: actions/checkout@v7
      - uses: subosito/flutter-action@v2
      - run: dart pub global activate fpack 1.1.18 && echo "$HOME/.pub-cache/bin" >> $GITHUB_PATH
      - run: fpack build ipa dmg --json > result.json
```

Signed and notarized macOS builds need certificates in a temporary keychain and notary credentials: [macOS: export the certificates for CI](/platforms/macos-credentials#ci) has the complete import step with an App Store Connect API key. Use `--notarize-no-wait` plus a later `fpack notarize finish` if you prefer not to hold a runner while Apple processes the upload.

## Windows

```yaml
  windows:
    runs-on: windows-latest
    steps:
      - uses: actions/checkout@v7
      - uses: subosito/flutter-action@v2
      - run: dart pub global activate fpack 1.1.18
      - run: fpack build windows exe
```

Inno Setup is preinstalled on GitHub's Windows runners; for code signing, base64 the `.pfx` into a secret and decode it on the runner ([Windows](/platforms/windows#5-sign-the-code)).

## Linux and web

```yaml
  linux:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v7
      - uses: subosito/flutter-action@v2
      - run: sudo apt-get install -y ninja-build libgtk-3-dev rpm
      - run: dart pub global activate fpack 1.1.18 && echo "$HOME/.pub-cache/bin" >> $GITHUB_PATH
      - run: fpack build linux deb rpm web
```

For snaps on `ubuntu-24.04`, set `linux.snap.build_mode: destructive`; use `ubuntu-24.04-arm` for an arm64 snap ([Linux](/platforms/linux#7-snap)).

## Tips

- **Cache the core**: the native core (about 4 MB) is downloaded once per version into `~/.cache/fpack` (Linux), `~/Library/Caches/fpack` (macOS) or `%LOCALAPPDATA%\fpack` (Windows). Cache that folder to skip the download.
- **Machine-readable results**: `fpack build … --json > result.json` writes every artifact with path, size, SHA-256 and notarization state; human output goes to stderr.
- **Fail fast**: exit code 3 means a prerequisite is missing; nothing was built.
- **A version per run**: pass `--build-number "$GITHUB_RUN_NUMBER"` so reruns never collide with existing artifacts ([versions](/guides/project-and-output#versions)).
- **Preview first**: `fpack build --all --dry-run` in a pull request shows what a release would do.
