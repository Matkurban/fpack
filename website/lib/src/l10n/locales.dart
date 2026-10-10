import 'package:jaspr/dom.dart';
import 'package:jaspr/jaspr.dart';

import '../../l10n/app_localizations.dart';

export '../../l10n/app_localizations.dart' show AppLocalizations, loadLocalizations;

/// The site's locales (see `supportedLocales` and the ARB files in lib/l10n).
enum AppLocale {
  en,
  zh;

  String get languageCode => name;
}

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

  /// The UI strings of this locale (`lib/l10n/app_<locale>.arb`).
  AppLocalizations get strings => AppLocalizations(languageCode);
}

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

  final String Function(AppLocalizations t) message;

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
Map<String, String> localizedAttribute(String name, String Function(AppLocalizations t) message) => {
  for (final locale in AppLocale.values) 'data-l10n-$name-${locale.languageCode}': message(locale.strings),
};
