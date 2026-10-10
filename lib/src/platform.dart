import 'dart:ffi' show Abi;
import 'dart:io' show Platform;

import 'i18n.dart';

/// The Go GOOS/GOARCH pair for the current host.
class HostTarget {
  const HostTarget(this.os, this.arch);

  /// Detects the host. Throws [UnsupportedError] for platforms without a core.
  factory HostTarget.current() => fromAbi(Abi.current());

  /// Maps a Dart [Abi] to a Go target.
  static HostTarget fromAbi(Abi abi) {
    const table = {
      Abi.macosArm64: HostTarget('darwin', 'arm64'),
      Abi.macosX64: HostTarget('darwin', 'amd64'),
      Abi.linuxX64: HostTarget('linux', 'amd64'),
      Abi.linuxArm64: HostTarget('linux', 'arm64'),
      Abi.windowsX64: HostTarget('windows', 'amd64'),
      Abi.windowsArm64: HostTarget('windows', 'arm64'),
    };
    final t = table[abi];
    if (t == null) {
      throw UnsupportedError(
        tr(
          'fpack has no native core for $abi. Supported: macOS, Linux and '
              'Windows on x64/arm64. You can build one with Go and point '
              'FPACK_CORE at it.',
          'fpack 没有适用于 $abi 的原生核心。支持：x64/arm64 上的 macOS、Linux '
              '和 Windows。可以用 Go 自行构建，并用 FPACK_CORE 指向它。',
        ),
      );
    }
    return t;
  }

  final String os;
  final String arch;

  bool get isWindows => os == 'windows';

  /// e.g. `darwin-arm64`
  String get id => '$os-$arch';

  /// File name of the core binary inside `prebuilt/<id>/` and the cache.
  String get exeName => isWindows ? 'fpack-core.exe' : 'fpack-core';

  /// Asset name in GitHub releases, e.g. `fpack-core-darwin-arm64`.
  String get assetName => 'fpack-core-$id${isWindows ? '.exe' : ''}';

  @override
  bool operator ==(Object other) =>
      other is HostTarget && other.os == os && other.arch == arch;

  @override
  int get hashCode => Object.hash(os, arch);

  @override
  String toString() => id;
}

/// Platform-specific cache root for fpack, honoring FPACK_HOME.
String defaultCacheRoot(Map<String, String> env, HostTarget host) {
  final override = env['FPACK_HOME'];
  if (override != null && override.isNotEmpty) return override;
  final home = env['HOME'] ?? env['USERPROFILE'] ?? '.';
  switch (host.os) {
    case 'darwin':
      return '$home/Library/Caches/fpack';
    case 'windows':
      final local = env['LOCALAPPDATA'];
      return local != null && local.isNotEmpty
          ? '$local\\fpack'
          : '$home\\AppData\\Local\\fpack';
    default:
      final xdg = env['XDG_CACHE_HOME'];
      final base = xdg != null && xdg.isNotEmpty ? xdg : '$home/.cache';
      return '$base/fpack';
  }
}

/// Whether the current process runs on Windows (used for path joining).
bool get hostIsWindows => Platform.isWindows;
