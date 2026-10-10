# fpack documentation website

The site at https://matkurban.github.io/fpack/, built with [Jaspr](https://jaspr.site) and `jaspr_content`.

## Localization

There is one URL per page; every page contains all locales and the browser shows one:
`?lang=en|zh` (saved), the saved choice (`localStorage`), then the browser language
(`zh*` → Chinese, otherwise English). An inline script in `<head>` sets
`<html data-locale>` before the first paint and `web/site.css` hides the other locale,
so there is no flash of the wrong language. Without JavaScript the English page is shown.
The old `/zh/<page>` URLs redirect to `/<page>?lang=zh`.

- **UI strings** (sidebar, search, language switch, table of contents, footer, 404):
  `lib/l10n/app_en.arb` (template) and `lib/l10n/app_zh.arb`. Typed accessors
  (`lib/l10n/strings*.g.dart`) are generated with [slang](https://pub.dev/packages/slang):
  run `dart run slang` after editing an ARB file.
- **Pages**: `content/<page>.md` (English) and `content/<page>.zh.md` (Chinese), one
  route per page (`lib/src/content/localized_loader.dart`). The reference parts of
  `commands`, `configuration`, `environment` and `changelog` are generated from the CLI:
  `cd ../go && FPACK_UPDATE=1 go test ./internal/cli -run TestWebsiteUpToDate`.
- **Navigation**: `lib/src/components/site_navigation.dart` (one definition, ARB titles).
- **Search**: `tool/postbuild.py` writes one index entry per locale and section; `web/site.js`
  searches the active locale.

`dart test` checks that both ARB files have the same keys and that every page exists in
both languages.

## Develop

```sh
dart pub get
jaspr serve            # http://localhost:8080
jaspr build && python3 tool/postbuild.py --base /fpack/ --site-url https://matkurban.github.io/fpack/
```
