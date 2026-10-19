---
title: "Troubleshooting"
description: "Find the symptom, get the cause and the fix: installation, environment, Android, iOS, macOS, Windows, Linux."
---

## First: read the failure

When a build fails, fpack prints the key error excerpt (Gradle's "What went wrong", Xcode errors, …), **one concrete fix**, and the path of the full log in `build/fpack/logs/`. Three tools help every time:

| Tool | Use |
| --- | --- |
| `fpack doctor` | checks every prerequisite and prints the fix command for each |
| `fpack build … --dry-run` | shows every command without running it (passwords as `***`) |
| `-v` / `--verbose` | streams the full tool output instead of an excerpt |

Exit codes are explained on [Exit codes and JSON](/exit-codes).

## Installation and startup

| Symptom | Cause and fix |
| --- | --- |
| `command not found: fpack` | `~/.pub-cache/bin` (Windows `%LOCALAPPDATA%\Pub\Cache\bin`) is not on `PATH`: [Installation](/installation#2-put-it-on-your-path) |
| the first run says it falls back to a local build | the core could not be downloaded; check the network, `FPACK_DOWNLOAD_URL`, or use a bundled/offline install ([how the core is found](/installation#launcher)) |
| macOS says the developer cannot be verified | fpack removes the quarantine attribute when it installs the core; if it is still blocked, run `FPACK_REBUILD=1 fpack --version` to build it with local Go |
| a local-path install prints "Resolving dependencies…" on every run | pub's fixed behavior for path-activated packages; install from pub.dev or git, or filter the lines when you consume `--json` |
| `no Flutter project found in … or its parents` | `cd` into the project or use `fpack -C <dir>`; at a repository root fpack lists the Flutter apps below it |
| `Flutter SDK not found` | install Flutter, or `--flutter <sdk>` / `FPACK_FLUTTER`; FVM projects are detected automatically ([Which Flutter?](/guides/build-variants#which-flutter)) |
| a path dependency is missing | fpack checks before building and names the package; put it at the expected relative path |
| `which targets?` | name targets (`fpack build apk`), use `--all`, or set `build.targets` |
| `invalid configuration: …` | fpack.yaml has a mistake; the message names the line and suggests the key. [Reference](/configuration) |

## Output files

| Symptom | Cause and fix |
| --- | --- |
| `already exists (a real run stops unless --force)` | an artifact of this version exists: bump the version, `--build-number N`, or `--force` |
| `needs macOS` / `needs Windows` / `needs Linux` | Flutter cannot cross-compile; build on a runner of that OS ([concepts](/concepts#targets-and-artifacts)) |
| a target is skipped by `--all` | it needs a tool that is missing; `fpack doctor` says which |

## Android

| Symptom | Cause and fix |
| --- | --- |
| `no Java (JDK 17+) found` | install Android Studio (bundles a JDK) or JDK 17/21, then `flutter config --jdk-dir <path>` |
| `keystore password was incorrect` / `alias … does not exist` | check `FPACK_ANDROID_KEYSTORE_PASSWORD` / `FPACK_ANDROID_KEY_ALIAS`; fpack verifies them with keytool before building |
| `… defines productFlavors … but no flavor is set` | pass `--flavor` or set `build.flavor` |
| `flavor "x" not found` | use one of the listed flavors; `fpack doctor` shows the detected flavors |
| release builds are signed with the DEBUG key | configure signing ([Android step 4](/platforms/android#4-tell-fpack-about-it)); Google Play rejects debug-signed files |
| `store_file is set but missing store_password` | set `FPACK_ANDROID_KEYSTORE_PASSWORD` or `android.signing.store_password` |

## iOS

| Symptom | Cause and fix |
| --- | --- |
| no signing certificate / `No Accounts` | sign in to Xcode, or import the certificate and profile |
| `No profiles for '…' were found` | with manual signing list the bundle id (and extensions) in `ios.provisioning_profiles` |
| the team in a message is wrong | set `ios.team_id`; it wins over the project's `DEVELOPMENT_TEAM` |
| `pod install` failed | `cd ios && pod install` to see the error; `pod repo update` |
| duplicate build number on upload | bump `+N` or `--build-number` |

See [iOS](/platforms/ios#if-something-goes-wrong).

## macOS

| Symptom | Cause and fix |
| --- | --- |
| `signing identity … not found in the keychain` | use one of the listed Developer ID certificates exactly as printed, or build unsigned with `--no-sign` |
| notarization `Invalid` | fpack downloads Apple's log (`notary-log-<file>.json`) and summarizes the issues; usually unsigned nested code or a missing Hardened Runtime |
| upload timeouts while notarizing | fpack retries the upload twice; otherwise check the network and run the build again (nothing was submitted) |
| notarization takes long, you pressed Ctrl-C | Apple keeps processing: `fpack notarize status`, then `fpack notarize finish` |
| the signed app does not launch (`RBSRequestErrorDomain Code=5`, POSIX 163, "Launch failed") | restricted entitlements (push, iCloud, …) need a Developer ID provisioning profile: [Provisioning profiles](/platforms/macos-profiles); `--smoke-test` catches it before notarization |
| `uses restricted entitlements … only with a provisioning profile`, `embeds Xcode's development profile`, `is a development/ad hoc profile`, `is not authorized by` | see the validation table on [Provisioning profiles](/platforms/macos-profiles#3-what-fpack-checks) |
| macOS log: `Unsatisfied entitlements` / `No matching profile found` | the profile does not cover the signature (wrong certificate, team, app id or entitlement value); `fpack doctor` names which |
| `the app was not running … after launch` (smoke test) | run the printed `log show` command; usually unauthorized entitlements or a crash at startup |
| pkg is unsigned | set `macos.sign.installer_identity` (the Developer ID *Installer* certificate) |

## Windows

| Symptom | Cause and fix |
| --- | --- |
| `Inno Setup (ISCC.exe) not found` | `winget install --id JRSoftware.InnoSetup -e` or set `windows.inno_setup.iscc` |
| `Unable to read file … app.dill` | the project path has non-ASCII characters: move it, or enable UTF-8 in the Windows region settings |
| signtool not found | install the Windows SDK (`winget install Microsoft.WindowsSDK.10.0.26100`) or set `windows.sign.signtool` |
| signature verification warning | a self-signed certificate: only a warning; set `windows.sign.verify: false` to silence |

## Linux

| Symptom | Cause and fix |
| --- | --- |
| `Linux desktop toolchain incomplete, missing: …` | `sudo apt-get install -y clang cmake ninja-build pkg-config libgtk-3-dev` |
| `dpkg-deb` / `rpmbuild` / `appimagetool` / `snapcraft` missing | install them ([Linux](/platforms/linux)); fpack lists every missing tool at once |
| snap needs sudo | run `sudo -v` first (or passwordless sudo as on CI), or use `linux.snap.build_mode: lxd` |
| snap: LXD not initialised | `sudo lxd init --auto` and add your user to the `lxd` group |
| the snap does not match the target CPU | it contains the bundle built on this machine: build arm64 snaps on arm64 |

## Still stuck?

Open an issue at [github.com/Matkurban/fpack/issues](https://github.com/Matkurban/fpack/issues) with the output of `fpack doctor`, `fpack --version` and the log file named in the error (remove secrets first; fpack already redacts passwords).
