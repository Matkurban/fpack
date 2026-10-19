---
title: "macOS: provisioning profiles"
description: "Make apps with push, iCloud or other restricted entitlements launch after Developer ID signing, and catch problems before notarization."
---

Most apps need none of this page. You need it if your app's entitlements include **restricted** keys such as push notifications (`com.apple.developer.aps-environment`), iCloud, associated domains or keychain access groups.

## The problem

`flutter build macos` embeds Xcode's *development* provisioning profile ("Mac Team Provisioning Profile"). Once fpack re-signs the app with Developer ID, that profile no longer authorizes the restricted entitlements, and macOS refuses to launch the app (`RBSRequestErrorDomain Code=5`, POSIX 163, "Launch failed"), even though signing and notarization succeeded. The fix is a **Developer ID provisioning profile** for your app id.

## 1. Create the profile

You need to be Account Holder or Admin.

1. [developer.apple.com](https://developer.apple.com/account) → Certificates, IDs & Profiles → **Identifiers**: open the app id (for example `com.example.app`, the `PRODUCT_BUNDLE_IDENTIFIER` in `macos/Runner/Configs/AppInfo.xcconfig`) and enable the capabilities your entitlements use (Push Notifications, iCloud, Associated Domains, …). Save.
2. **Profiles** → **+** → under *Distribution* choose **Developer ID** → platform macOS → the app id → the **Developer ID Application** certificate you sign with. It must be that exact certificate: check its SHA-1 with `security find-identity -v -p codesigning`; a profile made for another certificate is rejected at launch. Name it, **Generate**, **Download**.
3. Put the `.provisionprofile` in the project (for example `macos/MyApp.provisionprofile`). It contains no secret, but you can keep it out of git and point to it from CI with `FPACK_MACOS_PROVISIONING_PROFILE`.
4. Make the release entitlements match the profile. Developer ID profiles authorize `aps-environment` **production**, so `macos/Runner/Release.entitlements` needs `<key>com.apple.developer.aps-environment</key><string>production</string>`.

## 2. Tell fpack

```yaml
macos:
  sign:
    identity: "Developer ID Application: Your Name (ABCDE12345)"
    provisioning_profile: macos/MyApp.provisionprofile      # or FPACK_MACOS_PROVISIONING_PROFILE / --provisioning-profile
    provisioning_profiles:                                  # extensions and login items: bundle id → profile
      com.example.app.share: macos/MyAppShare.provisionprofile
  smoke_test: true                                          # or FPACK_MACOS_SMOKE_TEST / --smoke-test
```

Before signing, fpack copies the profile to `Contents/embedded.provisionprofile` (and the mapped profiles into app extensions and login items), then signs inside-out: nested code, nested bundles with their own entitlements, then the app. Repeat the profile creation for every extension or login item that has its own bundle id.

## 3. What fpack checks

Every profile is checked with `security cms -D -i` before signing; a problem stops the build with a message instead of producing an app that will not launch. `fpack doctor` reports the same findings.

- the profile is a **Developer ID** profile (`ProvisionsAllDevices`);
- it belongs to the same team as the signing certificate and was generated for *that* certificate;
- the app id matches (`TEAM.bundle.id`, wildcards allowed);
- it has not expired;
- every restricted entitlement used for signing is authorized with a matching value (for example `aps-environment` must be `production`).

Without a profile, restricted entitlements or an embedded development profile stop the build before signing. To sign anyway, set `macos.sign.profile_check: false` (`FPACK_MACOS_PROFILE_CHECK=false`); the profile is still embedded.

| Message | Meaning and fix |
| --- | --- |
| `… uses restricted entitlements (…) that a Developer ID app may only use with a provisioning profile` | no profile configured: set `macos.sign.provisioning_profile` |
| `the app embeds Xcode's development profile "…"` | Xcode embedded its development profile: configure a Developer ID profile so fpack replaces it |
| `… is a development/ad hoc profile (N registered devices)` | you downloaded a Development profile: create a **Developer ID** one |
| `… is not a Developer ID profile (ProvisionsAllDevices is missing)` | an App Store profile: create a **Developer ID** one |
| `… was generated for another Developer ID Application certificate (…), not the signing certificate …` | the profile lists other certificates than the one fpack signs with (for example after a certificate was recreated); macOS logs `Unsatisfied entitlements` and refuses to launch: edit the profile, select the signing certificate, download it again |
| `… belongs to team X, but the signing certificate is from team Y` | profile and certificate are from different teams |
| `… is for app id TEAM.a, not TEAM.b` | the profile was made for another bundle id |
| `… expired on YYYY-MM-DD` | generate and download the profile again |
| `entitlement … is not authorized by …` | enable the capability for the app id, regenerate the profile |
| `aps-environment is "development" in the entitlements but the profile allows "production"` | set `production` in the release entitlements |

## 4. Smoke test: does the signed app start?

Enable it with `macos.smoke_test: true`, `FPACK_MACOS_SMOKE_TEST=true` or `--smoke-test`:

```bash
fpack build macos dmg --smoke-test
```

fpack runs `open -n` on the signed app, waits `macos.smoke_test_seconds` (default 5, from 1 to 300), checks with `pgrep` that the app is still running, and quits it with `pkill`. If it is not running, the build fails with *the app was not running … after launch (it crashed or macOS refused to start it)* and prints a `log show` command that shows why (typically `amfid` or RunningBoard messages such as `Unsatisfied entitlements`, `No matching profile found`). It runs **before notarization**, so a broken app is never uploaded.

## Inspect profiles and apps yourself

```bash
security cms -D -i macos/MyApp.provisionprofile                          # the profile's contents
codesign -d --entitlements - --xml MyApp.app                              # entitlements of a built app
security cms -D -i MyApp.app/Contents/embedded.provisionprofile           # the embedded profile
```

## Next

Back to the [macOS tutorial](/platforms/macos), or [troubleshooting](/troubleshooting#macos).
