---
title: "How fpack works"
description: "Targets, the build pipeline, where files go and what fpack never touches. Read this once; everything else builds on it."
---

fpack is a packaging tool, not a build system. It drives the tools you already use (`flutter`, Gradle, Xcode, signtool, `dpkg-deb`, …) in the right order, adds signing, notarization and packaging, and collects the results. This page explains the few ideas the rest of the documentation relies on.

## Targets and artifacts

A **target** is one kind of file you want to ship. You name targets on the command line:

```bash
fpack build apk aab       # two targets
fpack build macos dmg pkg # three targets
```

| Platform | Targets | What you get |
| --- | --- | --- |
| Android | `apk`, `aab` | APK (universal and/or per ABI), Android App Bundle |
| iOS | `ipa` | IPA for App Store Connect, ad hoc, enterprise, or unsigned |
| macOS | `macos`, `dmg`, `pkg` | zipped `.app`, disk image, installer package (signed, notarized) |
| Windows | `windows`, `exe`, `msix` | portable zip, Inno Setup installer, MSIX |
| Linux | `linux`, `deb`, `rpm`, `appimage`, `snap` | tar.gz, Debian package, RPM, AppImage, Snap |
| Web | `web` | zip of the deployable site |

An **artifact** is a file a target produced. Details of every target are on [Targets](/targets).

::: note
Flutter cannot cross-compile desktop and iOS apps. You build iOS and macOS on a Mac, Windows targets on Windows and Linux targets on Linux. Android and web build anywhere. [`fpack build --all`](/guides/many-targets) builds everything *this* machine can build and skips the rest with a reason; naming a target the machine cannot build is an error that points you to a CI runner of that OS.
:::

## The build pipeline

For every run fpack does the same steps, so failures happen early and are easy to place:

1. **Read** the project (`pubspec.yaml`, `android/app/build.gradle`, the Xcode project) and `fpack.yaml` if there is one. Invalid configuration stops here, with the line and a suggestion.
2. **Check** each target (the same checks as [`fpack doctor`](/commands#fpack-doctor)): tools, SDKs, certificates, keystore passwords, flavors. Targets that cannot be built are reported *before* anything slow starts.
3. **Build** with Flutter once per platform. `macos` + `dmg` + `pkg` share one `flutter build macos`; `linux` + `deb` + `rpm` + `appimage` + `snap` share one `flutter build linux`.
4. **Package and sign** each target: copy or repack, sign, notarize, verify.
5. **Checksum and summarize**: write `SHA256SUMS` next to the artifacts and print a table of what was produced.

`--dry-run` stops after step 2 and prints the exact commands steps 3 and 4 would run.

## Where files go

```text
my_app/
  dist/1.0.0+1/            artifacts and SHA256SUMS (output.dir)
  build/fpack/logs/        full log of every command
  build/fpack/secrets/     temporary key material, mode 0600, deleted after the build
  build/fpack/…            generated ExportOptions.plist, snapcraft.yaml, …
```

- **Artifacts** go to `dist/<version>+<build>/`. File names follow a template ([names and output](/guides/project-and-output)). Existing artifacts are never overwritten unless you pass `--force`: bump the version, pass `--build-number N`, or force it.
- **Everything else** fpack creates is under `build/fpack/` and can be deleted with [`fpack clean`](/commands#fpack-clean).
- **Your sources are read-only.** fpack does not edit Gradle files, Xcode projects, `pubspec.yaml`, icon or splash configs. Add `dist/` to `.gitignore`.

## Configuration: flags, variables, file

fpack needs no configuration. When you add some, it comes from four places; the first one that sets a value wins:

1. command-line flags (`--flavor prod`)
2. `FPACK_*` environment variables (`FPACK_FLAVOR=prod`)
3. `fpack.yaml` in the project root (`build.flavor: prod`)
4. defaults (read from your project, or fpack's own)

`fpack init` writes an `fpack.yaml` that lists every key, commented out, with explanations; uncomment what you need. Inside the file, `${VAR}` and `${VAR:-default}` read environment variables, which is how secrets stay out of the file ([Secrets and variables](/guides/secrets)).

## Safety nets

| You want to know | Run |
| --- | --- |
| What can this machine build, and what is missing? | `fpack doctor` (or `fpack list`) |
| Which commands will run? | `fpack build … --dry-run` |
| Why did it fail? | read the excerpt and fix fpack prints; the full log path is shown; add `-v` to stream tool output |
| What exactly was produced? | `dist/<version>/SHA256SUMS`, or `--json` |

## Vocabulary

| Term | Meaning |
| --- | --- |
| target | a kind of artifact: `apk`, `dmg`, `deb`, … |
| flavor | an Android productFlavor / Xcode scheme ([Flavors](/guides/build-variants)) |
| mode | `release` (default), `profile` or `debug` |
| launcher / core | the small Dart program you install, and the native binary it downloads and runs ([Installation](/installation#launcher)) |
| notarization | Apple's malware scan for macOS software distributed outside the App Store ([macOS](/platforms/macos)) |

## Next

Install fpack on [the next page](/installation), then follow [your first build](/getting-started).
