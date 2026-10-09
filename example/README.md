# fpack example: Aurora Notes

A small but complete Flutter app (Android, iOS, macOS, Windows, Linux, Web) with a
fully commented [`fpack.yaml`](fpack.yaml) that shows the common setups:

| Setup | Where |
| --- | --- |
| Android flavors `dev` / `prod` with per-flavor defines | `build.flavor`, `build.dart_define_from_file`, `android/app/build.gradle.kts` |
| Android release signing from environment variables (no keystore in git) | `android.signing`, `FPACK_ANDROID_*` |
| Per-ABI APKs + universal APK | `android.split_per_abi: both` |
| iOS ad-hoc export | `ios.export_method` |
| macOS DMG layout, pkg installer, Developer ID signing + notarization | `macos.dmg`, `macos.pkg`, `macos.sign` |
| Windows Inno Setup installer (multi-language) and MSIX | `windows.inno_setup`, `windows.msix` |
| Linux deb / rpm / AppImage with .desktop metadata, AppStream and maintainer scripts | `linux` |
| Web build with base href | `web` |
| File names, output folder, checksums, hooks | `output`, `hooks` |

📖 Full documentation: <https://matkurban.github.io/fpack/>

## Run fpack on it

```bash
dart pub global activate fpack       # once; make sure ~/.pub-cache/bin is on PATH

cd example                           # this folder
flutter pub get
fpack doctor                         # what can this machine build, what is missing
fpack build --dry-run                # show the exact commands for build.targets (apk, web)
fpack build                          # → dist/1.0.0+1/aurora-notes-prod-1.0.0+1-android-universal.apk …
```

More targets (each host builds its own platforms, Android and Web build anywhere):

```bash
fpack build apk aab                  # Android (flavor prod by default)
FPACK_FLAVOR=dev fpack build apk     # dev flavor + config/dev.json
fpack build linux deb rpm appimage   # on Linux – one flutter build, four packages
fpack build macos dmg pkg            # on macOS – zip, DMG and pkg installer
fpack build ipa --no-codesign        # on macOS without an Apple account
fpack build windows exe msix         # on Windows – zip, Inno Setup installer, MSIX
fpack build --all                    # everything this machine can build
```

Artifacts and `SHA256SUMS` go to `dist/1.0.0+1/`; temporary files to `build/fpack/`.
fpack never modifies the project's own files.

## Signing (optional)

Nothing secret is committed. Provide credentials through environment variables
(or CI secrets) and fpack picks them up:

```bash
# Android
export FPACK_ANDROID_KEYSTORE=~/keys/upload.jks
export FPACK_ANDROID_KEYSTORE_PASSWORD='…'
export FPACK_ANDROID_KEY_ALIAS=upload

# macOS (Developer ID + notarization with a keychain profile)
export FPACK_MACOS_SIGN_IDENTITY="Developer ID Application: Your Name (TEAMID)"
export FPACK_MACOS_INSTALLER_IDENTITY="Developer ID Installer: Your Name (TEAMID)"
export FPACK_MACOS_NOTARY_PROFILE=aurora-notary

# Windows
export FPACK_WINDOWS_CERTIFICATE='C:\certs\codesign.pfx'
export FPACK_WINDOWS_CERTIFICATE_PASSWORD='…'
```

See the [signing guides](https://matkurban.github.io/fpack/platforms/android) for how to
create keystores, certificates and notarization credentials.

## Files

```
lib/main.dart                 the app (reads API_BASE / FLAVOR_NAME / BUILD_CHANNEL defines)
config/dev.json, prod.json    per-flavor --dart-define-from-file values
fpack.yaml                    commented fpack configuration
packaging/                    icon, AppStream metainfo, deb/rpm scripts
scripts/post_package.sh       post_package hook example
LICENSE.txt                   shown by the Windows installer
```
