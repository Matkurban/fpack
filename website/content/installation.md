---
title: "Installation"
description: "Install fpack and understand how it finds its native core."
---

Requires Dart 3.8+ (bundled with Flutter). Pick one:

| Method | Command | Notes |
| --- | --- | --- |
| pub.dev | `dart pub global activate fpack` | recommended |
| git | `dart pub global activate --source git https://github.com/Matkurban/fpack` | the first run downloads the core from the GitHub Release (SHA-256 verified); builds it with local Go only if the download is unavailable |
| local path | `dart pub global activate --source path /path/to/fpack` | works offline: the release tarball contains prebuilt binaries for all 6 platforms |

Make sure `~/.pub-cache/bin` (Windows: `%LOCALAPPDATA%\Pub\Cache\bin`) is on your `PATH`:

```bash
echo 'export PATH="$PATH:$HOME/.pub-cache/bin"' >> ~/.zshrc && source ~/.zshrc
fpack --version
```

<a id="launcher"></a>

## How the launcher finds the native core

On every run `fpack` looks for an `fpack-core` whose version **exactly matches the Dart package** (checked with `fpack-core --core-version`), in this order. **A verified prebuilt binary always wins; a local build is only the last resort**:

1. the `FPACK_CORE` environment variable (development)
2. the user cache: macOS `~/Library/Caches/fpack/<version>/<os>-<arch>/`, Linux `$XDG_CACHE_HOME/fpack/…` (default `~/.cache/fpack`), Windows `%LOCALAPPDATA%\fpack\…` (override with `FPACK_HOME`)
3. the bundled prebuilt binary `prebuilt/<os>-<arch>/` (when present): verified against the SHA-256 in `prebuilt/manifest.json`, then copied into the cache
4. a download of this platform's binary (≈4 MB) from the GitHub Release, verified against `checksums.txt` — a mismatching file is never executed
5. **only if the download fails or is disabled**: a local build from the bundled Go sources (`go build -trimpath`, dependencies are vendored, no network needed), with a clear note that it fell back to a local build and which Go it used:

   ```
   fpack: note: no verified prebuilt core available (download: …connection refused…);
   fpack: falling back to a local build from the bundled sources with go1.27.2 (/usr/local/go/bin/go) – first run only, ~30s…
   ```

Controls:

| Variable | Effect |
| --- | --- |
| `FPACK_NO_DOWNLOAD=1` | offline / air-gapped: never download; use the bundled binary or build locally with Go |
| `FPACK_DOWNLOAD_URL` | download from a mirror (a directory with `checksums.txt` and `fpack-core-<os>-<arch>`, laid out like the GitHub Release) |
| `FPACK_GO=/path/to/go` / `FPACK_GO=none` | the Go used for the fallback build / never build locally |
| `FPACK_REBUILD=1` | development: ignore the cache and build with local Go first (after changing the Go sources) |
| `FPACK_HOME` | cache directory |

A cached core is reused until the version changes; to replace a locally built core with the downloaded binary, delete the cache directory (`fpack --wrapper-info` shows its path).

`fpack --wrapper-info` shows each step's path and the binary that is finally used.

> **pub.dev package size trade-off**: the binaries for 6 platforms are ≈23 MB (≈10 MB compressed). Bundling them in the pub package → works offline right after install, but every user downloads every platform's binary; not bundling → a tiny package (≈350 KB) that downloads this platform's ≈4 MB binary from the GitHub Release on first run (and only builds with local Go, ≈30 s, if that is unavailable). The release workflow's repository variable `BUNDLE_BINARIES` decides; the `fpack-vX.Y.Z.tar.gz` on the GitHub Release always contains all binaries for offline / intranet use.

## Requirements per target

fpack itself only needs Dart. The targets need their usual toolchains – run `fpack doctor` to see what is missing on your machine, with an install command for each item:

| Targets | Needs |
| --- | --- |
| `apk`, `aab` | Flutter, Android SDK, Java 17+ (Android Studio bundles one) |
| `ipa`, `macos`, `dmg`, `pkg` | macOS with Xcode (+ CocoaPods when the project has a Podfile) |
| `windows`, `exe`, `msix` | Windows with Visual Studio (C++ desktop workload); Inno Setup 6 for `exe`; the `msix` dev dependency for `msix` |
| `linux`, `deb`, `rpm`, `appimage` | Linux with clang, cmake, ninja, pkg-config, GTK 3 headers; `dpkg-deb`, `rpmbuild`, `appimagetool` |
| `web` | Flutter only |

## Upgrading

```bash
dart pub global activate fpack    # installs the latest version
fpack --version
```

Each fpack version uses its own native core (downloaded once per version and cached).
