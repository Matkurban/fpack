---
title: "JSON Schema"
description: "Editor completion, hover documentation and validation for fpack.yaml."
---

fpack publishes a JSON Schema for `fpack.yaml`, generated from the same key registry that drives validation, `fpack init` and these docs. With it, your editor completes keys, shows each key's documentation (in English and Chinese) and underlines mistakes while you type.

## Automatic

`fpack init` writes this as the first line of `fpack.yaml`:

```text
# yaml-language-server: $schema=https://raw.githubusercontent.com/Matkurban/fpack/main/schema/fpack.schema.json
```

- **VS Code**: install the [YAML extension](https://marketplace.visualstudio.com/items?itemName=redhat.vscode-yaml). It reads the line above.
- **JetBrains IDEs** (IntelliJ, Android Studio): the YAML support honors the same line (the *Json Schema* settings can also map the file manually).

## Offline copy

```bash
fpack schema -o .vscode/fpack.schema.json
```

Point your editor at the local file, for example in VS Code `settings.json`:

```json
{
  "yaml.schemas": { ".vscode/fpack.schema.json": "fpack.yaml" }
}
```

`fpack schema` without `-o` prints to stdout, so you can pipe it anywhere.

## What it validates

- the key names (nothing unknown), nested sections, types, allowed values and numeric ranges;
- hover text with the description, default, example and the matching `FPACK_*` variable or flag.

fpack validates the file itself on every run with the same rules, and adds checks a schema cannot express (a signing password without a key file, for example). See [the key reference](/configuration).
