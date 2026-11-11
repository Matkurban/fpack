import '../l10n/locales.dart';

/// A multilingual value holding translations for all supported locales.
class L10n<T> {
  const L10n(this.values);

  /// Convenient constructor for bilingual (English + Chinese) values.
  L10n.bilingual({required T en, required T zh})
      : values = {
          AppLocale.en: en,
          AppLocale.zh: zh,
        };

  final Map<AppLocale, T> values;

  /// English value (fallback).
  T get en => values[AppLocale.en]!;

  /// Chinese value.
  T get zh => values[AppLocale.zh] ?? en;

  /// Retrieves the value for the given [locale].
  T forLocale(AppLocale locale) => values[locale] ?? en;
}

/// Helper function to define bilingual text concisely.
L10n<String> t({required String en, required String zh}) =>
    L10n.bilingual(en: en, zh: zh);

/// Helper for text that is identical in all languages (e.g. commands, target names).
L10n<String> literal(String text) =>
    L10n({for (final l in AppLocale.values) l: text});
