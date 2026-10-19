---
title: "macOS: certificates and credentials"
description: "Create Developer ID certificates and notarization credentials, and export them for CI."
---

Distributing a macOS app outside the App Store needs a **Developer ID Application** certificate (signs the `.app`, zip and DMG), optionally a **Developer ID Installer** certificate (signs the pkg), and one kind of **notarization credential**. All of it requires a paid [Apple Developer Program](https://developer.apple.com/programs/) membership, and Developer ID certificates can only be created by the team's **Account Holder**.

The [macOS tutorial](/platforms/macos) shows how to use them.

## 1. Create the Developer ID certificates

Do this on a Mac. Pick one way.

**In Xcode** (easiest): Xcode → Settings… → Accounts → select the team → Manage Certificates… → **+** → *Developer ID Application* (and *Developer ID Installer* if you build pkgs). Certificate and private key go straight into the login keychain.

**On the web**: Keychain Access → Certificate Assistant → Request a Certificate From a Certificate Authority… (save the request to disk) → [developer.apple.com/account/resources/certificates](https://developer.apple.com/account/resources/certificates/list) → **+** → Developer ID Application / Developer ID Installer → upload the request → download the `.cer` and double-click it — on the Mac that created the request, which holds the private key.

Check that they are installed:

```bash
security find-identity -v -p codesigning   # "Developer ID Application: Your Name (ABCDE12345)"
security find-identity -v -p basic         # also lists "Developer ID Installer: …"
```

Tell fpack the names exactly as printed; the ten characters in parentheses are your **Team ID**:

| Purpose | fpack.yaml | Variable | Flag |
| --- | --- | --- | --- |
| App signing identity | `macos.sign.identity` | `FPACK_MACOS_SIGN_IDENTITY` | `--sign-identity` |
| pkg signing identity | `macos.sign.installer_identity` | `FPACK_MACOS_INSTALLER_IDENTITY` | `--installer-identity` |

<a id="notary"></a>

## 2. Create notarization credentials

Pick **one** kind. fpack prefers them in this order: keychain profile, API key, Apple ID. Any of them turns notarization on.

### a) Keychain profile (best on your own Mac)

Apple ID + app-specific password + Team ID, stored once in the keychain.

1. Open [account.apple.com](https://account.apple.com) → Sign-In and Security → **App-Specific Passwords** and generate one (it looks like `abcd-efgh-ijkl-mnop`).
2. Find your Team ID: [developer.apple.com/account](https://developer.apple.com/account) → Membership details, or the ten characters in the certificate name.
3. Store it (omit `--password` to be prompted instead):

   ```bash
   xcrun notarytool store-credentials fpack-notary \
     --apple-id you@example.com --team-id ABCDE12345 --password abcd-efgh-ijkl-mnop
   xcrun notarytool history --keychain-profile fpack-notary   # check
   ```

4. Use it: `macos.sign.notary_profile: fpack-notary` (`FPACK_MACOS_NOTARY_PROFILE`, `--notary-profile`).

Without a stored profile you can pass the Apple ID directly: `notary_apple_id`, `notary_team_id` and `notary_password` (`FPACK_NOTARY_APPLE_ID`, `FPACK_NOTARY_TEAM_ID`, `FPACK_NOTARY_PASSWORD`). Keep the password in an environment variable or CI secret; fpack redacts it in logs, `--dry-run` output and `NOTARIZATION.md`.

### b) App Store Connect API key (best for CI)

Not tied to a person's Apple ID and revocable at any time.

1. [App Store Connect](https://appstoreconnect.apple.com) → Users and Access → Integrations → App Store Connect API → Team Keys → **+**, access *Developer* or higher.
2. Download `AuthKey_<KEY_ID>.p8`. **It can be downloaded only once**, so keep it safe. Note the **Key ID** and the **Issuer ID** shown on the page (individual keys have no issuer; leave it out).
3. Tell fpack:

   | Value | fpack.yaml | Variable |
   | --- | --- | --- |
   | path of the `.p8` file | `macos.sign.notary_api_key` | `FPACK_NOTARY_API_KEY` |
   | Key ID | `macos.sign.notary_api_key_id` | `FPACK_NOTARY_API_KEY_ID` |
   | Issuer ID | `macos.sign.notary_api_issuer` | `FPACK_NOTARY_API_ISSUER` |

   Check: `xcrun notarytool history --key AuthKey_ABC123DEF4.p8 --key-id ABC123DEF4 --issuer <Issuer ID>`.

<a id="ci"></a>

## 3. Export the certificates (.p12) for CI

A CI runner has no keychain with your certificates; you import them from secrets.

1. Keychain Access → login keychain → **My Certificates** → select "Developer ID Application: …" (expand it: the private key must be underneath) → File → Export Items… → format *Personal Information Exchange (.p12)* → choose an export password. Repeat for "Developer ID Installer: …" if you build pkgs (or select both and export one file).
2. Base64-encode it into CI secrets (for example `MACOS_CERTS_P12_BASE64` and `MACOS_CERTS_P12_PASSWORD`); the API key likewise (`NOTARY_API_KEY_P8_BASE64`):

   ```bash
   base64 -i DeveloperID.p12 | pbcopy
   base64 -i AuthKey_ABC123DEF4.p8 | pbcopy
   ```

3. Import them into a temporary keychain on the runner and build (GitHub Actions):

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

   `set-key-partition-list` lets `codesign` and `productbuild` use the private key without a prompt; `-lut 21600` keeps the keychain unlocked for six hours.

::: danger
Never commit certificates, `.p12` / `.p8` files or passwords. Keep them in CI secrets. fpack redacts passwords in all output, but it cannot protect a file you commit.
:::

## Next

Back to the [macOS tutorial](/platforms/macos), or on to [provisioning profiles](/platforms/macos-profiles) if your app uses push, iCloud or other restricted entitlements.
