/// Thin launcher for the fpack native core.
///
/// The real work happens in a Go binary (`fpack-core`). This library finds
/// or provisions the binary matching this package version and runs it with
/// the user's arguments, inheriting stdio.
library;

export 'src/launcher.dart' show runFpack;
export 'src/platform.dart' show HostTarget;
export 'src/resolver.dart' show CoreResolver, ResolvedCore, ResolveException;
export 'src/sha256.dart' show sha256Hex, sha256OfFile;
export 'src/version.dart' show packageVersion;
