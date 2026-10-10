import 'dart:async';
import 'dart:convert';
import 'dart:io';

import 'i18n.dart';
import 'platform.dart';
import 'sha256.dart';
import 'version.dart';

/// Where a core binary came from.
enum CoreSource { env, cache, bundled, built, downloaded }

/// A usable core binary.
class ResolvedCore {
  ResolvedCore(this.path, this.source);
  final String path;
  final CoreSource source;
  @override
  String toString() => '$path (${source.name})';
}

/// No core binary could be found or provisioned.
class ResolveException implements Exception {
  ResolveException(this.message, this.attempts);
  final String message;
  final List<String> attempts;
  @override
  String toString() => '$message\n${attempts.map((a) => '  - $a').join('\n')}';
}

/// Signature of a function that downloads [url] into [dest].
typedef Downloader = Future<void> Function(Uri url, File dest);

/// Finds or provisions the `fpack-core` binary for this package version.
///
/// Order (verified prebuilt binaries first, a local build only as the
/// last resort):
///   1. `FPACK_CORE` (explicit path, for development)
///   2. the per-user cache (`<cache>/<version>/<os>-<arch>/fpack-core`),
///      filled by one of the steps below on first run
///   3. the prebuilt binary bundled in `prebuilt/<os>-<arch>/`, verified
///      against `prebuilt/manifest.json` and copied into the cache
///   4. download from the GitHub release, verified against `checksums.txt`
///      (skipped with `FPACK_NO_DOWNLOAD=1`)
///   5. only when 3 and 4 are unavailable: `go build` from the bundled Go
///      sources with a local Go toolchain (`FPACK_GO=<path>` picks it,
///      `FPACK_GO=none` disables it), with a note naming the Go version
///
/// `FPACK_REBUILD=1` forces step 5 first (development).
///
/// Every candidate must report exactly [packageVersion] via
/// `fpack-core --core-version`, so the wrapper and core never drift.
class CoreResolver {
  CoreResolver({
    required this.packageRoot,
    required this.host,
    Map<String, String>? environment,
    String? cacheRoot,
    this.version = packageVersion,
    Downloader? downloader,
    this.log = _stderrLog,
  }) : env = environment ?? Platform.environment,
       _downloader =
           downloader ??
           ((url, dest) => httpDownload(
             url,
             dest,
             environment: environment ?? Platform.environment,
           )) {
    this.cacheRoot = cacheRoot ?? defaultCacheRoot(env, host);
  }

  final String packageRoot;
  final HostTarget host;
  final Map<String, String> env;
  final String version;
  final void Function(String) log;
  final Downloader _downloader;
  late final String cacheRoot;

  final List<String> attempts = [];

  String get _sep => host.isWindows ? '\\' : '/';
  String _join(List<String> parts) => parts.join(_sep);

  /// Cached binary location for this version and host.
  String get cachedPath => _join([cacheRoot, version, host.id, host.exeName]);
  String get _stampPath => '$cachedPath.stamp';
  String get bundledPath =>
      _join([packageRoot, 'prebuilt', host.id, host.exeName]);
  String get manifestPath => _join([packageRoot, 'prebuilt', 'manifest.json']);
  String get goModuleDir => _join([packageRoot, 'go']);

  bool _flag(String name) {
    final v = (env[name] ?? '').toLowerCase();
    return v == '1' || v == 'true' || v == 'yes';
  }

  Future<ResolvedCore> resolve() async {
    final explicit = env['FPACK_CORE'];
    if (explicit != null && explicit.isNotEmpty) {
      if (!File(explicit).existsSync()) {
        throw ResolveException(
          tr(
            'FPACK_CORE points to a missing file: $explicit',
            'FPACK_CORE 指向的文件不存在：$explicit',
          ),
          [
            tr(
              'set FPACK_CORE to a built fpack-core binary, or unset it',
              '将 FPACK_CORE 设为已构建的 fpack-core 二进制文件，或取消该变量',
            ),
          ],
        );
      }
      final v = await coreVersion(explicit);
      if (v != version) {
        log(
          tr(
            'fpack: warning: FPACK_CORE=$explicit reports version '
                '${v ?? "unknown"}, wrapper is $version',
            'fpack：警告：FPACK_CORE=$explicit 的版本为 '
                '${v ?? "未知"}，包装器版本为 $version',
          ),
        );
      }
      return ResolvedCore(explicit, CoreSource.env);
    }

    final rebuild = _flag('FPACK_REBUILD');
    if (rebuild) {
      final g = await _fromGo(reason: 'FPACK_REBUILD=1');
      if (g != null) return g;
    } else if (await _cacheValid()) {
      return ResolvedCore(cachedPath, CoreSource.cache);
    }
    final b = await _fromBundled();
    if (b != null) return b;
    final d = await _fromDownload();
    if (d != null) return d;
    if (!rebuild) {
      final g = await _fromGo(reason: _downloadFailure ?? 'download failed');
      if (g != null) return g;
    }
    throw ResolveException(
      tr(
        'fpack: could not find, download or build the native core for '
            '${host.id} (version $version).',
        'fpack：无法找到、下载或构建 ${host.id} 的原生核心（版本 $version）。',
      ),
      [
        ...attempts,
        tr(
              'fix: check the network (or set FPACK_DOWNLOAD_URL to a mirror of '
                  'the GitHub release), or install Go (https://go.dev/dl) for a '
                  'local build, or download fpack-core-${host.id} from '
                  'https://github.com/Matkurban/fpack/releases/tag/v$version '
                  'and set FPACK_CORE',
              '修复：检查网络（或把 FPACK_DOWNLOAD_URL 设为 GitHub Release 的镜像），'
                  '或安装 Go（https://go.dev/dl）在本地构建，或从 '
                  'https://github.com/Matkurban/fpack/releases/tag/v$version '
                  '下载 fpack-core-${host.id} 并设置 FPACK_CORE',
            ),
      ],
    );
  }

  /// Why the download step did not produce a core (for the fallback note).
  String? _downloadFailure;

  // ------------------------------------------------------------------ cache

  /// The cache entry is valid when its stamp matches size + mtime, which
  /// avoids re-hashing on every launch.
  Future<bool> _cacheValid() async {
    final f = File(cachedPath);
    final stamp = File(_stampPath);
    if (!f.existsSync() || !stamp.existsSync()) return false;
    try {
      final data = jsonDecode(await stamp.readAsString()) as Map;
      final st = f.statSync();
      if (data['version'] != version ||
          data['size'] != st.size ||
          data['mtime'] != st.modified.millisecondsSinceEpoch) {
        return false;
      }
      // A reinstalled package may bundle a different binary under the same
      // version (e.g. a rebuilt archive): follow the manifest when present.
      final bundled = _manifestSha();
      // (Compared with the manifest seen at install time, so a core built
      // with Go because the bundled one could not run stays valid.)
      return bundled == null || bundled == data['manifest'];
    } catch (_) {
      return false;
    }
  }

  Future<void> _writeStamp(String source, String sha) async {
    final st = File(cachedPath).statSync();
    await File(_stampPath).writeAsString(
      jsonEncode({
        'version': version,
        'source': source,
        'sha256': sha,
        'size': st.size,
        'mtime': st.modified.millisecondsSinceEpoch,
        'manifest': _manifestSha(),
      }),
    );
  }

  /// Atomically installs [src] into the cache (copy to temp, chmod, rename).
  Future<String?> _install(
    File src,
    String source,
    String sha, {
    bool move = false,
  }) async {
    try {
      Directory(File(cachedPath).parent.path).createSync(recursive: true);
      final tmp = '$cachedPath.tmp$pid';
      if (move) {
        await src.rename(tmp);
      } else {
        await src.copy(tmp);
      }
      await _chmodX(tmp);
      final dst = File(cachedPath);
      if (dst.existsSync()) dst.deleteSync();
      await File(tmp).rename(cachedPath);
      await _writeStamp(source, sha);
      return cachedPath;
    } on FileSystemException catch (e) {
      attempts.add(
        'cache: cannot write ${File(cachedPath).parent.path}: '
        '${e.message} (set FPACK_HOME to a writable directory)',
      );
      return null;
    }
  }

  Future<void> _chmodX(String path) async {
    if (host.isWindows) return;
    final r = await Process.run('chmod', ['755', path]);
    if (r.exitCode != 0) {
      throw FileSystemException('chmod failed: ${r.stderr}', path);
    }
    if (host.os == 'darwin') {
      // A browser-downloaded archive marks extracted files as quarantined
      // and File.copy keeps that attribute; Gatekeeper would then refuse to
      // run the (ad-hoc signed) core. The binary was verified by SHA-256.
      try {
        await Process.run('xattr', ['-d', 'com.apple.quarantine', path]);
      } catch (_) {}
    }
  }

  // ---------------------------------------------------------------- bundled

  /// SHA-256 of the bundled binary for this host according to the
  /// manifest, or null when there is no usable manifest/binary.
  String? _manifestSha() {
    try {
      if (!File(bundledPath).existsSync()) return null;
      final m = jsonDecode(File(manifestPath).readAsStringSync()) as Map;
      if (m['version'] != version) return null;
      return ((m['binaries'] as Map?)?[host.id] as Map?)?['sha256'] as String?;
    } catch (_) {
      return null;
    }
  }

  Future<ResolvedCore?> _fromBundled() async {
    final bin = File(bundledPath);
    if (!bin.existsSync()) {
      attempts.add('bundled: no prebuilt binary at $bundledPath');
      return null;
    }
    final mf = File(manifestPath);
    if (!mf.existsSync()) {
      attempts.add('bundled: $manifestPath missing');
      return null;
    }
    Map manifest;
    try {
      manifest = jsonDecode(mf.readAsStringSync()) as Map;
    } catch (e) {
      attempts.add('bundled: invalid manifest.json ($e)');
      return null;
    }
    if (manifest['version'] != version) {
      attempts.add(
        'bundled: prebuilt version ${manifest['version']} != '
        'wrapper $version (rebuild with scripts/build_binaries.sh)',
      );
      return null;
    }
    final entry = (manifest['binaries'] as Map?)?[host.id] as Map?;
    final want = entry?['sha256'] as String?;
    final got = await sha256OfFile(bin);
    if (want == null || want != got) {
      attempts.add('bundled: checksum mismatch for $bundledPath');
      return null;
    }
    final installed = await _install(bin, 'bundled', got);
    if (installed != null && await coreVersion(installed) == version) {
      return ResolvedCore(installed, CoreSource.bundled);
    }
    // Cache not writable: run the bundled file in place if it is executable.
    if (installed == null && await coreVersion(bin.path) == version) {
      return ResolvedCore(bin.path, CoreSource.bundled);
    }
    attempts.add('bundled: binary does not run on this machine');
    return null;
  }

  // --------------------------------------------------------------- go build

  /// Locates a Go toolchain: FPACK_GO, PATH, then common install locations.
  String? findGo() {
    final explicit = env['FPACK_GO'];
    if (explicit != null && explicit.isNotEmpty) {
      return explicit == 'none' ? null : explicit;
    }
    final exe = host.isWindows ? 'go.exe' : 'go';
    final pathSep = host.isWindows ? ';' : ':';
    for (final dir in (env['PATH'] ?? '').split(pathSep)) {
      if (dir.isEmpty) continue;
      final p = _join([dir, exe]);
      if (File(p).existsSync()) return p;
    }
    final home = env['HOME'] ?? env['USERPROFILE'] ?? '';
    final candidates = host.isWindows
        ? [r'C:\Program Files\Go\bin\go.exe', '$home\\go\\bin\\go.exe']
        : [
            '/usr/local/go/bin/go',
            '/opt/homebrew/bin/go',
            '/usr/local/bin/go',
            '/usr/lib/go/bin/go',
            '$home/go/bin/go',
            '$home/sdk/go/bin/go',
          ];
    for (final p in candidates) {
      if (File(p).existsSync()) return p;
    }
    return null;
  }

  /// `go env GOVERSION` of [go] (e.g. go1.27.2), or null.
  Future<String?> goVersion(String go) async {
    try {
      final r = await Process.run(go, [
        'env',
        'GOVERSION',
      ]).timeout(const Duration(seconds: 20));
      final v = '${r.stdout}'.trim();
      return r.exitCode == 0 && v.isNotEmpty ? v : null;
    } catch (_) {
      return null;
    }
  }

  Future<ResolvedCore?> _fromGo({required String reason}) async {
    if (!File(_join([goModuleDir, 'go.mod'])).existsSync()) {
      attempts.add('go build: Go sources not found in $goModuleDir');
      return null;
    }
    if (env['FPACK_GO'] == 'none') {
      attempts.add('go build: disabled by FPACK_GO=none');
      return null;
    }
    final go = findGo();
    if (go == null) {
      attempts.add(
        'go build: no Go toolchain found (PATH, /usr/local/go/bin, '
        '/opt/homebrew/bin; or set FPACK_GO)',
      );
      return null;
    }
    final gv = await goVersion(go) ?? 'unknown Go version';
    log(
      tr(
        'fpack: note: no verified prebuilt core available ($reason);\n'
            'fpack: falling back to a local build from the bundled sources with '
            '$gv ($go) – first run only, ~30s…',
        'fpack：提示：没有可用的已校验预编译核心（$reason）；\n'
            'fpack：改用 $gv（$go）从随包源码本地构建——仅首次运行，约 30 秒…',
      ),
    );
    final out = File(cachedPath).parent;
    try {
      out.createSync(recursive: true);
    } on FileSystemException catch (e) {
      attempts.add('go build: cannot create ${out.path}: ${e.message}');
      return null;
    }
    final tmp = '$cachedPath.build$pid';
    final hasVendor = Directory(_join([goModuleDir, 'vendor'])).existsSync();
    final r = await Process.run(
      go,
      [
        'build',
        '-trimpath',
        if (hasVendor) '-mod=vendor',
        '-ldflags',
        '-s -w',
        '-o',
        tmp,
        './cmd/fpack-core',
      ],
      workingDirectory: goModuleDir,
      environment: {
        'CGO_ENABLED': '0',
        'GOFLAGS': '',
        if ((env['GOTOOLCHAIN'] ?? '').isEmpty) 'GOTOOLCHAIN': 'local',
      },
    );
    if (r.exitCode != 0) {
      final err = '${r.stderr}'.trim().split('\n').take(8).join('\n    ');
      attempts.add('go build failed (exit ${r.exitCode}):\n    $err');
      try {
        File(tmp).deleteSync();
      } catch (_) {}
      return null;
    }
    final sha = await sha256OfFile(File(tmp));
    final installed = await _install(File(tmp), 'go build', sha, move: true);
    if (installed == null) return null;
    final v = await coreVersion(installed);
    if (v != version) {
      attempts.add(
        'go build: built core reports ${v ?? "nothing"}, '
        'expected $version',
      );
      return null;
    }
    log(
      tr(
        'fpack: built the native core locally with $gv.',
        'fpack：已用 $gv 在本地构建原生核心。',
      ),
    );
    return ResolvedCore(installed, CoreSource.built);
  }

  // --------------------------------------------------------------- download

  Uri get downloadBase {
    final o = env['FPACK_DOWNLOAD_URL'];
    if (o != null && o.isNotEmpty) {
      return Uri.parse(o.endsWith('/') ? o : '$o/');
    }
    return Uri.parse(
      'https://github.com/Matkurban/fpack/releases/download/v$version/',
    );
  }

  Future<ResolvedCore?> _fromDownload() async {
    void fail(String why) {
      attempts.add('download: $why');
      _downloadFailure = 'download: $why';
    }

    if (_flag('FPACK_NO_DOWNLOAD')) {
      fail('disabled by FPACK_NO_DOWNLOAD');
      return null;
    }
    final base = downloadBase;
    final dir = File(cachedPath).parent;
    final tmp = File('$cachedPath.download$pid');
    final sums = File('$cachedPath.checksums$pid');
    try {
      dir.createSync(recursive: true);
      log(
        tr(
          'fpack: downloading the native core from $base …',
          'fpack：正在从 $base 下载原生核心…',
        ),
      );
      await _downloader(base.resolve('checksums.txt'), sums);
      await _downloader(base.resolve(host.assetName), tmp);
      final want = parseChecksums(sums.readAsStringSync())[host.assetName];
      final got = await sha256OfFile(tmp);
      if (want == null) {
        fail('${host.assetName} missing from checksums.txt');
        return null;
      }
      if (want != got) {
        fail(
          'SHA-256 mismatch for ${host.assetName} '
          '(expected $want, got $got) – refusing to run it',
        );
        return null;
      }
      final installed = await _install(tmp, 'download', got, move: true);
      if (installed == null) return null;
      if (await coreVersion(installed) != version) {
        fail('core does not report version $version');
        return null;
      }
      return ResolvedCore(installed, CoreSource.downloaded);
    } catch (e) {
      fail('$e');
      return null;
    } finally {
      for (final f in [tmp, sums]) {
        try {
          if (f.existsSync()) f.deleteSync();
        } catch (_) {}
      }
    }
  }

  /// Runs `<path> --core-version`; null when it can't run.
  Future<String?> coreVersion(String path) async {
    try {
      final r = await Process.run(path, [
        '--core-version',
      ]).timeout(const Duration(seconds: 20));
      if (r.exitCode != 0) return null;
      return '${r.stdout}'.trim();
    } catch (_) {
      return null;
    }
  }
}

/// Parses `sha256sum`/`shasum -a 256` output into {file: hash}.
Map<String, String> parseChecksums(String text) {
  final out = <String, String>{};
  for (final line in const LineSplitter().convert(text)) {
    final m = RegExp(r'^([0-9a-fA-F]{64})\s+\*?(.+)$').firstMatch(line.trim());
    if (m != null) out[m.group(2)!.trim()] = m.group(1)!.toLowerCase();
  }
  return out;
}

void _stderrLog(String s) => stderr.writeln(s);

/// Downloads [url] to [dest]. Honors HTTPS_PROXY / HTTP_PROXY / NO_PROXY
/// from [environment] (dart:io does not by default) and gives up when the
/// connection stalls for [stallTimeout], so a blocked or throttled GitHub
/// download falls through to the next resolution step instead of hanging.
Future<void> httpDownload(
  Uri url,
  File dest, {
  Map<String, String>? environment,
  Duration stallTimeout = const Duration(seconds: 30),
}) async {
  final env = environment ?? Platform.environment;
  final client = HttpClient()
    ..connectionTimeout = const Duration(seconds: 20)
    ..findProxy = (uri) =>
        HttpClient.findProxyFromEnvironment(uri, environment: env);
  try {
    final req = await client.getUrl(url).timeout(stallTimeout);
    req.followRedirects = true;
    req.maxRedirects = 10;
    final res = await req.close().timeout(stallTimeout);
    if (res.statusCode != 200) {
      await res.drain<void>();
      throw HttpException('HTTP ${res.statusCode}', uri: url);
    }
    final out = dest.openSync(mode: FileMode.write);
    try {
      // Stream.timeout fires when no chunk arrives within stallTimeout.
      await for (final chunk in res.timeout(stallTimeout)) {
        out.writeFromSync(chunk);
      }
    } on TimeoutException {
      throw TimeoutException(
        'no data for ${stallTimeout.inSeconds}s from ${url.host} '
        '(connection stalled)',
      );
    } finally {
      out.closeSync();
    }
  } finally {
    client.close(force: true);
  }
}
