import 'package:jaspr/dom.dart';
import 'package:jaspr/jaspr.dart';

import '../../l10n/strings.g.dart';

export '../../l10n/strings.g.dart' show AppLocale, Translations;

/// Locale metadata used by the site (content files, `<html lang>`, storage).
extension SiteLocale on AppLocale {
  /// The default locale: served without JavaScript and used for SEO.
  static const AppLocale fallback = AppLocale.en;

  /// BCP 47 tag for `lang` attributes.
  String get htmlLang => switch (this) {
    AppLocale.en => 'en',
    AppLocale.zh => 'zh-CN',
  };

  /// Suffix of this locale's content files: `page.md` (default) or `page.zh.md`.
  String get fileSuffix => this == fallback ? '' : '.$languageCode';

  /// The typed translations generated from `lib/l10n/app_<locale>.arb`.
  Translations get strings => _strings[this]!;
}

final Map<AppLocale, Translations> _strings = {
  for (final locale in AppLocale.values) locale: locale.buildSync(),
};

/// Attributes marking an element as belonging to [locale]. CSS hides elements
/// of every locale except the active one (`<html data-locale>`), see web/site.css.
Map<String, String> localeAttributes(AppLocale locale) => {
  'data-l10n': locale.languageCode,
  'lang': locale.htmlLang,
};

/// A UI string from the ARB files, rendered once per locale so that switching
/// the language needs no reload and never shows the wrong language.
class Localized extends StatelessComponent {
  const Localized(this.message, {super.key});

  final String Function(Translations t) message;

  @override
  Component build(BuildContext context) => Component.fragment([
    for (final locale in AppLocale.values)
      span(attributes: localeAttributes(locale), [Component.text(message(locale.strings))]),
  ]);
}

/// Renders [builder] once per locale, each copy wrapped in a block that is
/// only visible while its locale is active.
class LocalizedBlock extends StatelessComponent {
  const LocalizedBlock(this.builder, {super.key});

  final Component Function(AppLocale locale) builder;

  @override
  Component build(BuildContext context) => Component.fragment([
    for (final locale in AppLocale.values)
      div(classes: 'l10n-block', attributes: localeAttributes(locale), [builder(locale)]),
  ]);
}

/// `data-*` attributes carrying an attribute value for every locale, applied by
/// web/site.js when the language changes (e.g. `data-l10n-placeholder-zh`).
Map<String, String> localizedAttribute(String name, String Function(Translations t) message) => {
  for (final locale in AppLocale.values) 'data-l10n-$name-${locale.languageCode}': message(locale.strings),
};
