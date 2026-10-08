#!/usr/bin/env bash
# Cross-compiles fpack-core for all supported hosts into prebuilt/ and writes
# prebuilt/manifest.json (read by the Dart wrapper) and prebuilt/checksums.txt
# (sha256sum format, published with GitHub releases).
#
#   scripts/build_binaries.sh                # all 6 targets
#   scripts/build_binaries.sh darwin/arm64   # just one
#   OUT=dist/bin scripts/build_binaries.sh   # also copy release assets there
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
"$ROOT/scripts/check_versions.sh" >/dev/null

GO="${GO:-$(command -v go || echo /usr/local/go/bin/go)}"
# Release binaries are built with exactly the toolchain pinned in go.mod.
want="$(sed -n 's/^toolchain //p' go/go.mod)"
have="$("$GO" env GOVERSION)"
if [ -n "$want" ] && [ "$have" != "$want" ] && [ "${ALLOW_GO_MISMATCH:-}" != 1 ]; then
  echo "error: go.mod pins $want but $GO is $have (set GO=/path/to/$want/bin/go, or ALLOW_GO_MISMATCH=1)" >&2
  exit 1
fi
VERSION="$(sed -n 's/^const Version = "\(.*\)"/\1/p' go/internal/version/version.go)"
TARGETS=("$@")
if [ ${#TARGETS[@]} -eq 0 ]; then
  TARGETS=(darwin/arm64 darwin/amd64 linux/amd64 linux/arm64 windows/amd64 windows/arm64)
fi

sha256() {
  if command -v sha256sum >/dev/null; then sha256sum "$1" | cut -d' ' -f1; else shasum -a 256 "$1" | cut -d' ' -f1; fi
}

mkdir -p prebuilt
echo "fpack-core $VERSION  ($("$GO" version | cut -d' ' -f3))"
for t in "${TARGETS[@]}"; do
  os="${t%/*}"; arch="${t#*/}"
  exe="fpack-core"; [ "$os" = windows ] && exe="fpack-core.exe"
  out="prebuilt/$os-$arch/$exe"
  rm -rf "prebuilt/$os-$arch"; mkdir -p "prebuilt/$os-$arch"
  (cd go && CGO_ENABLED=0 GOOS="$os" GOARCH="$arch" GOFLAGS=-mod=vendor GOTOOLCHAIN=local \
    "$GO" build -trimpath -ldflags "-s -w" -o "../$out" ./cmd/fpack-core)
  printf '  %-15s %6s KB\n' "$os/$arch" "$(( $(wc -c < "$out") / 1024 ))"
done

# manifest.json + checksums.txt cover every binary present in prebuilt/.
{
  echo "{"
  echo "  \"version\": \"$VERSION\","
  echo "  \"binaries\": {"
  first=1
  for d in prebuilt/*-*/; do
    id="$(basename "$d")"
    f="$(ls "$d" | head -1)"
    [ -n "$f" ] || continue
    [ $first -eq 1 ] || echo ","
    first=0
    printf '    "%s": {"file": "%s", "sha256": "%s", "size": %s}' "$id" "$f" "$(sha256 "$d$f")" "$(wc -c < "$d$f" | tr -d ' ')"
  done
  echo
  echo "  }"
  echo "}"
} > prebuilt/manifest.json

: > prebuilt/checksums.txt
for d in prebuilt/*-*/; do
  id="$(basename "$d")"; f="$(ls "$d" | head -1)"
  ext=""; [[ "$f" == *.exe ]] && ext=".exe"
  echo "$(sha256 "$d$f")  fpack-core-$id$ext" >> prebuilt/checksums.txt
done

if [ -n "${OUT:-}" ]; then
  mkdir -p "$OUT"
  for d in prebuilt/*-*/; do
    id="$(basename "$d")"; f="$(ls "$d" | head -1)"
    ext=""; [[ "$f" == *.exe ]] && ext=".exe"
    cp "$d$f" "$OUT/fpack-core-$id$ext"
  done
  cp prebuilt/checksums.txt "$OUT/checksums.txt"
  echo "release assets → $OUT"
fi
echo "manifest → prebuilt/manifest.json"
