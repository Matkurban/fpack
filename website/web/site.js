// fpack docs: in-page anchors, tabs, language switching and client-side search.
//
// Every page contains all locales; <html data-locale> (set before the first
// paint by the inline script in the page head) selects the visible one, see
// site.css. The search index (search-index.json, one entry per locale and
// section) is generated from the built pages by tool/postbuild.py.
(function () {
  var base = (document.currentScript && document.currentScript.src || '').replace(/site\.js(\?.*)?$/, '');
  var STORAGE_KEY = 'fpack-lang';
  function locale() { return document.documentElement.getAttribute('data-locale') === 'zh' ? 'zh' : 'en'; }
  // Applies attribute translations (data-l10n-<attr>-<locale>) and the title.
  function applyLocale() {
    var l = locale();
    document.querySelectorAll('*').forEach(function (el) {
      for (var i = 0; i < el.attributes.length; i++) {
        var m = /^data-l10n-(.+)-(en|zh)$/.exec(el.attributes[i].name);
        if (m && m[2] === l && m[1] !== 'empty') el.setAttribute(m[1], el.attributes[i].value);
      }
    });
    var title = document.querySelector('meta[name="fpack:title-' + l + '"]');
    if (title) document.title = title.getAttribute('content');
  }
  function setLocale(l) {
    var d = document.documentElement;
    d.setAttribute('data-locale', l);
    d.lang = l === 'zh' ? 'zh-CN' : 'en';
    try { localStorage.setItem(STORAGE_KEY, l); } catch (e) {}
    var url = new URL(location.href);
    if (url.searchParams.has('lang')) { url.searchParams.set('lang', l); history.replaceState(null, '', url); }
    applyLocale();
  }

  // ---- in-page anchors -------------------------------------------------------
  // The pages have <base href="/fpack/">, so a bare "#id" link would resolve to
  // the site root and navigate to the home page. Links to a heading of the
  // current page (the table of contents, the heading permalinks) are handled
  // here: scroll, update the hash, keep the query (?lang) and the page.
  function samePage(a) {
    var u;
    try { u = new URL(a.getAttribute('href'), location.href); } catch (e) { return null; }
    var norm = function (p) { return p.replace(/\/index\.html$/, '/').replace(/\/$/, ''); };
    return u.origin === location.origin && norm(u.pathname) === norm(location.pathname) && u.hash.length > 1 ? u : null;
  }
  function scrollToHash(hash, smooth) {
    var el = document.getElementById(decodeURIComponent(hash.slice(1)));
    if (!el) return false;
    el.scrollIntoView({ behavior: smooth ? 'smooth' : 'auto', block: 'start' });
    return true;
  }
  function initAnchors() {
    document.addEventListener('click', function (ev) {
      if (ev.defaultPrevented || ev.button !== 0 || ev.metaKey || ev.ctrlKey || ev.shiftKey || ev.altKey) return;
      var a = ev.target.closest && ev.target.closest('a[href]');
      if (!a) return;
      var raw = a.getAttribute('href');
      var u = raw.charAt(0) === '#' ? new URL(location.href) : samePage(a);
      if (!u) return;
      var hash = raw.charAt(0) === '#' ? raw : u.hash;
      if (!document.getElementById(decodeURIComponent(hash.slice(1)))) return;
      ev.preventDefault();
      history.pushState(null, '', location.pathname + location.search + hash);
      scrollToHash(hash, true);
      var side = document.querySelector('.sidebar-container.open');
      if (side) side.classList.remove('open');
    });
    window.addEventListener('popstate', function () { if (location.hash) scrollToHash(location.hash, false); });
    if (location.hash) {
      // Headings of the inactive locale are hidden; wait for layout, then jump.
      requestAnimationFrame(function () { scrollToHash(location.hash, false); });
    }
  }
  // Highlights the table-of-contents entry of the heading being read.
  function initScrollSpy() {
    var links = Array.prototype.slice.call(document.querySelectorAll('aside.toc a[href*="#"]'));
    if (!links.length || !('IntersectionObserver' in window)) return;
    var byId = {};
    links.forEach(function (a) { byId[decodeURIComponent(a.getAttribute('href').split('#')[1])] = a; });
    var current = null;
    function mark(id) {
      var a = byId[id];
      if (!a || a === current) return;
      if (current) current.classList.remove('active');
      a.classList.add('active'); current = a;
    }
    var seen = {};
    var io = new IntersectionObserver(function (entries) {
      entries.forEach(function (e) { seen[e.target.id] = e.isIntersecting; });
      var first = Object.keys(byId).find(function (id) {
        var el = document.getElementById(id);
        return seen[id] && el && el.offsetParent !== null;
      });
      if (first) mark(first);
    }, { rootMargin: '-72px 0px -65% 0px' });
    Object.keys(byId).forEach(function (id) { var el = document.getElementById(id); if (el) io.observe(el); });
  }

  // ---- tabs (::: tabs blocks) ------------------------------------------------
  var TAB_KEY = 'fpack-tab';
  function defaultTab() {
    var p = (navigator.userAgentData && navigator.userAgentData.platform) || navigator.platform || navigator.userAgent || '';
    p = p.toLowerCase();
    return p.indexOf('mac') >= 0 ? 'macos' : p.indexOf('win') >= 0 ? 'windows' : p.indexOf('linux') >= 0 ? 'linux' : '';
  }
  function key(label) { return label.toLowerCase().replace(/[^a-z0-9]+/g, ' ').trim().split(' ')[0] || label; }
  function selectTab(group, label) {
    var bar = group.querySelector(':scope > .tab-bar');
    group.querySelectorAll(':scope > .tab-panel').forEach(function (p) { p.hidden = p.getAttribute('data-tab') !== label; });
    bar.querySelectorAll('button').forEach(function (b) {
      var on = b.getAttribute('data-tab') === label;
      b.setAttribute('aria-selected', on ? 'true' : 'false'); b.tabIndex = on ? 0 : -1;
    });
  }
  function initTabs() {
    var saved = null;
    try { saved = localStorage.getItem(TAB_KEY); } catch (e) {}
    document.querySelectorAll('[data-tabs]').forEach(function (group) {
      var panels = group.querySelectorAll(':scope > .tab-panel');
      if (!panels.length) return;
      var bar = document.createElement('div');
      bar.className = 'tab-bar'; bar.setAttribute('role', 'tablist');
      panels.forEach(function (p) {
        var label = p.getAttribute('data-tab');
        var b = document.createElement('button');
        b.type = 'button'; b.setAttribute('role', 'tab'); b.setAttribute('data-tab', label); b.textContent = label;
        b.addEventListener('click', function () {
          try { localStorage.setItem(TAB_KEY, key(label)); } catch (e) {}
          // every tab group of the page that has the same kind of tab follows
          document.querySelectorAll('[data-tabs]').forEach(function (g) {
            var m = Array.prototype.find.call(g.querySelectorAll(':scope > .tab-panel'), function (q) { return key(q.getAttribute('data-tab')) === key(label); });
            if (m) selectTab(g, m.getAttribute('data-tab'));
          });
        });
        bar.appendChild(b);
      });
      group.classList.add('js');
      group.insertBefore(bar, group.firstChild);
      var want = saved || defaultTab();
      var pick = Array.prototype.find.call(panels, function (p) { return key(p.getAttribute('data-tab')) === want; }) || panels[0];
      selectTab(group, pick.getAttribute('data-tab'));
    });
  }

  var index = null, loading = null;
  function load() {
    if (index) return Promise.resolve(index);
    if (!loading) loading = fetch(base + 'search-index.json').then(function (r) { return r.json(); }).then(function (d) { index = d; return d; });
    return loading;
  }
  function esc(s) { return s.replace(/[&<>"]/g, function (c) { return { '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;' }[c]; }); }
  function mark(text, terms) {
    var out = esc(text);
    terms.forEach(function (t) {
      if (!t) return;
      var re = new RegExp(t.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'), 'gi');
      out = out.replace(re, function (m) { return '<mark>' + m + '</mark>'; });
    });
    return out;
  }
  function search(q, lang) {
    var terms = q.toLowerCase().split(/\s+/).filter(Boolean);
    if (!terms.length) return [];
    var res = [];
    index.forEach(function (e) {
      if (e.lang !== lang) return;
      var title = e.title.toLowerCase(), section = (e.section || '').toLowerCase(), text = e.text.toLowerCase();
      var score = 0;
      for (var i = 0; i < terms.length; i++) {
        var t = terms[i], s = 0;
        if (section.indexOf(t) >= 0) s += 6;
        if (title.indexOf(t) >= 0) s += 4;
        if (text.indexOf(t) >= 0) s += 1;
        if (!s) return;
        score += s;
      }
      var pos = text.indexOf(terms[0]);
      var start = Math.max(0, pos - 50);
      var snippet = (start > 0 ? '…' : '') + e.text.substr(start, 150) + (e.text.length > start + 150 ? '…' : '');
      res.push({ e: e, score: score, snippet: snippet });
    });
    res.sort(function (a, b) { return b.score - a.score; });
    return res.slice(0, 12).map(function (r) { r.terms = terms; return r; });
  }
  function init() {
    applyLocale();
    initAnchors();
    initTabs();
    initScrollSpy();
    document.querySelectorAll('[data-language-switch]').forEach(function (b) {
      b.addEventListener('click', function () { setLocale(locale() === 'zh' ? 'en' : 'zh'); });
    });
    var input = document.getElementById('search-input');
    var box = document.getElementById('search-results');
    if (!input || !box) return;
    var active = -1;
    function render() {
      var q = input.value.trim();
      if (!q) { box.classList.remove('open'); box.innerHTML = ''; return; }
      load().then(function () {
        var lang = locale();
        var res = search(q, lang);
        active = -1;
        box.innerHTML = res.length ? res.map(function (r) {
          return '<a href="' + base + r.e.url.replace(/^\//, '') + '">' +
            '<div class="r-title">' + mark(r.e.title, r.terms) + (r.e.section ? ' <span class="r-section">› ' + mark(r.e.section, r.terms) + '</span>' : '') + '</div>' +
            '<div class="r-snippet">' + mark(r.snippet, r.terms) + '</div></a>';
        }).join('') : '<div class="r-empty">' + esc(input.getAttribute('data-l10n-empty-' + lang) || '') + '</div>';
        box.classList.add('open');
      });
    }
    input.addEventListener('input', render);
    input.addEventListener('focus', function () { load(); if (input.value) render(); });
    input.addEventListener('keydown', function (ev) {
      var links = box.querySelectorAll('a');
      if (ev.key === 'ArrowDown' || ev.key === 'ArrowUp') {
        ev.preventDefault();
        if (!links.length) return;
        active = (active + (ev.key === 'ArrowDown' ? 1 : -1) + links.length) % links.length;
        links.forEach(function (l, i) { l.classList.toggle('active', i === active); });
        links[active].scrollIntoView({ block: 'nearest' });
      } else if (ev.key === 'Enter') {
        var target = links[active >= 0 ? active : 0];
        if (target) window.location.href = target.href;
      } else if (ev.key === 'Escape') {
        box.classList.remove('open'); input.blur();
      }
    });
    document.addEventListener('click', function (ev) { if (!ev.target.closest('.search')) box.classList.remove('open'); });
    document.addEventListener('keydown', function (ev) {
      if (ev.key === '/' && document.activeElement !== input && !/input|textarea/i.test(document.activeElement.tagName)) {
        ev.preventDefault(); input.focus();
      }
    });
  }
  if (document.readyState === 'loading') document.addEventListener('DOMContentLoaded', init); else init();
})();
