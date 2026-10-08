# prebuilt/

Native `fpack-core` binaries, one folder per host:

```
prebuilt/
  manifest.json          # {"version": "...", "binaries": {"darwin-arm64": {"sha256": ...}}}
  checksums.txt          # sha256sum format, same names as GitHub release assets
  darwin-arm64/fpack-core
  darwin-amd64/fpack-core
  linux-amd64/fpack-core
  linux-arm64/fpack-core
  windows-amd64/fpack-core.exe
  windows-arm64/fpack-core.exe
```

They are produced by `scripts/build_binaries.sh` and are **not committed to
git** (see `.gitignore`); release archives and (optionally) the pub.dev
package include them. The Dart wrapper verifies each binary against
`manifest.json` before copying it into the user cache.
