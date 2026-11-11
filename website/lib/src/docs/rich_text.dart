import 'package:jaspr/dom.dart';
import 'package:jaspr/jaspr.dart';

/// Parses inline text formatting (bold, italic, code, links) into Jaspr components.
class RichText extends StatelessComponent {
  const RichText(this.text, {super.key});

  final String text;

  @override
  Component build(BuildContext context) {
    return Component.fragment(parseInline(text));
  }

  /// Parses inline markdown-like syntax into a list of components.
  static List<Component> parseInline(String input) {
    final components = <Component>[];

    // Regex matching:
    // 1: `code`
    // 2: **bold**
    // 3: *italic*
    // 4: [text](url)
    final pattern = RegExp(
      r'(`([^`]+)`)'
      r'|(\*\*([^*]+)\*\*)'
      r'|(\*([^*]+)\*)'
      r'|(\[([^\]]+)\]\(([^)]+)\))',
    );

    var lastIndex = 0;
    for (final match in pattern.allMatches(input)) {
      if (match.start > lastIndex) {
        components.add(Component.text(input.substring(lastIndex, match.start)));
      }

      if (match.group(2) != null) {
        // `code`
        components.add(code([Component.text(match.group(2)!)]));
      } else if (match.group(4) != null) {
        // **bold**
        components.add(strong([Component.text(match.group(4)!)]));
      } else if (match.group(6) != null) {
        // *italic*
        components.add(em([Component.text(match.group(6)!)]));
      } else if (match.group(8) != null && match.group(9) != null) {
        // [text](url)
        components.add(a(href: match.group(9)!, [Component.text(match.group(8)!)]));
      }

      lastIndex = match.end;
    }

    if (lastIndex < input.length) {
      components.add(Component.text(input.substring(lastIndex)));
    }

    return components;
  }
}

