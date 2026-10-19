---
title: "Flavors, entry points and defines"
description: "Build variants: flavors, entry points, compile-time defines, build modes, obfuscation and extra Flutter arguments."
---

This guide covers everything that changes *what* is compiled: flavors, the entry file, `--dart-define` values, the build mode, obfuscation, and arguments passed straight to Flutter. Skip it if you build one plain variant.

## Flavors

A flavor is a named variant of your app. On Android it is a Gradle `productFlavor`; on iOS and macOS it is an Xcode scheme. You only need the flag if your project defines flavors.

```bash
fpack build apk aab --flavor prod
```

Also `build.flavor` in `fpack.yaml` and `FPACK_FLAVOR`.

- fpack reads the flavors from `android/app/build.gradle(.kts)` and the Xcode schemes (shared, per-user and workspace).
- A name that is not among the detected flavors is an error (exit code 3) with a suggestion; if no flavors are detected, it is only a warning.
- When the project defines flavors but none is selected, fpack stops *before* a full Gradle build would fail.
- Artifacts get the flavor in their name (`{-flavor}`, `{flavor}`).
- A flavor that exists only on Android is not passed to iOS or macOS builds when the Xcode projects define no custom schemes (with a note).
- Web builds do not use flavors.

## Entry point

```bash
fpack build apk -t lib/main_prod.dart       # --target FILE
```

`build.target` in `fpack.yaml`, `FPACK_ENTRY`. Combine with a flavor: `fpack build apk --flavor prod -t lib/main_prod.dart`.

## Compile-time variables

Pass values into the app with `--dart-define` (repeatable) or files:

```bash
fpack build apk --dart-define API_URL=https://api.example.com --dart-define DEBUG_MENU=false
fpack build apk --dart-define-from-file config/prod.json
fpack build apk --dart-define-from-file config/base.json --dart-define-from-file config/prod.env
```

```yaml
build:
  dart_define:
    API_URL: https://api.example.com
  dart_define_from_file: [config/prod.json]     # .json or .env files
```

The web target also has `web.web_define` for values in `web/index.html` ([Web](/platforms/web)).

::: warning
`--dart-define` values are compiled into the app and can be extracted. Do not put real secrets in them.
:::

## Build mode

```bash
fpack build apk --release     # the default; same as -m release / --mode release
fpack build apk --profile     # profile mode, for performance testing
fpack build apk --debug       # debug build
```

`build.mode`, `FPACK_MODE`. Non-release modes add `{mode}` to file names (`{-mode}`).

## Obfuscation and debug symbols

```bash
fpack build apk aab --obfuscate
fpack build apk --obfuscate --split-debug-info build/symbols
```

`--obfuscate` hides Dart symbol names. fpack stores the symbol files in `<output>/debug-info/<platform>` by default; `--split-debug-info DIR` chooses another folder. **Keep the symbols**: you need them to read stack traces from obfuscated releases. Settings: `build.obfuscate`, `build.split_debug_info`, `FPACK_OBFUSCATE`.

## Icons and other flutter options

```yaml
build:
  tree_shake_icons: false          # same as --no-tree-shake-icons
  extra_args: [--no-pub]           # added to every flutter build
```

Arguments after `--` on the command line go to `flutter build` as well:

```bash
fpack build apk -- --no-tree-shake-icons
```

Platform-specific extras live under each platform: `android.extra_args`, `ios.extra_args`, `macos.extra_args`, `windows.extra_args`, `linux.extra_args`, `web.extra_args` (and `windows.msix.extra_args`, `linux.appimage.extra_args`, `linux.snap.extra_args`).

## Which Flutter?

fpack finds the Flutter SDK in this order: `--flutter` → `FPACK_FLUTTER` → `flutter.sdk` in `fpack.yaml` → FVM (`.fvm/flutter_sdk`, `.fvmrc`) → `FLUTTER_ROOT` → `PATH` → common install locations (`~/develop/flutter`, `~/flutter`, …).

```bash
fpack build apk --flutter ~/sdks/flutter-3.24
```

## Next

[Secrets and variables](/guides/secrets).
