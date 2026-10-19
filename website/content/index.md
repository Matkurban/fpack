---
title: "fpack"
description: "One command turns a Flutter project into signed release files for every platform."
---

<div class="hero-badges">

[![pub package](https://img.shields.io/pub/v/fpack.svg)](https://pub.dev/packages/fpack) [![CI](https://github.com/Matkurban/fpack/actions/workflows/ci.yml/badge.svg)](https://github.com/Matkurban/fpack/actions/workflows/ci.yml) [![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](https://github.com/Matkurban/fpack/blob/main/LICENSE)

</div>

`flutter build` gives you a build folder. **fpack gives you the files you ship**: a signed APK and AAB, an IPA, a notarized DMG, a Windows installer, `.deb` / `.rpm` / AppImage / Snap packages and a web zip, each named consistently, checksummed and collected in one `dist/` folder.

```bash
dart pub global activate fpack
cd my_app
fpack doctor              # what can this machine build? what is missing, and how do I install it?
fpack build apk aab       # → dist/1.0.0+1/my_app-1.0.0+1-android-universal.apk, … .aab
fpack build --all         # everything this machine can build; the rest is skipped with a reason
```

## Start here

New to fpack? Read these four pages in order. About twenty minutes, and you end with a real artifact.

::: cards
- [**1 · How fpack works**](/concepts) targets, the build pipeline, where files go, what fpack never touches
- [**2 · Install**](/installation) one command, and what to do when `fpack` is not found
- [**3 · Your first build**](/getting-started) `doctor`, a dry run, a real build, reading the result
- [**4 · Pick your platform**](#platform-tutorials) a complete walk-through for each platform
:::

## Platform tutorials

Each tutorial goes from prerequisites to a signed, verified file you can publish.

::: cards
- [**Android**](/platforms/android) keystore, signed APK / AAB, per-ABI splits, Google Play
- [**iOS**](/platforms/ios) team, export methods, ad hoc, unsigned, App Store Connect
- [**macOS**](/platforms/macos) zip / DMG / pkg, Developer ID signing, notarization
- [**Windows**](/platforms/windows) portable zip, Inno Setup installer, MSIX, code signing
- [**Linux**](/platforms/linux) tar.gz, deb, rpm, AppImage, Snap (also in LXD / Multipass)
- [**Web**](/platforms/web) a deployable zip, base href, WebAssembly
:::

## Configure it for your project

fpack works with no configuration. When you need more, the guides are organised by *what you want to do*:

::: cards
- [**Names, versions and output**](/guides/project-and-output) app metadata, version overrides, file names, checksums
- [**Flavors, entry points and defines**](/guides/build-variants) `--flavor`, `-t`, `--dart-define`, obfuscation
- [**Secrets and variables**](/guides/secrets) keep passwords out of files; precedence of flags, variables and file
- [**Hooks**](/guides/hooks) run your scripts before and after a build or package
- [**Many targets, one command**](/guides/many-targets) `--all`, dry runs, JSON, monorepos
:::

## Look something up

::: cards
- [**Commands**](/commands) every command and option
- [**fpack.yaml reference**](/configuration) every key, with type, default and example
- [**Environment variables**](/environment) every `FPACK_*` variable
- [**Targets**](/targets) names, aliases, artifacts, required tools
- [**Exit codes and JSON**](/exit-codes) for scripts and CI
- [**JSON Schema**](/schema) editor completion for `fpack.yaml`
- [**CI recipes**](/ci) GitHub Actions for every platform
- [**Troubleshooting**](/troubleshooting) by symptom
:::

## Why fpack

- **Zero configuration.** It reads `pubspec.yaml`, Gradle and the Xcode projects. `fpack.yaml` is optional.
- **It never edits your project.** Signing data is injected through environment variables and temporary files; artifacts go to `dist/`, scratch files to `build/fpack/`.
- **Safe to try.** `fpack doctor` tells you what is missing and how to install it; `--dry-run` prints every command it would run.
- **Signing and notarization included.** Android keystores (v1–v4), iOS export options, macOS Developer ID with resumable notarization, Windows Authenticode (pfx, certificate store, Azure Trusted Signing).
- **Everything is configurable.** 216 validated `fpack.yaml` keys, a JSON Schema for editor completion, `${VAR}` interpolation and dedicated `FPACK_*` variables.
- **Built for CI.** Stable exit codes, `--json` results, secrets redacted from logs, English and Chinese output.
- **Fast and dependency-free.** A native core (about 10 ms startup) behind a thin Dart launcher that downloads the matching, SHA-256 verified binary.

A complete example project with a commented `fpack.yaml` lives in [`example/`](https://github.com/Matkurban/fpack/tree/main/example).
