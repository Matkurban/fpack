import 'dart:ffi';
import 'dart:io';

/// Language of the wrapper's own messages, resolved like the Go core:
/// `--lang` > `FPACK_LANG` > the operating system's language. Chinese (any
/// zh variant) gives Chinese, everything else English. See
/// go/internal/i18n/i18n.go for the per-OS rules.
bool? _zh;
List<String> _args = const [];

/// Records the command line for `--lang`; detection itself is lazy (only
/// when the wrapper actually prints something).
void initLang(List<String> args) {
  _args = args;
  _zh = null;
}

bool get isZh => _zh ??= detectZh(
  override: langFlag(_args),
  env: Platform.environment,
  os: Platform.operatingSystem,
  osLanguage: osLanguage,
);

/// Picks the message for the active language.
String tr(String en, String zh) => isZh ? zh : en;

String? langFlag(List<String> args) {
  for (var i = 0; i < args.length; i++) {
    final a = args[i];
    if (a == '--') break;
    if (a.startsWith('--lang=')) return a.substring(7);
    if (a == '--lang' && i + 1 < args.length) return args[i + 1];
  }
  return null;
}

/// null for values that are neither Chinese nor English.
bool? parseLang(String? v) {
  final s = (v ?? '').trim().toLowerCase();
  if (s.isEmpty) return null;
  if (s.startsWith('zh') || s == 'cn' || s == 'chinese' || s.startsWith('中文')) {
    return true;
  }
  if (s.startsWith('en') || s == 'english') return false;
  return null;
}

bool detectZh({
  String? override,
  required Map<String, String> env,
  required String os,
  required String? Function(String os) osLanguage,
}) {
  final o = parseLang(override);
  if (o != null) return o;
  final f = parseLang(env['FPACK_LANG']);
  if (f != null) return f;
  final desktop = os == 'macos' || os == 'windows';
  final explicit = [
    'LC_ALL',
    'LC_MESSAGES',
    if (!desktop) ...['LANGUAGE', 'LANG'],
  ];
  final e = _fromEnv(explicit, env);
  if (e != null) return e;
  if (desktop) {
    final v = osLanguage(os);
    if (v != null && v.isNotEmpty) return parseLang(v) == true;
    return _fromEnv(['LANGUAGE', 'LANG'], env) ?? false;
  }
  return false;
}

bool? _fromEnv(List<String> keys, Map<String, String> env) {
  for (final k in keys) {
    var v = (env[k] ?? '').trim();
    if (k == 'LANGUAGE') v = v.split(':').first;
    if (v.isEmpty || v == 'C' || v == 'POSIX' || v.startsWith('C.')) continue;
    return parseLang(v) == true;
  }
  return null;
}

/// OS UI language tag, or null. macOS shares the core's cache
/// (~/Library/Caches/fpack/os-language, keyed by the global preferences'
/// mtime); Windows asks kernel32 directly.
String? osLanguage(String os) {
  try {
    if (os == 'macos') return _macLanguage();
    if (os == 'windows') return _windowsLanguage();
  } catch (_) {}
  return null;
}

String? _macLanguage() {
  final home = Platform.environment['HOME'] ?? '';
  if (home.isEmpty) return null;
  final prefs = File('$home/Library/Preferences/.GlobalPreferences.plist');
  final cache = File('$home/Library/Caches/fpack/os-language');
  String? stamp;
  if (prefs.existsSync()) {
    stamp = '${prefs.statSync().modified.microsecondsSinceEpoch}';
    if (cache.existsSync()) {
      final parts = cache.readAsStringSync().trim().split(' ');
      if (parts.length == 2 && parts[0] == stamp) return parts[1];
    }
  }
  String run(String key) {
    final r = Process.runSync('defaults', ['read', '-g', key]);
    return r.exitCode == 0 ? '${r.stdout}'.trim() : '';
  }

  var v = firstAppleLanguage(run('AppleLanguages'));
  if (v.isEmpty) v = run('AppleLocale');
  if (v.isEmpty) return null;
  return v;
}

/// First entry of `defaults read -g AppleLanguages` output.
String firstAppleLanguage(String out) {
  for (final f in out.split(RegExp(r'[()",\s]+'))) {
    if (f.isNotEmpty) return f;
  }
  return '';
}

typedef _UILangNative = Uint16 Function();
typedef _UILang = int Function();

String? _windowsLanguage() {
  final k32 = DynamicLibrary.open('kernel32.dll');
  final f = k32.lookupFunction<_UILangNative, _UILang>(
    'GetUserDefaultUILanguage',
  );
  final id = f();
  if (id == 0) return null;
  return (id & 0x3ff) == 0x04 ? 'zh' : 'en'; // LANG_CHINESE
}
