import 'package:jaspr/dom.dart' hide footer, header;
import 'package:jaspr/jaspr.dart';
import 'package:jaspr_content/jaspr_content.dart';

import '../content/localized_headings.dart';
import '../l10n/locales.dart';
import '../nav.dart';

/// Picks the language before the first paint: `?lang=` (saved), the saved
/// choice, then the browser language. Sets `<html data-locale lang>`, which the
/// CSS uses to show only that locale (no flash of the other language).
const localeBootstrapScript = '''
(function(){var d=document.documentElement,k='fpack-lang',ok=function(v){return v==='en'||v==='zh'},
q=new URLSearchParams(location.search).get('lang'),s=null;try{s=localStorage.getItem(k)}catch(e){}
if(ok(q)){s=q;try{localStorage.setItem(k,q)}catch(e){}}
var b=((navigator.languages&&navigator.languages[0])||navigator.language||'').toLowerCase();
var l=ok(s)?s:(b.indexOf('zh')===0?'zh':'en');d.setAttribute('data-locale',l);d.lang=l==='zh'?'zh-CN':'en';})();''';

/// The docs layout of jaspr_content with every locale's title, description
/// and table of contents rendered side by side (see [LocalizedBlock]).
class SiteLayout extends DocsLayout {
  const SiteLayout({super.sidebar, super.header, super.footer});

  @override
  Iterable<Component> buildHead(Page page) sync* {
    yield script(content: localeBootstrapScript);
    yield* super.buildHead(page);
    final url = page.url;
    yield link(rel: 'alternate', href: url, attributes: {'hreflang': 'x-default'});
    for (final locale in AppLocale.values) {
      yield link(
        rel: 'alternate',
        href: locale == SiteLocale.fallback ? url : '$url?lang=${locale.languageCode}',
        attributes: {'hreflang': locale.htmlLang},
      );
    }
    for (final MapEntry(key: lang, value: data) in localizedData(page).entries) {
      if (data['title'] case final String title) {
        yield meta(attributes: {'name': 'fpack:title-$lang', 'content': '$title | fpack'});
      }
    }
    yield link(rel: 'icon', type: 'image/svg+xml', href: '/images/logo.svg');
    yield link(rel: 'stylesheet', href: '/site.css');
    yield script(src: '/site.js', defer: true);
  }

  @override
  Component buildBody(Page page, Component child) {
    final locales = localizedData(page);
    final tocs = page.data['tocs'] as Map<String, List<TocItem>>? ?? const {};
    return div(classes: 'docs', [
      if (header case final header?)
        div(classes: 'header-container', attributes: {if (sidebar != null) 'data-has-sidebar': ''}, [header]),
      div(classes: 'main-container', [
        div(classes: 'sidebar-barrier', attributes: {'role': 'button'}, []),
        if (sidebar case final sidebar?) div(classes: 'sidebar-container', [sidebar]),
        main_([
          div([
            div(classes: 'content-container', [
              if (locales.isNotEmpty)
                LocalizedBlock((locale) {
                  final data = locales[locale.languageCode] ?? const {};
                  return div(classes: 'content-header', [
                    if (data['title'] case final String title) h1([Component.text(title)]),
                    if (data['description'] case final String description) p([Component.text(description)]),
                  ]);
                }),
              child,
              pageNavigation(page.url),
              if (footer case final footer?) div(classes: 'content-footer', [footer]),
            ]),
            aside(classes: 'toc', [
              if (tocs.values.any((toc) => toc.isNotEmpty))
                LocalizedBlock(
                  (locale) => div([
                    h3([Component.text(locale.strings.tocTitle)]),
                    ul([
                      for (final item in tocs[locale.languageCode] ?? const <TocItem>[])
                        li(styles: Styles(padding: Padding.only(left: (0.75 * item.level).em)), [
                          a(href: _pageAnchor(page.url, item.id), [Component.text(item.text)]),
                        ]),
                    ]),
                  ]),
                ),
            ]),
          ]),
        ]),
      ]),
    ]);
  }
}

/// Title and description of every locale (set by the content loader).
Map<String, Map<String, Object?>> localizedData(Page page) => {
  for (final MapEntry(:key, :value) in ((page.data.page['locales'] as Map?) ?? const {}).entries)
    if (value is Map) '$key': value.cast<String, Object?>(),
};

/// An in-page link that survives `<base href>`: a bare `#id` would resolve
/// against the base (the site root) and navigate to the home page.
String _pageAnchor(String url, String id) => '${url == '/' ? '' : url}/#$id';

/// Previous / next links in reading order, one block per locale.
Component pageNavigation(String url) {
  final (previous, next) = neighbours(url);
  if (previous == null && next == null) return const Component.empty();
  return LocalizedBlock(
    (locale) => nav(
      classes: 'page-nav',
      attributes: {'aria-label': locale.strings.nextPage},
      [
        if (previous != null)
          a(classes: 'page-nav-prev', href: previous.href, [
            span([Component.text('← ${locale.strings.previousPage}')]),
            strong([Component.text(previous.title(locale.strings))]),
          ])
        else
          const Component.empty(),
        if (next != null)
          a(classes: 'page-nav-next', href: next.href, [
            span([Component.text('${locale.strings.nextPage} →')]),
            strong([Component.text(next.title(locale.strings))]),
          ]),
      ],
    ),
  );
}
