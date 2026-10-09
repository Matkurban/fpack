import 'dart:async';
import 'dart:io';
import 'dart:isolate';

import 'platform.dart';
import 'resolver.dart';
import 'version.dart';

/// Resolves the native core and runs it with [args], inheriting stdio.
/// Returns the exit code to use.
Future<int> runFpack(List<String> args) async {
  final HostTarget host;
  try {
    host = HostTarget.current();
  } on UnsupportedError catch (e) {
    stderr.writeln('fpack: ${e.message}');
    return 3;
  }

  final root = await packageRoot();
  final resolver = CoreResolver(packageRoot: root, host: host);

  if (args.isNotEmpty && args.first == '--wrapper-info') {
    return _wrapperInfo(resolver);
  }

  final ResolvedCore core;
  try {
    core = await resolver.resolve();
  } on ResolveException catch (e) {
    stderr.writeln(e);
    return 3;
  }
  return runCore(core.path, args);
}

/// Starts [core] and forwards SIGINT/SIGTERM until it exits.
Future<int> runCore(String core, List<String> args) async {
  final Process proc;
  try {
    proc = await Process.start(
      core,
      args,
      mode: ProcessStartMode.inheritStdio,
      environment: {'FPACK_WRAPPER_VERSION': packageVersion},
    );
  } on ProcessException catch (e) {
    stderr.writeln(
      'fpack: cannot start $core: ${e.message}\n'
      '  fix: delete the cached binary or run with FPACK_REBUILD=1',
    );
    return 3;
  }

  // A terminal Ctrl-C reaches both processes (same process group); the
  // core treats a burst as a single press. Signals sent only to this
  // process (kill, IDE stop buttons) are forwarded.
  final subs = <StreamSubscription<ProcessSignal>>[];
  void forward(ProcessSignal s) {
    subs.add(s.watch().listen((sig) => proc.kill(sig)));
  }

  forward(ProcessSignal.sigint);
  if (!Platform.isWindows) forward(ProcessSignal.sigterm);

  final code = await proc.exitCode;
  for (final s in subs) {
    await s.cancel();
  }
  // Child killed by a signal: report 128+N like a shell would.
  return code < 0 ? 128 - code : code;
}

/// Directory of this package (works for pub, git and path activation).
Future<String> packageRoot() async {
  final libUri = await Isolate.resolvePackageUri(
    Uri.parse('package:fpack/fpack.dart'),
  );
  if (libUri != null && libUri.scheme == 'file') {
    return File.fromUri(libUri).parent.parent.path;
  }
  // Fallback: compiled snapshot next to bin/.
  return File(Platform.script.toFilePath()).parent.parent.path;
}

Future<int> _wrapperInfo(CoreResolver r) async {
  final go = r.env['FPACK_GO'] == 'none' ? null : r.findGo();
  final b = StringBuffer()
    ..writeln('fpack wrapper $packageVersion')
    ..writeln('  host:      ${r.host.id}')
    ..writeln('  package:   ${r.packageRoot}')
    ..writeln(
      '  bundled:   ${r.bundledPath}'
      '${File(r.bundledPath).existsSync() ? '' : '  (missing)'}',
    )
    ..writeln(
      '  cache:     ${r.cachedPath}'
      '${File(r.cachedPath).existsSync() ? '' : '  (empty)'}',
    )
    ..writeln('  download:  ${r.downloadBase}')
    ..writeln(
      '  go:        ${go ?? '(not found)'}'
      '${go == null ? '' : ' (${await r.goVersion(go) ?? '?'}; only used '
                'when no verified prebuilt core is available)'}',
    )
    ..writeln(
      '  order:     FPACK_CORE → cache → bundled → download → local go build',
    );
  try {
    final core = await r.resolve();
    b.writeln('  using:     $core');
    b.writeln('  core:      ${await r.coreVersion(core.path) ?? '?'}');
  } on ResolveException catch (e) {
    b.writeln('  using:     (none)\n$e');
    stdout.write(b);
    return 3;
  }
  stdout.write(b);
  return 0;
}
