import 'package:jaspr/jaspr.dart';
import 'package:jaspr_content/components/sidebar.dart';

import '../l10n/locales.dart';

/// The sidebar: one definition, titles from the ARB files.
List<SidebarGroup> sidebarGroups(Translations t) => [
  SidebarGroup(
    links: [
      SidebarLink(text: t.navOverview, href: '/'),
      SidebarLink(text: t.navGettingStarted, href: '/getting-started'),
      SidebarLink(text: t.navInstallation, href: '/installation'),
    ],
  ),
  SidebarGroup(
    title: t.navReference,
    links: [
      SidebarLink(text: t.navCommands, href: '/commands'),
      SidebarLink(text: t.navTargets, href: '/targets'),
      SidebarLink(text: t.navConfiguration, href: '/configuration'),
      SidebarLink(text: t.navEnvironment, href: '/environment'),
    ],
  ),
  SidebarGroup(
    title: t.navPlatforms,
    links: [
      const SidebarLink(text: 'Android', href: '/platforms/android'),
      const SidebarLink(text: 'iOS', href: '/platforms/ios'),
      SidebarLink(text: t.navMacos, href: '/platforms/macos'),
      const SidebarLink(text: 'Windows', href: '/platforms/windows'),
      const SidebarLink(text: 'Linux', href: '/platforms/linux'),
      const SidebarLink(text: 'Web', href: '/platforms/web'),
    ],
  ),
  SidebarGroup(
    title: t.navMore,
    links: [
      SidebarLink(text: t.navCi, href: '/ci'),
      SidebarLink(text: t.navSkills, href: '/skills'),
      SidebarLink(text: t.navTroubleshooting, href: '/troubleshooting'),
      SidebarLink(text: t.navFaq, href: '/faq'),
      SidebarLink(text: t.navArchitecture, href: '/architecture'),
      SidebarLink(text: t.navChangelog, href: '/changelog'),
    ],
  ),
];

/// The sidebar in every locale (only the active one is visible).
class SiteSidebar extends StatelessComponent {
  const SiteSidebar({super.key});

  @override
  Component build(BuildContext context) => LocalizedBlock((locale) => Sidebar(groups: sidebarGroups(locale.strings)));
}
