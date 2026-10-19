---
title: "Secrets and variables"
description: "Which setting wins, how ${VAR} works, the FPACK_* variables, and how to keep passwords out of files and logs."
---

Signing needs passwords and keys. fpack is designed so that they never have to be written to `fpack.yaml`, the command line or the logs.

## Which setting wins

For every setting, the first source that provides a value wins:

1. command-line flag (`--flavor prod`)
2. `FPACK_*` environment variable (`FPACK_FLAVOR=prod`)
3. `fpack.yaml` (`build.flavor: prod`)
4. default

Each key in the [reference](/configuration) lists its variable and flag; the [environment variables page](/environment) lists them all.

## `${VAR}` in `fpack.yaml`

Any value in `fpack.yaml` can read an environment variable when the file is loaded:

```yaml
android:
  signing:
    store_file: ~/keys/upload.jks
    store_password: ${KEYSTORE_PASSWORD}          # an error if it is not set
    key_alias: ${KEY_ALIAS:-upload}               # a default when it is not set
output:
  dir: ${DIST_DIR:-dist}
```

Use this for anything secret or machine-specific. Commit the file, keep the values in your shell, a password manager or CI secrets.

::: note
`${VAR}` is expanded **when the configuration is loaded**. In [hooks](/guides/hooks) write `$VAR` (Windows `%VAR%`) for variables that only exist while the hook runs.
:::

## Dedicated variables

Signing and notarization settings have their own variables, so a CI job needs no `fpack.yaml` edits:

| Area | Variables |
| --- | --- |
| Android | `FPACK_ANDROID_KEYSTORE`, `FPACK_ANDROID_KEYSTORE_BASE64`, `FPACK_ANDROID_KEYSTORE_PASSWORD`, `FPACK_ANDROID_KEY_ALIAS`, `FPACK_ANDROID_KEY_PASSWORD` |
| iOS | `FPACK_IOS_EXPORT_METHOD`, `FPACK_IOS_EXPORT_OPTIONS_PLIST`, `FPACK_IOS_CODESIGN` |
| macOS signing | `FPACK_MACOS_SIGN`, `FPACK_MACOS_SIGN_IDENTITY`, `FPACK_MACOS_INSTALLER_IDENTITY`, `FPACK_MACOS_PROVISIONING_PROFILE`, `FPACK_MACOS_PROFILE_CHECK`, `FPACK_MACOS_SMOKE_TEST` |
| macOS notarization | `FPACK_MACOS_NOTARIZE`, `FPACK_MACOS_NOTARY_PROFILE`, `FPACK_NOTARIZE_WAIT`, `FPACK_NOTARY_APPLE_ID`, `FPACK_NOTARY_TEAM_ID`, `FPACK_NOTARY_PASSWORD`, `FPACK_NOTARY_API_KEY`, `FPACK_NOTARY_API_KEY_ID`, `FPACK_NOTARY_API_ISSUER` |
| Windows | `FPACK_WINDOWS_CERTIFICATE`, `FPACK_WINDOWS_CERTIFICATE_PASSWORD`, `FPACK_WINDOWS_CERT_THUMBPRINT`, `FPACK_WINDOWS_CERT_SUBJECT`, `FPACK_WINDOWS_SIGN_DLIB`, `FPACK_WINDOWS_SIGN_DLIB_METADATA` |

## What fpack protects

- Passwords are redacted in logs, in `--dry-run` output (shown as `***`), in `NOTARIZATION.md` and in `--json`.
- Passwords are never put on a command line; they travel in environment variables or temporary files.
- Key material fpack writes (for example a keystore decoded from `FPACK_ANDROID_KEYSTORE_BASE64`) goes to `build/fpack/secrets/` with mode 0600 and is **deleted after the build**.
- It checks credentials *before* the long build: a wrong keystore password or alias, a missing signing identity.

## In CI

1. Store secrets in your CI's secret store.
2. Map them to the `FPACK_*` variables in the job's `env`.
3. For files (keystore, `.p12`, API key `.p8`), store them as base64 and decode on the runner.

```yaml
- run: fpack build apk aab
  env:
    FPACK_ANDROID_KEYSTORE_BASE64: ${{ secrets.KEYSTORE_BASE64 }}
    FPACK_ANDROID_KEYSTORE_PASSWORD: ${{ secrets.KEYSTORE_PASSWORD }}
    FPACK_ANDROID_KEY_ALIAS: upload
```

Complete workflows per platform: [CI recipes](/ci). Platform specifics: [Android](/platforms/android#in-ci), [macOS](/platforms/macos-credentials#ci), [Windows](/platforms/windows#5-sign-the-code).

::: danger
Never commit keystores, `.p12` / `.p8` files, certificates or passwords. If one leaks, treat it as compromised: change the passwords, revoke the API key, or create a new certificate.
:::

## Next

[Hooks](/guides/hooks).
