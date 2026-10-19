---
title: "iOS"
description: "Build an IPA for the App Store, TestFlight or ad hoc distribution: team, export methods, profiles and upload."
---

This tutorial produces an `.ipa` for App Store Connect (TestFlight and the App Store), an ad hoc IPA for registered devices, or an unsigned IPA you sign later. Target: `ipa`. It builds on macOS only.

## What you need

- A Mac with **Xcode** (and **CocoaPods** if the project has a Podfile).
- A paid Apple Developer Program membership, your **Team ID** (ten characters, shown in the Apple Developer portal under Membership) and an app id (`PRODUCT_BUNDLE_IDENTIFIER`).
- Xcode signed in to your account (Xcode → Settings → Accounts) so that automatic signing can create certificates and profiles.

## 1. Check the machine

```bash
fpack doctor ipa
```

Doctor checks Xcode, CocoaPods, the team (`ios.team_id`, or the project's `DEVELOPMENT_TEAM`) and the signing certificates in your keychain. Problems come with the command that fixes them.

## 2. Tell fpack your team

If the Xcode project already has a development team (Runner → Signing & Capabilities), nothing is needed. Otherwise set it. This also makes fpack generate the `ExportOptions.plist` for you; your project is not modified.

```yaml
ios:
  team_id: ABCDE12345
```

## 3. Build the App Store IPA

```bash
fpack build ipa
```

This runs `flutter build ipa` with Flutter's default export method (App Store Connect) and copies the result to `dist/1.0.0+1/my_app-1.0.0+1-ios-arm64.ipa`. When the build fails, the key Xcode errors (certificates, provisioning profiles, team, pods…) are translated into concrete fixes.

## 4. Upload it

Upload with **Transporter** (Mac App Store), with `xcrun altool --upload-app`, or let Xcode do it: set the export destination to `upload` and fpack's generated options send the build to App Store Connect itself.

```yaml
ios:
  team_id: ABCDE12345
  destination: upload       # export (default) writes the IPA only; upload sends it to App Store Connect
```

You can also upload from a [hook](/guides/hooks): `post_package.ipa: [xcrun altool --upload-app …]`.

::: tip
App Store Connect needs a new build number for every upload. Bump the number after `+` in `pubspec.yaml`, or pass `--build-number 43`. With `ios.manage_app_version_and_build_number: true` (the default) Apple may also manage it for the distribution.
:::

## Other ways to distribute

### Export methods

| Method | For |
| --- | --- |
| `app-store-connect` (default), `app-store` | TestFlight and the App Store |
| `release-testing`, `ad-hoc` | specific registered devices |
| `development`, `debugging` | development devices |
| `enterprise` | in-house distribution (Enterprise Program) |

```bash
fpack build ipa --export-method ad-hoc
```

The same setting is `ios.export_method` in `fpack.yaml` and `FPACK_IOS_EXPORT_METHOD`.

### Manual signing with provisioning profiles

By default Xcode manages signing. For manual signing (typical on CI) say which certificate and which profile signs each bundle id, including app extensions:

```yaml
ios:
  export_method: ad-hoc
  team_id: ABCDE12345
  signing_style: manual
  signing_certificate: Apple Distribution
  provisioning_profiles:
    com.example.app: My App AdHoc
    com.example.app.share: My App Share AdHoc
```

fpack writes these into the generated `ExportOptions.plist` (`build/fpack/`). Any other ExportOptions key can be added under `ios.export_options` (for example `iCloudContainerEnvironment: Production`), and `ios.thinning`, `ios.upload_symbols` and `ios.strip_swift_symbols` map to their plist keys.

### Your own ExportOptions.plist

```bash
fpack build ipa --export-options-plist ios/ExportOptions.plist
```

Your file wins over every generated option. Setting: `ios.export_options_plist`, `FPACK_IOS_EXPORT_OPTIONS_PLIST`.

### Unsigned IPA

```bash
fpack build ipa --no-codesign
```

Builds with `--no-codesign` and zips the `Payload/` folder, for example to re-sign later in another system. Setting: `ios.codesign: false`, `FPACK_IOS_CODESIGN`.

## Flavors

iOS flavors are Xcode schemes. fpack detects shared, per-user and workspace schemes. When a project has no custom schemes, the flavor is not passed to `flutter build ipa` (with a note), so a flavor that exists only on Android does not break iOS builds. See [Flavors](/guides/build-variants#flavors).

## In CI

Run on a `macos-latest` runner with certificates imported into a temporary keychain; the macOS credentials page shows the keychain steps ([export the certificates for CI](/platforms/macos-credentials#ci)). Then:

```yaml
- run: fpack build ipa --export-method app-store-connect --json > result.json
```

## If something goes wrong

| Message | Fix |
| --- | --- |
| no signing certificate / `No Accounts` | sign in to Xcode, or import the certificate and profile |
| `No profiles for 'com.example.app' were found` | with manual signing, list the bundle id (and extensions) in `ios.provisioning_profiles` |
| the team name in the hint is wrong | set `ios.team_id` (it wins over the project's `DEVELOPMENT_TEAM`) |
| `pod install` fails | run `pod repo update`, or `cd ios && pod install` to see the error |
| build number already used | bump `+N` in `pubspec.yaml` or pass `--build-number` |

More on the [troubleshooting page](/troubleshooting#ios).
