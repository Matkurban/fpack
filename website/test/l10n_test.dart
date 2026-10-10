import 'dart:convert';
import 'dart:io';

import 'package:fpack_docs/src/content/localized_headings.dart';
import 'package:fpack_docs/src/content/localized_loader.dart';
import 'package:fpack_docs/src/l10n/locales.dart';
import 'package:path/path.dart' as p;
import 'package:test/test.dart';

Map<String, Object?> arb(AppLocale locale) =>
    jsonDecode(File('lib/l10n/app_${locale.languageCode}.arb').readAsStringSync()) as Map<String, Object?>;

Set<String> messageKeys(Map<String, Object?> arb) => arb.keys.where((k) => !k.startsWith('@')).toSet();

void main() {
  group('ARB files', () {
    final template = messageKeys(arb(SiteLocale.fallback));

    for (final locale in AppLocale.values) {
      test('app_${locale.languageCode}.arb has exactly the keys of the template', () {
        final data = arb(locale);
        expect(data['@@locale'], locale.languageCode);
        expect(messageKeys(data), template);
        for (final key in template) {
          expect(data[key], isA<String>().having((s) => s.trim(), 'trimmed', isNotEmpty), reason: key);
        }
      });
    }
  });

  group('content', () {
    final files = Directory('content')
        .listSync(recursive: true)
        .whereType<File>()
        .map((f) => p.posix.joinAll(p.split(p.relative(f.path, from: 'content'))))
        .where((f) => f.endsWith('.md'))
        .toSet();
    final pages = files.where((f) => pageLocale(f) == SiteLocale.fallback).toSet();

    test('there are pages', () => expect(pages, contains('index.md')));

    for (final locale in AppLocale.values) {
      test('every page exists in ${locale.languageCode}', () {
        final missing = [
          for (final page in pages)
            if (!files.contains(localizedPath(page, locale))) localizedPath(page, locale),
        ];
        expect(missing, isEmpty);
      });

      test('no ${locale.languageCode} file without a default page', () {
        final orphans = files.where(
          (f) => pageLocale(f) == locale && !pages.any((page) => localizedPath(page, locale) == f),
        );
        expect(orphans, isEmpty);
      });
    }

    test('every translation has a title', () {
      for (final file in files) {
        final (data, _) = splitFrontmatter(File('content/$file').readAsStringSync());
        expect(data['title'], isA<String>(), reason: file);
      }
    });

    test('no links to the old /zh/ pages', () {
      for (final file in files) {
        expect(File('content/$file').readAsStringSync(), isNot(contains('](/zh')), reason: file);
      }
    });
  });

  test('locale of a content file', () {
    expect(pageLocale('platforms/macos.md'), AppLocale.en);
    expect(pageLocale('platforms/macos.zh.md'), AppLocale.zh);
    expect(localizedPath('platforms/macos.md', AppLocale.zh), 'platforms/macos.zh.md');
    expect(localizedPath('index.md', AppLocale.en), 'index.md');
  });

  test('slugify keeps every script', () {
    expect(slugify('Output language'), 'output-language');
    expect(slugify('输出语言'), '输出语言');
    expect(slugify('对应 fpack.yaml 键的变量'), '对应-fpackyaml-键的变量');
    expect(slugify('`fpack build`'), 'fpack-build');
  });
}
