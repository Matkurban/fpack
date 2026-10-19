import 'package:jaspr/jaspr.dart';
import 'package:jaspr_content/components/sidebar.dart';

import '../l10n/locales.dart';
import '../nav.dart';

/// The sidebar groups for one locale, derived from [navigation].
List<SidebarGroup> sidebarGroups(AppLocalizations t) => [
  for (final group in navigation)
    SidebarGroup(
      title: group.title?.call(t),
      links: [for (final page in group.pages) SidebarLink(text: page.title(t), href: page.href)],
    ),
];

/// The sidebar in every locale (only the active one is visible).
class SiteSidebar extends StatelessComponent {
  const SiteSidebar({super.key});

  @override
  Component build(BuildContext context) => LocalizedBlock((locale) => Sidebar(groups: sidebarGroups(locale.strings)));
}
