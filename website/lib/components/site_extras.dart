import 'package:jaspr/dom.dart';
import 'package:jaspr/jaspr.dart';
import 'package:jaspr_content/jaspr_content.dart';

/// Link to the same page in the other language.
class LanguageSwitch extends StatelessComponent {
  const LanguageSwitch({super.key});

  @override
  Component build(BuildContext context) {
    final url = context.page.url;
    final isZh = url == '/zh' || url.startsWith('/zh/');
    final other = isZh ? (url == '/zh' ? '/' : url.substring(3)) : (url == '/' ? '/zh/' : '/zh$url');
    return a(
      classes: 'lang-switch',
      href: other,
      attributes: {'hreflang': isZh ? 'en' : 'zh-CN', 'title': isZh ? 'English' : '中文'},
      [Component.text(isZh ? 'EN' : '中文')],
    );
  }
}

/// Search box; behavior and the index come from web/search.js and
/// search-index.json (generated after the static build).
class SearchBox extends StatelessComponent {
  const SearchBox({super.key});

  @override
  Component build(BuildContext context) {
    final isZh = context.page.url == '/zh' || context.page.url.startsWith('/zh/');
    return Component.fragment([
      Document.head(
        children: [
          link(rel: 'icon', type: 'image/svg+xml', href: '/images/logo.svg'),
          link(rel: 'stylesheet', href: '/site.css'),
          script(src: '/search.js', attributes: {'defer': ''}),
        ],
      ),
      div(classes: 'search', [
        input(
          type: InputType.search,
          attributes: {
            'id': 'search-input',
            'placeholder': isZh ? '搜索文档… (/)' : 'Search docs… (/)',
            'aria-label': isZh ? '搜索' : 'Search',
            'autocomplete': 'off',
            'data-lang': isZh ? 'zh' : 'en',
          },
        ),
        div(classes: 'search-results', attributes: {'id': 'search-results', 'role': 'listbox'}, []),
      ]),
    ]);
  }
}
