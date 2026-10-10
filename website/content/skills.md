---
title: "AI agent skill"
description: "The fpack-cli package skill teaches AI coding agents how to use fpack."
---

fpack ships a [package skill](https://dart.dev/tools/pub/package-skills) named **`fpack-cli`** in its pub package (`skills/fpack-cli/SKILL.md`). It gives AI coding agents (Cursor, Claude Code, Gemini, Copilot, Cline, Codex…) concise, authoritative instructions for fpack: installation, `fpack init`, `doctor` and `list`, every build target, configuration keys and `FPACK_*` variables, Android/macOS/Windows signing, macOS notarization, CI and troubleshooting, with links back to this site.

## Install the skill in your project

Skills are installed from your project's dependencies, so add fpack as a dev dependency (it doesn't affect your app) and run the [`skills`](https://pub.dev/packages/skills) tool:

```sh
dart pub add dev:fpack
dart run skills@ get --package fpack          # pick your agent interactively
dart run skills@ get --package fpack --agent cursor   # or name it: claude, codex, copilot, cline, …
```

The skill is copied into your project's agent folder (for example `.agents/skills/fpack-cli/`); commit it if your team uses the same agent. Re-run the command after upgrading fpack: the skill is versioned with the package.

You still run fpack itself from the global install (`dart pub global activate fpack`).

## What agents do with it

When you ask an agent to “build a signed APK and AAB”, “notarize the DMG” or “add a release workflow”, the skill tells it to:

- use `fpack build <targets>` (with `--dry-run` first) instead of hand-written `flutter build` and packaging scripts;
- keep secrets in environment variables, never in `fpack.yaml`;
- use only real configuration keys (`fpack init`, `fpack schema`, [Configuration](/configuration));
- use `--json` and the exit codes in CI, and `fpack doctor` when prerequisites are missing.

Without installing anything, you can also point an agent at the file directly: [skills/fpack-cli/SKILL.md](https://github.com/Matkurban/fpack/blob/main/skills/fpack-cli/SKILL.md).
