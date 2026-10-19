/// Authoring helpers for the Markdown pages: `::: kind` blocks.
///
/// ```text
/// ::: tip Optional title
/// Text with **Markdown**.
/// :::
///
/// ::: tabs
/// @@ macOS / Linux
/// ...content of the first tab...
/// @@ Windows
/// ...content of the second tab...
/// :::
///
/// ::: cards
/// - [**Title**](/page) one line of text
/// :::
/// ```
///
/// Callout kinds are `tip`, `note`, `warning` and `danger`. Blocks may be
/// nested (a callout inside a tab); lines inside fenced code are left alone.
/// The output is plain HTML with blank lines around the Markdown, which the
/// Markdown parser renders as usual; web/site.js turns `data-tabs` blocks into
/// tab bars (without JavaScript every tab is shown under its label).
library;

/// Callout kinds and the strings of their default titles.
const calloutKinds = {'tip', 'note', 'warning', 'danger'};

/// Expands the directives of [markdown]; [title] gives the default title of a
/// callout kind in the page's language.
String expandDirectives(String markdown, String Function(String kind) title) {
  final lines = markdown.replaceAll('\r\n', '\n').split('\n');
  final out = <String>[];
  _expand(lines, 0, lines.length, out, title);
  return out.join('\n');
}

final _open = RegExp(r'^:::\s*([a-z]+)(?:\s+(.*\S))?\s*$');
final _close = RegExp(r'^:::\s*$');
final _fence = RegExp(r'^\s*(```|~~~)');

/// Index of the line closing the block that opened just before [start].
int _findClose(List<String> lines, int start, int end) {
  var depth = 1;
  var fenced = false;
  for (var i = start; i < end; i++) {
    final line = lines[i];
    if (_fence.hasMatch(line)) fenced = !fenced;
    if (fenced) continue;
    if (_open.hasMatch(line)) depth++;
    if (_close.hasMatch(line) && --depth == 0) return i;
  }
  throw FormatException('unterminated ::: block opened before line $start');
}

void _expand(List<String> lines, int start, int end, List<String> out, String Function(String) title) {
  var fenced = false;
  var i = start;
  while (i < end) {
    final line = lines[i];
    if (_fence.hasMatch(line)) fenced = !fenced;
    final m = fenced ? null : _open.firstMatch(line);
    if (m == null) {
      out.add(line);
      i++;
      continue;
    }
    final kind = m.group(1)!;
    final label = m.group(2);
    final close = _findClose(lines, i + 1, end);
    switch (kind) {
      case 'tabs':
        _tabs(lines, i + 1, close, out, title);
      case 'cards':
        out
          ..add('<div class="cards">')
          ..add('');
        _expand(lines, i + 1, close, out, title);
        out
          ..add('')
          ..add('</div>');
      default:
        if (!calloutKinds.contains(kind)) {
          throw FormatException('unknown ::: block "$kind" on line ${i + 1}');
        }
        out
          ..add('<div class="admonition admonition-$kind">')
          ..add('')
          ..add('<p class="admonition-title">${_escape(label ?? title(kind))}</p>')
          ..add('');
        _expand(lines, i + 1, close, out, title);
        out
          ..add('')
          ..add('</div>');
    }
    i = close + 1;
  }
}

void _tabs(List<String> lines, int start, int end, List<String> out, String Function(String) title) {
  final starts = <int>[];
  var fenced = false;
  var depth = 0;
  for (var i = start; i < end; i++) {
    final line = lines[i];
    if (_fence.hasMatch(line)) fenced = !fenced;
    if (fenced) continue;
    if (_open.hasMatch(line)) depth++;
    if (_close.hasMatch(line)) depth--;
    if (depth == 0 && line.startsWith('@@ ')) starts.add(i);
  }
  if (starts.isEmpty) throw FormatException('::: tabs needs at least one "@@ Label" line (line $start)');
  out
    ..add('<div class="tabs-group" data-tabs="">')
    ..add('');
  for (var n = 0; n < starts.length; n++) {
    final label = lines[starts[n]].substring(3).trim();
    out
      ..add('<div class="tab-panel" data-tab="${_escape(label)}">')
      ..add('')
      ..add('<p class="tab-label">${_escape(label)}</p>')
      ..add('');
    _expand(lines, starts[n] + 1, n + 1 < starts.length ? starts[n + 1] : end, out, title);
    out
      ..add('')
      ..add('</div>')
      ..add('');
  }
  out.add('</div>');
}

String _escape(String s) =>
    s.replaceAll('&', '&amp;').replaceAll('<', '&lt;').replaceAll('>', '&gt;').replaceAll('"', '&quot;');
