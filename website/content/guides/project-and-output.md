---
title: "Names, versions and output"
description: "Application metadata, version overrides, where artifacts go, how they are named, and checksums."
---

Everything in this guide is optional: fpack reads sensible defaults from your project. Use it when installers need proper names, when CI must set the version, or when you want a different folder or file naming scheme.

## Application metadata

Installers and packages need a human name, a publisher and similar details. fpack takes them from your project and lets you override them in the `app` section:

```yaml
app:
  display_name: My App                   # installer title, Start menu, .desktop Name=
  description: A fast and simple notes app
  publisher: Example Inc                 # Windows installer publisher, msix publisher display name, deb Maintainer fallback, rpm Vendor
  identifier: com.example.app            # Inno Setup AppId seed, msix identity name, pkg identifier fallback
  homepage: https://example.com
  support_url: https://example.com/support
  maintainer: Example Team <dev@example.com>   # deb Maintainer, rpm Packager
  copyright: © 2026 Example Inc
  license: MIT                           # SPDX id for rpm / deb
```

| Key | Used for | Default |
| --- | --- | --- |
| `app.name` | the `{app}` part of file names | pubspec `name` |
| `app.display_name` | installer title, Start menu, `.desktop` `Name=` | macOS `PRODUCT_NAME`, else pubspec `name` |
| `app.description` | deb Description, rpm Summary, `.desktop` Comment, msix description | pubspec `description` |
| `app.publisher` | Windows publisher, msix publisher display name, deb Maintainer fallback, rpm Vendor | `CompanyName` in `windows/runner/Runner.rc` |
| `app.identifier` | Inno Setup AppId seed, msix identity, pkg identifier fallback | Linux `APPLICATION_ID`, Android `applicationId` or iOS bundle id |
| `app.homepage`, `app.support_url` | installer URLs, deb Homepage, rpm URL | support URL falls back to the homepage |
| `app.maintainer` | deb Maintainer, rpm Packager | `app.publisher` |
| `app.copyright`, `app.license` | copyright line and license | `© <year> <publisher>`, `Proprietary` |

`fpack init` pre-fills these from your project.

## Versions

The version comes from `pubspec.yaml`: `version: 1.2.0+5` means build name `1.2.0` and build number `5`. Override either for one run, for example in CI:

```bash
fpack build apk --build-name 1.2.0 --build-number 42
```

Also `build.build_name` / `build.build_number` in `fpack.yaml` and `FPACK_BUILD_NAME` / `FPACK_BUILD_NUMBER`. The version appears in `dist/<version>+<build>/` and in file names as `{version}` and `{build}`.

```bash
fpack build apk --build-number "$GITHUB_RUN_NUMBER"
```

## Where artifacts go

By default `dist/{version}{+build}`, relative to the project. Change it with `-o` (`--output`):

```bash
fpack build apk -o release/android
```

or in `fpack.yaml` (`output.dir`, variable `FPACK_OUTPUT_DIR`). `output.dir` accepts the placeholders `{app}`, `{version}`, `{build}`, `{flavor}` and `{mode}`.

```yaml
output:
  dir: "dist/{app}/{version}"
```

## File names

Every artifact is named from a template, `output.name`:

```text
{app}{-flavor}-{version}{+build}-{platform}{-arch}{-variant}{-mode}
→ my_app-1.0.0+1-android-universal.apk
```

Override it for everything (`output.name`) or per target (`output.names`):

```yaml
output:
  name: "{app}-{version}-{platform}{-arch}"
  names:
    exe: "{app}-setup-{version}"
    web: "{app}-web"
```

| Placeholder | Value |
| --- | --- |
| `{app}` | `app.name` (default: pubspec `name`) |
| `{version}` / `{build}` | build name / build number |
| `{flavor}` | the flavor (empty where Flutter does not use it) |
| `{platform}` | `android`, `ios`, `macos`, `windows`, `linux`, `web` |
| `{arch}` | `arm64-v8a`, `universal`, `x64`, … |
| `{variant}` | `setup`, `portable`, … |
| `{mode}` | empty for release, otherwise `profile` or `debug` |
| `{target}` | the target name |
| `{date}` | build date `YYYYMMDD` |

`{-x}` and `{+x}` add the separator (`-` or `+`) only when the value is not empty, so release builds without a flavor have no stray dashes. In `output.dir` the per-artifact placeholders are empty.

## Overwriting

fpack never overwrites an existing artifact: building the same version twice stops with *already exists*. Bump the version, change the build number, or allow overwriting:

```bash
fpack build apk --force
```

(`output.overwrite: true`, `FPACK_OVERWRITE`). A dry run shows which files already exist.

## Checksums

Next to the artifacts fpack writes `SHA256SUMS`. Use SHA-512 or turn it off:

```yaml
output:
  checksums: true               # false: no checksum file
  checksum_algorithm: sha512    # writes SHA512SUMS
```

Verify with `shasum -a 256 -c SHA256SUMS` (macOS), `sha256sum -c SHA256SUMS` (Linux). With `--json` the same hashes are in the result ([Exit codes and JSON](/exit-codes#json)). After `fpack notarize finish` the file is rewritten for the stapled artifacts.

## Next

[Flavors, entry points and defines](/guides/build-variants), or look up every key in the [reference](/configuration#app).
