// fpack docs: language switching and client-side search.
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
