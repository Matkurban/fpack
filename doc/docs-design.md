# fpack documentation: design

The documentation site (<https://matkurban.github.io/fpack/>, source in `website/`) is the single
home of the user documentation. This file records why it is organised the way it is, so changes keep
the structure instead of eroding it. Pages are written in English (`page.md`) and Chinese
(`page.zh.md`); both must always exist and cover the same content.

## Audience

| Reader | Wants | Starts at |
| --- | --- | --- |
| Flutter developer who has never packaged an app | "How do I get a file I can ship?" | Overview → How fpack works → Install → Your first build |
| Developer shipping to one store / platform | A complete walk-through for *their* platform, including signing | Platform tutorials |
| Maintainer tuning a pipeline | The right `fpack.yaml` keys for a task, with examples | Configuration guides |
| CI engineer | Copy-paste workflows, secrets, exit codes, JSON | CI recipes, Exit codes |
| Returning user | One exact flag / key / variable | Reference, search |

## Learning path (shallow → deep)

1. **Start here** – what fpack is and how it works (concepts), install, first build. No `fpack.yaml`
   needed; ends with a working artifact.
2. **Platform tutorials** – one end-to-end walk-through per platform (prerequisites → doctor →
   unsigned build → signing → release build → verify → publish → CI note → problems).
   macOS is split in three pages because signing and notarization are long: the tutorial, how to
   get certificates and credentials, and provisioning profiles.
3. **Configuration guides** – organised by *task*, not by key: names/versions/output, flavors and
   defines, secrets, hooks, many targets. Each guide shows the keys, the flags and the environment
   variable for the same setting and links to the reference.
4. **Reference** – complete and generated where possible: commands and flags (from `fpack help`), all
   `fpack.yaml` keys (key registry), environment variables, targets, exit codes and the JSON result,
   the JSON Schema.
5. **Help and project** – CI recipes, troubleshooting by symptom, FAQ, AI agent skill, architecture,
   changelog.

Every tutorial and guide ends with "Next" links; the site also shows previous/next links in reading
order (`website/lib/src/nav.dart` is the one place that defines the order).

## Page map

```text
/                         Overview: what, why, the learning path
/concepts                 How fpack works: targets, pipeline, precedence, what it never touches
/installation             Install, PATH, how the native core is found, requirements per target
/getting-started          Your first build
/platforms/android        apk / aab: keystore, signing, splits, verification, Google Play
/platforms/ios            ipa: team, export methods, ad hoc, unsigned, App Store Connect
/platforms/macos          zip / dmg / pkg: signing, notarization, waiting, DMG layout, pkg
/platforms/macos-credentials   Developer ID certificates, notary credentials, .p12 for CI
/platforms/macos-profiles Provisioning profiles, restricted entitlements, smoke test
/platforms/windows        zip / exe (Inno Setup) / msix, code signing
/platforms/linux          tar.gz / deb / rpm / AppImage / snap (destructive, lxd, multipass)
/platforms/web            web zip, base href, wasm
/guides/project-and-output     app metadata, versions, output directory, file names, checksums
/guides/build-variants    flavors, entry points, dart-define, modes, obfuscation, extra flutter args
/guides/secrets           precedence, ${VAR}, FPACK_* variables, redaction, CI secrets
/guides/hooks             pre_build / pre_package / post_package / post_build
/guides/many-targets      --all, shared builds, skipping, dry run, JSON, monorepos, clean
/commands                 every command and option (generated help)
/configuration            every fpack.yaml key (generated)
/environment              every variable (generated)
/targets                  every target, artifact name, aliases, tools
/exit-codes               exit codes (generated), JSON result
/schema                   the JSON Schema and editor completion
/ci  /troubleshooting  /faq  /skills  /architecture  /changelog
```

## Conventions

- **One idea per heading, headings are the table of contents.** Use `##` for steps and topics and
  `###` sparingly; the right-hand "On this page" list is built from them.
- **Tutorials use numbered steps** ("1. Check the machine"), each with the command, the expected
  output (real output, shortened with `…`) and what to do if it differs.
- **Callouts**: `::: tip`, `::: note`, `::: warning`, `::: danger` (see
  `website/lib/src/content/directives.dart`). Warnings are for things that cost time or money
  (store rejection, lost keys); notes for context.
- **Per-OS commands** use `::: tabs` with `@@ macOS`, `@@ Windows`, `@@ Linux` panels; the reader's
  OS is preselected and the choice is remembered.
- **Examples** are real: `fpack.yaml` snippets are parsed by the config parser in the tests, command
  lines are checked against the CLI. Use `my_app`, `com.example.app`, `ABCDE12345` as placeholders.
- **Secrets never appear.** Show `${FPACK_…}` references or `…`.
- **Generated blocks** (`<!-- BEGIN GENERATED … -->`) are never edited by hand:
  `FPACK_UPDATE=1 go test ./internal/cli -run 'TestWebsiteUpToDate|TestDocs'`.
- **Both languages move together.** Same headings in the same order; commands, keys and flags are not
  translated. UI strings live in `website/lib/l10n/*.arb` (`website/tool/l10n.sh`).
- **No release notes in docs and no docs in release notes**: documentation changes are not
  changelog entries.
- Links between pages are site-absolute (`/platforms/android#signing`); never to README sections.

## Coverage

`go/internal/cli/docs_coverage_test.go` is the checklist: every command, every build flag, every
target and alias, every public `FPACK_*` variable, every exit code and every `fpack.yaml` key must
appear on the documentation site, flags and targets in a tutorial or guide and not only in the
reference; and a table maps each feature (notarization, snap build modes, provisioning profiles,
smoke test, `ios.team_id`, flavors, …) to the page that explains it. Adding a feature to the code
without documenting it fails the test.

The other checks: `docs_test.go` (variables, keys, YAML examples and `fpack …` command lines are
real; generated blocks are current), `website/test/structure_test.dart` (every page is in the
navigation and exists in both languages; directives render), and `website/tool/check_links.py`,
run by the Docs workflow on the built site: every internal link and `#anchor` resolves, and an
anchor used inside a Chinese block exists in the Chinese block (English headings are hidden there).
