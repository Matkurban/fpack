/// UI strings of the docs site, the template of the ARB files (package:intl).
///
/// After changing a message here run `tool/l10n.sh`: it extracts
/// `lib/l10n/app_en.arb` and regenerates the lookup code in `lib/l10n/gen/`
/// from all ARB files. Translations live in `lib/l10n/app_<locale>.arb`.
library;

import 'package:intl/intl.dart';

import 'gen/messages_all.dart';

/// The supported locales, the first being the default.
const supportedLocales = ['en', 'zh'];

/// Loads the messages of every supported locale; call once before rendering.
Future<void> loadLocalizations() => Future.wait([for (final l in supportedLocales) initializeMessages(l)]);

/// Typed access to the messages of one [locale].
class AppLocalizations {
  const AppLocalizations(this.locale);

  final String locale;

  String get siteDescription => Intl.message(
    'Package Flutter apps for every platform with one command.',
    name: 'siteDescription',
    locale: locale,
  );

  String get navOverview => Intl.message('Overview', name: 'navOverview', locale: locale);

  String get navGettingStarted => Intl.message('Getting started', name: 'navGettingStarted', locale: locale);

  String get navInstallation => Intl.message('Installation', name: 'navInstallation', locale: locale);

  String get navReference =>
      Intl.message('Reference', name: 'navReference', desc: 'Sidebar group title', locale: locale);

  String get navCommands => Intl.message('Commands', name: 'navCommands', locale: locale);

  String get navTargets => Intl.message('Targets and artifacts', name: 'navTargets', locale: locale);

  String get navConfiguration => Intl.message('Configuration', name: 'navConfiguration', locale: locale);

  String get navEnvironment => Intl.message('Environment variables', name: 'navEnvironment', locale: locale);

  String get navPlatforms =>
      Intl.message('Platform guides', name: 'navPlatforms', desc: 'Sidebar group title', locale: locale);

  String get navMacos => Intl.message('macOS (signing, notarization)', name: 'navMacos', locale: locale);

  String get navMore => Intl.message('More', name: 'navMore', desc: 'Sidebar group title', locale: locale);

  String get navCi => Intl.message('CI recipes', name: 'navCi', locale: locale);

  String get navSkills => Intl.message('AI agent skill', name: 'navSkills', locale: locale);

  String get navTroubleshooting => Intl.message('Troubleshooting', name: 'navTroubleshooting', locale: locale);

  String get navFaq => Intl.message('FAQ', name: 'navFaq', locale: locale);

  String get navArchitecture => Intl.message('Architecture', name: 'navArchitecture', locale: locale);

  String get navChangelog => Intl.message('Changelog', name: 'navChangelog', locale: locale);

  String get searchPlaceholder => Intl.message('Search docs… (/)', name: 'searchPlaceholder', locale: locale);

  String get searchLabel => Intl.message('Search', name: 'searchLabel', locale: locale);

  String get searchNoResults => Intl.message('No results', name: 'searchNoResults', locale: locale);

  String get languageSwitch => Intl.message(
    '中文',
    name: 'languageSwitch',
    desc: 'Label of the button that switches to the other language',
    locale: locale,
  );

  String get languageSwitchTitle => Intl.message('切换到中文', name: 'languageSwitchTitle', locale: locale);

  String get tocTitle => Intl.message('On this page', name: 'tocTitle', locale: locale);

  String get footerLicense => Intl.message('fpack is MIT licensed.', name: 'footerLicense', locale: locale);

  String get footerEdit => Intl.message('Edit this page on GitHub', name: 'footerEdit', locale: locale);

  String get notFoundTitle => Intl.message('Page not found', name: 'notFoundTitle', locale: locale);

  String get notFoundMessage =>
      Intl.message('This page does not exist or has moved.', name: 'notFoundMessage', locale: locale);

  String get notFoundHome => Intl.message('Go to the documentation home', name: 'notFoundHome', locale: locale);
}
