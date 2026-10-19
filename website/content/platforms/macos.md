---
title: "macOS"
description: "Build a zipped app, a DMG or a pkg installer, sign it with Developer ID and notarize it with Apple."
---

This tutorial builds macOS apps for distribution **outside the Mac App Store**: a zipped `.app` (`macos`), a disk image (`dmg`) and an installer package (`pkg`), signed with a Developer ID certificate and notarized by Apple so Gatekeeper opens them without warnings. All three build on macOS only and share a single `flutter build macos`.

## What you need

- A Mac with **Xcode** (and CocoaPods when the project has a Podfile).
- For signing and notarization: a paid Apple Developer Program membership, a **Developer ID Application** certificate (and **Developer ID Installer** for `pkg`) in your keychain, and one notarization credential. [Certificates and credentials](/platforms/macos-credentials) shows how to get each.

You can follow the first steps without any of that.

## 1. Check the machine

```bash
fpack doctor macos dmg pkg
```

Doctor checks Xcode, CocoaPods, `hdiutil` / `create-dmg`, `pkgbuild` and `productbuild`, lists the signing identities in your keychain and validates provisioning profiles and entitlements.

## 2. A first, unsigned build

```bash
fpack build macos dmg
```

With nothing configured, the `.app` keeps the signature Xcode gave it, the DMG is unsigned, and notes in the summary explain how to enable signing. You get:

```text
dist/1.0.0+1/
  my_app-1.0.0+1-macos-universal.zip
  my_app-1.0.0+1-macos-universal.dmg
```

This is fine for testing on your own Mac. On other Macs, Gatekeeper blocks unsigned, un-notarized downloads: continue with step 3.

::: tip
For quick local packages after signing is set up, add `--no-notarize` (skips the Apple upload) or `--no-sign` (skips signing and notarization).
:::

## 3. Configure signing and notarization

Signing is enabled only through fpack's own settings: `macos.sign` in `fpack.yaml`, `FPACK_MACOS_*` variables, or flags. Highest precedence last: file → variables → flags.

```yaml
macos:
  sign:
    identity: "Developer ID Application: Your Name (ABCDE12345)"   # turns signing on
    notary_profile: fpack-notary                                    # turns notarization on
```

The rules:

- an `identity` turns signing on; any notary credential turns notarization on; turning signing off also turns notarization off;
- `--sign` without an identity picks the first "Developer ID Application" certificate in your keychain; a missing identity lists the available ones;
- `--notarize` (or `notarize: true`) without any credential uses the keychain profile named `NotaryProfile`;
- signing uses the **Hardened Runtime** (`macos.sign.hardened_runtime`, on by default, required for notarization) and the entitlements in `macos/Runner/Release.entitlements` (or `macos.sign.entitlements`);
- fpack does not read other tools' settings in `pubspec.yaml`, such as the `dmg:` section of the `dmg` package.

There are three kinds of notarization credential, in priority order: a keychain profile (`notary_profile`, best on your own Mac), an App Store Connect API key (`notary_api_key`, `notary_api_key_id`, `notary_api_issuer`, best for CI) and an Apple ID with an app-specific password (`notary_apple_id`, `notary_team_id`, `notary_password`). How to create each is on [Certificates and credentials](/platforms/macos-credentials#notary).

| Setting | fpack.yaml | Variable | Flag |
| --- | --- | --- | --- |
| signing on / off | `macos.sign.enabled` | `FPACK_MACOS_SIGN` | `--sign`, `--no-sign` |
| app identity | `macos.sign.identity` | `FPACK_MACOS_SIGN_IDENTITY` | `--sign-identity` |
| pkg identity | `macos.sign.installer_identity` | `FPACK_MACOS_INSTALLER_IDENTITY` | `--installer-identity` |
| notarization on / off | `macos.sign.notarize` | `FPACK_MACOS_NOTARIZE` | `--notarize`, `--no-notarize` |
| keychain profile | `macos.sign.notary_profile` | `FPACK_MACOS_NOTARY_PROFILE` | `--notary-profile` |
| provisioning profile | `macos.sign.provisioning_profile` | `FPACK_MACOS_PROVISIONING_PROFILE` | `--provisioning-profile` |
| smoke test | `macos.smoke_test` | `FPACK_MACOS_SMOKE_TEST` | `--smoke-test` |

## 4. Build, sign and notarize

```bash
fpack build macos dmg
```

For a DMG, fpack copies the `.app`, signs the nested code with `codesign --deep`, re-signs the app with the Hardened Runtime and entitlements, verifies it, builds the disk image, signs the DMG, uploads it with `xcrun notarytool submit`, waits for Apple, staples the ticket with `stapler staple` and checks the result with `spctl`. For the zip (`macos`), the zip is submitted, the `.app` is stapled, then it is zipped again.

## 5. Waiting for notarization

Apple usually needs a few minutes. fpack shows the elapsed time and writes two files to the output folder as soon as the upload has a submission id: `NOTARIZATION.md` (artifact, SHA-256, submission id, the credential used but never a password, and copy-paste commands) and `notarization.json`.

::: note
Ctrl-C stops *waiting*; Apple keeps processing. You can continue later, from any shell on this Mac.
:::

You can also not wait at all:

```bash
fpack build dmg --notarize-no-wait      # submit, write the record, finish
fpack notarize status                   # ask Apple about every pending submission
fpack notarize finish                   # wait, staple accepted files, update SHA256SUMS
```

- `fpack notarize status [dir|ID]` reports the state of each submission recorded in `notarization.json` (or of one submission id).
- `fpack notarize finish [dir]` waits for the results, staples accepted DMGs and pkgs (an app zip is unpacked, the app stapled and zipped again), rewrites `SHA256SUMS`, and for rejected submissions downloads Apple's log (`notary-log-<file>.json`) and summarizes the issues.
- The same behavior as `--notarize-no-wait` can be set with `macos.notarize.wait: false` or `FPACK_NOTARIZE_WAIT=false`. The summary and `--json` then report `notarization.state: submitted`.
- Apple ID submissions need `FPACK_NOTARY_PASSWORD` set when you run `notarize`.

A rejected submission (`Invalid`) usually means unsigned nested code or a missing Hardened Runtime; the summarized log says which file.

## 6. Verify the result

```bash
spctl --assess --type open --context context:primary-signature -vv dist/1.0.0+1/my_app-1.0.0+1-macos-universal.dmg
xcrun stapler validate dist/1.0.0+1/my_app-1.0.0+1-macos-universal.dmg
codesign --verify --deep --strict --verbose=2 /Applications/MyApp.app
```

## 7. Make the DMG yours

fpack builds DMGs with `hdiutil` by default and with `create-dmg` when it is installed or when you set layout keys (`macos.dmg.tool: auto`). Force one with `--dmg-tool hdiutil|create-dmg`.

```yaml
macos:
  dmg:
    volume_name: My App
    background: macos/dmg/background.png
    window_size: [660, 400]
    window_position: [200, 120]
    icon_size: 128
    app_position: [180, 190]
    applications_position: [480, 190]
    format: ULFO
```

`volume_icon` and `license` (a license agreement shown on mount) are also available; `format` and `filesystem` work with both tools. Install `create-dmg` with `brew install create-dmg`.

## 8. A pkg installer

```bash
fpack build pkg
```

fpack runs `pkgbuild` to create a component package that installs the app into `/Applications` (not relocatable, so upgrades replace the copy there), then `productbuild` to create a distribution package that supports Apple silicon and Intel without a Rosetta prompt.

```yaml
macos:
  pkg:
    identifier: com.example.app.pkg
    title: My App
    welcome: macos/installer/welcome.html
    license: macos/installer/license.rtf
    min_os: "11.0"
    postinstall: macos/installer/postinstall.sh
```

- **Signing**: a pkg needs a separate **Developer ID Installer** certificate (`macos.sign.installer_identity`, `--installer-identity`); the result is checked with `pkgutil --check-signature`. Without it you get an unsigned pkg and a note; `--no-sign` turns pkg signing off.
- **Notarization**: shares the DMG settings; only signed pkgs are submitted and stapled, then assessed with `spctl --assess --type install`.
- More options: `readme`, `conclusion`, `background`, `install_location`, `version`, `preinstall`, `require_restart`, `relocatable` (scripts are made executable automatically). `min_os` defaults to the project's `MACOSX_DEPLOYMENT_TARGET`.

## 9. Apps with push, iCloud or other restricted entitlements

If your app uses restricted entitlements (push notifications, iCloud, associated domains, keychain access groups), a Developer ID provisioning profile is required, otherwise the signed app will not launch. Read [Provisioning profiles](/platforms/macos-profiles); add `--smoke-test` to catch it before notarization.

## Flavors and CI

Flavors are Xcode schemes ([Flavors](/guides/build-variants#flavors)). For signed builds in CI you import certificates into a temporary keychain: [export the certificates for CI](/platforms/macos-credentials#ci) has the full workflow.

## If something goes wrong

| Message | Fix |
| --- | --- |
| `signing identity … not found in the keychain` | use one of the listed identities exactly as printed, or `--no-sign` |
| notarization `Invalid` | read the summarized Apple log; usually unsigned nested code or missing Hardened Runtime |
| upload timeouts while notarizing | fpack retries the upload twice; check the network and build again (nothing was submitted) |
| the signed app does not launch (`RBSRequestErrorDomain Code=5`) | restricted entitlements without a Developer ID profile: [Provisioning profiles](/platforms/macos-profiles) |
| pkg is unsigned | set `macos.sign.installer_identity` (a different certificate from the app's) |

More on the [troubleshooting page](/troubleshooting#macos).
