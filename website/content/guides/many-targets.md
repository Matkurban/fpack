---
title: "Many targets, one command"
description: "Build everything this machine can build, preview plans, get JSON results, work in monorepos and clean up."
---

## Choose the targets

```bash
fpack build apk aab ipa          # name them
fpack build --all                # every target this machine can build
fpack build                      # the targets in build.targets
```

```yaml
build:
  targets: [apk, aab, ipa, dmg]
```

Without targets and without `build.targets`, fpack stops with exit code 2 and asks which targets to build. Aliases work on the command line and in `build.targets`:

| Alias | Target |
| --- | --- |
| `bundle`, `appbundle` | `aab` |
| `android` | `apk` |
| `ios` | `ipa` |
| `mac`, `osx`, `app` | `macos` |
| `win`, `zip`, `portable` | `windows` |
| `setup`, `installer`, `inno` | `exe` |
| `tar`, `tgz`, `tar.gz` | `linux` |
| `debian` | `deb` |
| `fedora` | `rpm` |
| `image`, `AppImage` | `appimage` |
| `snapcraft` | `snap` |

## What `--all` skips

`--all` builds what this machine *can* build and skips the rest, each with the reason (for example "needs macOS"). Targets needing a missing optional tool (`appimage` without appimagetool, `snap` without snapcraft) are skipped too. Naming a target explicitly is stricter: a target this machine cannot build is an error (exit code 3) that points to a CI runner of that OS.

## Shared builds

One `flutter build` is shared per platform in a run: `macos` + `dmg` + `pkg` build once, `linux` + `deb` + `rpm` + `appimage` + `snap` build once, `windows` + `exe` + `msix` build once. Asking for many formats of one platform costs little extra time.

## Rehearse and inspect

```bash
fpack build --all --dry-run     # print the plan and every command; run nothing
fpack list                      # every target: platform, extension, whether this machine can build it
fpack doctor apk ipa dmg        # prerequisites of these targets only
```

## Machine-readable results

```bash
fpack build --all --json > result.json
```

With `--json`, stdout carries only the JSON result (targets, status, artifact paths, sizes, hashes, notarization state, durations, error excerpts and fixes); human output goes to stderr. Fields and examples: [Exit codes and JSON](/exit-codes#json).

## Interrupting

One Ctrl-C stops the running flutter, Gradle or xcodebuild process group gracefully; a second one kills it. The exit code is 130. An interrupted notarization wait keeps processing at Apple; finish it with `fpack notarize finish`.

## Monorepos and other directories

fpack walks up from the current directory to `pubspec.yaml`. At a repository root with several Flutter projects it lists them; pick one with `-C`:

```bash
fpack -C apps/client build web --base-href /app/
fpack --config ci/fpack.ci.yaml build apk      # another config file
```

(`--project DIR` / `-C`, `--config FILE`, `FPACK_CONFIG`.)

## Verbose output, colors and language

| Option | Effect |
| --- | --- |
| `-v`, `--verbose` | stream the full output of every tool |
| `--no-color` | no colors (also `NO_COLOR`, `FPACK_NO_COLOR`; `FORCE_COLOR` forces them) |
| `--lang zh\|en` | output language (also `FPACK_LANG`; default: your OS language) |
| `-y`, `--yes` | assume yes / use defaults for questions (for example in `fpack init`, `fpack clean`) |

Without a TTY, spinners and colors are off and fpack prints a heartbeat line every 60 seconds so a CI job does not time out on silence.

## Clean up

```bash
fpack clean                     # remove build/fpack (logs, temporary files)
fpack clean --dist              # also delete the output directory (asks first)
fpack clean --flutter-clean     # also run `flutter clean`
fpack clean --all --dry-run     # show what would be deleted
```

## Next

The [reference](/commands) for every command and option, and [CI recipes](/ci).
