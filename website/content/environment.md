---
title: "Environment variables"
description: "All FPACK_* variables and the variables fpack reads."
---

| Variable | Effect |
| --- | --- |
| `FPACK_FLUTTER` | Flutter SDK path |
| `FPACK_CONFIG` | configuration file path |
| `FPACK_MODE` `FPACK_FLAVOR` `FPACK_ENTRY` `FPACK_BUILD_NAME` `FPACK_BUILD_NUMBER` | build parameters |
| `FPACK_OUTPUT_DIR` `FPACK_OVERWRITE` `FPACK_SPLIT_PER_ABI` `FPACK_OBFUSCATE` | output / Android / obfuscation |
| `FPACK_ANDROID_KEYSTORE` `FPACK_ANDROID_KEYSTORE_BASE64` `FPACK_ANDROID_KEYSTORE_PASSWORD` `FPACK_ANDROID_KEY_ALIAS` `FPACK_ANDROID_KEY_PASSWORD` | Android signing |
| `FPACK_IOS_EXPORT_METHOD` `FPACK_IOS_EXPORT_OPTIONS_PLIST` `FPACK_IOS_CODESIGN` | iOS export |
| `FPACK_MACOS_SIGN` `FPACK_MACOS_SIGN_IDENTITY` `FPACK_MACOS_INSTALLER_IDENTITY` `FPACK_MACOS_NOTARIZE` `FPACK_MACOS_NOTARY_PROFILE` `FPACK_NOTARIZE_WAIT` `FPACK_DMG_TOOL` | macOS |
| `FPACK_NOTARY_APPLE_ID` `FPACK_NOTARY_TEAM_ID` `FPACK_NOTARY_PASSWORD` `FPACK_NOTARY_API_KEY` `FPACK_NOTARY_API_KEY_ID` `FPACK_NOTARY_API_ISSUER` | macOS notary credentials (Apple ID / API key) |
| `FPACK_WINDOWS_CERTIFICATE` `FPACK_WINDOWS_CERTIFICATE_PASSWORD` `FPACK_WINDOWS_CERT_THUMBPRINT` | Windows code signing |
| `FPACK_LANG` | `zh` / `en` (default: `LC_ALL`, `LC_MESSAGES`, `LANG`, then the macOS/Windows system language) |
| `NO_COLOR` / `FPACK_NO_COLOR` / `FORCE_COLOR` | colors |
| `FPACK_CORE` | use this native core binary (development) |
| `FPACK_HOME` | core cache directory |
| `FPACK_REBUILD=1` | build the core with local Go first (after changing the Go sources) |
| `FPACK_GO` | the Go used for the fallback build (`none` disables local builds) |
| `FPACK_NO_DOWNLOAD=1` / `FPACK_DOWNLOAD_URL` | never download (build locally) / download mirror (intranet); see [how the launcher finds the native core](/installation#launcher) |

## Output language

fpack (the Go core and the Dart launcher) speaks Chinese when your operating system's language is Chinese (any `zh` variant: Simplified, Traditional, Hong Kong…) and English otherwise. Everything follows it: messages, help, `doctor` / `list` output, error hints, the comments of the `fpack.yaml` written by `fpack init`, and `NOTARIZATION.md`.

Precedence (first match wins):

1. `--lang zh|en`
2. `FPACK_LANG=zh|en`
3. the operating system language:
   - **macOS**: the first language in System Settings → General → Language & Region (`defaults read -g AppleLanguages`), then `AppleLocale`. Terminal usually sets `LANG` from the *region* (often `en_US.UTF-8` even with a Chinese UI), so `LANG` / `LANGUAGE` are only used if that lookup fails; an explicit `LC_ALL` or `LC_MESSAGES` still wins. The result is cached in `~/Library/Caches/fpack/os-language` and refreshed when the system preferences change.
   - **Windows**: the user's display language (`GetUserDefaultUILanguage`, no subprocess); `LANG` set by Git Bash/MSYS is ignored, an explicit `LC_ALL` / `LC_MESSAGES` still wins.
   - **Linux and others**: `LC_ALL`, `LC_MESSAGES`, `LANGUAGE` (first entry), `LANG`; `C` / `POSIX` are skipped.

## Variables that map to fpack.yaml keys

Generated from the key registry. They override `fpack.yaml` and are overridden by command-line flags. Any value in `fpack.yaml` can also reference arbitrary variables with `${VAR}` / `${VAR:-default}`.

<!-- BEGIN GENERATED ENV -->
| Variable | fpack.yaml key | Type | Description |
| --- | --- | --- | --- |
| `FPACK_FLUTTER` | `flutter.sdk` | path | Flutter SDK root to use. |
| `FPACK_MODE` | `build.mode` | `release` \\| `profile` \\| `debug` | Build mode. |
| `FPACK_FLAVOR` | `build.flavor` | string | Flavor / Xcode scheme (--flavor); appears in file names as {flavor}. |
| `FPACK_ENTRY` | `build.target` | path | Entry point (flutter -t). |
| `FPACK_BUILD_NAME` | `build.build_name` | string/number | Version name ({version}). |
| `FPACK_BUILD_NUMBER` | `build.build_number` | string/number | Build number ({build}). |
| `FPACK_OBFUSCATE` | `build.obfuscate` | bool | Obfuscate Dart code; symbols go to split_debug_info (default <output>/debug-info/<platform>). |
| `FPACK_OUTPUT_DIR` | `output.dir` | string | Artifact directory (relative to the project; placeholders allowed). |
| `FPACK_OVERWRITE` | `output.overwrite` | bool | Replace existing artifacts instead of stopping. |
| `FPACK_SPLIT_PER_ABI` | `android.split_per_abi` | `false` \\| `true` \\| `both` | false = one universal APK, true = one APK per ABI, both = universal + per-ABI. |
| `FPACK_ANDROID_KEYSTORE` | `android.signing.store_file` | path | Keystore file (.jks/.keystore). FPACK_ANDROID_KEYSTORE_BASE64 can provide it in CI. |
| `FPACK_ANDROID_KEYSTORE_PASSWORD` | `android.signing.store_password` | string | Keystore password. |
| `FPACK_ANDROID_KEY_ALIAS` | `android.signing.key_alias` | string | Key alias. |
| `FPACK_ANDROID_KEY_PASSWORD` | `android.signing.key_password` | string | Key password. |
| `FPACK_IOS_EXPORT_METHOD` | `ios.export_method` | `app-store-connect` \\| `app-store` \\| `release-testing` \\| `ad-hoc` \\| `development` \\| `debugging` \\| `enterprise` | IPA export method. |
| `FPACK_IOS_EXPORT_OPTIONS_PLIST` | `ios.export_options_plist` | path | Your own ExportOptions.plist; wins over every generated option below. |
| `FPACK_IOS_CODESIGN` | `ios.codesign` | bool | false = unsigned IPA (Payload/ zip) for re-signing later. |
| `FPACK_MACOS_SIGN` | `macos.sign.enabled` | bool | Re-sign the .app with Developer ID (and sign the DMG). true without identity picks the first "Developer ID Application" identity. false (--no-sign) also disables pkg signing and notarization. |
| `FPACK_MACOS_SIGN_IDENTITY` | `macos.sign.identity` | string | codesign identity for the app; setting it turns signing on. |
| `FPACK_MACOS_NOTARIZE` | `macos.sign.notarize` | bool | Notarize and staple the zip, DMG and signed pkg (uploads to Apple). |
| `FPACK_MACOS_NOTARY_PROFILE` | `macos.sign.notary_profile` | string | Keychain profile from `xcrun notarytool store-credentials <name>` (recommended locally). |
| `FPACK_NOTARY_APPLE_ID` | `macos.sign.notary_apple_id` | string | Apple ID for notarization (with notary_team_id + notary_password), instead of a profile. |
| `FPACK_NOTARY_TEAM_ID` | `macos.sign.notary_team_id` | string | Team ID for Apple ID notarization. |
| `FPACK_NOTARY_PASSWORD` | `macos.sign.notary_password` | string | App-specific password for Apple ID notarization. |
| `FPACK_NOTARY_API_KEY` | `macos.sign.notary_api_key` | path | App Store Connect API key (.p8) for notarization (CI friendly). |
| `FPACK_NOTARY_API_KEY_ID` | `macos.sign.notary_api_key_id` | string | API key ID. |
| `FPACK_NOTARY_API_ISSUER` | `macos.sign.notary_api_issuer` | string | API issuer UUID (omit for individual keys). |
| `FPACK_MACOS_INSTALLER_IDENTITY` | `macos.sign.installer_identity` | string | Signs the .pkg; a separate certificate from the app's "Developer ID Application". Unset = unsigned pkg. |
| `FPACK_NOTARIZE_WAIT` | `macos.notarize.wait` | bool | true: stay attached until Apple answers (shows elapsed time; Ctrl-C stops waiting, the submission continues at Apple). false: submit, write NOTARIZATION.md / notarization.json and finish with status "submitted"; later run `fpack notarize finish` to staple. |
| `FPACK_DMG_TOOL` | `macos.dmg.tool` | `auto` \\| `hdiutil` \\| `create-dmg` | auto = create-dmg when installed (or when layout keys are set), else hdiutil. |
| `FPACK_WINDOWS_CERTIFICATE` | `windows.sign.certificate` | path | Code signing certificate (.pfx). Setting it (or thumbprint) signs the app .exe, the installer and the MSIX. |
| `FPACK_WINDOWS_CERTIFICATE_PASSWORD` | `windows.sign.password` | string | Certificate password. |
| `FPACK_WINDOWS_CERT_THUMBPRINT` | `windows.sign.thumbprint` | string | SHA-1 thumbprint of a certificate in the Windows certificate store (instead of a .pfx). |
<!-- END GENERATED ENV -->
