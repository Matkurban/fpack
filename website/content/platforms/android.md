---
title: "Android"
description: "Build a signed APK and AAB: keystore, signing, ABI splits, verification and Google Play."
---

This tutorial takes an Android app from "builds with the debug key" to a signed `.aab` for Google Play and signed `.apk` files for direct distribution. Targets: `apk`, `aab`. They build on macOS, Windows and Linux.

## What you need

- Flutter, the Android SDK (with accepted licenses) and **Java 17 or newer**. Android Studio bundles all of it.
- An Android project created by Flutter (`android/` folder). fpack **never edits your Gradle files**.

## 1. Check the machine

```bash
fpack doctor apk aab
```

Expect `✓ Android SDK`, `✓ Android SDK licenses accepted` and a Java entry. If Java is reported missing, install Android Studio or JDK 17/21 and run `flutter config --jdk-dir <path>`. Without a signing setup, doctor also warns: *release builds are signed with the DEBUG key … Google Play will reject it*. That is what step 3 fixes.

## 2. Build unsigned (debug-signed) first

```bash
fpack build apk
```

The result is `dist/1.0.0+1/my_app-1.0.0+1-android-universal.apk`. It installs on devices and emulators, but it is signed with the Android **debug** key, so Google Play will not accept it. This step proves the toolchain works before you add signing.

## 3. Create an upload keystore

A keystore holds the key that signs your app. Create one once and keep it safe:

```bash
keytool -genkey -v -keystore ~/keys/upload.jks -keyalg RSA -keysize 2048 \
  -validity 10000 -alias upload
```

`keytool` asks for a keystore password, a key password and your details.

::: warning
Back up the keystore file and both passwords somewhere private (a password manager, an encrypted drive). Keep them **out of git**. With Google Play App Signing, the keystore you create here is the *upload* key: Google re-signs the app with its own key for users, and can reset a lost upload key. Apps distributed outside Google Play cannot recover a lost key.
:::

## 4. Tell fpack about it

fpack signs release builds through the Android Gradle plugin's standard `android.injected.signing.*` properties, passed as `ORG_GRADLE_PROJECT_*` environment variables. Passwords never appear on the command line, in logs or in `--dry-run` output.

The simplest way is environment variables:

::: tabs
@@ macOS / Linux
```bash
export FPACK_ANDROID_KEYSTORE=~/keys/upload.jks
export FPACK_ANDROID_KEYSTORE_PASSWORD='…'
export FPACK_ANDROID_KEY_ALIAS=upload
export FPACK_ANDROID_KEY_PASSWORD='…'       # defaults to the keystore password
```
@@ Windows (PowerShell)
```powershell
$env:FPACK_ANDROID_KEYSTORE = "$HOME\keys\upload.jks"
$env:FPACK_ANDROID_KEYSTORE_PASSWORD = '…'
$env:FPACK_ANDROID_KEY_ALIAS = 'upload'
$env:FPACK_ANDROID_KEY_PASSWORD = '…'       # defaults to the keystore password
```
:::

Or put the non-secret parts in `fpack.yaml` and keep the passwords in variables:

```yaml
android:
  signing:
    store_file: ~/keys/upload.jks
    store_password: ${FPACK_ANDROID_KEYSTORE_PASSWORD}
    key_alias: upload
    key_password: ${FPACK_ANDROID_KEY_PASSWORD}
```

If the project already has its own `android/key.properties` with `signingConfigs`, do not configure fpack signing: the project's setup is used.

## 5. Build the release files

```bash
fpack build apk aab
```

Before Gradle starts, fpack checks the keystore with `keytool`: a wrong password or alias fails immediately and names which one is wrong. After the build it verifies the artifacts (`apksigner`, `keytool -printcert`) and prints the signer; a remaining *Android Debug* certificate is a warning.

```text
dist/1.0.0+1/
  my_app-1.0.0+1-android-universal.apk
  my_app-1.0.0+1-android.aab
  SHA256SUMS
```

## 6. Verify it yourself (optional)

```bash
keytool -printcert -jarfile dist/1.0.0+1/my_app-1.0.0+1-android.aab
apksigner verify --verbose --print-certs dist/1.0.0+1/my_app-1.0.0+1-android-universal.apk
```

`apksigner` lives in `$ANDROID_HOME/build-tools/<version>/`.

## 7. Upload to Google Play

Upload `my_app-…-android.aab` in the Play Console (Release → Production or Testing → Create release). Play builds optimized APKs per device from the bundle. Each upload needs a higher build number: increase the number after `+` in `pubspec.yaml`, or pass `--build-number 43`.

## APK variants

### One APK per ABI

```bash
fpack build apk --split-per-abi           # one APK per ABI instead of the universal APK
fpack build apk --split-per-abi=both      # per-ABI APKs and the universal APK
fpack build apk --abis arm64-v8a,x86_64   # limit the ABIs that are built
```

Per-ABI APKs are much smaller; the universal APK runs everywhere. Default ABIs: `armeabi-v7a`, `arm64-v8a`, `x86_64`. In `fpack.yaml`: `android.split_per_abi` (`false`, `true`, `both`) and `android.abis`; the variable is `FPACK_SPLIT_PER_ABI`.

### Signature schemes

By default Gradle's own signing is used. Setting any of `android.signing.v1` … `v4` makes fpack re-sign the APKs with `apksigner`, so you control the schemes:

```yaml
android:
  signing:
    store_file: ~/keys/upload.jks
    store_password: ${FPACK_ANDROID_KEYSTORE_PASSWORD}
    key_alias: upload
    v1: true     # JAR signing, needed below Android 7
    v2: true     # Android 7+
    v3: true     # Android 9+, key rotation
    v4: true     # Android 11+ incremental install; also writes <apk>.idsig
```

### Gradle properties and extra arguments

`android.project_args` passes Gradle project properties (`flutter build -P key=value`) that your `build.gradle` can read with `project.findProperty`, for example to toggle R8 minification. `android.extra_args` appends arguments to `flutter build apk` / `appbundle`:

```yaml
android:
  project_args: { minify: "true" }
  extra_args: [--android-skip-build-dependency-validation]
```

## Flavors

If your project defines `productFlavors`, pass `--flavor prod` (or `build.flavor`); see [Flavors, entry points and defines](/guides/build-variants#flavors). Projects without flavors never need the flag.

## In CI

Put the keystore into a secret as base64 and let fpack write it for the run:

```bash
base64 -i ~/keys/upload.jks | pbcopy      # macOS; Linux: base64 -w0 upload.jks
```

Set `FPACK_ANDROID_KEYSTORE_BASE64`, `FPACK_ANDROID_KEYSTORE_PASSWORD`, `FPACK_ANDROID_KEY_ALIAS` (and `FPACK_ANDROID_KEY_PASSWORD` if it differs). fpack writes the keystore to `build/fpack/secrets/` with mode 0600 and deletes it after the build. A complete workflow is on [CI recipes](/ci#android).

## If something goes wrong

| Message | Fix |
| --- | --- |
| `no Java (JDK 17+) found` | install Android Studio or JDK 17/21, then `flutter config --jdk-dir <path>` |
| `keystore password was incorrect` / `alias … does not exist` | check `FPACK_ANDROID_KEYSTORE_PASSWORD` / `FPACK_ANDROID_KEY_ALIAS` |
| `… defines productFlavors … but no flavor is set` | pass `--flavor` or set `build.flavor` |
| Google Play rejects the upload (debug certificate) | complete step 4; check the signer printed after the build |

More on the [troubleshooting page](/troubleshooting#android).
