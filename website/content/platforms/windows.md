---
title: "Windows"
description: "Portable zip, Inno Setup installer, MSIX and code signing."
---

With `windows.sign.certificate` (.pfx) + `password`, or `thumbprint` (certificate store): right after `flutter build windows` the app .exe is signed with signtool (so the zip, installer and MSIX all contain a signed exe), the Inno Setup installer and uninstaller are signed through `SignTool=`, and the MSIX uses the same certificate; the timestamp server defaults to `http://timestamp.digicert.com` and signtool is located in the Windows SDK automatically.

Inno Setup installers support publisher/URL, copyright, install directory, Start menu group, desktop shortcut, license/info pages, icon and wizard images, minimum Windows version, `privileges` (`user` without admin / `admin` / `ask`) and multiple languages (`languages: [zh-CN, en]`, the first is the default; fpack ships the Chinese language file older Inno Setup versions lack). MSIX display name, publisher, capabilities, file associations, protocols, version etc. can be set directly in `windows.msix` and take precedence over pubspec's `msix_config`.

## Targets

| Target | Artifact | Needs |
| --- | --- | --- |
| `windows` | `-windows-x64-portable.zip` | Visual Studio with the "Desktop development with C++" workload |
| `exe` | `-windows-x64-setup.exe` | [Inno Setup 6](https://jrsoftware.org/isinfo.php) (`ISCC.exe`; found in the default install folders or via `windows.inno_setup.iscc`) |
| `msix` | `-windows-x64.msix` | `dart pub add --dev msix` |

All three share one `flutter build windows`.

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
