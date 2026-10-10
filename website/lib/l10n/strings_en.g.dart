///
/// Generated file. Do not edit.
///
// coverage:ignore-file
// ignore_for_file: type=lint, unused_import
// dart format off

part of 'strings.g.dart';

// Path: <root>
typedef TranslationsEn = Translations; // ignore: unused_element
class Translations with BaseTranslations<AppLocale, Translations> {
	/// You can call this constructor and build your own translation instance of this locale.
	/// Constructing via the enum [AppLocale.build] is preferred.
	Translations({Map<String, Node>? overrides, PluralResolver? cardinalResolver, PluralResolver? ordinalResolver, TranslationMetadata<AppLocale, Translations>? meta})
		: assert(overrides == null, 'Set "translation_overrides: true" in order to enable this feature.'),
		  _meta = meta ?? TranslationMetadata(
		    locale: AppLocale.en,
		    overrides: overrides ?? {},
		    cardinalResolver: cardinalResolver,
		    ordinalResolver: ordinalResolver,
		  ) {
		_meta.setFlatMapFunction(_flatMapFunction);
	}

	/// Metadata for the translations of <en>.
	final TranslationMetadata<AppLocale, Translations> _meta;
	@override TranslationMetadata<AppLocale, Translations> get $meta => _meta;

	/// Access flat map
	dynamic operator[](String key) => _meta.getTranslation(key);

	late final Translations _root = this; // ignore: unused_field

	Translations $copyWith({TranslationMetadata<AppLocale, Translations>? meta}) => Translations(meta: meta ?? this.$meta);

	// Translations

	/// en: 'Package Flutter apps for every platform with one command.'
	String get siteDescription => 'Package Flutter apps for every platform with one command.';

	/// en: 'Overview'
	String get navOverview => 'Overview';

	/// en: 'Getting started'
	String get navGettingStarted => 'Getting started';

	/// en: 'Installation'
	String get navInstallation => 'Installation';

	/// Sidebar group title
	///
	/// en: 'Reference'
	String get navReference => 'Reference';

	/// en: 'Commands'
	String get navCommands => 'Commands';

	/// en: 'Targets and artifacts'
	String get navTargets => 'Targets and artifacts';

	/// en: 'Configuration'
	String get navConfiguration => 'Configuration';

	/// en: 'Environment variables'
	String get navEnvironment => 'Environment variables';

	/// Sidebar group title
	///
	/// en: 'Platform guides'
	String get navPlatforms => 'Platform guides';

	/// en: 'macOS (signing, notarization)'
	String get navMacos => 'macOS (signing, notarization)';

	/// Sidebar group title
	///
	/// en: 'More'
	String get navMore => 'More';

	/// en: 'CI recipes'
	String get navCi => 'CI recipes';

	/// en: 'AI agent skill'
	String get navSkills => 'AI agent skill';

	/// en: 'Troubleshooting'
	String get navTroubleshooting => 'Troubleshooting';

	/// en: 'FAQ'
	String get navFaq => 'FAQ';

	/// en: 'Architecture'
	String get navArchitecture => 'Architecture';

	/// en: 'Changelog'
	String get navChangelog => 'Changelog';

	/// en: 'Search docs… (/)'
	String get searchPlaceholder => 'Search docs… (/)';

	/// en: 'Search'
	String get searchLabel => 'Search';

	/// en: 'No results'
	String get searchNoResults => 'No results';

	/// Label of the button that switches to the other language
	///
	/// en: '中文'
	String get languageSwitch => '中文';

	/// en: '切换到中文'
	String get languageSwitchTitle => '切换到中文';

	/// en: 'On this page'
	String get tocTitle => 'On this page';

	/// en: 'fpack is MIT licensed.'
	String get footerLicense => 'fpack is MIT licensed.';

	/// en: 'Edit this page on GitHub'
	String get footerEdit => 'Edit this page on GitHub';

	/// en: 'Page not found'
	String get notFoundTitle => 'Page not found';

	/// en: 'This page does not exist or has moved.'
	String get notFoundMessage => 'This page does not exist or has moved.';

	/// en: 'Go to the documentation home'
	String get notFoundHome => 'Go to the documentation home';
}

/// The flat map containing all translations for locale <en>.
/// Only for edge cases! For simple maps, use the map function of this library.
///
/// The Dart AOT compiler has issues with very large switch statements,
/// so the map is split into smaller functions (512 entries each).
extension on Translations {
	dynamic _flatMapFunction(String path) {
		return switch (path) {
			'siteDescription' => 'Package Flutter apps for every platform with one command.',
			'navOverview' => 'Overview',
			'navGettingStarted' => 'Getting started',
			'navInstallation' => 'Installation',
			'navReference' => 'Reference',
			'navCommands' => 'Commands',
			'navTargets' => 'Targets and artifacts',
			'navConfiguration' => 'Configuration',
			'navEnvironment' => 'Environment variables',
			'navPlatforms' => 'Platform guides',
			'navMacos' => 'macOS (signing, notarization)',
			'navMore' => 'More',
			'navCi' => 'CI recipes',
			'navSkills' => 'AI agent skill',
			'navTroubleshooting' => 'Troubleshooting',
			'navFaq' => 'FAQ',
			'navArchitecture' => 'Architecture',
			'navChangelog' => 'Changelog',
			'searchPlaceholder' => 'Search docs… (/)',
			'searchLabel' => 'Search',
			'searchNoResults' => 'No results',
			'languageSwitch' => '中文',
			'languageSwitchTitle' => '切换到中文',
			'tocTitle' => 'On this page',
			'footerLicense' => 'fpack is MIT licensed.',
			'footerEdit' => 'Edit this page on GitHub',
			'notFoundTitle' => 'Page not found',
			'notFoundMessage' => 'This page does not exist or has moved.',
			'notFoundHome' => 'Go to the documentation home',
			_ => null,
		};
	}
}
