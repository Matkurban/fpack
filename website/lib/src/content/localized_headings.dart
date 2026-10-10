import 'package:jaspr_content/jaspr_content.dart';

import '../l10n/locales.dart';

/// A table-of-contents entry of one locale.
class TocItem {
  const TocItem(this.text, this.id, this.level);

  final String text;
  final String id;

  /// 0 for h2, 1 for h3.
  final int level;
}

/// Gives the headings of every locale block unique, readable ids and collects
/// one table of contents per locale (page data `tocs`).
///
/// The default locale keeps the ids of the Markdown parser. Other locales get
/// ids from their own heading text (Unicode letters are kept, so a Chinese
/// heading gets a Chinese anchor); an id already used by another locale gets
/// the locale as suffix (`fpack-build-zh`).
class LocalizedHeadingsExtension implements PageExtension {
  const LocalizedHeadingsExtension();

  static final _heading = RegExp(r'^h([1-3])$');

  @override
  Future<List<Node>> apply(Page page, List<Node> nodes) async {
    final used = <String>{};
    final tocs = <String, List<TocItem>>{for (final l in AppLocale.values) l.languageCode: []};

    Node visit(Node node, AppLocale locale) {
      if (node is! ElementNode) return node;
      final blockLocale = AppLocale.values.where((l) => l.languageCode == node.attributes['data-l10n']).firstOrNull;
      locale = blockLocale ?? locale;
      final level = _heading.firstMatch(node.tag)?.group(1);
      if (level == null) {
        return ElementNode(node.tag, node.attributes, [for (final c in node.children ?? <Node>[]) visit(c, locale)]);
      }
      final text = node.innerText.trim();
      var id = locale == SiteLocale.fallback ? node.attributes['id'] ?? slugify(text) : slugify(text);
      if (!used.add(id)) {
        id = uniqueId(id, locale, used);
      }
      if (level != '1') tocs[locale.languageCode]!.add(TocItem(text, id, int.parse(level) - 2));
      return ElementNode(node.tag, {...node.attributes, 'id': id}, node.children);
    }

    final result = [for (final node in nodes) visit(node, SiteLocale.fallback)];
    page.apply(data: {'tocs': tocs});
    return result;
  }
}

/// A lowercase, hyphenated anchor that keeps letters of every script.
String slugify(String text) {
  final slug = text
      .toLowerCase()
      .replaceAll(RegExp(r'[^\p{L}\p{N}\s_-]', unicode: true), '')
      .trim()
      .replaceAll(RegExp(r'[\s_]+'), '-');
  return slug.isEmpty ? 'section' : slug;
}

String uniqueId(String id, AppLocale locale, Set<String> used) {
  var candidate = '$id-${locale.languageCode}';
  for (var n = 2; !used.add(candidate); n++) {
    candidate = '$id-${locale.languageCode}-$n';
  }
  return candidate;
}
