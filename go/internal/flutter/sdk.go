// Package flutter locates the Flutter SDK: --flutter flag, FPACK_FLUTTER,
// fpack.yaml flutter.sdk, FVM (.fvm/flutter_sdk or .fvmrc), FLUTTER_ROOT,
// PATH, then common install locations.
package flutter

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/Matkurban/fpack/go/internal/runner"
)

// SDK is a located Flutter SDK.
type SDK struct {
	Root        string
	Flutter     string // flutter executable
	Dart        string // dart executable
	Version     string // 3.47.6
	Channel     string
	DartVersion string
	Source      string // how it was found
}

// Candidate is an explicit SDK location with a label for messages.
type Candidate struct {
	Path   string
	Source string
}

// Locate finds the SDK. explicit candidates (flag, env, config) are tried in
// order first; an explicit path that is not a Flutter SDK is an error rather
// than silently falling back.
func Locate(projectRoot string, explicit []Candidate) (*SDK, error) {
	for _, c := range explicit {
		if c.Path == "" {
			continue
		}
		root := normalizeRoot(expandHome(c.Path))
		if root == "" {
			return nil, &NotSDKError{Path: c.Path, Source: c.Source}
		}
		return load(root, c.Source)
	}
	if root, src := fvm(projectRoot); root != "" {
		return load(root, src)
	}
	if r := os.Getenv("FLUTTER_ROOT"); r != "" {
		if root := normalizeRoot(r); root != "" {
			return load(root, "FLUTTER_ROOT")
		}
	}
	if p := runner.Which("flutter"); p != "" {
		if real, err := filepath.EvalSymlinks(p); err == nil {
			p = real
		}
		if root := normalizeRoot(filepath.Dir(filepath.Dir(p))); root != "" {
			return load(root, "PATH")
		}
	}
	for _, c := range commonLocations() {
		if root := normalizeRoot(c); root != "" {
			return load(root, c)
		}
	}
	return nil, ErrNotFound
}

// ErrNotFound means no SDK could be located.
var ErrNotFound = errors.New("Flutter SDK not found")

// NotSDKError is an explicit path that is not a Flutter SDK.
type NotSDKError struct{ Path, Source string }

func (e *NotSDKError) Error() string {
	return e.Source + ": " + e.Path + " is not a Flutter SDK (expected bin/flutter inside)"
}

func expandHome(p string) string {
	if strings.HasPrefix(p, "~") {
		if h, err := os.UserHomeDir(); err == nil {
			return filepath.Join(h, p[1:])
		}
	}
	return p
}

// normalizeRoot accepts an SDK root or its bin/ dir or the flutter binary.
func normalizeRoot(p string) string {
	if p == "" {
		return ""
	}
	if real, err := filepath.EvalSymlinks(p); err == nil {
		p = real
	}
	for _, cand := range []string{p, filepath.Dir(p), filepath.Dir(filepath.Dir(p))} {
		if isSDK(cand) {
			abs, _ := filepath.Abs(cand)
			return abs
		}
	}
	return ""
}

func isSDK(root string) bool {
	_, err := os.Stat(filepath.Join(root, "bin", flutterExe()))
	if err != nil {
		return false
	}
	_, err = os.Stat(filepath.Join(root, "packages", "flutter_tools"))
	return err == nil
}

func flutterExe() string {
	if runtime.GOOS == "windows" {
		return "flutter.bat"
	}
	return "flutter"
}

func dartExe() string {
	if runtime.GOOS == "windows" {
		return "dart.bat"
	}
	return "dart"
}

func fvm(projectRoot string) (string, string) {
	for d := projectRoot; d != "" && d != filepath.Dir(d); d = filepath.Dir(d) {
		link := filepath.Join(d, ".fvm", "flutter_sdk")
		if root := normalizeRoot(link); root != "" {
			return root, "FVM (" + link + ")"
		}
		// fvm 3: .fvmrc {"flutter": "3.24.0"}
		if data, err := os.ReadFile(filepath.Join(d, ".fvmrc")); err == nil {
			var rc struct {
				Flutter string `json:"flutter"`
			}
			if json.Unmarshal(data, &rc) == nil && rc.Flutter != "" {
				for _, base := range fvmCacheDirs() {
					if root := normalizeRoot(filepath.Join(base, rc.Flutter)); root != "" {
						return root, "FVM (.fvmrc " + rc.Flutter + ")"
					}
				}
			}
		}
	}
	return "", ""
}

func fvmCacheDirs() []string {
	var out []string
	if c := os.Getenv("FVM_CACHE_PATH"); c != "" {
		out = append(out, filepath.Join(c, "versions"))
	}
	if h, err := os.UserHomeDir(); err == nil {
		out = append(out, filepath.Join(h, "fvm", "versions"))
		if runtime.GOOS == "windows" {
			out = append(out, filepath.Join(os.Getenv("LOCALAPPDATA"), "fvm", "versions"))
		}
	}
	return out
}

func commonLocations() []string {
	h, _ := os.UserHomeDir()
	locs := []string{
		filepath.Join(h, "flutter"),
		filepath.Join(h, "develop", "flutter"),
		filepath.Join(h, "development", "flutter"),
		filepath.Join(h, "dev", "flutter"),
		filepath.Join(h, "sdk", "flutter"),
		filepath.Join(h, "fvm", "default"),
		filepath.Join(h, "snap", "flutter", "common", "flutter"),
	}
	switch runtime.GOOS {
	case "darwin":
		locs = append(locs, "/opt/homebrew/share/flutter", "/usr/local/share/flutter", "/Applications/flutter")
	case "linux":
		locs = append(locs, "/opt/flutter", "/usr/local/flutter", "/usr/lib/flutter")
	case "windows":
		locs = append(locs, `C:\flutter`, `C:\src\flutter`, `C:\tools\flutter`, filepath.Join(os.Getenv("LOCALAPPDATA"), "flutter"))
	}
	return locs
}

func load(root, source string) (*SDK, error) {
	s := &SDK{Root: root, Source: source,
		Flutter: filepath.Join(root, "bin", flutterExe()),
		Dart:    filepath.Join(root, "bin", dartExe())}
	s.readVersion()
	return s, nil
}

func (s *SDK) readVersion() {
	// Fast path: the version file written by the tool (no process spawn).
	if data, err := os.ReadFile(filepath.Join(s.Root, "bin", "cache", "flutter.version.json")); err == nil {
		var v struct {
			FrameworkVersion string `json:"frameworkVersion"`
			FlutterVersion   string `json:"flutterVersion"`
			Channel          string `json:"channel"`
			DartSdkVersion   string `json:"dartSdkVersion"`
		}
		if json.Unmarshal(data, &v) == nil {
			s.Version = firstNonEmpty(v.FlutterVersion, v.FrameworkVersion)
			s.Channel = v.Channel
			s.DartVersion = strings.Fields(v.DartSdkVersion + " ")[0]
			if s.Version != "" {
				return
			}
		}
	}
	if data, err := os.ReadFile(filepath.Join(s.Root, "version")); err == nil {
		s.Version = strings.TrimSpace(string(data))
	}
	if data, err := os.ReadFile(filepath.Join(s.Root, "bin", "cache", "dart-sdk", "version")); err == nil {
		s.DartVersion = strings.TrimSpace(string(data))
	}
}

// QueryVersion asks the tool itself (slow; used when the fast path failed).
func (s *SDK) QueryVersion() {
	if s.Version != "" {
		return
	}
	out, ok := runner.Output(90*time.Second, s.Flutter, "--version", "--machine")
	if !ok {
		return
	}
	if i := strings.Index(out, "{"); i >= 0 {
		var v map[string]any
		if json.Unmarshal([]byte(out[i:]), &v) == nil {
			s.Version, _ = v["frameworkVersion"].(string)
			s.Channel, _ = v["channel"].(string)
			if d, ok := v["dartSdkVersion"].(string); ok {
				s.DartVersion = strings.Fields(d + " ")[0]
			}
		}
	}
}

func firstNonEmpty(v ...string) string {
	for _, s := range v {
		if s != "" {
			return s
		}
	}
	return ""
}

var verRe = regexp.MustCompile(`^(\d+)\.(\d+)\.(\d+)`)

// AtLeast reports whether the SDK version is >= major.minor.
func (s *SDK) AtLeast(major, minor int) bool {
	m := verRe.FindStringSubmatch(s.Version)
	if m == nil {
		return true // unknown: don't block
	}
	var a, b int
	for _, c := range m[1] {
		a = a*10 + int(c-'0')
	}
	for _, c := range m[2] {
		b = b*10 + int(c-'0')
	}
	return a > major || (a == major && b >= minor)
}

// Settings returns Flutter's tool settings (flutter config values such as
// android-sdk and jdk-dir).
func Settings() map[string]string {
	out := map[string]string{}
	h, err := os.UserHomeDir()
	if err != nil {
		return out
	}
	paths := []string{filepath.Join(h, ".flutter_settings")}
	if x := os.Getenv("XDG_CONFIG_HOME"); x != "" {
		paths = append(paths, filepath.Join(x, "flutter", "settings"))
	}
	paths = append(paths, filepath.Join(h, ".config", "flutter", "settings"))
	if runtime.GOOS == "windows" {
		paths = append(paths, filepath.Join(os.Getenv("APPDATA"), ".flutter_settings"), filepath.Join(os.Getenv("APPDATA"), "flutter", "settings"))
	}
	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		var m map[string]any
		if json.Unmarshal(data, &m) == nil {
			for k, v := range m {
				if s, ok := v.(string); ok {
					out[k] = s
				}
			}
		}
	}
	return out
}
