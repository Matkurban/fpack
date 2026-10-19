---
title: "Hooks"
description: "Run your own scripts before and after a build or a package: upload, checks, notifications."
---

Hooks are shell commands fpack runs at fixed points. Use them to generate code before building, check artifact sizes, upload to a store or a server, or send a notification.

```yaml
hooks:
  pre_build:
    - dart run build_runner build --delete-conflicting-outputs
  pre_package:
    apk: ['./scripts/check_size.sh']
  post_package:
    dmg: ['./scripts/upload.sh "$FPACK_ARTIFACT"']
  post_build:
    - ./scripts/notify.sh
```

## When each hook runs

| Hook | When | Notes |
| --- | --- | --- |
| `hooks.pre_build` | once, before the first flutter build | a failure stops the run |
| `hooks.pre_package.<target>` | before a target's packaging steps | sets `FPACK_TARGET` |
| `hooks.post_package.<target>` | after a target's artifacts exist | sets `FPACK_ARTIFACT` (the first file) and `FPACK_ARTIFACTS` |
| `hooks.post_build` | once, at the end | sets `FPACK_ARTIFACTS` (one per line) and `FPACK_SUCCESS` (`1` / `0`). It also runs after build failures, but not when every target already failed its checks (a missing tool, an unknown flavor) |

A failing hook means exit code 1.

## Environment

Hooks run in the project root through `sh -c` (Windows: `cmd /S /C`, so quoted paths work as typed) with these variables:

| Variable | Value |
| --- | --- |
| `FPACK_PROJECT_ROOT` | the project directory |
| `FPACK_OUTPUT_DIR` | the artifact directory |
| `FPACK_VERSION`, `FPACK_BUILD_NUMBER` | the version name and build number |
| `FPACK_MODE`, `FPACK_FLAVOR` | build mode and flavor |
| `FPACK_TARGET` | the target (`pre_package` / `post_package`) |
| `FPACK_ARTIFACT`, `FPACK_ARTIFACTS` | the artifact(s) (`post_package`, `post_build`) |
| `FPACK_SUCCESS` | `1` or `0` (`post_build`) |

Write `$VAR` (Windows `cmd`: `%VAR%`) for variables that exist only while the hook runs. `${VAR}` is expanded earlier, when `fpack.yaml` is loaded ([Secrets and variables](/guides/secrets)).

## Examples

Upload a signed IPA after packaging:

```yaml
hooks:
  post_package:
    ipa: ['xcrun altool --upload-app -f "$FPACK_ARTIFACT" -t ios --apiKey "$ASC_KEY_ID" --apiIssuer "$ASC_ISSUER"']
```

Copy everything to a release server, once, after the last target:

```yaml
hooks:
  post_build:
    - 'test "$FPACK_SUCCESS" = 1 && rsync -a "$FPACK_OUTPUT_DIR/" releases@example.com:/srv/app/'
```

Fail the build when the APK is too large:

```yaml
hooks:
  pre_package: { apk: ['./scripts/check_apk_size.sh'] }
```

## Next

[Many targets, one command](/guides/many-targets).
