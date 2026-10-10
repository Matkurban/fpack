import 'dart:convert';

import 'package:file/file.dart';
import 'package:file/local.dart';
import 'package:jaspr_content/jaspr_content.dart';
import 'package:path/path.dart' as p;
import 'package:yaml/yaml.dart';

import '../l10n/locales.dart';

/// Loads one route per page from [directory], merging the page's translations.
///
/// A page `foo` is `foo.md` (English, required) plus `foo.<locale>.md` for every
/// other locale (e.g. `foo.zh.md`). All translations are rendered into the same
/// page, each inside a `data-l10n` block; the browser shows the active one.
/// The English front matter becomes the page data (used for `<title>`, SEO),
/// the localized titles and descriptions are available under `locales`.
class LocalizedContentLoader extends RouteLoaderBase<LocalizedPageSource> {
  LocalizedContentLoader(this.directory, {this.fileSystem = const LocalFileSystem()});

  final String directory;
  final FileSystem fileSystem;

  @override
  Future<List<LocalizedPageSource>> loadPageSources() async {
    final root = fileSystem.directory(directory);
    final files = root
        .listSync(recursive: true)
        .whereType<File>()
        .map((f) => p.posix.joinAll(p.split(p.relative(f.path, from: directory))))
        .where((path) => path.endsWith('.md'))
        .toSet();
    return [
      for (final path in files.where((f) => pageLocale(f) == SiteLocale.fallback).toList()..sort())
        LocalizedPageSource(path, this, {
          for (final locale in AppLocale.values) locale: root.childFile(localizedPath(path, locale)),
        }),
    ];
  }

  @override
  Future<String> readPartial(String path, Page page) => fileSystem.file(path).readAsString();

  @override
  String readPartialSync(String path, Page page) => fileSystem.file(path).readAsStringSync();
}

/// The locale of a content file: `foo.zh.md` → zh, anything else → the default.
AppLocale pageLocale(String path) {
  final name = p.posix.basenameWithoutExtension(path);
  return AppLocale.values.firstWhere(
    (l) => l != SiteLocale.fallback && name.endsWith(l.fileSuffix),
    orElse: () => SiteLocale.fallback,
  );
}

/// The file of [path] (a default-locale page) for [locale].
String localizedPath(String path, AppLocale locale) => '${path.substring(0, path.length - 3)}${locale.fileSuffix}.md';

class LocalizedPageSource extends PageSource {
  LocalizedPageSource(super.path, super.loader, this.files);

  /// The content file of every locale.
  final Map<AppLocale, File> files;

  @override
  Future<Page> buildPage() async {
    final locales = <String, Object?>{};
    final body = StringBuffer();
    for (final MapEntry(key: locale, value: file) in files.entries) {
      if (!await file.exists()) {
        throw StateError('${file.path} is missing: every page needs a translation for each locale');
      }
      final (data, markdown) = splitFrontmatter(await file.readAsString());
      locales[locale.languageCode] = data;
      body
        ..writeln('<div class="l10n-block" data-l10n="${locale.languageCode}" lang="${locale.htmlLang}">')
        ..writeln()
        ..writeln(markdown.trim())
        ..writeln()
        ..writeln('</div>');
    }
    final data = {...?locales[SiteLocale.fallback.languageCode] as Map?, 'locales': locales};
    // JSON is valid YAML, so the merged data goes back through the regular
    // front matter parsing of jaspr_content.
    return Page(
      path: path,
      url: url,
      content: '---\n${jsonEncode(data)}\n---\n$body',
      config: config,
      loader: loader,
    );
  }
}

/// Splits a Markdown file into its YAML front matter and body.
(Map<String, Object?>, String) splitFrontmatter(String source) {
  final match = RegExp(r'^---\r?\n(.*?)\r?\n---\r?\n', dotAll: true).firstMatch(source);
  if (match == null) return (const {}, source);
  final yaml = loadYaml(match.group(1)!);
  final data = yaml is Map ? jsonDecode(jsonEncode(yaml)) as Map<String, Object?> : <String, Object?>{};
  return (data, source.substring(match.end));
}
