---
title: "Configuration"
description: "fpack.yaml: validation, environment interpolation, hooks and every key."
---

Every key is optional. `fpack init` writes a file that **lists every key**, each with an English (`--lang zh`: Chinese) comment describing what it does, its allowed values, default, an example and the matching environment variable / flag, pre-filled with values detected in the project (application ID, flavors, version, team ID, Inno AppId…); leave the keys you do not need commented out.

- **Validation**: unknown keys (with "did you mean …"), type errors (`line 12: android.signing.v1: expected true or false, got "maybe"`), allowed values and ranges, conflicting combinations; files used by the selected targets are checked before building.
- **Editor completion**: the first line of the generated file is `# yaml-language-server: $schema=…/schema/fpack.schema.json`, so VS Code (YAML extension) / JetBrains give completion, hover docs (English/Chinese) and validation. `fpack schema -o fpack.schema.json` exports a local copy.
- **Environment variables**: `${VAR}` / `${VAR:-default}` are expanded on load in any value (`obfuscate: ${OBF:-false}` works too); build, signing and notarization settings also have dedicated `FPACK_*` variables (see the [full list](/environment)).

```yaml
# yaml-language-server: $schema=https://raw.githubusercontent.com/Matkurban/fpack/main/schema/fpack.schema.json
app:
  display_name: XueHua IM
  publisher: XueHua Tech             # default: CompanyName in windows/runner/Runner.rc
  homepage: https://xuehua.example.com
  license: MIT

build:
  targets: [apk, aab, ipa, dmg]      # built by `fpack build` without targets
  flavor: prod
  dart_define_from_file: [config/prod.json]

output:
  dir: "dist/{version}{+build}"
  names: { exe: "{app}-setup-{version}", web: "{app}-web" }   # per-target names; {target} {date} also work
  checksum_algorithm: sha512         # SHA512SUMS

hooks:
  pre_build: [dart run build_runner build --delete-conflicting-outputs]
  post_package: { dmg: ['./scripts/upload.sh "$FPACK_ARTIFACT"'] }

android:
  split_per_abi: both
  signing:
    store_file: ~/keys/upload.jks
    store_password: ${FPACK_ANDROID_KEYSTORE_PASSWORD}
    key_alias: upload
    v4: true                         # any of v1–v4 re-signs with apksigner; v4 also writes the .idsig

ios:
  export_method: ad-hoc
  team_id: ABCDE12345                # fpack then generates ExportOptions.plist
  provisioning_profiles: { com.xuehua.im: XueHua AdHoc }

macos:
  sign:
    identity: "Developer ID Application: XueHua Tech (ABCDE12345)"
    notary_profile: XueHua           # or notary_api_key/_id/_issuer, or notary_apple_id/_team_id/_password
  notarize:
    wait: true                       # false: submit without waiting (staple later with fpack notarize finish)
  dmg: { background: macos/dmg/bg.png, window_size: [660, 400], format: ULFO }
  pkg: { min_os: "11.0", postinstall: macos/installer/postinstall.sh }

windows:
  inno_setup: { languages: [zh-CN, en], privileges: ask, desktop_icon: checked }
  sign: { certificate: C:/certs/codesign.pfx, password: ${WINDOWS_CERT_PASSWORD} }
  msix: { publisher: "CN=XueHua Tech", capabilities: [internetClient] }

linux:
  categories: [Network, InstantMessaging]
  metainfo: linux/com.xuehua.im.metainfo.xml
  deb: { depends: ["libgtk-3-0 | libgtk-3-0t64"], postinst: linux/postinst }
  rpm: { requires: [gtk3], license: MIT }

web:
  base_href: /app/
  source_maps: true
```

**Hooks** (`hooks`): `pre_build` (once before the first flutter build; a failure stops the run), `pre_package.<target>` / `post_package.<target>` (around packaging, with `FPACK_TARGET`, `FPACK_ARTIFACT`, `FPACK_ARTIFACTS`), `post_build` (once at the end, with `FPACK_ARTIFACTS`, `FPACK_SUCCESS` = `1`/`0`; it also runs after build failures, but is skipped when every target already failed its checks, e.g. a missing tool or unknown flavor). All hooks run in the project root through `sh -c` (Windows: `cmd /C`) with `FPACK_PROJECT_ROOT`, `FPACK_OUTPUT_DIR`, `FPACK_VERSION`, `FPACK_BUILD_NUMBER`, `FPACK_MODE`, `FPACK_FLAVOR`; write `$VAR` for variables that only exist while the hook runs (`${VAR}` is expanded when the configuration is loaded). A failing hook means exit code 1.

## File name templates

`output.name` and `output.names.<target>` accept these placeholders; `{-x}` / `{+x}` add the separator only when the value is not empty:

| Placeholder | Value |
| --- | --- |
| `{app}` | `app.name` (default: pubspec `name`) |
| `{version}` / `{build}` | build name / build number |
| `{flavor}` | the flavor (empty where Flutter does not use it) |
| `{platform}` | `android`, `ios`, `macos`, `windows`, `linux`, `web` |
| `{arch}` | `arm64-v8a`, `universal`, `x64`, … |
| `{variant}` | `setup`, `portable`, … |
| `{mode}` | empty for release, otherwise `profile` / `debug` |
| `{target}` | the target name |
| `{date}` | build date `YYYYMMDD` |

## Key reference

Every key of `fpack.yaml` (generated from fpack's key registry – the same source as validation, the JSON Schema and `fpack init`).

<!-- BEGIN GENERATED KEYS -->
### `app`

Application metadata used by installers and packages.

| Key | Type | Default | Targets | Env / flag | Description |
| --- | --- | --- | --- | --- | --- |
| `app.name` | string | `pubspec name` | all |  | Base name of artifact files ({app} in output.name). Example: `xue_hua_im` |
| `app.display_name` | string | `macOS PRODUCT_NAME, else pubspec name` | exe, msix, pkg, deb, rpm, appimage, linux |  | Human-readable app name: installer title, Start menu, .desktop Name=. Example: `XueHua IM` |
| `app.description` | string | `pubspec description` | deb, rpm, appimage, msix |  | Short description: deb Description, rpm Summary, .desktop Comment=, msix description. Example: `A fast and secure messenger` |
| `app.publisher` | string | `CompanyName in windows/runner/Runner.rc` | exe, msix, deb, rpm |  | Company / author: Windows installer publisher, msix publisher display name, deb Maintainer fallback, rpm Vendor. Example: `XueHua Tech` |
| `app.identifier` | string | `Linux APPLICATION_ID, Android applicationId or iOS bundle id` | exe, msix, pkg, appimage |  | Reverse-DNS app id: Inno Setup AppId seed, msix identity name, pkg identifier fallback. Example: `com.xuehua.im` |
| `app.homepage` | url | — | exe, deb, rpm |  | Website: Inno Setup publisher URL, deb Homepage, rpm URL. Example: `https://xuehua.example.com` |
| `app.support_url` | url | `app.homepage` | exe |  | Support link (Windows "Apps & features"). Example: `https://xuehua.example.com/support` |
| `app.maintainer` | string | `app.publisher` | deb, rpm |  | "Name <email>" for deb Maintainer and rpm Packager. Example: `XueHua Team <dev@xuehua.example.com>` |
| `app.copyright` | string | `© <year> <publisher>` | exe, deb, rpm |  | Copyright line: Inno Setup AppCopyright and version info, deb copyright file. Example: `© 2026 XueHua Tech` |
| `app.license` | string | `Proprietary` | rpm, deb |  | License (SPDX id): rpm License, deb copyright file. Example: `MIT` |

### `flutter`

Flutter SDK selection.

| Key | Type | Default | Targets | Env / flag | Description |
| --- | --- | --- | --- | --- | --- |
| `flutter.sdk` | path | `FLUTTER_ROOT, fvm, PATH` | all | `FPACK_FLUTTER`<br>`--flutter` | Flutter SDK root to use. Example: `~/fvm/versions/stable` |

### `build`

Options for every flutter build.

| Key | Type | Default | Targets | Env / flag | Description |
| --- | --- | --- | --- | --- | --- |
| `build.targets` | list (or one string) | — | — |  | Targets built by `fpack build` without arguments. Aliases accepted by the command line (e.g. `bundle`, `ios`, `setup`) work here too. Example: `[apk, aab, ipa, dmg]` |
| `build.mode` | `release` \\| `profile` \\| `debug` | `release` | all | `FPACK_MODE`<br>`--mode` | Build mode. Example: `release` |
| `build.flavor` | string | — | apk, aab, ipa, macos, dmg, pkg | `FPACK_FLAVOR`<br>`--flavor` | Flavor / Xcode scheme (--flavor); appears in file names as {flavor}. Example: `prod` |
| `build.target` | path | `lib/main.dart` | all | `FPACK_ENTRY`<br>`-t, --target` | Entry point (flutter -t). Example: `lib/main_prod.dart` |
| `build.dart_define` | map or `KEY=VALUE` list | — | all | `--dart-define` | --dart-define values (map or list of KEY=VALUE). Example: `{API_URL: https://api.example.com}` |
| `build.dart_define_from_file` | list (or one string) | — | all | `--dart-define-from-file` | --dart-define-from-file files (.json or .env). Example: `[config/prod.json]` |
| `build.build_name` | string/number | `pubspec version before +` | all | `FPACK_BUILD_NAME`<br>`--build-name` | Version name ({version}). Example: `1.2.0` |
| `build.build_number` | string/number | `pubspec version after +` | all | `FPACK_BUILD_NUMBER`<br>`--build-number` | Build number ({build}). Example: `42` |
| `build.obfuscate` | bool | `false` | apk, aab, ipa, macos, dmg, pkg, windows, exe, msix, linux, deb, rpm, appimage | `FPACK_OBFUSCATE`<br>`--obfuscate` | Obfuscate Dart code; symbols go to split_debug_info (default <output>/debug-info/<platform>). Example: `true` |
| `build.split_debug_info` | string | `<output>/debug-info/<platform> when obfuscating` | apk, aab, ipa, macos, dmg, pkg, windows, exe, msix, linux, deb, rpm, appimage | `--split-debug-info` | Directory for Dart debug symbols (--split-debug-info). Example: `build/symbols` |
| `build.tree_shake_icons` | bool | `true` | all |  | false = --no-tree-shake-icons (keep all icon font glyphs). Example: `false` |
| `build.extra_args` | list (or one string) | — | all |  | Extra arguments for every flutter build (also: everything after -- on the command line). Example: `[--no-pub]` |

### `output`

Where artifacts go and how they are named.

| Key | Type | Default | Targets | Env / flag | Description |
| --- | --- | --- | --- | --- | --- |
| `output.dir` | string | `dist/{version}{+build}` | all | `FPACK_OUTPUT_DIR`<br>`-o, --output` | Artifact directory (relative to the project; placeholders allowed). Example: `"dist/{version}{+build}"` |
| `output.name` | string | `{app}{-flavor}-{version}{+build}-{platform}{-arch}{-variant}{-mode}` | all |  | File name template without extension. Placeholders: {app} {version} {build} {platform} {arch} {variant} {mode} {flavor} {target} {date}; {-x}/{+x} add the separator only when x is set. Example: `"{app}-{version}-{platform}{-arch}"` |
| `output.names` | map | — | all |  | Per-target file name templates (target → template), overriding output.name. Example: `{exe: "{app}-setup-{version}", web: "{app}-web"}` |
| `output.overwrite` | bool | `false` | all | `FPACK_OVERWRITE`<br>`-f, --force` | Replace existing artifacts instead of stopping. Example: `true` |
| `output.checksums` | bool | `true` | all |  | Write a checksum file next to the artifacts. Example: `false` |
| `output.checksum_algorithm` | `sha256` \\| `sha512` | `sha256` | all |  | Checksum algorithm (file SHA256SUMS or SHA512SUMS). Example: `sha512` |

### `hooks`

Shell commands run around the build (working directory: project root).

| Key | Type | Default | Targets | Env / flag | Description |
| --- | --- | --- | --- | --- | --- |
| `hooks.pre_build` | list (or one string) | — | all |  | Commands run once before the first flutter build; a failure stops the build. Example: `[dart run build_runner build --delete-conflicting-outputs]` |
| `hooks.post_build` | list (or one string) | — | all |  | Commands run once after all targets (also after build failures, but not when every target already failed its checks); FPACK_ARTIFACTS lists the produced files (one per line), FPACK_SUCCESS is 1/0. Example: `[./scripts/upload.sh]` |
| `hooks.pre_package` | map: target → commands | — | all |  | Per target (target → commands), before its packaging steps; FPACK_TARGET is set. Example: `{apk: [./scripts/check_size.sh]}` |
| `hooks.post_package` | map: target → commands | — | all |  | Per target, after its artifacts exist; FPACK_ARTIFACT (first file) and FPACK_ARTIFACTS are set. Example: `{dmg: [./scripts/upload_dmg.sh "$FPACK_ARTIFACT"]}` |

### `android`

Android: apk, aab.

| Key | Type | Default | Targets | Env / flag | Description |
| --- | --- | --- | --- | --- | --- |
| `android.split_per_abi` | `false` \\| `true` \\| `both` | `false` | apk | `FPACK_SPLIT_PER_ABI`<br>`--split-per-abi` | false = one universal APK, true = one APK per ABI, both = universal + per-ABI. Example: `both` |
| `android.abis` | list (or one string) | `armeabi-v7a, arm64-v8a, x86_64` | apk, aab | `--abis` | Target ABIs (--target-platform). Example: `[arm64-v8a, armeabi-v7a]` |
| `android.project_args` | map | — | apk, aab |  | Gradle project properties (flutter -P key=value), readable in build.gradle with project.findProperty (e.g. to toggle minify/R8). Example: `{minify: "true"}` |
| `android.extra_args` | list (or one string) | — | apk, aab |  | Extra arguments for flutter build apk/appbundle. Example: `[--android-skip-build-dependency-validation]` |

#### `android.signing`

Release signing, injected without editing Gradle files. Keep passwords in environment variables.

| Key | Type | Default | Targets | Env / flag | Description |
| --- | --- | --- | --- | --- | --- |
| `android.signing.store_file` | path | `none (android/key.properties of the project is used)` | apk, aab | `FPACK_ANDROID_KEYSTORE` | Keystore file (.jks/.keystore). FPACK_ANDROID_KEYSTORE_BASE64 can provide it in CI. Example: `~/keys/upload.jks` |
| `android.signing.store_password` | string | — | apk, aab | `FPACK_ANDROID_KEYSTORE_PASSWORD` | Keystore password. Example: `${KEYSTORE_PASSWORD}` |
| `android.signing.key_alias` | string | — | apk, aab | `FPACK_ANDROID_KEY_ALIAS` | Key alias. Example: `upload` |
| `android.signing.key_password` | string | `store_password` | apk, aab | `FPACK_ANDROID_KEY_PASSWORD` | Key password. Example: `${KEY_PASSWORD}` |
| `android.signing.v1` | bool | `apksigner default (on when minSdk < 24)` | apk |  | APK signature scheme v1 (JAR signing, needed below Android 7). Setting any of v1-v4 makes fpack re-sign the APKs with apksigner using android.signing. Example: `true` |
| `android.signing.v2` | bool | `true` | apk |  | APK signature scheme v2 (Android 7+). Example: `true` |
| `android.signing.v3` | bool | `true` | apk |  | APK signature scheme v3 (Android 9+, key rotation). Example: `true` |
| `android.signing.v4` | bool | `false` | apk |  | APK signature scheme v4 (incremental install, Android 11+); writes <apk>.idsig next to the APK. Example: `true` |

### `ios`

iOS: ipa. Setting any of team_id … export_options makes fpack generate ExportOptions.plist.

| Key | Type | Default | Targets | Env / flag | Description |
| --- | --- | --- | --- | --- | --- |
| `ios.export_method` | `app-store-connect` \\| `app-store` \\| `release-testing` \\| `ad-hoc` \\| `development` \\| `debugging` \\| `enterprise` | `app-store-connect` | ipa | `FPACK_IOS_EXPORT_METHOD`<br>`--export-method` | IPA export method. Example: `ad-hoc` |
| `ios.export_options_plist` | path | — | ipa | `FPACK_IOS_EXPORT_OPTIONS_PLIST`<br>`--export-options-plist` | Your own ExportOptions.plist; wins over every generated option below. Example: `ios/ExportOptions.plist` |
| `ios.codesign` | bool | `true` | ipa | `FPACK_IOS_CODESIGN`<br>`--no-codesign` | false = unsigned IPA (Payload/ zip) for re-signing later. Example: `false` |
| `ios.team_id` | string | `DEVELOPMENT_TEAM of the Xcode project` | ipa |  | Apple team ID (teamID). Example: `ABCDE12345` |
| `ios.signing_style` | `automatic` \\| `manual` | `automatic` | ipa |  | signingStyle: automatic or manual (manual needs provisioning_profiles). Example: `manual` |
| `ios.signing_certificate` | string | — | ipa |  | signingCertificate (manual signing), e.g. "Apple Distribution". Example: `Apple Distribution` |
| `ios.provisioning_profiles` | map | — | ipa |  | provisioningProfiles: bundle id → profile name or UUID (include extensions). Example: `{com.xuehua.im: XueHua AdHoc}` |
| `ios.upload_symbols` | bool | `true` | ipa |  | uploadSymbols for App Store Connect. Example: `false` |
| `ios.manage_app_version_and_build_number` | bool | `true` | ipa |  | manageAppVersionAndBuildNumber: let App Store Connect bump the build number. Example: `false` |
| `ios.destination` | `export` \\| `upload` | `export` | ipa |  | export = write the IPA locally, upload = send it to App Store Connect (ExportOptions destination). Example: `upload` |
| `ios.thinning` | string | `<none>` | ipa |  | thinning for ad-hoc/development/enterprise exports. Example: `<thin-for-all-variants>` |
| `ios.strip_swift_symbols` | bool | `true` | ipa |  | stripSwiftSymbols. Example: `false` |
| `ios.export_options` | map (any) | — | ipa |  | Any other ExportOptions.plist keys (added as-is). Example: `{iCloudContainerEnvironment: Production}` |
| `ios.extra_args` | list (or one string) | — | ipa |  | Extra arguments for flutter build ipa. Example: `[--no-tree-shake-icons]` |

### `macos`

macOS: macos (.app zip), dmg, pkg.

| Key | Type | Default | Targets | Env / flag | Description |
| --- | --- | --- | --- | --- | --- |
| `macos.extra_args` | list (or one string) | — | macos, dmg, pkg |  | Extra arguments for flutter build macos. Example: `[--no-tree-shake-icons]` |

#### `macos.sign`

Developer ID signing and notarization (only from this file, FPACK_MACOS_* and flags).

| Key | Type | Default | Targets | Env / flag | Description |
| --- | --- | --- | --- | --- | --- |
| `macos.sign.enabled` | bool | `true when identity is set` | macos, dmg, pkg | `FPACK_MACOS_SIGN`<br>`--sign / --no-sign` | Re-sign the .app with Developer ID (and sign the DMG). true without identity picks the first "Developer ID Application" identity. false (--no-sign) also disables pkg signing and notarization. Example: `true` |
| `macos.sign.identity` | string | — | macos, dmg, pkg | `FPACK_MACOS_SIGN_IDENTITY`<br>`--sign-identity` | codesign identity for the app; setting it turns signing on. Example: `"Developer ID Application: Your Name (TEAMID)"` |
| `macos.sign.entitlements` | path | `macos/Runner/Release.entitlements` | macos, dmg, pkg |  | Entitlements used when re-signing the app. Example: `macos/Runner/Release.entitlements` |
| `macos.sign.hardened_runtime` | bool | `true` | macos, dmg, pkg |  | Sign with the hardened runtime (required for notarization). Example: `true` |
| `macos.sign.notarize` | bool | `true when a notary credential is set` | macos, dmg, pkg | `FPACK_MACOS_NOTARIZE`<br>`--notarize / --no-notarize` | Notarize and staple the zip, DMG and signed pkg (uploads to Apple). Example: `true` |
| `macos.sign.notary_profile` | string | — | macos, dmg, pkg | `FPACK_MACOS_NOTARY_PROFILE`<br>`--notary-profile` | Keychain profile from `xcrun notarytool store-credentials <name>` (recommended locally). Example: `NotaryProfile` |
| `macos.sign.notary_apple_id` | string | — | macos, dmg, pkg | `FPACK_NOTARY_APPLE_ID` | Apple ID for notarization (with notary_team_id + notary_password), instead of a profile. Example: `dev@example.com` |
| `macos.sign.notary_team_id` | string | — | macos, dmg, pkg | `FPACK_NOTARY_TEAM_ID` | Team ID for Apple ID notarization. Example: `ABCDE12345` |
| `macos.sign.notary_password` | string | — | macos, dmg, pkg | `FPACK_NOTARY_PASSWORD` | App-specific password for Apple ID notarization. Example: `${NOTARY_PASSWORD}` |
| `macos.sign.notary_api_key` | path | — | macos, dmg, pkg | `FPACK_NOTARY_API_KEY` | App Store Connect API key (.p8) for notarization (CI friendly). Example: `~/keys/AuthKey_ABC123.p8` |
| `macos.sign.notary_api_key_id` | string | — | macos, dmg, pkg | `FPACK_NOTARY_API_KEY_ID` | API key ID. Example: `ABC123DEF4` |
| `macos.sign.notary_api_issuer` | string | — | macos, dmg, pkg | `FPACK_NOTARY_API_ISSUER` | API issuer UUID (omit for individual keys). Example: `69a6de7e-…` |
| `macos.sign.installer_identity` | string | — | pkg | `FPACK_MACOS_INSTALLER_IDENTITY`<br>`--installer-identity` | Signs the .pkg; a separate certificate from the app's "Developer ID Application". Unset = unsigned pkg. Example: `"Developer ID Installer: Your Name (TEAMID)"` |

#### `macos.notarize`

How fpack waits for Apple's notary service. Every submission is recorded in <output>/NOTARIZATION.md and notarization.json with copy-paste commands.

| Key | Type | Default | Targets | Env / flag | Description |
| --- | --- | --- | --- | --- | --- |
| `macos.notarize.wait` | bool | `true` | macos, dmg, pkg | `FPACK_NOTARIZE_WAIT`<br>`--notarize-no-wait` | true: stay attached until Apple answers (shows elapsed time; Ctrl-C stops waiting, the submission continues at Apple). false: submit, write NOTARIZATION.md / notarization.json and finish with status "submitted"; later run `fpack notarize finish` to staple. Example: `false` |

#### `macos.dmg`

Disk image layout. Window/icon layout needs create-dmg (brew install create-dmg).

| Key | Type | Default | Targets | Env / flag | Description |
| --- | --- | --- | --- | --- | --- |
| `macos.dmg.tool` | `auto` \\| `hdiutil` \\| `create-dmg` | `auto` | dmg | `FPACK_DMG_TOOL`<br>`--dmg-tool` | auto = create-dmg when installed (or when layout keys are set), else hdiutil. Example: `create-dmg` |
| `macos.dmg.volume_name` | string | `.app name` | dmg |  | Volume name shown when the DMG is mounted. Example: `XueHua IM` |
| `macos.dmg.volume_icon` | path | — | dmg |  | Volume icon (.icns). create-dmg. Example: `macos/dmg/volume.icns` |
| `macos.dmg.background` | path | — | dmg |  | Window background image. create-dmg. Example: `macos/dmg/background.png` |
| `macos.dmg.window_position` | `[x, y]` | `[200, 120]` | dmg |  | Window position [x, y]. create-dmg. Example: `[200, 120]` |
| `macos.dmg.window_size` | `[x, y]` | `[660, 400]` | dmg |  | Window size [width, height]. create-dmg. Example: `[660, 400]` |
| `macos.dmg.icon_size` | int | `128` | dmg |  | Icon size in the window. create-dmg. Example: `128` |
| `macos.dmg.app_position` | `[x, y]` | `[180, 190]` | dmg |  | Position of the app icon [x, y]. create-dmg. Example: `[180, 190]` |
| `macos.dmg.applications_position` | `[x, y]` | `[480, 190]` | dmg |  | Position of the Applications link [x, y]. create-dmg. Example: `[480, 190]` |
| `macos.dmg.format` | `UDZO` \\| `UDBZ` \\| `ULFO` \\| `ULMO` \\| `UDRO` | `UDZO` | dmg |  | Image format: UDZO (zlib), UDBZ (bzip2), ULFO (lzfse, macOS 10.11+), ULMO (lzma, 10.15+), UDRO (read-only, uncompressed). Example: `ULFO` |
| `macos.dmg.filesystem` | `HFS+` \\| `APFS` | `HFS+` | dmg |  | Filesystem of the image. Example: `APFS` |
| `macos.dmg.license` | path | — | dmg |  | License agreement shown when the DMG is opened (.txt/.rtf). create-dmg. Example: `LICENSE.txt` |

#### `macos.pkg`

Installer package (pkgbuild + productbuild).

| Key | Type | Default | Targets | Env / flag | Description |
| --- | --- | --- | --- | --- | --- |
| `macos.pkg.identifier` | string | `macOS bundle id` | pkg |  | Package identifier (pkgutil --pkgs). Example: `com.xuehua.im` |
| `macos.pkg.version` | string | `build name` | pkg |  | Package version. Example: `1.2.0` |
| `macos.pkg.install_location` | string | `/Applications` | pkg |  | Absolute directory the app is installed into. Example: `/Applications` |
| `macos.pkg.title` | string | `.app name` | pkg |  | Installer window title. Example: `XueHua IM` |
| `macos.pkg.welcome` | path | — | pkg |  | Welcome page (.html/.rtf/.txt). Example: `macos/installer/welcome.html` |
| `macos.pkg.readme` | path | — | pkg |  | Read-me page (.html/.rtf/.txt). Example: `macos/installer/readme.html` |
| `macos.pkg.license` | path | — | pkg |  | License page the user must accept (.html/.rtf/.txt). Example: `macos/installer/license.rtf` |
| `macos.pkg.conclusion` | path | — | pkg |  | Conclusion page (.html/.rtf/.txt). Example: `macos/installer/done.html` |
| `macos.pkg.background` | path | — | pkg |  | Background image (light and dark mode). Example: `macos/installer/background.png` |
| `macos.pkg.min_os` | string | `MACOSX_DEPLOYMENT_TARGET of the project` | pkg |  | Minimum macOS version; Installer refuses older systems. Example: `10.15` |
| `macos.pkg.preinstall` | path | — | pkg |  | Script run before installing (made executable automatically). Example: `macos/installer/preinstall.sh` |
| `macos.pkg.postinstall` | path | — | pkg |  | Script run after installing. Example: `macos/installer/postinstall.sh` |
| `macos.pkg.relocatable` | bool | `false` | pkg |  | true = Installer updates a moved copy of the app wherever it is; false = always install_location. Example: `true` |
| `macos.pkg.require_restart` | bool | `false` | pkg |  | Ask the user to restart after installing. Example: `true` |

### `windows`

Windows: windows (zip), exe (Inno Setup), msix.

| Key | Type | Default | Targets | Env / flag | Description |
| --- | --- | --- | --- | --- | --- |
| `windows.extra_args` | list (or one string) | — | windows, exe, msix |  | Extra arguments for flutter build windows. Example: `[--no-tree-shake-icons]` |

#### `windows.inno_setup`

Inno Setup installer (.exe).

| Key | Type | Default | Targets | Env / flag | Description |
| --- | --- | --- | --- | --- | --- |
| `windows.inno_setup.app_id` | string | `stable GUID derived from app.identifier` | exe |  | AppId: keep it stable forever so upgrades replace the old install. Example: `8D3B5E6A-1C2D-4E5F-8A9B-0C1D2E3F4A5B` |
| `windows.inno_setup.script` | path | — | exe |  | Your own .iss script; fpack passes /DAppName /DAppVersion /DAppPublisher /DAppExeName /DSourceDir /DAppId /DAppURL. Example: `windows/installer.iss` |
| `windows.inno_setup.iscc` | path | `ISCC on PATH or in Program Files` | exe |  | Path to ISCC.exe. Example: `C:/Program Files (x86)/Inno Setup 6/ISCC.exe` |
| `windows.inno_setup.publisher` | string | `app.publisher` | exe |  | AppPublisher. Example: `XueHua Tech` |
| `windows.inno_setup.publisher_url` | url | `app.homepage` | exe |  | AppPublisherURL. Example: `https://xuehua.example.com` |
| `windows.inno_setup.support_url` | url | `app.support_url` | exe |  | AppSupportURL. Example: `https://xuehua.example.com/support` |
| `windows.inno_setup.updates_url` | url | — | exe |  | AppUpdatesURL. Example: `https://xuehua.example.com/download` |
| `windows.inno_setup.default_dir` | string | `{autopf}\<display name>` | exe |  | Default install directory (Inno constants allowed). Example: `'{autopf}\XueHua'` |
| `windows.inno_setup.group_name` | string | `display name` | exe |  | Start menu folder. Example: `XueHua` |
| `windows.inno_setup.desktop_icon` | `none` \\| `unchecked` \\| `checked` | `unchecked` | exe |  | Desktop shortcut task: none, unchecked (offered), checked (on by default). Example: `checked` |
| `windows.inno_setup.run_after_install` | bool | `true` | exe |  | Offer "Launch <app>" on the last page. Example: `false` |
| `windows.inno_setup.license_file` | path | — | exe |  | License page (.txt/.rtf). Example: `LICENSE.txt` |
| `windows.inno_setup.info_before` | path | — | exe |  | Information page before installing (.txt/.rtf). Example: `docs/before.txt` |
| `windows.inno_setup.info_after` | path | — | exe |  | Information page after installing (.txt/.rtf). Example: `docs/after.txt` |
| `windows.inno_setup.setup_icon` | path | `windows/runner/resources/app_icon.ico` | exe |  | Installer icon (.ico). Example: `windows/installer/setup.ico` |
| `windows.inno_setup.wizard_image` | path | — | exe |  | Large wizard image (.bmp/.png, 164×314 at 100%). Example: `windows/installer/wizard.bmp` |
| `windows.inno_setup.wizard_small_image` | path | — | exe |  | Small wizard image (.bmp/.png, 55×55). Example: `windows/installer/wizard-small.bmp` |
| `windows.inno_setup.wizard_style` | `modern` \\| `classic` | `modern` | exe |  | Wizard style. Example: `classic` |
| `windows.inno_setup.languages` | list (or one string) | `[en]` | exe |  | Installer languages; the first is the default, the user picks one if several. zh-CN/zh-TW need Inno Setup 6.5+ (fpack ships the files otherwise). Known: en zh-CN zh-TW ja ko de fr es it pt-BR pt ru uk tr pl nl cs ar he. Example: `[zh-CN, en]` |
| `windows.inno_setup.privileges` | `user` \\| `admin` \\| `ask` | `ask` | exe |  | user = per-user install (no UAC), admin = all users (UAC), ask = let the user choose. Example: `admin` |
| `windows.inno_setup.compression` | string | `lzma2/max` | exe |  | Compression (lzma2/max, lzma2/ultra64, zip, none …). Example: `lzma2/ultra64` |
| `windows.inno_setup.min_version` | string | `10.0` | exe |  | Minimum Windows version (MinVersion). Example: `10.0.17763` |

#### `windows.sign`

Authenticode signing with signtool: the app .exe and the installer.

| Key | Type | Default | Targets | Env / flag | Description |
| --- | --- | --- | --- | --- | --- |
| `windows.sign.certificate` | path | — | windows, exe, msix | `FPACK_WINDOWS_CERTIFICATE` | Code signing certificate (.pfx). Setting it (or thumbprint) signs the app .exe, the installer and the MSIX. Example: `C:/certs/codesign.pfx` |
| `windows.sign.password` | string | — | windows, exe, msix | `FPACK_WINDOWS_CERTIFICATE_PASSWORD` | Certificate password. Example: `${WINDOWS_CERT_PASSWORD}` |
| `windows.sign.thumbprint` | string | — | windows, exe | `FPACK_WINDOWS_CERT_THUMBPRINT` | SHA-1 thumbprint of a certificate in the Windows certificate store (instead of a .pfx). Example: `1A2B3C…` |
| `windows.sign.timestamp_url` | url | `http://timestamp.digicert.com` | windows, exe |  | RFC 3161 timestamp server. Example: `http://timestamp.sectigo.com` |
| `windows.sign.signtool` | path | `signtool on PATH or in the Windows SDK` | windows, exe |  | Path to signtool.exe. Example: `C:/Program Files (x86)/Windows Kits/10/bin/10.0.22621.0/x64/signtool.exe` |
| `windows.sign.description` | string | `display name` | windows, exe |  | Description shown in the UAC prompt (/d). Example: `XueHua IM` |

#### `windows.msix`

MSIX package (needs the msix dev_dependency). These keys override pubspec msix_config.

| Key | Type | Default | Targets | Env / flag | Description |
| --- | --- | --- | --- | --- | --- |
| `windows.msix.display_name` | string | `app.display_name` | msix |  | Display name. Example: `XueHua IM` |
| `windows.msix.publisher_display_name` | string | `app.publisher` | msix |  | Publisher display name. Example: `XueHua Tech` |
| `windows.msix.identity_name` | string | `app.identifier` | msix |  | Package identity name. Example: `com.xuehua.im` |
| `windows.msix.publisher` | string | `from the certificate` | msix |  | Publisher (certificate subject); required for the Store. Example: `CN=XueHua Tech, O=XueHua Tech, C=CN` |
| `windows.msix.version` | string | `build name as a.b.c.0` | msix |  | MSIX version (a.b.c.d). Example: `1.2.0.0` |
| `windows.msix.logo` | path | `msix default / app icon` | msix |  | Logo image (≥ 400×400 PNG). Example: `windows/msix/logo.png` |
| `windows.msix.description` | string | `app.description` | msix |  | Package description. Example: `A fast and secure messenger` |
| `windows.msix.capabilities` | list (or one string) | — | msix |  | Capabilities. Example: `[internetClient, microphone, webcam]` |
| `windows.msix.languages` | list (or one string) | — | msix |  | Languages. Example: `[zh-cn, en-us]` |
| `windows.msix.file_extensions` | list (or one string) | — | msix |  | File extensions the app opens. Example: `[.xhim]` |
| `windows.msix.protocol_activation` | list (or one string) | — | msix |  | URL protocols that activate the app. Example: `[xuehua]` |
| `windows.msix.execution_alias` | string | — | msix |  | Command-line alias. Example: `xuehua` |
| `windows.msix.start_at_login` | bool | `false` | msix |  | Start the app at login. Example: `true` |
| `windows.msix.os_min_version` | string | `10.0.17763.0` | msix |  | Minimum Windows version. Example: `10.0.19041.0` |
| `windows.msix.store` | bool | `false` | msix |  | Build for the Microsoft Store (unsigned, Store signs it). Example: `true` |
| `windows.msix.sign` | bool | `true` | msix |  | Sign the MSIX (false requires publisher). Example: `false` |
| `windows.msix.certificate` | path | `windows.sign.certificate, else msix test certificate` | msix |  | Certificate (.pfx) for the MSIX. Example: `C:/certs/codesign.pfx` |
| `windows.msix.certificate_password` | string | `windows.sign.password` | msix |  | Certificate password. Example: `${WINDOWS_CERT_PASSWORD}` |
| `windows.msix.extra_args` | list (or one string) | — | msix |  | Extra arguments for dart run msix:create. Example: `[--trim-logo, "false"]` |

### `linux`

Linux: linux (tar.gz), deb, rpm, appimage.

| Key | Type | Default | Targets | Env / flag | Description |
| --- | --- | --- | --- | --- | --- |
| `linux.package_name` | string | `app name, lowercase with dashes` | deb, rpm, appimage |  | deb/rpm package name and command name in /usr/bin. Example: `xuehua-im` |
| `linux.prefix` | string | `/opt/<package_name>` | deb, rpm |  | Directory the app bundle is installed into. Example: `/usr/lib/xuehua-im` |
| `linux.icon` | path | `flutter_launcher_icons image, else web/icons/Icon-512.png` | deb, rpm, appimage |  | PNG icon for menus and the AppImage. Example: `assets/icon/icon.png` |
| `linux.icon_sizes` | list of int | `[16, 32, 48, 64, 128, 256, 512]` | deb, rpm |  | hicolor icon sizes installed (resized from linux.icon; never upscaled). Example: `[48, 128, 256]` |
| `linux.categories` | list (or one string) | `[Utility]` | deb, rpm, appimage |  | freedesktop.org menu categories (.desktop Categories=). Example: `[Network, InstantMessaging]` |
| `linux.generic_name` | string | — | deb, rpm, appimage |  | .desktop GenericName=. Example: `Instant Messenger` |
| `linux.keywords` | list (or one string) | — | deb, rpm, appimage |  | .desktop Keywords= (search terms). Example: `[chat, im, message]` |
| `linux.mime_types` | list (or one string) | — | deb, rpm, appimage |  | .desktop MimeType= (files / URL schemes the app opens). Example: `[x-scheme-handler/xuehua]` |
| `linux.startup_wm_class` | string | `binary name (APPLICATION_ID on GTK)` | deb, rpm, appimage |  | .desktop StartupWMClass= (groups windows with the launcher). Example: `xue_hua_im` |
| `linux.metainfo` | path | — | deb, rpm, appimage |  | AppStream metainfo installed to /usr/share/metainfo (software centers). Example: `linux/packaging/com.xuehua.im.metainfo.xml` |
| `linux.appimagetool` | path | `appimagetool on PATH` | appimage |  | Path to appimagetool. Example: `~/.local/bin/appimagetool` |
| `linux.extra_args` | list (or one string) | — | linux, deb, rpm, appimage |  | Extra arguments for flutter build linux. Example: `[--no-tree-shake-icons]` |

#### `linux.deb`

Debian/Ubuntu package.

| Key | Type | Default | Targets | Env / flag | Description |
| --- | --- | --- | --- | --- | --- |
| `linux.deb.depends` | list (or one string) | `[libgtk-3-0 \| libgtk-3-0t64]` | deb |  | Depends. Example: `[libgtk-3-0, libsecret-1-0]` |
| `linux.deb.recommends` | list (or one string) | — | deb |  | Recommends. Example: `[gnome-keyring]` |
| `linux.deb.suggests` | list (or one string) | — | deb |  | Suggests. Example: `[libnotify-bin]` |
| `linux.deb.conflicts` | list (or one string) | — | deb |  | Conflicts. Example: `[xuehua-im-beta]` |
| `linux.deb.section` | string | `utils` | deb |  | Section. Example: `net` |
| `linux.deb.priority` | `required` \\| `important` \\| `standard` \\| `optional` \\| `extra` | `optional` | deb |  | Priority. Example: `optional` |
| `linux.deb.preinst` | path | — | deb |  | Maintainer script run before unpacking. Example: `linux/packaging/preinst` |
| `linux.deb.postinst` | path | — | deb |  | Maintainer script run after installing. Example: `linux/packaging/postinst` |
| `linux.deb.prerm` | path | — | deb |  | Maintainer script run before removal. Example: `linux/packaging/prerm` |
| `linux.deb.postrm` | path | — | deb |  | Maintainer script run after removal. Example: `linux/packaging/postrm` |

#### `linux.rpm`

Fedora/RHEL/openSUSE package.

| Key | Type | Default | Targets | Env / flag | Description |
| --- | --- | --- | --- | --- | --- |
| `linux.rpm.requires` | list (or one string) | `[gtk3]` | rpm |  | Requires. Example: `[gtk3, libsecret]` |
| `linux.rpm.group` | string | `Applications/Internet` | rpm |  | Group. Example: `Applications/Communications` |
| `linux.rpm.license` | string | `app.license` | rpm |  | License. Example: `MIT` |
| `linux.rpm.pre` | path | — | rpm |  | %pre script. Example: `linux/packaging/pre.sh` |
| `linux.rpm.post` | path | — | rpm |  | %post script. Example: `linux/packaging/post.sh` |
| `linux.rpm.preun` | path | — | rpm |  | %preun script. Example: `linux/packaging/preun.sh` |
| `linux.rpm.postun` | path | — | rpm |  | %postun script. Example: `linux/packaging/postun.sh` |

#### `linux.appimage`

AppImage.

| Key | Type | Default | Targets | Env / flag | Description |
| --- | --- | --- | --- | --- | --- |
| `linux.appimage.update_information` | string | — | appimage |  | Embedded update information (AppImageUpdate). Example: `gh-releases-zsync\|xuehua\|im\|latest\|*x86_64.AppImage.zsync` |
| `linux.appimage.extra_args` | list (or one string) | — | appimage |  | Extra arguments for appimagetool. Example: `[--comp, zstd]` |

### `web`

Web: web (zip).

| Key | Type | Default | Targets | Env / flag | Description |
| --- | --- | --- | --- | --- | --- |
| `web.base_href` | string | `index.html as is` | web | `--base-href` | <base href>; must start and end with /. Example: `/app/` |
| `web.wasm` | bool | `false` | web | `--wasm` | Compile to WebAssembly (with JS fallback). Example: `true` |
| `web.source_maps` | bool | `false` | web |  | Generate source maps. Example: `true` |
| `web.csp` | bool | `false` | web |  | No dynamic code generation (Content-Security-Policy). Example: `true` |
| `web.optimization_level` | int | `4` | web |  | dart2js / dart2wasm optimization level 0-4 (-O). Example: `2` |
| `web.static_assets_url` | url | — | web |  | Serve static assets from another domain (must end with /). Example: `https://cdn.example.com/app/` |
| `web.web_resources_cdn` | bool | `true` | web |  | false = bundle CanvasKit instead of loading it from the CDN. Example: `false` |
| `web.web_define` | map | — | web |  | --web-define template variables for web/index.html. Example: `{API_URL: https://api.example.com}` |
| `web.extra_args` | list (or one string) | — | web |  | Extra arguments for flutter build web. Example: `[--dump-info]` |

<!-- END GENERATED KEYS -->
