---
title: "FAQ"
description: "Frequently asked questions."
---

**How do I read a failed build?** fpack shows the key error excerpt (Gradle's "What went wrong", Xcode error lines, …), one concrete fix, and the full log path `build/fpack/logs/…`. Add `-v` to stream the full output.

**A path dependency is missing** (e.g. `xue_hua_sdk: path: ../xue_hua_sdk`): fpack checks before building and names the missing package; put it at the expected location relative to the project.

**macOS says the developer cannot be verified**: fpack removes the quarantine attribute when it installs the core binary; if it is still blocked, run `FPACK_REBUILD=1 fpack --version` to build it with local Go.

**With a local path install every run starts with "Resolving dependencies…"**: that is pub's fixed behavior for path-activated packages (git / pub.dev installs don't do it). If another program consumes the `--json` output, install from git or pub.dev, or filter those lines.

**Will it change my project?** No. fpack only reads project files; `flutter_launcher_icons.yaml`, `flutter_native_splash.yaml`, `distribute_options.yaml`, `package_rename_config.yaml`, `pubspec.yaml` and the like are never modified. Consider adding `dist/` to `.gitignore`.
