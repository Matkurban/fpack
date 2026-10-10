/// Static documentation site for fpack: one URL per page, every page in all
/// locales; the browser shows the visitor's language (see lib/src/l10n).
library;

import 'package:jaspr/server.dart';
import 'package:jaspr_content/components/callout.dart';
import 'package:jaspr_content/components/code_block.dart';
import 'package:jaspr_content/components/github_button.dart';
import 'package:jaspr_content/components/header.dart';
import 'package:jaspr_content/components/theme_toggle.dart';
import 'package:jaspr_content/jaspr_content.dart';
import 'package:jaspr_content/theme.dart';

import 'grammars.dart';
import 'main.server.options.dart';
import 'src/components/header_items.dart';
import 'src/components/not_found.dart';
import 'src/components/site_footer.dart';
import 'src/components/site_navigation.dart';
import 'src/content/localized_headings.dart';
import 'src/content/localized_loader.dart';
import 'src/l10n/locales.dart';
import 'src/layout/site_layout.dart';

Future<void> main() async {
  Jaspr.initializeApp(options: defaultServerOptions);
  await loadLocalizations();

  runApp(
    ContentApp.custom(
      loaders: [
        LocalizedContentLoader('content'),
        MemoryLoader(
          pages: [MemoryPage.builder(path: '404.md', builder: (_) => const NotFound())],
        ),
      ],
      configResolver: PageConfig.all(
        dataLoaders: [FilesystemDataLoader('content/_data')],
        parsers: [MarkdownParser()],
        extensions: [const LocalizedHeadingsExtension(), HeadingAnchorsExtension()],
        components: [
          Callout(),
          CodeBlock(defaultLanguage: 'text', grammars: docGrammars),
        ],
        layouts: [
          SiteLayout(
            header: Header(
              title: 'fpack',
              logo: '/images/logo.svg',
              items: [
                const SearchBox(),
                const LanguageSwitch(),
                ThemeToggle(),
                GitHubButton(repo: 'Matkurban/fpack'),
              ],
            ),
            sidebar: const SiteSidebar(),
            footer: const SiteFooter(),
          ),
        ],
        theme: ContentTheme(
          primary: ThemeColor(ThemeColors.sky.$600, dark: ThemeColors.sky.$400),
          background: ThemeColor(ThemeColors.slate.$50, dark: ThemeColors.zinc.$950),
          colors: [ContentColors.quoteBorders.apply(ThemeColors.sky.$400)],
        ),
      ),
    ),
  );
}
