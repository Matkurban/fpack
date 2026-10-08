# Changelog

## 0.1.0

- First release.
- Targets: apk (universal / per-ABI / both), aab, ipa (export method, ExportOptions.plist, unsigned), macos (zip, Developer ID signing, optional notarization), dmg (hdiutil / create-dmg, signing, notarization, stapling), windows (portable zip), exe (Inno Setup), msix, linux (tar.gz), deb, rpm, AppImage, web (zip).
- Commands: build, doctor, list, init, clean, version.
- Android signing injected via environment (no Gradle edits), keystore verification, signer check.
- macOS signing defaults read from the pubspec `dmg:` section (read-only).
- `--dry-run`, `--json`, zh/en output, CI-friendly output, Ctrl-C handling, SHA256SUMS.
- Dart launcher with verified bundled binaries, Go build fallback and checksum-verified downloads.
