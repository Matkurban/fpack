#!/usr/bin/env bash
# Fails if pubspec.yaml, lib/src/version.dart and the Go core disagree.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
pub="$(sed -n 's/^version: *//p' "$ROOT/pubspec.yaml" | tr -d '[:space:]')"
dart="$(sed -n "s/^const String packageVersion = '\(.*\)';/\1/p" "$ROOT/lib/src/version.dart")"
go="$(sed -n 's/^const Version = "\(.*\)"/\1/p' "$ROOT/go/internal/version/version.go")"
if [ "$pub" != "$dart" ] || [ "$pub" != "$go" ]; then
  echo "version mismatch: pubspec=$pub dart=$dart go=$go" >&2
  exit 1
fi
if [ -n "${1:-}" ] && [ "${1#v}" != "$pub" ]; then
  echo "tag $1 does not match version $pub" >&2
  exit 1
fi
echo "$pub"
