---
title: "Changelog"
description: "Changes in every fpack release."
---

<!-- BEGIN GENERATED CHANGELOG -->
## 1.1.6

### Added
- AI agent skill: the package now ships the [package skill](https://dart.dev/tools/pub/package-skills) `fpack-cli` (`skills/fpack-cli/SKILL.md`), which teaches AI coding agents how to use fpack (install, `init`, `doctor`, targets, configuration and `FPACK_*` variables, signing, macOS notarization, CI, troubleshooting). Install it with `dart pub add dev:fpack` and `dart run skills@ get --package fpack`. Documented on the website (AI agent skill page) and in the READMEs.

### Changed
- The CI examples in the READMEs, the website and the skill pin the current fpack version and the action versions used by fpack's own workflows; a test keeps them current.

## 1.1.5

### Changed
- The output language follows the operating system language: Chinese (any `zh` variant) → Chinese, everything else → English. Precedence: `--lang` > `FPACK_LANG` > OS language. On macOS the UI language (`AppleLanguages`) now wins over Terminal's `LANG` (usually derived from the region, e.g. `en_US.UTF-8` on a Chinese system); on Windows the display language (`GetUserDefaultUILanguage`) wins over `LANG` from Git Bash; explicit `LC_ALL` / `LC_MESSAGES` still override. Linux reads `LC_ALL`, `LC_MESSAGES`, `LANGUAGE`, `LANG`. The macOS lookup is cached and refreshed when the system preferences change.
- The Dart launcher's own messages (downloading/building the core, errors and fixes) are now localized too.
- Every user-facing text follows the selected language: config validation errors, usage lines, "did you mean" suggestions, the `error:` prefix, example values in the `fpack init` template and docs. Tests check that English output contains no Chinese and Chinese output (including the generated `fpack.yaml` and `NOTARIZATION.md`) has no English prose left.

## 1.1.4

### Added
- Documentation website: https://matkurban.github.io/fpack/ (English) and https://matkurban.github.io/fpack/zh/ (Chinese) – getting started, every command, every `fpack.yaml` key and `FPACK_*` variable (generated from the core, so it stays accurate), platform guides (Android signing, iOS, macOS signing + notarization incl. how to get certificates and credentials, Windows, Linux, web), CI recipes, troubleshooting, FAQ, search and dark mode. Built with Jaspr in `website/`, deployed by GitHub Actions.
- `example/`: Aurora Notes, a small Flutter app with Android flavors, `--dart-define-from-file` per flavor, hooks, Linux packaging metadata and a fully commented `fpack.yaml` (signing via environment variables, macOS dmg/pkg, Windows exe/msix, Linux deb/rpm/AppImage, web). Packaged for real in E2E.

### Changed
- Documentation links printed by the CLI (`fpack help`, config errors, `fpack init` header, signing notes) and in the JSON schema now point to the documentation website; `pubspec.yaml` lists it as `documentation`.

### Fixed
- A flavor that only exists on Android (e.g. `build.flavor: prod` with Gradle productFlavors) no longer breaks iOS/macOS builds of a project whose Xcode projects define no custom schemes: the flavor is not passed to `flutter build ios/macos` there (Flutter would reject it), with a note. Scheme detection now also includes per-user and workspace schemes, like `xcodebuild -list`.

## 1.1.3

### Changed
- An unknown `--flavor` / `build.flavor` is now an error (exit 3) when the project defines flavors (Android productFlavors, shared Xcode schemes), with a "did you mean" suggestion or the list of available flavors. If no flavors are detected it stays a warning, since detection is heuristic.
- `fpack doctor` (and `fpack targets`) no longer list `apk`/`aab` as ready to build when Java is missing or older than 17; builds still try (Gradle may find a JDK fpack cannot see).
- `hooks.post_build` is skipped when every target failed its checks before building (nothing was built); it still runs after build failures with `FPACK_SUCCESS=0`.
- The JSON schema accepts target aliases in `build.targets` (`bundle`, `ios`, `setup`, …), matching the command line.

### Fixed
- `hooks.post_build` documentation: `FPACK_SUCCESS` is `1`/`0`.
- `${VAR}` interpolation now works in every key, including booleans, numbers, enums and lists (e.g. `sign: ${SIGN}`), as documented; previously typed keys failed to parse.
- `build.targets` in `fpack.yaml` is validated even when targets are given on the command line, with "did you mean" suggestions (typo suggestions now treat swapped letters as one edit: `wbe` → `web`).
- Unknown placeholders in `output.name` / `output.names.<target>` are reported by validation instead of ending up literally in file names.
- Wrong-type sections now say "expected a section of keys" instead of Go type names.
- JSON schema: accepts `${VAR}` for non-string keys, validates `android.abis` and the target names under `output.names`.
- All missing tools of a target are reported at once (not one per run); flavor hints no longer have a missing target name; `doctor` no longer repeats the same tool line for deb/rpm/appimage.
- The monorepo hint quotes project paths with spaces (`fpack -C 'my app'`).
- Launcher download: honors `HTTPS_PROXY`/`HTTP_PROXY`/`NO_PROXY`, and gives up after 30 s without data (instead of hanging) so it falls back to the next resolution step.
- README: accurate key count and description of environment overrides.

### CI
- E2E builds projects in paths with spaces and CJK characters on Linux and Windows, including the monorepo-root hint and running from a subdirectory.

## 1.1.2

### Fixed
- macOS notarization: `notarytool submit` uploads that fail because of the network (e.g. `abortedUpload … HTTPClientError.deadlineExceeded` from Apple's S3 storage) are retried twice (after 20 s and 60 s) instead of failing the target; nothing is recorded until Apple returns a submission id.
- Failed build steps now pick their hint from the command output first: an upload timeout no longer suggests re-creating the keychain profile ("create the profile once …"); it explains that the network failed and nothing was submitted.

## 1.1.1

### Launcher: verified prebuilt cores first
- The Dart launcher now prefers a verified prebuilt binary over a local `go build`. Resolution order: `FPACK_CORE` → cached core of the matching version → bundled `prebuilt/` binary (SHA-256 from `manifest.json`) → download from the GitHub Release (SHA-256 from `checksums.txt`) → **only if the download fails or is disabled**, a local build from the bundled Go sources.
- The local fallback is announced clearly: `fpack: note: no verified prebuilt core available (<reason>)` followed by the Go version and path used, and a confirmation after the build.
- Controls: `FPACK_NO_DOWNLOAD=1` (offline: bundled binary or local build), `FPACK_DOWNLOAD_URL` (mirror), `FPACK_GO=<path>` / `FPACK_GO=none` (pick Go / never build locally — a failed download is then a clear error with download links), `FPACK_REBUILD=1` (developers: build locally first), `FPACK_HOME`.
- `fpack --wrapper-info` shows the Go version and the resolution order.
- When nothing works, the error lists every attempt plus concrete fixes (network / mirror, install Go, or download the binary and set `FPACK_CORE`).

### Documentation
- `README.md` is now English (shown on pub.dev); the Chinese README moved to `README.ZH.md`; `README.en.md` was removed. Both have a language switch at the top.
- New section on obtaining macOS signing and notarization credentials (README, README.ZH and `doc/configuration.md` §2.13): Developer ID Application/Installer certificates, `notarytool store-credentials` keychain profile (Apple ID + app-specific password from account.apple.com + Team ID), App Store Connect API key (.p8, Key ID, Issuer ID), exporting certificates as .p12 from Keychain Access and importing them on CI runners, mapped to the fpack.yaml keys and `FPACK_*` variables.
- Hints in the core point to `README.md` / `README.ZH.md` / `doc/configuration.md`.

## 1.1.0

### Everything configurable
- One key registry drives `FPACK_*` environment variables, validation, the JSON schema (`schema/fpack.schema.json`, `fpack schema`), the `fpack init` template and the key/env tables in `doc/configuration.md` — they can no longer drift apart. 188 keys in total, 121 new:
  - **app**: `copyright`, `license`, `support_url`; `publisher` falls back to the CompanyName in `windows/runner/Runner.rc`.
  - **build**: `tree_shake_icons`. **output**: `names` (per-target templates), `checksum_algorithm` (`sha256` | `sha512`); new placeholders `{target}` and `{date}`.
  - **hooks**: `pre_build`, `pre_package.<target>`, `post_package.<target>`, `post_build` with `FPACK_*` environment (artifact paths, version, success).
  - **android**: `project_args`, `signing.v1`–`v4` (re-sign with apksigner, v4 writes `.idsig`).
  - **ios**: `team_id`, `signing_style`, `signing_certificate`, `provisioning_profiles`, `destination`, `thinning`, `upload_symbols`, `strip_swift_symbols`, `manage_app_version_and_build_number`, `export_options` → fpack generates ExportOptions.plist.
  - **macos**: `sign.hardened_runtime`, notary API key (`notary_api_key`, `notary_api_key_id`, `notary_api_issuer`) and Apple ID (`notary_apple_id`, `notary_team_id`, `notary_password`) credentials, `notarize.wait`; DMG layout (`background` via create-dmg, `window_position`, `window_size`, `icon_size`, `app_position`, `applications_position`, `volume_icon`, `license`, `format`, `filesystem`); pkg `min_os`, `require_restart`, `relocatable`, `preinstall`, `postinstall`, `version`.
  - **windows**: `sign.*` (signtool with .pfx or thumbprint: app exe, Inno installer + uninstaller, msix; `timestamp_url`, `description`, `signtool`); Inno Setup `publisher`, `publisher_url`, `support_url`, `updates_url`, `default_dir`, `group_name`, `desktop_icon`, `run_after_install`, `license_file`, `info_before`, `info_after`, `setup_icon`, `wizard_image`, `wizard_small_image`, `wizard_style`, `compression`, `min_version`, `privileges`, `languages` (bundled Chinese translation); `msix.*` metadata (display name, publisher, identity, capabilities, languages, logo, file extensions, protocol activation, execution alias, start at login, store, version, certificate, sign, os_min_version).
  - **linux**: `prefix`, `generic_name`, `keywords`, `mime_types`, `startup_wm_class`, `icon_sizes` (hicolor theme), `metainfo`; deb `section`, `priority`, `recommends`, `suggests`, `conflicts`, `preinst`/`postinst`/`prerm`/`postrm`; rpm `group`, `license`, `pre`/`post`/`preun`/`postun`; AppImage `update_information`, `extra_args`.
  - **web**: `source_maps`, `optimization_level`, `csp`, `web_define`, `static_assets_url`, `web_resources_cdn`.
- Validation: unknown keys with "did you mean", type errors with line numbers, enums and ranges, conflicting combinations, missing files for the targets being built.
- `fpack init` lists every key with zh/en comments (`--lang`), allowed values, defaults, examples and env/flag names, pre-filled from the project (ids, flavor, publisher, define files, team id, Inno AppId…); asks for the default flavor when Android flavors exist; never overwrites: an existing file gets `fpack.yaml.new` + a diff (`--force` replaces). The generated file references the JSON schema for editor completion.
- New environment variables: `FPACK_NOTARIZE_WAIT`, `FPACK_NOTARY_APPLE_ID`, `FPACK_NOTARY_TEAM_ID`, `FPACK_NOTARY_PASSWORD`, `FPACK_NOTARY_API_KEY`, `FPACK_NOTARY_API_KEY_ID`, `FPACK_NOTARY_API_ISSUER`, `FPACK_WINDOWS_CERTIFICATE`, `FPACK_WINDOWS_CERTIFICATE_PASSWORD`, `FPACK_WINDOWS_CERT_THUMBPRINT`.

### Friendlier macOS notarization
- Right after `notarytool submit` returns an id, fpack writes `NOTARIZATION.md` (zh/en) and `notarization.json` into the output directory: artifact + SHA-256, submission id, time, the credential kind (never secrets) and copy-paste `notarytool info/wait/log`, `stapler staple/validate`, `spctl` and `fpack notarize` commands. The files are updated with the final status; for Invalid submissions Apple's log is downloaded and summarized with fixes.
- While waiting: elapsed time and a hint that Ctrl-C only stops waiting (the submission continues at Apple); Ctrl-C prints where the record is and keeps the uploaded file.
- `macos.notarize.wait: false` / `FPACK_NOTARIZE_WAIT=false` / `--notarize-no-wait`: submit and finish; the summary and `--json` mark the artifact `submitted`.
- New commands `fpack notarize status [dir|id]` and `fpack notarize finish [dir]` (wait, staple DMG/pkg/app zip, rewrite checksums, summarize rejections).

### Other
- `fpack schema` command.
- `--json`: artifacts carry `notarization` (state, id, status, record path).

### Fixed
- Android projects with productFlavors but no flavor failed only after a full Gradle build ("Gradle build failed to produce an .apk file"); fpack now stops at preflight and suggests `--flavor`.
- A flavor set for Linux/Windows/Web printed a warning per target and leaked into file names; it is now a single note and `{flavor}` is empty for platforms without flavor support.
- apksigner re-signing (`android.signing.v1`–`v4`) failed when the output directory did not exist yet; APKs are now signed in the staging directory and moved into place.
- Real builds ignored `output.names` and the `pre_package` / `post_package` hooks (only `--dry-run` applied them).
- `appimagetool` wrote the `.zsync` file (with `linux.appimage.update_information`) into the project root; it now runs in the staging directory and the `.zsync` lands next to the AppImage in the output directory (listed in the checksums).
- The APK signature check verifies the configured schemes (v1 with `--min-sdk-version 21`, v4 with the `.idsig` file), so the summary note lists all of them.
- A DMG left in the staging directory by a failed run (e.g. rejected by notarization) made the next `create-dmg` refuse ("Output file already exists"); the whole DMG stage is reset now.
- The deb `copyright` file had a stray space after removing "©" from `app.copyright`.

## 1.0.0

- First release.
- Targets: apk (universal / per-ABI / both), aab, ipa (export method, ExportOptions.plist, unsigned), macos (zip, Developer ID signing, optional notarization), dmg (hdiutil / create-dmg, signing, notarization, stapling), pkg (pkgbuild + productbuild installer into /Applications, Developer ID Installer signing, notarization, stapling), windows (portable zip), exe (Inno Setup), msix, linux (tar.gz), deb, rpm, AppImage, web (zip).
- Commands: build, doctor, list, init, clean, version.
- Android signing injected via environment (no Gradle edits), keystore verification, signer check.
- macOS Developer ID signing / notarization configured only through fpack.yaml `macos.sign` (+ `installer_identity` for pkg), `FPACK_MACOS_*` and flags; unsigned (Xcode signature kept) when not configured. The pubspec `dmg:` section of the `dmg` package is not read (not in build, doctor or init).
- `fpack init` writes a commented `macos.sign` / `macos.pkg` section with placeholders and lists the keychain's Developer ID identities as hints.
- `--dry-run`, `--json`, zh/en output, CI-friendly output, Ctrl-C handling, SHA256SUMS.
- Dart launcher with verified bundled binaries, Go build fallback and checksum-verified downloads.
- Complete configuration reference: [doc/configuration.md](doc/configuration.md).
- Tested end to end on a real app (Android apk/aab, iOS ipa, macOS app/dmg) and in CI on Windows, macOS and Linux.
<!-- END GENERATED CHANGELOG -->
