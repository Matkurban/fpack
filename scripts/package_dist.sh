#!/usr/bin/env bash
# Builds all prebuilt binaries and creates a self-contained archive that can
# be installed offline with `dart pub global activate --source path`.
#
#   scripts/package_dist.sh [output.tar.gz]
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
OUT="${1:-$ROOT/../fpack-dist.tar.gz}"
case "$OUT" in /*) ;; *) OUT="$PWD/$OUT" ;; esac
cd "$ROOT"
VERSION="$(scripts/check_versions.sh)"
scripts/build_binaries.sh
STAGE="$(mktemp -d)"
trap 'rm -rf "$STAGE"' EXIT
DEST="$STAGE/fpack"
mkdir -p "$DEST"
# Tracked files (respects .gitignore) + prebuilt binaries + git history.
git ls-files -z | xargs -0 -I{} cp --parents {} "$DEST/" 2>/dev/null || \
  git ls-files | while read -r f; do mkdir -p "$DEST/$(dirname "$f")"; cp -p "$f" "$DEST/$f"; done
cp -R prebuilt "$DEST/"
cp -R .git "$DEST/.git"
tar -C "$STAGE" -czf "$OUT" fpack
echo "fpack $VERSION → $OUT ($(du -h "$OUT" | cut -f1))"
