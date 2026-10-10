import 'package:fpack/src/i18n.dart';
import 'package:test/test.dart';

void main() {
  bool zh(String os, Map<String, String> env, {String? osLang, String? flag}) =>
      detectZh(override: flag, env: env, os: os, osLanguage: (_) => osLang);

  test('flag and FPACK_LANG win', () {
    expect(zh('macos', {}, osLang: 'zh-Hans-CN', flag: 'en'), isFalse);
    expect(zh('linux', {'LANG': 'en_US.UTF-8'}, flag: 'zh'), isTrue);
    expect(zh('macos', {'FPACK_LANG': 'en'}, osLang: 'zh-Hans'), isFalse);
    expect(zh('linux', {'FPACK_LANG': 'zh_CN'}), isTrue);
  });

  test('macOS: UI language beats Terminal LANG', () {
    expect(zh('macos', {'LANG': 'en_US.UTF-8'}, osLang: 'zh-Hans-CN'), isTrue);
    expect(zh('macos', {'LANG': 'zh_CN.UTF-8'}, osLang: 'en-CN'), isFalse);
    expect(zh('macos', {'LC_ALL': 'en_US.UTF-8'}, osLang: 'zh-Hans'), isFalse);
    expect(zh('macos', {'LANG': 'zh_CN.UTF-8'}), isTrue); // lookup failed
    expect(zh('macos', {}, osLang: 'ja-JP'), isFalse);
  });

  test('Windows: UI language beats LANG', () {
    expect(zh('windows', {'LANG': 'en_US.UTF-8'}, osLang: 'zh'), isTrue);
    expect(zh('windows', {}, osLang: 'en'), isFalse);
    expect(zh('windows', {}), isFalse);
  });

  test('Linux: environment', () {
    expect(zh('linux', {'LANG': 'zh_TW.UTF-8'}), isTrue);
    expect(zh('linux', {'LANG': 'de_DE.UTF-8'}), isFalse);
    expect(
      zh('linux', {'LANGUAGE': 'zh_CN:en', 'LANG': 'en_US.UTF-8'}),
      isTrue,
    );
    expect(zh('linux', {'LC_ALL': 'C.UTF-8', 'LANG': 'zh_CN.UTF-8'}), isTrue);
    expect(zh('linux', {}), isFalse);
  });

  test('helpers', () {
    expect(langFlag(['build', '--lang', 'zh']), 'zh');
    expect(langFlag(['--lang=en']), 'en');
    expect(langFlag(['--', '--lang', 'zh']), isNull);
    expect(firstAppleLanguage('(\n    "zh-Hans-CN",\n    en\n)'), 'zh-Hans-CN');
  });
}
