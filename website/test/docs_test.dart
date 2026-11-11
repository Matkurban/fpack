import 'package:fpack_docs/src/docs/doc_node.dart';
import 'package:fpack_docs/src/docs/l10n.dart';
import 'package:fpack_docs/src/docs/pages/concepts_doc.dart';
import 'package:fpack_docs/src/docs/rich_text.dart';
import 'package:fpack_docs/src/l10n/locales.dart';
import 'package:test/test.dart';

void main() {
  group('Pure Dart Docs', () {
    test('L10n returns correct language values', () {
      final val = t(en: 'Hello', zh: '你好');
      expect(val.forLocale(AppLocale.en), 'Hello');
      expect(val.forLocale(AppLocale.zh), '你好');
      expect(val.en, 'Hello');
      expect(val.zh, '你好');
    });

    test('RichText parser handles inline markdown tokens', () {
      final tokens = RichText.parseInline('A **target** with `code` and [link](/test).');
      expect(tokens.length, 7);
    });

    test('ConceptsDoc generates metadata and TOC for both locales', () {
      const doc = ConceptsDoc();
      expect(doc.title.en, 'How fpack works');
      expect(doc.title.zh, 'fpack 如何工作');

      final tocEn = doc.toc(AppLocale.en);
      final tocZh = doc.toc(AppLocale.zh);

      expect(tocEn.length, tocZh.length);
      expect(tocEn.map((t) => t.text), contains('Targets and artifacts'));
      expect(tocZh.map((t) => t.text), contains('目标与产物'));
      expect(tocEn.map((t) => t.id), contains('targets-and-artifacts'));
      expect(tocZh.map((t) => t.id), contains('目标与产物'));
    });

    test('ConceptsDoc converts to Jaspr MemoryPage', () {
      const doc = ConceptsDoc();
      final memoryPage = doc.toMemoryPage();

      expect(memoryPage.path, 'concepts.html');
      final initialData = memoryPage.initialData['page'] as Map<String, Object?>;
      expect(initialData['title'], 'How fpack works');

      final locales = initialData['locales'] as Map<String, Object?>;
      expect(locales.containsKey('en'), isTrue);
      expect(locales.containsKey('zh'), isTrue);

      final tocs = initialData['tocs'] as Map<String, Object?>;
      expect(tocs.containsKey('en'), isTrue);
      expect(tocs.containsKey('zh'), isTrue);
    });

    test('All DocHeadings have both English and Chinese text', () {
      const doc = ConceptsDoc();
      for (final node in doc.nodes) {
        if (node is DocHeading) {
          expect(node.text.en.trim(), isNotEmpty);
          expect(node.text.zh.trim(), isNotEmpty);
          expect(node.idFor(AppLocale.en), isNotEmpty);
          expect(node.idFor(AppLocale.zh), isNotEmpty);
        }
      }
    });
  });
}
