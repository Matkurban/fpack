---
title: "Architecture and development"
description: "How fpack is built, and how to develop and release it."
---

```
fpack/
  bin/fpack.dart            Dart entry point (pub executable)
  lib/src/                  launcher: platform detection, core resolution (cache → bundled → download → local go build), SHA-256, signal forwarding
  go/                       native core (Go, dependencies vendored)
    cmd/fpack-core/         main
    internal/cli            argument parsing, commands, help
    internal/build          orchestration: classify → preflight → shared flutter steps → package → checksums → summary
    internal/targets        per-target preflight, flutter args, artifact lookup, packaging commands (dry-run and real run share the code)
    internal/config         key registry (drives env vars, validation, JSON Schema, init template and doc tables), fpack.yaml parsing
    internal/project        pubspec / Gradle / Xcode project info (read-only)
    internal/flutter        Flutter SDK lookup (FVM / PATH / …)
    internal/runner         subprocesses (process groups, Ctrl-C, logs, secret redaction)
    internal/hints          common errors → fixes
    internal/ui, i18n, pack, host, doctor, version
  schema/                   fpack.schema.json (generated from the key registry)
  prebuilt/                 prebuilt binaries + manifest.json (release artifacts, not in git)
  scripts/                  build_binaries.sh, check_versions.sh, package_dist.sh
  .github/workflows/        ci.yml, e2e.yml, release.yml
```

## Development and releases

```bash
cd go && go vet ./... && go test ./...      # Go core
FPACK_UPDATE=1 go test ./internal/config    # after changing the key registry: regenerate the schema and doc/configuration.md tables
dart pub get && dart analyze && dart test   # Dart launcher
scripts/build_binaries.sh                   # cross-compile 6 platforms → prebuilt/
scripts/package_dist.sh ../fpack-dist.tar.gz  # self-contained offline bundle
FPACK_CORE=$PWD/prebuilt/darwin-arm64/fpack-core fpack doctor   # debug with a specific core
```

Releasing: bump the version in `pubspec.yaml`, `lib/src/version.dart` and `go/internal/version/version.go` together (`scripts/check_versions.sh` checks them), update `CHANGELOG.md`, push a `vX.Y.Z` tag. `release.yml` cross-compiles, creates the GitHub Release (binaries + `checksums.txt` + offline bundle) and optionally publishes to pub.dev.
