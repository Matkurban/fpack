// Package version holds the single source of truth for the fpack core version.
//
// The Dart wrapper (lib/src/version.dart) and pubspec.yaml must carry the same
// value; scripts/check_versions.sh enforces this in CI.
package version

// Version is the fpack release version. Keep in sync with pubspec.yaml.
const Version = "0.1.0"

// Commit may be injected at build time with -ldflags "-X .../version.Commit=abc".
var Commit = ""
