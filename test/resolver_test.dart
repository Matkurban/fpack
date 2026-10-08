@TestOn('linux || mac-os')
library;

import 'dart:convert';
import 'dart:ffi' show Abi;
import 'dart:io';

import 'package:fpack/src/platform.dart';
import 'package:fpack/src/resolver.dart';
import 'package:fpack/src/sha256.dart';
import 'package:test/test.dart';

const host = HostTarget('linux', 'amd64');

String fakeCore(String version) =>
    '#!/bin/sh\nif [ "\$1" = "--core-version" ]; then echo $version; fi\n';

void main() {
  late Directory tmp;
  late String pkg;
  late String cache;
  final logs = <String>[];

  setUp(() {
    tmp = Directory.systemTemp.createTempSync('fpack_resolver');
    pkg = '${tmp.path}/pkg';
    cache = '${tmp.path}/cache';
    Directory(pkg).createSync();
    logs.clear();
  });
  tearDown(() => tmp.deleteSync(recursive: true));

  CoreResolver resolver({
    Map<String, String> env = const {},
    Downloader? downloader,
    String version = '1.0.0',
  }) => CoreResolver(
    packageRoot: pkg,
    host: host,
    environment: {'FPACK_GO': 'none', 'FPACK_NO_DOWNLOAD': '1', ...env},
    cacheRoot: cache,
    version: version,
    downloader: downloader,
    log: logs.add,
  );

  void bundle(String content, {String? sha, String version = '1.0.0'}) {
    final f = File('$pkg/prebuilt/linux-amd64/fpack-core')
      ..createSync(recursive: true)
      ..writeAsStringSync(content);
    File('$pkg/prebuilt/manifest.json').writeAsStringSync(
      jsonEncode({
        'version': version,
        'binaries': {
          'linux-amd64': {'sha256': sha ?? sha256Hex(f.readAsBytesSync())},
        },
      }),
    );
  }

  test(
    'bundled binary is verified, copied to the cache and made executable',
    () async {
      bundle(fakeCore('1.0.0'));
      final r = resolver();
      final core = await r.resolve();
      expect(core.source, CoreSource.bundled);
      expect(core.path, '$cache/1.0.0/linux-amd64/fpack-core');
      expect(File('${core.path}.stamp').existsSync(), isTrue);
      expect(await r.coreVersion(core.path), '1.0.0');

      // Second run: served from the cache without touching prebuilt/.
      File('$pkg/prebuilt/manifest.json').deleteSync();
      final again = await resolver().resolve();
      expect(again.source, CoreSource.cache);
    },
  );

  test('a rebuilt bundled binary (same version) replaces the cache', () async {
    bundle(fakeCore('1.0.0'));
    expect((await resolver().resolve()).source, CoreSource.bundled);
    expect((await resolver().resolve()).source, CoreSource.cache);
    bundle('${fakeCore('1.0.0')}# rebuilt\n');
    final again = await resolver().resolve();
    expect(again.source, CoreSource.bundled);
    expect(File(again.path).readAsStringSync(), contains('# rebuilt'));
  });

  test('a modified cache entry is not trusted', () async {
    bundle(fakeCore('1.0.0'));
    final core = await resolver().resolve();
    File(core.path).writeAsStringSync('${fakeCore('1.0.0')}# changed\n');
    final again = await resolver().resolve();
    expect(again.source, CoreSource.bundled);
  });

  test('checksum mismatch rejects the bundled binary', () async {
    bundle(fakeCore('1.0.0'), sha: '0' * 64);
    final r = resolver();
    await expectLater(r.resolve(), throwsA(isA<ResolveException>()));
    expect(r.attempts.join('\n'), contains('checksum mismatch'));
  });

  test('version mismatch between wrapper and prebuilt is rejected', () async {
    bundle(fakeCore('0.0.9'), version: '0.0.9');
    final r = resolver();
    await expectLater(r.resolve(), throwsA(isA<ResolveException>()));
    expect(r.attempts.join('\n'), contains('prebuilt version 0.0.9'));
  });

  test('core reporting a different version is rejected', () async {
    bundle(fakeCore('9.9.9'));
    final r = resolver();
    await expectLater(r.resolve(), throwsA(isA<ResolveException>()));
    expect(r.attempts.join('\n'), contains('does not run'));
  });

  test('FPACK_CORE wins and warns on version drift', () async {
    final f = File('${tmp.path}/my-core')..writeAsStringSync(fakeCore('0.2.0'));
    await Process.run('chmod', ['755', f.path]);
    final core = await resolver(env: {'FPACK_CORE': f.path}).resolve();
    expect(core.source, CoreSource.env);
    expect(logs.single, contains('reports version 0.2.0'));
  });

  test('missing FPACK_CORE file is a clear error', () async {
    await expectLater(
      resolver(env: {'FPACK_CORE': '/nope/core'}).resolve(),
      throwsA(isA<ResolveException>()),
    );
  });

  test('download is verified against checksums.txt', () async {
    final good = fakeCore('1.0.0');
    final served = <String, String>{
      'checksums.txt':
          '${sha256Hex(utf8.encode(good))}  fpack-core-linux-amd64\n'
          '${'1' * 64}  fpack-core-darwin-arm64\n',
      'fpack-core-linux-amd64': good,
    };
    final urls = <Uri>[];
    Future<void> dl(Uri u, File dest) async {
      urls.add(u);
      dest.writeAsStringSync(served[u.pathSegments.last]!);
    }

    final r = resolver(
      env: {'FPACK_NO_DOWNLOAD': '', 'FPACK_DOWNLOAD_URL': 'https://x/y'},
      downloader: dl,
    );
    final core = await r.resolve();
    expect(core.source, CoreSource.downloaded);
    expect(urls.map((u) => u.toString()), [
      'https://x/y/checksums.txt',
      'https://x/y/fpack-core-linux-amd64',
    ]);

    // Tampered binary -> refused.
    served['fpack-core-linux-amd64'] = '${good}evil';
    File(core.path).deleteSync();
    final r2 = resolver(
      env: {'FPACK_NO_DOWNLOAD': '', 'FPACK_DOWNLOAD_URL': 'https://x/y'},
      downloader: dl,
    );
    await expectLater(r2.resolve(), throwsA(isA<ResolveException>()));
    expect(r2.attempts.join('\n'), contains('SHA-256 mismatch'));
  });

  test('default download URL is the GitHub release of this version', () {
    expect(
      resolver(env: {}).downloadBase.toString(),
      'https://github.com/Matkurban/fpack/releases/download/v1.0.0/',
    );
  });

  test('go build from sources', () async {
    final go = Process.runSync('which', ['go']).stdout.toString().trim();
    if (go.isEmpty) {
      markTestSkipped('no go toolchain');
      return;
    }
    // Use the real Go sources of this repository.
    final repo = Directory.current.path;
    final r = CoreResolver(
      packageRoot: repo,
      host: HostTarget.current(),
      environment: {
        ...Platform.environment,
        'FPACK_REBUILD': '1',
        'FPACK_NO_DOWNLOAD': '1',
      },
      cacheRoot: cache,
      log: logs.add,
    );
    final core = await r.resolve();
    expect(core.source, CoreSource.built);
    expect(logs.single, contains('building the native core'));
  }, timeout: const Timeout(Duration(minutes: 3)));

  test('parseChecksums handles sha256sum and shasum formats', () {
    final m = parseChecksums('${'a' * 64}  file1\n${'B' * 64} *file2\nnoise\n');
    expect(m, {'file1': 'a' * 64, 'file2': 'b' * 64});
  });

  test('host mapping', () {
    expect(HostTarget.fromAbi(Abi.macosArm64).id, 'darwin-arm64');
    expect(HostTarget.fromAbi(Abi.windowsX64).exeName, 'fpack-core.exe');
    expect(
      HostTarget.fromAbi(Abi.linuxArm64).assetName,
      'fpack-core-linux-arm64',
    );
    expect(
      () => HostTarget.fromAbi(Abi.androidArm64),
      throwsA(isA<UnsupportedError>()),
    );
  });

  test('cache roots per OS', () {
    expect(
      defaultCacheRoot({
        'HOME': '/Users/a',
      }, const HostTarget('darwin', 'arm64')),
      '/Users/a/Library/Caches/fpack',
    );
    expect(
      defaultCacheRoot({'HOME': '/h', 'XDG_CACHE_HOME': '/x'}, host),
      '/x/fpack',
    );
    expect(defaultCacheRoot({'HOME': '/h'}, host), '/h/.cache/fpack');
    expect(
      defaultCacheRoot({
        'LOCALAPPDATA': r'C:\L',
      }, const HostTarget('windows', 'amd64')),
      r'C:\L\fpack',
    );
    expect(defaultCacheRoot({'FPACK_HOME': '/f', 'HOME': '/h'}, host), '/f');
  });
}
