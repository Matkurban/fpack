import 'package:jaspr/dom.dart';
import 'package:jaspr/jaspr.dart';
import 'package:jaspr_content/jaspr_content.dart';

import '../content/localized_loader.dart';
import '../l10n/locales.dart';

const _repository = 'https://github.com/Matkurban/fpack';

/// Footer with an edit link to the page's file of the active locale.
class SiteFooter extends StatelessComponent {
  const SiteFooter({super.key});

  @override
  Component build(BuildContext context) {
    final path = context.page.path;
    return footer(classes: 'site-footer', [
      LocalizedBlock(
        (locale) => p([
          Component.text(locale.strings.footerLicense),
          Component.text(' · '),
          a(href: '$_repository/blob/main/website/content/${localizedPath(path, locale)}', [
            Component.text(locale.strings.footerEdit),
          ]),
        ]),
      ),
    ]);
  }
}
