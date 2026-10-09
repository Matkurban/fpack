@TestOn('linux || mac-os')
library;

import 'dart:async';
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

  /// A fake `go` that answers `go env GOVERSION` and "builds" a core that
  /// reports [coreVersion]; every call is appended to calls.log.
  String fakeGo({String coreVersion = '1.0.0'}) {
    Directory('$pkg/go').createSync(recursive: true);
    File('$pkg/go/go.mod').writeAsStringSync('module x\n');
    final f = File('${tmp.path}/fakego')
      ..writeAsStringSync(
        '#!/bin/sh\n'
        'echo "\$*" >> ${tmp.path}/calls.log\n'
        'if [ "\$1" = env ]; then echo go1.99.0; exit 0; fi\n'
        'out=""; prev=""\n'
        'for a in "\$@"; do [ "\$prev" = -o ] && out=\$a; prev=\$a; done\n'
        "printf '%s' '${fakeCore(coreVersion)}' > \"\$out\"\n",
      );
    Process.runSync('chmod', ['755', f.path]);
    return f.path;
  }

  bool goCalled() => File('${tmp.path}/calls.log').existsSync();

  Future<void> Function(Uri, File) server(String core) {
    final served = <String, String>{
      'checksums.txt':
          '${sha256Hex(utf8.encode(core))}  fpack-core-linux-amd64\n',
      'fpack-core-linux-amd64': core,
    };
    return (u, dest) async =>
        dest.writeAsStringSync(served[u.pathSegments.last]!);
  }

  Future<void> offline(Uri u, File dest) async =>
      throw const SocketException('network is unreachable');

  const dlEnv = {'FPACK_NO_DOWNLOAD': '', 'FPACK_DOWNLOAD_URL': 'https://x/y'};

  test('order: a verified download wins over a local go build', () async {
    final go = fakeGo();
    final r = resolver(
      env: {...dlEnv, 'FPACK_GO': go},
      downloader: server(fakeCore('1.0.0')),
    );
    final core = await r.resolve();
    expect(core.source, CoreSource.downloaded);
    expect(goCalled(), isFalse);
    expect(logs.join('\n'), isNot(contains('local build')));
  });

  test('order: the bundled binary wins over download and go', () async {
    bundle(fakeCore('1.0.0'));
    final go = fakeGo();
    var downloads = 0;
    final r = resolver(
      env: {...dlEnv, 'FPACK_GO': go},
      downloader: (u, d) async => downloads++,
    );
    expect((await r.resolve()).source, CoreSource.bundled);
    expect(downloads, 0);
    expect(goCalled(), isFalse);
  });

  test('order: go build only when the download fails, with a note', () async {
    final go = fakeGo();
    final r = resolver(env: {...dlEnv, 'FPACK_GO': go}, downloader: offline);
    final core = await r.resolve();
    expect(core.source, CoreSource.built);
    final log = logs.join('\n');
    expect(log, contains('falling back to a local build'));
    expect(log, contains('go1.99.0'));
    expect(log, contains(go));
    expect(log, contains('network is unreachable'));
    expect(await r.coreVersion(core.path), '1.0.0');
    // Next run: cache, no new build.
    File('${tmp.path}/calls.log').deleteSync();
    expect(
      (await resolver(env: {'FPACK_GO': go}).resolve()).source,
      CoreSource.cache,
    );
    expect(goCalled(), isFalse);
  });

  test('order: a tampered download falls back to go build', () async {
    final go = fakeGo();
    final good = fakeCore('1.0.0');
    final r = resolver(
      env: {...dlEnv, 'FPACK_GO': go},
      downloader: (u, dest) async => dest.writeAsStringSync(
        u.pathSegments.last == 'checksums.txt'
            ? '${sha256Hex(utf8.encode(good))}  fpack-core-linux-amd64\n'
            : '${good}evil',
      ),
    );
    expect((await r.resolve()).source, CoreSource.built);
    expect(logs.join('\n'), contains('SHA-256 mismatch'));
  });

  test('FPACK_NO_DOWNLOAD=1 goes straight to the local build', () async {
    final go = fakeGo();
    var downloads = 0;
    final r = resolver(
      env: {'FPACK_GO': go},
      downloader: (u, d) async => downloads++,
    );
    expect((await r.resolve()).source, CoreSource.built);
    expect(downloads, 0);
    expect(logs.join('\n'), contains('disabled by FPACK_NO_DOWNLOAD'));
  });

  test('FPACK_GO=none and no download: clear error with fixes', () async {
    fakeGo(); // sources present, but Go disabled
    final r = resolver(env: {...dlEnv}, downloader: offline);
    final e = await r.resolve().then<Object?>((_) => null, onError: (e) => e);
    expect(e, isA<ResolveException>());
    final text = e.toString();
    expect(text, contains('network is unreachable'));
    expect(text, contains('disabled by FPACK_GO=none'));
    expect(text, contains('FPACK_DOWNLOAD_URL'));
    expect(text, contains('set FPACK_CORE'));
  });

  test('FPACK_REBUILD=1 builds locally even when a download works', () async {
    final go = fakeGo();
    var downloads = 0;
    final r = resolver(
      env: {...dlEnv, 'FPACK_GO': go, 'FPACK_REBUILD': '1'},
      downloader: (u, d) async => downloads++,
    );
    expect((await r.resolve()).source, CoreSource.built);
    expect(downloads, 0);
    expect(logs.join('\n'), contains('FPACK_REBUILD=1'));
  });

  test('a failed go build after a failed download reports both', () async {
    final go = fakeGo(coreVersion: '0.0.1');
    final r = resolver(env: {...dlEnv, 'FPACK_GO': go}, downloader: offline);
    await expectLater(r.resolve(), throwsA(isA<ResolveException>()));
    expect(r.attempts.join('\n'), contains('expected 1.0.0'));
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
    expect(logs.join('\n'), contains('falling back to a local build'));
    expect(logs.join('\n'), contains('FPACK_REBUILD=1'));
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

  test('httpDownload gives up when the connection stalls', () async {
    final srv = await HttpServer.bind(InternetAddress.loopbackIPv4, 0);
    srv.listen((req) async {
      req.response
        ..statusCode = 200
        ..contentLength = 100
        ..add([1, 2, 3]);
      await req.response.flush(); // then stall
    });
    final dir = Directory.systemTemp.createTempSync('fpack-dl');
    final sw = Stopwatch()..start();
    await expectLater(
      httpDownload(
        Uri.parse('http://127.0.0.1:${srv.port}/core'),
        File('${dir.path}/core'),
        environment: const {},
        stallTimeout: const Duration(milliseconds: 300),
      ),
      throwsA(isA<TimeoutException>()),
    );
    expect(sw.elapsed, lessThan(const Duration(seconds: 10)));
    await srv.close(force: true);
    dir.deleteSync(recursive: true);
  });

  test('httpDownload uses HTTP_PROXY from the environment', () async {
    final seen = <String>[];
    final proxy = await HttpServer.bind(InternetAddress.loopbackIPv4, 0);
    proxy.listen((req) async {
      seen.add('${req.uri}');
      req.response.write('via proxy');
      await req.response.close();
    });
    final dir = Directory.systemTemp.createTempSync('fpack-dl');
    final dest = File('${dir.path}/sums');
    await httpDownload(
      Uri.parse('http://downloads.example.invalid/checksums.txt'),
      dest,
      environment: {'HTTP_PROXY': 'http://127.0.0.1:${proxy.port}'},
    );
    expect(dest.readAsStringSync(), 'via proxy');
    expect(seen.single, contains('downloads.example.invalid/checksums.txt'));
    await proxy.close(force: true);
    dir.deleteSync(recursive: true);
  });
}
