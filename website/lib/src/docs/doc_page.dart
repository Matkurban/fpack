import 'package:jaspr/dom.dart';
import 'package:jaspr/jaspr.dart';
import 'package:jaspr_content/jaspr_content.dart';

import '../content/localized_headings.dart';
import '../l10n/locales.dart';
import 'doc_node.dart';
import 'l10n.dart';

/// Base class for all pure Dart documentation pages.
///
/// A single [DocPage] defines both English and Chinese content using
/// [L10n] without duplicating structural code.
abstract class DocPage {
  const DocPage();

  /// URL path of the page (without leading slash or extension, e.g. 'concepts' or 'index').
  String get path;

  /// Page title in all supported languages.
  L10n<String> get title;

  /// Page description for metadata/SEO in all supported languages.
  L10n<String> get description;

  /// The list of structured document nodes.
  List<DocNode> get nodes;

  /// Extracts the Table of Contents for [locale].
  List<TocItem> toc(AppLocale locale) {
    final items = <TocItem>[];
    for (final node in nodes) {
      if (node is DocHeading) {
        items.add(TocItem(
          node.text.forLocale(locale),
          node.idFor(locale),
          node.level - 2, // H2 -> 0, H3 -> 1
        ));
      }
    }
    return items;
  }

  /// Converts this [DocPage] into a Jaspr Content [MemoryPage].
  MemoryPage toMemoryPage() {
    final pagePath = switch (path) {
      '' || '/' => 'index.html',
      final p when p.endsWith('.html') || p.endsWith('.md') => p,
      final p => '$p.html',
    };

    return MemoryPage.builder(
      path: pagePath,
      initialData: {
        'page': <String, Object?>{
          'title': title.en,
          'description': description.en,
          'locales': {
            for (final l in AppLocale.values)
              l.languageCode: {
                'title': title.forLocale(l),
                'description': description.forLocale(l),
              },
          },
          'tocs': {
            for (final l in AppLocale.values)
              l.languageCode: toc(l),
          },
        },
      },
      builder: (context) => Component.fragment([
        for (final locale in AppLocale.values)
          div(
            classes: 'l10n-block',
            attributes: localeAttributes(locale),
            [
              for (final node in nodes) node.render(locale),
            ],
          ),
      ]),
    );
  }
}

