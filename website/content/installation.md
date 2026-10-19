---
title: "Installation"
description: "Install fpack, put it on your PATH, and learn how it finds its native core."
---

fpack needs only **Dart 3.8+**, which is bundled with Flutter. The targets you build need their usual toolchains (see [what each target needs](#requirements-per-target)); `fpack doctor` checks them for you.

## 1. Install

```bash
dart pub global activate fpack
```

This installs the `fpack` command from [pub.dev](https://pub.dev/packages/fpack). Other ways to install:

| Method | Command | Notes |
| --- | --- | --- |
| pub.dev | `dart pub global activate fpack` | recommended |
| git | `dart pub global activate --source git https://github.com/Matkurban/fpack` | the first run downloads the core from the GitHub Release (SHA-256 verified); it builds with a local Go only if the download is unavailable |
| local path | `dart pub global activate --source path /path/to/fpack` | works offline: the release tarball contains prebuilt binaries for all 6 platforms |

## 2. Put it on your PATH

Dart installs global tools into the pub cache. That folder must be on your `PATH`:

::: tabs
@@ macOS
```bash
echo 'export PATH="$PATH:$HOME/.pub-cache/bin"' >> ~/.zshrc && source ~/.zshrc
```
@@ Linux
```bash
echo 'export PATH="$PATH:$HOME/.pub-cache/bin"' >> ~/.bashrc && source ~/.bashrc
```
@@ Windows
Add `%LOCALAPPDATA%\Pub\Cache\bin` to your user `PATH` (Settings → System → About → Advanced system settings → Environment Variables), then open a new terminal.
:::

## 3. Check

```bash
fpack --version
```

It prints the installed version number. If the shell says `command not found`, the PATH step above is missing. Then try [your first build](/getting-started).

::: tip
`fpack --wrapper-info` shows which native core is used and where it came from. It is the first thing to run when something about the installation looks odd.
:::

## Upgrading

```bash
dart pub global activate fpack    # installs the latest version
fpack --version
```

Each fpack version uses its own native core, downloaded once per version and cached.

## Requirements per target

fpack itself only needs Dart. Each target needs its usual toolchain:

| Targets | Needs |
| --- | --- |
| `apk`, `aab` | Flutter, Android SDK, Java 17+ (Android Studio bundles one) |
| `ipa`, `macos`, `dmg`, `pkg` | macOS with Xcode (+ CocoaPods when the project has a Podfile) |
| `windows`, `exe`, `msix` | Windows with Visual Studio (C++ desktop workload); Inno Setup 6 for `exe`; the `msix` dev dependency for `msix` |
| `linux`, `deb`, `rpm`, `appimage`, `snap` | Linux with clang, cmake, ninja, pkg-config, GTK 3 headers; `dpkg-deb`, `rpmbuild`, `appimagetool`, `snapcraft` (plus LXD or Multipass unless you build on Ubuntu 24.04) |
| `web` | Flutter only |

Run `fpack doctor` in a project to see exactly what is missing on *your* machine, with an install command for each item.

<a id="launcher"></a>

## How the launcher finds the native core

The `fpack` command is a small Dart launcher. The real work is done by `fpack-core`, a native Go binary (about 4 MB, ~10 ms startup). On every run the launcher looks for a core whose version **exactly matches the Dart package** (checked with `fpack-core --core-version`), in this order. **A verified prebuilt binary always wins; a local build is only the last resort.**

1. the `FPACK_CORE` environment variable (development)
2. the user cache: macOS `~/Library/Caches/fpack/<version>/<os>-<arch>/`, Linux `$XDG_CACHE_HOME/fpack/…` (default `~/.cache/fpack`), Windows `%LOCALAPPDATA%\fpack\…` (override with `FPACK_HOME`)
3. the bundled prebuilt binary `prebuilt/<os>-<arch>/` (when present): verified against the SHA-256 in `prebuilt/manifest.json`, then copied into the cache
4. a download of this platform's binary from the GitHub Release, verified against `checksums.txt`: a mismatching file is never executed
5. **only if the download fails or is disabled**: a local build from the bundled Go sources (`go build -trimpath`, dependencies are vendored, no network needed), with a clear note that it fell back to a local build and which Go it used:

   ```text
   fpack: note: no verified prebuilt core available (download: …connection refused…);
   fpack: falling back to a local build from the bundled sources with go1.27.2 (/usr/local/go/bin/go) – first run only, ~30s…
   ```

`fpack --wrapper-info` prints each step's path and the binary that is finally used. A cached core is reused until the version changes; to replace a locally built core with the downloaded binary, delete the cache directory.

### Offline, mirrors and development

| Variable | Effect |
| --- | --- |
| `FPACK_NO_DOWNLOAD=1` | offline / air-gapped: never download; use the bundled binary or build locally with Go |
| `FPACK_DOWNLOAD_URL` | download from a mirror (a directory with `checksums.txt` and `fpack-core-<os>-<arch>` (`.exe` on Windows), laid out like the GitHub Release) |
| `FPACK_GO=/path/to/go` / `FPACK_GO=none` | the Go used for the fallback build / never build locally |
| `FPACK_REBUILD=1` | development: ignore the cache and build with local Go first (after changing the Go sources) |
| `FPACK_HOME` | cache directory |
| `FPACK_CORE` | use exactly this core binary |

::: note
**Package size trade-off.** The binaries for 6 platforms are about 23 MB (10 MB compressed). Bundling them in the pub package makes fpack work offline right after install but every user downloads every platform's binary; not bundling gives a tiny package (about 350 KB) that downloads this platform's binary on first run. The release workflow's repository variable `BUNDLE_BINARIES` decides; the `fpack-vX.Y.Z.tar.gz` on the GitHub Release always contains all binaries for offline and intranet use.
:::

## Next

[Your first build](/getting-started): `doctor`, a dry run, and a real build.
