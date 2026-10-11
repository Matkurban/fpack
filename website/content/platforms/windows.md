---
title: "Windows"
description: "Portable zip, Inno Setup installer, MSIX and code signing."
---

Code signing is optional. Choose one certificate source in `windows.sign`:

- a `.pfx` file: `certificate` + `password` (`FPACK_WINDOWS_CERTIFICATE` / `FPACK_WINDOWS_CERTIFICATE_PASSWORD`; the password is never printed),
- the certificate store: `thumbprint` (`/sha1`) or `subject` (`/n`), with `store` (`/s`, default `My`) and `machine_store: true` (`/sm`),
- a signtool plugin for cloud signing, e.g. Azure Trusted Signing: `dlib` + `dlib_metadata` (`/dlib` / `/dmdf`).

Right after `flutter build windows`, fpack signs `files` of the Release folder (default `*.exe` and `*.dll`), so the portable zip, the installer and the MSIX all contain signed binaries. The Inno Setup installer and uninstaller are signed through `SignTool=`; the MSIX is signed by `msix:create` with the same certificate (with `dlib`, fpack signs the finished MSIX). Every signature uses SHA-256 and an RFC 3161 timestamp (`timestamp_url`, default `http://timestamp.digicert.com`), with `description` (`/d`) and `url` (`/du`); afterwards `signtool verify /pa` checks it (`verify: false` turns that off; a failure, e.g. a self-signed test certificate, is only a warning). signtool is found on PATH or in the newest Windows SDK (or set `signtool`); `fpack doctor` reports it and how to install the SDK (`winget install Microsoft.WindowsSDK.10.0.26100`, needs admin; without admin rights, extract signtool from the `Microsoft.Windows.SDK.BuildTools` NuGet package and set `signtool`).

```yaml
windows:
  sign:
    certificate: ${WINDOWS_PFX}            # or thumbprint: / subject: / dlib:
    password: ${WINDOWS_PFX_PASSWORD}
    url: https://example.com
```

Inno Setup installers support publisher/URL, copyright, install directory, Start menu group, desktop shortcut, license/info pages, icon and wizard images, minimum Windows version, `privileges` (`user` without admin / `admin` / `ask`) and multiple languages (`languages: [zh-CN, en]`, the first is the default; fpack ships the Chinese language file older Inno Setup versions lack). MSIX display name, publisher, capabilities, file associations, protocols, version etc. can be set directly in `windows.msix` and take precedence over pubspec's `msix_config`.

## Targets

| Target | Artifact | Needs |
| --- | --- | --- |
| `windows` | `-windows-x64-portable.zip` | Visual Studio with the "Desktop development with C++" workload |
| `exe` | `-windows-x64-setup.exe` | [Inno Setup 6](https://jrsoftware.org/isinfo.php) (`ISCC.exe`; found in the default install folders or via `windows.inno_setup.iscc`) |
| `msix` | `-windows-x64.msix` | `dart pub add --dev msix` |

All three share one `flutter build windows`.

## Notes

- **Project path**: keep the project in a path with only ASCII characters. `flutter build windows` fails in paths with e.g. Chinese characters (`Unable to read file … app.dill`); `fpack doctor` warns about it. Alternatively enable "Beta: Use Unicode UTF-8 for worldwide language support" in the Windows region settings.
- **MSIX description**: the `msix` package reads the description only from `pubspec.yaml` (`msix_config.description`, else `description`); when `windows.msix.description` / `app.description` differ, fpack prints the entry to add.
- **Hooks** run through `cmd /S /C`, so use `%VAR%` for hook variables (`%FPACK_OUTPUT_DIR%`); quoted paths work as typed.

## Installer example

```yaml
windows:
  inno_setup:
    languages: [en, zh-CN]      # the first is the default
    privileges: user            # user | admin | ask
    desktop_icon: checked       # checked | unchecked | none
    license_file: LICENSE.txt
  msix:
    publisher: "CN=Your Company"
    capabilities: [internetClient]
  sign:
    certificate: C:/certs/codesign.pfx
    password: ${WINDOWS_CERT_PASSWORD}
```

`windows.inno_setup.app_id` is generated once by `fpack init` – keep it stable so upgrades replace the installed version. To use your own `.iss` script set `windows.inno_setup.script`.
