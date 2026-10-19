---
title: "Exit codes and JSON output"
description: "What every exit code means, and the machine-readable JSON result for scripts and CI."
---

Scripts and CI jobs should look at the exit code first, and at the JSON result for details.

## Exit codes

<!-- BEGIN GENERATED EXIT -->
| Code | Meaning |
| --- | --- |
| 0 | success (`--dry-run`: the plan was produced) |
| 1 | at least one target failed to build/package, or a hook failed; `fpack doctor`: problems were found; `fpack notarize`: a submission was rejected or a query failed |
| 2 | usage or configuration error (unknown flag or target, no targets and no `build.targets`, invalid fpack.yaml, no Flutter project found) |
| 3 | prerequisites missing / artifact already exists / nothing buildable / Flutter SDK not found; `fpack notarize`: no notarization record found |
| 130 | interrupted by Ctrl-C (an interrupted notarization wait keeps processing on Apple's side) |
<!-- END GENERATED EXIT -->

Use them to react: `3` means a prerequisite is missing and nothing was built (fix the machine, do not retry blindly); `2` means your command or configuration is wrong; `1` means something was built or checked and failed.

```bash
fpack build apk aab || echo "fpack exited with $?"
```

<a id="json"></a>

## JSON output

`--json` prints exactly one JSON document to stdout; all human-readable output goes to stderr, so you can redirect safely:

```bash
fpack build apk aab --json > result.json
fpack doctor --json
fpack list --json
```

A successful web build looks like this (shortened):

```json
{
  "fpackVersion": "1.1.16",
  "project": "my_app",
  "version": "1.0.0",
  "buildNumber": "1",
  "flutter": "3.47.7",
  "host": "linux/amd64",
  "mode": "release",
  "outputDir": "/work/my_app/dist/1.0.0+1",
  "checksumsFile": "/work/my_app/dist/1.0.0+1/SHA256SUMS",
  "logDir": "/work/my_app/build/fpack/logs",
  "targets": [
    {
      "target": "web",
      "platform": "web",
      "status": "success",
      "artifacts": [
        {
          "path": "/work/my_app/dist/1.0.0+1/my_app-1.0.0+1-web.zip",
          "file": "my_app-1.0.0+1-web.zip",
          "kind": "Web (zip)",
          "size": 14431167,
          "sha256": "7f78faa3…"
        }
      ],
      "durationMs": 22802
    }
  ],
  "exitCode": 0,
  "durationMs": 22813,
  "dryRun": false,
  "success": true
}
```

### Fields of `fpack build --json`

| Field | Meaning |
| --- | --- |
| `fpackVersion`, `project`, `projectRoot`, `version`, `buildNumber`, `flutter`, `flutterRoot`, `host`, `mode`, `flavor` | what was built, with which tools |
| `outputDir`, `checksumsFile`, `logDir` | where artifacts, checksums and logs are |
| `targets[]` | one entry per target |
| `targets[].status` | `success`, `failed`, `skipped` or `planned` (dry run) |
| `targets[].reason`, `fix` | why a target was skipped and how to make it buildable |
| `targets[].error`, `hint`, `log`, `errorExcerpt[]` | failure details: message, suggested fix, full log path, key lines |
| `targets[].warnings[]`, `notes[]`, `commands[]` | warnings, notes and the commands that ran or would run |
| `targets[].artifacts[]` | `path`, `file`, `kind`, `arch`, `size`, `sha256` (and `sha512`) |
| `targets[].artifacts[].notarization` | for notarized macOS files: `state` (`submitted`, `accepted`, `stapled`, `invalid`), `id`, `status`, `record` (the `NOTARIZATION.md` path) |
| `targets[].durationMs`, `durationMs` | timing |
| `hookError` | the error of a failed hook |
| `hooks[]` | the hook commands a dry run would execute |
| `conflicts[]` | existing artifacts a real run would stop on (use `--force` to replace them) |
| `exitCode`, `success`, `dryRun`, `interrupted` | the outcome |

### Examples

```bash
# upload every artifact
jq -r '.targets[].artifacts[].path' result.json

# did every target succeed?
jq -e '.success' result.json

# list skipped targets and why
jq -r '.targets[] | select(.status=="skipped") | "\(.target): \(.reason)"' result.json
```

## Next

[CI recipes](/ci) put these to work; [troubleshooting](/troubleshooting) explains what to do for each failure.
