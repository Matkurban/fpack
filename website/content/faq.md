---
title: "FAQ"
description: "Short answers to frequently asked questions."
---

## About fpack

**Will fpack change my project?** No. fpack only reads project files. `flutter_launcher_icons.yaml`, `flutter_native_splash.yaml`, `distribute_options.yaml`, `package_rename_config.yaml`, `pubspec.yaml` and Gradle/Xcode files are never modified. Everything it creates is in `dist/` and `build/fpack/`. Consider adding `dist/` to `.gitignore`.

**Do I need `fpack.yaml`?** No. Zero configuration works. `fpack init` helps when you want metadata, signing or custom names ([Your first build](/getting-started#6-create-fpackyaml-when-you-need-it)).

**How is it different from `flutter build`?** `flutter build` makes a build folder. fpack runs it, then signs, packages (DMG, pkg, installer, deb, rpm, AppImage, Snap, zip), notarizes, verifies, names, checksums and collects the files, for any number of targets in one command.

**Can I build iOS or Windows on Linux?** No: Flutter cannot cross-compile them. Use a CI runner of the right OS ([CI recipes](/ci)).

**Does fpack send my code or secrets anywhere?** No. It runs local tools. Notarization uploads the artifact to Apple, as `notarytool` does; the launcher downloads its core from the GitHub Release.

## Using it

**How do I read a failed build?** fpack shows the key error excerpt, one concrete fix and the full log path `build/fpack/logs/…`. Add `-v` to stream the full output ([Troubleshooting](/troubleshooting)).

**How do I build just one variant?** Name the target (`fpack build apk`) and add `--flavor` if the project has flavors ([Flavors](/guides/build-variants#flavors)).

**Where do I set the version?** In `pubspec.yaml` (`version: 1.2.0+5`) or per run with `--build-name` / `--build-number` ([Versions](/guides/project-and-output#versions)).

**Why does fpack refuse to build ("already exists")?** It never overwrites artifacts by default. Bump the version, use `--build-number`, or `--force`.

**How do I keep passwords out of `fpack.yaml`?** `${VAR}` references and `FPACK_*` variables ([Secrets and variables](/guides/secrets)).

**How do I use it in CI?** [CI recipes](/ci). Use `--json` and the exit codes for decisions.

**The Chinese output: how do I switch language?** `--lang zh|en` or `FPACK_LANG` ([Environment variables](/environment#output-language)).

## macOS

**macOS says the developer cannot be verified** when starting fpack: fpack removes the quarantine attribute when it installs its core; if it is still blocked, run `FPACK_REBUILD=1 fpack --version` to build it with local Go.

**Do I need a paid Apple account?** For Developer ID signing and notarization, yes ([certificates and credentials](/platforms/macos-credentials)). Unsigned builds for your own Mac need nothing.

## Installation

**With a local path install every run starts with "Resolving dependencies…"**: that is pub's fixed behavior for path-activated packages (git and pub.dev installs don't do it). If another program consumes the `--json` output, install from pub.dev or git, or filter those lines.

**A path dependency is missing** (for example `my_sdk: path: ../my_sdk`): fpack checks before building and names the missing package; put it at the expected location relative to the project.
