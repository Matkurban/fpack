---
title: "Windows"
description: "Build a portable zip, an Inno Setup installer or an MSIX package, and sign them."
---

This tutorial builds Windows apps: a portable zip (`windows`), an installer (`exe`, with Inno Setup) and an `.msix` package (`msix`), with optional code signing. It builds on Windows only; the three targets share one `flutter build windows`.

## What you need

- **Visual Studio** with the *Desktop development with C++* workload.
- For `exe`: [Inno Setup 6](https://jrsoftware.org/isinfo.php) (`winget install --id JRSoftware.InnoSetup -e`). fpack finds `ISCC.exe` in the default install folders, or at `windows.inno_setup.iscc`.
- For `msix`: the `msix` package as a dev dependency: `dart pub add --dev msix`.
- For signing: `signtool` (Windows SDK) and a code signing certificate.
- A project path with **ASCII characters only**. `flutter build windows` fails in paths with, for example, Chinese characters (`Unable to read file … app.dill`); doctor warns about it. Alternatively enable "Beta: Use Unicode UTF-8 for worldwide language support" in the Windows region settings.

## 1. Check the machine

```powershell
fpack doctor windows exe msix
```

## 2. Build the portable zip

```powershell
fpack build windows
```

The result is `dist\1.0.0+1\my_app-1.0.0+1-windows-x64-portable.zip`: the Release folder zipped. Users unzip it and run the `.exe`.

## 3. Build the installer

```powershell
fpack build exe
```

fpack generates an Inno Setup script from your project metadata and compiles it to `my_app-1.0.0+1-windows-x64-setup.exe`. `fpack init` generates `windows.inno_setup.app_id` once: **keep it stable** so upgrades replace the installed version.

```yaml
windows:
  inno_setup:
    languages: [en, zh-CN]      # the first is the default; fpack ships the Chinese file older Inno versions lack
    privileges: user            # user (no admin) | admin | ask
    desktop_icon: checked       # checked | unchecked | none
    license_file: LICENSE.txt
```

Inno Setup also supports publisher and URLs, copyright, install directory, Start menu group, info pages, setup icon, wizard images, minimum Windows version and compression (see the [reference](/configuration#windowsinno_setup)). To use your own script, set `windows.inno_setup.script`. Company and product details come from `app.*` ([Names, versions and output](/guides/project-and-output)).

## 4. Build the MSIX

```powershell
fpack build msix
```

fpack runs `dart run msix:create` with settings from `windows.msix`, which take precedence over `msix_config` in `pubspec.yaml`:

```yaml
windows:
  msix:
    publisher: "CN=Your Company"
    capabilities: [internetClient]
    protocol_activation: [myapp]
```

Display name, publisher display name, identity name, version, logo, languages, file extensions, execution alias, start at login, minimum OS version and Store mode are all available.

::: note
The `msix` package reads the description only from `pubspec.yaml` (`msix_config.description`, else `description`). When `windows.msix.description` or `app.description` differ, fpack prints the entry to add.
:::

## 5. Sign the code

Code signing is optional but avoids SmartScreen warnings. Choose **one** certificate source in `windows.sign`:

| Source | Keys | Variables |
| --- | --- | --- |
| a `.pfx` file | `certificate` + `password` | `FPACK_WINDOWS_CERTIFICATE`, `FPACK_WINDOWS_CERTIFICATE_PASSWORD` |
| the certificate store | `thumbprint` (`/sha1`) or `subject` (`/n`), with `store` (default `My`) and `machine_store: true` | `FPACK_WINDOWS_CERT_THUMBPRINT`, `FPACK_WINDOWS_CERT_SUBJECT` |
| a signtool plugin (cloud signing, e.g. Azure Trusted Signing) | `dlib` + `dlib_metadata` | `FPACK_WINDOWS_SIGN_DLIB`, `FPACK_WINDOWS_SIGN_DLIB_METADATA` |

```yaml
windows:
  sign:
    certificate: ${WINDOWS_PFX}            # or thumbprint: / subject: / dlib:
    password: ${WINDOWS_PFX_PASSWORD}
    url: https://example.com
```

Right after `flutter build windows`, fpack signs `files` of the Release folder (default `*.exe` and `*.dll`), so the portable zip, the installer and the MSIX all contain signed binaries. The Inno Setup installer and uninstaller are signed through `SignTool=`; the MSIX is signed by `msix:create` with the same certificate (with `dlib`, fpack signs the finished MSIX). Every signature uses SHA-256 and an RFC 3161 timestamp (`timestamp_url`, default `http://timestamp.digicert.com`), with `description` (`/d`) and `url` (`/du`). Afterwards `signtool verify /pa` checks it; a failure (for example a self-signed test certificate) is only a warning, and `verify: false` turns the check off.

signtool is found on `PATH` or in the newest Windows SDK, or set `signtool`. `fpack doctor` reports it and how to install it: `winget install Microsoft.WindowsSDK.10.0.26100` (needs admin; without admin rights, extract signtool from the `Microsoft.Windows.SDK.BuildTools` NuGet package and set `signtool`).

::: tip
Keep the certificate password in a variable (`${WINDOWS_PFX_PASSWORD}`) or CI secret. fpack never prints it.
:::

## Hooks and quoting

[Hooks](/guides/hooks) run through `cmd /S /C` on Windows, so use `%VAR%` for hook variables (`%FPACK_OUTPUT_DIR%`); quoted paths work as typed.

## In CI

Use `windows-latest`; see [CI recipes](/ci#windows).

## If something goes wrong

| Message | Fix |
| --- | --- |
| `Inno Setup (ISCC.exe) not found` | `winget install --id JRSoftware.InnoSetup -e`, or set `windows.inno_setup.iscc` |
| `Unable to read file … app.dill` | move the project to an ASCII-only path |
| msix target fails with a missing package | `dart pub add --dev msix` |
| signtool not found | install the Windows SDK or set `windows.sign.signtool` |

More on the [troubleshooting page](/troubleshooting#windows).
