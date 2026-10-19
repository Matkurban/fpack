import 'dart:io';

import 'package:fpack_docs/src/content/directives.dart';
import 'package:fpack_docs/src/l10n/locales.dart';
import 'package:fpack_docs/src/nav.dart';
import 'package:path/path.dart' as p;
import 'package:test/test.dart';

String contentFile(String href, String suffix) =>
    p.join('content', href == '/' ? 'index$suffix' : '${href.substring(1)}$suffix');

void main() {
  setUpAll(loadLocalizations);

  group('navigation', () {
    test('every page has English and Chinese content', () {
      for (final page in navigationPages) {
        expect(File(contentFile(page.href, '.md')).existsSync(), isTrue, reason: page.href);
        expect(File(contentFile(page.href, '.zh.md')).existsSync(), isTrue, reason: page.href);
      }
    });

    test('every content page is in the navigation exactly once', () {
      final hrefs = navigationPages.map((p) => p.href).toList();
      expect(hrefs.toSet().length, hrefs.length);
      final files = Directory('content')
          .listSync(recursive: true)
          .whereType<File>()
          .map((f) => p.relative(f.path, from: 'content').replaceAll(r'\', '/'))
          .where((f) => f.endsWith('.md') && !f.endsWith('.zh.md') && !f.startsWith('_'));
      for (final f in files) {
        final href = f == 'index.md' ? '/' : '/${f.substring(0, f.length - 3)}';
        expect(hrefs, contains(href), reason: '$f is not in lib/src/nav.dart');
      }
    });

    test('every page has a title in every locale', () {
      for (final locale in AppLocale.values) {
        for (final page in navigationPages) {
          expect(page.title(locale.strings), isNotEmpty);
        }
      }
    });

    test('previous and next follow the reading order', () {
      expect(neighbours('/').$1, isNull);
      expect(neighbours('/').$2!.href, '/concepts');
      expect(neighbours('/platforms/android').$1!.href, '/getting-started');
      expect(neighbours('/changelog').$2, isNull);
      expect(neighbours('/nope'), (null, null));
    });
  });

  group('directives', () {
    String title(String kind) => kind.toUpperCase();

    test('callouts', () {
      final html = expandDirectives('a\n\n::: warning Careful\nBody **bold**\n:::\n\nb', title);
      expect(html, contains('admonition admonition-warning'));
      expect(html, contains('Careful'));
      expect(html, contains('Body **bold**'));
      expect(html, isNot(contains(':::')));
    });

    test('default callout title', () {
      expect(expandDirectives('::: tip\nx\n:::', title), contains('TIP'));
    });

    test('tabs', () {
      final html = expandDirectives(
        '::: tabs\n@@ macOS\n```bash\nbrew x\n```\n@@ Windows\n```powershell\nwinget x\n```\n:::',
        title,
      );
      expect(html, contains('data-tabs'));
      expect(html, contains('data-tab="macOS"'));
      expect(html, contains('data-tab="Windows"'));
      expect(html, contains('brew x'));
    });

    test('fenced code is left alone', () {
      const md = '```text\n::: tip\n```';
      expect(expandDirectives(md, title), md);
    });
  });
}
