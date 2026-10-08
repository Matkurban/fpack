@TestOn('linux || mac-os')
library;

import 'dart:io';

import 'package:fpack/src/launcher.dart';
import 'package:test/test.dart';

void main() {
  late Directory tmp;
  setUp(() => tmp = Directory.systemTemp.createTempSync('fpack_launcher'));
  tearDown(() => tmp.deleteSync(recursive: true));

  Future<String> script(String body) async {
    final f = File('${tmp.path}/core')..writeAsStringSync('#!/bin/sh\n$body\n');
    await Process.run('chmod', ['755', f.path]);
    return f.path;
  }

  test('exit code and wrapper version are passed through', () async {
    final out = '${tmp.path}/out';
    final core = await script(
      'echo "\$FPACK_WRAPPER_VERSION \$*" > $out\nexit 7',
    );
    expect(await runCore(core, ['build', 'apk']), 7);
    expect(File(out).readAsStringSync().trim(), '1.0.0 build apk');
  });

  test('a core killed by a signal maps to 128+N', () async {
    final core = await script(r'kill -TERM $$');
    expect(await runCore(core, []), 143);
  });

  test('missing core is a clean error', () async {
    expect(await runCore('${tmp.path}/nope', []), 3);
  });

  test('package root contains the Go sources', () async {
    final root = await packageRoot();
    expect(File('$root/go/go.mod').existsSync(), isTrue);
  });
}
