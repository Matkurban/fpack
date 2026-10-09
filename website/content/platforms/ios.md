---
title: "iOS"
description: "IPA export, signing and flavors."
---

```bash
fpack build ipa                                   # Flutter default (App Store Connect)
fpack build ipa --export-method ad-hoc
fpack build ipa --export-options-plist ios/ExportOptions.plist
fpack build ipa --no-codesign                     # unsigned IPA for re-signing later
```

`doctor` checks Xcode, CocoaPods (when there is a Podfile), the project's `DEVELOPMENT_TEAM` and signing certificates; on failure the key Xcode errors (certificates, provisioning profiles, team, pods…) are translated into concrete fixes.

## Export options

| Setting | fpack.yaml | Flag / env |
| --- | --- | --- |
| Export method | `ios.export_method` | `--export-method`, `FPACK_IOS_EXPORT_METHOD` |
| Your own ExportOptions.plist | `ios.export_options_plist` | `--export-options-plist`, `FPACK_IOS_EXPORT_OPTIONS_PLIST` |
| Unsigned IPA | `ios.codesign: false` | `--no-codesign`, `FPACK_IOS_CODESIGN` |
| Team / signing style / profiles | `ios.team_id`, `ios.signing_style`, `ios.provisioning_profiles` | – |

With `ios.team_id` (and optionally `signing_style`, `signing_certificate`, `provisioning_profiles`, `export_options`) fpack generates the ExportOptions.plist in `build/fpack/` – your project is not modified.

## Flavors

iOS flavors are Xcode schemes. fpack detects shared, per-user and workspace schemes; when a project has no custom schemes the flavor is not passed to `flutter build ipa` (with a note), so a flavor used only on Android does not break iOS builds.

## Uploading

The IPA in `dist/` can be uploaded with Transporter or `xcrun altool --upload-app`, e.g. from a `post_package.ipa` hook.
