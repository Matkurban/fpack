// Package targets implements one packaging target per output format (apk,
// aab, ipa, macos, dmg, pkg, windows, exe, msix, linux, deb, rpm, appimage, web)
// behind a common interface.
//
// A target is planned in three phases so that --dry-run shows exactly what a
// real run would do:
//
//  1. Steps:   the `flutter build …` invocations it needs (shared by key, so
//     macos+dmg+pkg or linux+deb+rpm run Flutter only once).
//  2. Locate:  where Flutter put its output (predicted in dry-run).
//  3. Package: a list of Ops (external commands or internal file actions)
//     producing the final artifacts in the output directory.
package targets

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Matkurban/fpack/go/internal/config"
	"github.com/Matkurban/fpack/go/internal/flutter"
	"github.com/Matkurban/fpack/go/internal/host"
	"github.com/Matkurban/fpack/go/internal/i18n"
	"github.com/Matkurban/fpack/go/internal/pack"
	"github.com/Matkurban/fpack/go/internal/project"
	"github.com/Matkurban/fpack/go/internal/runner"
)

// Target is one output format.
type Target interface {
	Name() string
	Platform() host.Platform
	// Formats lists produced file extensions (for `fpack list`).
	Formats() []string
	// Description is a one-line human description.
	Description() string
	// Optional targets need extra tools; `--all` skips them quietly when the
	// tools are missing instead of failing.
	Optional() bool
	// Preflight checks prerequisites that would make the build fail.
	Preflight(c *Context) []Issue
	// Steps returns the flutter build invocations required.
	Steps(c *Context) ([]FlutterStep, error)
	// Locate finds Flutter's outputs; predicted=true returns expected paths
	// without touching the disk (dry-run, conflict checks).
	Locate(c *Context, predicted bool, since time.Time) (Inputs, error)
	// Package plans the operations that produce final artifacts.
	Package(c *Context, in Inputs) (*Plan, error)
}

// FlutterStep is one `flutter build` invocation.
type FlutterStep struct {
	Key      string // dedupe key: targets with the same key share the build
	Platform host.Platform
	Args     []string // arguments after the flutter executable
	Env      []string
	Secret   []string
	Warnings []string
	// Prepare runs before the flutter build (e.g. writing a generated
	// ExportOptions.plist), After runs once it succeeded (e.g. signing the
	// runner executable). Both are listed in dry runs.
	Prepare []Op
	After   []Op
}

// Inputs are paths located after the Flutter build.
type Inputs map[string]string

// Op is one packaging operation: an external command or an internal action.
type Op struct {
	Desc string
	Cmd  *runner.Cmd
	Fn   func() error
	// Check inspects a command's result; a returned note is attached to the
	// artifacts, an error fails the op.
	Check func(res runner.Result) (note string, err error)
	// Optional ops only warn on failure (e.g. signature verification).
	Optional bool
	// Hint is shown if the op fails.
	Hint string
}

// Plan is the packaging plan of a target.
type Plan struct {
	Ops       []Op
	Artifacts []Artifact
	Warnings  []string
	Notes     []string
}

// Artifact is a final file in the output directory.
type Artifact struct {
	Path    string `json:"path"`
	Kind    string `json:"kind"`
	Arch    string `json:"arch,omitempty"`
	Variant string `json:"variant,omitempty"`
}

// Issue is a preflight finding.
type Issue struct {
	Fatal bool
	Msg   string
	Fix   string
	// IfFails: a warning that only matters if the build then fails (e.g. a
	// certificate missing locally that Xcode may still fetch through
	// cloud-managed signing). Real builds show it only on failure;
	// dry runs and doctor show it up front.
	IfFails bool
}

func warnIfFails(msg, fix string) Issue { return Issue{Msg: msg, Fix: fix, IfFails: true} }

func fatal(msg, fix string) Issue { return Issue{Fatal: true, Msg: msg, Fix: fix} }
func warn(msg, fix string) Issue  { return Issue{Msg: msg, Fix: fix} }

// Tools finds executables and runs quick probes. Tests use a fake.
type Tools interface {
	Find(name string) string
	Probe(name string, args ...string) (string, bool)
	// ProbeEnv is Probe with extra environment variables (secrets are passed
	// this way, never on the command line).
	ProbeEnv(env []string, name string, args ...string) (string, bool)
}

// SystemTools is the real Tools implementation.
type SystemTools struct{}

// Find looks in PATH and well-known install locations.
func (SystemTools) Find(name string) string {
	if p := runner.Which(name); p != "" {
		return p
	}
	for _, p := range knownLocations(name) {
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p
		}
	}
	return ""
}

// Probe runs a short command (10s timeout).
func (SystemTools) Probe(name string, args ...string) (string, bool) {
	return runner.Output(20*time.Second, name, args...)
}

// ProbeEnv runs a short command with extra environment variables.
func (SystemTools) ProbeEnv(env []string, name string, args ...string) (string, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	c := exec.CommandContext(ctx, name, args...)
	c.Env = append(os.Environ(), env...)
	out, err := c.CombinedOutput()
	return strings.TrimSpace(string(out)), err == nil
}

// Context carries everything a target needs.
type Context struct {
	Project *project.Project
	Config  *config.Config
	SDK     *flutter.SDK
	Host    host.Host
	Tools   Tools

	AppName     string // artifact base name
	BuildName   string // effective version name
	BuildNumber string // effective build number
	OutDir      string // absolute output directory
	WorkDir     string // absolute build/fpack directory
	Interactive bool
	DryRun      bool
	// PassArgs are extra flutter args given after `--`.
	PassArgs []string

	Mac     MacSigning
	Signing AndroidSigning

	// Started is the build start (for the {date} placeholder).
	Started time.Time
	// CurrentTarget is the target being planned (set by PackageFor).
	CurrentTarget string

	androidEnv *AndroidEnv
}

// Flavor returns the configured flavor.
func (c *Context) Flavor() string { return c.Config.Build.Flavor }

// Mode returns release/profile/debug.
func (c *Context) Mode() string { return c.Config.Mode() }

// DisplayName returns the human application name.
func (c *Context) DisplayName() string {
	if c.Config.App.DisplayName != "" {
		return c.Config.App.DisplayName
	}
	return c.Project.Name
}

// Stage returns (without creating) a scratch directory for a target.
func (c *Context) Stage(name string) string { return filepath.Join(c.WorkDir, "stage", name) }

// Rel shows a path relative to the project root when possible.
func (c *Context) Rel(p string) string {
	if r, err := filepath.Rel(c.Project.Root, p); err == nil && !strings.HasPrefix(r, "..") {
		return r
	}
	return p
}

// ArtifactPath returns the final path for an artifact.
func (c *Context) ArtifactPath(p host.Platform, arch, variant, ext string) (string, error) {
	tmpl, key := c.Config.NameTemplate(), "output.name"
	if n := c.Config.Output.Names[c.CurrentTarget]; n != "" {
		tmpl, key = n, "output.names."+c.CurrentTarget
	}
	started := c.Started
	if started.IsZero() {
		started = time.Now()
	}
	name, err := pack.Render(tmpl, pack.Fields{
		App: c.AppName, Version: c.BuildName, Build: c.BuildNumber, Platform: string(p),
		Arch: arch, Variant: variant, Mode: c.Mode(), Flavor: c.Flavor(),
		Target: c.CurrentTarget, Date: started.Format("20060102"),
	})
	if err != nil {
		return "", fmt.Errorf("%s: %w", key, err)
	}
	return filepath.Join(c.OutDir, name+ext), nil
}

// Android returns lazily detected Android SDK / Java info.
func (c *Context) Android() *AndroidEnv {
	if c.androidEnv == nil {
		c.androidEnv = DetectAndroidEnv(c.Tools)
	}
	return c.androidEnv
}

// SetAndroidEnv injects Android info (tests).
func (c *Context) SetAndroidEnv(e *AndroidEnv) { c.androidEnv = e }

// ---- registry ----

var registry = []Target{
	&APK{}, &AAB{}, &IPA{}, &MacApp{}, &DMG{}, &Pkg{}, &WinZip{}, &WinExe{}, &Msix{},
	&LinuxTar{}, &Deb{}, &Rpm{}, &AppImage{}, &WebZip{},
}

// All returns every target in display order.
func All() []Target { return registry }

// Names returns all target names.
func Names() []string {
	var n []string
	for _, t := range registry {
		n = append(n, t.Name())
	}
	return n
}

var aliases = map[string]string{
	"appbundle": "aab", "bundle": "aab", "android": "apk",
	"ios": "ipa", "mac": "macos", "app": "macos", "osx": "macos",
	"win": "windows", "zip": "windows", "portable": "windows", "setup": "exe", "installer": "exe", "inno": "exe",
	"tar": "linux", "tgz": "linux", "tar.gz": "linux", "debian": "deb", "fedora": "rpm",
	"image": "appimage", "AppImage": "appimage",
}

// Get returns a target by name or alias.
func Get(name string) (Target, bool) {
	n := strings.ToLower(strings.TrimSpace(name))
	if a, ok := aliases[n]; ok {
		n = a
	}
	for _, t := range registry {
		if t.Name() == n {
			return t, true
		}
	}
	return nil, false
}

// ForPlatform returns the targets of a platform.
func ForPlatform(p host.Platform) []Target {
	var out []Target
	for _, t := range registry {
		if t.Platform() == p {
			out = append(out, t)
		}
	}
	return out
}

// ---- shared flutter args ----

// supportsFlavor reports platforms where `flutter build --flavor` works.
func supportsFlavor(p host.Platform) bool {
	return p == host.Android || p == host.IOS || p == host.MacOS
}

// CommonArgs builds `flutter build <sub>` arguments shared by all platforms.
func CommonArgs(c *Context, p host.Platform, sub string) ([]string, []string) {
	var warns []string
	args := []string{"build", sub, "--" + c.Mode()}
	if f := c.Flavor(); f != "" {
		if supportsFlavor(p) {
			args = append(args, "--flavor", f)
		} else {
			warns = append(warns, i18n.F("--flavor is not supported for %s builds by Flutter; building without flavor.", "Flutter 的 %s 构建不支持 --flavor，将忽略 flavor。", p))
		}
	}
	if t := c.Config.Build.Target; t != "" {
		args = append(args, "--target", t)
	}
	if c.Config.Build.BuildName != "" {
		args = append(args, "--build-name", string(c.Config.Build.BuildName))
	}
	if c.Config.Build.BuildNumber != "" {
		args = append(args, "--build-number", string(c.Config.Build.BuildNumber))
	}
	if p == host.Web {
		if c.Config.Obfuscate() || c.Config.Build.SplitDebugInfo != "" {
			warns = append(warns, i18n.S("--obfuscate/--split-debug-info do not apply to web builds (dart2js minifies by default).", "web 构建不支持 --obfuscate/--split-debug-info（dart2js 默认已压缩混淆）。"))
		}
	} else {
		if c.Config.Obfuscate() {
			args = append(args, "--obfuscate")
		}
		if c.Config.Obfuscate() || c.Config.Build.SplitDebugInfo != "" {
			args = append(args, "--split-debug-info="+c.SymbolsDir(p))
		}
	}
	for _, d := range c.Config.Build.DartDefine {
		args = append(args, "--dart-define="+d)
	}
	for _, f := range c.Config.Build.DartDefineFromFile {
		args = append(args, "--dart-define-from-file="+f)
	}
	if t := c.Config.Build.TreeShakeIcons; t != nil && !*t {
		args = append(args, "--no-tree-shake-icons")
	}
	return args, warns
}

// SymbolsDir is where --split-debug-info symbols go: next to the artifacts
// by default so they are kept together with the release.
func (c *Context) SymbolsDir(p host.Platform) string {
	base := c.Config.Build.SplitDebugInfo
	if base == "" {
		base = filepath.Join(c.OutDir, "debug-info")
	} else if !filepath.IsAbs(base) {
		base = filepath.Join(c.Project.Root, base)
	}
	return c.Rel(filepath.Join(base, string(p)))
}

// tailArgs appends config extra args (global, platform) and passthrough args.
func tailArgs(c *Context, platformExtra []string) []string {
	var out []string
	out = append(out, c.Config.Build.ExtraArgs...)
	out = append(out, platformExtra...)
	out = append(out, c.PassArgs...)
	return out
}

// ModeCap returns "Release"/"Profile"/"Debug".
func ModeCap(mode string) string {
	if mode == "" {
		return "Release"
	}
	return strings.ToUpper(mode[:1]) + mode[1:]
}

// ---- small helpers ----

func cmd(name string, args ...string) *runner.Cmd { return &runner.Cmd{Name: name, Args: args} }

func moveOp(c *Context, src, dst string) Op {
	return Op{Desc: i18n.F("move to %s", "移动到 %s", c.Rel(dst)), Fn: func() error { return pack.MoveFile(src, dst) }}
}

func copyOp(c *Context, src, dst string) Op {
	return Op{Desc: i18n.F("copy %s → %s", "复制 %s → %s", c.Rel(src), c.Rel(dst)), Fn: func() error { return pack.CopyFile(src, dst) }}
}

func resetDirOp(c *Context, dir string) Op {
	return Op{Desc: i18n.F("prepare %s", "准备 %s", c.Rel(dir)), Fn: func() error {
		if err := os.RemoveAll(dir); err != nil {
			return err
		}
		return os.MkdirAll(dir, 0o755)
	}}
}

// findNewest returns the newest path matching pattern modified at/after since.
func findNewest(pattern string, since time.Time, filter func(string) bool) string {
	matches, _ := filepath.Glob(pattern)
	var best string
	var bestT time.Time
	for _, m := range matches {
		if filter != nil && !filter(filepath.Base(m)) {
			continue
		}
		st, err := os.Stat(m)
		if err != nil {
			continue
		}
		if !since.IsZero() && st.ModTime().Before(since.Add(-2*time.Second)) {
			continue
		}
		if best == "" || st.ModTime().After(bestT) {
			best, bestT = m, st.ModTime()
		}
	}
	return best
}

func exists(p string) bool { _, err := os.Stat(p); return err == nil }

func notFound(what, where string) error {
	return fmt.Errorf("%s", i18n.F("flutter finished but %s was not found in %s", "flutter 已完成，但在 %[2]s 中找不到 %[1]s", what, where))
}

func sortedKeys(m map[string]string) []string {
	var k []string
	for x := range m {
		k = append(k, x)
	}
	sort.Strings(k)
	return k
}

// installHint returns an install command for a tool on this host.
func installHint(h host.Host, pk map[string]string) string { return h.Install(pk) }

// flavorCheck warns when --flavor is not among the flavors/schemes detected
// in the project (detection is heuristic, so this is never fatal).
func flavorCheck(c *Context, p host.Platform) (Issue, bool) {
	f := c.Flavor()
	if f == "" {
		return Issue{}, true
	}
	found := c.Project.Flavors(p)
	for _, x := range found {
		if strings.EqualFold(x, f) {
			return Issue{}, true
		}
	}
	where := map[host.Platform]string{host.Android: "android/app/build.gradle(.kts) productFlavors", host.IOS: "ios/Runner.xcodeproj schemes", host.MacOS: "macos/Runner.xcodeproj schemes"}[p]
	list := strings.Join(found, ", ")
	if list == "" {
		list = i18n.S("none", "无")
	}
	return warn(i18n.F("flavor %q not found in %s (found: %s)", "在 %[2]s 中找不到 flavor %[1]q（已找到：%[3]s）", f, where, list),
		i18n.S("check the spelling, or see https://docs.flutter.dev/deployment/flavors", "请检查拼写，或参考 https://docs.flutter.dev/deployment/flavors")), false
}
