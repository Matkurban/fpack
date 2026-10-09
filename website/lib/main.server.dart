/// Static documentation site for fpack (English at /, Chinese at /zh/).
library;

import 'package:jaspr/server.dart';
import 'package:jaspr_content/components/callout.dart';
import 'package:jaspr_content/components/code_block.dart';
import 'package:jaspr_content/components/github_button.dart';
import 'package:jaspr_content/components/header.dart';
import 'package:jaspr_content/components/sidebar.dart';
import 'package:jaspr_content/components/theme_toggle.dart';
import 'package:jaspr_content/jaspr_content.dart';
import 'package:jaspr_content/theme.dart';

import 'components/site_extras.dart';
import 'main.server.options.dart';
import 'nav.dart';

void main() {
  Jaspr.initializeApp(options: defaultServerOptions);

  PageConfig config(String lang) => PageConfig(
    dataLoaders: [FilesystemDataLoader('content/_data')],
    parsers: [MarkdownParser()],
    extensions: [HeadingAnchorsExtension(), TableOfContentsExtension()],
    components: [Callout(), CodeBlock()],
    layouts: [
      DocsLayout(
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
        sidebar: Sidebar(groups: sidebarGroups(lang)),
      ),
    ],
    theme: ContentTheme(
      primary: ThemeColor(ThemeColors.sky.$600, dark: ThemeColors.sky.$400),
      background: ThemeColor(ThemeColors.slate.$50, dark: ThemeColors.zinc.$950),
      colors: [ContentColors.quoteBorders.apply(ThemeColors.sky.$400)],
    ),
  );

  final en = config('en');
  final zh = config('zh');

  runApp(
    ContentApp.custom(
      loaders: [FilesystemLoader('content')],
      configResolver: (source) => source.path.startsWith('zh/') || source.path == 'zh' ? zh : en,
    ),
  );
}
