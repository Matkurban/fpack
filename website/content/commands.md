---
title: "Commands"
description: "Every fpack command and option."
---

```
fpack build [targets…] [options] [-- args passed to flutter]
fpack doctor          check the prerequisites of every target, with fixes
fpack list            list all targets, their artifact formats and whether this machine can build them
fpack init            write an fpack.yaml listing every key with English/Chinese comments (-y/--yes uses the
                      detected defaults; if it exists, writes fpack.yaml.new and shows a diff, --force overwrites)
fpack schema          print the JSON Schema of fpack.yaml (-o FILE writes a file)
fpack notarize status [dir|ID]   query the state of macOS notary submissions (reads dist/…/notarization.json)
fpack notarize finish [dir]      wait for the notary result, staple the ticket, update SHA256SUMS
fpack clean           remove build/fpack; --dist also removes dist/; --flutter-clean runs flutter clean; --all everything
fpack version         version information (same as --version / -V)
fpack help <command>  help for a command
```

Global options: `-C, --project DIR`, `--config FILE`, `--flutter SDK`, `--lang zh|en`, `-v, --verbose`, `--json`, `--no-color`, `-y, --yes`.

## build options

| Option | Description |
| --- | --- |
| `-a, --all` | build every target this machine can build; skip the rest with a reason |
| `-n, --dry-run` | print the full plan and exact commands, run nothing |
| `-m, --mode` / `--release` `--profile` `--debug` | build mode (default release) |
| `--flavor NAME` | Android productFlavor / Xcode scheme. A name that is not among the detected flavors is an error (exit 3, with a suggestion); if no flavors are detected it is only a warning |
| `-t, --target FILE` | entry point, e.g. `lib/main_prod.dart` |
| `--dart-define K=V` (repeatable) / `--dart-define-from-file FILE` | compile-time variables |
| `--build-name X.Y.Z` / `--build-number N` | override the pubspec version |
| `--split-per-abi[=true\|both]` / `--abis LIST` | per-ABI APKs; `both` also keeps the universal APK |
| `--obfuscate` / `--split-debug-info DIR` | obfuscation; symbols default to `<output>/debug-info/<platform>` |
| `-o, --output DIR` / `-f, --force` | output directory / allow overwriting |
| `--export-method M` / `--export-options-plist FILE` / `--no-codesign` | iOS export |
| `--sign` `--no-sign` `--sign-identity ID` `--installer-identity ID` `--notarize` `--no-notarize` `--notarize-no-wait` `--notary-profile NAME` `--dmg-tool T` | macOS signing (app / pkg), notarization (`--notarize-no-wait`: submit without waiting), DMG tool |
| `--base-href PATH` / `--wasm` | Web |
| `-- …` | everything after is passed to `flutter build`, e.g. `-- --no-tree-shake-icons` |

Examples:

```bash
fpack build apk aab --flavor prod --dart-define-from-file env/prod.json
fpack build apk --split-per-abi=both
fpack build ipa --export-method ad-hoc
fpack build ipa --no-codesign                # unsigned IPA (Payload layout)
fpack build macos dmg                        # signs/notarizes per fpack.yaml macos.sign (unsigned if not configured)
fpack build dmg --no-notarize                # quick local package
fpack build dmg --notarize-no-wait           # submit and exit; later: fpack notarize finish
fpack build macos dmg pkg                    # zip + DMG + pkg installer from a single flutter build
fpack build --all --json > result.json
fpack -C apps/client build web --base-href /app/
```

**Precedence**: command-line flags > `FPACK_*` environment variables > `fpack.yaml` > defaults.

**Interrupting**: one Ctrl-C stops the running flutter/gradle/xcodebuild gracefully (the whole process group), a second one kills it; exit code 130.

**Monorepos**: in a project subdirectory fpack walks up to `pubspec.yaml`; at the repository root it lists the Flutter projects it found — pick one with `-C <dir>`.

**Flutter SDK lookup**: `--flutter` → `FPACK_FLUTTER` → `flutter.sdk` in `fpack.yaml` → FVM (`.fvm/flutter_sdk`, `.fvmrc`) → `FLUTTER_ROOT` → `PATH` → common install locations (`~/develop/flutter`, `~/flutter`, …).

## Full help of every command

The output of `fpack help <command>` (generated from the CLI, always matches the released version).

<!-- BEGIN GENERATED HELP -->
### `fpack build`

```text
build and package targets into dist/

Usage:
  fpack build [targets...] [options] [-- extra flutter args]

Options:
  -a, --all                     build every target this machine can build (others are skipped with a reason)
  -n, --dry-run                 print the plan and exact commands, run nothing
  -m, --mode MODE               release (default) | profile | debug
  --release                     same as --mode release
  --profile                     same as --mode profile
  --debug                       same as --mode debug
  --flavor NAME                 build flavor (Android productFlavor / Xcode scheme)
  -t, --target FILE             entry point, e.g. lib/main_prod.dart
  --dart-define K=V             compile-time variable (repeatable)
  --dart-define-from-file FILE  JSON/.env file with defines (repeatable)
  --build-name X.Y.Z            override version name from pubspec.yaml
  --build-number N              override build number from pubspec.yaml
  --split-per-abi[=true|both]   APK: one file per ABI; =both also keeps the universal APK
  --abis LIST                   Android ABIs, e.g. arm64-v8a,armeabi-v7a
  --obfuscate                   obfuscate Dart code (symbols go to <output>/debug-info)
  --split-debug-info DIR        where to store debug symbols
  -o, --output DIR              output directory (default dist/{version}{+build})
  -f, --force                   overwrite existing artifacts
  --export-method M             iOS: app-store | ad-hoc | development | enterprise …
  --export-options-plist FILE   iOS: ExportOptions.plist
  --no-codesign                 iOS: build an unsigned IPA
  --sign                        macOS: Developer ID sign the app/DMG
  --no-sign                     macOS: do not sign (also disables notarization)
  --sign-identity ID            macOS: codesign identity (Developer ID Application)
  --installer-identity ID       macOS: .pkg signing identity (Developer ID Installer)
  --notarize                    macOS: notarize + staple the zip/DMG/pkg
  --no-notarize                 macOS: skip notarization (faster local builds)
  --notarize-no-wait            macOS: submit for notarization and finish without waiting (later: fpack notarize finish)
  --notary-profile NAME         macOS: notarytool keychain profile
  --dmg-tool T                  macOS: auto | hdiutil | create-dmg
  --base-href PATH              web: base href, e.g. /app/
  --wasm                        web: build with WebAssembly

Global options:
  -h, --help         show help
  -C, --project DIR  Flutter project directory (default: current directory or a parent)
  --config FILE      config file (default: fpack.yaml in the project)
  --flutter SDK      Flutter SDK root (default: FVM, PATH, FLUTTER_ROOT…)
  --lang zh|en       output language (default: from locale)
  -v, --verbose      stream full tool output
  --json             print a machine-readable JSON result to stdout
  --no-color         disable colors
  -y, --yes          assume yes / use defaults for questions

Examples:
  fpack build apk
  fpack build apk aab --flavor prod --dart-define-from-file env/prod.json
  fpack build apk --split-per-abi=both
  fpack build ipa --export-method ad-hoc
  fpack build ipa --no-codesign
  fpack build macos dmg                 # Developer ID signing/notarization from fpack.yaml macos.sign
  fpack build dmg --no-notarize
  fpack build macos dmg pkg             # one flutter build, three artifacts
  fpack build --all --json > result.json
  fpack build web --base-href /app/ -- --no-web-resources-cdn

Targets:
  apk       Android APK (universal and/or per-ABI)
  aab       Android App Bundle for Google Play
  ipa       iOS IPA (App Store / Ad Hoc / Enterprise / unsigned)
  macos     macOS .app, zipped with ditto (optionally Developer ID signed)
  dmg       macOS disk image (hdiutil or create-dmg, sign + notarize)
  pkg       macOS installer package (pkgbuild + productbuild, Developer ID Installer signing + notarization)
  windows   Windows portable app folder (zip)
  exe       Windows installer .exe (Inno Setup)
  msix      Windows MSIX package (needs the msix dev_dependency)
  linux     Linux app bundle (tar.gz)
  deb       Debian/Ubuntu package (dpkg-deb)
  rpm       Fedora/RHEL/openSUSE package (rpmbuild)
  appimage  Linux AppImage (appimagetool)
  web       Web build (zip of build/web, ready to deploy)
```

### `fpack doctor`

```text
check prerequisites per target and show how to fix what is missing

Usage:
  fpack doctor [targets...]

Global options:
  -h, --help         show help
  -C, --project DIR  Flutter project directory (default: current directory or a parent)
  --config FILE      config file (default: fpack.yaml in the project)
  --flutter SDK      Flutter SDK root (default: FVM, PATH, FLUTTER_ROOT…)
  --lang zh|en       output language (default: from locale)
  -v, --verbose      stream full tool output
  --json             print a machine-readable JSON result to stdout
  --no-color         disable colors
  -y, --yes          assume yes / use defaults for questions

Examples:
  fpack doctor
  fpack doctor apk ipa dmg
  fpack doctor --json
```

### `fpack list`

```text
list targets, output formats and what this machine can build

Usage:
  fpack list

Global options:
  -h, --help         show help
  -C, --project DIR  Flutter project directory (default: current directory or a parent)
  --config FILE      config file (default: fpack.yaml in the project)
  --flutter SDK      Flutter SDK root (default: FVM, PATH, FLUTTER_ROOT…)
  --lang zh|en       output language (default: from locale)
  -v, --verbose      stream full tool output
  --json             print a machine-readable JSON result to stdout
  --no-color         disable colors
  -y, --yes          assume yes / use defaults for questions

Examples:
  fpack list
  fpack list --json
```

### `fpack init`

```text
create a commented fpack.yaml for this project (interactive, or --yes for defaults)

Usage:
  fpack init [--yes] [--force]

Options:
  -f, --force  overwrite an existing fpack.yaml

Global options:
  -h, --help         show help
  -C, --project DIR  Flutter project directory (default: current directory or a parent)
  --config FILE      config file (default: fpack.yaml in the project)
  --flutter SDK      Flutter SDK root (default: FVM, PATH, FLUTTER_ROOT…)
  --lang zh|en       output language (default: from locale)
  -v, --verbose      stream full tool output
  --json             print a machine-readable JSON result to stdout
  --no-color         disable colors
  -y, --yes          assume yes / use defaults for questions

Examples:
  fpack init
  fpack init --yes
```

### `fpack schema`

```text
print the JSON schema of fpack.yaml (editor autocompletion and validation)

Usage:
  fpack schema [-o FILE]

Options:
  -o, --output FILE  write to FILE instead of stdout

Global options:
  -h, --help         show help
  -C, --project DIR  Flutter project directory (default: current directory or a parent)
  --config FILE      config file (default: fpack.yaml in the project)
  --flutter SDK      Flutter SDK root (default: FVM, PATH, FLUTTER_ROOT…)
  --lang zh|en       output language (default: from locale)
  -v, --verbose      stream full tool output
  --json             print a machine-readable JSON result to stdout
  --no-color         disable colors
  -y, --yes          assume yes / use defaults for questions

Examples:
  fpack schema -o .vscode/fpack.schema.json
  # fpack init adds this line so VS Code / IntelliJ (YAML plugin) use it automatically:
  # yaml-language-server: $schema=https://raw.githubusercontent.com/Matkurban/fpack/main/schema/fpack.schema.json
```

### `fpack notarize`

```text
check or finish macOS notarization submissions (after Ctrl-C or --notarize-no-wait)

Usage:
  fpack notarize status [DIR|ID]
  fpack notarize finish [DIR]

Global options:
  -h, --help         show help
  -C, --project DIR  Flutter project directory (default: current directory or a parent)
  --config FILE      config file (default: fpack.yaml in the project)
  --flutter SDK      Flutter SDK root (default: FVM, PATH, FLUTTER_ROOT…)
  --lang zh|en       output language (default: from locale)
  -v, --verbose      stream full tool output
  --json             print a machine-readable JSON result to stdout
  --no-color         disable colors
  -y, --yes          assume yes / use defaults for questions

Examples:
  fpack notarize status                  # every submission in the newest notarization.json
  fpack notarize status dist/1.2.0+5     # a specific output directory
  fpack notarize status 2efe2717-52ef-…  # one submission id
  fpack notarize finish                  # wait, staple accepted files, update SHA256SUMS

status asks Apple for the state of each submission recorded in
<output>/notarization.json (written by fpack build right after the upload).
finish waits for pending submissions, staples the ticket to accepted
DMG/pkg/app zips (the app is unzipped, stapled and zipped again), rewrites
the checksum file, and for rejected ones downloads and summarizes Apple's log.
Credentials come from fpack.yaml macos.sign (or the recorded keychain
profile / API key; Apple ID submissions need FPACK_NOTARY_PASSWORD).
```

### `fpack clean`

```text
remove fpack's temporary files (and optionally dist/ or flutter build output)

Usage:
  fpack clean [--dist] [--flutter-clean] [--all]

Options:
  --dist           also delete the output directory (asks first)
  --flutter-clean  also run `flutter clean`
  --all            --dist + --flutter-clean
  -n, --dry-run    show what would be deleted

Global options:
  -h, --help         show help
  -C, --project DIR  Flutter project directory (default: current directory or a parent)
  --config FILE      config file (default: fpack.yaml in the project)
  --flutter SDK      Flutter SDK root (default: FVM, PATH, FLUTTER_ROOT…)
  --lang zh|en       output language (default: from locale)
  -v, --verbose      stream full tool output
  --json             print a machine-readable JSON result to stdout
  --no-color         disable colors
  -y, --yes          assume yes / use defaults for questions

Examples:
  fpack clean
  fpack clean --dist --yes
  fpack clean --all
```

### `fpack version`

```text
print version information

Usage:
  fpack version

Global options:
  -h, --help         show help
  -C, --project DIR  Flutter project directory (default: current directory or a parent)
  --config FILE      config file (default: fpack.yaml in the project)
  --flutter SDK      Flutter SDK root (default: FVM, PATH, FLUTTER_ROOT…)
  --lang zh|en       output language (default: from locale)
  -v, --verbose      stream full tool output
  --json             print a machine-readable JSON result to stdout
  --no-color         disable colors
  -y, --yes          assume yes / use defaults for questions

Examples:
  fpack --version
```
<!-- END GENERATED HELP -->
