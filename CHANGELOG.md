# Changelog

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
