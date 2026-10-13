---
title: "macOS"
description: "Developer ID signing, notarization, DMG and pkg installers, and how to get certificates and credentials."
---

## Signing and notarization

For distribution outside the App Store (Developer ID). Enabled **only** through fpack's own settings (low → high precedence):

1. `macos.sign` in `fpack.yaml` (`fpack init` writes a commented section with placeholders and lists the Developer ID certificates found in your keychain)
2. environment variables `FPACK_MACOS_SIGN`, `FPACK_MACOS_SIGN_IDENTITY`, `FPACK_MACOS_INSTALLER_IDENTITY`, `FPACK_MACOS_NOTARIZE`, `FPACK_MACOS_NOTARY_PROFILE`, `FPACK_MACOS_PROVISIONING_PROFILE`, `FPACK_MACOS_PROFILE_CHECK`, `FPACK_MACOS_SMOKE_TEST`
3. flags `--sign/--no-sign`, `--sign-identity`, `--installer-identity`, `--notarize/--no-notarize`, `--notary-profile`, `--provisioning-profile`, `--smoke-test`

```yaml
macos:
  sign:
    identity: "Developer ID Application: Your Name (TEAMID)"   # turns signing on
    notary_profile: XueHua                                     # turns notarization on
```

One notary credential (highest priority first): keychain profile `notary_profile` (recommended locally) → App Store Connect API key `notary_api_key` + `notary_api_key_id` (+ `notary_api_issuer`, recommended for CI) → Apple ID `notary_apple_id` + `notary_team_id` + `notary_password` (app-specific password, redacted in logs). Signing uses the Hardened Runtime by default (`hardened_runtime`, required for notarization).

Rules: an identity turns signing on; any notary credential turns notarization on; turning signing off also turns notarization off; `--sign` without an identity picks the first "Developer ID Application" certificate in the keychain; `--notarize` (or `notarize: true`) without any credential uses the keychain profile `NotaryProfile`; a missing identity lists the available ones. **With nothing configured** the .app keeps the Xcode project's own signature, the DMG is unsigned, and the artifact notes explain how to configure signing. fpack does not read other tools' settings in `pubspec.yaml` (such as the `dmg:` section of the [`dmg`](https://pub.dev/packages/dmg) package).

DMG flow: copy the .app → `codesign --deep` the nested code → re-sign the app with the Hardened Runtime + `macos/Runner/Release.entitlements` → verify → `hdiutil` (or `create-dmg`, with background image, window size/position, icon size and positions, volume icon, license; `format`/`filesystem` work with both tools) builds the DMG → sign the DMG → `xcrun notarytool submit` → wait for the result → `stapler staple` → `spctl` assessment.

**Waiting for notarization**: Apple usually needs a few minutes. As soon as the upload has a submission ID, fpack writes `NOTARIZATION.md` (language follows `--lang`) and `notarization.json` into the output directory: artifact path and SHA-256, submission ID, submit time, the credential used (keychain profile name / API key ID / Apple ID — never a password) and copy-paste commands (`xcrun notarytool info/wait/log …`, `xcrun stapler staple/validate …`, `spctl -a -vv …`, `fpack notarize status/finish`), and prints the file's path. While waiting it shows the elapsed time and the hint: *Notarization usually takes a few minutes. Ctrl-C stops waiting — Apple keeps processing; query it later with the commands in NOTARIZATION.md.* The file is updated with the final state (Accepted / Invalid); on rejection fpack downloads Apple's log (`notary-log-<file>.json`), summarizes the issues and suggests fixes.

- **No waiting**: `macos.notarize.wait: false` (or `FPACK_NOTARIZE_WAIT=false`, `--notarize-no-wait`) submits, writes the record and finishes; the summary and `--json` report the artifact's notarization state as `submitted` (not stapled yet).
- **Continue later**: `fpack notarize status [dir|ID]` asks Apple about all pending submissions; `fpack notarize finish [dir]` waits for the results, staples accepted DMGs/pkgs (an app zip is unpacked, the .app stapled and re-zipped), rewrites `SHA256SUMS` and updates the record; rejected ones get their log downloaded and summarized. Also works after a Ctrl-C.

How to obtain the certificates and notary credentials is described in the next section, [Getting macOS signing certificates and notarization credentials](/platforms/macos#credentials).

`fpack build macos` produces a signed zip; with notarization on, the zip is notarized too (submit the zip → staple the .app → re-zip).

<a id="credentials"></a>

## Getting macOS signing certificates and notarization credentials

Distributing a macOS app outside the App Store needs a **Developer ID Application** certificate (signs the .app / zip / DMG), optionally a **Developer ID Installer** certificate (signs the pkg), and one kind of **notarization credential**. All of this requires a paid [Apple Developer Program](https://developer.apple.com/programs/) membership, and Developer ID certificates can only be created by the team's **Account Holder**.

**1. Create the Developer ID certificates (on a Mac)**

- Xcode → Settings… → Accounts → select the team → Manage Certificates… → **+** → *Developer ID Application* (and *Developer ID Installer* if you build pkgs). Certificate and private key go straight into the login keychain.
- Or on the web: Keychain Access → Certificate Assistant → Request a Certificate From a Certificate Authority… (save the CSR to disk) → [developer.apple.com/account/resources/certificates](https://developer.apple.com/account/resources/certificates/list) → **+** → Developer ID Application / Developer ID Installer → upload the CSR → download the `.cer` and double-click it — on the Mac that created the CSR, which holds the private key.
- Check:

  ```bash
  security find-identity -v -p codesigning   # "Developer ID Application: Your Name (ABCDE12345)"
  security find-identity -v -p basic         # also lists "Developer ID Installer: …"
  ```

- Tell fpack (use the quoted names exactly as printed; the 10 characters in parentheses are your **Team ID**):

  | Purpose | fpack.yaml | Environment | Flag |
  | --- | --- | --- | --- |
  | App signing identity | `macos.sign.identity` | `FPACK_MACOS_SIGN_IDENTITY` | `--sign-identity` |
  | pkg signing identity | `macos.sign.installer_identity` | `FPACK_MACOS_INSTALLER_IDENTITY` | `--installer-identity` |

**2. Notarization credentials (pick one)**

*a) Keychain profile (recommended on your own Mac)*: Apple ID + app-specific password + Team ID, stored once.

1. [account.apple.com](https://account.apple.com) → Sign-In and Security → **App-Specific Passwords** → generate one (looks like `abcd-efgh-ijkl-mnop`).
2. Team ID: [developer.apple.com/account](https://developer.apple.com/account) → Membership details, or the 10 characters in the certificate name.
3. Store it in the keychain (omit `--password` to be prompted):

   ```bash
   xcrun notarytool store-credentials fpack-notary \
     --apple-id you@example.com --team-id ABCDE12345 --password abcd-efgh-ijkl-mnop
   xcrun notarytool history --keychain-profile fpack-notary   # check
   ```

4. fpack: `macos.sign.notary_profile: fpack-notary` (or `FPACK_MACOS_NOTARY_PROFILE`, `--notary-profile`).

Without a stored profile you can pass the Apple ID directly: `notary_apple_id` / `notary_team_id` / `notary_password` (`FPACK_NOTARY_APPLE_ID`, `FPACK_NOTARY_TEAM_ID`, `FPACK_NOTARY_PASSWORD`). Keep the password in an environment variable or CI secret; fpack redacts it in logs, `--dry-run` output and NOTARIZATION.md.

*b) App Store Connect API key (recommended for CI)*: not tied to a person's Apple ID, revocable at any time.

1. [App Store Connect](https://appstoreconnect.apple.com) → Users and Access → Integrations → App Store Connect API → Team Keys → **+**, access Developer (or higher).
2. Download `AuthKey_<KEY_ID>.p8` — **it can be downloaded only once**, keep it safe — and note the **Key ID** and the **Issuer ID** shown on the page (individual keys have no issuer; leave it out).
3. fpack:

   | Value | fpack.yaml | Environment |
   | --- | --- | --- |
   | path of the `.p8` file | `macos.sign.notary_api_key` | `FPACK_NOTARY_API_KEY` |
   | Key ID | `macos.sign.notary_api_key_id` | `FPACK_NOTARY_API_KEY_ID` |
   | Issuer ID | `macos.sign.notary_api_issuer` | `FPACK_NOTARY_API_ISSUER` |

   Check: `xcrun notarytool history --key AuthKey_ABC123DEF4.p8 --key-id ABC123DEF4 --issuer <Issuer ID>`.

Priority: `notary_profile` → API key → Apple ID; any of them turns notarization on.

**3. Export the certificates (.p12) for CI**

1. Keychain Access → login keychain → **My Certificates** → select "Developer ID Application: …" (expand it: the private key must be underneath) → File → Export Items… → format *Personal Information Exchange (.p12)* → choose an export password. Repeat for "Developer ID Installer: …" if you build pkgs (or select both and export them into one .p12).
2. Base64-encode it into CI secrets (e.g. `MACOS_CERTS_P12_BASE64`, `MACOS_CERTS_P12_PASSWORD`), the API key likewise (`NOTARY_API_KEY_P8_BASE64`):

   ```bash
   base64 -i DeveloperID.p12 | pbcopy
   base64 -i AuthKey_ABC123DEF4.p8 | pbcopy
   ```

3. Import them into a temporary keychain on the runner (GitHub Actions):

   ```yaml
   - name: import Developer ID certificates
     env:
       P12_BASE64: ${{ secrets.MACOS_CERTS_P12_BASE64 }}
       P12_PASSWORD: ${{ secrets.MACOS_CERTS_P12_PASSWORD }}
       KEYCHAIN_PASSWORD: ${{ secrets.KEYCHAIN_PASSWORD }}   # any random string
       API_KEY_BASE64: ${{ secrets.NOTARY_API_KEY_P8_BASE64 }}
     run: |
       KEYCHAIN="$RUNNER_TEMP/signing.keychain-db"
       echo "$P12_BASE64" | base64 --decode > "$RUNNER_TEMP/certs.p12"
       security create-keychain -p "$KEYCHAIN_PASSWORD" "$KEYCHAIN"
       security set-keychain-settings -lut 21600 "$KEYCHAIN"
       security unlock-keychain -p "$KEYCHAIN_PASSWORD" "$KEYCHAIN"
       security import "$RUNNER_TEMP/certs.p12" -k "$KEYCHAIN" -P "$P12_PASSWORD" \
         -T /usr/bin/codesign -T /usr/bin/productbuild -T /usr/bin/pkgbuild
       security set-key-partition-list -S apple-tool:,apple: -s -k "$KEYCHAIN_PASSWORD" "$KEYCHAIN"
       security list-keychains -d user -s "$KEYCHAIN" $(security list-keychains -d user | tr -d '"')
       security find-identity -v -p codesigning "$KEYCHAIN"
       echo "$API_KEY_BASE64" | base64 --decode > "$RUNNER_TEMP/AuthKey.p8"
       rm "$RUNNER_TEMP/certs.p12"
   - name: fpack build macos dmg pkg
     env:
       FPACK_MACOS_SIGN_IDENTITY: "Developer ID Application: Your Name (ABCDE12345)"
       FPACK_MACOS_INSTALLER_IDENTITY: "Developer ID Installer: Your Name (ABCDE12345)"
       FPACK_NOTARY_API_KEY: ${{ runner.temp }}/AuthKey.p8
       FPACK_NOTARY_API_KEY_ID: ${{ secrets.NOTARY_API_KEY_ID }}
       FPACK_NOTARY_API_ISSUER: ${{ secrets.NOTARY_API_ISSUER }}
     run: fpack build macos dmg pkg
   - name: clean up keychain
     if: always()
     run: security delete-keychain "$RUNNER_TEMP/signing.keychain-db" || true
   ```

   `set-key-partition-list` lets codesign / productbuild use the private key without a prompt; `-lut 21600` keeps the keychain unlocked for 6 hours. Never commit certificates, .p12/.p8 files or passwords.

## Provisioning profiles

Provisioning profiles and restricted entitlements: `flutter build macos` embeds Xcode's development profile ("Mac Team Provisioning Profile"). Once the app is re-signed with Developer ID, restricted entitlements such as `com.apple.developer.aps-environment` (push), iCloud, associated domains or keychain access groups are no longer authorized, and the app does not launch (`RBSRequestErrorDomain Code=5`). Create a **Developer ID** provisioning profile for the app id (developer.apple.com → Profiles → + → Developer ID) and set it:

```yaml
macos:
  sign:
    identity: "Developer ID Application: Your Name (ABCDE12345)"
    provisioning_profile: macos/MyApp.provisionprofile         # or FPACK_MACOS_PROVISIONING_PROFILE / --provisioning-profile
    provisioning_profiles:                                     # nested bundles: bundle id → profile
      com.example.app.share: macos/MyAppShare.provisionprofile
  smoke_test: true                                             # or FPACK_MACOS_SMOKE_TEST / --smoke-test
```

Before signing, fpack copies the profile to `Contents/embedded.provisionprofile` (and the mapped profiles into extensions and login items), then signs inside-out: nested code, nested bundles with their own entitlements, the app. It checks every profile with `security cms -D -i`: Developer ID type (`ProvisionsAllDevices`), the signing certificate's team, the app id (`TEAM.bundle.id`, wildcards allowed), the expiry date, and that every restricted entitlement used for signing is authorized with a matching value (e.g. `aps-environment` must be `production`). Without a profile, restricted entitlements or an embedded development profile stop the build before signing; `fpack doctor` reports the same. `macos.sign.profile_check: false` (`FPACK_MACOS_PROFILE_CHECK=false`) skips the checks. `macos.smoke_test` opens the signed app, checks that it is still running after `smoke_test_seconds` (default 5) and quits it.

**Getting a Developer ID provisioning profile** (Account Holder or Admin):

1. developer.apple.com → Certificates, IDs & Profiles → **Identifiers**: open the app id (e.g. `com.example.app`, the `PRODUCT_BUNDLE_IDENTIFIER` in `macos/Runner/Configs/AppInfo.xcconfig`) and enable the capabilities your entitlements use (Push Notifications, iCloud, Associated Domains, …). Save.
2. **Profiles** → **+** → under *Distribution* choose **Developer ID** → platform macOS → the app id → the **Developer ID Application** certificate you sign with (the exact certificate: check its SHA-1 with `security find-identity -v -p codesigning`; a profile made for another certificate is rejected at launch) → name it → **Generate** → **Download**.
3. Put the `.provisionprofile` in the project (e.g. `macos/MyApp.provisionprofile`; it holds no secret, but you may keep it out of git and point to it with `FPACK_MACOS_PROVISIONING_PROFILE` in CI). Repeat for every extension/login item with its own bundle id and map them under `provisioning_profiles`.
4. Make the release entitlements match the profile: Developer ID profiles authorize `aps-environment` **production**, so set `<key>com.apple.developer.aps-environment</key><string>production</string>` in `macos/Runner/Release.entitlements`.
5. Inspect a profile with `security cms -D -i macos/MyApp.provisionprofile`; check a built app with `codesign -d --entitlements - --xml MyApp.app` and `security cms -D -i MyApp.app/Contents/embedded.provisionprofile`.

**Validation messages** (all stop the build before signing; `fpack doctor` shows them too):

| Message | Meaning / fix |
| --- | --- |
| `… uses restricted entitlements (…) that a Developer ID app may only use with a provisioning profile` | no profile configured: set `macos.sign.provisioning_profile` |
| `the app embeds Xcode's development profile "…"` / `… embeds Xcode's development profile` (doctor) | Xcode embedded the Mac Team (development) profile: configure a Developer ID profile so fpack replaces it |
| `… is a development/ad hoc profile (N registered devices)` | you downloaded a Development profile: create a **Developer ID** one |
| `… is not a Developer ID profile (ProvisionsAllDevices is missing)` | App Store profile: create a **Developer ID** one |
| `… was generated for another Developer ID Application certificate (…), not the signing certificate …` | the profile lists other certificates than the one fpack signs with (e.g. after a certificate was recreated); macOS logs `Unsatisfied entitlements` and refuses to launch: edit the profile, select the signing certificate, download it again |
| `… belongs to team X, but the signing certificate is from team Y` | profile and certificate are from different teams |
| `… is for app id TEAM.a, not TEAM.b` | the profile was made for another bundle id |
| `… expired on YYYY-MM-DD` | generate and download the profile again |
| `entitlement … is not authorized by …` | enable the capability for the app id, regenerate the profile |
| `aps-environment is "development" in the entitlements but the profile allows "production"` | set `production` in the release entitlements |

If you really need to sign anyway, `macos.sign.profile_check: false` (or `FPACK_MACOS_PROFILE_CHECK=false`) skips the checks (the profile is still embedded).

**Smoke test**: with `macos.smoke_test: true` (or `--smoke-test`), fpack runs `open -n` on the signed app, waits `macos.smoke_test_seconds` (default 5, 1-300), checks with `pgrep` that the app is still running and quits it with `pkill`. If it is not running, the build fails with `the app was not running … after launch (it crashed or macOS refused to start it)` and a `log show` command that shows why (amfid / RunningBoard). It runs before notarization, so a broken app is never uploaded.

## macOS installer (pkg)

`fpack build pkg` uses `pkgbuild` to create a component package that installs the app into `/Applications` (not relocatable, so upgrades always replace the copy in /Applications), then `productbuild` to create a distribution package (Apple silicon and Intel, no Rosetta prompt). `macos.pkg` sets `identifier`, `install_location`, `title` and the installer's `welcome` / `readme` / `license` / `conclusion` pages (.html/.rtf/.txt) and `background` (image).

- **Signing**: a pkg needs a separate **"Developer ID Installer"** certificate (not the "Developer ID Application" one that signs the app), set with `macos.sign.installer_identity` / `FPACK_MACOS_INSTALLER_IDENTITY` / `--installer-identity`; the signed pkg is checked with `pkgutil --check-signature`. Without it an unsigned pkg is produced with a note; `--no-sign` turns pkg signing off as well.
- **App signing**: with `macos.sign.identity` set, the app inside the pkg is re-signed with Developer ID just like for the zip/DMG.
- **Notarization**: shares the DMG settings; only signed pkgs are submitted and stapled, then assessed with `spctl --assess --type install`.
- **More options**: `min_os` (refuse to install below this version; default is the project's `MACOSX_DEPLOYMENT_TARGET`), `require_restart`, `relocatable`, `preinstall` / `postinstall` scripts (made executable automatically), `version`.
- `doctor` checks pkgbuild/productbuild and lists the installer certificates in the keychain.
