import 'package:jaspr/dom.dart';
import 'package:jaspr/jaspr.dart';

import '../content/localized_headings.dart';
import '../l10n/locales.dart';
import 'l10n.dart';
import 'rich_text.dart';

/// Base class for all structured document nodes.
///
/// Nodes are defined once and rendered for each supported [AppLocale].
abstract class DocNode {
  const DocNode();

  /// Renders this node for the given [locale].
  Component render(AppLocale locale);
}

/// Headings (H2, H3) that automatically generate TOC anchors.
class DocHeading extends DocNode {
  const DocHeading(this.text, {this.level = 2, this.customId});

  final L10n<String> text;
  final int level;
  final L10n<String>? customId;

  /// Computes the slug anchor id for [locale].
  String idFor(AppLocale locale) {
    if (customId != null) return customId!.forLocale(locale);
    return slugify(text.forLocale(locale));
  }

  @override
  Component render(AppLocale locale) {
    final title = text.forLocale(locale);
    final id = idFor(locale);
    final children = RichText.parseInline(title);

    return switch (level) {
      2 => h2(id: id, children),
      3 => h3(id: id, children),
      _ => h2(id: id, children),
    };
  }
}

/// A paragraph of text with rich inline formatting support.
class DocParagraph extends DocNode {
  const DocParagraph(this.text);

  final L10n<String> text;

  @override
  Component render(AppLocale locale) {
    return p(RichText.parseInline(text.forLocale(locale)));
  }
}

/// A code block with optional syntax highlighting and language tag.
class DocCode extends DocNode {
  const DocCode(this.codeText, {this.language = 'text'});

  final L10n<String> codeText;
  final String language;

  @override
  Component render(AppLocale locale) {
    final content = codeText.forLocale(locale);
    return pre([
      code(
        classes: language.isNotEmpty ? 'language-$language' : null,
        attributes: {'language': language},
        [Component.text(content)],
      ),
    ]);
  }
}

/// Table columns and rows defined once in pure Dart.
class DocTable extends DocNode {
  const DocTable({required this.headers, required this.rows});

  final List<L10n<String>> headers;
  final List<List<dynamic>> rows;

  @override
  Component render(AppLocale locale) {
    return table([
      thead([
        tr([
          for (final h in headers)
            th(RichText.parseInline(h.forLocale(locale))),
        ]),
      ]),
      tbody([
        for (final row in rows)
          tr([
            for (final cell in row)
              td(switch (cell) {
                L10n<String> l10n => RichText.parseInline(l10n.forLocale(locale)),
                String str => RichText.parseInline(str),
                Component comp => [comp],
                _ => [Component.text('$cell')],
              }),
          ]),
      ]),
    ]);
  }
}

/// Admonition / Callout box kinds.
enum CalloutKind {
  tip,
  note,
  warning,
  danger;

  String defaultTitle(AppLocale locale) => locale.strings.calloutTitle(name);
}

/// Callout block (tip, note, warning, danger).
class DocCallout extends DocNode {
  const DocCallout({
    required this.kind,
    required this.content,
    this.title,
  });

  const DocCallout.tip(this.content, {this.title}) : kind = CalloutKind.tip;
  const DocCallout.note(this.content, {this.title}) : kind = CalloutKind.note;
  const DocCallout.warning(this.content, {this.title}) : kind = CalloutKind.warning;
  const DocCallout.danger(this.content, {this.title}) : kind = CalloutKind.danger;

  final CalloutKind kind;
  final L10n<String>? title;
  final L10n<String> content;

  @override
  Component render(AppLocale locale) {
    final titleText = title?.forLocale(locale) ?? kind.defaultTitle(locale);
    return div(
      classes: 'admonition admonition-${kind.name}',
      [
        p(classes: 'admonition-title', [Component.text(titleText)]),
        p(RichText.parseInline(content.forLocale(locale))),
      ],
    );
  }
}

/// An individual tab in [DocTabs].
class DocTab {
  const DocTab({required this.label, required this.nodes});

  final String label;
  final List<DocNode> nodes;
}

/// Multi-OS or multi-tab selector.
class DocTabs extends DocNode {
  const DocTabs(this.tabs);

  final List<DocTab> tabs;

  @override
  Component render(AppLocale locale) {
    return div(
      attributes: {'data-tabs': ''},
      [
        for (final tab in tabs)
          div(
            attributes: {'data-tab': tab.label},
            [
              for (final node in tab.nodes) node.render(locale),
            ],
          ),
      ],
    );
  }
}

/// An ordered or unordered list.
class DocList extends DocNode {
  const DocList(this.items, {this.ordered = false});

  final List<L10n<String>> items;
  final bool ordered;

  @override
  Component render(AppLocale locale) {
    final listItems = [
      for (final item in items)
        li(RichText.parseInline(item.forLocale(locale))),
    ];
    return ordered ? ol(listItems) : ul(listItems);
  }
}

/// Custom escape hatch to render arbitrary Jaspr components.
class DocCustom extends DocNode {
  const DocCustom(this.builder);

  final Component Function(AppLocale locale) builder;

  @override
  Component render(AppLocale locale) => builder(locale);
}
