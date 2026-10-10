import 'package:jaspr/dom.dart';
import 'package:jaspr/jaspr.dart';

import '../l10n/locales.dart';

/// Body of 404.html.
class NotFound extends StatelessComponent {
  const NotFound({super.key});

  @override
  Component build(BuildContext context) => div(classes: 'not-found', [
    LocalizedBlock(
      (locale) => div([
        h1([Component.text(locale.strings.notFoundTitle)]),
        p([Component.text(locale.strings.notFoundMessage)]),
        p([
          a(href: '/', [Component.text(locale.strings.notFoundHome)]),
        ]),
      ]),
    ),
  ]);
}
