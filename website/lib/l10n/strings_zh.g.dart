///
/// Generated file. Do not edit.
///
// coverage:ignore-file
// ignore_for_file: type=lint, unused_import
// dart format off

import 'package:intl/intl.dart';
import 'package:slang/generated.dart';
import 'strings.g.dart';

// Path: <root>
class TranslationsZh with BaseTranslations<AppLocale, Translations> implements Translations {
	/// You can call this constructor and build your own translation instance of this locale.
	/// Constructing via the enum [AppLocale.build] is preferred.
	TranslationsZh({Map<String, Node>? overrides, PluralResolver? cardinalResolver, PluralResolver? ordinalResolver, TranslationMetadata<AppLocale, Translations>? meta})
		: assert(overrides == null, 'Set "translation_overrides: true" in order to enable this feature.'),
		  _meta = meta ?? TranslationMetadata(
		    locale: AppLocale.zh,
		    overrides: overrides ?? {},
		    cardinalResolver: cardinalResolver,
		    ordinalResolver: ordinalResolver,
		  ) {
		_meta.setFlatMapFunction(_flatMapFunction);
	}

	/// Metadata for the translations of <zh>.
	final TranslationMetadata<AppLocale, Translations> _meta;
	@override TranslationMetadata<AppLocale, Translations> get $meta => _meta;

	/// Access flat map
	@override dynamic operator[](String key) => _meta.getTranslation(key);

	late final TranslationsZh _root = this; // ignore: unused_field

	@override 
	TranslationsZh $copyWith({TranslationMetadata<AppLocale, Translations>? meta}) => TranslationsZh(meta: meta ?? this.$meta);

	// Translations
	@override String get siteDescription => '一条命令，把 Flutter 应用打包成所有平台的发布文件。';
	@override String get navOverview => '概览';
	@override String get navGettingStarted => '快速开始';
	@override String get navInstallation => '安装';
	@override String get navReference => '参考';
	@override String get navCommands => '命令';
	@override String get navTargets => '目标与产物';
	@override String get navConfiguration => '配置';
	@override String get navEnvironment => '环境变量';
	@override String get navPlatforms => '平台指南';
	@override String get navMacos => 'macOS（签名、公证）';
	@override String get navMore => '更多';
	@override String get navCi => 'CI 示例';
	@override String get navSkills => 'AI 智能体技能';
	@override String get navTroubleshooting => '故障排查';
	@override String get navFaq => '常见问题';
	@override String get navArchitecture => '架构';
	@override String get navChangelog => '更新日志';
	@override String get searchPlaceholder => '搜索文档… (/)';
	@override String get searchLabel => '搜索';
	@override String get searchNoResults => '没有找到结果';
	@override String get languageSwitch => 'EN';
	@override String get languageSwitchTitle => 'Switch to English';
	@override String get tocTitle => '本页内容';
	@override String get footerLicense => 'fpack 基于 MIT 许可证开源。';
	@override String get footerEdit => '在 GitHub 上编辑此页';
	@override String get notFoundTitle => '页面不存在';
	@override String get notFoundMessage => '该页面不存在或已移动。';
	@override String get notFoundHome => '返回文档首页';
}

/// The flat map containing all translations for locale <zh>.
/// Only for edge cases! For simple maps, use the map function of this library.
///
/// The Dart AOT compiler has issues with very large switch statements,
/// so the map is split into smaller functions (512 entries each).
extension on TranslationsZh {
	dynamic _flatMapFunction(String path) {
		return switch (path) {
			'siteDescription' => '一条命令，把 Flutter 应用打包成所有平台的发布文件。',
			'navOverview' => '概览',
			'navGettingStarted' => '快速开始',
			'navInstallation' => '安装',
			'navReference' => '参考',
			'navCommands' => '命令',
			'navTargets' => '目标与产物',
			'navConfiguration' => '配置',
			'navEnvironment' => '环境变量',
			'navPlatforms' => '平台指南',
			'navMacos' => 'macOS（签名、公证）',
			'navMore' => '更多',
			'navCi' => 'CI 示例',
			'navSkills' => 'AI 智能体技能',
			'navTroubleshooting' => '故障排查',
			'navFaq' => '常见问题',
			'navArchitecture' => '架构',
			'navChangelog' => '更新日志',
			'searchPlaceholder' => '搜索文档… (/)',
			'searchLabel' => '搜索',
			'searchNoResults' => '没有找到结果',
			'languageSwitch' => 'EN',
			'languageSwitchTitle' => 'Switch to English',
			'tocTitle' => '本页内容',
			'footerLicense' => 'fpack 基于 MIT 许可证开源。',
			'footerEdit' => '在 GitHub 上编辑此页',
			'notFoundTitle' => '页面不存在',
			'notFoundMessage' => '该页面不存在或已移动。',
			'notFoundHome' => '返回文档首页',
			_ => null,
		};
	}
}
