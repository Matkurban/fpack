import 'package:jaspr/dom.dart';
import 'package:jaspr/jaspr.dart';

import '../l10n/locales.dart';

/// Toggles the language (web/site.js); shows the name of the other language.
class LanguageSwitch extends StatelessComponent {
  const LanguageSwitch({super.key});

  @override
  Component build(BuildContext context) => button(
    classes: 'lang-switch',
    attributes: {
      'type': 'button',
      'data-language-switch': '',
      'title': SiteLocale.fallback.strings.languageSwitchTitle,
      ...localizedAttribute('title', (t) => t.languageSwitchTitle),
    },
    [Localized((t) => t.languageSwitch)],
  );
}

/// Search box; behavior and the per-locale index come from web/site.js and
/// search-index.json (generated after the static build by tool/postbuild.py).
class SearchBox extends StatelessComponent {
  const SearchBox({super.key});

  @override
  Component build(BuildContext context) {
    final t = SiteLocale.fallback.strings;
    return div(classes: 'search', [
      input(
        type: InputType.search,
        attributes: {
          'id': 'search-input',
          'placeholder': t.searchPlaceholder,
          'aria-label': t.searchLabel,
          'autocomplete': 'off',
          ...localizedAttribute('placeholder', (t) => t.searchPlaceholder),
          ...localizedAttribute('aria-label', (t) => t.searchLabel),
          ...localizedAttribute('empty', (t) => t.searchNoResults),
        },
      ),
      div(classes: 'search-results', attributes: {'id': 'search-results', 'role': 'listbox'}, []),
    ]);
  }
}
