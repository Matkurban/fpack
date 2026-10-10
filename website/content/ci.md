---
title: "CI recipes"
description: "GitHub Actions workflows for every platform."
---

```yaml
jobs:
  android:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v7
      - uses: subosito/flutter-action@v2
      - run: dart pub global activate fpack 1.1.6 && echo "$HOME/.pub-cache/bin" >> $GITHUB_PATH
      - run: fpack build apk aab --split-per-abi=both
        env:
          FPACK_ANDROID_KEYSTORE_BASE64: ${{ secrets.KEYSTORE_BASE64 }}
          FPACK_ANDROID_KEYSTORE_PASSWORD: ${{ secrets.KEYSTORE_PASSWORD }}
          FPACK_ANDROID_KEY_ALIAS: upload
      - uses: actions/upload-artifact@v7
        with: { name: android, path: dist/ }
  apple:
    runs-on: macos-latest
    steps:
      - uses: actions/checkout@v7
      - uses: subosito/flutter-action@v2
      - run: dart pub global activate fpack 1.1.6 && echo "$HOME/.pub-cache/bin" >> $GITHUB_PATH
      - run: fpack build ipa dmg --json > result.json
  windows:
    runs-on: windows-latest
    steps:
      - uses: actions/checkout@v7
      - uses: subosito/flutter-action@v2
      - run: dart pub global activate fpack 1.1.6
      - run: fpack build windows exe
  linux:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v7
      - uses: subosito/flutter-action@v2
      - run: sudo apt-get install -y ninja-build libgtk-3-dev rpm
      - run: dart pub global activate fpack 1.1.6 && echo "$HOME/.pub-cache/bin" >> $GITHUB_PATH
      - run: fpack build linux deb rpm web
```


Without a TTY, spinners and colors are turned off and a heartbeat line is printed every 60 seconds so CI does not time out on silence.

## Tips

- **Pin the version**: `dart pub global activate fpack 1.1.6` makes CI reproducible.
- **Cache the core**: the native core is downloaded once per version into `~/.cache/fpack` (Linux), `~/Library/Caches/fpack` (macOS) or `%LOCALAPPDATA%\fpack` (Windows); cache that folder to skip the ~4 MB download.
- **Machine-readable results**: `fpack build … --json > result.json` writes every artifact with path, size, SHA-256 and notarization state; human output goes to stderr.
- **Fail fast**: exit code 3 means a prerequisite is missing – nothing was built.
- **Upload everything**: `dist/<version>+<build>/` contains the artifacts and `SHA256SUMS`.

## Signed macOS builds in CI

See [macOS → export the certificates for CI](/platforms/macos#credentials) for the complete keychain import step with an App Store Connect API key.

## Android signing secrets

```bash
base64 -i upload.jks | pbcopy   # → secret KEYSTORE_BASE64
```

Set `FPACK_ANDROID_KEYSTORE_BASE64`, `FPACK_ANDROID_KEYSTORE_PASSWORD`, `FPACK_ANDROID_KEY_ALIAS` (and `FPACK_ANDROID_KEY_PASSWORD` if it differs). fpack writes the keystore to `build/fpack/secrets/` with mode 0600 and deletes it after the build; passwords are redacted in logs.
