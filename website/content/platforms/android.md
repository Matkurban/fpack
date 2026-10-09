---
title: "Android"
description: "Signing, flavors, APK splits and app bundles."
---

fpack **does not edit Gradle files**. It signs release builds through the Android Gradle plugin's standard `android.injected.signing.*` properties (passed as `ORG_GRADLE_PROJECT_*` environment variables); passwords never appear on the command line, in logs or in `--dry-run` output (shown as `***`).

```bash
export FPACK_ANDROID_KEYSTORE=~/keys/upload-keystore.jks
export FPACK_ANDROID_KEYSTORE_PASSWORD='…'
export FPACK_ANDROID_KEY_ALIAS=upload
export FPACK_ANDROID_KEY_PASSWORD='…'       # defaults to the keystore password
fpack build apk aab
```

- In CI, put the base64 of the keystore into `FPACK_ANDROID_KEYSTORE_BASE64`; fpack writes it to `build/fpack/secrets/` (mode 0600) and deletes it after the build.
- Before building, `keytool` checks the password and alias and tells you which one is wrong.
- After building, `apksigner` / `keytool -printcert` verify the artifacts and show the signer; a remaining Android Debug certificate triggers a warning.
- If the project already has its own `key.properties` + `signingConfigs`, just don't configure fpack signing and the project's setup is used.
- Without signing configured and with release using the debug key (the `flutter create` default) fpack notes: fine for testing, Google Play will reject it.

## Creating an upload keystore

```bash
keytool -genkey -v -keystore ~/keys/upload.jks -keyalg RSA -keysize 2048 \
  -validity 10000 -alias upload
base64 -i ~/keys/upload.jks | pbcopy     # for the FPACK_ANDROID_KEYSTORE_BASE64 CI secret
```

Keep the keystore and its passwords out of git. Google Play App Signing re-signs your app with the app signing key; the keystore above is the *upload* key.

## Flavors

`--flavor NAME` (or `build.flavor`, `FPACK_FLAVOR`) selects an Android productFlavor. fpack reads the flavors from `android/app/build.gradle(.kts)`: an unknown name is an error with a suggestion, and when the project defines flavors but none is selected fpack stops before a full Gradle build fails. Artifacts get the flavor in their name (`{-flavor}`).

A flavor that exists only on Android is not passed to iOS/macOS builds when the Xcode projects define no custom schemes.

## APK splits and ABIs

```bash
fpack build apk --split-per-abi           # one APK per ABI
fpack build apk --split-per-abi=both      # per-ABI APKs + the universal APK
fpack build apk --abis arm64-v8a,x86_64   # limit the ABIs
```

## App bundles

`fpack build aab` produces `-android.aab`; `keytool -printcert -jarfile` verifies its signer after the build.
