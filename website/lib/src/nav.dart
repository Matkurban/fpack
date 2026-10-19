import '../l10n/app_localizations.dart';

/// One documentation page in the navigation.
class NavPage {
  const NavPage(this.href, this.title);

  /// Site-relative URL, e.g. `/platforms/android`.
  final String href;

  /// The page title in the given locale (ARB strings, or a literal).
  final String Function(AppLocalizations t) title;
}

/// A titled group of pages in the sidebar.
class NavGroup {
  const NavGroup(this.title, this.pages);

  final String Function(AppLocalizations t)? title;
  final List<NavPage> pages;
}

/// The page map of the documentation: reading order from the first
/// introduction to the reference. The sidebar and the previous/next links are
/// both derived from it, so a page appears in exactly one place.
final List<NavGroup> navigation = [
  NavGroup((t) => t.navStart, [
    NavPage('/', (t) => t.navOverview),
    NavPage('/concepts', (t) => t.navConcepts),
    NavPage('/installation', (t) => t.navInstallation),
    NavPage('/getting-started', (t) => t.navGettingStarted),
  ]),
  NavGroup((t) => t.navPlatforms, [
    NavPage('/platforms/android', (_) => 'Android'),
    NavPage('/platforms/ios', (_) => 'iOS'),
    NavPage('/platforms/macos', (t) => t.navMacos),
    NavPage('/platforms/macos-credentials', (t) => t.navMacosCredentials),
    NavPage('/platforms/macos-profiles', (t) => t.navMacosProfiles),
    NavPage('/platforms/windows', (_) => 'Windows'),
    NavPage('/platforms/linux', (_) => 'Linux'),
    NavPage('/platforms/web', (_) => 'Web'),
  ]),
  NavGroup((t) => t.navGuides, [
    NavPage('/guides/project-and-output', (t) => t.navGuideProject),
    NavPage('/guides/build-variants', (t) => t.navGuideVariants),
    NavPage('/guides/secrets', (t) => t.navGuideSecrets),
    NavPage('/guides/hooks', (t) => t.navGuideHooks),
    NavPage('/guides/many-targets', (t) => t.navGuideMulti),
  ]),
  NavGroup((t) => t.navReference, [
    NavPage('/commands', (t) => t.navCommands),
    NavPage('/configuration', (t) => t.navConfiguration),
    NavPage('/environment', (t) => t.navEnvironment),
    NavPage('/targets', (t) => t.navTargets),
    NavPage('/exit-codes', (t) => t.navExitCodes),
    NavPage('/schema', (t) => t.navSchema),
  ]),
  NavGroup((t) => t.navMore, [
    NavPage('/ci', (t) => t.navCi),
    NavPage('/troubleshooting', (t) => t.navTroubleshooting),
    NavPage('/faq', (t) => t.navFaq),
    NavPage('/skills', (t) => t.navSkills),
    NavPage('/architecture', (t) => t.navArchitecture),
    NavPage('/changelog', (t) => t.navChangelog),
  ]),
];

/// Every page in reading order.
List<NavPage> get navigationPages => [for (final group in navigation) ...group.pages];

/// The pages before and after [href] in reading order (either may be null).
(NavPage?, NavPage?) neighbours(String href) {
  final pages = navigationPages;
  final i = pages.indexWhere((p) => p.href == href);
  if (i < 0) return (null, null);
  return (i > 0 ? pages[i - 1] : null, i + 1 < pages.length ? pages[i + 1] : null);
}
