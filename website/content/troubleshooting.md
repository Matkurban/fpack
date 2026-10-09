---
title: "Troubleshooting"
description: "Reading failures, common problems and exit codes."
---

When a build fails, fpack prints the key error excerpt (Gradle's "What went wrong", Xcode errors, …), one concrete fix and the path of the full log in `build/fpack/logs/`. Add `-v` to stream the full tool output, and `--dry-run` to see every command without running it.

## Common problems

| Message | Fix |
| --- | --- |
| `Flutter SDK not found` | install Flutter or pass `--flutter <sdk>` / set `FPACK_FLUTTER`; FVM projects are detected automatically |
| `no Java (JDK 17+) found` | install Android Studio (bundles a JDK) or JDK 17/21, then `flutter config --jdk-dir <path>` |
| `flavor "x" not found` | use one of the listed flavors; `fpack doctor` shows the detected flavors |
| `… defines productFlavors … but no flavor is set` | pass `--flavor` or set `build.flavor` |
| `keystore password was incorrect` / `alias … does not exist` | check `FPACK_ANDROID_KEYSTORE_PASSWORD` / `FPACK_ANDROID_KEY_ALIAS` – fpack verifies them with keytool before building |
| `signing identity … not found in the keychain` | use one of the listed Developer ID certificates or build unsigned with `--no-sign` |
| notarization `Invalid` | fpack downloads Apple's log (`notary-log-<file>.json`) and summarizes the issues; usually unsigned nested code or a missing Hardened Runtime |
| upload timeouts while notarizing | fpack retries the upload twice; otherwise check the network and run the build again – nothing was submitted |
| `already exists (a real run stops unless --force)` | bump the version, use `--build-number N` or pass `--force` |
| `needs macOS/Windows/Linux` | Flutter cannot cross-compile desktop/iOS apps; build on a runner of that OS |
| `ISCC.exe not found` | install Inno Setup 6 or set `windows.inno_setup.iscc` |
| `dpkg-deb` / `rpmbuild` / `appimagetool` missing | install them (see [Linux](/platforms/linux)) – fpack lists every missing tool at once |
| At the repository root: "found Flutter projects …" | pick one with `fpack -C <dir>` |

## Exit codes

| Code | Meaning |
| --- | --- |
| 0 | success (`--dry-run`: the plan was produced) |
| 1 | at least one target failed to build/package, or a hook failed; `fpack notarize`: a submission was rejected or a query failed |
| 2 | usage or configuration error (unknown flag, unknown target, invalid fpack.yaml) |
| 3 | prerequisites missing / artifact already exists / nothing buildable / Flutter not found |
| 130 | interrupted by Ctrl-C (an interrupted notarization wait keeps processing on Apple's side) |

With `--json`, stdout carries only the JSON result (targets, states, artifact paths, sizes, SHA-256/512, notarization state `notarization.state`, durations, error excerpts, fixes); all human-readable output goes to stderr.
